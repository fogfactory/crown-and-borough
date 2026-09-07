package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type supplySource struct {
	territoryID   models.TerritoryID
	ownerID       models.PlayerID
	production    int
	demand        int
	rations       map[models.TerritoryID]int
	stockConsumed int
	reachable     map[models.TerritoryID]int
}

type supplyAssignment struct {
	army     models.Army
	demand   int
	source   *supplySource
	distance int
}

type famineCandidate struct {
	army     models.Army
	demand   int
	sourceID models.TerritoryID
	distance int
}

// resolveSupply calculates the complete start-of-turn supply phase before any
// order can move an army or change control.
func resolveSupply(ctx *resolutionContext) {
	receivedRations := resolveRations(ctx)
	produceNeutralVillageStocks(ctx)
	allSources := make([]*supplySource, 0)
	directFamine := make([]famineCandidate, 0)
	assignedFamine := make([]famineCandidate, 0)

	for _, ownerID := range sortedPlayerIDs(ctx.state.Players) {
		sources := controlledSupplySources(ctx, ownerID)
		allSources = append(allSources, sources...)
		assignments, direct := assignSupply(ctx, ownerID, sources, receivedRations)
		directFamine = append(directFamine, direct...)
		delta := resolveSupplyStocks(ctx, sources)
		assignedFamine = append(assignedFamine, selectAssignedFamine(ctx, assignments, delta)...)
	}

	for _, source := range allSources {
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeSupply,
			Phase:         0,
			SourceID:      source.territoryID,
			OwnerID:       source.ownerID,
			Production:    source.production,
			Demand:        source.demand,
			Rations:       cloneRations(source.rations),
			StockConsumed: source.stockConsumed,
			StockAfter:    ctx.state.TerritoryStates[source.territoryID].Resources,
		})
	}

	sortDirectFamine(ctx, directFamine)
	for _, candidate := range directFamine {
		ctx.resolveFamine(candidate)
	}
	sortAssignedFamine(ctx, assignedFamine)
	for _, candidate := range assignedFamine {
		ctx.resolveFamine(candidate)
	}
	updateSupplyEventStocks(ctx)
}

func produceNeutralVillageStocks(ctx *resolutionContext) {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID != nil || !ctx.hasInfrastructure(territoryID, models.InfraTypeVillage) {
			continue
		}
		production := sourceProduction(ctx, territoryID)
		state.Resources += production
		ctx.state.TerritoryStates[territoryID] = state
		ctx.events = append(ctx.events, Event{
			Type:       EventTypeSupply,
			Phase:      0,
			SourceID:   territoryID,
			Production: production,
			StockAfter: state.Resources,
		})
	}
}

func updateSupplyEventStocks(ctx *resolutionContext) {
	for index := range ctx.events {
		if ctx.events[index].Type != EventTypeSupply {
			continue
		}
		state, exists := ctx.state.TerritoryStates[ctx.events[index].SourceID]
		if exists {
			ctx.events[index].StockAfter = state.Resources
		}
	}
}

func resolveRations(ctx *resolutionContext) map[models.ArmyID]int {
	received := make(map[models.ArmyID]int, len(ctx.startArmiesByID))
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		army := ctx.startArmyAt(territoryID)
		if army == nil {
			continue
		}
		distribution := distributeRations(rationProduction(ctx, territoryID), []models.Army{*army})
		received[army.ID] = distribution[army.ID]
	}
	return received
}

// armyCost returns the exponential stockable-resource cost before local rations
// are deducted.
func armyCost(size, costBase int) int {
	cost := 1
	for troops := 1; troops < size; troops++ {
		cost *= costBase
	}
	return cost
}

