package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionHandlerServesConfiguredVersion(t *testing.T) {
	recorder := httptest.NewRecorder()
	VersionHandler(" v0.3.1 ").ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/version", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/version = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var response struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if response.Version != "v0.3.1" {
		t.Errorf("version = %q, want v0.3.1", response.Version)
	}
}

func TestVersionHandlerDefaultsEmptyVersion(t *testing.T) {
	recorder := httptest.NewRecorder()
	VersionHandler(" ").ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/version", nil),
	)

	if got := recorder.Body.String(); got != `{"version":"dev"}
` {
		t.Errorf("empty version response = %q, want dev", got)
	}
}
