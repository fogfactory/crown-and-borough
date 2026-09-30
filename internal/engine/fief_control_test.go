package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// fiefControlTestState builds a three-territory fief (AAA capital, BBB, CCC)
// owned by P1, with no army anywhere yet: individual test cases place armies
// directly on ctx.state.Armies to control whether they existed at the start
// of the turn (startArmiesByID) or only appeared this turn (see
// updateTerritorialControl's occupation pass).
func fiefControlTestState(t *testing.T, extraArmies []models.Army) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		extraArmies,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	return state
}

// placeCurrentArmyOnly adds an army straight to ctx.state.Armies (and
// refreshes the current-state indexes) without touching startArmiesByID, to
// simulate an army that only reaches territoryID this turn - the same
// after-movement state updateTerritorialControl actually observes in
// progressChainsAndControl.
func placeCurrentArmyOnly(ctx *resolutionContext, armyID models.ArmyID, ownerID models.PlayerID, territoryID models.TerritoryID, size int) {
	ctx.state.Armies = append(ctx.state.Armies, models.Army{ID: armyID, OwnerID: ownerID, TerritoryID: territoryID, Size: size})
	ctx.rebuildIndexes()
}

// TestFiefMemberOccupationDoesNotChangeControl verifies that an enemy army
// newly stopping on a non-capital fief member occupies it without changing
// its controller, and reports the occupation exactly once (titres.md "Contrôle
// et occupation", #196).
func TestFiefMemberOccupationDoesNotChangeControl(t *testing.T) {
	state := fiefControlTestState(t, nil)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	placeCurrentArmyOnly(ctx, "A1", "P2", "BBB", 3)

	updateTerritorialControl(ctx)

	if got, _ := ctx.controllerNow("BBB"); got != "P1" {
		t.Fatalf("BBB controller = %q, want P1 (occupation does not change control)", got)
	}
	if len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no control_changed", ctx.events)
	}
	occupied := eventsOfType(ctx.events, EventTypeFiefMemberOccupied)
	if len(occupied) != 1 {
		t.Fatalf("occupied events = %#v, want exactly one", occupied)
	}
	event := occupied[0]
	if event.TerritoryID != "BBB" || event.FiefID != "F1" || event.OwnerID != "P1" || event.CaptorPlayerID != "P2" {
		t.Errorf("occupied event = %#v, want BBB/F1, owner P1, captor P2", event)
	}
}

// TestFiefMemberOccupationNotRepeatedWhileGarrisonStays verifies that a
// garrison already occupying a fief member at the start of the turn does not
// re-trigger the occupation event every turn it merely stays (titres.md).
func TestFiefMemberOccupationNotRepeatedWhileGarrisonStays(t *testing.T) {
	state := fiefControlTestState(t, []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 3}})
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	updateTerritorialControl(ctx)

	if len(eventsOfType(ctx.events, EventTypeFiefMemberOccupied)) != 0 {
		t.Errorf("events = %#v, want no repeated occupation event", ctx.events)
	}
	if got, _ := ctx.controllerNow("BBB"); got != "P1" {
		t.Fatalf("BBB controller = %q, want P1 (still occupied, not conquered)", got)
	}
}

// TestFiefMemberOccupationByRevolt verifies that a NEUTRAL revolt army
// stopping on a non-capital fief member counts as an occupation like any
// other player's army, without ever taking control (titres.md).
func TestFiefMemberOccupationByRevolt(t *testing.T) {
	state := fiefControlTestState(t, nil)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	placeCurrentArmyOnly(ctx, "A1", models.NeutralPlayerID, "BBB", 4)

	updateTerritorialControl(ctx)

	if got, _ := ctx.controllerNow("BBB"); got != "P1" {
		t.Fatalf("BBB controller = %q, want P1 (revolt never takes control)", got)
	}
	occupied := eventsOfType(ctx.events, EventTypeFiefMemberOccupied)
	if len(occupied) != 1 || occupied[0].CaptorPlayerID != models.NeutralPlayerID {
		t.Fatalf("occupied events = %#v, want one with captor NEUTRAL", occupied)
	}
}

