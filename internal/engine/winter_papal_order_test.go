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

func TestPapalEventsCarryTheOrderID(t *testing.T) {
	state := papalTestState(t)
	state.Excommunications = []models.Excommunication{{Noble: "N5", Reason: models.ExcommunicationPapal, By: "N1"}}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			papalOrder("O1", models.WinterOrderTypeLiftExcommunication, "SAM"),
			papalOrder("O2", models.WinterOrderTypeExcommunicate, "LEO"),
		},
	})
	for _, kind := range []EventType{EventTypeExcommunication, EventTypeExcommunicationLifted} {
		events := eventsOfType(resolution.Events, kind)
		if len(events) != 1 || events[0].OrderID == "" {
			t.Errorf("%s events = %+v, want one carrying its order id", kind, events)
		}
	}
}

func TestWinterAidsPapalAndCardinal(t *testing.T) {
	state := papalTestState(t)
	state.Excommunications = []models.Excommunication{{Noble: "N5", Reason: models.ExcommunicationPapal, By: "N1"}}
	state.Cardinals = []models.NobleID{"N2"}
	aids := ForecastWinterAids(state, testBalance(), "P1")
	if aids == nil {
		t.Fatal("aids = nil in winter")
	}
	targets := map[models.NobleCode]models.NobleCode{}
	for _, target := range aids.Excommunicable {
		targets[target.Code] = target.Blocker
	}
	if _, ok := targets["HUG"]; ok {
		t.Error("the pope can excommunicate himself")
	}
	if blocker, ok := targets["LEO"]; !ok || blocker != "SAM" {
		t.Errorf("LEO target = %q, %v; want blocked by SAM", blocker, ok)
	}
	if _, ok := targets["ABE"]; !ok || targets["ABE"] != "" {
		t.Errorf("ABE should be free to excommunicate: %v", targets)
	}
	if len(aids.Liftable) != 1 || aids.Liftable[0] != "SAM" {
		t.Errorf("liftable = %v, want [SAM]", aids.Liftable)
	}
	if len(aids.BuyableCardinals) != 0 {
		t.Errorf("buyable cardinals = %v, want none for a player without bishop", aids.BuyableCardinals)
	}
	other := ForecastWinterAids(state, testBalance(), "P3")
	if len(other.Excommunicable) != 0 || len(other.Liftable) != 0 {
		t.Errorf("a non-pope gets papal aids: %+v", other)
	}
}

func TestWinterAidsFiefSitesNeedThreeConnectedControlledTerritoriesAndACastle(t *testing.T) {
	state := electionTestState(t)
	aids := ForecastWinterAids(state, testBalance(), "P1")
	if len(aids.FiefSites) != 0 {
		t.Errorf("sites = %+v, want none with two controlled territories", aids.FiefSites)
	}
	if ForecastWinterAids(&models.GameState{Season: models.SeasonSpring}, testBalance(), "P1") != nil {
		t.Error("aids outside winter")
	}
}
