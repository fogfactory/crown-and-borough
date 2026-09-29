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

// rationProductionParts breaks the local ration production of one territory
// into its terrain and good harvest components, plus the terrain rations the
// bad harvest calamity suppresses.
type rationProductionParts struct {
	terrain    int
	bonus      int
	suppressed int
}

func (parts rationProductionParts) total() int {
	return parts.terrain + parts.bonus
}

// sourceProductionParts breaks the stockable production of one source into its
// base, mill, and regional bonus components.
type sourceProductionParts struct {
	base       int
	mill       int
	bonus      int
	suppressed int
}

func (parts sourceProductionParts) total() int {
	return parts.base + parts.mill + parts.bonus
}

// consumptionDetail tracks what one player army demanded and received during
// the supply phase, for the consumption section of the turn report.
type consumptionDetail struct {
	ownerID               models.PlayerID
	territoryID           models.TerritoryID
	sourceID              models.TerritoryID
	size                  int
	demand                int
	receivedLocal         int
	receivedTransfer      int
	famined               bool
	troopsLost            int
	savedByPillage        bool
	pillageInfrastructure models.InfraType
	resourceCredit        int
	creditTerritory       models.TerritoryID
}

// resolveSupply calculates the complete end-of-turn ravitaillement phase,
// after every order has moved an army or changed control this turn (#208): it
// reads and feeds armies where they actually stand once combat, movement and
// control are settled, not where they started the turn. A famine here never
// weakens the army in this same turn's own combat (already resolved earlier);
// it only sets models.Army.Starving for the penalty the following turn reads
// (see ctx.famished's seeding in newResolutionContext).
func resolveSupply(ctx *resolutionContext) {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		ctx.supplyStockBefore[territoryID] = ctx.state.TerritoryStates[territoryID].Resources
	}
	// Starving is recomputed in full every turn: clear it here so an army
	// that meets its demand this turn is no longer flagged, then resolveFamine
	// (below) sets it back on whichever one fails to.
	for index := range ctx.state.Armies {
		ctx.state.Armies[index].Starving = false
	}
	resolveTerritoryIncome(ctx)
	receivedRations := resolveRations(ctx)
	produceNeutralStocks(ctx)
	// Mill production is captured now, before any famine auto-pillage can
	// remove the infrastructure that just produced it (see #195).
	millProductions := computeMillProduction(ctx)
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
		for _, source := range sources {
			ctx.supplyStockConsumed[source.territoryID] += source.stockConsumed
		}
	}

	for _, source := range allSources {
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeSupply,
			Phase:         6,
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
	resolveNeutralFamines(ctx)
	updateSupplyEventStocks(ctx)
	ctx.emitProductionEvents()
	ctx.emitConsumptionEvents()
	ctx.emitMillProductionEvents(millProductions)
}

// resolveNeutralFamines applies the neutral starvation rule: neutral armies
// never lose strength to a famine, but lose one troop when the local
// production of their territory cannot feed them.
func resolveNeutralFamines(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.armiesByID) {
		army := ctx.armiesByID[armyID]
		if army.OwnerID != models.NeutralPlayerID || army.Size <= 1 {
			continue
		}
		local := rationProduction(ctx, army.TerritoryID)
		if local >= armyCost(army.Size, ctx.balance.CostBase) {
			continue
		}
		sizeBefore := army.Size
		army.Size--
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeFamine,
			Phase:       6,
			ArmyID:      army.ID,
			OwnerID:     models.NeutralPlayerID,
			RegionSeed:  regionForTerritory(ctx, army.TerritoryID),
			TerritoryID: army.TerritoryID,
			Troops:      sizeBefore,
			TroopsLost:  1,
			Season:      ctx.state.Season,
			Year:        ctx.state.Year(),
		})
	}
}

