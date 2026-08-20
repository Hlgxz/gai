package websocket

import (
	"net/http"
	"strings"

	ghttp "github.com/Hlgxz/gai/http"
	gws "github.com/gorilla/websocket"
)

// Conn is a gorilla websocket connection.
type Conn = gws.Conn

// Upgrader is the default upgrader. Call Configure / AllowOrigins in production.
var Upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     AllowOrigins(),
}

// AllowOrigins returns a CheckOrigin that accepts the given origins.
// An empty list denies every browser Origin (non-browser clients still work).
// Pass "*" to allow any origin.
func AllowOrigins(origins ...string) func(*http.Request) bool {
	allow := map[string]struct{}{}
	anyOrigin := false
	for _, o := range origins {
		if o == "*" {
			anyOrigin = true
			continue
		}
		allow[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if anyOrigin {
			return true
		}
		_, ok := allow[strings.TrimRight(origin, "/")]
		return ok
	}
}

// Configure sets Upgrader.CheckOrigin from an origin allow-list.
func Configure(origins []string) {
	Upgrader.CheckOrigin = AllowOrigins(origins...)
}

// Upgrade hijacks the HTTP connection into a WebSocket.
func Upgrade(c *ghttp.Context) (*Conn, error) {
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, err
	}
	c.Abort()
	return conn, nil
}
