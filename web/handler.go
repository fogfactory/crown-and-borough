package webassets

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// NewHandler serves static frontend files and falls back to index.html for
// client-side routes. API and health paths are deliberately never rewritten
// to the SPA.
func NewHandler(files fs.FS) http.Handler {
	if files == nil {
		panic("webassets: a filesystem is required")
	}

	static := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if reservedPath(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		name := requestFileName(r.URL.Path)
		if name != "" && fileExists(files, name) {
			setCacheHeaders(w, name)
			if name == "index.html" {
				serveIndex(w, r, files)
				return
			}
			static.ServeHTTP(w, r)
			return
		}
		if name != "" && path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		serveIndex(w, r, files)
	})
}

// NewEmbeddedHandler serves the Vite output compiled into the Go binary.
func NewEmbeddedHandler() http.Handler {
	return NewHandler(EmbeddedFS())
}

func reservedPath(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/healthz" || strings.HasPrefix(requestPath, "/healthz/")
}

func requestFileName(requestPath string) string {
	clean := path.Clean("/" + requestPath)
	return strings.TrimPrefix(clean, "/")
}

func fileExists(files fs.FS, name string) bool {
	file, err := files.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	return err == nil && !info.IsDir()
}

func setCacheHeaders(w http.ResponseWriter, name string) {
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
}

func serveIndex(w http.ResponseWriter, r *http.Request, files fs.FS) {
	data, err := fs.ReadFile(files, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(data))
}
