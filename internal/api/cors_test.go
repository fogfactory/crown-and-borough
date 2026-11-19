package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithCORSOptions(t *testing.T) {
	t.Setenv("PUBLIC_WEB_ORIGIN", "")
	t.Setenv("PUBLIC_APP_URL", "")
	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/games", nil)

	WithCORS(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("OPTIONS /api/games = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if nextCalled {
		t.Error("OPTIONS request reached the wrapped handler")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("OPTIONS response body = %q, want empty", recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != developmentOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, developmentOrigin)
	}
}

func TestWithCORSAllowsTheClientVersionHeader(t *testing.T) {
	handler := WithCORSMode(http.NotFoundHandler(), false)
	request := httptest.NewRequest(http.MethodOptions, "/api/games/G/orders", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, ClientVersionHeader) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want it to include %s", got, ClientVersionHeader)
	}
}

func TestWithCORSAllowOrigin(t *testing.T) {
	tests := []struct {
		name, webOrigin, appURL, origin, want string
	}{
		{"unknown origin gets only the configured one", "https://app.example.com", "", "https://evil.example.org", "https://app.example.com"},
		{"configured origin, path trimmed", "https://app.example.com/some/path", "", "https://app.example.com", "https://app.example.com"},
		{"no Origin falls back to the development origin", "", "", "", developmentOrigin},
		{"PUBLIC_APP_URL fallback", "", "https://game.example.com/play", "https://game.example.com", "https://game.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PUBLIC_WEB_ORIGIN", tt.webOrigin)
			t.Setenv("PUBLIC_APP_URL", tt.appURL)
			request := httptest.NewRequest(http.MethodGet, "/api/games", nil)
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })
			recorder := httptest.NewRecorder()

			WithCORSMode(next, false).ServeHTTP(recorder, request)

			if !nextCalled {
				t.Error("request did not reach the wrapped handler")
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
			if got := recorder.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
		})
	}
}

func TestWithCORSAdvertisesOnlyTheAllowedMethods(t *testing.T) {
	recorder := httptest.NewRecorder()
	WithCORSMode(http.NotFoundHandler(), false).ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/api/games", nil))

	got := recorder.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{"GET", "POST", "PUT", "OPTIONS"} {
		if !strings.Contains(got, method) {
			t.Errorf("Access-Control-Allow-Methods = %q, missing %s", got, method)
		}
	}
	for _, method := range []string{"DELETE", "PATCH"} {
		if strings.Contains(got, method) {
			t.Errorf("Access-Control-Allow-Methods = %q, must not include %s", got, method)
		}
	}
	if strings.Contains(recorder.Header().Get("Access-Control-Allow-Headers"), "X-Dev-Player") {
		t.Error("production mode must not advertise X-Dev-Player")
	}
}