// TestPositionalControlUnaffectedOutsideFief verifies that control outside
// any fief stays purely positional after the transitive-control change: an
// enemy army stopping on an ordinary territory still takes it over exactly
// as before (no regression, titres.md).
func TestPositionalControlUnaffectedOutsideFief(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		nil,
	)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	// P1 held AAA positionally when the turn started (its army has since
	// left); P2's army now stands there.
	ctx.startControl["AAA"] = "P1"
	placeCurrentArmyOnly(ctx, "A1", "P2", "AAA", 2)

	updateTerritorialControl(ctx)

	if got, _ := ctx.controllerNow("AAA"); got != "P2" {
		t.Fatalf("AAA controller = %q, want P2 (positional control)", got)
	}
	changed := eventsOfType(ctx.events, EventTypeControlChanged)
	if len(changed) != 1 || changed[0].PreviousOwnerID != "P1" || changed[0].OwnerID != "P2" {
		t.Fatalf("control_changed events = %#v", changed)
	}
	if len(eventsOfType(ctx.events, EventTypeFiefMemberOccupied)) != 0 {
		t.Errorf("events = %#v, want no fief occupation event outside any fief", ctx.events)
	}
}

// TestFiefCapitalCaptureThroughUpdateTerritorialControl verifies the full
// updateTerritorialControl pipeline (not transferFiefOnCapitalCapture called
// directly): capturing the capital transfers every other member the same
// pass, and the capital itself never also reports as a "fief member
// occupied" (that event is reserved for non-capital members, titres.md).
func TestFiefCapitalCaptureThroughUpdateTerritorialControl(t *testing.T) {
	state := fiefControlTestState(t, nil)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	placeCurrentArmyOnly(ctx, "A1", "P2", "AAA", 5)

	updateTerritorialControl(ctx)

	for _, territoryID := range []models.TerritoryID{"AAA", "BBB", "CCC"} {
		if got, _ := ctx.controllerNow(territoryID); got != "P2" {
			t.Errorf("%s controller = %q, want P2", territoryID, got)
		}
	}
	if len(ctx.state.Fiefs) != 1 || ctx.state.Fiefs[0].OwnerID != "P2" || ctx.state.Fiefs[0].HolderNobleID != nil {
		t.Fatalf("fief = %#v, want owner P2, vacant", ctx.state.Fiefs)
	}
	if len(eventsOfType(ctx.events, EventTypeFiefConquered)) != 1 {
		t.Fatalf("events = %#v, want exactly one fief_conquered", ctx.events)
	}
	transferred := eventsOfType(ctx.events, EventTypeControlChanged)
	if len(transferred) != 3 {
		t.Fatalf("control_changed events = %#v, want one for the capital plus one per other member", transferred)
	}
	if len(eventsOfType(ctx.events, EventTypeFiefMemberOccupied)) != 0 {
		t.Errorf("events = %#v, want no fief_member_occupied event for a capital capture", ctx.events)
	}
}

// TestFiefMemberOccupationReportedWhenCapitalFallsWithPreExistingGarrison
// verifies that a member already garrisoned by the former owner's own army
// before the turn still reports as newly occupied the turn its fief's
// capital falls to a new controller: the dedup guard must key off the
// controller changing, not just off the same army staying put.
func TestFiefMemberOccupationReportedWhenCapitalFallsWithPreExistingGarrison(t *testing.T) {
	state := fiefControlTestState(t, []models.Army{{ID: "G1", OwnerID: "P1", TerritoryID: "BBB", Size: 3}})
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	placeCurrentArmyOnly(ctx, "A1", "P2", "AAA", 5)

	updateTerritorialControl(ctx)

	if got, _ := ctx.controllerNow("BBB"); got != "P2" {
		t.Fatalf("BBB controller = %q, want P2 (transferred with the fallen capital)", got)
	}
	occupied := eventsOfType(ctx.events, EventTypeFiefMemberOccupied)
	if len(occupied) != 1 {
		t.Fatalf("occupied events = %#v, want exactly one for BBB under its new controller", occupied)
	}
	event := occupied[0]
	if event.TerritoryID != "BBB" || event.FiefID != "F1" || event.OwnerID != "P2" || event.CaptorPlayerID != "P1" {
		t.Errorf("occupied event = %#v, want BBB/F1, owner P2, captor P1", event)
	}
}
