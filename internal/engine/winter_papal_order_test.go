package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func papalTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := electionTestState(t)
	addNoble(state, "N5", "SAM", "P2", "CCC")
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N1"}, {Region: "R2", Noble: "N2"}}
	state.Cardinals = []models.NobleID{"N1", "N2"}
	pope := models.NobleID("N1")
	state.Pope = &pope
	return state
}

func papalOrder(id models.OrderID, kind models.WinterOrderType, noble models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: kind, NobleCode: noble}
}

func TestPopeExcommunicatesAndStripsTitles(t *testing.T) {
	state := papalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {papalOrder("O1", models.WinterOrderTypeExcommunicate, "LEO")},
	})
	got := resolution.State
	excommunication, ok := got.ExcommunicationOf("N2")
	if !ok || excommunication.Reason != models.ExcommunicationPapal || excommunication.By != "N1" {
		t.Fatalf("excommunication = %+v, %v", excommunication, ok)
	}
	if got.IsCardinal("N2") || got.IsBishop("N2") {
		t.Error("excommunicated cardinal kept a title")
	}
	if len(eventsOfType(resolution.Events, EventTypeExcommunication)) != 1 {
		t.Errorf("events = %+v, want one excommunication", resolution.Events)
	}
}

func TestExcommunicationLimits(t *testing.T) {
	state := papalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			papalOrder("O1", models.WinterOrderTypeExcommunicate, "ABE"),
			papalOrder("O2", models.WinterOrderTypeExcommunicate, "LEO"),
			papalOrder("O3", models.WinterOrderTypeExcommunicate, "OTO"),
		},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "" || reasons["O2"] != "excommunication_limit" || reasons["O3"] != "excommunication_limit" {
		t.Errorf("rejections = %v", reasons)
	}
}

func TestOneExcommunicatedAtATimePerOpponent(t *testing.T) {
	state := papalTestState(t)
	state.Excommunications = []models.Excommunication{{Noble: "N5", Reason: models.ExcommunicationPapal, By: "N1"}}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {papalOrder("O1", models.WinterOrderTypeExcommunicate, "LEO")},
	})
	if got := electionRejections(resolution.Events)["O1"]; got != "excommunication_slot_taken" {
		t.Errorf("reason = %q, want excommunication_slot_taken", got)
	}
	resolution = resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			papalOrder("O1", models.WinterOrderTypeLiftExcommunication, "SAM"),
			papalOrder("O2", models.WinterOrderTypeExcommunicate, "LEO"),
		},
	})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Errorf("rejections = %v, want none after the lift", reasons)
	}
	if _, ok := resolution.State.ExcommunicationOf("N5"); ok {
		t.Error("N5 still excommunicated after lift")
	}
}

func TestLiftDoesNotRestoreTitlesAndExOfficioIsNotLiftable(t *testing.T) {
	state := papalTestState(t)
	state.Excommunications = []models.Excommunication{{Noble: "N5", Reason: models.ExcommunicationExOfficio}}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			papalOrder("O1", models.WinterOrderTypeLiftExcommunication, "SAM"),
			papalOrder("O2", models.WinterOrderTypeLiftExcommunication, "ABE"),
		},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "excommunication_not_liftable" || reasons["O2"] != "not_excommunicated" {
		t.Errorf("rejections = %v", reasons)
	}
}

func TestOnlyActivePopeCanSanction(t *testing.T) {
	state := papalTestState(t)
	setNobleStatus(state, "N1", models.NobleStatusDungeon)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {papalOrder("O1", models.WinterOrderTypeExcommunicate, "LEO")},
		"P2": {papalOrder("O2", models.WinterOrderTypeExcommunicate, "ABE")},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "not_pope" || reasons["O2"] != "not_pope" {
		t.Errorf("rejections = %v, want not_pope twice", reasons)
	}
}

func TestPopeCannotExcommunicateSelf(t *testing.T) {
	state := papalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {papalOrder("O1", models.WinterOrderTypeExcommunicate, "HUG")},
	})
	if got := electionRejections(resolution.Events)["O1"]; got != "cannot_excommunicate_self" {
		t.Errorf("reason = %q", got)
	}
}

func TestPopeDeathEndsPapalExcommunications(t *testing.T) {
	state := papalTestState(t)
	addNoble(state, "N6", "ELO", "P3", "DDD")
	state.Excommunications = []models.Excommunication{
		{Noble: "N5", Reason: models.ExcommunicationPapal, By: "N1"},
		{Noble: "N6", Reason: models.ExcommunicationExOfficio},
	}
	ctx := newResolutionContext(state, testBalance())
	ctx.executeNoble(*ctx.noblesByID["N1"])
	if state.Pope != nil {
		t.Error("the throne is not vacant after the pope's death")
	}
	if _, ok := state.ExcommunicationOf("N5"); ok {
		t.Error("papal excommunication outlived the pope")
	}
	if _, ok := state.ExcommunicationOf("N6"); !ok {
		t.Error("ex officio excommunication ended with the pope")
	}
	if err := state.Validate(); err != nil {
		t.Errorf("state invalid after the death: %v", err)
	}
}
