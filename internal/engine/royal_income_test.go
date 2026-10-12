package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// royalIncomeTestState extends taxTestState with a king (P2, noble KKK) whose
// capital is KCP, and a queen (P3, noble QQQ) whose capital is QCP.
func royalIncomeTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := taxTestState(t)
	for _, id := range []models.TerritoryID{"KCP", "QCP"} {
		state.Territories = append(state.Territories, territory(string(id), string(id)))
		state.TerritoryStates[id] = models.TerritoryState{}
	}
	addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "KCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I5", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "QCP"})
	setCapital(state, "P2", "I4")
	setCapital(state, "P3", "I5")
	addNoble(state, "N-K", "KKK", "P2", "KCP")
	addNoble(state, "N-Q", "QQQ", "P3", "QCP")
	state.Nobles[len(state.Nobles)-1].Sex = models.SexFemale
	king, queen := models.NobleID("N-K"), models.NobleID("N-Q")
	state.King, state.Queen = &king, &queen
	state.SpecialDeck.Hands["P2"] = []models.SpecialCardID{"C2"}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1"}
	return state
}

func royalSource(events []Event, source string) (Event, bool) {
	for _, event := range events {
		if event.Type == EventTypeIncome && event.Reason == source {
			return event, true
		}
	}
	return Event{}, false
}

// TestRoyalRentPaidToKingAndQueen: 1 R per fief to the king's capital, half
// rounded up to the queen's.
func TestRoyalRentPaidToKingAndQueen(t *testing.T) {
	state := royalIncomeTestState(t)
	for _, id := range []models.TerritoryID{"QAA", "QBB"} {
		state.Territories = append(state.Territories, territory(string(id), string(id)))
		state.TerritoryStates[id] = models.TerritoryState{}
	}
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "F2", Title: models.FiefTitleBarony, CapitalTerritoryID: "QCP",
		Territories: []models.TerritoryID{"QCP", "QAA", "QBB"}, OwnerID: "P3",
	})
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	king, ok := royalSource(resolution.Events, incomeSourceRoyalRentKing)
	if !ok || king.OwnerID != "P2" || king.DestinationID != "KCP" || king.Production != 2 {
		t.Fatalf("king rent = %#v, want 2 R to KCP for P2", king)
	}
	queen, ok := royalSource(resolution.Events, incomeSourceRoyalRentQueen)
	if !ok || queen.OwnerID != "P3" || queen.DestinationID != "QCP" || queen.Production != 1 {
		t.Fatalf("queen rent = %#v, want 1 R to QCP for P3", queen)
	}
}

// TestRoyalRentVacantThrone: the regent, a dowager queen, receives the king's
// rent on top of the queen's.
func TestRoyalRentVacantThrone(t *testing.T) {
	state := royalIncomeTestState(t)
	state.King = nil
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	king, ok := royalSource(resolution.Events, incomeSourceRoyalRentKing)
	if !ok || king.OwnerID != "P3" || king.Production != 1 {
		t.Errorf("king rent = %#v, want 1 R to the regent's player P3", king)
	}
	if queen, ok := royalSource(resolution.Events, incomeSourceRoyalRentQueen); !ok || queen.OwnerID != "P3" || queen.Production != 1 {
		t.Errorf("queen rent = %#v, want 1 R to P3", queen)
	}
}

// TestRoyalRentVacantThroneWithRegentInDungeon: no regent, no king's rent.
func TestRoyalRentVacantThroneWithRegentInDungeon(t *testing.T) {
	state := royalIncomeTestState(t)
	state.King = nil
	for index := range state.Nobles {
		if state.Nobles[index].ID == *state.Queen {
			state.Nobles[index].Status = models.NobleStatusDungeon
		}
	}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if _, ok := royalSource(resolution.Events, incomeSourceRoyalRentKing); ok {
		t.Error("a vacant throne without regent paid a king's rent")
	}
}

// TestRoyalTaxSendsSupplementToKing: the fief keeps its base income, the king
// receives the same amount again.
func TestRoyalTaxSendsSupplementToKing(t *testing.T) {
	state := royalIncomeTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P2": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-K", TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := resolution.State.TerritoryStates["FCP"].Resources; got != 4 {
		t.Errorf("FCP stock = %d, want 4 (base income kept)", got)
	}
	supplement, ok := royalSource(resolution.Events, incomeSourceRoyalTax)
	if !ok || supplement.OwnerID != "P2" || supplement.DestinationID != "KCP" || supplement.Production != 4 {
		t.Fatalf("supplement = %#v, want 4 R to KCP for P2", supplement)
	}
	// 4 R of supplement + 1 R of king's rent + 1 R of KCP's own income.
	if got := resolution.State.TerritoryStates["KCP"].Resources; got != 6 {
		t.Errorf("KCP stock = %d, want 6", got)
	}
}

