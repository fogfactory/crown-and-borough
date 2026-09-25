package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ForecastMillIncome computes, for every player, the normal mill production
// they will receive on the next action turn, ignoring any calamity or bonus
// card already drawn this turn (like ForecastIncome/ForecastFamineRisk): the
// command post projection must never leak an undrawn weather card's effect.
// Since #195, each mill credits exactly one destination (its adjacent castle
// under the same control, else its adjacent village under the same control,
// else itself), so this sums exactly the production the player's own
// settlements and self-supplied mills will receive; it no longer double-
// counts a mill adjacent to several of the player's settlements. Ravitaillement
// never happens in winter, so a winter state always forecasts nil.
func ForecastMillIncome(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]int {
	if state == nil || state.Season == models.SeasonWinter {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	totals := make(map[models.PlayerID]int, len(ctx.state.Players))
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		total := 0
		for _, source := range controlledSupplySources(ctx, ownerID) {
			total += source.production
		}
		totals[ownerID] = total
	}
	return totals
}

// MillProductionForecast is one mill's projected harvest-and-weather-adjusted
// production and its single destination, for the territory detail panel's
// mill line (see #195's millRecipient).
type MillProductionForecast struct {
	Production  int
	Destination models.TerritoryID
}

// ForecastMillProduction computes, for every mill, the normal production and
// single destination it will credit on the next action turn, ignoring any
// calamity or bonus card already drawn this turn (like ForecastMillIncome).
// Ravitaillement never happens in winter, so a winter state always forecasts
// nil.
func ForecastMillProduction(state *models.GameState, balance assetgen.Balance) map[models.TerritoryID]MillProductionForecast {
	if state == nil || state.Season == models.SeasonWinter {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	forecasts := make(map[models.TerritoryID]MillProductionForecast)
	for _, mill := range computeMillProduction(ctx) {
		forecasts[mill.millID] = MillProductionForecast{Production: mill.total(), Destination: mill.destinationID}
	}
	return forecasts
}
