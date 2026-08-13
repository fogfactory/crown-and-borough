package api

import (
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

// RulesHandler serves the player-facing rules document loaded from the
// versioned assets directory. French remains the API default for callers that
// do not provide a language; the web client sends its persisted choice.
func RulesHandler(rules assetgen.Rules) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		document, ok := rules.Document(r.URL.Query().Get("lang"))
		if !ok {
			http.Error(w, "rules translation not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(document)
	}
}
