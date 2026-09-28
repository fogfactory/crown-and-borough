package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestBuildOrderApply verifies that C C on a controlled village fortifies it
// in place (same InfraID, InfraType stays village) rather than replacing it
// with a castle (#193).
func TestBuildOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || infrastructure.ID != "I0" || infrastructure.Type != models.InfraTypeVillage || !infrastructure.Fortified {
		t.Fatalf("infrastructure = %#v, events = %#v, want fortified village I0", infrastructure, ctx.events)
	}
	fortifications := eventsOfType(ctx.events, EventTypeFortify)
	if len(fortifications) != 1 || fortifications[0].ResourceSpent != testBalance().Costs.Castle {
		t.Fatalf("fortify events = %#v, want one event spending %d", ctx.events, testBalance().Costs.Castle)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("stock = %d, want 0 after spending the fortification cost", got)
	}
}

// TestBuildOrderRejectsFortifyingAnAlreadyFortifiedVillage verifies that a
// second C C on an already-fortified village is rejected with no charge
// (#193).
func TestBuildOrderRejectsFortifyingAnAlreadyFortifiedVillage(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA", Fortified: true})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "village_already_fortified" {
		t.Fatalf("rejected events = %#v, want village_already_fortified", rejected)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10 {
		t.Errorf("stock = %d, want unchanged 10 (no prelevement)", got)
	}
}

// TestBuildOrderRejectsOccupiedTerritory verifies that a winter build order
// on a territory occupied against its controller is rejected without any
// prelevement (titres.md, #196).
func TestBuildOrderRejectsOccupiedTerritory(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})

	if infrastructure := ctx.infrastructureAt("AAA"); infrastructure != nil {
		t.Fatalf("infrastructure = %#v, want none built on an occupied territory", infrastructure)
	}
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "territory_occupied_by_other_player" {
		t.Fatalf("rejected events = %#v, want territory_occupied_by_other_player", rejected)
	}
	if got := ctx.state.TerritoryStates["AAA"].Resources; got != 10 {
		t.Errorf("stock = %d, want unchanged 10 (no prelevement)", got)
	}
}
