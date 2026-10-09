package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// appeasementState: P1 holds AAA (village with 10 R) with its bishop N1; a
// two-troop rebel army stands on BBB (same region, bishop N1's) and a
// one-troop one on CCC (another region, no bishop).
func appeasementState(t *testing.T) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA", "CCC"), territory("CCC", "CCC", "BBB")},
		nil,
	)
	state.Regions = []models.Region{
		{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}},
		{ID: "CCC", Seed: "CCC", Territories: []models.TerritoryID{"CCC"}},
	}
	addNoble(state, "N1", "BIS", "P1", "AAA")
	addAnchorArmy(t, state, "A1", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	stock := state.TerritoryStates["AAA"]
	stock.Resources = 10
	state.TerritoryStates["AAA"] = stock
	state.Bishops = []models.Bishop{{Region: "AAA", Noble: "N1"}}
	for id, at := range map[models.ArmyID]models.TerritoryID{"A8": "BBB", "A9": "CCC"} {
		size := 2
		if id == "A9" {
			size = 1
		}
		state.Armies = append(state.Armies, models.Army{ID: id, OwnerID: models.NeutralPlayerID, TerritoryID: at, Size: size})
		territoryState := state.TerritoryStates[at]
		armyID := id
		territoryState.Army = &armyID
		state.TerritoryStates[at] = territoryState
	}
	state.NextArmyID = nextArmyID(state.Armies)
	validateTestState(t, state)
	return state
}

func appease(orderType models.DeckOrderType, noble models.NobleID, target models.TerritoryID) map[models.PlayerID][]models.DeckOrder {
	return map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: orderType, TargetNobleID: noble, TargetTerritoryID: target}},
	}
}

func TestPaidAppeasementRemovesTheRebelArmyAndCostsBasePowerSize(t *testing.T) {
	state := appeasementState(t)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeasePaid, "N1", "BBB"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if hasArmy(resolution.State, "A8") {
		t.Fatalf("rebel army survived a paid appeasement")
	}
	events := eventsOfType(resolution.Events, EventTypeRevoltAppeased)
	if len(events) != 1 || events[0].Reason != "paid" || events[0].Cost != 4 {
		t.Fatalf("appeasement events = %#v, want one paid event costing 4 (2^2)", events)
	}
}

func TestPaidAppeasementIsRejectedWithoutEnoughResources(t *testing.T) {
	state := appeasementState(t)
	stock := state.TerritoryStates["AAA"]
	stock.Resources = 3
	state.TerritoryStates["AAA"] = stock
	resolution, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeasePaid, "N1", "BBB"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if !hasArmy(resolution.State, "A8") {
		t.Fatalf("rebel army removed although the price was not paid")
	}
}

func TestBishopCannotPayOutsideTheirBishopric(t *testing.T) {
	state := appeasementState(t)
	requireAppeasementRejected(t, state, appease(models.DeckOrderTypeAppeasePaid, "N1", "CCC"), "appeasement_requires_own_bishopric", "A9")
}

func TestCardinalPaysAnywhere(t *testing.T) {
	state := appeasementState(t)
	state.Cardinals = []models.NobleID{"N1"}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeasePaid, "N1", "CCC"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if hasArmy(resolution.State, "A9") {
		t.Fatalf("rebel army survived the cardinal's paid appeasement")
	}
}

func TestRiteOnlyWorksInTheClericsRegion(t *testing.T) {
	state := appeasementState(t)
	requireAppeasementRejected(t, state, appease(models.DeckOrderTypeAppeaseRite, "N1", "CCC"), "appeasement_requires_own_region", "A9")
}

