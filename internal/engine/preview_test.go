package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func previewTestGame(t *testing.T, seed string) *models.GameState {
	t.Helper()
	game, err := CreateGame(seed, []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), loadGameTestAssets(t))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	return game
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}

func TestPreviewOrdersKeepsValidChainLinesNextToErrors(t *testing.T) {
	game := previewTestGame(t, "preview-chain")
	before := mustJSON(t, game)
	noble := gameNobleForPlayer(t, game, "P1")
	army := gameArmyForPlayer(t, game, "P1")
	text := noble.Code + "\nH " + string(army.TerritoryID) + "\nNOT AN ORDER"

	preview, err := PreviewOrders(game, testBalance(), "P1", OrdersInput{
		Chains: []ChainSubmission{{Player: "P1", Noble: models.NobleCode(noble.Code), Text: text}},
	})
	if err != nil {
		t.Fatalf("PreviewOrders: %v", err)
	}
	if len(preview.Chains) != 1 || len(preview.Chains[0].Orders) != 1 || preview.Chains[0].Orders[0].Type != models.OrderTypeHold {
		t.Fatalf("chains = %#v, want the hold line", preview.Chains)
	}
	if len(preview.Errors) != 1 || preview.Errors[0].Line != 3 {
		t.Fatalf("errors = %#v, want one error on line 3", preview.Errors)
	}
	if preview.WinterCost != nil || len(preview.Winter) != 0 {
		t.Fatalf("action preview has winter data: %#v", preview)
	}
	if after := mustJSON(t, game); after != before {
		t.Fatal("PreviewOrders mutated the game state")
	}
}

func TestPreviewOrdersReportsChainsTheEngineWouldNotReceive(t *testing.T) {
	game := previewTestGame(t, "preview-reception")
	noble := gameNobleForPlayer(t, game, "P1")
	enemy := gameArmyForPlayer(t, game, "P2")
	text := noble.Code + "\nH " + string(enemy.TerritoryID)

	preview, err := PreviewOrders(game, testBalance(), "P1", OrdersInput{
		Chains: []ChainSubmission{{Player: "P1", Noble: models.NobleCode(noble.Code), Text: text}},
	})
	if err != nil {
		t.Fatalf("PreviewOrders: %v", err)
	}
	if len(preview.Errors) != 1 || preview.Errors[0].Code != "not_received" || preview.Errors[0].MessageKey == "" {
		t.Fatalf("errors = %#v, want one not_received error with a message key", preview.Errors)
	}
}

func TestPreviewOrdersSimulatesEachWinterLine(t *testing.T) {
	game := previewTestGame(t, "preview-winter")
	game.Turn = 4
	game.Season = models.SeasonForTurn(game.Turn)
	if game.Season != models.SeasonWinter {
		t.Fatalf("turn 4 season = %s, want winter", game.Season)
	}
	before := mustJSON(t, game)
	capital := gameArmyForPlayer(t, game, "P1").TerritoryID
	sheet := "R T " + string(capital) + "\nNOT A LINE\n\nC C " + string(capital)

	preview, err := PreviewOrders(game, testBalance(), "P1", OrdersInput{
		Winter: []WinterSubmission{{Player: "P1", Lines: sheet}},
	})
	if err != nil {
		t.Fatalf("PreviewOrders: %v", err)
	}
	if len(preview.Winter) != 3 {
		t.Fatalf("winter lines = %#v, want three non-empty lines", preview.Winter)
	}
	troop, malformed, castle := preview.Winter[0], preview.Winter[1], preview.Winter[2]
	if troop.Line != 1 || troop.Order == nil || !troop.Applied || troop.Cost != testBalance().Costs.Troop || troop.Territory != capital {
		t.Errorf("troop line = %#v, want an applied recruit paid %d on %s", troop, testBalance().Costs.Troop, capital)
	}
	if malformed.Line != 2 || malformed.ParseError == nil || malformed.Order != nil {
		t.Errorf("malformed line = %#v, want a parse error on line 2", malformed)
	}
	if castle.Line != 4 || castle.Order == nil || castle.Applied || castle.Reason != "structure_present" {
		t.Errorf("castle line = %#v, want a structure_present rejection on line 4", castle)
	}
	want := &WinterCostPreview{Spent: testBalance().Costs.Troop, Available: testBalance().StartingResources}
	if !reflect.DeepEqual(preview.WinterCost, want) {
		t.Errorf("winter cost = %#v, want %#v", preview.WinterCost, want)
	}
	if after := mustJSON(t, game); after != before {
		t.Fatal("PreviewOrders mutated the game state")
	}
}
