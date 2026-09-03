package orders

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestParseDeckOrdersAliasesAndComments(t *testing.T) {
	game := orderTestState()
	game.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS", "BOI"}}}
	parsed, parseErrors := ParseDeckOrders(`
		d c bt # discard
		d c ah
		t c
		t c
		p fw ros
		j RA ros
	`, game)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseDeckOrders errors = %#v", parseErrors)
	}
	if len(parsed) != 6 {
		t.Fatalf("parsed = %#v, want 6 orders", parsed)
	}
	if parsed[0].Type != models.DeckOrderTypeDiscard || parsed[0].Kind != models.CardKindFairWeather {
		t.Errorf("discard = %#v", parsed[0])
	}
	if parsed[1].Kind != models.CardKindAbundantHarvest || parsed[2].Type != models.DeckOrderTypeDraw || parsed[3].Type != models.DeckOrderTypeDraw {
		t.Errorf("aliases = %#v", parsed)
	}
	if parsed[4].Type != models.DeckOrderTypePlay || parsed[4].Kind != models.CardKindFairWeather || parsed[4].RegionSeed != "ROS" {
		t.Errorf("play = %#v", parsed[4])
	}
}

func TestParseDeckOrdersRejectsInvalidShapesKindsAndSeeds(t *testing.T) {
	game := orderTestState()
	game.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS", "BOI"}}}
	_, parseErrors := ParseDeckOrders("P PL ROS\nP BT XXX\nP BT ROS EXTRA\nX C BT\nP XX ROS", game)
	if len(parseErrors) != 5 {
		t.Fatalf("parseErrors = %#v, want 5 errors", parseErrors)
	}
	wantCodes := []string{ParseCodeSpecialKind, ParseCodeSpecialRegion, ParseCodeTooManyTargets, ParseCodeUnknownSymbol, ParseCodeSpecialKind}
	for index, want := range wantCodes {
		if parseErrors[index].Code != want {
			t.Errorf("error[%d].Code = %q, want %q", index, parseErrors[index].Code, want)
		}
	}
	if parsed, _ := ParseDeckOrders("P N NNN", game); parsed != nil {
		t.Fatalf("parsed P N NNN = %#v, want nil", parsed)
	}
	if parsed, parseErrors := ParseDeckOrders("R C AH", game); parsed != nil || len(parseErrors) != 1 {
		t.Fatalf("parsed R C AH = %#v/%#v, want one error", parsed, parseErrors)
	}
}

func TestParseDeckOrdersPreservesLineNumbersAndAtomicity(t *testing.T) {
	game := orderTestState()
	game.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS"}}}
	parsed, parseErrors := ParseDeckOrders("\n# comment\nT C\nP BT BAD\n", game)
	if parsed != nil || len(parseErrors) != 1 || parseErrors[0].Line != 4 {
		t.Fatalf("parsed/errors = %#v/%#v, want nil and line 4", parsed, parseErrors)
	}
}
