package api

import (
	"net/http"
	"strings"
)

const defaultVersion = "dev"

// VersionHandler exposes the build version used by the running application.
func VersionHandler(version string) http.HandlerFunc {
	version = strings.TrimSpace(version)
	if version == "" {
		version = defaultVersion
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, struct {
			Version string `json:"version"`
		}{Version: version})
	}
}

// ClientVersionHeader carries the build version of the client.
const ClientVersionHeader = "X-Client-Version"

// RequireClientVersion refuses the writes (every method but GET, HEAD and
// OPTIONS) of a client that does not declare the running build, with 409
// client_outdated: a page left open across a deploy has to be reloaded instead
// of sending requests in a format the server no longer reads the same way.
// Development builds ("dev") are not checked.
func RequireClientVersion(version string, next http.Handler) http.Handler {
	version = strings.TrimSpace(version)
	if version == "" || version == defaultVersion {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			client := strings.TrimSpace(r.Header.Get(ClientVersionHeader))
			if client != defaultVersion && client != version {
				writeAPIError(w, http.StatusConflict, "client_outdated", "the page runs an older version of the game: reload it")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
