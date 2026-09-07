package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestElectCapitalOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, nil)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	ctx := newResolutionContext(state, testBalance())
	electCapitalOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	if player := ctx.playerByID("P1"); player == nil || player.CapitalCastleID == nil || *player.CapitalCastleID != "I1" {
		t.Fatalf("capital = %#v, want I1", player)
	}
	if len(eventsOfType(ctx.events, EventTypeCapitalElected)) != 1 {
		t.Fatalf("capital events = %#v, want one event", ctx.events)
	}
}