func TestRiteOutcomesFollowTheDie(t *testing.T) {
	seen := map[string]bool{}
	for turn := 1; turn <= 60 && len(seen) < 3; turn++ {
		state := appeasementState(t)
		state.Turn = turn
		if models.SeasonForTurn(turn) == models.SeasonWinter {
			continue
		}
		state.Season = models.SeasonForTurn(turn)
		resolution, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeaseRite, "N1", "BBB"))
		if err != nil {
			t.Fatalf("turn %d: %v", turn, err)
		}
		events := eventsOfType(resolution.Events, EventTypeRevoltAppeased)
		if len(events) != 1 {
			t.Fatalf("turn %d: events = %#v", turn, events)
		}
		reason := events[0].Reason
		seen[reason] = true
		rebels := hasArmy(resolution.State, "A8")
		cleric := false
		for _, noble := range resolution.State.Nobles {
			cleric = cleric || noble.ID == "N1"
		}
		switch reason {
		case "succeeded":
			if rebels || !cleric {
				t.Fatalf("succeeded: rebels=%v cleric=%v", rebels, cleric)
			}
		case "failed":
			if !rebels || !cleric {
				t.Fatalf("failed: rebels=%v cleric=%v", rebels, cleric)
			}
		case "failed_death":
			if !rebels || cleric {
				t.Fatalf("failed_death: rebels=%v cleric=%v", rebels, cleric)
			}
			if removed := resolution.State.RemovedNobles; len(removed) != 1 || removed[0].Cause != models.DeathCauseMartyr {
				t.Fatalf("removed = %#v, want a martyr", removed)
			}
		default:
			t.Fatalf("unexpected reason %q", reason)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("outcomes seen = %v, want succeeded, failed and failed_death", seen)
	}
}

func TestAppeasementWithoutRebelArmyIsRejected(t *testing.T) {
	state := appeasementState(t)
	state.Cardinals = []models.NobleID{"N1"}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeasePaid, "N1", "AAA"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if containsEvent(resolution.Events, EventTypeRevoltAppeased) {
		t.Fatalf("appeasement of a territory without rebels must be rejected")
	}
}

func TestAbbessRitesWhereverSheStandsButCannotPay(t *testing.T) {
	state := appeasementState(t)
	state.Bishops, state.Nobles[0].Dignities = nil, []models.Dignity{models.DignityAbbess}
	state.Nobles[0].Sex, state.Nobles[0].AbbeyRegion = models.SexFemale, "CCC"
	if _, err := ResolveWithDeckOrders(state, testBalance(), appease(models.DeckOrderTypeAppeaseRite, "N1", "BBB")); err != nil {
		t.Fatalf("rite in the region where she stands: %v", err)
	}
	requireAppeasementRejected(t, state, appease(models.DeckOrderTypeAppeaseRite, "N1", "CCC"), "appeasement_requires_own_region", "A9")
	requireAppeasementRejected(t, state, appease(models.DeckOrderTypeAppeasePaid, "N1", "BBB"), "appeasement_requires_cleric", "A8")
}

// requireAppeasementRejected resolves the orders and checks that only the
// appeasement is rejected, with the reason: the turn still resolves.
func requireAppeasementRejected(t *testing.T, state *models.GameState, orders map[models.PlayerID][]models.DeckOrder, reason string, standing models.ArmyID) {
	t.Helper()
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("a rejected appeasement must not fail the turn: %v", err)
	}
	rejected := false
	for _, event := range eventsOfType(resolution.Events, EventTypeRejected) {
		rejected = rejected || event.Reason == reason
	}
	if !rejected {
		t.Fatalf("no %q rejection in %#v", reason, resolution.Events)
	}
	if !hasArmy(resolution.State, standing) {
		t.Fatalf("rebel army %s must stand after a rejected appeasement", standing)
	}
}

func TestOneAppeasementPerNobleAndAnInvalidOneDoesNotBlockTheOthers(t *testing.T) {
	state := appeasementState(t)
	state.Cardinals = []models.NobleID{"N1"}
	orders := map[models.PlayerID][]models.DeckOrder{"P1": {
		{ID: "O1", Type: models.DeckOrderTypeAppeasePaid, TargetNobleID: "N1", TargetTerritoryID: "CCC"},
		{ID: "O2", Type: models.DeckOrderTypeAppeasePaid, TargetNobleID: "N1", TargetTerritoryID: "BBB"},
		{ID: "O3", Type: models.DeckOrderTypeAppeasePaid, TargetNobleID: "NOPE", TargetTerritoryID: "BBB"},
	}}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if hasArmy(resolution.State, "A9") || !hasArmy(resolution.State, "A8") {
		t.Fatalf("only the first appeasement of the noble must apply")
	}
	if got := len(eventsOfType(resolution.Events, EventTypeRejected)); got != 2 {
		t.Fatalf("rejections = %d, want 2", got)
	}
}