// produceNeutralStocks credits the local production of every unclaimed
// territory that carries a village: its base village_income, plus any mill
// routed to it, per neutralVillageProductionBreakdown. A neutral mill never
// produces on its own any more: unlike a village, a mill outside every fief
// and capital is inert while no army stands on it (#215, see millActive), so
// an unclaimed, self-supplied mill (no eligible adjacent village to route to
// instead, see #195) simply produces nothing until an army reclaims it.
func produceNeutralStocks(ctx *resolutionContext) {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID != nil {
			continue
		}
		if !ctx.hasInfrastructure(territoryID, models.InfraTypeVillage) {
			continue
		}
		parts := neutralVillageProductionBreakdown(ctx, territoryID)
		ctx.supplySources[territoryID] = parts
		production := parts.total()
		state.Resources += production
		ctx.state.TerritoryStates[territoryID] = state
		ctx.events = append(ctx.events, Event{
			Type:       EventTypeSupply,
			Phase:      6,
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
	received := make(map[models.ArmyID]int, len(ctx.armiesByID))
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		army := ctx.currentArmyAt(territoryID)
		if army == nil {
			continue
		}
		parts := rationProductionBreakdown(ctx, territoryID)
		ctx.supplyRations[territoryID] = parts
		distribution := distributeRations(parts.total(), []models.Army{*army}, ctx.balance.CostBase)
		received[army.ID] = distribution[army.ID]
		if army.OwnerID == models.NeutralPlayerID {
			continue
		}
		detail := ctx.consumptionDetailFor(*army)
		detail.size = army.Size
		detail.ownerID = army.OwnerID
		detail.territoryID = army.TerritoryID
		detail.demand = armyCost(army.Size, ctx.balance.CostBase)
		detail.receivedLocal = received[army.ID]
	}
	return received
}

// rationProductionBreakdown splits the local ration production of a territory
// into terrain and regional bonus parts. The bad harvest calamity suppresses
// all terrain rations of its region; the good harvest bonus doubles them.
func rationProductionBreakdown(ctx *resolutionContext, territoryID models.TerritoryID) rationProductionParts {
	territory := ctx.territoriesByID[territoryID]
	if territory == nil {
		return rationProductionParts{}
	}
	terrain := ctx.balance.RationTerrain[territory.Terrain]
	regionSeed := regionForTerritory(ctx, territoryID)
	switch {
	case ctx.famineRegions[regionSeed]:
		return rationProductionParts{suppressed: terrain}
	case ctx.goodHarvestRegions[regionSeed]:
		return rationProductionParts{terrain: terrain, bonus: terrain}
	}
	return rationProductionParts{terrain: terrain}
}

func rationProduction(ctx *resolutionContext, territoryID models.TerritoryID) int {
	return rationProductionBreakdown(ctx, territoryID).total()
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

// distributeRations assigns local production to the largest armies first,
// up to each army's demand. Equal sizes are resolved by the territory trigram;
// a same-territory tie is invalid game data and therefore intentionally
// preserves the input order.
func distributeRations(rations int, armies []models.Army, costBase int) map[models.ArmyID]int {
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
		granted := min(rations, armyCost(army.Size, costBase))
		received[army.ID] = granted
		rations -= granted
	}
	return received
}

func terrainRationProduction(ctx *resolutionContext, territoryID models.TerritoryID) int {
	territory := ctx.territoriesByID[territoryID]
	if territory == nil {
		return 0
	}
	return ctx.balance.RationTerrain[territory.Terrain]
}

func controlledSupplySources(ctx *resolutionContext, ownerID models.PlayerID) []*supplySource {
	sources := make([]*supplySource, 0)
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID == nil || *state.OwnerID != ownerID {
			continue
		}
		if ctx.occupiedAgainstController(territoryID, ctx.currentArmyAt(territoryID)) {
			// A controlled cell occupied by another player's (or a revolt's)
			// army is no longer a supply source for its controller: neither
			// side may draw from it (titres.md, economie.md#portée-de-ravitaillement).
			continue
		}
		isSettlement := ctx.hasSettlement(territoryID)
		selfSuppliedMill := !isSettlement && ctx.isSelfSuppliedMill(territoryID)
		if !isSettlement && !selfSuppliedMill && state.Resources == 0 {
			continue
		}
		production := 0
		switch {
		case isSettlement:
			parts := sourceProductionBreakdown(ctx, territoryID)
			ctx.supplySources[territoryID] = parts
			production = parts.total()
		case selfSuppliedMill:
			parts := millSelfProductionBreakdown(ctx, territoryID)
			ctx.supplySources[territoryID] = parts
			production = parts.total()
		}
		sources = append(sources, &supplySource{
			territoryID: territoryID,
			ownerID:     ownerID,
			production:  production,
			rations:     make(map[models.TerritoryID]int),
			reachable:   supplyNetwork(ctx, territoryID, ownerID),
		})
	}
	return sources
}

