package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestRecruitNobleOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: state.TerritoryStates["AAA"].OwnerID, Army: state.TerritoryStates["AAA"].Army, Resources: 2, Infrastructures: []models.InfraID{"I1"}}
	ctx := newResolutionContext(state, testBalance())
	recruitNobleOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", firstNameRNG: newWinterRNG(state.Seed, state.Turn)})
	if len(state.Nobles) != 1 {
		t.Fatalf("nobles = %d, want 1", len(state.Nobles))
	}
	if len(eventsOfType(ctx.events, EventTypeRecruit)) != 1 {
		t.Fatalf("recruit events = %#v, want one event", ctx.events)
	}
}
