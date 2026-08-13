package assetgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	validCommunes = "code;nom;terrain\n" +
		"ROS;Rosemont;plain\n" +
		"BCL;Boisclair;forest\n" +
		"BRU;Bruyères;hill\n" +
		"MDO;Mont-Dore;mountain\n" +
		"FOU;Fougères;swamp\n" +
		"BLV;Belval;any\n"
	validPrenoms = "code;nom\n" +
		"GUI;Guillaume\n" +
		"ADE;Adélaïde\n" +
		"MAH;Mahaut\n"
	validBalance = `# The loader accepts YAML documentation comments.
base_production: 1
supply_range: 3
depot_range_bonus: 2
infra_rations_bonus: 2
cost_base: 2
pillage_bonus: 2
noble_command_bonus: 1
castle_defense_bonus: 1
ration_terrain:
  plain: 1
  forest: 1
  hill: 1
  mountain: 0
  swamp: 0
winter_stock_divisor: 2
village_stock_cap: 1
castle_stock_cap: 2
costs:
  castle: 10
  mill: 3
  troop: 1
  noble: 2
  supply_depot: 3
  liberation: 0
starting_nobles: 1
starting_troops: 1
starting_resources: 10`
)

func writeAssets(t *testing.T, dir, communes, prenoms string) {
	t.Helper()
	if communes == "" {
		communes = validCommunes
	}
	if prenoms == "" {
		prenoms = validPrenoms
	}
	for name, content := range map[string]string{
		"communes.csv": communes,
		"prenoms.csv":  prenoms,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func writeBalance(t *testing.T, dir, content string) {
	t.Helper()
	if content == "" {
		content = validBalance
	}
	if err := os.WriteFile(filepath.Join(dir, "balance.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write balance.yaml: %v", err)
	}
}

func TestLoadRealBalance(t *testing.T) {
	balance, err := LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("LoadBalance(real asset) = %v", err)
	}
	if balance.BaseProduction != 1 || balance.DepotRangeBonus != 2 || balance.NobleCommandBonus != 1 || balance.WinterStockDivisor != 2 || balance.VillageStockCap != 1 || balance.CastleStockCap != 2 || balance.Costs.Castle != 10 || balance.Costs.Liberation != 0 {
		t.Errorf("loaded costs = %#v / %#v", balance, balance.Costs)
	}
	if len(balance.FirstNames) < 100 {
		t.Errorf("len(FirstNames) = %d, want >= 100", len(balance.FirstNames))
	}
}

func TestLoadBalanceValid(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "")
	writeBalance(t, dir, "")

	balance, err := LoadBalance(dir)
	if err != nil {
		t.Fatalf("LoadBalance(valid asset) = %v", err)
	}
	if balance.SupplyRange != 3 || balance.NobleCommandBonus != 1 || balance.RationTerrain["swamp"] != 0 {
		t.Errorf("loaded balance = %#v", balance)
	}
	if len(balance.FirstNames) != 3 {
		t.Errorf("len(FirstNames) = %d, want 3", len(balance.FirstNames))
	}
}

func TestLoadBalanceInvalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing explicit zero cost",
			content: strings.Replace(validBalance, "  supply_depot: 3\n", "", 1),
			want:    "costs.supply_depot",
		},
		{
			name:    "missing noble command bonus",
			content: strings.Replace(validBalance, "noble_command_bonus: 1\n", "", 1),
			want:    "noble_command_bonus",
		},
		{
			name:    "missing terrain value",
			content: strings.Replace(validBalance, "  mountain: 0\n  swamp: 0\n", "  mountain: 0\n", 1),
			want:    "ration_terrain.swamp",
		},
		{
			name:    "unknown setting",
			content: strings.Replace(validBalance, "base_production: 1\n", "unknown_setting: 1\nbase_production: 1\n", 1),
			want:    "unknown_setting",
		},
		{
			name:    "malformed YAML",
			content: "base_production: [\n",
			want:    "invalid YAML",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeAssets(t, dir, "", "")
			writeBalance(t, dir, tt.content)

			if _, err := LoadBalance(dir); err == nil {
				t.Fatal("LoadBalance = nil error, want error")
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestLoadRealAssets(t *testing.T) {
	assets, err := Load("../../../assets")
	if err != nil {
		t.Fatalf("Load(real assets) = %v", err)
	}
	if len(assets.Communes) < 400 {
		t.Errorf("len(Communes) = %d, want >= 400", len(assets.Communes))
	}
	if len(assets.Prenoms) < 100 {
		t.Errorf("len(Prenoms) = %d, want >= 100", len(assets.Prenoms))
	}

	seenTerrains := make(map[string]bool, len(communeTerrains))
	for _, a := range assets.Communes {
		if !isTrigram(a.Code) {
			t.Errorf("commune %q: invalid code %q", a.Name, a.Code)
		}
		if a.Name == "" {
			t.Errorf("commune with empty name, code %q", a.Code)
		}
		if !validTerrains[a.Terrain] {
			t.Errorf("commune %q: invalid terrain %q", a.Name, a.Terrain)
		}
		seenTerrains[a.Terrain] = true
	}
	for _, terrain := range communeTerrains {
		if !seenTerrains[terrain] {
			t.Errorf("no commune with terrain %q", terrain)
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

	assertNoDuplicateCommunes(t, assets.Communes)
	assertNoDuplicates(t, assets.Prenoms)
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

func assertNoDuplicateCommunes(t *testing.T, communes []Commune) {
	t.Helper()
	seenCodes := make(map[string]bool)
	seenNames := make(map[string]bool)
	for _, commune := range communes {
		if seenCodes[commune.Code] {
			t.Errorf("duplicate code %q", commune.Code)
		}
		if seenNames[commune.Name] {
			t.Errorf("duplicate name %q", commune.Name)
		}
		seenCodes[commune.Code] = true
		seenNames[commune.Name] = true
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	writeAssets(t, dir, "", "")

	assets, err := Load(dir)
	if err != nil {
		t.Fatalf("Load(valid assets) = %v", err)
	}
	if len(assets.Communes) != 6 || len(assets.Prenoms) != 3 {
		t.Errorf("counts = %d/%d, want 6/3", len(assets.Communes), len(assets.Prenoms))
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
	writeAssets(t, dir, "", "")
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
	writeAssets(t, dir, "", "")
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
	writeAssets(t, dir, "code;nom;terrain\n", "")

	if _, err := Load(dir); err == nil {
		t.Fatal("Load(header-only communes.csv) = nil error, want error")
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name     string
		communes string
		prenoms  string
	}{
		{
			name:     "duplicate code",
			communes: "code;nom;terrain\nVIL;Villeneuve;plain\nVIL;Villefort;forest\n",
		},
		{
			name:     "duplicate name",
			communes: "code;nom;terrain\nVIL;Villeneuve;plain\nVFO;Villeneuve;forest\n",
		},
		{
			name:     "invalid terrain",
			communes: "code;nom;terrain\nVIL;Villeneuve;desert\n",
		},
		{
			name:     "malformed row",
			communes: "code;nom;terrain\nVIL;Villeneuve;plain;extra\n",
		},
		{
			name:     "short row",
			communes: "code;nom;terrain\nVIL;Villeneuve\n",
		},
		{
			name:     "lowercase code",
			communes: "code;nom;terrain\nvil;Villeneuve;plain\n",
		},
		{
			name:     "two-letter code",
			communes: "code;nom;terrain\nVI;Villeneuve;plain\n",
		},
		{
			name:     "four-letter code",
			communes: "code;nom;terrain\nVILL;Villeneuve;plain\n",
		},
		{
			name:     "empty name",
			communes: "code;nom;terrain\nVIL;;plain\n",
		},
		{
			name:     "empty terrain",
			communes: "code;nom;terrain\nVIL;Villeneuve;\n",
		},
		{
			name: "missing affinity",
			communes: "code;nom;terrain\n" +
				"ROS;Rosemont;plain\n" +
				"VIL;Villeneuve;plain\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeAssets(t, dir, tt.communes, tt.prenoms)

			_, err := Load(dir)
			if err == nil {
				t.Fatal("Load = nil error, want error")
			}
			if !strings.Contains(err.Error(), "communes.csv") {
				t.Errorf("error %q does not mention communes.csv", err)
			}
		})
	}
}
