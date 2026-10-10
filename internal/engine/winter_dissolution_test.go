package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func dissolve(id models.OrderID, target models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeDissolveMarriage, NobleCode: target}
}

// dissolutionTestState makes P1's HUG (N1) the pope and marries LEO (N2, P2)
// to OTO (N4, P1)... OTO belongs to P1, so the pope's owner owns one spouse.
func dissolutionTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := inquiryTestState(t)
	state.Pope = new(models.NobleID)
	*state.Pope = "N1"
	for i := range state.Nobles {
		switch state.Nobles[i].ID {
		case "N2":
			state.Nobles[i].Sex = models.SexMale
		case "N4":
			state.Nobles[i].Sex = models.SexFemale
		}
	}
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N4", Turn: 0}}
	return state
}

func TestPopeDissolvesOwnSpouseMarriage(t *testing.T) {
	state := dissolutionTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {dissolve("O1", "OTO")}})
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
	if _, married := resolution.State.MarriageOf("N4"); married {
		t.Fatalf("marriage still active")
	}
	if len(resolution.State.Marriages) != 1 || !resolution.State.Marriages[0].Dissolved {
		t.Fatalf("marriages = %#v, want one dissolved record", resolution.State.Marriages)
	}
	if len(eventsOfType(resolution.Events, EventTypeMarriageDissolved)) != 1 {
		t.Fatalf("missing dissolution event")
	}
}

func TestDissolutionNeedsPopeAndRequest(t *testing.T) {
	state := dissolutionTestState(t)
	// A spouse owner's request alone is not granted.
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P2": {dissolve("O1", "LEO")}})
	if _, married := resolution.State.MarriageOf("N2"); !married {
		t.Fatalf("request without the pope dissolved the marriage")
	}
	// A marriage of two other houses needs the pope and a spouse's request.
	state = dissolutionTestState(t)
	for i := range state.Nobles {
		if state.Nobles[i].ID == "N3" {
			state.Nobles[i].Sex = models.SexFemale
		}
	}
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N3", Turn: 0}}
	resolution = resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {dissolve("O1", "LEO")}})
	if _, married := resolution.State.MarriageOf("N2"); !married {
		t.Fatalf("papal order without request dissolved the marriage")
	}
	resolution = resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {dissolve("O1", "LEO")}, "P3": {dissolve("O2", "ABE")},
	})
	if _, married := resolution.State.MarriageOf("N2"); married {
		t.Fatalf("pope plus request must dissolve the marriage")
	}
}

func TestDissolutionFreesSpousesForRemarriageClaimsKept(t *testing.T) {
	state := dissolutionTestState(t)
	if marriage, ok := state.MarriageCovering("N2", "P1", 0); !ok {
		t.Fatalf("setup: marriage must cover turn 0 (%#v)", marriage)
	}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {dissolve("O1", "OTO")}})
	if _, ok := resolution.State.MarriageCovering("N2", "P1", resolution.State.Turn-1); !ok && resolution.State.Turn > 0 {
		t.Fatalf("dissolved marriage must still cover earlier turns")
	}
}
