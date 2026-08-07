// Package api contains the HTTP handlers and middleware used by the server.
package api

import (
	"net/http"
	"net/url"
	"strconv"
)

const (
	developmentOrigin = "http://localhost:5173"
	// DefaultPlayers is the player count served when ?players is absent. The
	// development map variants are deliberately bounded so the in-memory cache
	// contains the supported two-to-five-player maps only.
	DefaultPlayers = 4
	minimumPlayers = 2
	maximumPlayers = 5
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

// WithCORS permits the Vite development server to request the local API.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", developmentOrigin)
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
