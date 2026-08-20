package mail_test

import (
	"strings"
	"testing"

	"github.com/Hlgxz/gai/mail"
)

func TestLogMailerAndAttachments(t *testing.T) {
	l := &mail.LogMailer{}
	msg := mail.Message{
		From:    "a@b.com",
		To:      []string{"c@d.com"},
		Subject: "hi",
		HTML:    "<p>ok</p>",
		Attachments: []mail.Attachment{{
			Name: "note.txt",
			Data: []byte("hello"),
		}},
	}
	if err := l.Send(msg); err != nil {
		t.Fatal(err)
	}
	if len(l.Sent) != 1 || l.Sent[0].Subject != "hi" {
		t.Fatalf("%+v", l.Sent)
	}
	if !strings.Contains(l.Sent[0].HTML, "ok") {
		t.Fatal("html")
	}
}
