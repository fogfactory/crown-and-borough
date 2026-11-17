package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWithSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, production := range []bool{false, true} {
		rec := httptest.NewRecorder()
		WithSecurityHeaders(next, production).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		for _, k := range []string{"X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy", "X-Frame-Options", "Content-Security-Policy"} {
			if rec.Header().Get(k) == "" {
				t.Errorf("production=%v: missing %s", production, k)
			}
		}
		if rec.Header().Get("Content-Security-Policy-Report-Only") != "" {
			t.Errorf("production=%v: the policy must be enforced, not report-only", production)
		}
		if got := rec.Header().Get("Strict-Transport-Security") != ""; got != production {
			t.Errorf("production=%v: Strict-Transport-Security present=%v", production, got)
		}
	}
}

func TestContentSecurityPolicyStaysStrict(t *testing.T) {
	policy := ContentSecurityPolicy(true)
	for _, source := range strings.Fields(strings.ReplaceAll(policy, ";", " ")) {
		if source == "*" || source == "'unsafe-eval'" || source == "http:" || source == "https:" {
			t.Errorf("policy allows %s", source)
		}
	}
	for _, directive := range strings.Split(policy, "; ") {
		if strings.Contains(directive, "'unsafe-inline'") && !strings.HasPrefix(directive, "style-src ") {
			t.Errorf("'unsafe-inline' outside style-src: %s", directive)
		}
	}
	if strings.Contains(policy, "127.0.0.1") || strings.Contains(policy, "localhost") {
		t.Error("the production policy must not allow the loopback interface")
	}
	local := ContentSecurityPolicy(false)
	for _, directive := range []string{"connect-src", "frame-src"} {
		var found bool
		for _, d := range strings.Split(local, "; ") {
			if strings.HasPrefix(d, directive+" ") && strings.Contains(d, "http://127.0.0.1:*") && strings.Contains(d, "http://localhost:*") {
				found = true
			}
		}
		if !found {
			t.Errorf("the local policy must allow the emulators in %s", directive)
		}
	}
}

func TestIsProduction(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	if !IsProduction(false) || IsProduction(true) {
		t.Error("production must follow the development mode")
	}
	t.Setenv("FIRESTORE_EMULATOR_HOST", "127.0.0.1:8081")
	if IsProduction(false) {
		t.Error("a Firestore emulator is not production")
	}
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "127.0.0.1:9099")
	if IsProduction(false) {
		t.Error("an Auth emulator is not production")
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
	if _, ok := hosting["Content-Security-Policy-Report-Only"]; ok {
		t.Error("firebase.json must enforce the policy, not report it")
	}
}
