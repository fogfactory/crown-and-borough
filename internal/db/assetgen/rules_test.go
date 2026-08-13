package assetgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	french := "# Règles françaises\n"
	english := "# English rules\n"
	if err := os.WriteFile(filepath.Join(dir, playerRulesAsset), []byte(french), 0o644); err != nil {
		t.Fatalf("write French rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, englishRulesAsset), []byte(english), 0o644); err != nil {
		t.Fatalf("write English rules: %v", err)
	}

	rules, err := LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}
	if got, ok := rules.Document(""); !ok || string(got) != french {
		t.Errorf("default document = %q, %t; want %q, true", got, ok, french)
	}
	if got, ok := rules.Document("EN"); !ok || string(got) != english {
		t.Errorf("English document = %q, %t; want %q, true", got, ok, english)
	}
	if _, ok := rules.Document("de"); ok {
		t.Error("German document exists, want missing translation")
	}

	got, ok := rules.Document("fr")
	if !ok {
		t.Fatal("French document missing")
	}
	got[0] = 'X'
	unchanged, _ := rules.Document("fr")
	if string(unchanged) != french {
		t.Errorf("stored document changed through returned bytes: %q", unchanged)
	}
}

func TestLoadRulesRequiresNonEmptyFrenchDocument(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing", want: playerRulesAsset},
		{name: "empty", content: " \n\t", want: "empty file"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if test.content != "" {
				if err := os.WriteFile(filepath.Join(dir, playerRulesAsset), []byte(test.content), 0o644); err != nil {
					t.Fatalf("write French rules: %v", err)
				}
			}
			if _, err := LoadRules(dir); err == nil {
				t.Fatal("LoadRules = nil error, want error")
			} else if !strings.Contains(err.Error(), test.want) {
				t.Errorf("error %q does not contain %q", err, test.want)
			}
		})
	}
}

func TestLoadRulesAllowsMissingTranslation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, playerRulesAsset), []byte("# Français\n"), 0o644); err != nil {
		t.Fatalf("write French rules: %v", err)
	}

	rules, err := LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}
	if _, ok := rules.Document("en"); ok {
		t.Error("English document exists without an asset")
	}
}

func TestLoadRealRules(t *testing.T) {
	rules, err := LoadRules("../../../assets")
	if err != nil {
		t.Fatalf("LoadRules(real assets) = %v", err)
	}
	document, ok := rules.Document("fr")
	if !ok || !strings.Contains(string(document), "# Règles du jeu") {
		t.Errorf("French document = %q, %t", document, ok)
	}
	english, ok := rules.Document("en")
	if !ok || !strings.Contains(string(english), "# Game Rules") {
		t.Errorf("English document = %q, %t", english, ok)
	}
	for _, heading := range []string{"## 4. Aide-mémoire des ordres", "## 5. Ordres d'hiver"} {
		if !strings.Contains(string(document), heading) {
			t.Errorf("French document does not contain %q", heading)
		}
	}
	for _, heading := range []string{"## 4. Order Cheat Sheet", "## 5. Winter Orders"} {
		if !strings.Contains(string(english), heading) {
			t.Errorf("English document does not contain %q", heading)
		}
	}
}
