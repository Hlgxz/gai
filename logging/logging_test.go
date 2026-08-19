package logging_test

import (
	"log/slog"
	"net/http/httptest"
	"testing"

	ghttp "github.com/Hlgxz/gai/http"
	"github.com/Hlgxz/gai/logging"
)

func TestSetupAndFromRequest(t *testing.T) {
	logger := logging.Setup(logging.Config{Level: "debug", Output: "stderr"})
	if logger == nil {
		t.Fatal("nil logger")
	}
	req := httptest.NewRequest("GET", "/x", nil)
	c := ghttp.NewContext(httptest.NewRecorder(), req)
	c.Set("request_id", "abc")
	l := logging.FromRequest(c)
	if l == nil {
		t.Fatal("expected request logger")
	}
	l.Info("ok")
	_ = slog.Default()
}
