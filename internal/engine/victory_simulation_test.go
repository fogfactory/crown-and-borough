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
