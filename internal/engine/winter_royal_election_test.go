package engine

import (
	"fmt"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// holdFief makes the noble the titular of a new barony of its owner.
func holdFief(state *models.GameState, id models.NobleID) {
	var owner models.PlayerID
	for _, noble := range state.Nobles {
		if noble.ID == id {
			owner = noble.OwnerID
		}
	}
	n := len(state.Territories)
	middle := models.TerritoryID(fmt.Sprintf("Y%c%c", 'A'+n/26, 'A'+n%26))
	state.Territories = append(state.Territories, territory(string(middle), string(middle)))
	state.TerritoryStates[middle] = models.TerritoryState{}
	holdAsFiefMember(state, middle, owner)
	holder := id
	state.Fiefs[len(state.Fiefs)-1].HolderNobleID = &holder
}

// royalTestState extends the election test state: HUG (P1) and LEO (P2) are
// barons and bishops, ABE (P3) is a baron without clergy. Three titled nobles,
// two bishops: the royal election is open.
func royalTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := electionTestState(t)
	for _, id := range []models.NobleID{"N1", "N2", "N3"} {
		holdFief(state, id)
	}
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N1"}, {Region: "R2", Noble: "N2"}}
	validateTestState(t, state)
	return state
}

func royalCandidacy(id models.OrderID, noble models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeCandidacy, Election: models.ElectionKing, NobleCode: noble}
}

func royalBallot(id models.OrderID, noble models.NobleCode) models.WinterOrder {
	order := royalCandidacy(id, noble)
	order.Type = models.WinterOrderTypeVote
	return order
}

func royalResult(t *testing.T, events []Event) *ElectionResult {
	t.Helper()
	for _, event := range eventsOfType(events, EventTypeElectionResult) {
		if event.Election.Kind == models.ElectionKing {
			return event.Election
		}
	}
	t.Fatalf("no royal election result in %#v", events)
	return nil
}

func TestRoyalElectionCrownsAnointedCandidate(t *testing.T) {
	state := royalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P3": {royalCandidacy("O1", "ABE"), royalBallot("O2", "ABE")},
		"P1": {royalBallot("O1", "ABE")},
		"P2": {royalBallot("O1", "ABE")},
	})
	if resolution.State.King == nil || *resolution.State.King != "N3" {
		t.Fatalf("king = %v, want N3", resolution.State.King)
	}
	result := royalResult(t, resolution.Events)
	if result.Result != ElectionElected || result.Winner != "N3" || result.Cast != 3 || result.MinVotes != 3 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Votes != 3 || !result.Candidates[0].Anointed {
		t.Fatalf("candidates = %#v, want ABE with 3 anointed voices", result.Candidates)
	}
	validateTestState(t, resolution.State)
}

func TestRoyalElectionNeedsAnointment(t *testing.T) {
	state := royalTestState(t)
	// P3 gets two more titled nobles, none of them clergy: ABE weighs 3 voices
	// but no bishop backs him.
	addNoble(state, "N5", "EDO", "P3", "DDD")
	addNoble(state, "N6", "GUY", "P3", "DDD")
	holdFief(state, "N5")
	holdFief(state, "N6")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P3": {royalCandidacy("O1", "ABE"), royalBallot("O2", "ABE")},
	})
	if resolution.State.King != nil {
		t.Fatalf("king = %v, want a vacant throne without the clergy's blessing", *resolution.State.King)
	}
	result := royalResult(t, resolution.Events)
	if result.Result != ElectionNoAnointment || result.Candidates[0].Votes != 3 || result.Candidates[0].Anointed {
		t.Fatalf("result = %#v, want 3 voices without anointment", result)
	}
}

func TestRoyalElectionVoicesAndThresholds(t *testing.T) {
	t.Run("below three voices", func(t *testing.T) {
		resolution := resolveElectionSheets(t, royalTestState(t), map[models.PlayerID][]models.WinterOrder{
			"P1": {royalCandidacy("O1", "HUG"), royalBallot("O2", "HUG")},
			"P2": {royalBallot("O1", "HUG")},
		})
		if resolution.State.King != nil {
			t.Fatal("two voices must not elect a king")
		}
		if got := royalResult(t, resolution.Events).Result; got != ElectionNoMajority {
			t.Fatalf("result = %q, want no_majority", got)
		}
	})
	t.Run("tie", func(t *testing.T) {
		resolution := resolveElectionSheets(t, royalTestState(t), map[models.PlayerID][]models.WinterOrder{
			"P1": {royalCandidacy("O1", "HUG"), royalBallot("O2", "HUG")},
			"P2": {royalCandidacy("O1", "LEO"), royalBallot("O2", "LEO")},
			"P3": {royalCandidacy("O1", "ABE"), royalBallot("O2", "ABE")},
		})
		if resolution.State.King != nil || royalResult(t, resolution.Events).Result != ElectionTie {
			t.Fatalf("king = %v, result = %#v", resolution.State.King, royalResult(t, resolution.Events))
		}
	})
	t.Run("dungeon noble gives no voice", func(t *testing.T) {
		state := royalTestState(t)
		addNoble(state, "N5", "EDO", "P1", "AAA") // keeps three voting titles open
		holdFief(state, "N5")
		setNobleStatus(state, "N3", models.NobleStatusDungeon)
		resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
			"P1": {royalCandidacy("O1", "HUG"), royalBallot("O2", "HUG")},
			"P3": {royalBallot("O1", "HUG")},
		})
		if reason := electionRejections(resolution.Events)["O1"]; reason != electionNoVoiceError {
			t.Fatalf("dungeon vote rejected with %q, want no_voice", reason)
		}
	})
}

