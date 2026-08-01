package assetgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	validCommunes = "code;nom\n" +
		"ROS;Rosemont\n" +
		"VIL;Villeneuve\n" +
		"GRI;Griffecourt\n"
	validPrenoms = "code;nom\n" +
		"GUI;Guillaume\n" +
		"ADE;Adélaïde\n" +
		"MAH;Mahaut\n"
	validQuals = "prefix;qualificatif;terrain\n" +
		"F;Forêt;forest\n" +
		"H;Marches;any\n" +
		"P;Plaines;plain\n"
)

func writeAssets(t *testing.T, dir, communes, prenoms, quals string) {
	t.Helper()
	if communes == "" {
		communes = validCommunes
	}
	if prenoms == "" {
		prenoms = validPrenoms
	}
	if quals == "" {
		quals = validQuals
	}
	for name, content := range map[string]string{
		"communes.csv":      communes,
		"prenoms.csv":       prenoms,
		"qualificatifs.csv": quals,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestLoadRealAssets(t *testing.T) {
	assets, err := Load("../../../assets")
	if err != nil {
		t.Fatalf("Load(real assets) = %v", err)
	}
	if len(assets.Communes) < 180 {
		t.Errorf("len(Communes) = %d, want >= 180", len(assets.Communes))
	}
	if len(assets.Prenoms) < 100 {
		t.Errorf("len(Prenoms) = %d, want >= 100", len(assets.Prenoms))
	}
	if len(assets.Qualificatifs) < 8 {
		t.Errorf("len(Qualificatifs) = %d, want >= 8", len(assets.Qualificatifs))
	}

	for _, a := range assets.Communes {
		if !isTrigram(a.Code) {
			t.Errorf("commune %q: invalid code %q", a.Name, a.Code)
		}
		if a.Name == "" {
			t.Errorf("commune with empty name, code %q", a.Code)
		}
	}
	for _, a := range assets.Prenoms {
		if !isTrigram(a.Code) {
			t.Errorf("prénom %q: invalid code %q", a.Name, a.Code)
		}
		if a.Name == "" {
			t.Errorf("prénom with empty name, code %q", a.Code)
		}
	}
	for _, q := range assets.Qualificatifs {
		if !isPrefix(q.Prefix) {
			t.Errorf("qualificatif %q: invalid prefix %q", q.Name, q.Prefix)
		}
		if !validTerrains[q.Terrain] {
			t.Errorf("qualificatif %q: invalid terrain %q", q.Name, q.Terrain)
		}
	}

	assertNoDuplicates(t, assets.Communes)
	assertNoDuplicates(t, assets.Prenoms)
	seen := make(map[string]bool)
	for _, q := range assets.Qualificatifs {
		if seen[q.Prefix] {
			t.Errorf("qualificatif %q: duplicate prefix %q", q.Name, q.Prefix)
		}
		seen[q.Prefix] = true
	}
}

func assertNoDuplicates(t *testing.T, assets []Asset) {
	t.Helper()
	seenCodes := make(map[string]bool)
	seenNames := make(map[string]bool)
	for _, a := range assets {
		if seenCodes[a.Code] {
			t.Errorf("duplicate code %q", a.Code)
		}
		if seenNames[a.Name] {
			t.Errorf("duplicate name %q", a.Name)
		}
		seenCodes[a.Code] = true
		seenNames[a.Name] = true
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "", "")

	assets, err := Load(dir)
	if err != nil {
		t.Fatalf("Load(valid assets) = %v", err)
	}
	if len(assets.Communes) != 3 || len(assets.Prenoms) != 3 || len(assets.Qualificatifs) != 3 {
		t.Errorf("counts = %d/%d/%d, want 3/3/3",
			len(assets.Communes), len(assets.Prenoms), len(assets.Qualificatifs))
	}
}

func TestLoadEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Fatal("Load(empty dir) = nil error, want error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "", "")
	if err := os.Remove(filepath.Join(dir, "prenoms.csv")); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatal("Load(missing prenoms.csv) = nil error, want error")
	} else if !strings.Contains(err.Error(), "prenoms.csv") {
		t.Errorf("error %q does not mention prenoms.csv", err)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "", "")
	if err := os.WriteFile(filepath.Join(dir, "communes.csv"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatal("Load(empty communes.csv) = nil error, want error")
	} else if !strings.Contains(err.Error(), "communes.csv") {
		t.Errorf("error %q does not mention communes.csv", err)
	}
}

func TestLoadHeaderOnly(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "code;nom\n", "", "")

	if _, err := Load(dir); err == nil {
		t.Fatal("Load(header-only communes.csv) = nil error, want error")
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name     string
		communes string
		prenoms  string
		quals    string
	}{
		{
			name:     "duplicate code",
			communes: "code;nom\nVIL;Villeneuve\nVIL;Villefort\n",
		},
		{
			name:     "duplicate name",
			communes: "code;nom\nVIL;Villeneuve\nVFO;Villeneuve\n",
		},
		{
			name:  "duplicate prefix",
			quals: "prefix;qualificatif;terrain\nF;Forêt;forest\nF;Falaise;mountain\n",
		},
		{
			name:  "invalid terrain",
			quals: "prefix;qualificatif;terrain\nF;Forêt;desert\n",
		},
		{
			name:     "malformed row",
			communes: "code;nom\nVIL;Villeneuve;extra\n",
		},
		{
			name:     "short row",
			communes: "code;nom\nVIL\n",
		},
		{
			name:     "lowercase code",
			communes: "code;nom\nvil;Villeneuve\n",
		},
		{
			name:     "two-letter code",
			communes: "code;nom\nVI;Villeneuve\n",
		},
		{
			name:     "four-letter code",
			communes: "code;nom\nVILL;Villeneuve\n",
		},
		{
			name:     "empty name",
			communes: "code;nom\nVIL;\n",
		},
		{
			name:  "invalid prefix",
			quals: "prefix;qualificatif;terrain\nFF;Forêt;forest\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeAssets(t, dir, tt.communes, tt.prenoms, tt.quals)

			_, err := Load(dir)
			if err == nil {
				t.Fatal("Load = nil error, want error")
			}
			wantFile := "communes.csv"
			if tt.quals != "" && tt.communes == "" && tt.prenoms == "" {
				wantFile = "qualificatifs.csv"
			}
			if !strings.Contains(err.Error(), wantFile) {
				t.Errorf("error %q does not mention %s", err, wantFile)
			}
		})
	}
}
