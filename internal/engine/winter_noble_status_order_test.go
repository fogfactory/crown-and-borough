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

func TestDungeonOrderUnmasksChevalierDEon(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}})
	addNoble(state, "N1", "NOB", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityChevalierDEon}
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	ctx := newResolutionContext(state, testBalance())
	dungeonOrder{order: models.WinterOrder{ID: "O1", NobleCode: "NOB"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P2"})
	if !state.Nobles[0].EonUnmasked {
		t.Error("EonUnmasked = false, want true once imprisoned")
	}
	if events := eventsOfType(ctx.events, EventTypeEonUnmasked); len(events) != 1 || events[0].NobleID != "N1" {
		t.Errorf("eon_unmasked events = %+v, want one on N1", events)
	}
}

func TestUnmaskedEonIsExcommunicatedAndLosesReligiousTitles(t *testing.T) {
	state := winterTestState(t, []models.Territory{territory("AAA", "AAA")}, []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}})
	addNoble(state, "N1", "NOB", "P1", "AAA")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityChevalierDEon}
	state.Regions = []models.Region{{ID: "ROS", Territories: []models.TerritoryID{"AAA"}}}
	pope := models.NobleID("N1")
	state.Bishops = []models.Bishop{{Region: "ROS", Noble: "N1"}}
	state.Cardinals = []models.NobleID{"N1"}
	state.Pope = &pope
	ctx := newResolutionContext(state, testBalance())
	ctx.unmaskEon(&state.Nobles[0], 0)
	excommunication, ok := state.ExcommunicationOf("N1")
	if !ok || excommunication.Reason != models.ExcommunicationExOfficio {
		t.Fatalf("excommunication = %+v, %v; want ex officio", excommunication, ok)
	}
	if got := state.ReligiousTitleOf("N1"); got != models.ReligiousTitleNone || state.Pope != nil {
		t.Errorf("titles remain: %q pope=%v", got, state.Pope)
	}
}
