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
	request.Header.Set("Origin", developmentOrigin)

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

const testOrigin = "https://app.example.com"

// preflight runs an OPTIONS request from origin through the CORS middleware
// configured for testOrigin.
func preflight(t *testing.T, development bool, origin string) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("PUBLIC_WEB_ORIGIN", testOrigin)
	request := httptest.NewRequest(http.MethodOptions, "/api/games", nil)
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()
	WithCORSMode(http.NotFoundHandler(), development).ServeHTTP(recorder, request)
	return recorder
}

func TestWithCORSAllowsTheClientVersionHeader(t *testing.T) {
	recorder := preflight(t, false, testOrigin)
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, ClientVersionHeader) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want it to include %s", got, ClientVersionHeader)
	}
}

func TestWithCORSAllowOrigin(t *testing.T) {
	tests := []struct {
		name, webOrigin, appURL, origin string
		development                     bool
		want                            string
	}{
		{"unknown origin gets no CORS header", "https://app.example.com", "", "https://evil.example.org", false, ""},
		{"unknown origin is not reflected in development", "", "", "https://evil.example.org", true, ""},
		{"origin differing by scheme is refused", "https://app.example.com", "", "http://app.example.com", false, ""},
		{"origin differing by port is refused", "https://app.example.com", "", "https://app.example.com:8443", false, ""},
		{"null origin is refused", "https://app.example.com", "", "null", false, ""},
		{"configured origin, path trimmed", "https://app.example.com/some/path", "", "https://app.example.com", false, "https://app.example.com"},
		{"no Origin gets no CORS header", "https://app.example.com", "", "", false, ""},
		{"PUBLIC_WEB_ORIGIN wins over PUBLIC_APP_URL", "https://web.example.com", "https://game.example.com", "https://game.example.com", false, ""},
		{"PUBLIC_APP_URL fallback", "", "https://game.example.com/play", "https://game.example.com", false, "https://game.example.com"},
		{"development origin is allowed in development", "", "", developmentOrigin, true, developmentOrigin},
		{"development origin is never a production fallback", "", "", developmentOrigin, false, ""},
		{"development origin is refused when another is configured", "https://app.example.com", "", developmentOrigin, false, ""},
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

			WithCORSMode(next, tt.development).ServeHTTP(recorder, request)

			if !nextCalled {
				t.Error("request did not reach the wrapped handler")
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
			if tt.want == "" && recorder.Header().Get("Access-Control-Allow-Methods") != "" {
				t.Error("a refused origin must get no CORS header")
			}
			if got := recorder.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
		})
	}
}

func TestWithCORSPreflightFromUnknownOrigin(t *testing.T) {
	recorder := preflight(t, false, "https://evil.example.org")
	if recorder.Code != http.StatusNoContent {
		t.Errorf("preflight = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	for name := range recorder.Header() {
		if strings.HasPrefix(name, "Access-Control-") {
			t.Errorf("preflight from an unknown origin set %s", name)
		}
	}
}

func TestWithCORSAdvertisesOnlyTheAllowedMethods(t *testing.T) {
	recorder := preflight(t, false, testOrigin)

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
