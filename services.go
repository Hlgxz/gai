package gai

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Hlgxz/gai/cache"
	"github.com/Hlgxz/gai/client"
	"github.com/Hlgxz/gai/event"
	"github.com/Hlgxz/gai/lock"
	"github.com/Hlgxz/gai/mail"
	"github.com/Hlgxz/gai/queue"
	"github.com/Hlgxz/gai/schedule"
	"github.com/Hlgxz/gai/session"
	"github.com/Hlgxz/gai/storage"
	"github.com/Hlgxz/gai/tracing"
	"github.com/Hlgxz/gai/websocket"
	"github.com/redis/go-redis/v9"
)

// UseServices wires cache, session, queue, mail, storage, events, HTTP client,
// lock, scheduler, and tracing from app.* config. Redis is shared when
// app.redis.addr is set; otherwise in-memory backends are used.
func (app *Application) UseServices() *Application {
	cfg := app.config
	httpClient := client.New()
	app.Instance("http.client", httpClient)
	app.Instance("events", event.New())

	var rdb *redis.Client
	addr := cfg.GetString("app.redis.addr", "")
	if addr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.GetString("app.redis.password", ""),
			DB:       cfg.GetInt("app.redis.db", 0),
		})
		app.Instance("redis", rdb)
		app.OnShutdown(func(ctx context.Context) error {
			return rdb.Close()
		})
	}

	app.wireLock(rdb)
	app.wireCache(rdb)
	app.wireSession(rdb)
	app.wireQueue(rdb)
	app.wireMail()
	app.wireStorage()
	app.wireSchedule()
	app.wireTracing()

	origins := cfg.GetString("app.websocket.origins", "")
	if origins != "" {
		websocket.Configure(splitCSV(origins))
	}

	return app
}

func (app *Application) wireLock(rdb *redis.Client) {
	var locker lock.Locker = lock.NewMemory()
	if rdb != nil && app.config.GetString("app.lock.driver", "redis") != "memory" {
		locker = lock.NewRedis(rdb)
	}
	app.Instance("lock", locker)
}

func (app *Application) wireCache(rdb *redis.Client) {
	driver := app.config.GetString("app.cache.driver", "memory")
	var store cache.Store
	if driver == "redis" && rdb != nil {
		store = cache.NewRedis(rdb)
	} else {
		mem := cache.NewMemory()
		store = mem
		app.OnShutdown(func(context.Context) error {
			mem.Stop()
			return nil
		})
	}
	mgr := &cache.Manager{
		Store:  store,
		Prefix: app.config.GetString("app.cache.prefix", "gai:"),
	}
	if locker, err := app.Resolve("lock"); err == nil {
		if l, ok := locker.(lock.Locker); ok {
			mgr.Locker = l
		}
	}
	app.Instance("cache", mgr)
}

func (app *Application) wireSession(rdb *redis.Client) {
	driver := app.config.GetString("app.session.driver", "memory")
	var store session.Store
	if driver == "redis" && rdb != nil {
		store = session.NewRedisStore(rdb)
	} else {
		store = session.NewMemoryStore()
	}
	ttl := time.Duration(app.config.GetInt("app.session.ttl", 86400)) * time.Second
	mgr := session.New(store)
	mgr.CookieName = app.config.GetString("app.session.cookie", "gai_session")
	mgr.TTL = ttl
	mgr.Secure = app.config.GetBool("app.session.secure", false)
	app.Instance("session", mgr)
}

func (app *Application) wireQueue(rdb *redis.Client) {
	driver := app.config.GetString("app.queue.driver", "memory")
	var q queue.Queue
	if driver == "redis" && rdb != nil {
		q = queue.NewRedis(rdb, app.config.GetString("app.queue.key", "gai:queue"))
	} else {
		q = queue.NewMemory(app.config.GetInt("app.queue.size", 256))
	}
	mgr := queue.New(q).
		SetMaxTries(app.config.GetInt("app.queue.retry", 3)).
		SetTimeout(time.Duration(app.config.GetInt("app.queue.timeout", 60)) * time.Second).
		SetConcurrency(app.config.GetInt("app.queue.concurrency", 1))
	app.Instance("queue", mgr)
}

func (app *Application) wireMail() {
	driver := app.config.GetString("app.mail.driver", "log")
	var mailer mail.Mailer
	if driver == "smtp" {
		mailer = &mail.SMTPMailer{
			Host:     app.config.GetString("app.mail.host", "localhost"),
			Port:     app.config.GetInt("app.mail.port", 587),
			Username: app.config.GetString("app.mail.username", ""),
			Password: app.config.GetString("app.mail.password", ""),
			From:     app.config.GetString("app.mail.from", ""),
			TLS:      app.config.GetBool("app.mail.tls", true),
		}
	} else {
		mailer = &mail.LogMailer{}
	}
	app.Instance("mail", mailer)
}

func (app *Application) wireStorage() {
	mgr := storage.NewManager()
	root := app.config.GetString("app.storage.local.root", "storage/app")
	base := app.config.GetString("app.storage.local.url", "/storage")
	mgr.Add("local", &storage.Local{Root: root, BaseURL: base})

	bucket := app.config.GetString("app.storage.s3.bucket", "")
	if bucket != "" {
		mgr.Add("s3", &storage.S3{
			Endpoint:  app.config.GetString("app.storage.s3.endpoint", ""),
			Region:    app.config.GetString("app.storage.s3.region", "us-east-1"),
			Bucket:    bucket,
			AccessKey: app.config.GetString("app.storage.s3.access_key", ""),
			SecretKey: app.config.GetString("app.storage.s3.secret_key", ""),
			BaseURL:   app.config.GetString("app.storage.s3.url", ""),
			PathStyle: app.config.GetBool("app.storage.s3.path_style", true),
		})
	}
	def := app.config.GetString("app.storage.default", "local")
	if _, err := mgr.Disk(def); err != nil {
		slog.Warn("gai: storage default disk missing, using local", "disk", def)
	}
	app.Instance("storage", mgr)
}

func (app *Application) wireSchedule() {
	s := schedule.New()
	if locker, err := app.Resolve("lock"); err == nil {
		if l, ok := locker.(lock.Locker); ok {
			s.UseLock(l)
		}
	}
	app.Instance("schedule", s)
}

func (app *Application) wireTracing() {
	enabled := app.config.GetBool("app.tracing.enabled", false)
	tracing.Setup(tracing.Config{
		ServiceName: app.config.GetString("app.name", "gai"),
		Enabled:     enabled,
	})
	if ep := app.config.GetString("app.tracing.endpoint", ""); ep != "" {
		tracing.EnableOTLP(ep)
	}
	if enabled {
		app.router.Use(tracing.Middleware())
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
