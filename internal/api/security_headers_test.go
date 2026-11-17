package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWithSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, hsts := range []bool{false, true} {
		rec := httptest.NewRecorder()
		WithSecurityHeaders(next, hsts).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		for _, k := range []string{"X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy", "X-Frame-Options", "Content-Security-Policy-Report-Only"} {
			if rec.Header().Get(k) == "" {
				t.Errorf("hsts=%v: missing %s", hsts, k)
			}
		}
		if got := rec.Header().Get("Strict-Transport-Security") != ""; got != hsts {
			t.Errorf("hsts=%v: Strict-Transport-Security present=%v", hsts, got)
		}
	}
}

// TestSecurityHeadersMatchFirebaseHosting keeps the headers set by the Go
// server and the ones configured for Firebase Hosting identical.
func TestSecurityHeadersMatchFirebaseHosting(t *testing.T) {
	raw, err := os.ReadFile("../../firebase.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Hosting struct {
			Headers []struct {
				Headers []struct{ Key, Value string }
			}
		}
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	hosting := map[string]string{}
	for _, rule := range config.Hosting.Headers {
		for _, h := range rule.Headers {
			hosting[h.Key] = h.Value
		}
	}

	rec := httptest.NewRecorder()
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	WithSecurityHeaders(next, true).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for key := range rec.Header() {
		if got, want := hosting[key], rec.Header().Get(key); got != want {
			t.Errorf("%s: firebase.json has %q, server sets %q", key, got, want)
		}
	}
}
