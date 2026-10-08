package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// electionTestState has two bishoprics. ROS (seed AAA, then BBB and CCC) is
// held by P1 on AAA and BBB and by P2 on CCC; WOO (seed DDD) is held by P3.
// HUG and OTO belong to P1, LEO to P2, ABE to P3, all free men.
func electionTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterTestState(t, []models.Territory{
		territory("AAA", "AAA", "BBB"),
		territory("BBB", "BBB", "AAA", "CCC"),
		territory("CCC", "CCC", "BBB", "DDD"),
		territory("DDD", "DDD", "CCC"),
	}, nil)
	state.Regions = []models.Region{
		{ID: "R1", Name: "Ros", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}},
		{ID: "R2", Name: "Woo", Seed: "DDD", Territories: []models.TerritoryID{"DDD"}},
	}
	addAnchorArmy(t, state, "A1", "P1", "AAA")
	addAnchorArmy(t, state, "A2", "P1", "BBB")
	addAnchorArmy(t, state, "A3", "P2", "CCC")
	addAnchorArmy(t, state, "A4", "P3", "DDD")
	addNoble(state, "N1", "HUG", "P1", "AAA")
	addNoble(state, "N2", "LEO", "P2", "CCC")
	addNoble(state, "N3", "ABE", "P3", "DDD")
	addNoble(state, "N4", "OTO", "P1", "AAA")
	return state
}

func candidacy(id models.OrderID, noble models.NobleCode, seat models.TerritoryID) models.WinterOrder {
	kind := models.ElectionBishop
	if seat == "" {
		kind = models.ElectionPope
	}
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeCandidacy, Election: kind, NobleCode: noble, TerritoryID: seat}
}

func ballot(id models.OrderID, noble models.NobleCode, seat models.TerritoryID) models.WinterOrder {
	order := candidacy(id, noble, seat)
	order.Type = models.WinterOrderTypeVote
	return order
}

func resolveElectionSheets(t *testing.T, state *models.GameState, sheets map[models.PlayerID][]models.WinterOrder) Resolution {
	t.Helper()
	resolution, err := ResolveWinter(state, testBalance(), sheets)
	if err != nil {
		t.Fatalf("ResolveWinter() = %v", err)
	}
	return resolution
}

func electionRejections(events []Event) map[models.OrderID]string {
	reasons := map[models.OrderID]string{}
	for _, event := range eventsOfType(events, EventTypeRejected) {
		reasons[event.OrderID] = event.Reason
	}
	return reasons
}

func TestBishopElectionRelativeMajorityAndInvestiture(t *testing.T) {
	state := electionTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", "AAA"), ballot("O2", "HUG", "AAA")},
		"P2": {candidacy("O1", "LEO", "AAA"), ballot("O2", "LEO", "AAA")},
	})
	// P1 holds the seat (2) and BBB (1); P2 holds CCC (1).
	if bishop, ok := resolution.State.BishopOf("R1"); !ok || bishop != "N1" {
		t.Fatalf("bishop of R1 = %q, %v, want N1", bishop, ok)
	}
	results := eventsOfType(resolution.Events, EventTypeElectionResult)
	if len(results) != 2 {
		t.Fatalf("election results = %d, want both open bishoprics", len(results))
	}
	ros := results[0].Election
	if ros.Region != "R1" || ros.Result != ElectionElected || ros.Winner != "N1" || ros.Cast != 4 {
		t.Fatalf("R1 result = %#v", ros)
	}
	if len(ros.Candidates) != 2 || ros.Candidates[0].Votes != 3 || ros.Candidates[1].Votes != 1 {
		t.Fatalf("R1 candidates = %#v, want totals 3 and 1", ros.Candidates)
	}
	if results[1].Election.Region != "R2" || results[1].Election.Result != ElectionNoCandidate {
		t.Fatalf("R2 result = %#v, want no candidate", results[1].Election)
	}
	if len(eventsOfType(resolution.Events, EventTypeElectionOpened)) != 2 || len(eventsOfType(resolution.Events, EventTypeInvestiture)) != 1 {
		t.Fatalf("events = %#v", resolution.Events)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Elections) != 2 {
		t.Fatalf("report elections = %d, want 2", len(report.Elections))
	}
}