func TestRoyalElectionOpening(t *testing.T) {
	sheets := map[models.PlayerID][]models.WinterOrder{"P1": {royalCandidacy("O1", "HUG")}}
	closed := func(name string, state *models.GameState) {
		t.Run(name, func(t *testing.T) {
			resolution := resolveElectionSheets(t, state, sheets)
			if reason := electionRejections(resolution.Events)["O1"]; reason != "election_not_open" {
				t.Fatalf("rejections = %#v, want election_not_open", electionRejections(resolution.Events))
			}
		})
	}
	state := royalTestState(t)
	state.Bishops = state.Bishops[:1]
	closed("one bishop is not enough to anoint", state)

	state = royalTestState(t)
	state.Fiefs = state.Fiefs[:2]
	closed("fewer than three voting titles", state)

	state = royalTestState(t)
	king := models.NobleID("N4")
	state.King = &king
	closed("throne already held", state)

	t.Run("a cardinal alone anoints", func(t *testing.T) {
		state := royalTestState(t)
		state.Bishops = state.Bishops[:1]
		state.Cardinals = []models.NobleID{"N1"}
		resolution := resolveElectionSheets(t, state, sheets)
		if reason := electionRejections(resolution.Events)["O1"]; reason != "" {
			t.Fatalf("candidacy rejected with %q", reason)
		}
	})
}

func TestRoyalCandidateEligibility(t *testing.T) {
	state := royalTestState(t)
	// OTO has no fief; a married baron stays eligible; a bastard is not.
	state.Marriages = append(state.Marriages, models.Marriage{NobleA: "N1", NobleB: "N9"})
	state.Nobles = append(state.Nobles, models.Noble{ID: "N9", Code: "ANN", Name: "ANN", Sex: models.SexFemale, OwnerID: "P2", LocationID: "CCC", Status: models.NobleStatusFree})
	for index := range state.Nobles {
		if state.Nobles[index].ID == "N3" {
			state.Nobles[index].Dignities = []models.Dignity{models.DignityBastard}
		}
	}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {royalCandidacy("O1", "HUG"), royalCandidacy("O2", "OTO")},
		"P2": {royalCandidacy("O1", "ANN")},
		"P3": {royalCandidacy("O1", "ABE")},
	})
	if reason := electionRejections(resolution.Events)["O2"]; reason != "candidate_not_eligible" {
		t.Fatalf("OTO (no fief) rejected with %q", reason)
	}
	result := royalResult(t, resolution.Events)
	if len(result.Candidates) != 1 || result.Candidates[0].NobleCode != "HUG" {
		t.Fatalf("candidates = %#v, want only the married baron HUG", result.Candidates)
	}
}

func TestRoyalCrownGivesNoVoiceBeforeNextWinter(t *testing.T) {
	state := royalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P3": {royalCandidacy("O1", "ABE"), royalBallot("O2", "ABE")},
		"P1": {royalBallot("O1", "ABE")},
		"P2": {royalBallot("O1", "ABE")},
	})
	if len(ForecastWinterElections(resolution.State, testBalance(), "P1")) != 0 {
		t.Fatal("no election must be open once the throne is held")
	}
}

func TestRoyalHostageVoiceGoesToItsOwner(t *testing.T) {
	state := royalTestState(t)
	// ABE (P3) is held hostage by P1: his voice stays with P3, never with P1.
	setNobleStatus(state, "N3", models.NobleStatusHostage)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {royalCandidacy("O1", "HUG"), royalBallot("O2", "HUG")},
		"P2": {royalBallot("O1", "HUG")},
		"P3": {royalBallot("O1", "HUG")},
	})
	result := royalResult(t, resolution.Events)
	if result.Result != ElectionElected || result.Candidates[0].Votes != 3 {
		t.Fatalf("result = %#v, want HUG elected with the three voices", result)
	}
}

func TestRoyalElectionAnointedByOneCardinal(t *testing.T) {
	state := royalTestState(t)
	state.Bishops = state.Bishops[:1]
	state.Cardinals = []models.NobleID{"N1"}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P3": {royalCandidacy("O1", "ABE"), royalBallot("O2", "ABE")},
		"P1": {royalBallot("O1", "ABE")},
		"P2": {royalBallot("O1", "ABE")},
	})
	if resolution.State.King == nil || *resolution.State.King != "N3" {
		t.Fatalf("king = %v, want N3 anointed by P1's cardinal alone", resolution.State.King)
	}
}

func TestBastardCardPlayedOnCandidateRemovesHimFromTheRace(t *testing.T) {
	state := royalTestState(t)
	giveDignityCard(state, "P3")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P3": {
			royalCandidacy("O1", "ABE"),
			{ID: "O2", Type: models.WinterOrderTypeDignity, NobleCode: "ABE", CardCode: models.DignityBastardCardCode},
		},
		"P1": {royalBallot("O1", "ABE")},
		"P2": {royalBallot("O1", "ABE")},
	})
	if resolution.State.King != nil {
		t.Fatalf("king = %v, a bastard cannot be crowned", *resolution.State.King)
	}
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "candidate_not_eligible" && reasons["O1"] != "unknown_candidate" {
		t.Fatalf("rejections = %#v, want the candidacy refused", reasons)
	}
	if got := royalResult(t, resolution.Events).Result; got != ElectionNoCandidate {
		t.Fatalf("result = %q, want no_candidate", got)
	}
}
