// Package api contains the HTTP handlers and middleware used by the server.
package api

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	developmentOrigin = "http://localhost:5173"
	// DefaultPlayers is the player count served when ?players is absent. The
	// development map variants are deliberately bounded so the in-memory cache
	// contains the supported two-to-sixteen-player maps only.
	DefaultPlayers = 4
	minimumPlayers = 2
	maximumPlayers = 16
)

// MapHandler resolves and serves a map for the requested player count.
func MapHandler(resolve func(players int) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		players, ok := requestedPlayers(r)
		if !ok {
			http.Error(w, "invalid players", http.StatusBadRequest)
			return
		}

		mapJSON, err := resolve(players)
		if err != nil {
			http.Error(w, "failed to resolve map", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mapJSON)
	}
}

func requestedPlayers(r *http.Request) (int, bool) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return 0, false
	}

	requested, ok := values["players"]
	if !ok {
		return DefaultPlayers, true
	}
	if len(requested) != 1 {
		return 0, false
	}

	players, err := strconv.Atoi(requested[0])
	if err != nil || players < minimumPlayers || players > maximumPlayers {
		return 0, false
	}
	return players, true
}

// WithCORS permits the Vite development server to request the local API and
// keeps the development identity header available for the legacy hotseat
// surface.
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
