package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis is a list-backed queue using RPUSH / BLPOP.
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

func (r *Redis) Push(job Job) error {
	if job.Delay > 0 {
		time.Sleep(0) // delay is applied on pop via available_at
	}
	raw, err := json.Marshal(redisJob{
		Name:        job.Name,
		Payload:     job.Payload,
		AvailableAt: time.Now().Add(job.Delay).Unix(),
	})
	if err != nil {
		return err
	}
	return r.client.RPush(context.Background(), r.key, raw).Err()
}

func (r *Redis) Pop(ctx context.Context) (*Job, error) {
	for {
		res, err := r.client.BLPop(ctx, 2*time.Second, r.key).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(res) < 2 {
			continue
		}
		var j redisJob
		if err := json.Unmarshal([]byte(res[1]), &j); err != nil {
			continue
		}
		wait := time.Until(time.Unix(j.AvailableAt, 0))
		if wait > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
		return &Job{Name: j.Name, Payload: j.Payload}, nil
	}
}

type redisJob struct {
	Name        string `json:"name"`
	Payload     []byte `json:"payload"`
	AvailableAt int64  `json:"available_at"`
}
