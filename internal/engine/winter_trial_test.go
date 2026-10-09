package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func trialBy(id models.OrderID, cardinal, target models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeTrial, NobleCode: cardinal, TargetCode: target}
}

// winterTrialState makes HUG (N1) and OTO (N4) of P1 cardinals, LEO (N2, P2)
// an excommunicated man and ABE (N3, P3) a free man.
func winterTrialState(t *testing.T) *models.GameState {
	t.Helper()
	state := cardinalTestState(t)
	state.Cardinals = []models.NobleID{"N1", "N4"}
	state.Excommunicate(models.Excommunication{Noble: "N2", Reason: models.ExcommunicationExOfficio, Turn: 1})
	return state
}

func hasNoble(state *models.GameState, id models.NobleID) bool {
	for _, noble := range state.Nobles {
		if noble.ID == id {
			return true
		}
	}
	return false
}

func TestTwoCardinalsOfOnePlayerExecuteAnExcommunicatedNoble(t *testing.T) {
	state := winterTrialState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO")},
	})
	if hasNoble(resolution.State, "N2") {
		t.Fatalf("the excommunicated noble must be executed")
	}
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if len(trials) != 1 || trials[0].Reason != "trial_executed" || trials[0].Phase != winterPhase {
		t.Fatalf("trial events = %#v", trials)
	}
	if len(resolution.State.RemovedNobles) != 1 || resolution.State.RemovedNobles[0].Cause != models.DeathCauseExecution {
		t.Fatalf("removed = %#v", resolution.State.RemovedNobles)
	}
}

func TestTwoCardinalsOfTwoPlayersExecute(t *testing.T) {
	state := winterTrialState(t)
	state.Cardinals = []models.NobleID{"N1", "N4"}
	for i := range state.Nobles {
		if state.Nobles[i].ID == "N4" {
			state.Nobles[i].OwnerID = "P3"
		}
	}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO")},
		"P3": {trialBy("O1", "OTO", "LEO")},
	})
	if hasNoble(resolution.State, "N2") {
		t.Fatalf("the noble must be executed")
	}
}

func TestTrialNeedsTwoDistinctCardinals(t *testing.T) {
	state := winterTrialState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO")},
	})
	if !hasNoble(resolution.State, "N2") || len(eventsOfType(resolution.Events, EventTypeTrial)) != 0 {
		t.Fatalf("a single cardinal must not try anyone")
	}
}

func TestTrialOrdersOnDifferentTargetsDoNotAdd(t *testing.T) {
	state := winterTrialState(t)
	state.Excommunicate(models.Excommunication{Noble: "N3", Reason: models.ExcommunicationExOfficio, Turn: 1})
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "ABE")},
	})
	if !hasNoble(resolution.State, "N2") || !hasNoble(resolution.State, "N3") {
		t.Fatalf("orders on different targets must be without effect")
	}
}

func TestTrialOrderWithoutCardinalIsRejected(t *testing.T) {
	state := winterTrialState(t)
	state.Cardinals = []models.NobleID{"N1"}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO"), trialBy("O4", "HUG", "ABE")},
		"P2": {trialBy("O3", "HUG", "LEO")},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O2"] != "not_cardinal" || reasons["O3"] != "noble_not_owned" || reasons["O4"] != "trial_limit" || reasons["O1"] != "" {
		t.Fatalf("rejections = %#v", reasons)
	}
	if !hasNoble(resolution.State, "N2") {
		t.Fatalf("a single backing cardinal must not execute")
	}
}

func TestPopeAloneCannotTry(t *testing.T) {
	state := winterTrialState(t)
	state.Cardinals = nil
	state.Pope = ptrNoble("N1")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO")},
	})
	if !hasNoble(resolution.State, "N2") {
		t.Fatalf("the pope alone is never enough")
	}
}

