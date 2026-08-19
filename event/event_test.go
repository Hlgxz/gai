package event_test

import (
	"testing"

	"github.com/Hlgxz/gai/event"
)

func TestDispatch(t *testing.T) {
	d := event.New()
	var got string
	d.Listen("user.created", func(p any) { got = p.(string) })
	d.Dispatch("user.created", "ada")
	if got != "ada" {
		t.Fatalf("got %q", got)
	}
}
