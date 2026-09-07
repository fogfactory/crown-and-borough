package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestAbundantHarvestOrderApply(t *testing.T) {
	state := models.NewGameState()
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{{ID: "C1", Kind: models.CardKindAbundantHarvest}},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}, Discard: []models.SpecialCardID{},
	}
	ctx := newResolutionContext(state, testBalance())
	abundantHarvestOrder{playerID: "P1", order: models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindAbundantHarvest}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonAutumn})
	if len(state.SpecialDeck.Hands["P1"]) != 0 || len(state.SpecialDeck.Discard) != 1 || len(ctx.deckIntents) != 1 {
		t.Fatalf("abundant harvest application = %#v/%#v/%#v", state.SpecialDeck.Hands, state.SpecialDeck.Discard, ctx.deckIntents)
	}
}
