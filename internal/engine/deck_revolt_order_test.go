package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestRevoltOrderApplyRequiresFamineAndConsumesCard(t *testing.T) {
	state := models.NewGameState()
	state.Territories = []models.Territory{
		{ID: "AAA", Name: "AAA", Terrain: models.TerrainPlain},
	}
	state.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"AAA"}}}
	state.Auguries[1] = models.YearAugury{Year: 1, Calamities: []models.Calamity{{Kind: models.CardKindFamine, Year: 1, Season: models.SeasonSpring, RegionSeed: "ROS"}}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{{ID: "C1", Kind: models.CardKindRevolt}},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}}, Discard: []models.SpecialCardID{},
	}
	ctx := newResolutionContext(state, testBalance())
	revoltOrder{playerID: "P1", order: models.DeckOrder{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "AAA"}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1", season: models.SeasonSpring})
	if len(state.SpecialDeck.Hands["P1"]) != 0 || len(state.SpecialDeck.Discard) != 1 || len(ctx.deckIntents) != 1 {
		t.Fatalf("revolt application = %#v/%#v/%#v", state.SpecialDeck.Hands, state.SpecialDeck.Discard, ctx.deckIntents)
	}
}
