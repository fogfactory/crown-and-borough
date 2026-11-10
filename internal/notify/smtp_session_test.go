package notify

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTPServer is a minimal SMTP server on 127.0.0.1 that records one
// session: the commands received, whether they came over TLS and the message.
type fakeSMTPServer struct {
	listener  net.Listener
	tlsConfig *tls.Config
	roots     *x509.CertPool
	// implicitTLS serves TLS from the first byte; otherwise STARTTLS is
	// offered unless noStartTLS is set.
	implicitTLS bool
	noStartTLS  bool
	rejectAuth  bool

	mu       sync.Mutex
	commands []string // verbs, suffixed with "+tls" once the session is secured
	auth     string   // decoded AUTH PLAIN credentials
	sender   string
	rcpt     string
	data     string
	done     chan struct{}
}

func newFakeSMTPServer(t *testing.T, configure func(*fakeSMTPServer)) *fakeSMTPServer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fake smtp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:         true,

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &fakeSMTPServer{
		listener: listener,
		tlsConfig: &tls.Config{Certificates: []tls.Certificate{{
			Certificate: [][]byte{der},
			PrivateKey:  key,
		}}},
		roots: roots,
		done:  make(chan struct{}),
	}
	if configure != nil {
		configure(server)
	}
	go server.serve()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-server.done:
		case <-time.After(5 * time.Second):
			t.Error("fake SMTP server did not finish its session")
		}
	})
	return server
}

// mailer returns an SMTPMailer talking to the server and trusting its
// certificate.
func (s *fakeSMTPServer) mailer(t *testing.T, username, password string) *SMTPMailer {
	t.Helper()
	host, portText, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	mailer, err := NewSMTPMailer(SMTPConfig{
		Host: host, Port: port, Username: username, Password: password,
		From: "Crown & Borough <game@example.test>",
	})
	if err != nil {
		t.Fatal(err)
	}
	mailer.rootCAs = s.roots
	mailer.implicitTLS = s.implicitTLS
	return mailer
}

func (s *fakeSMTPServer) record(verb string, secured bool) {
	if secured {
		verb += "+tls"
	}
	s.mu.Lock()
	s.commands = append(s.commands, verb)
	s.mu.Unlock()
}

func (s *fakeSMTPServer) serve() {
	defer close(s.done)
	connection, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
	secured := false
	if s.implicitTLS {
		connection = tls.Server(connection, s.tlsConfig)
		secured = true
	}
	reader := bufio.NewReader(connection)
	reply := func(line string) { _, _ = connection.Write([]byte(line + "\r\n")) }
	reply("220 fake ESMTP")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb, argument, _ := strings.Cut(line, " ")
		verb = strings.ToUpper(verb)
		s.record(verb, secured)
		switch verb {
		case "EHLO":
			switch {
			case !secured && !s.noStartTLS:
				reply("250-fake")
				reply("250 STARTTLS")
			case secured:
				reply("250-fake")
				reply("250 AUTH PLAIN")
			default:
				reply("250 fake")
			}
		case "STARTTLS":
			reply("220 go ahead")
			secure := tls.Server(connection, s.tlsConfig)
			if err := secure.Handshake(); err != nil {
				return
			}
			connection = secure
			reader = bufio.NewReader(connection)
			secured = true
		case "AUTH":
			_, encoded, _ := strings.Cut(argument, " ")
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			s.mu.Lock()
			s.auth = string(decoded)
			s.mu.Unlock()
			if s.rejectAuth {
				reply("535 authentication failed")
				continue
			}
			reply("235 ok")
		case "MAIL":
			s.mu.Lock()
			s.sender = argument
			s.mu.Unlock()
			reply("250 ok")
		case "RCPT":
			s.mu.Lock()
			s.rcpt = argument
			s.mu.Unlock()
			reply("250 ok")
		case "DATA":
			reply("354 send")
			var data strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			s.mu.Lock()
			s.data = data.String()
			s.mu.Unlock()
			reply("250 queued")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("502 not implemented")
		}
	}
}

func (s *fakeSMTPServer) session() (commands []string, auth, sender, rcpt, data string) {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commands, s.auth, s.sender, s.rcpt, s.data
}

