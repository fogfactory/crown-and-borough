package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/api"
	webassets "github.com/fogfactory/crown-and-borough/web"
)

var requiredSecurityHeaders = []string{
	"X-Content-Type-Options",
	"Referrer-Policy",
	"Permissions-Policy",
	"X-Frame-Options",
	"Content-Security-Policy",
}

func requireSecurityHeaders(t *testing.T, label string, rec *httptest.ResponseRecorder, hsts bool) {
	t.Helper()
	for _, name := range requiredSecurityHeaders {
		if rec.Header().Get(name) == "" {
			t.Errorf("%s (%d): missing %s", label, rec.Code, name)
		}
	}
	if got := rec.Header().Get("Strict-Transport-Security") != ""; got != hsts {
		t.Errorf("%s (%d): Strict-Transport-Security present = %v, want %v", label, rec.Code, got, hsts)
	}
}

func TestSecurityHeadersOnEveryKindOfResponse(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	limit := api.RateLimitConfig{Requests: 1, Window: time.Hour}
	for _, development := range []bool{false, true} {
		// Production only allows the configured origin; development falls
		// back to the Vite origin.
		origin := "http://localhost:5173"
		if !development {
			origin = "https://app.example.com"
		}
		t.Setenv("PUBLIC_WEB_ORIGIN", "")
		if !development {
			t.Setenv("PUBLIC_WEB_ORIGIN", origin)
		}
		handler := httpHandler(newTestServer(t, development), limit, development)
		hsts := !development
		cases := []struct {
			label, method, path string
			want                int
		}{
			{"api success", http.MethodGet, "/api/version", http.StatusOK},
			{"api not found", http.MethodGet, "/api/unknown", http.StatusNotFound},
			{"preflight", http.MethodOptions, "/api/games", http.StatusNoContent},
		}
		for _, tc := range cases {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Origin", origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("dev=%v %s: status %d, want %d", development, tc.label, rec.Code, tc.want)
			}
			requireSecurityHeaders(t, tc.label, rec, hsts)
			if tc.method == http.MethodOptions && rec.Header().Get("Access-Control-Allow-Origin") == "" {
				t.Errorf("dev=%v preflight lost its CORS headers", development)
			}
		}

		// The first mutating request spends the only token, the second is limited.
		var last *httptest.ResponseRecorder
		for range 2 {
			last = httptest.NewRecorder()
			handler.ServeHTTP(last, httptest.NewRequest(http.MethodPost, "/api/games", nil))
		}
		if last.Code != http.StatusTooManyRequests {
			t.Fatalf("dev=%v second POST /api/games = %d, want 429", development, last.Code)
		}
		requireSecurityHeaders(t, "rate limited", last, hsts)
	}
}

func TestSecurityHeadersOnFrontendPages(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	if _, err := fs.Stat(webassets.EmbeddedFS(), "index.html"); err != nil && os.Getenv("CI") == "" {
		t.Skip("frontend not built; run make web-build (always enforced in CI)")
	}
	handler := httpHandler(newTestServer(t, false), api.RateLimitConfig{}, false)
	for _, path := range []string{"/", "/games/game-123"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, rec.Code)
		}
		requireSecurityHeaders(t, "GET "+path, rec, true)
	}
}
