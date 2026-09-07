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
