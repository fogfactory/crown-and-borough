package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestReturnOrphanFiefsToCrown(t *testing.T) {
	king := models.NobleID("N1")
	build := func() *models.GameState {
		return &models.GameState{
			Players: []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}},
			Nobles: []models.Noble{
				{ID: "N1", OwnerID: "P1", Sex: models.SexMale},
				{ID: "N2", OwnerID: "P2", Sex: models.SexMale},
			},
			King: &king,
			Fiefs: []models.Fief{
				{ID: "F1", OwnerID: "P3"},
				{ID: "F2", OwnerID: "P2"},
				{ID: "F3", OwnerID: "P3", Crown: true},
				{ID: "F4", OwnerID: "P3", HolderNobleID: ptrNobleID("N9")},
			},
		}
	}

	state := build()
	returned := state.ReturnOrphanFiefsToCrown()
	if len(returned) != 2 || returned[0].Fief.ID != "F1" || returned[1].Fief.ID != "F3" || returned[0].PreviousOwner != "P3" {
		t.Fatalf("returned = %#v, want F1 and F3 taken from P3", returned)
	}
	for _, index := range []int{0, 2} {
		if fief := state.Fiefs[index]; fief.OwnerID != "P1" || !fief.Crown {
			t.Errorf("fief %s = %#v, want a crown fief of P1", fief.ID, fief)
		}
	}
	if fief := state.Fiefs[1]; fief.OwnerID != "P2" || fief.Crown {
		t.Errorf("fief F2 = %#v, want it left with P2, who has a noble", fief)
	}
	if fief := state.Fiefs[3]; fief.OwnerID != "P3" || fief.Crown {
		t.Errorf("fief F4 = %#v, want the held fief left alone", fief)
	}
	if again := state.ReturnOrphanFiefsToCrown(); len(again) != 0 {
		t.Errorf("second call returned %#v, want nothing", again)
	}

	state = build()
	state.King = nil
	if player, ok := state.CrownPlayer(); ok || player != "" || len(state.ReturnOrphanFiefsToCrown()) != 0 {
		t.Errorf("vacant throne: CrownPlayer = %q, %v; want none and no return", player, ok)
	}

	state = build()
	state.Nobles[0].Status = models.NobleStatusDungeon
	if player, ok := state.CrownPlayer(); !ok || player != "P1" {
		t.Errorf("king in a dungeon with no queen: CrownPlayer = %q, %v; want P1", player, ok)
	}
}
