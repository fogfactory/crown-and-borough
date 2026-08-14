package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"
)

var territoryTrigram = regexp.MustCompile(`^[A-Z]{3}$`)
var territoryMatricule = regexp.MustCompile(`\bT[0-9]+\b`)

func TestContractFixturesRoundTrip(t *testing.T) {
	for _, name := range []string{
		"map.json",
		"state-territory-id.json",
		"state-army-no-chain.json",
		"state-army-hidden-chain.json",
		"report-combat-exact.json",
		"report-combat-general.json",
	} {
		t.Run(name, func(t *testing.T) {
			value := readContractFixture(t, name)
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal fixture: %v", err)
			}

			var roundTripped any
			if err := json.Unmarshal(encoded, &roundTripped); err != nil {
				t.Fatalf("unmarshal round-tripped fixture: %v", err)
			}
			if !reflect.DeepEqual(value, roundTripped) {
				t.Errorf("JSON round trip changed fixture:\n got: %#v\nwant: %#v", roundTripped, value)
			}
		})
	}
}

func TestPublicContractFilesDoNotExposeTerritoryMatricules(t *testing.T) {
	for _, name := range []string{
		"specs/fixtures/map.json",
		"specs/fixtures/state-territory-id.json",
		"specs/fixtures/state-army-no-chain.json",
		"specs/fixtures/state-army-hidden-chain.json",
		"specs/fixtures/report-combat-exact.json",
		"specs/fixtures/report-combat-general.json",
		"assets/regles-joueurs.md",
		"assets/regles-joueurs.en.md",
	} {
		t.Run(name, func(t *testing.T) {
			data := readProjectFile(t, name)
			if match := territoryMatricule.Find(data); match != nil {
				t.Errorf("public contract file exposes territory matricule %q", match)
			}
		})
	}
}

func TestTerritoryContractUsesCanonicalTrigramID(t *testing.T) {
	for _, name := range []string{
		"map.json",
		"state-territory-id.json",
		"state-army-no-chain.json",
		"state-army-hidden-chain.json",
	} {
		t.Run(name, func(t *testing.T) {
			fixture := readContractFixture(t, name)
			territories := findTerritories(t, fixture)
			if len(territories) == 0 {
				t.Fatal("fixture contains no territories")
			}
			for index, territory := range territories {
				id, ok := territory["id"].(string)
				if !ok || !territoryTrigram.MatchString(id) {
					t.Errorf("territory %d id = %#v, want a three-letter uppercase trigram", index, territory["id"])
				}
				if indexOfTerritory(territories, id) != index {
					t.Errorf("territory id %q is duplicated", id)
				}
				if _, exists := territory["code"]; exists {
					t.Errorf("territory %q contains duplicate code field", id)
				}
			}
		})
	}
}

func TestChainProjectionDistinguishesAbsentAndHidden(t *testing.T) {
	noChain := findFirstArmyChain(t, readContractFixture(t, "state-army-no-chain.json"))
	if noChain != nil {
		t.Fatalf("army without a chain has chain %#v, want null", noChain)
	}

	hidden := findFirstArmyChain(t, readContractFixture(t, "state-army-hidden-chain.json"))
	hiddenObject, ok := hidden.(map[string]any)
	if !ok {
		t.Fatalf("hidden chain = %#v, want an object", hidden)
	}
	if got := hiddenObject["visibility"]; got != "hidden" {
		t.Errorf("hidden chain visibility = %#v, want hidden", got)
	}
	if len(hiddenObject) != 1 {
		t.Errorf("hidden chain exposes details: %#v", hiddenObject)
	}
	if reflect.DeepEqual(noChain, hidden) {
		t.Error("null chain and hidden chain must remain distinct JSON values")
	}
}

func TestCombatProjectionDistinguishesExactAndGeneral(t *testing.T) {
	exact := firstCombat(t, readContractFixture(t, "report-combat-exact.json"))
	if got := exact["visibility"]; got != "exact" {
		t.Errorf("exact combat visibility = %#v, want exact", got)
	}
	contenders, ok := exact["contenders"].([]any)
	if !ok || len(contenders) == 0 {
		t.Fatalf("exact combat contenders = %#v, want detailed contenders", exact["contenders"])
	}
	for index, value := range contenders {
		contender, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("exact contender %d = %#v, want object", index, value)
		}
		for _, field := range []string{"army", "owner", "force"} {
			if _, exists := contender[field]; !exists {
				t.Errorf("exact contender %d omits %q", index, field)
			}
		}
	}

	general := firstCombat(t, readContractFixture(t, "report-combat-general.json"))
	if got := general["visibility"]; got != "general" {
		t.Errorf("general combat visibility = %#v, want general", got)
	}
	for _, field := range []string{
		"contenders",
		"force",
		"army",
		"owner",
		"baseDefense",
		"defense",
		"castleBonus",
		"winner",
		"dislodged",
	} {
		if _, exists := general[field]; exists {
			t.Errorf("general combat exposes private field %q", field)
		}
	}
}

func readContractFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data := readProjectFile(t, filepath.Join("specs", "fixtures", name))

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("fixture %s = %T, want JSON object", name, value)
	}
	return object
}

func readProjectFile(t *testing.T, name string) []byte {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(sourceFile), "..", "..", filepath.FromSlash(name))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read project file %s: %v", name, err)
	}
	return data
}

func findTerritories(t *testing.T, fixture map[string]any) []map[string]any {
	t.Helper()
	value, ok := fixture["territories"]
	if !ok {
		t.Fatalf("fixture has no territories: %v", fixture)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("territories = %T, want array", value)
	}
	territories := make([]map[string]any, 0, len(items))
	for index, item := range items {
		territory, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("territory %d = %T, want object", index, item)
		}
		territories = append(territories, territory)
	}
	return territories
}

func findFirstArmyChain(t *testing.T, fixture map[string]any) any {
	t.Helper()
	for index, territory := range findTerritories(t, fixture) {
		army, ok := territory["army"]
		if army == nil {
			continue
		}
		armyObject, ok := army.(map[string]any)
		if !ok {
			t.Fatalf("territory %d army = %T, want object", index, army)
		}
		chain, exists := armyObject["chain"]
		if !exists {
			t.Fatalf("territory %d army has no chain field", index)
		}
		return chain
	}
	t.Fatalf("fixture has no army: %v", fixture)
	return nil
}

func indexOfTerritory(territories []map[string]any, id string) int {
	for index, territory := range territories {
		if territoryID, ok := territory["id"].(string); ok && territoryID == id {
			return index
		}
	}
	return -1
}

func firstCombat(t *testing.T, fixture map[string]any) map[string]any {
	t.Helper()
	value, ok := fixture["combats"]
	if !ok {
		t.Fatalf("fixture has no combats: %v", fixture)
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("combats = %#v, want a non-empty array", value)
	}
	combat, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("combat = %T, want object", items[0])
	}
	return combat
}
