package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRequireClientVersion(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for name, test := range map[string]struct {
		server string
		method string
		header string
		want   int
	}{
		"older client write is refused": {server: "v1.2.3", method: http.MethodPost, header: "v1.2.2", want: http.StatusConflict},
		"client without header":         {server: "v1.2.3", method: http.MethodPost, want: http.StatusConflict},
		"matching client":               {server: "v1.2.3", method: http.MethodPost, header: "v1.2.3", want: http.StatusNoContent},
		"development client":            {server: "v1.2.3", method: http.MethodPost, header: "dev", want: http.StatusNoContent},
		"reads are never refused":       {server: "v1.2.3", method: http.MethodGet, want: http.StatusNoContent},
		"development server":            {server: "dev", method: http.MethodPost, header: "v9", want: http.StatusNoContent},
		"unversioned server":            {server: "", method: http.MethodPut, want: http.StatusNoContent},
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/api/games/G/orders", nil)
			if test.header != "" {
				request.Header.Set(ClientVersionHeader, test.header)
			}
			recorder := httptest.NewRecorder()
			RequireClientVersion(test.server, ok).ServeHTTP(recorder, request)
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d", recorder.Code, test.want)
			}
			if test.want == http.StatusConflict && !strings.Contains(recorder.Body.String(), `"error":"client_outdated"`) {
				t.Fatalf("body = %s, want error client_outdated", recorder.Body.String())
			}
		})
	}
}
