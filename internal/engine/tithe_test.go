package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// titheState: P1 and P2 hold capital castles on PCP and QCP, in bishoprics PCP
// (bishop N1 of P1) and QCP (bishop N2 of P2); the bishopric MIL has no
// bishop and a level-3 mill, held by a P3 army. N1 and N2 are cardinals.
func titheState(t *testing.T) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("PCP", "PCP", "MIL"), territory("QCP", "QCP", "MIL"), territory("MIL", "MIL", "PCP", "QCP")},
		nil,
	)
	state.Season = models.SeasonSpring
	state.Regions = []models.Region{
		{ID: "PCP", Seed: "PCP", Territories: []models.TerritoryID{"PCP"}},
		{ID: "QCP", Seed: "QCP", Territories: []models.TerritoryID{"QCP"}},
		{ID: "MIL", Seed: "MIL", Territories: []models.TerritoryID{"MIL"}},
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "PCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "QCP"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 3, TerritoryID: "MIL"})
	setCapital(state, "P1", "I1")
	setCapital(state, "P2", "I2")
	addAnchorArmy(t, state, "A3", "P3", "MIL")
	addNoble(state, "N1", "AAA", "P1", "PCP")
	addNoble(state, "N2", "BBB", "P2", "QCP")
	state.Bishops = []models.Bishop{{Region: "PCP", Noble: "N1"}, {Region: "QCP", Noble: "N2"}}
	state.Cardinals = []models.NobleID{"N1", "N2"}
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{
			{ID: "C1", Kind: models.CardKindSeigneurialTax}, {ID: "C2", Kind: models.CardKindSeigneurialTax},
		},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}, "P2": {"C2"}},
	}
	validateTestState(t, state)
	return state
}

func tithe(noble models.NobleID, seed models.TerritoryID) []models.DeckOrder {
	return []models.DeckOrder{{ID: "O-" + models.OrderID(noble), Type: models.DeckOrderTypePlay, Kind: models.CardKindSeigneurialTax, TargetNobleID: noble, TargetTerritoryID: seed, Tithe: true}}
}

func supplyProduction(events []Event, source models.TerritoryID) int {
	for _, event := range eventsOfType(events, EventTypeSupply) {
		if event.SourceID == source {
			return event.Production
		}
	}
	return -1
}

func TestTitheDivertsMillProductionToTheCapital(t *testing.T) {
	state := titheState(t)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{"P1": tithe("N1", "MIL")})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := supplyProduction(resolution.Events, "PCP"); got != 3 {
		t.Errorf("capital production = %d, want the mill's 3 R", got)
	}
	mills := eventsOfType(resolution.Events, EventTypeMillProduction)
	if len(mills) != 1 || mills[0].Production != 0 {
		t.Errorf("mill events = %#v, want the mill's own production diverted", mills)
	}
	if len(resolution.State.TithedRegions) != 1 {
		t.Errorf("TithedRegions = %#v, want one window", resolution.State.TithedRegions)
	}
}

func TestLocalBishopTitheOutranksCardinalsAndPope(t *testing.T) {
	state := titheState(t)
	// P1's N1 is the bishop of PCP; P2's cardinal N2 also taxes PCP.
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1"}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": tithe("N1", "PCP"), "P2": tithe("N2", "PCP"),
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	canceled := eventsOfType(resolution.Events, EventTypeCardCanceled)
	if len(canceled) != 1 || canceled[0].OwnerID != "P2" || canceled[0].Reason != "tithe_outranked" {
		t.Fatalf("canceled = %#v, want P2's tithe outranked by the local bishop", canceled)
	}
}

func TestCardinalsShareTheMillAndLeaveTheRemainderOnSite(t *testing.T) {
	state := titheState(t)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": tithe("N1", "MIL"), "P2": tithe("N2", "MIL"),
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if p1, p2 := supplyProduction(resolution.Events, "PCP"), supplyProduction(resolution.Events, "QCP"); p1 != 1 || p2 != 1 {
		t.Errorf("capital production = %d and %d, want 1 R each (3 R split in two)", p1, p2)
	}
	mills := eventsOfType(resolution.Events, EventTypeMillProduction)
	if len(mills) != 1 || mills[0].Production != 1 {
		t.Errorf("mill events = %#v, want the 1 R remainder left on the mill", mills)
	}
}

func TestPopeTitheYieldsToACardinal(t *testing.T) {
	state := titheState(t)
	pope := models.NobleID("N1")
	state.Pope = &pope
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": tithe("N1", "QCP"), "P2": tithe("N2", "MIL"),
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if len(eventsOfType(resolution.Events, EventTypeCardCanceled)) != 0 {
		t.Errorf("tithes on different bishoprics must not conflict")
	}
	if got := supplyProduction(resolution.Events, "QCP"); got != 3 {
		t.Errorf("P2 capital production = %d, want the mill's 3 R", got)
	}
}

func TestBishopCannotTitheAnotherBishopric(t *testing.T) {
	state := titheState(t)
	state.Cardinals = nil
	_, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{"P1": tithe("N1", "MIL")})
	if err == nil {
		t.Fatalf("a bishop taxing a foreign bishopric must be rejected")
	}
}

func TestTitheOpensRevoltOnTheWholeBishopric(t *testing.T) {
	state := titheState(t)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{"P1": tithe("N1", "MIL")})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	next := resolution.State
	ctx := newResolutionContext(next, testBalance())
	if !ctx.revoltEligibleByTax("MIL") {
		t.Errorf("a tithed bishopric must stay open to Révolte on the following turn")
	}
}
