package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestBuildOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 10)
	ctx := newResolutionContext(state, testBalance())
	buildOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA", InfraType: models.InfraTypeCastle}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	infrastructure := ctx.infrastructureAt("AAA")
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle {
		t.Fatalf("infrastructure = %#v, events = %#v, want castle", infrastructure, ctx.events)
	}
	if len(eventsOfType(ctx.events, EventTypeBuild)) != 1 {
		t.Fatalf("build events = %#v, want one event", ctx.events)
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
