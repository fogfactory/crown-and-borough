package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestRecruitTroopOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	state.TerritoryStates["AAA"] = models.TerritoryState{OwnerID: state.TerritoryStates["AAA"].OwnerID, Army: state.TerritoryStates["AAA"].Army, Resources: 1, Infrastructures: []models.InfraID{"I1"}}
	ctx := newResolutionContext(state, testBalance())
	recruitTroopOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	if got := state.Armies[0].Size; got != 2 {
		t.Fatalf("army size = %d, want 2", got)
	}
	if len(eventsOfType(ctx.events, EventTypeRecruit)) != 1 {
		t.Fatalf("recruit events = %#v, want one event", ctx.events)
	}
}
