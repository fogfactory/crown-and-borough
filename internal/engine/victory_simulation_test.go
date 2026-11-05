package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSimulateVictoryClaimThenKillHandsTheFiefToTheHeir(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	holder := models.NobleID("N3")
	state.Fiefs = []models.Fief{{ID: "F1", OwnerID: "P2", HolderNobleID: &holder, Title: models.FiefTitleDuchy}}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}
	balance := allianceBalance()

	simulation, err := SimulateVictory(state, balance, []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.Fiefs[0].OwnerID; got != "P1" {
		t.Fatalf("fief owner after the claimed noble died = %q, want P1", got)
	}
	if got := simulation.Projected["P1"].Score.Titles; got != simulation.Current["P1"].Score.Titles+1 {
		t.Fatalf("P1 titles = %d, want one more than %d", got, simulation.Current["P1"].Score.Titles)
	}
	if len(state.Fiefs) != 1 || state.Fiefs[0].OwnerID != "P2" {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryKillOfTheKingCrownsTheHeirOfAClaim(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3", Turn: 1}}
	king := models.NobleID("N3")
	state.King = &king
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Sex = models.SexMale
	}

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.King; got == nil || *got != "N2" {
		t.Fatalf("king after the claimed king died = %v, want N2", got)
	}
	if state.King == nil || *state.King != "N3" {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryKillOfTheLastNobleReturnsTheFiefToTheCrown(t *testing.T) {
	state := allianceState()
	king := models.NobleID("N1")
	state.King = &king
	holder := models.NobleID("N3")
	state.Nobles = state.Nobles[:3]
	state.Fiefs = append(state.Fiefs, models.Fief{ID: "F2", OwnerID: "P2", HolderNobleID: &holder, Title: models.FiefTitleBarony})
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}

	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationKill, Noble: "CN3"}})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.Fiefs[1]; got.OwnerID != "P1" || !got.Crown || got.HolderNobleID != nil {
		t.Fatalf("fief after the last noble of P2 died = %#v, want a vacant crown fief of P1", got)
	}
	if state.Fiefs[1].OwnerID != "P2" || state.Fiefs[1].Crown {
		t.Fatal("simulation mutated the source state")
	}
}

func rankSimulationState() *models.GameState {
	state := allianceState()
	state.Fiefs = nil
	for index, id := range []models.NobleID{"N1", "N2", "N3", "N4"} {
		holder := id
		owner := models.PlayerID("P1")
		if index >= 2 {
			owner = "P2"
		}
		state.Fiefs = append(state.Fiefs, models.Fief{
			ID: models.FiefID("F" + string(id[1:])), OwnerID: owner, HolderNobleID: &holder,
			CapitalTerritoryID: models.TerritoryID("T" + string(id[1:])), Title: models.FiefTitleBarony,
		})
	}
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
	}
	// B (N2-N3) is older than A (N1-N4); both weigh the same.
	state.Marriages = []models.Marriage{
		{NobleA: "N1", NobleB: "N4", Turn: 2},
		{NobleA: "N2", NobleB: "N3", Turn: 1},
	}
	return state
}

func TestSimulateVictoryRankUpChangesTheActiveHead(t *testing.T) {
	state := rankSimulationState()
	balance := allianceBalance()
	head, _ := ActiveHeadMarriage(state, balance, "P1")
	if head.NobleA != "N2" {
		t.Fatalf("head before = %v, want the older marriage N2-N3", head)
	}

	simulation, err := SimulateVictory(state, balance, []SimulationAction{
		// Nobles placed above must hold an equal or higher title first.
		{Kind: SimulationRank, Fief: "T1"},
		{Kind: SimulationRank, Fief: "T3"},
		{Kind: SimulationRank, Fief: "T4"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	head, _ = ActiveHeadMarriage(simulation.State, balance, "P1")
	if head.NobleA != "N1" {
		t.Fatalf("head after the rank up = %v, want N1-N4", head)
	}
	if got := simulation.State.Fiefs[3].Title; got != models.FiefTitleCounty {
		t.Fatalf("fief title = %q, want county", got)
	}
	if state.Fiefs[3].Title != models.FiefTitleBarony {
		t.Fatal("simulation mutated the source state")
	}
}

func TestSimulateVictoryRankErrors(t *testing.T) {
	state := rankSimulationState()
	state.Fiefs[0].Title = models.FiefTitleDuchy
	for name, fief := range map[string]models.TerritoryID{"top rank": "T1", "unknown": "T9"} {
		if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationRank, Fief: fief}}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSimulateVictoryCrownAndDethrone(t *testing.T) {
	state := rankSimulationState()
	for index := range state.Nobles {
		state.Nobles[index].Sex = models.SexMale
	}
	simulation, err := SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationCrown, Noble: "CN1"},
		{Kind: SimulationCrown, Noble: "CN2"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if got := simulation.State.King; got == nil || *got != "N2" {
		t.Fatalf("king = %v, want N2", got)
	}
	if state.King != nil {
		t.Fatal("simulation mutated the source state")
	}
	simulation, err = SimulateVictory(state, allianceBalance(), []SimulationAction{
		{Kind: SimulationCrown, Noble: "CN1"},
		{Kind: SimulationDethrone},
	})
	if err != nil || simulation.State.King != nil {
		t.Fatalf("dethrone: king = %v, err = %v", simulation.State.King, err)
	}
	if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationDethrone}}); err == nil {
		t.Fatal("dethroning a vacant throne should fail")
	}
}

