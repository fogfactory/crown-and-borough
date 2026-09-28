package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestReleaseUnanchoredControlReleasesDepartedPositionalControl checks #215's
// core rule directly on releaseUnanchoredControl: a territory outside every
// fief and capital, controlled by a player whose army is no longer there,
// reverts to neutral (OwnerID nil), and reports it with a control_changed
// event, reason "abandoned", only because it carries an infrastructure.
func TestReleaseUnanchoredControlReleasesDepartedPositionalControl(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "BBB", "P1")
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.releaseUnanchoredControl()

	if owner := ctx.state.TerritoryStates["AAA"].OwnerID; owner != nil {
		t.Errorf("AAA owner = %v, want nil (released, no fief/capital/army anchor)", owner)
	}
	if owner := ctx.state.TerritoryStates["BBB"].OwnerID; owner != nil {
		t.Errorf("BBB owner = %v, want nil (released, no fief/capital/army anchor)", owner)
	}
	changed := eventsOfType(ctx.events, EventTypeControlChanged)
	if len(changed) != 1 {
		t.Fatalf("control_changed events = %#v, want exactly one (AAA carries a village, BBB carries nothing)", ctx.events)
	}
	if changed[0].TerritoryID != "AAA" || changed[0].PreviousOwnerID != "P1" || changed[0].OwnerID != "" || changed[0].Reason != "abandoned" {
		t.Errorf("control_changed event = %#v, want AAA released from P1, reason abandoned", changed[0])
	}
}

// TestReleaseUnanchoredControlKeepsCapitalWithoutArmy checks #215: a
// player's own capital is a permanent anchor and stays controlled even with
// no army on it, unlike an ordinary territory.
func TestReleaseUnanchoredControlKeepsCapitalWithoutArmy(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.releaseUnanchoredControl()

	if owner := ctx.state.TerritoryStates["AAA"].OwnerID; owner == nil || *owner != "P1" {
		t.Errorf("AAA owner = %v, want P1 (own capital, a permanent anchor)", owner)
	}
	if len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no control_changed for an anchored capital", ctx.events)
	}
}

// TestReleaseUnanchoredControlKeepsFiefMemberWithoutArmy is a non-regression
// check for #196: a fief member, capital or not, stays controlled by the
// fief's owner with no army on it, exactly like before #215.
func TestReleaseUnanchoredControlKeepsFiefMemberWithoutArmy(t *testing.T) {
	state := fiefControlTestState(t, nil)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.releaseUnanchoredControl()

	for _, territoryID := range []models.TerritoryID{"AAA", "BBB", "CCC"} {
		if owner := ctx.state.TerritoryStates[territoryID].OwnerID; owner == nil || *owner != "P1" {
			t.Errorf("%s owner = %v, want P1 (fief member, unaffected by #215)", territoryID, owner)
		}
	}
	if len(eventsOfType(ctx.events, EventTypeControlChanged)) != 0 {
		t.Errorf("events = %#v, want no control_changed for fief members", ctx.events)
	}
}

// TestReleaseUnanchoredControlKeepsArmyOccupiedTerritory checks #215's third
// anchor: a territory outside every fief and capital stays controlled while
// one of the controller's own armies currently stands on it.
func TestReleaseUnanchoredControlKeepsArmyOccupiedTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.releaseUnanchoredControl()

	if owner := ctx.state.TerritoryStates["AAA"].OwnerID; owner == nil || *owner != "P1" {
		t.Errorf("AAA owner = %v, want P1 (A1 still stands on it)", owner)
	}
}

// TestReleaseUnanchoredControlIsIdempotent checks that running the pass
// twice in a row (as ResolveWinter's two call sites could, given a state
// already neutral) leaves an already-released territory untouched and emits
// nothing the second time.
func TestReleaseUnanchoredControlIsIdempotent(t *testing.T) {
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA")},
		nil,
	)
	validateTestState(t, state)
	ctx := newResolutionContext(state, testBalance())

	ctx.releaseUnanchoredControl()
	ctx.releaseUnanchoredControl()

	if owner := ctx.state.TerritoryStates["AAA"].OwnerID; owner != nil {
		t.Errorf("AAA owner = %v, want nil (was already neutral)", owner)
	}
	if len(ctx.events) != 0 {
		t.Errorf("events = %#v, want none (nothing to release)", ctx.events)
	}
}
