package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Message is a simple outbound email.
type Message struct {
	From    string
	To      []string
	Subject string
	Text    string
	HTML    string
}

// Mailer sends messages.
type Mailer interface {
	Send(msg Message) error
}

// SMTPMailer sends mail via net/smtp.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s *SMTPMailer) Send(msg Message) error {
	from := msg.From
	if from == "" {
		from = s.From
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	body := buildBody(from, msg)

	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	return smtp.SendMail(addr, auth, from, msg.To, []byte(body))
}

func buildBody(from string, msg Message) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	if msg.HTML != "" {
		b.WriteString("MIME-Version: 1.0\r\n")
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.HTML)
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text)
	}
	return b.String()
}

// LogMailer writes messages to a callback (tests / local dev).
type LogMailer struct {
	Sent []Message
}

func (l *LogMailer) Send(msg Message) error {
	l.Sent = append(l.Sent, msg)
	return nil
}
