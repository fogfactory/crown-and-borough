package engine

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestCreateGameInitialSetupAndDeterminism(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := testBalance()
	players := []PlayerInit{{Name: "One"}, {Name: "Two"}}

	first, err := CreateGame("setup-test", players, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	second, err := CreateGame("setup-test", players, balance, assets)
	if err != nil {
		t.Fatalf("second CreateGame: %v", err)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first game: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second game: %v", err)
	}
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatal("same seed and inputs produced different game states")
	}

	mapData, err := GenerateMap("setup-test", len(players), assets)
	if err != nil {
		t.Fatalf("GenerateMap: %v", err)
	}
	villages := make(map[models.TerritoryID]bool)
	for _, territory := range mapData.Territories {
		villages[models.TerritoryID(territory.ID)] = territory.Village
	}
	starts := make(map[models.TerritoryID]bool)
	for _, player := range first.Players {
		if player.CapitalCastleID == nil {
			t.Fatalf("player %s has no capital", player.ID)
		}
		castle := infrastructureByID(first, *player.CapitalCastleID)
		if castle.Type != models.InfraTypeCastle {
			t.Fatalf("player %s capital is %s, want castle", player.ID, castle.Type)
		}
		if !villages[castle.TerritoryID] {
			t.Fatalf("player %s capital %s was not placed on a generated village", player.ID, castle.TerritoryID)
		}
		if starts[castle.TerritoryID] {
			t.Fatalf("starting territory %s was assigned twice", castle.TerritoryID)
		}
		starts[castle.TerritoryID] = true
		state := first.TerritoryStates[castle.TerritoryID]
		if state.OwnerID == nil || *state.OwnerID != player.ID {
			t.Fatalf("starting territory %s is not controlled by %s", castle.TerritoryID, player.ID)
		}
		if state.Resources != balance.StartingResources {
			t.Errorf("starting resources at %s = %d, want %d", castle.TerritoryID, state.Resources, balance.StartingResources)
		}
		if state.Army == nil {
			t.Fatalf("starting territory %s has no army", castle.TerritoryID)
		}
		army := armyByID(t, first, *state.Army)
		if army.OwnerID != player.ID || army.Size != balance.StartingTroops {
			t.Errorf("starting army = %#v, want owner %s size %d", army, player.ID, balance.StartingTroops)
		}
	}
	if len(first.Nobles) != len(players)*balance.StartingNobles {
		t.Fatalf("starting nobles = %d, want %d", len(first.Nobles), len(players)*balance.StartingNobles)
	}
	seenNobleCodes := make(map[string]bool)
	for _, noble := range first.Nobles {
		if seenNobleCodes[noble.Code] {
			t.Fatalf("duplicate starting noble code %s", noble.Code)
		}
		seenNobleCodes[noble.Code] = true
		if noble.Name == "" || noble.Status != models.NobleStatusFree {
			t.Errorf("invalid starting noble %#v", noble)
		}
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("created state is invalid: %v", err)
	}
}

func TestResolveTurnAdvancesEverySeasonAndIsPure(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("cycle-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	originalJSON, err := json.Marshal(game)
	if err != nil {
		t.Fatalf("marshal original: %v", err)
	}
	seasons := []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn, models.SeasonWinter}
	current := game
	for index, wantSeason := range seasons {
		if current.Season != wantSeason {
			t.Fatalf("before turn %d season = %s, want %s", current.Turn, current.Season, wantSeason)
		}
		report, resolveErr := ResolveTurn(current, testBalance(), OrdersInput{})
		if resolveErr != nil {
			t.Fatalf("ResolveTurn %s: %v", wantSeason, resolveErr)
		}
		if report.Header.Season != wantSeason || report.Header.Turn != index+1 {
			t.Errorf("report header = %#v, want turn %d season %s", report.Header, index+1, wantSeason)
		}
		current = report.State
	}
	if current.Turn != 5 || current.Season != models.SeasonSpring {
		t.Fatalf("after one year = turn %d season %s, want turn 5 spring", current.Turn, current.Season)
	}
	unchangedJSON, err := json.Marshal(game)
	if err != nil {
		t.Fatalf("marshal unchanged game: %v", err)
	}
	if !reflect.DeepEqual(originalJSON, unchangedJSON) {
		t.Fatal("ResolveTurn mutated its input game")
	}
}

func TestResolveTurnRejectsBadInputWithoutMutation(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("input-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	before, _ := json.Marshal(game)
	noble := game.Nobles[0]
	_, err = ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{{Player: noble.OwnerID, Noble: models.NobleCode(noble.Code), Text: "BAD\nH T01"}},
	})
	var inputErrors *InputErrors
	if !errors.As(err, &inputErrors) {
		t.Fatalf("ResolveTurn error = %v, want InputErrors", err)
	}
	if len(inputErrors.Errors) == 0 || inputErrors.Errors[0].Line != 1 {
		t.Fatalf("input errors = %#v, want a line 1 parse error", inputErrors.Errors)
	}
	after, _ := json.Marshal(game)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("invalid input mutated the game")
	}

	_, err = ResolveTurn(game, testBalance(), OrdersInput{Winter: []WinterSubmission{{Player: "P1", Lines: "R T T01"}}})
	if !errors.As(err, &inputErrors) {
		t.Fatalf("out-of-season error = %v, want InputErrors", err)
	}
}

func TestResolveTurnReportsLostReception(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("reception-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	noble := game.Nobles[0]
	freeTerritory := ""
	for _, territory := range game.Territories {
		if game.TerritoryStates[territory.ID].Army == nil {
			freeTerritory = territory.Code
			break
		}
	}
	if freeTerritory == "" {
		t.Fatal("created map has no unoccupied territory")
	}
	report, err := ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{{
			Player: noble.OwnerID,
			Noble:  models.NobleCode(noble.Code),
			Text:   noble.Code + "\nH " + freeTerritory,
		}},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}
	if len(report.Receptions) != 1 || report.Receptions[0].Received {
		t.Fatalf("receptions = %#v, want one lost chain", report.Receptions)
	}
}