func TestSimulateVictoryRankBlockedBySuccessionLine(t *testing.T) {
	// N1 is above N2 in the line and holds only a barony.
	_, err := SimulateVictory(rankSimulationState(), allianceBalance(), []SimulationAction{{Kind: SimulationRank, Fief: "T2"}})
	if err == nil {
		t.Fatal("expected the rank up of the second in line to be refused")
	}
}

func TestSimulateVictoryCrownRefusesIneligibleNobles(t *testing.T) {
	build := func() *models.GameState {
		state := rankSimulationState()
		for index := range state.Nobles {
			state.Nobles[index].Sex = models.SexMale
		}
		return state
	}
	king := models.NobleID("N1")
	pope := models.NobleID("N2")
	cases := map[string]struct {
		mutate func(*models.GameState)
		noble  models.NobleCode
	}{
		"woman":         {func(s *models.GameState) { s.Nobles[3].Sex = models.SexFemale }, "CN4"},
		"already king":  {func(s *models.GameState) { s.King = &king }, "CN1"},
		"pope":          {func(s *models.GameState) { s.Pope = &pope }, "CN2"},
		"bastard":       {func(s *models.GameState) { s.Nobles[2].Dignities = []models.Dignity{models.DignityBastard} }, "CN3"},
		"excommunicate": {func(s *models.GameState) { s.Excommunications = []models.Excommunication{{Noble: "N4"}} }, "CN4"},
		"unknown noble": {func(*models.GameState) {}, "CN9"},
	}
	for name, tc := range cases {
		state := build()
		tc.mutate(state)
		if _, err := SimulateVictory(state, allianceBalance(), []SimulationAction{{Kind: SimulationCrown, Noble: tc.noble}}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSimulateVictoryCrownedHeirChangesTheHeadMarriage(t *testing.T) {
	state := allianceState()
	for index := range state.Nobles {
		state.Nobles[index].Code = "C" + string(state.Nobles[index].ID)
		state.Nobles[index].Sex = models.SexMale
	}
	king := models.NobleID("N3")
	state.King = &king
	state.Fiefs = append(state.Fiefs,
		models.Fief{ID: "F2", OwnerID: "P2", HolderNobleID: ptrNoble("N4"), Title: models.FiefTitleDuchy},
		models.Fief{ID: "F3", OwnerID: "P2", HolderNobleID: ptrNoble("N5"), Title: models.FiefTitleCounty})
	state.Marriages = []models.Marriage{
		{NobleA: "N1", NobleB: "N3", Turn: 1},
		{NobleA: "N2", NobleB: "N4", Turn: 2},
		{NobleA: "N1", NobleB: "N5", Turn: 3},
	}
	balance := allianceBalance()

	vacant, err := SimulateVictory(state, balance, []SimulationAction{{Kind: SimulationKill, Noble: "CN3"}})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if head, _ := ActiveHeadMarriage(vacant.State, balance, "P1"); head.NobleB != "N5" {
		t.Fatalf("head with a vacant throne = %+v, want N1-N5", head)
	}

	crowned, err := SimulateVictory(state, balance, []SimulationAction{
		{Kind: SimulationClaim, Noble: "CN2", Other: "CN3"},
		{Kind: SimulationKill, Noble: "CN3"},
	})
	if err != nil {
		t.Fatalf("SimulateVictory: %v", err)
	}
	if head, _ := ActiveHeadMarriage(crowned.State, balance, "P1"); head.NobleA != "N2" {
		t.Fatalf("head once N2 is crowned = %+v, want N2-N4", head)
	}
	if got, want := crowned.Projected["P1"].Score.Titles, vacant.Projected["P1"].Score.Titles+1; got != want {
		t.Fatalf("P1 titles = %d, want %d (the crown)", got, want)
	}
}
