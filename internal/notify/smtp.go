package notify

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSMTPPort = 587
	implicitTLSPort = 465
)

// SMTPConfig is the connection to the mail server.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sender, an address or "Name <address>".
	From string
	// ImplicitTLS opens the connection in TLS (the usual mode of port 465)
	// instead of upgrading it with STARTTLS.
	ImplicitTLS bool
}

// SMTPMailer sends through an SMTP server with the standard library only:
// STARTTLS on the submission port (587), implicit TLS on 465. The connection
// is never used without TLS.
type SMTPMailer struct {
	config SMTPConfig
	from   mail.Address
	// implicitTLS opens the connection in TLS instead of upgrading it.
	implicitTLS bool
	// rootCAs replaces the system certificate pool; for tests.
	rootCAs *x509.CertPool
	now     func() time.Time
}

var _ Mailer = (*SMTPMailer)(nil)

// NewSMTPMailer validates the configuration.
func NewSMTPMailer(config SMTPConfig) (*SMTPMailer, error) {
	if config.Port == 0 {
		config.Port = defaultSMTPPort
	}
	if strings.ContainsAny(config.Host, " \r\n") || config.Host == "" {
		return nil, errors.New("notify: invalid SMTP host")
	}
	if config.Username != "" && config.Password == "" {
		return nil, errors.New("notify: SMTP_USERNAME is set without SMTP_PASSWORD")
	}
	from, err := mail.ParseAddress(config.From)
	if err != nil {
		return nil, fmt.Errorf("notify: invalid SMTP_FROM: %w", err)
	}
	return &SMTPMailer{config: config, from: *from, implicitTLS: config.ImplicitTLS, now: time.Now}, nil
}

// NewMailerFromEnv builds the mailer from SMTP_HOST, SMTP_PORT (default 587),
// SMTP_TLS, SMTP_USERNAME, SMTP_PASSWORD and SMTP_FROM. SMTP_TLS is "starttls"
// or "implicit"; left empty, port 465 means implicit TLS and any other port
// STARTTLS. When SMTP_HOST or SMTP_FROM is empty it returns the LogMailer,
// which sends nothing.
func NewMailerFromEnv(getenv func(string) string) (Mailer, error) {
	host := strings.TrimSpace(getenv("SMTP_HOST"))
	from := strings.TrimSpace(getenv("SMTP_FROM"))
	if host == "" || from == "" {
		return LogMailer{}, nil
	}
	port := defaultSMTPPort
	if value := strings.TrimSpace(getenv("SMTP_PORT")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return nil, fmt.Errorf("notify: invalid SMTP_PORT %q", value)
		}
		port = parsed
	}
	implicit := port == implicitTLSPort
	switch mode := strings.ToLower(strings.TrimSpace(getenv("SMTP_TLS"))); mode {
	case "":
	case "implicit":
		implicit = true
	case "starttls":
		implicit = false
	default:
		return nil, fmt.Errorf("notify: invalid SMTP_TLS %q, want starttls or implicit", mode)
	}
	return NewSMTPMailer(SMTPConfig{
		Host:     host,
		Port:     port,
		Username: strings.TrimSpace(getenv("SMTP_USERNAME")),
		// A secret often ends with the newline of the file it was created
		// from; spaces inside a password are kept.
		Password:    strings.TrimRight(getenv("SMTP_PASSWORD"), "\r\n"),
		From:        from,
		ImplicitTLS: implicit,
	})
}

func (m *SMTPMailer) Send(ctx context.Context, email Email) error {
	to, err := mail.ParseAddress(email.To)
	if err != nil {
		return fmt.Errorf("notify: invalid recipient: %w", err)
	}
	message, err := m.buildMessage(*to, email)
	if err != nil {
		return err
	}
	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if m.config.Username != "" {
		auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("notify: SMTP authentication: %w", err)
		}
	}
	if err := client.Mail(m.from.Address); err != nil {
		return fmt.Errorf("notify: SMTP sender: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("notify: SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("notify: SMTP data: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("notify: SMTP data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("notify: SMTP data: %w", err)
	}
	return client.Quit()
}

// connect opens the connection and secures it before any credential is sent.
func (m *SMTPMailer) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port))
	tlsConfig := &tls.Config{ServerName: m.config.Host, MinVersion: tls.VersionTLS12, RootCAs: m.rootCAs}
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("notify: SMTP connect: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	implicit := m.implicitTLS
	if implicit {
		connection = tls.Client(connection, tlsConfig)
	}
	client, err := smtp.NewClient(connection, m.config.Host)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("notify: SMTP greeting: %w", err)
	}
	if !implicit {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, errors.New("notify: SMTP server does not offer STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("notify: SMTP STARTTLS: %w", err)
		}
	}
	return client, nil
}

// buildMessage renders the RFC 5322 message: minimal headers, the subject in
// RFC 2047 encoded words and a quoted-printable UTF-8 plain text body.
func (m *SMTPMailer) buildMessage(to mail.Address, email Email) ([]byte, error) {
	if strings.ContainsAny(email.Subject, "\r\n") {
		return nil, errors.New("notify: invalid subject")
	}
	var identifier [12]byte
	if _, err := rand.Read(identifier[:]); err != nil {
		return nil, err
	}
	domain := "localhost"
	if _, after, ok := strings.Cut(m.from.Address, "@"); ok && after != "" {
		domain = after
	}
	var buffer bytes.Buffer
	header := func(name, value string) { buffer.WriteString(name + ": " + value + "\r\n") }
	header("From", m.from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", email.Subject))
	header("Date", m.now().UTC().Format(time.RFC1123Z))
	header("Message-ID", "<"+hex.EncodeToString(identifier[:])+"@"+domain+">")
	// Keeps auto-responders from answering a notification (RFC 3834).
	header("Auto-Submitted", "auto-generated")
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	buffer.WriteString("\r\n")
	body := quotedprintable.NewWriter(&buffer)
	if _, err := body.Write([]byte(strings.ReplaceAll(email.Body, "\r\n", "\n"))); err != nil {
		return nil, err
	}
	if err := body.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
