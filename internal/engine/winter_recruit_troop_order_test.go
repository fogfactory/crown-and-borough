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
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: state.TerritoryStates["AAA"].Army, Resources: 1, Infrastructures: infraPointer("I1")}
	ctx := newResolutionContext(state, testBalance())
	recruitTroopOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	if got := state.Armies[0].Size; got != 2 {
		t.Fatalf("army size = %d, want 2", got)
	}
	if len(eventsOfType(ctx.events, EventTypeRecruit)) != 1 {
		t.Fatalf("recruit events = %#v, want one event", ctx.events)
	}
}

// TestRecruitTroopOrderRejectedOnAbandonedTerritory checks #215: a territory
// a player once controlled positionally, outside every fief and capital,
// rejects R T once released -- exactly like it always rejected a never-held
// one -- as soon as its army leaves and no anchor keeps the territory.
func TestRecruitTroopOrderRejectedOnAbandonedTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if owner := controllerOf(resolution.State, "AAA"); owner != nil {
		t.Fatalf("AAA owner = %v, want nil after A1 departed (test setup drifted)", owner)
	}

	winterState := resolution.State
	winterState.Turn++
	winterState.Season = models.SeasonWinter
	ctx := newResolutionContext(winterState, testBalance())
	recruitTroopOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "territory_not_controlled" {
		t.Fatalf("events = %#v, want one rejection with reason territory_not_controlled", ctx.events)
	}
}
