package websocket

import (
	"net/http"

	ghttp "github.com/Hlgxz/gai/http"
	gws "github.com/gorilla/websocket"
)

// Conn is a gorilla websocket connection.
type Conn = gws.Conn

// Upgrader is the default upgrader. Override CheckOrigin in production.
var Upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