func TestVoteWithoutVoiceIsRejected(t *testing.T) {
	state := electionTestState(t)
	state.Armies[2].OwnerID = "P1" // CCC to P1: P2 has no voice left
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", "AAA"), ballot("O2", "HUG", "AAA")},
		"P2": {ballot("O1", "HUG", "AAA")},
	})
	if bishop, _ := resolution.State.BishopOf("R1"); bishop != "N1" {
		t.Fatalf("bishop of R1 = %q, want N1", bishop)
	}
	if reasons := electionRejections(resolution.Events); reasons["O1"] != "no_voice" {
		t.Fatalf("rejections = %#v, want P2 vote with no voice", reasons)
	}
}

func TestElectionTieHasNoWinner(t *testing.T) {
	state := electionTestState(t)
	state.Bishops = []models.Bishop{{Region: "R2", Noble: "N3"}}
	addNoble(state, "N8", "BIS", "P3", "DDD")
	// P2 weighs 1 voice (CCC) and P3 1 voice (its bishop), each backing its own
	// candidate.
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P2": {candidacy("O1", "LEO", "AAA"), ballot("O2", "LEO", "AAA")},
		"P3": {candidacy("O1", "BIS", "AAA"), ballot("O2", "BIS", "AAA")},
	})
	results := eventsOfType(resolution.Events, EventTypeElectionResult)
	if len(results) != 1 || results[0].Election.Result != ElectionTie {
		t.Fatalf("results = %#v, want one tie", results)
	}
	if _, ok := resolution.State.BishopOf("R1"); ok {
		t.Fatal("tie elected a bishop")
	}
}

func TestElectionNotOpenWhenBishopricHasANeutralTerritory(t *testing.T) {
	state := electionTestState(t)
	state.Armies = append(state.Armies[:2], state.Armies[3:]...)
	delete(state.TerritoryStates, "CCC")
	state.TerritoryStates["CCC"] = models.TerritoryState{}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", "AAA"), ballot("O2", "HUG", "AAA")},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "election_not_open" || reasons["O2"] != "election_not_open" {
		t.Fatalf("rejections = %#v, want election_not_open", reasons)
	}
	if len(resolution.State.Bishops) != 0 {
		t.Fatalf("bishops = %#v", resolution.State.Bishops)
	}
}

func TestNobleRunsInOneElectionOnly(t *testing.T) {
	state := electionTestState(t)
	state.Armies[3].OwnerID = "P1"
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			candidacy("O1", "HUG", "AAA"),
			candidacy("O2", "HUG", "DDD"), // same noble elsewhere
			candidacy("O3", "OTO", "AAA"), // second candidacy of the player in R1
			candidacy("O4", "OTO", "DDD"),
		},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O2"] != "candidate_already_running" || reasons["O3"] != "candidacy_already_filed" || reasons["O4"] != "" {
		t.Fatalf("rejections = %#v", reasons)
	}
	results := eventsOfType(resolution.Events, EventTypeElectionResult)
	if len(results[0].Election.Candidates) != 1 || len(results[1].Election.Candidates) != 1 {
		t.Fatalf("results = %#v", results)
	}
}

func TestElectionVoteRules(t *testing.T) {
	state := electionTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			candidacy("O1", "HUG", "AAA"),
			ballot("O2", "ABE", "AAA"), // not a candidate
			ballot("O3", "HUG", "AAA"),
			ballot("O4", "HUG", "AAA"), // second vote ignored
			ballot("O5", "HUG", "BBB"), // BBB is not a seat
		},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O2"] != "unknown_candidate" || reasons["O4"] != "vote_already_cast" || reasons["O5"] != "election_not_open" || reasons["O3"] != "" {
		t.Fatalf("rejections = %#v", reasons)
	}
	if got := eventsOfType(resolution.Events, EventTypeElectionResult)[0].Election.Candidates[0].Votes; got != 3 {
		t.Fatalf("HUG votes = %d, want 3", got)
	}
}