func TestResolveTurnDefersNonAdjacentOrderAndBreaksChain(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("deferred-non-adjacency", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	noble := game.Nobles[0]
	start := territoryByID(game.Territories, noble.LocationID)
	nextID := models.TerritoryID("")
	for _, adjacentID := range start.Adjacencies {
		if game.TerritoryStates[adjacentID].Army == nil {
			nextID = adjacentID
			break
		}
	}
	if nextID == "" {
		t.Fatal("starting territory has no unoccupied adjacent territory")
	}
	next := territoryByID(game.Territories, nextID)
	farID := models.TerritoryID("")
	for _, territory := range game.Territories {
		if territory.ID == next.ID {
			continue
		}
		adjacentToNext := false
		for _, adjacentID := range next.Adjacencies {
			if adjacentID == territory.ID {
				adjacentToNext = true
				break
			}
		}
		if !adjacentToNext {
			farID = territory.ID
			break
		}
	}
	if farID == "" {
		t.Fatal("map has no territory beyond the second territory's neighbors")
	}
	far := territoryByID(game.Territories, farID)

	report, err := ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{{
			Player: noble.OwnerID,
			Noble:  models.NobleCode(noble.Code),
			Text:   noble.Code + "\n" + start.Code + " A " + next.Code + "\nH " + next.Code + "\n" + next.Code + " A " + far.Code,
		}},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}
	if len(report.Receptions) != 1 || !report.Receptions[0].Received {
		t.Fatalf("receptions = %#v, want the non-adjacent chain received", report.Receptions)
	}
	if len(report.Orders) != 1 || report.Orders[0].Outcome != OutcomeSuccess {
		t.Fatalf("orders = %#v, want O1 success", report.Orders)
	}
	serializable := report
	serializable.State = nil
	if _, err := json.Marshal(serializable); err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	second, err := ResolveTurn(report.State, testBalance(), OrdersInput{})
	if err != nil {
		t.Fatalf("second ResolveTurn: %v", err)
	}
	if len(second.Orders) != 1 || second.Orders[0].Outcome != OutcomeSuccess || second.Orders[0].Progression != ProgressionAdvanced {
		t.Fatalf("second orders = %#v, want O2 hold success advanced", second.Orders)
	}

	third, err := ResolveTurn(second.State, testBalance(), OrdersInput{})
	if err != nil {
		t.Fatalf("third ResolveTurn: %v", err)
	}
	if len(third.Orders) != 1 || third.Orders[0].Outcome != OutcomeInvalid || third.Orders[0].Progression != ProgressionBroken {
		t.Fatalf("third orders = %#v, want O3 invalid and chain broken", third.Orders)
	}
	if third.Orders[0].Reason != "non_adjacent_destination" {
		t.Errorf("O3 reason = %q, want non_adjacent_destination", third.Orders[0].Reason)
	}
	if len(third.State.Chains) != 0 {
		t.Errorf("chains after break = %#v, want none", third.State.Chains)
	}

	fourth, err := ResolveTurn(third.State, testBalance(), OrdersInput{})
	if err != nil {
		t.Fatalf("fourth ResolveTurn: %v", err)
	}
	if len(fourth.Orders) != 0 {
		t.Errorf("fourth orders = %#v, want O4 never executed", fourth.Orders)
	}
}

func TestFullYearDeterminism(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := testBalance()
	players := []PlayerInit{{Name: "One"}, {Name: "Two"}}
	run := func() (*models.GameState, []TurnReport) {
		game, err := CreateGame("year-determinism", players, balance, assets)
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		start := territoryByID(game.Territories, game.Nobles[0].LocationID)
		reports := make([]TurnReport, 0, 4)
		for game.Season != models.SeasonWinter {
			var report TurnReport
			report, err = ResolveTurn(game, balance, OrdersInput{})
			if err != nil {
				t.Fatalf("action turn: %v", err)
			}
			reports = append(reports, report)
			game = report.State
		}
		report, err := ResolveTurn(game, balance, OrdersInput{
			Winter: []WinterSubmission{{Player: "P1", Lines: "R T " + start.Code}},
		})
		if err != nil {
			t.Fatalf("winter turn: %v", err)
		}
		reports = append(reports, report)
		return report.State, reports
	}
	firstState, firstReports := run()
	secondState, secondReports := run()
	firstStateJSON, err := json.Marshal(firstState)
	if err != nil {
		t.Fatalf("marshal first final state: %v", err)
	}
	secondStateJSON, err := json.Marshal(secondState)
	if err != nil {
		t.Fatalf("marshal second final state: %v", err)
	}
	if !reflect.DeepEqual(firstStateJSON, secondStateJSON) {
		t.Fatal("same full-year orders produced different final states")
	}
	firstReportsJSON, err := json.Marshal(firstReports)
	if err != nil {
		t.Fatalf("marshal first reports: %v", err)
	}
	secondReportsJSON, err := json.Marshal(secondReports)
	if err != nil {
		t.Fatalf("marshal second reports: %v", err)
	}
	if !reflect.DeepEqual(firstReportsJSON, secondReportsJSON) {
		t.Fatal("same full-year orders produced different reports")
	}
}

func infrastructureByID(state *models.GameState, id models.InfraID) models.Infrastructure {
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.ID == id {
			return infrastructure
		}
	}
	return models.Infrastructure{}
}

func loadGameTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	return assets
}
