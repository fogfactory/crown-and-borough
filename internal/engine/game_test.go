package engine

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
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
		if villages[castle.TerritoryID] {
			t.Fatalf("player %s capital %s was placed on a generated village", player.ID, castle.TerritoryID)
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

func TestCreateGameCountsCastlesVillagesAndStartingTerritories(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := testBalance()

	for playerCount := 2; playerCount <= 16; playerCount++ {
		players := make([]PlayerInit, playerCount)
		game, err := CreateGame("setup-counts", players, balance, assets)
		if err != nil {
			t.Fatalf("CreateGame(%d): %v", playerCount, err)
		}
		mapData, err := GenerateMap("setup-counts", playerCount, assets)
		if err != nil {
			t.Fatalf("GenerateMap(%d): %v", playerCount, err)
		}

		villageFlags := make(map[models.TerritoryID]bool, len(mapData.Territories))
		for _, territory := range mapData.Territories {
			villageFlags[models.TerritoryID(territory.ID)] = territory.Village
		}
		adjacencies := make(map[models.TerritoryID][]models.TerritoryID, len(game.Territories))
		for _, territory := range game.Territories {
			adjacencies[territory.ID] = territory.Adjacencies
		}

		infrastructures := make(map[models.InfraID]models.Infrastructure, len(game.Infrastructures))
		for _, infrastructure := range game.Infrastructures {
			infrastructures[infrastructure.ID] = infrastructure
		}

		castleCount := 0
		villageCount := 0
		startingTerritories := make(map[models.TerritoryID]bool, playerCount)
		startingIDs := make([]models.TerritoryID, 0, playerCount)
		for _, infrastructure := range game.Infrastructures {
			switch infrastructure.Type {
			case models.InfraTypeCastle:
				castleCount++
				startingIDs = append(startingIDs, infrastructure.TerritoryID)
				if villageFlags[infrastructure.TerritoryID] {
					t.Errorf("players=%d: castle %s is on village territory %s", playerCount, infrastructure.ID, infrastructure.TerritoryID)
				}
				if startingTerritories[infrastructure.TerritoryID] {
					t.Errorf("players=%d: starting territory %s was assigned twice", playerCount, infrastructure.TerritoryID)
				}
				startingTerritories[infrastructure.TerritoryID] = true
			case models.InfraTypeVillage:
				villageCount++
				if !villageFlags[infrastructure.TerritoryID] {
					t.Errorf("players=%d: village infrastructure %s is not on a generated village", playerCount, infrastructure.ID)
				}
				territoryState := game.TerritoryStates[infrastructure.TerritoryID]
				if territoryState.OwnerID != nil {
					t.Errorf("players=%d: village territory %s is controlled by %s", playerCount, infrastructure.TerritoryID, *territoryState.OwnerID)
				}
			}
		}

		if castleCount != playerCount {
			t.Errorf("players=%d: castles = %d, want %d", playerCount, castleCount, playerCount)
		}
		if len(startingTerritories) != playerCount {
			t.Errorf("players=%d: starting territories = %d, want %d", playerCount, len(startingTerritories), playerCount)
		}
		for firstIndex, firstID := range startingIDs {
			for _, secondID := range startingIDs[firstIndex+1:] {
				distance := testGraphDistance(adjacencies, firstID, secondID)
				if distance < MinimumStartingDistance {
					t.Errorf("players=%d: starting territories %s and %s have distance %d, want at least %d", playerCount, firstID, secondID, distance, MinimumStartingDistance)
				}
			}
		}
		if villageCount != playerCount+1 {
			t.Errorf("players=%d: neutral villages = %d, want %d", playerCount, villageCount, playerCount+1)
		}
		for territoryID, territoryState := range game.TerritoryStates {
			if len(territoryState.Infrastructures) > 1 {
				t.Errorf("players=%d: territory %s has %d infrastructures", playerCount, territoryID, len(territoryState.Infrastructures))
			}
			for _, infrastructureID := range territoryState.Infrastructures {
				if infrastructure, ok := infrastructures[infrastructureID]; !ok || infrastructure.TerritoryID != territoryID {
					t.Errorf("players=%d: infrastructure %s is not indexed by territory %s", playerCount, infrastructureID, territoryID)
				}
			}
		}
	}
}

func TestGameMapConfigScalesViewportWithPlayers(t *testing.T) {
	for _, playerCount := range []int{2, 4, 16} {
		config := GameMapConfig(playerCount)
		wantWidth := 1000 * playerCount / 4
		wantHeight := 700 * playerCount / 4
		if config.Width != wantWidth || config.Height != wantHeight {
			t.Errorf("players=%d: viewport = %dx%d, want %dx%d", playerCount, config.Width, config.Height, wantWidth, wantHeight)
		}
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

func TestResolveTurnFinishesAfterConfiguredYears(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGameWithYears("duration-test", []PlayerInit{{Name: "One"}, {Name: "Two"}}, 1, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGameWithYears: %v", err)
	}
	for turn := 0; turn < 4; turn++ {
		report, resolveErr := ResolveTurn(game, testBalance(), OrdersInput{})
		if resolveErr != nil {
			t.Fatalf("ResolveTurn %d: %v", turn+1, resolveErr)
		}
		game = report.State
	}
	if game.Turn != 5 || game.YearCount != 1 {
		t.Fatalf("final state = turn %d, years %d; want turn 5, year count 1", game.Turn, game.YearCount)
	}
	if !GameFinished(game) {
		t.Fatal("game should be finished after the fourth turn")
	}
	if _, err := ResolveTurn(game, testBalance(), OrdersInput{}); !errors.Is(err, ErrGameFinished) {
		t.Fatalf("resolution after game end = %v, want ErrGameFinished", err)
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
		Chains: []ChainSubmission{{Player: noble.OwnerID, Noble: models.NobleCode(noble.Code), Text: "BAD\nH AAA"}},
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

	_, err = ResolveTurn(game, testBalance(), OrdersInput{Winter: []WinterSubmission{{Player: "P1", Lines: "R T AAA"}}})
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
			freeTerritory = string(territory.ID)
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

func TestResolveTurnRejectsNonAdjacentChain(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("reject-non-adjacency", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	before, _ := json.Marshal(game)
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

	_, err = ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{{
			Player: noble.OwnerID,
			Noble:  models.NobleCode(noble.Code),
			Text:   noble.Code + "\n" + string(start.ID) + " A " + string(next.ID) + "\nH " + string(next.ID) + "\n" + string(next.ID) + " A " + string(far.ID),
		}},
	})
	if err == nil {
		t.Fatal("ResolveTurn() = nil, want an input error")
	}
	var inputErrors *InputErrors
	if !errors.As(err, &inputErrors) {
		t.Fatalf("ResolveTurn() error = %v, want InputErrors", err)
	}
	if len(inputErrors.Errors) != 1 || inputErrors.Errors[0].Code != orders.ValidationCodeNotAdjacent {
		t.Fatalf("input errors = %#v, want one not_adjacent error", inputErrors.Errors)
	}
	if !strings.Contains(inputErrors.Errors[0].Message, "not adjacent") {
		t.Fatalf("input error message = %q, want explicit adjacency text", inputErrors.Errors[0].Message)
	}
	if after, _ := json.Marshal(game); !reflect.DeepEqual(before, after) {
		t.Fatal("rejected input mutated the game")
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
			Winter: []WinterSubmission{{Player: "P1", Lines: "R T " + string(start.ID)}},
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

func testGraphDistance(adjacencies map[models.TerritoryID][]models.TerritoryID, start, target models.TerritoryID) int {
	distances := map[models.TerritoryID]int{start: 0}
	queue := []models.TerritoryID{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == target {
			return distances[current]
		}
		for _, adjacent := range adjacencies[current] {
			if _, visited := distances[adjacent]; visited {
				continue
			}
			distances[adjacent] = distances[current] + 1
			queue = append(queue, adjacent)
		}
	}
	return -1
}

func loadGameTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	return assets
}