func testEmail() Email {
	return Email{To: "alice@example.test", Subject: "Un nouveau tour a commencé", Body: "Ouvrir la partie : https://game.example.test/games/g1\n"}
}

func sendContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSMTPMailerUpgradesToTLSBeforeAuthenticating(t *testing.T) {
	server := newFakeSMTPServer(t, nil)
	if err := server.mailer(t, "user", "secret").Send(sendContext(t), testEmail()); err != nil {
		t.Fatalf("send: %v", err)
	}
	commands, auth, sender, rcpt, data := server.session()
	want := "EHLO,STARTTLS,EHLO+tls,AUTH+tls,MAIL+tls,RCPT+tls,DATA+tls,QUIT+tls"
	if got := strings.Join(commands, ","); got != want {
		t.Fatalf("commands = %s, want %s", got, want)
	}
	if auth != "\x00user\x00secret" {
		t.Errorf("credentials = %q", auth)
	}
	if !strings.HasPrefix(sender, "FROM:<game@example.test>") || rcpt != "TO:<alice@example.test>" {
		t.Errorf("envelope = %q -> %q", sender, rcpt)
	}
	if !strings.Contains(data, "To: <alice@example.test>\r\n") || !strings.Contains(data, "https://game.example.test/games/g1") {
		t.Errorf("message = %q", data)
	}
}

func TestSMTPMailerSpeaksImplicitTLS(t *testing.T) {
	server := newFakeSMTPServer(t, func(s *fakeSMTPServer) { s.implicitTLS = true })
	if err := server.mailer(t, "user", "secret").Send(sendContext(t), testEmail()); err != nil {
		t.Fatalf("send: %v", err)
	}
	commands, _, _, _, data := server.session()
	want := "EHLO+tls,AUTH+tls,MAIL+tls,RCPT+tls,DATA+tls,QUIT+tls"
	if got := strings.Join(commands, ","); got != want {
		t.Fatalf("commands = %s, want %s", got, want)
	}
	if data == "" {
		t.Error("no message received")
	}
}

func TestSMTPMailerSendsWithoutCredentials(t *testing.T) {
	server := newFakeSMTPServer(t, nil)
	if err := server.mailer(t, "", "").Send(sendContext(t), testEmail()); err != nil {
		t.Fatalf("send: %v", err)
	}
	commands, _, _, _, _ := server.session()
	if got := strings.Join(commands, ","); strings.Contains(got, "AUTH") {
		t.Fatalf("commands = %s, want no AUTH without a username", got)
	}
}

func TestSMTPMailerRefusesAServerWithoutSTARTTLS(t *testing.T) {
	server := newFakeSMTPServer(t, func(s *fakeSMTPServer) { s.noStartTLS = true })
	err := server.mailer(t, "user", "secret").Send(sendContext(t), testEmail())
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("error = %v, want a STARTTLS refusal", err)
	}
	commands, auth, _, _, data := server.session()
	if auth != "" || data != "" {
		t.Fatalf("credentials or message sent in clear: commands = %v", commands)
	}
}

func TestSMTPMailerRefusesAnUntrustedCertificate(t *testing.T) {
	server := newFakeSMTPServer(t, nil)
	mailer := server.mailer(t, "user", "secret")
	mailer.rootCAs = x509.NewCertPool() // trusts nothing
	if err := mailer.Send(sendContext(t), testEmail()); err == nil {
		t.Fatal("send succeeded with an untrusted certificate")
	}
	_, auth, _, _, data := server.session()
	if auth != "" || data != "" {
		t.Fatal("credentials or message sent to an untrusted server")
	}
}

func TestSMTPMailerReportsRejectedCredentials(t *testing.T) {
	server := newFakeSMTPServer(t, func(s *fakeSMTPServer) { s.rejectAuth = true })
	err := server.mailer(t, "user", "wrong").Send(sendContext(t), testEmail())
	if err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("error = %v, want an authentication error", err)
	}
	if _, _, _, _, data := server.session(); data != "" {
		t.Fatal("message sent after rejected credentials")
	}
}
