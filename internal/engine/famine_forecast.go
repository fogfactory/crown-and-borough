package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ArmyFamineRisk is one player army whose local production plus the supply
// sources it can reach do not appear to cover its ration demand under
// ForecastFamineRisk's heuristic (see that function's comment). Deficit is
// the estimated ration shortfall (demand minus what is locally and reachably
// available), not a troop count: the actual troop loss at resolution depends
// on assignSupply/resolveSupplyStocks/selectAssignedFamine, which this
// forecast deliberately does not simulate.
type ArmyFamineRisk struct {
	ArmyID      models.ArmyID      `json:"armyId"`
	TerritoryID models.TerritoryID `json:"territoryId"`
	Size        int                `json:"size"`
	Deficit     int                `json:"deficit"`
}

// FamineRiskForecast is one player's projected ravitaillement for the next
// action turn: TotalDemand is the summed ration cost of every one of their
// armies, regardless of risk, and ArmiesAtRisk lists the ones the heuristic
// flags.
type FamineRiskForecast struct {
	TotalDemand  int
	ArmiesAtRisk []ArmyFamineRisk
}

// ForecastFamineRisk computes, for every player, the normal ration demand of
// their armies for the next action turn and which of them look at risk of
// famine, ignoring any calamity or bonus card already drawn this turn (like
// ForecastIncome/ForecastTerritoryIncome): the command post projection must
// never leak an undrawn harvest card's effect. Ravitaillement never happens
// in winter, so a winter state always forecasts nil.
//
// The risk heuristic is intentionally simple, not a simulation of the actual
// competitive allocation performed by assignSupply/resolveSupplyStocks/
// selectAssignedFamine during resolution: it treats every one of the
// player's armies independently, and for each one adds up its own local
// production plus the full current stock and this turn's production of
// every controlled source it can reach through the supply network (same
// reachability as controlledSupplySources), without splitting a shared
// source's capacity across the player's other armies that could also reach
// it. Two armies that are each not "at risk" individually can therefore
// still compete for the same source and see one of them starve at
// resolution; the front must present this as an estimate, not a guarantee.
func ForecastFamineRisk(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]FamineRiskForecast {
	if state == nil || state.Season == models.SeasonWinter {
		return nil
	}
	clone := cloneGameState(state)
	ctx := newResolutionContext(clone, balance)
	resolveTerritoryIncome(ctx)
	receivedRations := resolveRations(ctx)

	forecasts := make(map[models.PlayerID]FamineRiskForecast, len(ctx.state.Players))
	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		sources := controlledSupplySources(ctx, ownerID)
		forecast := FamineRiskForecast{}
		for _, army := range startArmiesForPlayer(ctx, ownerID) {
			demand := armyCost(army.Size, ctx.balance.CostBase)
			forecast.TotalDemand += demand
			remaining := demand - receivedRations[army.ID]
			if remaining <= 0 {
				continue
			}
			if available := reachableSupplyCapacity(ctx, army.TerritoryID, sources); remaining > available {
				forecast.ArmiesAtRisk = append(forecast.ArmiesAtRisk, ArmyFamineRisk{
					ArmyID:      army.ID,
					TerritoryID: army.TerritoryID,
					Size:        army.Size,
					Deficit:     remaining - available,
				})
			}
		}
		forecasts[ownerID] = forecast
	}
	return forecasts
}

// reachableSupplyCapacity sums, for a single army considered alone, the full
// current stock and this turn's production of every one of the given
// controlled sources it can reach through the supply network. It does not
// split a source's capacity across several armies that could also reach it,
// which is the simplification ForecastFamineRisk's doc comment describes.
func reachableSupplyCapacity(ctx *resolutionContext, territoryID models.TerritoryID, sources []*supplySource) int {
	capacity := 0
	for _, source := range sources {
		if _, reachable := source.reachable[territoryID]; !reachable {
			continue
		}
		capacity += ctx.state.TerritoryStates[source.territoryID].Resources + source.production
	}
	return capacity
}
