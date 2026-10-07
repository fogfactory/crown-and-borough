package engine

import (
	"slices"
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
		DensityBonus: 1, HeadMinWeight: 5, MixedMinWeight: 2,
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

func TestMarriageCategoryThresholds(t *testing.T) {
	state := allianceState()
	state.Fiefs = append(state.Fiefs, models.Fief{ID: "F2", OwnerID: "P2", Title: models.FiefTitleCounty})
	holder := models.NobleID("N3")
	state.Fiefs[1].HolderNobleID = &holder
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3"}}
	// N1 = 7, N3 = head (3) + county (2) = 5.
	if got, ok := MarriageCategory(state, allianceBalance(), state.Marriages[0]); !ok || got != AllianceHead {
		t.Fatalf("category = %q, %v; want head", got, ok)
	}
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N4"}}
	if got, _ := MarriageCategory(state, allianceBalance(), state.Marriages[0]); got != AllianceMixed {
		t.Fatalf("category = %q, want mixed (N4 weighs 2)", got)
	}
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N5"}}
	if got, _ := MarriageCategory(state, allianceBalance(), state.Marriages[0]); got != AllianceSecondary {
		t.Fatalf("category = %q, want secondary", got)
	}
}

func TestMarriageCategoryReclassifiesOnTitleLossAndDeath(t *testing.T) {
	state := allianceState()
	state.Fiefs = append(state.Fiefs, models.Fief{ID: "F2", OwnerID: "P2", Title: models.FiefTitleCounty})
	holder := models.NobleID("N3")
	state.Fiefs[1].HolderNobleID = &holder
	state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "N3"}}
	balance := allianceBalance()
	if got, _ := MarriageCategory(state, balance, state.Marriages[0]); got != AllianceHead {
		t.Fatalf("before = %q, want head", got)
	}
	state.Fiefs[1].HolderNobleID = nil
	if got, _ := MarriageCategory(state, balance, state.Marriages[0]); got != AllianceMixed {
		t.Fatalf("after title loss = %q, want mixed", got)
	}
	state.Nobles = state.Nobles[1:]
	if _, ok := MarriageCategory(state, balance, state.Marriages[0]); ok {
		t.Fatal("a marriage ended by a death has no category")
	}
}

func TestActiveHeadMarriageIsHighestWeightAndRebasculates(t *testing.T) {
	balance := allianceBalance()
	balance.Alliance.HeadMinWeight = 3
	state := allianceState()
	state.Fiefs = append(state.Fiefs,
		models.Fief{ID: "F2", OwnerID: "P1", HolderNobleID: ptrNoble("N2"), Title: models.FiefTitleCounty},
		models.Fief{ID: "F3", OwnerID: "P2", HolderNobleID: ptrNoble("N4"), Title: models.FiefTitleCounty})
	// N2-N3: min(2+2, 3) = 3 (head). N1-N4: min(3+4, 2+2) = 4 (head, heavier).
	state.Marriages = []models.Marriage{{NobleA: "N2", NobleB: "N3", Turn: 1}, {NobleA: "N1", NobleB: "N4", Turn: 2}}
	active, ok := ActiveHeadMarriage(state, balance, "P1")
	if !ok || active.NobleA != "N1" {
		t.Fatalf("active = %+v, %v; want N1-N4", active, ok)
	}
	if got, _ := EffectiveMarriageCategory(state, balance, "P1", state.Marriages[0]); got != AllianceMixed {
		t.Fatalf("other head = %s, want mixed", got)
	}
	// The noble carrying the active head dies: the next head takes over.
	state.Nobles = slices.DeleteFunc(slices.Clone(state.Nobles), func(n models.Noble) bool { return n.ID == "N1" })
	active, ok = ActiveHeadMarriage(state, balance, "P1")
	if !ok || active.NobleA != "N2" {
		t.Fatalf("after death active = %+v, %v; want N2-N3", active, ok)
	}
	if got, _ := EffectiveMarriageCategory(state, balance, "P1", active); got != AllianceHead {
		t.Fatalf("new active = %s, want head", got)
	}
}

func ptrNoble(id models.NobleID) *models.NobleID { return &id }
