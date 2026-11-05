package api

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	appJWTLifetime = 9 * time.Minute
	// A cached installation token is renewed this long before it expires.
	appTokenMargin = 5 * time.Minute
)

// TokenSource yields the token that authenticates a GitHub API call.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

// GitHubAppTokenSource authenticates as a GitHub App installation: it signs a
// short-lived JWT with the App private key and exchanges it for an
// installation token, which it keeps until shortly before it expires.
type GitHubAppTokenSource struct {
	AppID          string
	InstallationID string
	BaseURL        string // defaults to https://api.github.com
	HTTPClient     *http.Client
	Now            func() time.Time

	key *rsa.PrivateKey

	mu      sync.Mutex
	token   string
	expires time.Time
}

// ParseAppPrivateKey reads a PEM RSA private key (PKCS#1, as downloaded from
// GitHub, or PKCS#8). Literal "\n" sequences are accepted so the key survives
// single-line environment variables.
func ParseAppPrivateKey(text string) (*rsa.PrivateKey, error) {
	text = strings.ReplaceAll(strings.TrimSpace(text), `\n`, "\n")
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return nil, errors.New("github app key is not valid PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("github app key is not a PKCS#1 or PKCS#8 private key")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("github app key is not an RSA key")
	}
	return key, nil
}

// NewGitHubAppIssueClient returns an issue client authenticated as a GitHub App
// installation. It returns nil without error when the App is not configured
// (any identifier or the key is empty), so the route can report itself disabled.
func NewGitHubAppIssueClient(appID, installationID, privateKey, repository string) (*GitHubIssueClient, error) {
	appID, installationID = strings.TrimSpace(appID), strings.TrimSpace(installationID)
	if appID == "" || installationID == "" || strings.TrimSpace(privateKey) == "" {
		return nil, nil
	}
	key, err := ParseAppPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return &GitHubIssueClient{
		Tokens:     &GitHubAppTokenSource{AppID: appID, InstallationID: installationID, key: key},
		Repository: repository,
	}, nil
}

func (s *GitHubAppTokenSource) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *GitHubAppTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.token != "" && now.Add(appTokenMargin).Before(s.expires) {
		return s.token, nil
	}
	jwt, err := s.signedJWT(now)
	if err != nil {
		return "", err
	}
	token, expires, err := s.exchange(ctx, jwt)
	if err != nil {
		return "", err
	}
	s.token, s.expires = token, expires
	return token, nil
}

func (s *GitHubAppTokenSource) signedJWT(now time.Time) (string, error) {
	segment := func(value any) (string, error) {
		raw, err := json.Marshal(value)
		return base64.RawURLEncoding.EncodeToString(raw), err
	}
	header, err := segment(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	// iat is backdated a minute to tolerate clock drift.
	claims, err := segment(map[string]any{
		"iat": now.Add(-time.Minute).Unix(),
		"exp": now.Add(appJWTLifetime).Unix(),
		"iss": s.AppID,
	})
	if err != nil {
		return "", err
	}
	signingInput := header + "." + claims
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *GitHubAppTokenSource) exchange(ctx context.Context, jwt string) (string, time.Time, error) {
	base := strings.TrimRight(s.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/app/installations/"+s.InstallationID+"/access_tokens", bytes.NewReader(nil))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("github app token request failed: %w", err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusCreated {
		return "", time.Time{}, fmt.Errorf("github app token request responded %d", response.StatusCode)
	}
	var granted struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &granted); err != nil || granted.Token == "" {
		return "", time.Time{}, errors.New("github app token response is invalid")
	}
	return granted.Token, granted.ExpiresAt, nil
}
