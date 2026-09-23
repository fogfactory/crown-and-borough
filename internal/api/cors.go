// Package api contains the HTTP handlers and middleware used by the server.
package api

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// developmentOrigin is the Vite development server allowed by default.
const developmentOrigin = "http://localhost:5173"

// WithCORS permits the Vite development server to request the local API and
// keeps the development identity header available.
func WithCORS(next http.Handler) http.Handler {
	return WithCORSMode(next, true)
}

// WithCORSMode is the production-safe variant of WithCORS. The development
// identity header is advertised only when the caller explicitly mounts the
// development API.
func WithCORSMode(next http.Handler, allowDevelopmentHeaders bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := corsOrigin(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		allowedHeaders := "Authorization, Content-Type"
		if allowDevelopmentHeaders {
			allowedHeaders += ", X-Dev-Player"
		}
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsOrigin(requestOrigin string) string {
	configured := strings.TrimSpace(os.Getenv("PUBLIC_WEB_ORIGIN"))
	if configured == "" {
		configured = strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
	}
	if configured == "" {
		configured = developmentOrigin
	}
	if parsed, err := url.Parse(configured); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		configured = parsed.Scheme + "://" + parsed.Host
	}
	if strings.TrimSpace(requestOrigin) == configured {
		return requestOrigin
	}
	return configured
}
