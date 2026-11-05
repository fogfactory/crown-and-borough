package models_test

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func royalState(status models.NobleStatus) *models.GameState {
	g := models.NewGameState()
	g.Players = []models.Player{{ID: "P1"}, {ID: "P2"}}
	g.Nobles = []models.Noble{{ID: "N1", OwnerID: "P1", Sex: models.SexMale, Status: status}}
	king := models.NobleID("N1")
	g.King = &king
	return g
}

func TestReigningKingPowersFollowCaptivity(t *testing.T) {
	for status, want := range map[models.NobleStatus]bool{
		models.NobleStatusFree:    true,
		models.NobleStatusHostage: true,
		models.NobleStatusDungeon: false,
	} {
		g := royalState(status)
		king, reigning := g.ReigningKing()
		if reigning != want || king.ID != "N1" {
			t.Errorf("%s: ReigningKing = %q, %v, want N1, %v", status, king.ID, reigning, want)
		}
		if !g.IsKing("N1") {
			t.Errorf("%s: the captured king must keep the crown", status)
		}
	}
	g := royalState(models.NobleStatusFree)
	g.King = nil
	if _, reigning := g.ReigningKing(); reigning {
		t.Error("a vacant throne has no reigning king")
	}
}

func TestExcommunicatedKingKeepsTheCrown(t *testing.T) {
	g := royalState(models.NobleStatusFree)
	g.Excommunicate(models.Excommunication{Noble: "N1", Reason: models.ExcommunicationExOfficio})
	if !g.IsKing("N1") {
		t.Fatal("excommunication must only strip the religious titles")
	}
}

func TestCrownHeirOfRanksClaimsOnTheKingAndTheQueenTogether(t *testing.T) {
	g := royalState(models.NobleStatusFree)
	g.Nobles = []models.Noble{
		{ID: "N3", OwnerID: "P2", Sex: models.SexMale},
		{ID: "N4", OwnerID: "P2", Sex: models.SexMale},
		{ID: "N5", OwnerID: "P2", Sex: models.SexFemale},
		{ID: "N6", OwnerID: "P2", Sex: models.SexMale, Dignities: []models.Dignity{models.DignityBastard}},
	}
	g.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N2"}}
	g.Turn = 10
	g.Claims = []models.Claim{
		{Heir: "N5", Target: "N1", Spouse: "N2", Turn: 2},
		{Heir: "N6", Target: "N2", Spouse: "N1", Turn: 3},
		{Heir: "N4", Target: "N2", Spouse: "N1", Turn: 6},
		{Heir: "N3", Target: "N1", Spouse: "N2", Turn: 8},
	}
	heir, found := g.CrownHeirOf("N1")
	if !found || heir != "N4" {
		t.Fatalf("heir = %q, %v, want N4 (woman and bastard skipped, claim on the queen counts)", heir, found)
	}
	g.Marriages[0].Dissolved = true
	if heir, _ := g.CrownHeirOf("N1"); heir != "N3" {
		t.Fatalf("heir = %q, want N3: a dissolved marriage leaves no queen", heir)
	}
}

func TestValidateRejectsAClaimFromOutsideTheSpousesHouse(t *testing.T) {
	g := religiousState()
	g.Players = append(g.Players, models.Player{ID: "P3"})
	g.Nobles[1].Sex = models.SexFemale
	g.Nobles[2].OwnerID = "P3"
	g.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N2"}}
	g.Claims = []models.Claim{{Heir: "N3", Target: "N1", Spouse: "N2"}}
	if err := g.Validate(); err == nil || !strings.Contains(err.Error(), "house of the spouse") {
		t.Fatalf("a claim by a third house must be rejected, got %v", err)
	}
	g.Nobles[2].OwnerID = "P2"
	if err := g.Validate(); err != nil {
		t.Fatalf("a claim by the spouse's house is valid: %v", err)
	}
}
