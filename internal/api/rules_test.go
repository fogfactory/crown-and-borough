package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

func TestRulesHandlerServesDefaultAndRequestedLanguage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "regles-joueurs.md"), []byte("# Français\n"), 0o644); err != nil {
		t.Fatalf("write French rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "regles-joueurs.en.md"), []byte("# English\n"), 0o644); err != nil {
		t.Fatalf("write English rules: %v", err)
	}
	rules, err := assetgen.LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}

	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "default", path: "/api/rules", want: "# Français\n"},
		{name: "English", path: "/api/rules?lang=en", want: "# English\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			RulesHandler(rules).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))

			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", test.path, recorder.Code, http.StatusOK)
			}
			if got := recorder.Header().Get("Content-Type"); got != "text/markdown; charset=utf-8" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := recorder.Body.String(); got != test.want {
				t.Errorf("body = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRulesHandlerRejectsUnknownLanguage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "regles-joueurs.md"), []byte("# Français\n"), 0o644); err != nil {
		t.Fatalf("write French rules: %v", err)
	}
	rules, err := assetgen.LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}

	recorder := httptest.NewRecorder()
	RulesHandler(rules).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/rules?lang=de", nil))
	if recorder.Code != http.StatusNotFound {
		t.Errorf("GET unknown rules language = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