func TestIneligibleCandidates(t *testing.T) {
	state := electionTestState(t)
	for index := range state.Nobles {
		switch state.Nobles[index].ID {
		case "N1":
			state.Nobles[index].Sex = models.SexFemale
		case "N4":
			state.Nobles[index].Status = models.NobleStatusHostage
		}
	}
	addNoble(state, "N5", "MAR", "P1", "AAA")
	addNoble(state, "N6", "SPO", "P2", "CCC")
	state.Marriages = []models.Marriage{{NobleA: "N5", NobleB: "N6", Turn: 1}}
	for index := range state.Nobles {
		if state.Nobles[index].ID == "N6" {
			state.Nobles[index].Sex = models.SexFemale
		}
	}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", "AAA"), candidacy("O2", "OTO", "AAA"), candidacy("O3", "MAR", "AAA")},
	})
	for _, id := range []models.OrderID{"O1", "O2", "O3"} {
		if reason := electionRejections(resolution.Events)[id]; reason != "candidate_not_eligible" {
			t.Errorf("order %s rejected with %q, want candidate_not_eligible", id, reason)
		}
	}
}

// conclaveTestState gives P1 a cardinal HUG (bishop of R1) and P2 a cardinal
// LEO (bishop of R2), plus a bishop OTO (P1) candidate; R1 and R2 have bishops
// so only the conclave is open.
func conclaveTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := electionTestState(t)
	addNoble(state, "N7", "BIS", "P3", "DDD")
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N1"}, {Region: "R2", Noble: "N2"}}
	state.Cardinals = []models.NobleID{"N1", "N2"}
	return state
}

func TestConclaveNeedsAbsoluteMajorityOfAllCardinals(t *testing.T) {
	state := conclaveTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", ""), ballot("O2", "HUG", "")},
		"P2": {ballot("O1", "HUG", "")},
	})
	if resolution.State.Pope == nil || *resolution.State.Pope != "N1" {
		t.Fatalf("pope = %v, want N1", resolution.State.Pope)
	}
	result := eventsOfType(resolution.Events, EventTypeElectionResult)[0].Election
	if result.Kind != models.ElectionPope || result.Required != 2 || result.Cast != 2 {
		t.Fatalf("conclave result = %#v", result)
	}

	state = conclaveTestState(t)
	resolution = resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", ""), ballot("O2", "HUG", "")},
	})
	if resolution.State.Pope != nil {
		t.Fatalf("pope = %v, one of two cardinals is not a majority", *resolution.State.Pope)
	}
	if got := eventsOfType(resolution.Events, EventTypeElectionResult)[0].Election.Result; got != ElectionNoMajority {
		t.Fatalf("result = %q, want no_majority", got)
	}
}

func TestConclaveClosedWithoutTwoCardinals(t *testing.T) {
	state := conclaveTestState(t)
	state.Cardinals = []models.NobleID{"N1"}
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", ""), ballot("O2", "HUG", "")},
	})
	if reasons := electionRejections(resolution.Events); reasons["O1"] != "election_not_open" {
		t.Fatalf("rejections = %#v", reasons)
	}
}

func TestDungeonCardinalKeepsMajorityDenominator(t *testing.T) {
	state := conclaveTestState(t)
	setNobleStatus(state, "N2", models.NobleStatusDungeon)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", ""), ballot("O2", "HUG", "")},
		"P2": {ballot("O1", "HUG", "")},
	})
	if resolution.State.Pope != nil {
		t.Fatal("a cardinal in a dungeon must still count in the majority denominator")
	}
	if reason := electionRejections(resolution.Events)["O1"]; reason != "no_voice" {
		t.Fatalf("dungeon cardinal vote rejected with %q, want no_voice", reason)
	}
}

func TestElectedTitleCountsOnlyFromInvestiture(t *testing.T) {
	state := electionTestState(t)
	state.Armies[3].OwnerID = "P1"
	// HUG wins R1 and OTO wins R2 the same winter; HUG's new bishop voice must
	// not weigh on R2.
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {candidacy("O1", "HUG", "AAA"), ballot("O2", "HUG", "AAA"), candidacy("O3", "OTO", "DDD"), ballot("O4", "OTO", "DDD")},
		"P2": {candidacy("O1", "LEO", "DDD"), ballot("O2", "LEO", "DDD")},
	})
	r2 := eventsOfType(resolution.Events, EventTypeElectionResult)[1].Election
	// The seat alone weighs 2: HUG's bishop title is only conferred at the
	// investiture, after every count.
	if r2.Candidates[0].NobleCode != "OTO" || r2.Candidates[0].Votes != 2 {
		t.Fatalf("R2 candidates = %#v, want OTO with 2 voices", r2.Candidates)
	}
}