// TestRoyalTaxYieldsToSeigneurialTax: the lord's tax has priority, whatever
// the submission order.
func TestRoyalTaxYieldsToSeigneurialTax(t *testing.T) {
	state := royalIncomeTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetTerritoryID: "FCP"}},
		"P2": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-K", TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := resolution.State.TerritoryStates["FCP"].Resources; got != 8 {
		t.Errorf("FCP stock = %d, want 8 (doubled by the lord)", got)
	}
	if _, ok := royalSource(resolution.Events, incomeSourceRoyalTax); ok {
		t.Error("royal supplement paid on a fief taxed by its lord")
	}
}

// TestRoyalTaxRejectedForNonKing: a plain noble cannot tax another's fief.
func TestRoyalTaxRejectedForNonKing(t *testing.T) {
	state := royalIncomeTestState(t)
	addNoble(state, "N-X", "XXX", "P3", "QCP")
	state.SpecialDeck.Hands["P3"] = []models.SpecialCardID{"C1"}
	state.SpecialDeck.Hands["P1"] = nil
	validateTestState(t, state)
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P3": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-X", TargetTerritoryID: "FCP"}},
	})
	if err == nil {
		t.Fatal("want an error: only the king may tax a fief that is not his")
	}
}

// TestRoyalTaxIncomeLineIsAFlowOfItsOwn: the supplement is reported as a royal
// tax line into the king's capital, naming the taxed fief, and not as the
// income of a fief of the king.
func TestRoyalTaxIncomeLineIsAFlowOfItsOwn(t *testing.T) {
	state := royalIncomeTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P2": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-K", TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	for _, line := range report.Income {
		if line.Source != incomeSourceRoyalTax {
			continue
		}
		if line.Fief != "" || line.Title != "" || line.Taxed != "FCP" || line.Destination != "KCP" || line.Credited != 4 {
			t.Fatalf("royal tax line = %#v, want 4 R into KCP naming the taxed fief FCP and no fief of its own", line)
		}
		return
	}
	t.Fatalf("income = %#v, want a royal tax line", report.Income)
}

// TestRegentLevysTheRoyalTax: with the king held in a dungeon, his wife the
// regent issues the royal tax and receives the supplement.
func TestRegentLevysTheRoyalTax(t *testing.T) {
	state := royalIncomeTestState(t)
	state.Marriages = []models.Marriage{{NobleA: "N-K", NobleB: "N-Q"}}
	setNobleStatus(state, "N-K", models.NobleStatusDungeon)
	state.SpecialDeck.Hands["P3"] = []models.SpecialCardID{"C2"}
	state.SpecialDeck.Hands["P2"] = nil
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P3": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-Q", TargetTerritoryID: "FCP"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	supplement, ok := royalSource(resolution.Events, incomeSourceRoyalTax)
	if !ok || supplement.OwnerID != "P3" || supplement.DestinationID != "QCP" || supplement.Production != 4 {
		t.Fatalf("supplement = %#v, want 4 R to QCP for the regent's player P3", supplement)
	}
}

// TestQueenWhoIsNotRegentCannotLevyTheRoyalTax: while the king reigns, his
// queen has no power of order.
func TestQueenWhoIsNotRegentCannotLevyTheRoyalTax(t *testing.T) {
	state := royalIncomeTestState(t)
	state.Marriages = []models.Marriage{{NobleA: "N-K", NobleB: "N-Q"}}
	state.SpecialDeck.Hands["P3"] = []models.SpecialCardID{"C2"}
	state.SpecialDeck.Hands["P2"] = nil
	validateTestState(t, state)
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P3": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-Q", TargetTerritoryID: "FCP"}},
	})
	if err == nil {
		t.Fatal("want an error: a queen who does not govern cannot tax another player's fief")
	}
}

// TestRoyalTaxOpensRevoltInTheSameSubmission: a royal tax makes every territory
// of the taxed fief eligible for Révolte, like any tax.
func TestRoyalTaxOpensRevoltInTheSameSubmission(t *testing.T) {
	state := royalIncomeTestState(t)
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C3", Kind: models.CardKindRevolt})
	state.SpecialDeck.Hands["P3"] = []models.SpecialCardID{"C3"}
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P2": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: "N-K", TargetTerritoryID: "FCP"}},
		"P3": {{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "OTH"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeRejected && event.OrderID == "O2" {
			t.Fatalf("revolt on OTH next to a royal tax = rejected (%q), want accepted", event.Reason)
		}
	}
}