// distributeRations grants at most one ration to each army. Equal sizes are
// resolved by the territory trigram; a same-territory tie is invalid game data
// and therefore intentionally preserves the input order.
func distributeRations(rations int, armies []models.Army) map[models.ArmyID]int {
	received := make(map[models.ArmyID]int, len(armies))
	ordered := append([]models.Army(nil), armies...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Size != ordered[j].Size {
			return ordered[i].Size > ordered[j].Size
		}
		return ordered[i].TerritoryID < ordered[j].TerritoryID
	})
	for _, army := range ordered {
		if rations == 0 {
			break
		}
		received[army.ID] = 1
		rations--
	}
	return received
}

func rationProduction(ctx *resolutionContext, territoryID models.TerritoryID) int {
	territory := ctx.territoriesByID[territoryID]
	if territory == nil {
		return 0
	}
	production := ctx.balance.RationTerrain[territory.Terrain]
	regionSeed := regionForTerritory(ctx, territoryID)
	if ctx.hasSettlement(territoryID) && !ctx.famineRegions[regionSeed] {
		production += ctx.balance.InfraRationsBonus
	}
	production += ctx.bonusRationRegions[regionSeed] * ctx.balance.SpecialOrders.Effects.BonusArmyRation
	return production
}

func controlledSupplySources(ctx *resolutionContext, ownerID models.PlayerID) []*supplySource {
	sources := make([]*supplySource, 0)
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID == nil || *state.OwnerID != ownerID || !ctx.hasSettlement(territoryID) {
			continue
		}
		sources = append(sources, &supplySource{
			territoryID: territoryID,
			ownerID:     ownerID,
			production:  sourceProduction(ctx, territoryID),
			rations:     make(map[models.TerritoryID]int),
			reachable:   supplyNetwork(ctx, territoryID, ownerID),
		})
	}
	return sources
}

func sourceProduction(ctx *resolutionContext, territoryID models.TerritoryID) int {
	production := ctx.balance.BaseProduction
	locations := append([]models.TerritoryID{territoryID}, ctx.sortedNeighbors(territoryID)...)
	for _, locationID := range locations {
		infrastructure := ctx.infrastructureAt(locationID)
		if infrastructure == nil || infrastructure.Type != models.InfraTypeMill {
			continue
		}
		regionSeed := regionForTerritory(ctx, locationID)
		if ctx.famineRegions[regionSeed] {
			continue
		}
		production += infrastructure.Level
		production += ctx.bonusMillRegions[regionSeed] * ctx.balance.SpecialOrders.Effects.BonusMillProduction
	}
	return production
}

// supplyNetwork visits each territory once per source. That makes every depot
// bonus apply once while preserving the shortest BFS distance used for source
// assignment.
func supplyNetwork(ctx *resolutionContext, sourceID models.TerritoryID, ownerID models.PlayerID) map[models.TerritoryID]int {
	type visit struct {
		territoryID models.TerritoryID
		distance    int
		remaining   int
	}

	reachable := map[models.TerritoryID]int{sourceID: 0}
	queue := []visit{{territoryID: sourceID, remaining: ctx.balance.SupplyRange}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.remaining == 0 {
			continue
		}
		for _, neighborID := range ctx.sortedNeighbors(current.territoryID) {
			if _, visited := reachable[neighborID]; visited {
				continue
			}
			state := ctx.state.TerritoryStates[neighborID]
			if state.OwnerID != nil && *state.OwnerID != ownerID {
				continue
			}
			remaining := current.remaining - 1
			if ctx.isControlledDepot(neighborID, ownerID) {
				remaining += ctx.balance.DepotRangeBonus
			}
			reachable[neighborID] = current.distance + 1
			queue = append(queue, visit{
				territoryID: neighborID,
				distance:    current.distance + 1,
				remaining:   remaining,
			})
		}
	}
	return reachable
}

