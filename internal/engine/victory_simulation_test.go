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
