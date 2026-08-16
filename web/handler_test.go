package webassets

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHandlerServesIndex(t *testing.T) {
	server := NewHandler(testFiles())

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); !strings.Contains(got, `<div id="root"></div>`) {
		t.Fatalf("GET / body = %q, want the frontend root", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("GET / Cache-Control = %q, want no-cache", got)
	}
}

func TestHandlerServesStaticAsset(t *testing.T) {
	server := NewHandler(testFiles())

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "console.log('ok')" {
		t.Errorf("GET /assets/app.js body = %q, want asset contents", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("asset Cache-Control = %q, want immutable cache", got)
	}
}

func TestHandlerFallsBackToIndexForClientRoute(t *testing.T) {
	server := NewHandler(testFiles())

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/games/game-123", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /games/game-123 = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); !strings.Contains(got, `<div id="root"></div>`) {
		t.Errorf("client route body = %q, want index.html", got)
	}
}

func TestHandlerDoesNotRewriteReservedPathsOrMissingAssets(t *testing.T) {
	server := NewHandler(testFiles())
	for _, requestPath := range []string{"/api/missing", "/healthz/ready", "/assets/missing.js"} {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if recorder.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want %d", requestPath, recorder.Code, http.StatusNotFound)
		}
	}
}

func testFiles() fs.FS {
	return fstest.MapFS{
		"index.html":       &fstest.MapFile{Data: []byte("<div id=\"root\"></div>")},
		"assets/app.js":    &fstest.MapFile{Data: []byte("console.log('ok')")},
		"assets/style.css": &fstest.MapFile{Data: []byte("#root { color: red; }")},
	}
}
