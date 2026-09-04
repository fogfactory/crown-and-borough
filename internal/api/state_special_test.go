package api

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestProjectStateForPlayerProjectsOnlyCurrentHand(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}, {ID: "C2", Kind: models.CardKindPlague}},
		DrawPile: []models.SpecialCardID{"C2"}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}, "P2": {}},
	}
	p1 := ProjectStateForPlayer(state, "P1")
	p2 := ProjectStateForPlayer(state, "P2")
	if len(p1.SpecialHand) != 1 || p1.SpecialHand[0] != models.CardKindFairWeather {
		t.Fatalf("P1 special hand = %#v, want fair_weather", p1.SpecialHand)
	}
	if len(p2.SpecialHand) != 0 {
		t.Fatalf("P2 special hand = %#v, want empty", p2.SpecialHand)
	}
}
