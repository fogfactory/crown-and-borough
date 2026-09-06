package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestNobleStatusOrdersApply(t *testing.T) {
	for _, test := range []struct {
		name   string
		order  ExecutableOrder
		status models.NobleStatus
	}{
		{name: "hostage", order: hostageOrder{order: models.WinterOrder{ID: "O1", NobleCode: "NOB"}}, status: models.NobleStatusHostage},
		{name: "dungeon", order: dungeonOrder{order: models.WinterOrder{ID: "O1", NobleCode: "NOB"}}, status: models.NobleStatusDungeon},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}})
			addNoble(state, "N1", "NOB", "P1", "AAA")
			setNobleStatus(state, "N1", models.NobleStatusHostage)
			ctx := newResolutionContext(state, testBalance())
			test.order.Apply(&ExecutionContext{resolution: ctx, playerID: "P2"})
			if got := state.Nobles[0].Status; got != test.status {
				t.Fatalf("status = %q, want %q", got, test.status)
			}
			if len(eventsOfType(ctx.events, EventTypeCapture)) != 1 {
				t.Fatalf("capture events = %#v, want one event", ctx.events)
			}
		})
	}
}
