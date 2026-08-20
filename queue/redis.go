package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisEnvelope struct {
	Job         Job   `json:"job"`
	AvailableAt int64 `json:"available_at"`
}

// Redis is a list + sorted-set queue (ready list, delayed ZSET, failed list).
type Redis struct {
	client redis.Cmdable
	key    string
}

func NewRedis(client redis.Cmdable, key string) *Redis {
	if key == "" {
		key = "gai:queue"
	}
	return &Redis{client: client, key: key}
}

func (r *Redis) readyKey() string   { return r.key }
func (r *Redis) delayedKey() string { return r.key + ":delayed" }
func (r *Redis) failedKey() string  { return r.key + ":failed" }

func (r *Redis) Push(job Job) error {
	env := redisEnvelope{Job: job, AvailableAt: time.Now().Add(job.Delay).Unix()}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if job.Delay > 0 {
		return r.client.ZAdd(ctx, r.delayedKey(), redis.Z{
			Score:  float64(env.AvailableAt),
			Member: raw,
		}).Err()
	}
	return r.client.RPush(ctx, r.readyKey(), raw).Err()
}

func (r *Redis) Pop(ctx context.Context) (*Job, error) {
	if err := r.promoteDelayed(ctx); err != nil {
		return nil, err
	}
	res, err := r.client.BLPop(ctx, 2*time.Second, r.readyKey()).Result()
	if err == redis.Nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return nil, redis.Nil
	}
	var env redisEnvelope
	if err := json.Unmarshal([]byte(res[1]), &env); err != nil {
		return nil, err
	}
	j := env.Job
	return &j, nil
}

func (r *Redis) promoteDelayed(ctx context.Context) error {
	now := float64(time.Now().Unix())
	items, err := r.client.ZRangeByScore(ctx, r.delayedKey(), &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(int64(now), 10),
	}).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	for _, item := range items {
		pipe := r.client.TxPipeline()
		pipe.ZRem(ctx, r.delayedKey(), item)
		pipe.RPush(ctx, r.readyKey(), item)
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Redis) Fail(job Job, cause error) error {
	if cause != nil && job.Error == "" {
		job.Error = cause.Error()
	}
	raw, err := encodeJob(job)
	if err != nil {
		return err
	}
	return r.client.RPush(context.Background(), r.failedKey(), raw).Err()
}

func (r *Redis) Retry(job Job, delay time.Duration) error {
	job.Delay = delay
	return r.Push(job)
}

func (r *Redis) Failed() []Job {
	raws, err := r.client.LRange(context.Background(), r.failedKey(), 0, 99).Result()
	if err != nil {
		return nil
	}
	out := make([]Job, 0, len(raws))
	for _, raw := range raws {
		j, err := decodeJob([]byte(raw))
		if err != nil {
			continue
		}
		out = append(out, j)
	}
	return out
}
