package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ForecastMillIncome computes, for every player, the normal mill production
// that will be credited to their controlled castles and villages on the next
// action turn, ignoring any calamity or bonus card already drawn this turn
// (like ForecastIncome/ForecastFamineRisk): the command post projection must
// never leak an undrawn weather card's effect. A mill currently credits every
// adjacent controlled castle or village independently regardless of who owns
// the mill (see sourceProductionBreakdown; issue #195 will narrow this to a
// single destination), so this sums exactly what each of the player's
// controlled settlements will receive. Ravitaillement never happens in
// winter, so a winter state always forecasts nil.
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
