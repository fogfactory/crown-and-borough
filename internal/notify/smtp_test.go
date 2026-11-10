package notify

import (
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestNewMailerFromEnv(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	for name, values := range map[string]map[string]string{
		"nothing":      {},
		"no host":      {"SMTP_FROM": "noreply@example.test"},
		"no sender":    {"SMTP_HOST": "smtp.example.test"},
		"blank values": {"SMTP_HOST": "  ", "SMTP_FROM": " "},
	} {
		mailer, err := NewMailerFromEnv(env(values))
		if _, ok := mailer.(LogMailer); err != nil || !ok {
			t.Errorf("%s: mailer = %T, %v; want the log mailer", name, mailer, err)
		}
	}

	mailer, err := NewMailerFromEnv(env(map[string]string{
		"SMTP_HOST": "smtp.example.test", "SMTP_FROM": "Crown <noreply@example.test>", "SMTP_USERNAME": "u", "SMTP_PASSWORD": "p",
	}))
	smtpMailer, ok := mailer.(*SMTPMailer)
	if err != nil || !ok || smtpMailer.config.Port != 587 || smtpMailer.from.Address != "noreply@example.test" {
		t.Fatalf("mailer = %#v, %v", mailer, err)
	}
	for _, port := range []string{"abc", "0", "70000"} {
		if _, err := NewMailerFromEnv(env(map[string]string{"SMTP_HOST": "h", "SMTP_FROM": "a@b.test", "SMTP_PORT": port})); err == nil {
			t.Errorf("port %q accepted", port)
		}
	}
	if _, err := NewMailerFromEnv(env(map[string]string{"SMTP_HOST": "h", "SMTP_FROM": "not an address"})); err == nil {
		t.Error("invalid sender accepted")
	}
}

func TestSMTPMessageHeadersAndBody(t *testing.T) {
	mailer, err := NewSMTPMailer(SMTPConfig{Host: "smtp.example.test", From: "Crown & Borough <noreply@example.test>"})
	if err != nil {
		t.Fatal(err)
	}
	mailer.now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	to := mail.Address{Address: "alice@example.test"}
	raw, err := mailer.buildMessage(to, Email{To: to.Address, Subject: "Un nouveau tour a commencé", Body: "Ligne « un »\nLigne deux\n"})
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
	if err != nil || subject != "Un nouveau tour a commencé" {
		t.Fatalf("subject = %q, %v", subject, err)
	}
	if strings.ContainsAny(message.Header.Get("Subject"), "é") {
		t.Fatalf("subject not encoded: %q", message.Header.Get("Subject"))
	}
	if got := message.Header.Get("Content-Type"); got != "text/plain; charset=UTF-8" {
		t.Fatalf("content type = %q", got)
	}
	if message.Header.Get("Auto-Submitted") != "auto-generated" {
		t.Errorf("Auto-Submitted = %q", message.Header.Get("Auto-Submitted"))
	}
	if message.Header.Get("MIME-Version") != "1.0" || message.Header.Get("To") != "<alice@example.test>" {
		t.Fatalf("headers = %v", message.Header)
	}
	if _, err := message.Header.Date(); err != nil {
		t.Fatalf("date: %v", err)
	}
	if from, err := mail.ParseAddress(message.Header.Get("From")); err != nil || from.Address != "noreply@example.test" {
		t.Fatalf("from = %q, %v", message.Header.Get("From"), err)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(message.Body))
	if err != nil || string(body) != "Ligne « un »\r\nLigne deux\r\n" {
		t.Fatalf("body = %q, %v", body, err)
	}
}

func TestSMTPMessageRejectsHeaderInjection(t *testing.T) {
	mailer, err := NewSMTPMailer(SMTPConfig{Host: "smtp.example.test", From: "noreply@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	to := mail.Address{Address: "alice@example.test"}
	if _, err := mailer.buildMessage(to, Email{Subject: "hi\r\nBcc: x@y.test"}); err == nil {
		t.Fatal("subject with a line break accepted")
	}
}
