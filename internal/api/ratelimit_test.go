package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func limited(t *testing.T, cfg RateLimitConfig) http.Handler {
	t.Helper()
	return WithRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), cfg)
}

func do(h http.Handler, method, path, remote, forwarded string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remote
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRateLimitQuotaAndRefill(t *testing.T) {
	now := time.Unix(1000, 0)
	h := limited(t, RateLimitConfig{Requests: 2, Window: time.Minute, Now: func() time.Time { return now }})
	for i := 0; i < 2; i++ {
		if rec := do(h, "POST", "/api/games", "1.1.1.1:1", ""); rec.Code != http.StatusNoContent {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	rec := do(h, "POST", "/api/games", "1.1.1.1:2", "")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "30" {
		t.Fatalf("got %d retry-after %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	if rec := do(h, "POST", "/api/games", "2.2.2.2:1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("other client limited: %d", rec.Code)
	}
	now = now.Add(30 * time.Second)
	if rec := do(h, "POST", "/api/games", "1.1.1.1:3", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("token not refilled: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/games", "1.1.1.1:3", ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after refill spent: %d", rec.Code)
	}
}

func TestRateLimitScope(t *testing.T) {
	h := limited(t, RateLimitConfig{Requests: 1, Window: time.Hour})
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/games"}, {"GET", "/healthz"}, {"POST", "/healthz"}, {"GET", "/assets/app.js"}, {"GET", "/api/games/x/state"},
	} {
		for i := 0; i < 3; i++ {
			if rec := do(h, c.method, c.path, "1.1.1.1:1", ""); rec.Code != http.StatusNoContent {
				t.Fatalf("%s %s limited", c.method, c.path)
			}
		}
	}
	do(h, "GET", "/api/auth/me", "3.3.3.3:1", "")
	if rec := do(h, "GET", "/api/auth/me", "3.3.3.3:1", ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("auth route not limited: %d", rec.Code)
	}
}

func TestRateLimitDisabled(t *testing.T) {
	h := limited(t, RateLimitConfig{})
	for i := 0; i < 5; i++ {
		if rec := do(h, "POST", "/api/games", "1.1.1.1:1", ""); rec.Code != http.StatusNoContent {
			t.Fatal("disabled limiter limited")
		}
	}
}

func TestClientIPTrustsOnlyProxyHops(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:99"
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 9.9.9.9")
	if got := clientIP(r, 0); got != "10.0.0.1" {
		t.Fatalf("no proxy: %s", got)
	}
	if got := clientIP(r, 1); got != "9.9.9.9" {
		t.Fatalf("one hop: %s", got)
	}
	if got := clientIP(r, 3); got != "10.0.0.1" {
		t.Fatalf("shortfall: %s", got)
	}
}
