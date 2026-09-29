package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Founding reasons reported on an EventTypeProsperityFounded event: which
// destination priority level or fallback tier produced it
// (economie.md#prospérité).
const (
	prosperityReasonFief          = "prosperity_fief"
	prosperityReasonControlled    = "prosperity_controlled"
	prosperityReasonFree          = "prosperity_free"
	prosperityReasonDepotUpgraded = "prosperity_depot_upgraded"
	prosperityReasonMillUpgraded  = "prosperity_mill_upgraded"
)

// resolveProsperity founds new villages from the winter conservation loss
// (economie.md#prospérité): the map-wide loss just applied by
// conserveWinterStocks, summed across every territory and player, triggers
// one founding per full multiple of balance.ProsperityLossThreshold crossed.
// Runs after conserveWinterStocks and before repatriateWinterStocks, so
// stockBefore still reflects the pre-conservation stock.
func (ctx *resolutionContext) resolveProsperity(stockBefore map[models.TerritoryID]int) {
	if ctx.balance.ProsperityLossThreshold < 1 {
		return
	}
	ranked, total := ctx.prosperityRankedLosses(stockBefore)
	triggered := total / ctx.balance.ProsperityLossThreshold
	if triggered > len(ranked) {
		triggered = len(ranked)
	}
	for index := 0; index < triggered; index++ {
		ctx.foundProsperityVillage(ranked[index])
	}
}

// prosperityRankedLosses returns every territory that lost stock to winter
// conservation this year, ranked by loss descending (trigram ascending on
// ties), along with the map-wide total loss.
func (ctx *resolutionContext) prosperityRankedLosses(stockBefore map[models.TerritoryID]int) ([]models.TerritoryID, int) {
	type loss struct {
		territoryID models.TerritoryID
		amount      int
	}
	losses := make([]loss, 0)
	total := 0
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		amount := stockBefore[territoryID] - ctx.state.TerritoryStates[territoryID].Resources
		if amount <= 0 {
			continue
		}
		total += amount
		losses = append(losses, loss{territoryID: territoryID, amount: amount})
	}
	sort.Slice(losses, func(i, j int) bool {
		if losses[i].amount != losses[j].amount {
			return losses[i].amount > losses[j].amount
		}
		return losses[i].territoryID < losses[j].territoryID
	})
	ranked := make([]models.TerritoryID, len(losses))
	for index, entry := range losses {
		ranked[index] = entry.territoryID
	}
	return ranked, total
}

// foundProsperityVillage resolves one triggered founding from originID: the
// nearest eligible free tile in priority order, falling back to upgrading a
// supply depot then a mill to a village when no tile qualifies anywhere. It
// mutates state immediately, so a village founded by this call is visible to
// the next one in the same winter (economie.md#prospérité).
func (ctx *resolutionContext) foundProsperityVillage(originID models.TerritoryID) {
	if destinationID, reason, found := ctx.prosperityDestination(originID); found {
		infrastructure := ctx.addWinterInfrastructure(models.InfraTypeVillage, destinationID)
		ctx.emitProsperityFounded(originID, destinationID, infrastructure, reason)
		return
	}
	if depotID := ctx.closestTerritoryWithInfrastructure(originID, models.InfraTypeSupplyDepot); depotID != "" {
		infrastructure := ctx.upgradeInfrastructureToVillage(depotID)
		ctx.emitProsperityFounded(originID, depotID, infrastructure, prosperityReasonDepotUpgraded)
		return
	}
	if millID := ctx.closestTerritoryWithInfrastructure(originID, models.InfraTypeMill); millID != "" {
		infrastructure := ctx.upgradeInfrastructureToVillage(millID)
		ctx.emitProsperityFounded(originID, millID, infrastructure, prosperityReasonMillUpgraded)
		return
	}
	// No free tile anywhere and no depot or mill to upgrade: nothing happens
	// (economie.md#prospérité).
}

func (ctx *resolutionContext) emitProsperityFounded(originID, destinationID models.TerritoryID, infrastructure *models.Infrastructure, reason string) {
	if infrastructure == nil {
		return
	}
	ownerID, _ := ctx.controllerAtStart(destinationID)
	ctx.events = append(ctx.events, Event{
		Type:               EventTypeProsperityFounded,
		Phase:              winterPhase,
		OwnerID:            ownerID,
		SourceID:           originID,
		DestinationID:      destinationID,
		InfrastructureID:   infrastructure.ID,
		InfrastructureType: models.InfraTypeVillage,
		Reason:             reason,
	})
}