// sourceProductionBreakdown splits the stockable production of a controlled
// source into its mill and regional weather bonus parts. Base production was
// replaced by territory income (see income.go), credited directly to its
// destination outside this ledger. A neighboring mill only contributes here
// when territoryID is its single designated recipient (see #195's
// millRecipient): a mill no longer credits every adjacent castle or village.
// An inert neighboring mill (see millActive, #215) contributes nothing.
func sourceProductionBreakdown(ctx *resolutionContext, territoryID models.TerritoryID) sourceProductionParts {
	parts := sourceProductionParts{}
	for _, neighborID := range ctx.sortedNeighbors(territoryID) {
		infrastructure := ctx.infrastructureAt(neighborID)
		if infrastructure == nil || infrastructure.Type != models.InfraTypeMill {
			continue
		}
		if !ctx.millActive(neighborID) {
			continue
		}
		if millRecipient(ctx, neighborID) != territoryID {
			continue
		}
		production, bonus, suppressed := millWeatherProduction(ctx, neighborID, infrastructure.Level)
		parts.mill += production
		parts.bonus += bonus
		parts.suppressed += suppressed
	}
	return parts
}

// neutralVillageProductionBreakdown splits the stockable production of an
// unclaimed village into its base, mill, and regional bonus parts: unlike a
// controlled source, a neutral village keeps producing locally into its own
// stock (village_income as its base), subject to the same harvest and
// weather rules as any other source. A neighboring mill only contributes
// here when this village is its single designated recipient (see #195), and
// only a neutral mill can route to a neutral village (see sameController); an
// inert one (see millActive, #215) contributes nothing regardless.
func neutralVillageProductionBreakdown(ctx *resolutionContext, territoryID models.TerritoryID) sourceProductionParts {
	parts := harvestAdjustedParts(ctx, territoryID, ctx.balance.VillageIncome)
	for _, neighborID := range ctx.sortedNeighbors(territoryID) {
		infrastructure := ctx.infrastructureAt(neighborID)
		if infrastructure == nil || infrastructure.Type != models.InfraTypeMill {
			continue
		}
		if !ctx.millActive(neighborID) {
			continue
		}
		if millRecipient(ctx, neighborID) != territoryID {
			continue
		}
		production, bonus, suppressed := millWeatherProduction(ctx, neighborID, infrastructure.Level)
		parts.mill += production
		parts.bonus += bonus
		parts.suppressed += suppressed
	}
	return parts
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
			if army := ctx.currentArmyAt(neighborID); army != nil && army.OwnerID != ownerID {
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

// transferNetwork is the supply graph used by a resource transfer. It has the
// same range and depot rules as ordinary supply, but permits the requested
// destination to be occupied by an enemy army. Enemy armies on intermediate
// territories still block the route, including an army belonging to the
// recipient when it is not the final destination.
func transferNetwork(ctx *resolutionContext, sourceID models.TerritoryID, ownerID models.PlayerID, targetID models.TerritoryID) map[models.TerritoryID]int {
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
			if neighborID != targetID {
				if army := ctx.currentArmyAt(neighborID); army != nil && army.OwnerID != ownerID {
					continue
				}
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
	for _, army := range liveArmiesForPlayer(ctx, ownerID) {
		detail := ctx.consumptionDetailFor(army)
		detail.size = army.Size
		detail.ownerID = army.OwnerID
		detail.territoryID = army.TerritoryID
		if detail.demand == 0 {
			detail.demand = armyCost(army.Size, ctx.balance.CostBase)
		}
		demand := armyCost(army.Size, ctx.balance.CostBase) - receivedRations[army.ID]
		if demand == 0 {
			continue
		}
		source, distance := closestSupplySource(ctx, army.TerritoryID, sources)
		if source == nil {
			direct = append(direct, famineCandidate{army: army, demand: demand})
			continue
		}
		detail.receivedTransfer = demand
		detail.sourceID = source.territoryID
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
		for _, assignment := range assignments {
			ctx.recordPoolRations(assignment.source.territoryID, assignment.army.TerritoryID, assignment.demand)
		}
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
	famined := make(map[models.ArmyID]bool)
	for _, assignment := range assignments {
		if delta == 0 {
			break
		}
		if detail := ctx.supplyConsumption[assignment.army.ID]; detail != nil {
			detail.receivedTransfer = 0
		}
		famined[assignment.army.ID] = true
		candidates = append(candidates, famineCandidate{
			army:     assignment.army,
			demand:   assignment.demand,
			sourceID: assignment.source.territoryID,
			distance: assignment.distance,
		})
		delta -= assignment.demand
	}
	for _, assignment := range assignments {
		if famined[assignment.army.ID] {
			continue
		}
		ctx.recordPoolRations(assignment.source.territoryID, assignment.army.TerritoryID, assignment.demand)
	}
	return candidates
}

// recordPoolRations traces the rations one source dispatched to the armies of
// a dependent territory, for the production ledger of the turn report.
func (ctx *resolutionContext) recordPoolRations(sourceID, territoryID models.TerritoryID, amount int) {
	if amount == 0 {
		return
	}
	if ctx.poolRations[sourceID] == nil {
		ctx.poolRations[sourceID] = make(map[models.TerritoryID]int)
	}
	ctx.poolRations[sourceID][territoryID] += amount
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
		Phase:       6,
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
		// Starving carries the penalty into next turn's own combat, support
		// and noble-bonus checks (ctx.famished's seeding in
		// newResolutionContext): this turn's own combat already resolved
		// earlier and is never reopened by a famine discovered only now
		// (#208).
		if army := ctx.armiesByID[candidate.army.ID]; army != nil {
			army.Starving = true
			if army.Size > 1 {
				army.Size--
				event.TroopsLost = 1
			}
		}
	}
	if detail := ctx.supplyConsumption[candidate.army.ID]; detail != nil {
		detail.famined = true
		detail.troopsLost = event.TroopsLost
		detail.savedByPillage = event.SavedByPillage
		detail.pillageInfrastructure = event.InfrastructureType
		detail.resourceCredit = event.ResourceCredit
		detail.creditTerritory = event.CreditTerritoryID
	}
	ctx.events = append(ctx.events, event)
}

func (ctx *resolutionContext) consumptionDetailFor(army models.Army) *consumptionDetail {
	detail, exists := ctx.supplyConsumption[army.ID]
	if !exists {
		detail = &consumptionDetail{}
		ctx.supplyConsumption[army.ID] = detail
	}
	return detail
}

// emitProductionEvents reports one production ledger per territory involved in
// the supply phase: local ration production, stockable source production, and
// the resulting stock movement.
func (ctx *resolutionContext) emitProductionEvents() {
	involved := make(map[models.TerritoryID]bool)
	for territoryID := range ctx.supplySources {
		involved[territoryID] = true
	}
	for territoryID := range ctx.supplyRations {
		involved[territoryID] = true
	}
	seeds := make([]models.TerritoryID, 0, len(involved))
	for territoryID := range involved {
		seeds = append(seeds, territoryID)
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	for _, territoryID := range seeds {
		rations := ctx.supplyRations[territoryID]
		source := ctx.supplySources[territoryID]
		state := ctx.state.TerritoryStates[territoryID]
		var ownerID models.PlayerID
		if state.OwnerID != nil {
			ownerID = *state.OwnerID
		}
		var sentRations map[models.TerritoryID]int
		if dispatch := ctx.poolRations[territoryID]; len(dispatch) > 0 {
			sentRations = make(map[models.TerritoryID]int, len(dispatch))
			for destination, amount := range dispatch {
				sentRations[destination] = amount
			}
		}
		ctx.events = append(ctx.events, Event{
			Type:                 EventTypeProduction,
			Phase:                6,
			TerritoryID:          territoryID,
			RegionSeed:           regionForTerritory(ctx, territoryID),
			OwnerID:              ownerID,
			TerrainRations:       rations.terrain,
			BonusRations:         rations.bonus,
			SuppressedRations:    rations.suppressed,
			BaseProduction:       source.base,
			MillProduction:       source.mill,
			BonusProduction:      source.bonus,
			SuppressedProduction: source.suppressed,
			Production:           rations.total() + source.total(),
			SentRations:          sentRations,
			StockBefore:          ctx.supplyStockBefore[territoryID],
			StockConsumed:        ctx.supplyStockConsumed[territoryID],
			StockAfter:           state.Resources,
			Season:               ctx.state.Season,
			Year:                 ctx.state.Year(),
		})
	}
}

// emitMillProductionEvents reports one line per mill: its level, single
// destination, harvest-and-weather-adjusted production, and any production
// lost to bad weather (see #195). mills is captured before famine's
// auto-pillage can remove the infrastructure that already produced it.
func (ctx *resolutionContext) emitMillProductionEvents(mills []millProduction) {
	for _, mill := range mills {
		var ownerID models.PlayerID
		if mill.ownerID != nil {
			ownerID = *mill.ownerID
		}
		ctx.events = append(ctx.events, Event{
			Type:                 EventTypeMillProduction,
			Phase:                6,
			TerritoryID:          mill.millID,
			DestinationID:        mill.destinationID,
			OwnerID:              ownerID,
			InfrastructureID:     mill.infrastructureID,
			InfrastructureType:   models.InfraTypeMill,
			Level:                mill.level,
			Production:           mill.production,
			BonusProduction:      mill.bonus,
			SuppressedProduction: mill.suppressed,
			Season:               ctx.state.Season,
			Year:                 ctx.state.Year(),
		})
	}
}

// emitConsumptionEvents reports one consumption line per player army, with the
// demand split between local rations and pooled source coverage, plus the
// famine effects when the demand was not met.
func (ctx *resolutionContext) emitConsumptionEvents() {
	armyIDs := make([]models.ArmyID, 0, len(ctx.supplyConsumption))
	for armyID := range ctx.supplyConsumption {
		armyIDs = append(armyIDs, armyID)
	}
	sortArmyIDs(armyIDs)
	for _, armyID := range armyIDs {
		detail := ctx.supplyConsumption[armyID]
		ctx.events = append(ctx.events, Event{
			Type:               EventTypeConsumption,
			Phase:              6,
			ArmyID:             armyID,
			OwnerID:            detail.ownerID,
			TerritoryID:        detail.territoryID,
			SourceID:           detail.sourceID,
			Troops:             detail.size,
			Demand:             detail.demand,
			ReceivedLocal:      detail.receivedLocal,
			ReceivedTransfer:   detail.receivedTransfer,
			TroopsLost:         detail.troopsLost,
			SavedByPillage:     detail.savedByPillage,
			InfrastructureType: detail.pillageInfrastructure,
			ResourceCredit:     detail.resourceCredit,
			CreditTerritoryID:  detail.creditTerritory,
			Season:             ctx.state.Season,
			Year:               ctx.state.Year(),
		})
	}
}

func (ctx *resolutionContext) hasSettlement(territoryID models.TerritoryID) bool {
	infrastructure := ctx.infrastructureAt(territoryID)
	return infrastructure != nil && (infrastructure.Type == models.InfraTypeCastle || infrastructure.Type == models.InfraTypeVillage)
}

// isControlledDepot reports whether territoryID carries a supply depot
// controlled by ownerID and usable by it: a depot on a cell occupied against
// its controller extends nobody's range, neither the controller's nor the
// occupant's (economie.md#portée-de-ravitaillement).
func (ctx *resolutionContext) isControlledDepot(territoryID models.TerritoryID, ownerID models.PlayerID) bool {
	state := ctx.state.TerritoryStates[territoryID]
	if state.OwnerID == nil || *state.OwnerID != ownerID {
		return false
	}
	infrastructure := ctx.infrastructureAt(territoryID)
	if infrastructure == nil || infrastructure.Type != models.InfraTypeSupplyDepot {
		return false
	}
	return !ctx.occupiedAgainstController(territoryID, ctx.currentArmyAt(territoryID))
}

func (ctx *resolutionContext) infrastructureAt(territoryID models.TerritoryID) *models.Infrastructure {
	state := ctx.state.TerritoryStates[territoryID]
	if state.Infrastructures == nil {
		return nil
	}
	return ctx.infrastructuresByID[*state.Infrastructures]
}

// liveArmiesForPlayer lists ownerID's armies where they actually stand now:
// resolveSupply resolves at the end of the turn, on post-combat,
// post-movement positions (#208), unlike the frozen start-of-turn snapshot
// ctx.startArmiesByID keeps for the earlier resolution phases.
func liveArmiesForPlayer(ctx *resolutionContext, ownerID models.PlayerID) []models.Army {
	armies := make([]models.Army, 0)
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		army := ctx.currentArmyAt(territoryID)
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
