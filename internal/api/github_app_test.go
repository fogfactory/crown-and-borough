package api

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testPEM(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return key, string(block)
}

func TestGitHubAppClientSignsJWTAndCachesInstallationToken(t *testing.T) {
	key, keyPEM := testPEM(t)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	var exchanges atomic.Int32
	var issueAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/42/access_tokens":
			exchanges.Add(1)
			jwt := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			parts := strings.Split(jwt, ".")
			if len(parts) != 3 {
				t.Errorf("jwt = %q", jwt)
				return
			}
			signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
			digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
			if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
				t.Errorf("signature: %v", err)
			}
			raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
			var claims map[string]any
			_ = json.Unmarshal(raw, &claims)
			if claims["iss"] != "7" || int64(claims["exp"].(float64)) <= now.Unix() {
				t.Errorf("claims = %v", claims)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"token":"inst-token","expires_at":"2026-01-01T13:00:00Z"}`))
		case "/repos/o/r/issues":
			issueAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"html_url":"https://github.com/o/r/issues/1"}`))
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	client, err := NewGitHubAppIssueClient("7", "42", keyPEM, "o/r")
	if err != nil || client == nil {
		t.Fatalf("client=%v err=%v", client, err)
	}
	client.BaseURL = upstream.URL
	source := client.Tokens.(*GitHubAppTokenSource)
	source.BaseURL, source.Now = upstream.URL, func() time.Time { return now }

	for range 2 {
		if _, err := client.CreateIssue(context.Background(), "t", "b", nil); err != nil {
			t.Fatal(err)
		}
	}
	if issueAuth != "Bearer inst-token" || exchanges.Load() != 1 {
		t.Fatalf("auth=%q exchanges=%d", issueAuth, exchanges.Load())
	}
	// Within the margin of expiry the token is renewed.
	source.Now = func() time.Time { return now.Add(56 * time.Minute) }
	if _, err := client.CreateIssue(context.Background(), "t", "b", nil); err != nil || exchanges.Load() != 2 {
		t.Fatalf("exchanges=%d err=%v", exchanges.Load(), err)
	}
}

func TestGitHubAppClientDisabledOrInvalid(t *testing.T) {
	_, keyPEM := testPEM(t)
	if client, err := NewGitHubAppIssueClient("", "42", keyPEM, "o/r"); client != nil || err != nil {
		t.Fatalf("missing app id = %v %v", client, err)
	}
	if client, err := NewGitHubAppIssueClient("7", "42", "  ", "o/r"); client != nil || err != nil {
		t.Fatalf("missing key = %v %v", client, err)
	}
	if _, err := NewGitHubAppIssueClient("7", "42", "not a key", "o/r"); err == nil {
		t.Fatal("invalid key must error")
	}
	escaped := strings.ReplaceAll(strings.TrimSpace(keyPEM), "\n", `\n`)
	if client, err := NewGitHubAppIssueClient("7", "42", escaped, "o/r"); client == nil || err != nil {
		t.Fatalf("escaped newlines = %v %v", client, err)
	}
}

func TestGitHubAppTokenExchangeFailureHidesDetails(t *testing.T) {
	_, keyPEM := testPEM(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()
	client, _ := NewGitHubAppIssueClient("7", "42", keyPEM, "o/r")
	client.Tokens.(*GitHubAppTokenSource).BaseURL = upstream.URL
	if _, err := client.CreateIssue(context.Background(), "t", "b", nil); err == nil || strings.Contains(err.Error(), "BEGIN") {
		t.Fatalf("err = %v", err)
	}
}
