package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLiberateNobleOrderApply(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA"), territory("BBB", "BBB")}, []models.Army{
		{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
		{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
	})
	addNoble(state, "N1", "NOB", "P1", "AAA")
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	setCapital(state, "P1", "I1")
	ctx := newResolutionContext(state, testBalance())
	liberateNobleOrder{order: models.WinterOrder{ID: "O1", NobleCode: "NOB"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P2"})
	if noble := ctx.noblesByID["N1"]; noble.Status != models.NobleStatusFree || noble.LocationID != "BBB" {
		t.Fatalf("noble = %#v, want free at BBB", noble)
	}
	if len(eventsOfType(ctx.events, EventTypeLiberation)) != 1 {
		t.Fatalf("liberation events = %#v, want one event", ctx.events)
	}
}
