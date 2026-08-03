// Package api contains the HTTP handlers and middleware used by the server.
package api

import "net/http"

const developmentOrigin = "http://localhost:5173"

// MapHandler serves bytes marshaled once during server startup.
func MapHandler(mapJSON []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(mapJSON)
	}
}

// WithCORS permits the Vite development server to request the local API.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", developmentOrigin)
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
