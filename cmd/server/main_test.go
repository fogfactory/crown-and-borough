package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	webassets "github.com/fogfactory/crown-and-borough/web"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newTestServer(t, false).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestVersion(t *testing.T) {
	t.Setenv("APP_VERSION", "")
	recorder := httptest.NewRecorder()
	newTestServer(t, false).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/version = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if response.Version != "dev" {
		t.Errorf("version = %q, want dev", response.Version)
	}
}

func TestServerServesEmbeddedFrontendAndClientRoutes(t *testing.T) {
	if _, err := fs.Stat(webassets.EmbeddedFS(), "index.html"); err != nil && os.Getenv("CI") == "" {
		t.Skip("frontend not built; run make web-build (always enforced in CI)")
	}
	server := newTestServer(t, true)

	for _, requestPath := range []string{"/", "/games/game-123"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want %d", requestPath, recorder.Code, http.StatusOK)
		}
		if !strings.Contains(recorder.Body.String(), `<div id="root"></div>`) {
			t.Errorf("GET %s did not return the embedded frontend", requestPath)
		}
	}
}
