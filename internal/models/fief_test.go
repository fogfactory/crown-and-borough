package models_test

import (
	"encoding/json"
	"strings"
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

func TestFreeNoblesAtKeepsLineOrderAndSkipsCaptives(t *testing.T) {
	state := &models.GameState{
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}},
		Nobles: []models.Noble{
			{ID: "N3", OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N1", OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusHostage},
			{ID: "N2", OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusFree},
			{ID: "N4", OwnerID: "P1", LocationID: "BBB", Status: models.NobleStatusFree},
			{ID: "N5", OwnerID: "P2", LocationID: "AAA", Status: models.NobleStatusFree},
		},
	}
	free := state.FreeNoblesAt("P1", "AAA")
	if len(free) != 2 || free[0].ID != "N2" || free[1].ID != "N3" {
		t.Fatalf("FreeNoblesAt = %#v, want N2 then N3", free)
	}
}

func TestSettleFiefTitlesOfMissingNobles(t *testing.T) {
	// P1 is the exile's house, P2 the usurper's.
	build := func(usurperAlive, exileAlive bool) *models.GameState {
		nobles := []models.Noble{{ID: "N9", OwnerID: "P3"}}
		if exileAlive {
			nobles = append(nobles, models.Noble{ID: "N1", OwnerID: "P1"})
		}
		if usurperAlive {
			nobles = append(nobles, models.Noble{ID: "N2", OwnerID: "P2"})
		}
		return &models.GameState{
			Players: []models.Player{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}},
			Nobles:  nobles,
			Fiefs: []models.Fief{{
				ID: "F1", OwnerID: "P2", HolderNobleID: ptrNobleID("N2"),
				ExiledNobleID: ptrNobleID("N1"),
			}},
		}
	}
	noHeir := func(*models.Fief, models.NobleID) *models.Noble { return nil }

	state := build(false, true)
	got := state.SettleFiefTitlesOfMissingNobles(noHeir)
	fief := state.Fiefs[0]
	if len(got) != 1 || got[0].Outcome != models.FiefTitleRestored || fief.OwnerID != "P1" ||
		fief.HolderNobleID == nil || *fief.HolderNobleID != "N1" || fief.ExiledNobleID != nil {
		t.Errorf("usurper dead: %#v %#v, want the exile restored to P1", got, fief)
	}

	state = build(true, false)
	got = state.SettleFiefTitlesOfMissingNobles(noHeir)
	fief = state.Fiefs[0]
	if len(got) != 1 || got[0].Outcome != models.FiefTitleLegitimised || fief.OwnerID != "P2" ||
		*fief.HolderNobleID != "N2" || fief.ExiledNobleID != nil {
		t.Errorf("exile dead: %#v %#v, want the usurper official", got, fief)
	}

	state = build(true, false)
	heir := models.Noble{ID: "N9", OwnerID: "P3"}
	got = state.SettleFiefTitlesOfMissingNobles(func(_ *models.Fief, missing models.NobleID) *models.Noble {
		if missing != "N1" {
			t.Errorf("heirOf asked for %q, want the exile N1", missing)
		}
		return &heir
	})
	fief = state.Fiefs[0]
	if len(got) != 1 || got[0].Outcome != models.FiefTitleReexiled || *fief.ExiledNobleID != "N9" ||
		!fief.Usurped() || *fief.HolderNobleID != "N2" {
		t.Errorf("exile dead with claim: %#v %#v, want N9 the new exile", got, fief)
	}

	state = build(false, false)
	got = state.SettleFiefTitlesOfMissingNobles(func(*models.Fief, models.NobleID) *models.Noble {
		t.Error("heirOf must not be asked when both usurper and exile are dead")
		return &heir
	})
	fief = state.Fiefs[0]
	if len(got) != 1 || got[0].Outcome != models.FiefTitleVacated || fief.OwnerID != "P2" ||
		fief.HolderNobleID != nil || fief.ExiledNobleID != nil {
		t.Errorf("both dead: %#v %#v, want a vacant fief left with P2", got, fief)
	}
}

func TestFiefLoadsWithoutUsurpationFields(t *testing.T) {
	var fief models.Fief
	raw := `{"id":"F1","title":"barony","capitalTerritory":"AAA","territories":["AAA"],"owner":"P1","holderNoble":"N1"}`
	if err := json.Unmarshal([]byte(raw), &fief); err != nil {
		t.Fatal(err)
	}
	if fief.ExiledNobleID != nil {
		t.Fatalf("fief = %#v, want no exile nor usurper", fief)
	}
	out, err := json.Marshal(fief)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "exiledNoble") || strings.Contains(string(out), "usurped") {
		t.Errorf("marshalled fief %s must omit the usurpation fields", out)
	}
}
