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
	allowed := configuredOrigin(allowDevelopmentHeaders)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		// A single origin is allowed: the SPA is served from the API origin
		// (or through a Hosting rewrite), so CORS only matters for the Vite
		// development server. Any other origin gets no CORS header at all and
		// the browser refuses to expose the response to its scripts. CORS is
		// not authentication: every protected route checks the token itself.
		if origin := matchOrigin(r.Header.Get("Origin"), allowed); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			allowedHeaders := "Authorization, Content-Type, " + ClientVersionHeader
			if allowDevelopmentHeaders {
				allowedHeaders += ", X-Dev-Player"
			}
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// configuredOrigin returns the one origin allowed to call the API, or "" when
// none is configured. The development origin is only a fallback in development
// mode, so it can never be enabled by a missing variable in production. It is
// resolved once, when the middleware is built.
func configuredOrigin(development bool) string {
	configured := strings.TrimSpace(os.Getenv("PUBLIC_WEB_ORIGIN"))
	if configured == "" {
		configured = strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
	}
	if configured == "" && development {
		configured = developmentOrigin
	}
	if parsed, err := url.Parse(configured); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		configured = parsed.Scheme + "://" + parsed.Host
	}
	return configured
}

// matchOrigin returns requestOrigin when it is exactly the allowed origin, and
// "" otherwise.
func matchOrigin(requestOrigin, allowed string) string {
	if allowed != "" && strings.TrimSpace(requestOrigin) == allowed {
		return allowed
	}
	return ""
}
