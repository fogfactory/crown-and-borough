package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestEmitAbandonedControlReportsDepartedPositionalControl checks #215's core
// rule directly on emitAbandonedControl: a territory outside every fief and
// capital, controlled at the start by a player whose army is no longer there,
// has no controller any more, and it is reported with a control_changed
// event, reason "abandoned", only because it carries an infrastructure.
func TestEmitAbandonedControlReportsDepartedPositionalControl(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())
	// P1 held both territories when the turn started; its army has left.
	ctx.startControl["AAA"] = "P1"
	ctx.startControl["BBB"] = "P1"

	ctx.emitAbandonedControl(ctx.startControl)

	for _, territoryID := range []models.TerritoryID{"AAA", "BBB"} {
		if owner, controlled := ctx.controllerNow(territoryID); controlled {
			t.Errorf("%s controller = %q, want none (no fief/capital/army anchor)", territoryID, owner)
		}
	}
	changed := eventsOfType(ctx.events, EventTypeControlChanged)
	if len(changed) != 1 {
		t.Fatalf("control_changed events = %#v, want exactly one (AAA carries a village, BBB carries nothing)", ctx.events)
	}
	if changed[0].TerritoryID != "AAA" || changed[0].PreviousOwnerID != "P1" || changed[0].OwnerID != "" || changed[0].Reason != "abandoned" {
		t.Errorf("control_changed event = %#v, want AAA released from P1, reason abandoned", changed[0])
	}
}

// TestControlKeepsCapitalWithoutArmy checks #215: a player's own capital is a
// permanent anchor and stays controlled even with no army on it, unlike an
// ordinary territory.
func TestControlKeepsCapitalWithoutArmy(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.emitAbandonedControl(ctx.startControl)

	if owner, controlled := ctx.controllerNow("AAA"); !controlled || owner != "P1" {
		t.Errorf("AAA controller = %q (%t), want P1 (own capital, a permanent anchor)", owner, controlled)
	}
	if len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no control_changed for an anchored capital", ctx.events)
	}
}

// TestControlKeepsFiefMemberWithoutArmy is a non-regression check for #196: a
// fief member, capital or not, stays controlled by the fief's owner with no
// army on it, exactly like before #215.
func TestControlKeepsFiefMemberWithoutArmy(t *testing.T) {
	state := fiefControlTestState(t, nil)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.emitAbandonedControl(ctx.startControl)

	for _, territoryID := range []models.TerritoryID{"AAA", "BBB", "CCC"} {
		if owner, controlled := ctx.controllerNow(territoryID); !controlled || owner != "P1" {
			t.Errorf("%s controller = %q (%t), want P1 (fief member, unaffected by #215)", territoryID, owner, controlled)
		}
	}
	if len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no control_changed for fief members", ctx.events)
	}
}

// TestControlKeepsArmyOccupiedTerritory checks #215's third anchor: a
// territory outside every fief and capital is controlled while one of the
// controller's own armies stands on it.
func TestControlKeepsArmyOccupiedTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.emitAbandonedControl(ctx.startControl)

	if owner, controlled := ctx.controllerNow("AAA"); !controlled || owner != "P1" {
		t.Errorf("AAA controller = %q (%t), want P1 (A1 still stands on it)", owner, controlled)
	}
	if len(ctx.events) != 0 {
		t.Errorf("events = %#v, want none (A1 still holds AAA)", ctx.events)
	}
}

// TestControlOfUncontrolledTerritoryIsNeverAbandoned checks that a territory
// nobody controlled has nothing to abandon.
func TestControlOfUncontrolledTerritoryIsNeverAbandoned(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.emitAbandonedControl(ctx.startControl)

	if len(ctx.events) != 0 {
		t.Errorf("events = %#v, want none (nothing to abandon)", ctx.events)
	}
}

// TestNeutralArmyOccupiesButNeverControls checks that a revolt army on a
// territory outside every fief and capital leaves it uncontrolled, while on an
// anchored territory it occupies it against its controller.
func TestNeutralArmyOccupiesButNeverControls(t *testing.T) {
	state := fiefControlTestState(t, nil)
	state.Territories = append(state.Territories, territory("DDD", "DDD", "CCC"))
	for index := range state.Territories {
		if state.Territories[index].ID == "CCC" {
			state.Territories[index].Adjacencies = append(state.Territories[index].Adjacencies, "DDD")
		}
	}
	state.TerritoryStates["DDD"] = models.TerritoryState{}
	placeArmyAt(state, "A1", models.NeutralPlayerID, "BBB", 2)
	placeArmyAt(state, "A2", models.NeutralPlayerID, "DDD", 2)
	state.NextArmyID = nextArmyID(state.Armies)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	if owner, controlled := ctx.controllerNow("DDD"); controlled {
		t.Errorf("DDD controller = %q, want none (a revolt army administers nothing)", owner)
	}
	if owner, controlled := ctx.controllerNow("BBB"); !controlled || owner != "P1" {
		t.Errorf("BBB controller = %q (%t), want P1 (fief member)", owner, controlled)
	}
	if !ctx.occupiedAgainstController("BBB", ctx.currentArmyAt("BBB")) {
		t.Errorf("BBB is not occupied against P1, want the revolt army to occupy it")
	}
	if ctx.occupiedAgainstController("DDD", ctx.currentArmyAt("DDD")) {
		t.Errorf("DDD is occupied against nobody, want no occupation without a controller")
	}
}

// TestControllerAtStartIsFrozen checks that the start-of-resolution snapshot
// does not follow armies and fiefs as the turn mutates them, while the current
// derivation does.
func TestControllerAtStartIsFrozen(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.armiesByID["A1"].TerritoryID = "BBB"
	ctx.rebuildIndexes()

	if owner, controlled := ctx.controllerAtStart("AAA"); !controlled || owner != "P1" {
		t.Errorf("AAA controller at start = %q (%t), want P1", owner, controlled)
	}
	if _, controlled := ctx.controllerAtStart("BBB"); controlled {
		t.Errorf("BBB is controlled at start, want it uncontrolled")
	}
	if _, controlled := ctx.controllerNow("AAA"); controlled {
		t.Errorf("AAA is controlled now, want it uncontrolled once A1 left")
	}
	if owner, controlled := ctx.controllerNow("BBB"); !controlled || owner != "P1" {
		t.Errorf("BBB controller now = %q (%t), want P1", owner, controlled)
	}
}
