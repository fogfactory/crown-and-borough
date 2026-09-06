package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestCardKindClassification(t *testing.T) {
	for _, kind := range []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt} {
		if !kind.IsValid() || !kind.IsBonus() || kind.IsCalamity() {
			t.Errorf("bonus kind %q classification is invalid", kind)
		}
	}
	for _, kind := range []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindFamine} {
		if !kind.IsValid() || kind.IsBonus() || !kind.IsCalamity() {
			t.Errorf("calamity kind %q classification is invalid", kind)
		}
	}
	if got, ok := models.CardKindFairWeather.CanceledCalamity(); !ok || got != models.CardKindBadWeather {
		t.Errorf("fair weather cancellation = %q/%t, want bad weather/true", got, ok)
	}
	if got, ok := models.CardKindAbundantHarvest.CanceledCalamity(); !ok || got != models.CardKindFamine {
		t.Errorf("abundant harvest cancellation = %q/%t, want famine/true", got, ok)
	}
	if _, ok := models.CardKindPlague.CanceledCalamity(); ok {
		t.Fatal("plague unexpectedly cancels a calamity")
	}
}

func TestValidateSpecialDeckCardLocations(t *testing.T) {
	state := validState()
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}, {ID: "C2", Kind: models.CardKindFairWeather}},
		DrawPile: []models.SpecialCardID{"C1"},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	if err := state.Validate(); err == nil {
		t.Fatal("Validate() = nil, want duplicate card location error")
	}
	state.SpecialDeck.DrawPile = nil
	state.SpecialDeck.Discard = []models.SpecialCardID{"C1"}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C2"}
	if err := state.Validate(); err != nil {
		t.Fatalf("valid special deck: %v", err)
	}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C2"}
	state.SpecialDeck.Cards[1].Kind = models.CardKindPlague
	if err := state.Validate(); err == nil {
		t.Fatal("Validate() = nil, want calamity in hand error")
	}
}

func TestValidateNeutralArmy(t *testing.T) {
	state := validState()
	state.Armies = append(state.Armies, models.Army{ID: "A3", OwnerID: models.NeutralPlayerID, TerritoryID: "FOU", Size: 2})
	state.TerritoryStates["FOU"] = models.TerritoryState{Army: ptrArmyID("A3")}
	state.NextArmyID = 4
	if err := state.Validate(); err != nil {
		t.Fatalf("neutral army: %v", err)
	}
	chainID := models.ChainID("C1")
	state.Armies[2].ChainID = &chainID
	if err := state.Validate(); err == nil {
		t.Fatal("Validate() = nil, want neutral chain error")
	}
}