// prosperityDestination finds the nearest free tile eligible to receive a
// prosperity founding from originID, in priority order: a fief of the player
// who controls originID, else a tile that player controls, else any
// remaining free tile. Returns the tile, the priority reason, and whether one
// was found.
func (ctx *resolutionContext) prosperityDestination(originID models.TerritoryID) (models.TerritoryID, string, bool) {
	if playerID, controlled := ctx.controllerAtStart(originID); controlled {
		if destinationID := ctx.prosperityFreeTile(originID, func(candidateID models.TerritoryID) bool {
			return ctx.territoryFief(playerID, candidateID) != nil
		}); destinationID != "" {
			return destinationID, prosperityReasonFief, true
		}
		if destinationID := ctx.prosperityFreeTile(originID, func(candidateID models.TerritoryID) bool {
			return ctx.controlsTerritory(playerID, candidateID)
		}); destinationID != "" {
			return destinationID, prosperityReasonControlled, true
		}
	}
	if destinationID := ctx.prosperityFreeTile(originID, func(models.TerritoryID) bool { return true }); destinationID != "" {
		return destinationID, prosperityReasonFree, true
	}
	return "", "", false
}

// prosperityFreeTile does a level-by-level BFS over crossable borders from
// originID, returning the closest free tile (no infrastructure), not
// adjacent to an existing village or castle, satisfying match, with a
// trigram tie-break among equidistant candidates. It returns "" when none is
// reachable.
func (ctx *resolutionContext) prosperityFreeTile(originID models.TerritoryID, match func(models.TerritoryID) bool) models.TerritoryID {
	type queueItem struct {
		territoryID models.TerritoryID
		distance    int
	}
	queue := []queueItem{{territoryID: originID}}
	visited := map[models.TerritoryID]bool{originID: true}
	for len(queue) > 0 {
		distance := queue[0].distance
		level := make([]queueItem, 0)
		for len(queue) > 0 && queue[0].distance == distance {
			level = append(level, queue[0])
			queue = queue[1:]
		}
		candidates := make([]models.TerritoryID, 0)
		for _, item := range level {
			if ctx.prosperityTileEligible(item.territoryID) && match(item.territoryID) {
				candidates = append(candidates, item.territoryID)
			}
		}
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool { return candidates[i] < candidates[j] })
			return candidates[0]
		}
		for _, item := range level {
			for _, neighborID := range ctx.sortedNeighbors(item.territoryID) {
				if !visited[neighborID] {
					visited[neighborID] = true
					queue = append(queue, queueItem{territoryID: neighborID, distance: distance + 1})
				}
			}
		}
	}
	return ""
}

// prosperityTileEligible reports whether territoryID carries no
// infrastructure and is not adjacent to an existing village or castle
// (economie.md#prospérité): the non-adjacency constraint applies at every
// destination priority level, not only the last.
func (ctx *resolutionContext) prosperityTileEligible(territoryID models.TerritoryID) bool {
	if ctx.infrastructureAt(territoryID) != nil {
		return false
	}
	for _, neighborID := range ctx.sortedNeighbors(territoryID) {
		if ctx.hasSettlement(neighborID) {
			return false
		}
	}
	return true
}

// closestTerritoryWithInfrastructure returns the territory closest to
// originID (crossable-border distance, trigram tie-break) carrying an
// infrastructure of kind, regardless of its controller, or "" if none
// exists. Used by the prosperity founding cascade (economie.md#prospérité):
// unlike prosperityFreeTile, it ignores adjacency and control entirely, since
// it upgrades infrastructure already standing rather than founding on a free
// tile.
func (ctx *resolutionContext) closestTerritoryWithInfrastructure(originID models.TerritoryID, kind models.InfraType) models.TerritoryID {
	distances := ctx.winterDistances(originID)
	best := models.TerritoryID("")
	bestDistance := 0
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		if !ctx.hasInfrastructure(territoryID, kind) {
			continue
		}
		distance, reachable := distances[territoryID]
		if !reachable {
			continue
		}
		if best == "" || distance < bestDistance {
			best = territoryID
			bestDistance = distance
		}
	}
	return best
}

// upgradeInfrastructureToVillage turns the infrastructure at territoryID
// into a fresh, unfortified level-1 village in place, keeping its ID and
// territory (economie.md#prospérité cascading fallback).
func (ctx *resolutionContext) upgradeInfrastructureToVillage(territoryID models.TerritoryID) *models.Infrastructure {
	infrastructure := ctx.infrastructureAt(territoryID)
	if infrastructure == nil {
		return nil
	}
	infrastructure.Type = models.InfraTypeVillage
	infrastructure.Level = 1
	infrastructure.Fortified = false
	return infrastructure
}