func assignSupply(
	ctx *resolutionContext,
	ownerID models.PlayerID,
	sources []*supplySource,
	receivedRations map[models.ArmyID]int,
) ([]supplyAssignment, []famineCandidate) {
	assignments := make([]supplyAssignment, 0)
	direct := make([]famineCandidate, 0)
	for _, army := range startArmiesForPlayer(ctx, ownerID) {
		demand := armyCost(army.Size, ctx.balance.CostBase) - receivedRations[army.ID]
		if demand == 0 {
			continue
		}
		source, distance := closestSupplySource(ctx, army.TerritoryID, sources)
		if source == nil {
			direct = append(direct, famineCandidate{army: army, demand: demand})
			continue
		}
		source.demand += demand
		if receivedRations[army.ID] > 0 {
			source.rations[army.TerritoryID] += receivedRations[army.ID]
		}
		assignments = append(assignments, supplyAssignment{
			army:     army,
			demand:   demand,
			source:   source,
			distance: distance,
		})
	}
	return assignments, direct
}

func closestSupplySource(ctx *resolutionContext, territoryID models.TerritoryID, sources []*supplySource) (*supplySource, int) {
	var closest *supplySource
	distance := 0
	for _, source := range sources {
		candidateDistance, reachable := source.reachable[territoryID]
		if !reachable {
			continue
		}
		if closest != nil && (candidateDistance > distance || candidateDistance == distance && source.territoryID >= closest.territoryID) {
			continue
		}
		closest = source
		distance = candidateDistance
	}
	return closest, distance
}

func resolveSupplyStocks(ctx *resolutionContext, sources []*supplySource) int {
	totalDemand := 0
	totalProduction := 0
	for _, source := range sources {
		totalDemand += source.demand
		totalProduction += source.production
	}
	if totalDemand <= totalProduction {
		for _, source := range sources {
			excess := source.production - source.demand
			if excess <= 0 {
				continue
			}
			state := ctx.state.TerritoryStates[source.territoryID]
			state.Resources += excess
			ctx.state.TerritoryStates[source.territoryID] = state
		}
		return 0
	}

	delta := totalDemand - totalProduction
	stocks := append([]*supplySource(nil), sources...)
	sort.SliceStable(stocks, func(i, j int) bool {
		left := ctx.state.TerritoryStates[stocks[i].territoryID]
		right := ctx.state.TerritoryStates[stocks[j].territoryID]
		if left.Resources != right.Resources {
			return left.Resources < right.Resources
		}
		return stocks[i].territoryID < stocks[j].territoryID
	})
	for _, source := range stocks {
		if delta == 0 {
			break
		}
		state := ctx.state.TerritoryStates[source.territoryID]
		consumed := min(state.Resources, delta)
		state.Resources -= consumed
		ctx.state.TerritoryStates[source.territoryID] = state
		source.stockConsumed += consumed
		delta -= consumed
	}
	return delta
}

func selectAssignedFamine(ctx *resolutionContext, assignments []supplyAssignment, delta int) []famineCandidate {
	if delta == 0 {
		return nil
	}
	sort.SliceStable(assignments, func(i, j int) bool {
		if assignments[i].distance != assignments[j].distance {
			return assignments[i].distance > assignments[j].distance
		}
		if assignments[i].army.Size != assignments[j].army.Size {
			return assignments[i].army.Size > assignments[j].army.Size
		}
		return assignments[i].army.TerritoryID < assignments[j].army.TerritoryID
	})
	candidates := make([]famineCandidate, 0)
	for _, assignment := range assignments {
		if delta == 0 {
			break
		}
		candidates = append(candidates, famineCandidate{
			army:     assignment.army,
			demand:   assignment.demand,
			sourceID: assignment.source.territoryID,
			distance: assignment.distance,
		})
		delta -= assignment.demand
	}
	return candidates
}

func sortDirectFamine(ctx *resolutionContext, candidates []famineCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].army.Size != candidates[j].army.Size {
			return candidates[i].army.Size > candidates[j].army.Size
		}
		return candidates[i].army.TerritoryID < candidates[j].army.TerritoryID
	})
}

