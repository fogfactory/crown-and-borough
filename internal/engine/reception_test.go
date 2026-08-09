package engine

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveTurnRejectsConcurrentReceptionsFromTwoNobles(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("concurrent-reception-nobles", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	p1 := game.Players[0].ID
	army := gameArmyForPlayer(t, game, p1)
	firstNoble := gameNobleForPlayer(t, game, p1)
	addNoble(game, "N3", "ZZZ", p1, army.TerritoryID)
	validateTestState(t, game)

	position := territoryByID(game.Territories, army.TerritoryID)
	report, err := ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{
			{
				Player: p1,
				Noble:  models.NobleCode(firstNoble.Code),
				Text:   firstNoble.Code + "\nH " + position.Code,
			},
			{
				Player: p1,
				Noble:  "ZZZ",
				Text:   "ZZZ\nH " + position.Code,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}

	assertReceptionResults(t, report, map[models.NobleCode]bool{
		models.NobleCode(firstNoble.Code): false,
		"ZZZ":                             false,
	})
	if len(report.State.Chains) != 0 {
		t.Fatalf("chains after concurrent reception = %#v, want none", report.State.Chains)
	}
	if army := armyByID(t, report.State, army.ID); army.ChainID != nil {
		t.Fatalf("army after concurrent reception = %#v, want no chain", army)
	}
	if len(report.Orders) != 0 {
		t.Fatalf("orders after concurrent reception = %#v, want none", report.Orders)
	}
}

func TestResolveTurnRejectsConcurrentReceptionsFromDistinctPlayers(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("concurrent-reception-players", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	p1 := game.Players[0].ID
	p2 := game.Players[1].ID
	army := gameArmyForPlayer(t, game, p1)
	p1Noble := gameNobleForPlayer(t, game, p1)
	p2Noble := gameNobleForPlayer(t, game, p2)
	position := territoryByID(game.Territories, army.TerritoryID)

	report, err := ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{
			{
				Player: p1,
				Noble:  models.NobleCode(p1Noble.Code),
				Text:   p1Noble.Code + "\nH " + position.Code,
			},
			{
				Player: p2,
				Noble:  models.NobleCode(p2Noble.Code),
				Text:   p2Noble.Code + "\nH " + position.Code,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}

	assertReceptionResults(t, report, map[models.NobleCode]bool{
		models.NobleCode(p1Noble.Code): false,
		models.NobleCode(p2Noble.Code): false,
	})
	for _, reception := range report.Receptions {
		wantPlayer := p1
		if reception.Noble == models.NobleCode(p2Noble.Code) {
			wantPlayer = p2
		}
		if reception.Player != wantPlayer {
			t.Errorf("reception for %s belongs to %s, want %s", reception.Noble, reception.Player, wantPlayer)
		}
	}
	if len(report.State.Chains) != 0 {
		t.Fatalf("chains after cross-player conflict = %#v, want none", report.State.Chains)
	}
}

func TestResolveTurnKeepsOtherArmyReceptionInSameSubmission(t *testing.T) {
	assets := loadGameTestAssets(t)
	game, err := CreateGame("concurrent-reception-isolation", []PlayerInit{{Name: "One"}, {Name: "Two"}}, testBalance(), assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	p1 := game.Players[0].ID
	firstArmy := gameArmyForPlayer(t, game, p1)
	secondArmy := gameArmyForPlayer(t, game, game.Players[1].ID)
	for index := range game.Armies {
		if game.Armies[index].ID == secondArmy.ID {
			game.Armies[index].OwnerID = p1
			break
		}
	}
	firstNoble := gameNobleForPlayer(t, game, p1)
	addNoble(game, "N3", "ZZZ", p1, firstArmy.TerritoryID)
	addNoble(game, "N4", "YYY", p1, firstArmy.TerritoryID)
	validateTestState(t, game)

	firstPosition := territoryByID(game.Territories, firstArmy.TerritoryID)
	secondPosition := territoryByID(game.Territories, secondArmy.TerritoryID)
	report, err := ResolveTurn(game, testBalance(), OrdersInput{
		Chains: []ChainSubmission{
			{
				Player: p1,
				Noble:  models.NobleCode(firstNoble.Code),
				Text:   firstNoble.Code + "\nH " + firstPosition.Code + "\nH " + firstPosition.Code,
			},
			{
				Player: p1,
				Noble:  "ZZZ",
				Text:   "ZZZ\nH " + secondPosition.Code,
			},
			{
				Player: p1,
				Noble:  "YYY",
				Text:   "YYY\nH " + secondPosition.Code,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveTurn: %v", err)
	}

	assertReceptionResults(t, report, map[models.NobleCode]bool{
		models.NobleCode(firstNoble.Code): true,
		"ZZZ":                             false,
		"YYY":                             false,
	})
	if len(report.Orders) != 1 || report.Orders[0].Army != firstArmy.ID || report.Orders[0].Outcome != OutcomeSuccess {
		t.Fatalf("orders = %#v, want one successful order for %s", report.Orders, firstArmy.ID)
	}
	chain := chainOf(report.State, firstArmy.ID)
	if chain == nil || chain.CurrentIndex != 1 {
		t.Fatalf("unrelated army chain = %#v, want received chain at index 1", chain)
	}
	if army := armyByID(t, report.State, secondArmy.ID); army.ChainID != nil {
		t.Fatalf("conflicted army = %#v, want no chain", army)
	}
}

func assertReceptionResults(t *testing.T, report TurnReport, want map[models.NobleCode]bool) {
	t.Helper()
	if len(report.Receptions) != len(want) {
		t.Fatalf("receptions = %#v, want %d entries", report.Receptions, len(want))
	}
	seen := make(map[models.NobleCode]bool, len(report.Receptions))
	for _, reception := range report.Receptions {
		expected, exists := want[reception.Noble]
		if !exists {
			t.Errorf("unexpected reception for noble %s", reception.Noble)
			continue
		}
		if seen[reception.Noble] {
			t.Errorf("duplicate reception for noble %s", reception.Noble)
		}
		seen[reception.Noble] = true
		if reception.Received != expected {
			t.Errorf("reception for noble %s = %#v, want received=%t", reception.Noble, reception, expected)
		}
		if !expected && !strings.HasPrefix(reception.Reason, "concurrent_reception:") {
			t.Errorf("reception for noble %s reason = %q, want concurrent_reception", reception.Noble, reception.Reason)
		}
	}
	for noble, expected := range want {
		if !seen[noble] {
			t.Errorf("missing reception for noble %s, want received=%t", noble, expected)
		}
	}
}

func gameArmyForPlayer(t *testing.T, game *models.GameState, playerID models.PlayerID) models.Army {
	t.Helper()
	for _, army := range game.Armies {
		if army.OwnerID == playerID {
			return army
		}
	}
	t.Fatalf("army for player %s not found", playerID)
	return models.Army{}
}

func gameNobleForPlayer(t *testing.T, game *models.GameState, playerID models.PlayerID) models.Noble {
	t.Helper()
	for _, noble := range game.Nobles {
		if noble.OwnerID == playerID {
			return noble
		}
	}
	t.Fatalf("noble for player %s not found", playerID)
	return models.Noble{}
}
