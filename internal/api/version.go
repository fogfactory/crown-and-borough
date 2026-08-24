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
