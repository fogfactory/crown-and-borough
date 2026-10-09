package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ForecastRevoltTargets lists the territories a Révolte card can be played on
// in the current season: an active famine in the region, a fief or bishopric
// taxed this turn or the previous one, or a trial that executed a lady there
// (specs/ordres-speciaux.md). It reads the persisted state only: a tax played
// on the sheet being written is not part of it. It returns nil in winter, where
// no card can be played.
func ForecastRevoltTargets(state *models.GameState, balance assetgen.Balance) []models.TerritoryID {
	if state == nil || state.Season == models.SeasonWinter {
		return nil
	}
	ctx := newResolutionContext(cloneGameState(state), balance)
	execution := &ExecutionContext{resolution: ctx, season: state.Season}
	var targets []models.TerritoryID
	for _, territory := range ctx.state.Territories {
		order := models.DeckOrder{Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: territory.ID}
		if playable, _ := (revoltCardDefinition{}).CanPlay(execution, order); playable {
			targets = append(targets, territory.ID)
		}
	}
	return targets
}
