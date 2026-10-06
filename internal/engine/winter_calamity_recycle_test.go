package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// A calamity card scheduled for a year leaves the augury at the next winter
// and goes back to the discard pile, so calamities never run out of the deck.
func TestWinterRecyclesTheCalamityCardsOfTheEndingYear(t *testing.T) {
	state := deckOrdersState(t)
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{
			{ID: "c1", Kind: models.CardKindPlague}, {ID: "c2", Kind: models.CardKindFamine},
			{ID: "b1", Kind: models.CardKindFairWeather},
		},
		DrawPile: []models.SpecialCardID{"b1"}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {}, "P2": {}},
	}
	state.Auguries[state.Year()] = models.YearAugury{
		Year: state.Year(), Revealed: true,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
		Calamities: []models.Calamity{
			{CardID: "c1", Kind: models.CardKindPlague, Year: state.Year(), Season: models.SeasonSpring, RegionSeed: "AAA"},
			{CardID: "c2", Kind: models.CardKindFamine, Year: state.Year(), Season: models.SeasonSummer, RegionSeed: "AAA"},
		},
	}
	resolution := resolveNobleDeckWinter(t, state, nil)
	if _, kept := resolution.State.Auguries[state.Year()]; kept {
		t.Error("the ending year's augury was kept")
	}
	deck := resolution.State.SpecialDeck
	where := func(id models.SpecialCardID) bool {
		for _, hand := range deck.Hands {
			if slices.Contains(hand, id) {
				return true
			}
		}
		return slices.Contains(deck.DrawPile, id) || slices.Contains(deck.Discard, id)
	}
	for _, id := range []models.SpecialCardID{"c1", "c2"} {
		if !where(id) {
			t.Errorf("calamity card %s is in no pile after the winter", id)
		}
	}
	validateTestState(t, resolution.State)
}
