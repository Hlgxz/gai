package gaitest_test

import (
	"net/http"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/gaitest"
	"github.com/Hlgxz/gai/mail"
	"github.com/Hlgxz/gai/router"
)

func TestPerformJSON(t *testing.T) {
	r := router.New()
	r.Post("/echo", func(c *ghttp.Context) {
		var in map[string]any
		_ = c.BindJSON(&in)
		c.Success(in)
	})
	resp := gaitest.Perform(r, http.MethodPost, "/echo", map[string]any{"name": "gai"})
	resp.AssertOK(t)
	resp.AssertJSONPath(t, "code", 0)
	resp.AssertJSONPath(t, "data.name", "gai")
}

func TestFakes(t *testing.T) {
	mailer := gaitest.NewMailRecorder()
	if err := mailer.Send(mail.Message{To: []string{"a@b.c"}, Subject: "hi", Text: "ok"}); err != nil {
		t.Fatal(err)
	}
	if len(mailer.Sent) != 1 {
		t.Fatalf("sent %d", len(mailer.Sent))
	}
	if gaitest.NewQueue() == nil {
		t.Fatal("nil queue")
	}
}
