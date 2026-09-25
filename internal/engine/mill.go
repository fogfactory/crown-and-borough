package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// millProduction is one mill's computed production and its single
// beneficiary this turn (see #195): reused by supply resolution, by the
// upgrade payment sources, and by the mill production forecast, so all three
// agree on exactly one destination per mill.
type millProduction struct {
	millID           models.TerritoryID
	ownerID          *models.PlayerID
	infrastructureID models.InfraID
	level            int
	destinationID    models.TerritoryID
	production       int // base mill production, 0 when suppressed by bad weather
	bonus            int // fair-weather bonus, equal to level when it applies
	suppressed       int // production lost to bad weather, equal to level when it applies
}

// total is the mill's harvest-and-weather-adjusted production actually
// credited to its destination this turn.
func (mill millProduction) total() int {
	return mill.production + mill.bonus
}

// selfSupplied reports whether the mill has no eligible adjacent castle or
// village and therefore stocks its own production locally.
func (mill millProduction) selfSupplied() bool {
	return mill.destinationID == mill.millID
}

// computeMillProduction lists every mill's production and single destination
// this turn, sorted by the mill's own trigram. It is a pure read of ctx
// (weather regions, infrastructure, control) and never mutates state.
func computeMillProduction(ctx *resolutionContext) []millProduction {
	var mills []millProduction
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		infrastructure := ctx.infrastructureAt(territoryID)
		if infrastructure == nil || infrastructure.Type != models.InfraTypeMill {
			continue
		}
		production, bonus, suppressed := millWeatherProduction(ctx, territoryID, infrastructure.Level)
		mills = append(mills, millProduction{
			millID:           territoryID,
			ownerID:          ctx.state.TerritoryStates[territoryID].OwnerID,
			infrastructureID: infrastructure.ID,
			level:            infrastructure.Level,
			destinationID:    millRecipient(ctx, territoryID),
			production:       production,
			bonus:            bonus,
			suppressed:       suppressed,
		})
	}
	return mills
}

// millWeatherProduction applies the weather calamity and bonus cards to one
// mill's level-based production: bad weather suppresses it entirely, fair
// weather doubles it (see #191). It returns the parts a sourceProductionParts
// accumulates: mill (renamed production here), bonus, and suppressed.
func millWeatherProduction(ctx *resolutionContext, millID models.TerritoryID, level int) (production, bonus, suppressed int) {
	region := regionForTerritory(ctx, millID)
	if ctx.badWeatherRegions[region] {
		return 0, 0, level
	}
	production = level
	if ctx.fairWeatherRegions[region] {
		bonus = level
	}
	return production, bonus, 0
}

// millRecipient is the single infrastructure credited by one mill's
// production (see #195): the adjacent castle under the same control as the
// mill's own territory, else the adjacent village under the same control,
// else the mill's own territory. Neutral (nil) control matches only neutral
// control, so a neutral mill never feeds a player's settlement. Among
// several same-type same-control neighbors, sortedNeighbors' ascending
// trigram order breaks the tie, exactly like #192's territory income
// destination.
func millRecipient(ctx *resolutionContext, millID models.TerritoryID) models.TerritoryID {
	millController := ctx.state.TerritoryStates[millID].OwnerID
	for _, infraType := range []models.InfraType{models.InfraTypeCastle, models.InfraTypeVillage} {
		for _, neighborID := range ctx.sortedNeighbors(millID) {
			if !ctx.hasInfrastructure(neighborID, infraType) {
				continue
			}
			if sameController(millController, ctx.state.TerritoryStates[neighborID].OwnerID) {
				return neighborID
			}
		}
	}
	return millID
}

// isSelfSuppliedMill reports whether territoryID carries a mill with no
// eligible adjacent castle or village, so its production stocks itself
// instead of a neighboring settlement.
func (ctx *resolutionContext) isSelfSuppliedMill(territoryID models.TerritoryID) bool {
	infrastructure := ctx.infrastructureAt(territoryID)
	return infrastructure != nil && infrastructure.Type == models.InfraTypeMill && millRecipient(ctx, territoryID) == territoryID
}

// millSelfProductionBreakdown is the harvest-and-weather-adjusted production
// of a self-supplied mill, in the same shape sourceProductionBreakdown uses
// for a settlement's mill income.
func millSelfProductionBreakdown(ctx *resolutionContext, millID models.TerritoryID) sourceProductionParts {
	infrastructure := ctx.infrastructureAt(millID)
	if infrastructure == nil || infrastructure.Type != models.InfraTypeMill {
		return sourceProductionParts{}
	}
	production, bonus, suppressed := millWeatherProduction(ctx, millID, infrastructure.Level)
	return sourceProductionParts{mill: production, bonus: bonus, suppressed: suppressed}
}

// sameController reports whether two territory controllers are the same
// player, treating nil (neutral) as its own distinct controller: a neutral
// territory only matches another neutral territory.
func sameController(left, right *models.PlayerID) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
