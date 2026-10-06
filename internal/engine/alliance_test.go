package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func allianceBalance() assetgen.Balance {
	return assetgen.Balance{Alliance: assetgen.AllianceBalance{
		SuccessionRanks: []int{3, 2, 1},
		TitleRanks: map[models.FiefTitle]int{
			models.FiefTitleBarony: 1, models.FiefTitleCounty: 2,
			models.FiefTitleMarquisate: 3, models.FiefTitleDuchy: 4,
		},
		DensityBonus: 1,
	}}
}

func allianceState() *models.GameState {
	holder := models.NobleID("N1")
	return &models.GameState{
		Players: []models.Player{{ID: "P1"}, {ID: "P2"}},
		Nobles: []models.Noble{
			{ID: "N1", OwnerID: "P1", Sex: models.SexMale},
			{ID: "N2", OwnerID: "P1", Sex: models.SexMale},
			{ID: "N3", OwnerID: "P2", Sex: models.SexFemale},
			{ID: "N4", OwnerID: "P2", Sex: models.SexFemale},
			{ID: "N5", OwnerID: "P2", Sex: models.SexFemale},
		},
		Fiefs: []models.Fief{{ID: "F1", OwnerID: "P1", HolderNobleID: &holder, Title: models.FiefTitleDuchy}},
	}
}

func TestAllianceWeightTakesWeakerSpouse(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3"}}
	// N1: head (3) + duchy (4) = 7; N3: head (3) = 3.
	if got, ok := AllianceWeight(state, allianceBalance(), state.Marriages[0]); !ok || got != 3 {
		t.Fatalf("weight = %d, %v; want 3", got, ok)
	}
}

func TestAllianceWeightLastSuccessionRankCoversLaterPositions(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N5"}}
	// N5 is third in P2's line (1); N2 second in P1's line (2).
	if got, _ := AllianceWeight(state, allianceBalance(), state.Marriages[0]); got != 1 {
		t.Fatalf("weight = %d, want 1", got)
	}
}

func TestAllianceWeightDensityBonusIsUncapped(t *testing.T) {
	state := allianceState()
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3"}, {NobleA: "N2", NobleB: "N4"}}
	if got, _ := AllianceWeight(state, allianceBalance(), state.Marriages[1]); got != 3 {
		t.Fatalf("second marriage weight = %d, want 3 (2 + density)", got)
	}
	if got, _ := AllianceWeight(state, allianceBalance(), state.Marriages[0]); got != 4 {
		t.Fatalf("first marriage weight = %d, want 4 (3 + density)", got)
	}
}

func TestAllianceWeightIgnoresBastardMarriages(t *testing.T) {
	state := allianceState()
	state.Nobles[1].Dignities = []models.Dignity{models.DignityBastard}
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3"}, {NobleA: "N2", NobleB: "N4"}}
	if _, ok := AllianceWeight(state, allianceBalance(), state.Marriages[1]); ok {
		t.Fatal("bastard marriage must have no weight")
	}
	if got, _ := AllianceWeight(state, allianceBalance(), state.Marriages[0]); got != 3 {
		t.Fatalf("weight = %d, want 3 (no density from a bastard marriage)", got)
	}
}
