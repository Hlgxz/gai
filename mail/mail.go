package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
)

// Attachment is a file attached to an outbound message.
type Attachment struct {
	Name        string
	ContentType string
	Data        []byte
}

// Message is a simple outbound email.
type Message struct {
	From        string
	To          []string
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
}

// Mailer sends messages.
type Mailer interface {
	Send(msg Message) error
}

// SMTPMailer sends mail via net/smtp, optionally with STARTTLS.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

func (s *SMTPMailer) Send(msg Message) error {
	from := msg.From
	if from == "" {
		from = s.From
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	body, err := buildBody(from, msg)
	if err != nil {
		return err
	}

	if !s.TLS {
		var auth smtp.Auth
		if s.Username != "" {
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		}
		return smtp.SendMail(addr, auth, from, msg.To, body)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if s.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, to := range msg.To {
		if err := c.Rcpt(to); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func buildBody(from string, msg Message) ([]byte, error) {
	if len(msg.Attachments) == 0 && msg.HTML == "" {
		var b strings.Builder
		b.WriteString("From: " + from + "\r\n")
		b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
		b.WriteString("Subject: " + msg.Subject + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text)
		return []byte(b.String()), nil
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	buf.WriteString("Subject: " + msg.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	w := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/mixed; boundary=" + w.Boundary() + "\r\n\r\n")

	if msg.HTML != "" {
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Type", "text/html; charset=UTF-8")
		part, err := w.CreatePart(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write([]byte(msg.HTML)); err != nil {
			return nil, err
		}
	} else if msg.Text != "" {
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Type", "text/plain; charset=UTF-8")
		part, err := w.CreatePart(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write([]byte(msg.Text)); err != nil {
			return nil, err
		}
	}

	for _, a := range msg.Attachments {
		ct := a.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		hdr := textproto.MIMEHeader{}
		hdr.Set("Content-Type", ct)
		hdr.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.Name))
		hdr.Set("Content-Transfer-Encoding", "base64")
		part, err := w.CreatePart(hdr)
		if err != nil {
			return nil, err
		}
		enc := make([]byte, base64.StdEncoding.EncodedLen(len(a.Data)))
		base64.StdEncoding.Encode(enc, a.Data)
		if _, err := part.Write(enc); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// LogMailer writes messages to a callback (tests / local dev).
type LogMailer struct {
	Sent []Message
}

func (l *LogMailer) Send(msg Message) error {
	l.Sent = append(l.Sent, msg)
	return nil
}
