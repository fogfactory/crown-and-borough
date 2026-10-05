package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSuccessionLineFollowsPurchaseOrder(t *testing.T) {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}, {ID: "P2"}}
	g.Nobles = []models.Noble{
		{ID: "N10", OwnerID: "P1"},
		{ID: "N2", OwnerID: "P1"},
		{ID: "N3", OwnerID: "P2"},
		{ID: "N7", OwnerID: "P1", Status: models.NobleStatusDungeon},
	}
	g.RemovedNobles = []models.RemovedNoble{{ID: "N1", OwnerID: "P1"}}

	var got []models.NobleID
	for _, n := range g.SuccessionLine("P1") {
		got = append(got, n.ID)
	}
	want := []models.NobleID{"N2", "N7", "N10"}
	if len(got) != len(want) {
		t.Fatalf("line = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line = %v, want %v", got, want)
		}
	}
	lines := g.SuccessionLines()
	if len(lines["P2"]) != 1 || len(lines["P1"]) != 3 {
		t.Errorf("SuccessionLines = %v", lines)
	}
}