func sortAssignedFamine(ctx *resolutionContext, candidates []famineCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance > candidates[j].distance
		}
		if candidates[i].army.Size != candidates[j].army.Size {
			return candidates[i].army.Size > candidates[j].army.Size
		}
		return candidates[i].army.TerritoryID < candidates[j].army.TerritoryID
	})
}

func (ctx *resolutionContext) resolveFamine(candidate famineCandidate) {
	event := Event{
		Type:        EventTypeFamine,
		Phase:       0,
		ArmyID:      candidate.army.ID,
		OwnerID:     candidate.army.OwnerID,
		TerritoryID: candidate.army.TerritoryID,
		SourceID:    candidate.sourceID,
		Troops:      candidate.army.Size,
	}
	infrastructure := ctx.infrastructureAt(candidate.army.TerritoryID)
	if infrastructure != nil {
		event.InfrastructureID = infrastructure.ID
		event.InfrastructureType = infrastructure.Type
		ctx.removeInfrastructure(infrastructure.ID)
		gain := ctx.balance.PillageBonus - candidate.demand
		if gain >= 0 {
			event.SavedByPillage = true
			if gain > 0 {
				creditTerritoryID := ctx.closestControlledSettlement(candidate.army.TerritoryID, candidate.army.OwnerID)
				if creditTerritoryID != "" {
					creditState := ctx.state.TerritoryStates[creditTerritoryID]
					creditState.Resources += gain
					ctx.state.TerritoryStates[creditTerritoryID] = creditState
					event.ResourceCredit = gain
					event.CreditTerritoryID = creditTerritoryID
				}
			}
		}
	}
	if !event.SavedByPillage {
		ctx.famished[candidate.army.ID] = true
		if army := ctx.armiesByID[candidate.army.ID]; army != nil && army.Size > 1 {
			army.Size--
			ctx.startArmiesByID[candidate.army.ID] = *army
			event.TroopsLost = 1
		}
	}
	ctx.events = append(ctx.events, event)
}

func (ctx *resolutionContext) hasSettlement(territoryID models.TerritoryID) bool {
	infrastructure := ctx.infrastructureAt(territoryID)
	return infrastructure != nil && (infrastructure.Type == models.InfraTypeCastle || infrastructure.Type == models.InfraTypeVillage)
}

func (ctx *resolutionContext) isControlledDepot(territoryID models.TerritoryID, ownerID models.PlayerID) bool {
	state := ctx.state.TerritoryStates[territoryID]
	infrastructure := ctx.infrastructureAt(territoryID)
	return state.OwnerID != nil && *state.OwnerID == ownerID && infrastructure != nil && infrastructure.Type == models.InfraTypeSupplyDepot
}

func (ctx *resolutionContext) infrastructureAt(territoryID models.TerritoryID) *models.Infrastructure {
	state := ctx.state.TerritoryStates[territoryID]
	if len(state.Infrastructures) == 0 {
		return nil
	}
	return ctx.infrastructuresByID[state.Infrastructures[0]]
}

func startArmiesForPlayer(ctx *resolutionContext, ownerID models.PlayerID) []models.Army {
	armies := make([]models.Army, 0)
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		army := ctx.startArmyAt(territoryID)
		if army != nil && army.OwnerID == ownerID {
			armies = append(armies, *army)
		}
	}
	return armies
}

func sortedPlayerIDs(players []models.Player) []models.PlayerID {
	ids := make([]models.PlayerID, 0, len(players))
	for _, player := range players {
		ids = append(ids, player.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedStateTerritoryIDs(ctx *resolutionContext) []models.TerritoryID {
	ids := make([]models.TerritoryID, 0, len(ctx.state.Territories))
	for _, territory := range ctx.state.Territories {
		ids = append(ids, territory.ID)
	}
	sortTerritoryIDs(ids)
	return ids
}

func cloneRations(source map[models.TerritoryID]int) map[models.TerritoryID]int {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[models.TerritoryID]int, len(source))
	for territoryID, rations := range source {
		clone[territoryID] = rations
	}
	return clone
}
