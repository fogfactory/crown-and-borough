package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestRecruitNobleOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: state.TerritoryStates["AAA"].Army, Resources: 2, Infrastructures: infraPointer("I1")}
	ctx := newResolutionContext(state, testBalance())
	recruitNobleOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", firstNameRNG: newWinterRNG(state.Seed, state.Turn)})
	if len(state.Nobles) != 1 {
		t.Fatalf("nobles = %d, want 1", len(state.Nobles))
	}
	if len(eventsOfType(ctx.events, EventTypeRecruit)) != 1 {
		t.Fatalf("recruit events = %#v, want one event", ctx.events)
	}
}

func TestRecruitNobleOrderRejectsFirstNameReservedByRemovedNoble(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: state.TerritoryStates["AAA"].Army, Resources: 2, Infrastructures: infraPointer("I1")}
	addNoble(state, "N1", "ADE", "P1", "AAA")
	// testBalance's three first names are ADE, GUI and MAH: ADE is held by a
	// live noble, GUI and MAH by dead nobles whose code must stay reserved
	// just like their id (specs/succession.md § Lignée).
	state.RemovedNobles = []models.RemovedNoble{
		{ID: "N8", Sex: models.SexMale, Code: "GUI", OwnerID: "P1", Cause: models.DeathCauseNatural},
		{ID: "N9", Sex: models.SexMale, Code: "MAH", OwnerID: "P1", Cause: models.DeathCauseNatural},
	}
	ctx := newResolutionContext(state, testBalance())
	recruitNobleOrder{order: models.WinterOrder{ID: "O1", TerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", firstNameRNG: newWinterRNG(state.Seed, state.Turn)})
	if len(state.Nobles) != 1 {
		t.Fatalf("nobles = %#v, want still only the pre-existing noble", state.Nobles)
	}
	rejected := eventsOfType(ctx.events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "no_available_first_name" {
		t.Fatalf("events = %#v, want a single no_available_first_name rejection", ctx.events)
	}
}
