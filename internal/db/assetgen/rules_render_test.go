package assetgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLoadRulesRendersBalanceValues(t *testing.T) {
	dir := t.TempDir()
	template := "{{ration_terrain.plain}} {{ration_terrain.forest}} {{ration_terrain.hill}} " +
		"{{ration_terrain.mountain}} {{ration_terrain.swamp}} {{infra_rations_bonus}} " +
		"{{base_production}} {{costs.mill_levels.0}} {{costs.mill_levels.1}} " +
		"{{costs.mill_levels.2}}\n"
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
		Costs:             Costs{MillLevels: []int{9, 10, 11}},
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
		if got, want := string(document), "3 2 4 5 6 7 8 9 10 11\n"; got != want {
			t.Errorf("rules[%s] = %q, want %q", language, got, want)
		}
	}
}

func TestLoadRulesRendersBalanceValuesInBothLanguages(t *testing.T) {
	balance, err := LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}
	rules, err := LoadRules("../../../assets", balance)
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}
	for _, language := range []string{"fr", "en"} {
		document, ok := rules.Document(language)
		if !ok {
			t.Fatalf("rules[%s] missing", language)
		}
		text := string(document)
		for _, value := range []string{"{{special_orders.deck_size}}", "{{special_orders.hand_limit}}", "{{special_orders.card.plague}}"} {
			if strings.Contains(text, value) {
				t.Errorf("rules[%s] still contains placeholder %q", language, value)
			}
		}
		if !strings.Contains(text, "30") || !strings.Contains(text, "4") {
			t.Errorf("rules[%s] does not contain rendered balance values", language)
		}
	}
}

func TestLoadRulesRendersArmyCostsFromCostBase(t *testing.T) {
	dir := t.TempDir()
	template := "{{army_cost.1}} {{army_cost.2}} {{army_cost.3}} {{army_cost.4}} {{army_cost.5}}\n"
	if err := os.WriteFile(filepath.Join(dir, playerRulesAsset), []byte(template), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}
	rules, err := LoadRules(dir, Balance{CostBase: 3})
	if err != nil {
		t.Fatalf("LoadRules = %v", err)
	}
	document, _ := rules.Document("fr")
	if got, want := string(document), "1 3 9 27 81\n"; got != want {
		t.Errorf("army costs = %q, want %q", got, want)
	}
}

var unresolvedPlaceholder = regexp.MustCompile(`\{\{[^}]*\}\}`)

func TestShippedRulesHaveNoUnresolvedPlaceholders(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "assets")
	balance, err := LoadBalance(dir)
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
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
		if matches := unresolvedPlaceholder.FindAllString(string(document), -1); len(matches) != 0 {
			t.Errorf("rules[%s] has unresolved placeholders: %s", language, strings.Join(matches, ", "))
		}
	}
}