func TestExcommunicatedCardinalDoesNotCount(t *testing.T) {
	state := winterTrialState(t)
	// P1's OTO is excommunicated too: only HUG remains.
	state.Excommunicate(models.Excommunication{Noble: "N4", Reason: models.ExcommunicationExOfficio, Turn: 1})
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO")},
	})
	if !hasNoble(resolution.State, "N2") {
		t.Fatalf("an excommunicated cardinal must not back a trial")
	}
}

func TestTrialOnFreeManIsUnfounded(t *testing.T) {
	state := winterTrialState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "ABE"), trialBy("O2", "OTO", "ABE")},
	})
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if !hasNoble(resolution.State, "N3") || len(trials) != 1 || trials[0].Reason != "trial_unfounded" {
		t.Fatalf("a man who is not excommunicated cannot be tried: %#v", trials)
	}
}

func TestTrialOnUnmaskedWitchSameWinter(t *testing.T) {
	state := winterTrialState(t)
	state.Excommunications = nil
	giveDignity(state, "N2", models.DignityWitch)
	// P1's third noble would be needed for the inquiry: HUG investigates while
	// OTO and HUG both back the trial.
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {inquiry("O1", "LEO"), trialBy("O2", "HUG", "LEO"), trialBy("O3", "OTO", "LEO")},
	})
	if hasNoble(resolution.State, "N2") {
		t.Fatalf("a witch unmasked this winter must be tried this winter")
	}
}

func TestTrialJudgedInIncreasingTargetCode(t *testing.T) {
	state := winterTrialState(t)
	state.Excommunicate(models.Excommunication{Noble: "N3", Reason: models.ExcommunicationExOfficio, Turn: 1})
	addNoble(state, "N5", "ZED", "P2", "CCC")
	addNoble(state, "N6", "YAN", "P2", "CCC")
	state.Regions = []models.Region{
		{ID: "R1", Name: "Ros", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}},
		{ID: "R2", Name: "Woo", Seed: "DDD", Territories: []models.TerritoryID{"DDD"}},
		{ID: "R3", Name: "Cee", Seed: "CCC", Territories: []models.TerritoryID{"CCC"}},
		{ID: "R4", Name: "Bee", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}},
	}
	state.Bishops = append(state.Bishops, models.Bishop{Region: "R3", Noble: "N5"}, models.Bishop{Region: "R4", Noble: "N6"})
	state.Cardinals = append(state.Cardinals, "N5", "N6")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO"), trialBy("O2", "OTO", "LEO")},
		"P2": {trialBy("O3", "ZED", "ABE"), trialBy("O4", "YAN", "ABE")},
	})
	trials := eventsOfType(resolution.Events, EventTypeTrial)
	if len(trials) != 2 || trials[0].NobleCode != "ABE" || trials[1].NobleCode != "LEO" {
		t.Fatalf("trials = %#v, want ABE then LEO", trials)
	}
	if hasNoble(resolution.State, "N2") || hasNoble(resolution.State, "N3") {
		t.Fatalf("both targets must be executed")
	}
}

func TestCardTrialOnExcommunicatedNoble(t *testing.T) {
	state := winterTrialState(t)
	if _, liable := (&resolutionContext{state: state}).trialVerdict(&state.Nobles[1]); !liable {
		t.Fatalf("an excommunicated noble is liable to the trial")
	}
	if _, liable := (&resolutionContext{state: state}).trialVerdict(&state.Nobles[2]); liable {
		t.Fatalf("a free man is not liable")
	}
}

func TestAcceptedTrialOrderIsAcknowledgedEvenWithoutSecondCardinal(t *testing.T) {
	state := winterTrialState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {trialBy("O1", "HUG", "LEO")},
	})
	filed := eventsOfType(resolution.Events, EventTypeTrialFiled)
	if len(filed) != 1 || filed[0].OrderID != "O1" || filed[0].NobleCode != "LEO" {
		t.Fatalf("trial_filed events = %#v", filed)
	}
}
