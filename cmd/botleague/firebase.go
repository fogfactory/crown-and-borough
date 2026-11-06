package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Tokens is a Firebase session. The refresh token survives restarts.
type Tokens struct {
	Email        string    `json:"email"`
	IDToken      string    `json:"idToken"`
	RefreshToken string    `json:"refreshToken"`
	Expires      time.Time `json:"expires"`
}

// firebaseAuth speaks the Identity Toolkit REST API, either of the Firebase
// Auth emulator or of the real service. Both authenticate with an email link
// (the only sign-in method of the game), so no browser is needed: the
// out-of-band code is read from the emulator, or from the link of the mail.
type firebaseAuth struct {
	server ServerConfig
	http   *http.Client
}

func newFirebaseAuth(server ServerConfig) *firebaseAuth {
	return &firebaseAuth{server: server, http: &http.Client{Timeout: 30 * time.Second}}
}

func (f *firebaseAuth) identityBase() string {
	if f.server.Mode == "emulator" {
		return strings.TrimRight(f.server.AuthEmulator, "/") + "/identitytoolkit.googleapis.com/v1"
	}
	return "https://identitytoolkit.googleapis.com/v1"
}

func (f *firebaseAuth) secureTokenURL() string {
	if f.server.Mode == "emulator" {
		return strings.TrimRight(f.server.AuthEmulator, "/") + "/securetoken.googleapis.com/v1/token"
	}
	return "https://securetoken.googleapis.com/v1/token"
}

func (f *firebaseAuth) post(ctx context.Context, endpoint string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := f.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &failure)
		if failure.Error.Message == "" {
			failure.Error.Message = strings.TrimSpace(string(raw))
		}
		return fmt.Errorf("firebase %s: %s", response.Status, failure.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// RequestLink asks Firebase to mail a sign-in link to the address.
func (f *firebaseAuth) RequestLink(ctx context.Context, email string) error {
	endpoint := f.identityBase() + "/accounts:sendOobCode?key=" + url.QueryEscape(f.server.APIKey)
	return f.post(ctx, endpoint, map[string]any{
		"requestType": "EMAIL_SIGNIN",
		"email":       email,
		"continueUrl": f.server.URL,
	}, nil)
}

// EmulatorCode returns the newest pending sign-in code of an address, read
// from the emulator admin API.
func (f *firebaseAuth) EmulatorCode(ctx context.Context, email string) (string, error) {
	endpoint := fmt.Sprintf("%s/emulator/v1/projects/%s/oobCodes", strings.TrimRight(f.server.AuthEmulator, "/"), f.server.FirebaseProject)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	response, err := f.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var listing struct {
		OobCodes []struct {
			Email       string `json:"email"`
			RequestType string `json:"requestType"`
			OobCode     string `json:"oobCode"`
		} `json:"oobCodes"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return "", fmt.Errorf("read emulator codes: %w", err)
	}
	for index := len(listing.OobCodes) - 1; index >= 0; index-- {
		code := listing.OobCodes[index]
		if strings.EqualFold(code.Email, email) && code.RequestType == "EMAIL_SIGNIN" {
			return code.OobCode, nil
		}
	}
	return "", fmt.Errorf("the emulator holds no sign-in code for %s", email)
}

// SignInWithCode completes an email-link sign-in.
func (f *firebaseAuth) SignInWithCode(ctx context.Context, email, code string) (Tokens, error) {
	var reply struct {
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    string `json:"expiresIn"`
	}
	endpoint := f.identityBase() + "/accounts:signInWithEmailLink?key=" + url.QueryEscape(f.server.APIKey)
	if err := f.post(ctx, endpoint, map[string]string{"email": email, "oobCode": code}, &reply); err != nil {
		return Tokens{}, err
	}
	return newTokens(email, reply.IDToken, reply.RefreshToken, reply.ExpiresIn), nil
}

// EmulatorSignIn requests and consumes a link in one go (emulator only).
func (f *firebaseAuth) EmulatorSignIn(ctx context.Context, email string) (Tokens, error) {
	if err := f.RequestLink(ctx, email); err != nil {
		return Tokens{}, err
	}
	code, err := f.EmulatorCode(ctx, email)
	if err != nil {
		return Tokens{}, err
	}
	return f.SignInWithCode(ctx, email, code)
}

// Refresh renews the ID token from the refresh token.
func (f *firebaseAuth) Refresh(ctx context.Context, tokens Tokens) (Tokens, error) {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.RefreshToken}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, f.secureTokenURL()+"?key="+url.QueryEscape(f.server.APIKey), strings.NewReader(form.Encode()))
	if err != nil {
		return tokens, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := f.http.Do(request)
	if err != nil {
		return tokens, err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode >= 300 {
		return tokens, fmt.Errorf("refresh token rejected: %s", strings.TrimSpace(string(raw)))
	}
	var reply struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    string `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return tokens, err
	}
	return newTokens(tokens.Email, reply.IDToken, reply.RefreshToken, reply.ExpiresIn), nil
}

func newTokens(email, idToken, refreshToken, expiresIn string) Tokens {
	seconds := 3600
	fmt.Sscanf(expiresIn, "%d", &seconds)
	return Tokens{
		Email:        email,
		IDToken:      idToken,
		RefreshToken: refreshToken,
		Expires:      time.Now().Add(time.Duration(seconds) * time.Second),
	}
}

// oobCodeFromLink extracts the oobCode of a sign-in link pasted from a mail.
func oobCodeFromLink(link string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return "", err
	}
	code := parsed.Query().Get("oobCode")
	if code == "" {
		return "", errors.New("the link has no oobCode parameter")
	}
	return code, nil
}

// tokenSource keeps one account signed in.
type tokenSource struct {
	mu     sync.Mutex
	auth   *firebaseAuth
	tokens Tokens
	onSave func(Tokens)
}

func (t *tokenSource) Token(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tokens.IDToken == "" {
		return "", fmt.Errorf("%s is not signed in", t.tokens.Email)
	}
	if time.Until(t.tokens.Expires) > 5*time.Minute {
		return t.tokens.IDToken, nil
	}
	return t.refreshLocked(ctx)
}

// Invalidate forces a refresh after a 401.
func (t *tokenSource) Invalidate(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.refreshLocked(ctx)
}

func (t *tokenSource) refreshLocked(ctx context.Context) (string, error) {
	renewed, err := t.auth.Refresh(ctx, t.tokens)
	if err != nil {
		return "", err
	}
	t.tokens = renewed
	if t.onSave != nil {
		t.onSave(renewed)
	}
	return renewed.IDToken, nil
}
