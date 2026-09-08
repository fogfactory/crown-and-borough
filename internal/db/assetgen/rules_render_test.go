package assetgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLoadRulesRendersTerrainRations(t *testing.T) {
	dir := t.TempDir()
	template := "{{ration_terrain.plain}} {{ration_terrain.forest}} {{ration_terrain.hill}} " +
		"{{ration_terrain.mountain}} {{ration_terrain.swamp}} {{infra_rations_bonus}} " +
		"{{base_production}}\n"
	for _, name := range []string{playerRulesAsset, englishRulesAsset} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(template), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	balance := Balance{
		RationTerrain: map[models.Terrain]int{
			models.TerrainPlain:    3,
			models.TerrainForest:   2,
			models.TerrainHill:     4,
			models.TerrainMountain: 5,
			models.TerrainSwamp:    6,
		},
		InfraRationsBonus: 7,
		BaseProduction:    8,
	}
	rules, err := LoadRules(dir, balance)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}

	for _, language := range []string{"fr", "en"} {
		document, ok := rules.Document(language)
		if !ok {
			t.Fatalf("rules[%s] missing", language)
		}
		if got, want := string(document), "3 2 4 5 6 7 8\n"; got != want {
			t.Errorf("rules[%s] = %q, want %q", language, got, want)
		}
	}
}
