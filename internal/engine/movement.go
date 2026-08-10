package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type disperseResolution struct {
	intent    *disperseIntent
	resolved  []bool
	remaining int
	invalid   bool
}

type joinResolution struct {
	targetID models.TerritoryID
	hostID   models.ArmyID
	fuse     bool
	pair     bool
}

type retreatPlan struct {
	dislodged     *dislodgedArmy
	destinationID models.TerritoryID
	destroyReason string
}

func executeMovementsAndRetreats(ctx *resolutionContext) error {
	resolveDispersions(ctx)
	resolveJoins(ctx)
	resolveVacatedDisperseDestinations(ctx)
	finalizeDispersions(ctx)
	riders := startingRiders(ctx)
	for _, armyID := range sortedArmyMap(ctx.dislodged) {
		displaced := ctx.dislodged[armyID]
		displaced.nobleIDs = append([]models.NobleID(nil), riders[armyID]...)
	}
	if err := executeNormalMovements(ctx, riders); err != nil {
		return err
	}
	if err := ctx.rebuildOccupancy(); err != nil {
		return err
	}
	executeLocalOrders(ctx)
	if err := executeRetreats(ctx); err != nil {
		return err
	}
	return nil
}

func startingRiders(ctx *resolutionContext) map[models.ArmyID][]models.NobleID {
	riders := make(map[models.ArmyID][]models.NobleID, len(ctx.startArmiesByID))
	for _, armyID := range sortedArmyMap(ctx.startArmiesByID) {
		army := ctx.startArmiesByID[armyID]
		riders[armyID] = ctx.noblesAt(army.TerritoryID)
	}
	return riders
}

func resolveDispersions(ctx *resolutionContext) {
	invalid := make(map[models.ArmyID]string)
	for {
		results, candidates, claims := ctx.disperseCandidates(invalid)
		for _, armyID := range sortedArmyMap(results) {
			result := results[armyID]
			available := ctx.startArmiesByID[result.intent.armyID].Size
			for index, targetID := range result.intent.targets {
				if !candidates[armyID][index] || available < 1 {
					continue
				}
				if targetID != result.intent.source && len(claims[targetID]) > 1 {
					continue
				}
				result.resolved[index] = true
				available--
			}
		}

		addedInvalid := false
		for _, armyID := range sortedArmyMap(results) {
			if reason := ctx.disperseInvalidReason(ctx.records[armyID], results[armyID]); reason != "" {
				invalid[armyID] = reason
				addedInvalid = true
			}
		}
		if addedInvalid {
			continue
		}

		ctx.disperseResults = results
		for _, armyID := range sortedArmyMap(ctx.disperseResults) {
			refreshDisperseOutcome(ctx, ctx.records[armyID], ctx.disperseResults[armyID])
		}
		for _, armyID := range sortedArmyMap(invalid) {
			ctx.records[armyID].invalidate(invalid[armyID])
		}
		return
	}
}

func (ctx *resolutionContext) disperseCandidates(invalid map[models.ArmyID]string) (map[models.ArmyID]*disperseResolution, map[models.ArmyID][]bool, map[models.TerritoryID]map[models.ArmyID]bool) {
	results := make(map[models.ArmyID]*disperseResolution, len(ctx.disperses))
	candidates := make(map[models.ArmyID][]bool, len(ctx.disperses))
	claims := make(map[models.TerritoryID]map[models.ArmyID]bool)
	for _, armyID := range sortedArmyMap(ctx.disperses) {
		record := ctx.records[armyID]
		if record == nil || record.outcome != "" || invalid[armyID] != "" {
			continue
		}
		intent := ctx.disperses[armyID]
		result := &disperseResolution{
			intent:   intent,
			resolved: make([]bool, len(intent.targets)),
		}
		candidate := make([]bool, len(intent.targets))
		for index, targetID := range intent.targets {
			candidate[index] = !ctx.attackedTerritories[targetID]
			if candidate[index] && targetID != intent.source {
				if occupant := ctx.startArmyAt(targetID); occupant != nil && !ctx.vacatesForDisperse(occupant.ID) {
					candidate[index] = false
				}
			}
			if candidate[index] && targetID != intent.source {
				if claims[targetID] == nil {
					claims[targetID] = make(map[models.ArmyID]bool)
				}
				claims[targetID][armyID] = true
			}
		}
		results[armyID] = result
		candidates[armyID] = candidate
	}
	return results, candidates, claims
}

func finalizeDispersions(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[armyID]
		record := ctx.records[armyID]
		if result.invalid || record.outcome == OutcomeInvalid {
			continue
		}
		if reason := ctx.disperseInvalidReason(record, result); reason != "" {
			record.invalidate(reason)
			record.partialD = false
			result.invalid = true
			continue
		}
		refreshDisperseOutcome(ctx, record, result)
	}
}

func (ctx *resolutionContext) vacatesForDisperse(armyID models.ArmyID) bool {
	attack := ctx.attacks[armyID]
	record := ctx.records[armyID]
	return attack != nil && record != nil && record.outcome == OutcomeSuccess
}

func resolveJoins(ctx *resolutionContext) {
	joinsByTarget := make(map[models.TerritoryID][]models.ArmyID)
	for _, armyID := range sortedArmyMap(ctx.joins) {
		record := ctx.records[armyID]
		if record == nil || record.outcome != "" {
			continue
		}
		join := ctx.joins[armyID]
		joinsByTarget[join.target] = append(joinsByTarget[join.target], armyID)
	}
	for _, targetID := range sortedTerritoryMap(joinsByTarget) {
		if ctx.attackedTerritories[targetID] || (!ctx.hasDisperseArrival(targetID) && !ctx.hasActiveDisperseAt(targetID)) {
			continue
		}
		for _, joiningID := range joinsByTarget[targetID] {
			ctx.records[joiningID].fail("join_convergence")
		}
	}
	for _, targetID := range sortedTerritoryMap(joinsByTarget) {
		members := joinsByTarget[targetID]
		sortArmyIDs(members)
		if ctx.attackedTerritories[targetID] {
			ctx.resolveJoinAtAttackTarget(targetID, members)
		}
	}
	for _, targetID := range sortedTerritoryMap(joinsByTarget) {
		members := pendingJoinMembers(ctx, joinsByTarget[targetID])
		if ctx.attackedTerritories[targetID] || len(members) < 2 {
			continue
		}
		if len(members) > 2 {
			for _, joiningID := range members {
				ctx.records[joiningID].fail("join_convergence")
			}
			continue
		}
		if ctx.startArmiesByID[members[0]].OwnerID != ctx.startArmiesByID[members[1]].OwnerID {
			for _, joiningID := range members {
				ctx.records[joiningID].fail("join_enemy_convergence")
			}
		}
	}
	for {
		progressed := false
		for _, targetID := range sortedTerritoryMap(joinsByTarget) {
			if ctx.attackedTerritories[targetID] {
				continue
			}
			members := pendingJoinMembers(ctx, joinsByTarget[targetID])
			switch len(members) {
			case 1:
				progressed = ctx.resolveSingleJoin(targetID, members[0], false) || progressed
			case 2:
				progressed = ctx.resolveJoinPairOrConvergence(targetID, members, false) || progressed
			}
		}
		if !progressed {
			break
		}
	}
	for _, targetID := range sortedTerritoryMap(joinsByTarget) {
		if ctx.attackedTerritories[targetID] {
			continue
		}
		members := pendingJoinMembers(ctx, joinsByTarget[targetID])
		switch len(members) {
		case 1:
			ctx.resolveSingleJoin(targetID, members[0], true)
		case 2:
			ctx.resolveJoinPairOrConvergence(targetID, members, true)
		}
	}
}

func pendingJoinMembers(ctx *resolutionContext, candidates []models.ArmyID) []models.ArmyID {
	members := make([]models.ArmyID, 0, len(candidates))
	for _, armyID := range candidates {
		if ctx.records[armyID].outcome == "" {
			members = append(members, armyID)
		}
	}
	sortArmyIDs(members)
	return members
}

func resolveVacatedDisperseDestinations(ctx *resolutionContext) {
	claims := disperseTargetClaimants(ctx)
	for {
		changed := false
		for _, armyID := range sortedArmyMap(ctx.disperseResults) {
			result := ctx.disperseResults[armyID]
			if result.invalid {
				continue
			}
			available := result.remaining
			for index, targetID := range result.intent.targets {
				if result.resolved[index] || targetID == result.intent.source || len(claims[targetID]) != 1 || ctx.attackedTerritories[targetID] || ctx.hasPendingJoinTarget(targetID) {
					continue
				}
				occupant := ctx.startArmyAt(targetID)
				if occupant == nil {
					continue
				}
				if !ctx.disperseOccupantVacates(occupant.ID, targetID) || available < 1 {
					continue
				}
				if !ctx.canResolveVacatedDisperseTarget(ctx.records[armyID], result, index) {
					continue
				}
				result.resolved[index] = true
				available--
				changed = true
			}
			result.remaining = available
		}
		if !changed {
			return
		}
	}
}

func (ctx *resolutionContext) canResolveVacatedDisperseTarget(record *orderRecord, result *disperseResolution, index int) bool {
	result.resolved[index] = true
	invalid := ctx.disperseInvalidReason(record, result) != ""
	result.resolved[index] = false
	return !invalid
}

func disperseTargetClaimants(ctx *resolutionContext) map[models.TerritoryID]map[models.ArmyID]bool {
	claims := make(map[models.TerritoryID]map[models.ArmyID]bool)
	for _, armyID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[armyID]
		if result.invalid {
			continue
		}
		intent := result.intent
		for _, targetID := range intent.targets {
			if targetID == intent.source {
				continue
			}
			if claims[targetID] == nil {
				claims[targetID] = make(map[models.ArmyID]bool)
			}
			claims[targetID][armyID] = true
		}
	}
	return claims
}

func (ctx *resolutionContext) disperseOccupantVacates(armyID models.ArmyID, territoryID models.TerritoryID) bool {
	if _, moved := ctx.joinResults[armyID]; moved {
		if record := ctx.records[armyID]; record != nil && record.outcome == OutcomeSuccess {
			return true
		}
	}
	return ctx.disperseVacatesSource(armyID, territoryID)
}

func (ctx *resolutionContext) disperseVacatesSource(armyID models.ArmyID, territoryID models.TerritoryID) bool {
	for _, carrierID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[carrierID]
		if result.invalid {
			continue
		}
		if result.intent.armyID != armyID || result.intent.source != territoryID {
			continue
		}
		if result.remaining != 0 {
			return false
		}
		for index, targetID := range result.intent.targets {
			if !result.resolved[index] || targetID == territoryID {
				return false
			}
		}
		return true
	}
	return false
}

func (ctx *resolutionContext) hasPendingJoinTarget(targetID models.TerritoryID) bool {
	for _, armyID := range sortedArmyMap(ctx.joins) {
		join := ctx.joins[armyID]
		record := ctx.records[armyID]
		if join.target == targetID && record.outcome != OutcomeFailure && record.outcome != OutcomeInvalid {
			return true
		}
	}
	return false
}

func refreshDisperseOutcome(ctx *resolutionContext, record *orderRecord, result *disperseResolution) {
	result.remaining = ctx.disperseRemaining(result)
	complete := !hasUnresolvedDisperse(result)
	record.partialD = !complete
	if complete {
		record.outcome = OutcomeSuccess
		record.reason = "disperse_complete"
	} else {
		record.outcome = OutcomeFailure
		record.reason = "disperse_partial"
	}
}

func (ctx *resolutionContext) disperseInvalidReason(record *orderRecord, result *disperseResolution) string {
	army := ctx.startArmiesByID[result.intent.armyID]
	if record.order.Liaison == models.LiaisonModeLoop {
		if len(result.intent.targets) > army.Size {
			return "disperse_no_residual"
		}
	}
	if ctx.disperseRemaining(result) == 0 && hasUnresolvedDisperse(result) && record.order.Liaison == models.LiaisonModeLoop {
		return "disperse_no_residual"
	}
	if ctx.disperseOriginStrength(result) == 0 && hasOrphanedDisperseNobles(result) {
		return "disperse_noble_left_behind"
	}
	return ""
}

func (ctx *resolutionContext) disperseRemaining(result *disperseResolution) int {
	return ctx.startArmiesByID[result.intent.armyID].Size - countResolvedDisperse(result)
}

func countResolvedDisperse(result *disperseResolution) int {
	count := 0
	for _, resolved := range result.resolved {
		if resolved {
			count++
		}
	}
	return count
}

func hasUnresolvedDisperse(result *disperseResolution) bool {
	for _, resolved := range result.resolved {
		if !resolved {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) disperseOriginStrength(result *disperseResolution) int {
	originStrength := ctx.disperseRemaining(result)
	for index, resolved := range result.resolved {
		if resolved && result.intent.targets[index] == result.intent.source {
			originStrength++
		}
	}
	return originStrength
}

func hasOrphanedDisperseNobles(result *disperseResolution) bool {
	assigned := make(map[models.NobleID]bool)
	resolvedTargets := make(map[models.TerritoryID]bool)
	for index, resolved := range result.resolved {
		if resolved {
			resolvedTargets[result.intent.targets[index]] = true
		}
	}
	for targetID, nobleIDs := range result.intent.assignments {
		for _, nobleID := range nobleIDs {
			assigned[nobleID] = true
			if !resolvedTargets[targetID] {
				return true
			}
		}
	}
	for _, nobleID := range result.intent.nobles {
		if !assigned[nobleID] {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) resolveJoinAtAttackTarget(targetID models.TerritoryID, members []models.ArmyID) {
	if len(members) == 1 {
		joiningID := members[0]
		joiningArmy := ctx.startArmiesByID[joiningID]
		attacks := ctx.attacksTargeting(targetID)
		result := ctx.contest.results[targetID]
		winner := ctx.startArmiesByID[result.winnerID]
		if len(attacks) == 1 && result.winnerID != "" && ctx.startArmyAt(targetID) == nil && winner.OwnerID == joiningArmy.OwnerID {
			ctx.joinResults[joiningID] = &joinResolution{targetID: targetID, hostID: winner.ID, fuse: true}
			record := ctx.records[joiningID]
			record.outcome = OutcomeSuccess
			record.reason = "join_attack_arrival"
			return
		}
	}
	for _, joiningID := range members {
		ctx.records[joiningID].fail("attacked_destination")
	}
}

func (ctx *resolutionContext) resolveSingleJoin(targetID models.TerritoryID, joiningID models.ArmyID, allowUnresolvedDeparture bool) bool {
	if ctx.hasDisperseArrival(targetID) || ctx.hasActiveDisperseAt(targetID) {
		ctx.records[joiningID].fail("join_convergence")
		return true
	}
	if !allowUnresolvedDeparture && ctx.hasUnresolvedJoinAt(targetID) {
		return false
	}
	joining := ctx.startArmiesByID[joiningID]
	if host := ctx.stationaryJoinHost(targetID); host != nil {
		if host.OwnerID != joining.OwnerID {
			ctx.records[joiningID].fail("enemy_destination")
			return true
		}
		ctx.joinResults[joiningID] = &joinResolution{targetID: targetID, hostID: host.ID, fuse: true}
		ctx.records[joiningID].outcome = OutcomeSuccess
		ctx.records[joiningID].reason = "join_host"
		return true
	}
	ctx.joinResults[joiningID] = &joinResolution{targetID: targetID}
	ctx.records[joiningID].outcome = OutcomeSuccess
	ctx.records[joiningID].reason = "join_move"
	return true
}

func (ctx *resolutionContext) resolveJoinPairOrConvergence(targetID models.TerritoryID, members []models.ArmyID, allowUnresolvedDeparture bool) bool {
	if len(members) != 2 || ctx.hasDisperseArrival(targetID) || ctx.hasActiveDisperseAt(targetID) || ctx.stationaryJoinHost(targetID) != nil {
		for _, joiningID := range members {
			ctx.records[joiningID].fail("join_convergence")
		}
		return true
	}
	if !allowUnresolvedDeparture && ctx.hasUnresolvedJoinAt(targetID) {
		return false
	}
	first := ctx.startArmiesByID[members[0]]
	second := ctx.startArmiesByID[members[1]]
	if first.OwnerID != second.OwnerID {
		for _, joiningID := range members {
			ctx.records[joiningID].fail("join_enemy_convergence")
		}
		return true
	}
	hostID := members[0]
	for _, joiningID := range members {
		ctx.joinResults[joiningID] = &joinResolution{targetID: targetID, hostID: hostID, fuse: true, pair: true}
		ctx.records[joiningID].outcome = OutcomeSuccess
		ctx.records[joiningID].reason = "join_pair"
	}
	return true
}

func (ctx *resolutionContext) attacksTargeting(targetID models.TerritoryID) []models.ArmyID {
	attacks := make([]models.ArmyID, 0)
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		if ctx.attacks[armyID].target == targetID {
			attacks = append(attacks, armyID)
		}
	}
	return attacks
}

func (ctx *resolutionContext) hasDisperseArrival(targetID models.TerritoryID) bool {
	for _, armyID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[armyID]
		if result.invalid {
			continue
		}
		for index, destinationID := range result.intent.targets {
			if destinationID == targetID && result.resolved[index] && destinationID != result.intent.source {
				return true
			}
		}
	}
	return false
}

func (ctx *resolutionContext) hasActiveDisperseAt(territoryID models.TerritoryID) bool {
	army := ctx.startArmyAt(territoryID)
	if army == nil {
		return false
	}
	for _, carrierID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[carrierID]
		if result.invalid {
			continue
		}
		if result.intent.armyID == army.ID && !ctx.disperseVacatesSource(army.ID, territoryID) {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) stationaryJoinHost(territoryID models.TerritoryID) *models.Army {
	host := ctx.startArmyAt(territoryID)
	if host == nil || ctx.dislodged[host.ID] != nil {
		return nil
	}
	if _, joining := ctx.joins[host.ID]; joining {
		if record := ctx.records[host.ID]; record != nil && record.outcome != OutcomeFailure && record.outcome != OutcomeInvalid {
			return nil
		}
	}
	if attack := ctx.attacks[host.ID]; attack != nil && ctx.records[host.ID].outcome == OutcomeSuccess {
		return nil
	}
	if ctx.disperseResultForArmy(host.ID) != nil {
		if ctx.disperseVacatesSource(host.ID, territoryID) {
			return nil
		}
	}
	return host
}

func (ctx *resolutionContext) disperseResultForArmy(armyID models.ArmyID) *disperseResolution {
	for _, carrierID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[carrierID]
		if result.intent.armyID == armyID {
			return result
		}
	}
	return nil
}

func (ctx *resolutionContext) hasUnresolvedJoinAt(territoryID models.TerritoryID) bool {
	host := ctx.startArmyAt(territoryID)
	if host == nil {
		return false
	}
	if _, joining := ctx.joins[host.ID]; !joining {
		return false
	}
	record := ctx.records[host.ID]
	return record != nil && record.outcome == ""
}

func executeNormalMovements(ctx *resolutionContext, riders map[models.ArmyID][]models.NobleID) error {
	live := make(map[models.ArmyID]models.Army, len(ctx.state.Armies)-len(ctx.dislodged))
	for _, army := range ctx.state.Armies {
		if ctx.dislodged[army.ID] != nil {
			continue
		}
		live[army.ID] = army
	}
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		record := ctx.records[armyID]
		if record == nil || record.outcome != OutcomeSuccess {
			continue
		}
		attack := ctx.attacks[armyID]
		army, exists := live[armyID]
		if !exists {
			return fmtMissingArmy(armyID)
		}
		army.TerritoryID = attack.target
		live[armyID] = army
		ctx.moveNobles(riders[armyID], attack.target, armyID)
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeMovement,
			Phase:         4,
			ArmyID:        armyID,
			SourceID:      attack.source,
			DestinationID: attack.target,
			OrderType:     models.OrderTypeAttack,
			Outcome:       OutcomeSuccess,
		})
	}
	for _, armyID := range sortedArmyMap(ctx.joinResults) {
		record := ctx.records[armyID]
		if record == nil || record.outcome != OutcomeSuccess {
			continue
		}
		join := ctx.joins[armyID]
		army, exists := live[armyID]
		if !exists {
			return fmtMissingArmy(armyID)
		}
		army.TerritoryID = join.target
		live[armyID] = army
		ctx.moveNobles(riders[armyID], join.target, armyID)
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeMovement,
			Phase:         4,
			ArmyID:        armyID,
			SourceID:      join.source,
			DestinationID: join.target,
			OrderType:     models.OrderTypeJoin,
			Outcome:       OutcomeSuccess,
		})
	}
	for _, armyID := range sortedArmyMap(ctx.disperseResults) {
		result := ctx.disperseResults[armyID]
		if err := applyDisperse(ctx, live, result); err != nil {
			return err
		}
	}
	for _, joiningID := range sortedArmyMap(ctx.joinResults) {
		resolution := ctx.joinResults[joiningID]
		if !resolution.fuse || resolution.hostID == joiningID {
			continue
		}
		host, hostExists := live[resolution.hostID]
		joining, joiningExists := live[joiningID]
		if !hostExists || !joiningExists {
			return fmtMissingArmy(joiningID)
		}
		host.Size += joining.Size
		if resolution.pair {
			host.ChainID = nil
		}
		live[resolution.hostID] = host
		chainID := models.ChainID("")
		if host.ChainID != nil {
			chainID = *host.ChainID
		}
		delete(live, joiningID)
		ctx.records[joiningID].fused = true
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeFusion,
			Phase:       4,
			ArmyID:      resolution.hostID,
			OtherArmyID: joiningID,
			ArmyIDs:     []models.ArmyID{resolution.hostID, joiningID},
			ChainID:     chainID,
			TerritoryID: resolution.targetID,
			OrderType:   models.OrderTypeJoin,
			Outcome:     OutcomeSuccess,
		})
	}
	ctx.state.Armies = armiesFromMap(live)
	return nil
}

func applyDisperse(
	ctx *resolutionContext,
	live map[models.ArmyID]models.Army,
	result *disperseResolution,
) error {
	intent := result.intent
	record := ctx.records[intent.recordArmyID]
	if record == nil || (record.outcome != OutcomeSuccess && !record.partialD) {
		return nil
	}
	army, exists := live[intent.armyID]
	if !exists {
		return fmtMissingArmy(intent.armyID)
	}
	remaining := result.remaining
	resolvedCount := countResolvedDisperse(result)
	if resolvedCount == 0 && remaining != army.Size {
		return fmt.Errorf("engine: dispersion %q has inconsistent remaining strength", record.armyID)
	}
	targetCounts := make(map[models.TerritoryID]int)
	firstResolvedIndex := -1
	for index, resolved := range result.resolved {
		if !resolved {
			continue
		}
		if firstResolvedIndex == -1 {
			firstResolvedIndex = index
		}
		targetCounts[intent.targets[index]]++
	}

	originalTarget := intent.source
	if firstResolvedIndex != -1 && (firstResolvedIndex == 0 || remaining == 0) {
		originalTarget = intent.targets[firstResolvedIndex]
	}

	delete(live, army.ID)
	groupIDs := make(map[models.TerritoryID]models.ArmyID)
	groupOrder := make([]models.TerritoryID, 0, len(targetCounts)+1)
	createdArmyIDs := make([]models.ArmyID, 0, len(targetCounts))
	sourceArmyID := models.ArmyID("")
	initialized := make(map[models.TerritoryID]bool)
	ensureGroup := func(targetID models.TerritoryID) models.ArmyID {
		if initialized[targetID] {
			return groupIDs[targetID]
		}
		groupID := models.ArmyID("")
		if targetID == originalTarget {
			groupID = army.ID
		} else {
			groupID = ctx.allocateArmyID()
			createdArmyIDs = append(createdArmyIDs, groupID)
		}
		groupSize := targetCounts[targetID]
		if targetID == intent.source {
			groupSize += remaining
		}
		group := army
		group.ID = groupID
		group.TerritoryID = targetID
		group.Size = groupSize
		if groupID != army.ID {
			group.ChainID = nil
		}
		live[groupID] = group
		groupIDs[targetID] = groupID
		groupOrder = append(groupOrder, targetID)
		initialized[targetID] = true
		if targetID == intent.source {
			sourceArmyID = groupID
		}
		return groupID
	}

	if firstResolvedIndex == -1 || originalTarget == intent.source {
		ensureGroup(intent.source)
	}
	for index, targetID := range intent.targets {
		if !result.resolved[index] {
			continue
		}
		ensureGroup(targetID)
	}
	if remaining > 0 && sourceArmyID == "" {
		ensureGroup(intent.source)
	}
	for _, targetID := range groupOrder {
		ctx.moveNobles(intent.assignments[targetID], targetID, groupIDs[targetID])
	}
	for index, targetID := range intent.targets {
		resolved := result.resolved[index]
		otherArmyID := models.ArmyID("")
		if resolved {
			groupID := groupIDs[targetID]
			if groupID != army.ID {
				otherArmyID = groupID
			}
		}
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeDispersion,
			Phase:         4,
			ArmyID:        army.ID,
			OtherArmyID:   otherArmyID,
			SourceID:      intent.source,
			DestinationID: targetID,
			Resolved:      resolved,
			Outcome:       record.outcome,
		})
	}
	if record.partialD && record.order.Liaison == models.LiaisonModeLoop {
		if err := ctx.updateLoopDispersePending(record, result, sourceArmyID); err != nil {
			return err
		}
	} else if result.intent.pending {
		chain := ctx.chainsByID[record.chainID]
		if chain == nil {
			return fmt.Errorf("engine: completed pending dispersion %q has no chain %q", record.armyID, record.chainID)
		}
		chain.PendingDisperse = nil
	}
	ctx.events = append(ctx.events, Event{
		Type:              EventTypeDispersion,
		Phase:             4,
		ArmyID:            army.ID,
		ArmyIDs:           append([]models.ArmyID(nil), createdArmyIDs...),
		ChainID:           record.chainID,
		SourceID:          intent.source,
		DestinationID:     live[army.ID].TerritoryID,
		Outcome:           record.outcome,
		RemainingStrength: remaining,
		Reason:            "disperse_summary",
	})
	return nil
}

func (ctx *resolutionContext) updateLoopDispersePending(
	record *orderRecord,
	result *disperseResolution,
	residualID models.ArmyID,
) error {
	if residualID == "" {
		return fmt.Errorf("engine: partial loop dispersion %q has no residual army", record.armyID)
	}
	chain := ctx.chainsByID[record.chainID]
	if chain == nil {
		return fmt.Errorf("engine: partial loop dispersion %q has no chain %q", record.armyID, record.chainID)
	}
	pendingTargets := pendingDisperseTargets(result)
	chain.PendingDisperse = &models.PendingDisperse{
		ArmyID:           residualID,
		SourceID:         result.intent.source,
		TargetIDs:        pendingTargets,
		NobleAssignments: ctx.pendingDisperseAssignments(result, pendingTargets),
	}
	return nil
}

func pendingDisperseTargets(result *disperseResolution) []models.TerritoryID {
	targets := make([]models.TerritoryID, 0, len(result.intent.targets))
	for index, targetID := range result.intent.targets {
		if !result.resolved[index] {
			targets = append(targets, targetID)
		}
	}
	return targets
}

func (ctx *resolutionContext) pendingDisperseAssignments(result *disperseResolution, targets []models.TerritoryID) map[models.TerritoryCode][]models.NobleCode {
	assignments := make(map[models.TerritoryCode][]models.NobleCode)
	record := ctx.records[result.intent.recordArmyID]
	if record == nil {
		return assignments
	}
	pendingCodes := make(map[models.TerritoryCode]bool, len(targets))
	for _, targetID := range targets {
		territory := ctx.territoriesByID[targetID]
		if territory != nil {
			pendingCodes[models.TerritoryCode(territory.Code)] = true
		}
	}
	for destinationCode, nobleCodes := range record.order.NobleAssignments {
		if !pendingCodes[destinationCode] {
			continue
		}
		for _, nobleCode := range nobleCodes {
			if nobleCode == "*" {
				assignments[destinationCode] = append(assignments[destinationCode], nobleCode)
				continue
			}
			nobleID, exists := ctx.noblesByCode[nobleCode]
			if !exists {
				continue
			}
			noble := ctx.noblesByID[nobleID]
			if noble != nil && noble.LocationID == result.intent.source {
				assignments[destinationCode] = append(assignments[destinationCode], nobleCode)
			}
		}
	}
	return assignments
}

func armiesFromMap(armies map[models.ArmyID]models.Army) []models.Army {
	ids := sortedArmyMap(armies)
	result := make([]models.Army, 0, len(ids))
	for _, armyID := range ids {
		result = append(result, armies[armyID])
	}
	return result
}

func executeLocalOrders(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.records) {
		record := ctx.records[armyID]
		if record.outcome != "" {
			continue
		}
		army := ctx.armiesByID[armyID]
		if army == nil {
			record.fail("army_destroyed")
			continue
		}
		switch record.order.Type {
		case models.OrderTypeHold:
			record.outcome = OutcomeSuccess
			record.reason = "held"
		case models.OrderTypeSupport:
			support := ctx.supports[armyID]
			record.outcome = OutcomeSuccess
			switch {
			case support == nil || !support.applies:
				record.reason = "support_void"
			case ctx.contest.cuts[armyID]:
				record.reason = "support_cut"
			default:
				record.reason = "support_applied"
			}
		case models.OrderTypePillage:
			ctx.executePillage(record, army)
		case models.OrderTypeHostage, models.OrderTypeDungeon:
			ctx.executeNobleStatusOrder(record, army)
		default:
			record.invalidate("unresolved_order")
		}
	}
}

func (ctx *resolutionContext) executePillage(record *orderRecord, army *models.Army) {
	state := ctx.state.TerritoryStates[army.TerritoryID]
	if len(state.Infrastructures) == 0 {
		record.invalidate("no_infrastructure")
		return
	}
	infrastructureID := state.Infrastructures[0]
	infrastructure := ctx.infrastructuresByID[infrastructureID]
	if infrastructure == nil {
		record.invalidate("unknown_infrastructure")
		return
	}
	infrastructureType := infrastructure.Type
	ctx.removeInfrastructure(infrastructureID)
	creditTerritoryID := ctx.closestControlledSettlement(army.TerritoryID, army.OwnerID)
	if creditTerritoryID != "" {
		creditState := ctx.state.TerritoryStates[creditTerritoryID]
		creditState.Resources += ctx.balance.PillageBonus
		ctx.state.TerritoryStates[creditTerritoryID] = creditState
	}
	record.outcome = OutcomeSuccess
	record.reason = "pillaged"
	ctx.events = append(ctx.events, Event{
		Type:               EventTypePillage,
		Phase:              4,
		ArmyID:             army.ID,
		TerritoryID:        army.TerritoryID,
		InfrastructureID:   infrastructureID,
		InfrastructureType: infrastructureType,
		ResourceCredit:     ctx.creditAmount(creditTerritoryID),
		CreditTerritoryID:  creditTerritoryID,
		Outcome:            OutcomeSuccess,
	})
}

func (ctx *resolutionContext) creditAmount(territoryID models.TerritoryID) int {
	if territoryID == "" {
		return 0
	}
	return ctx.balance.PillageBonus
}

func (ctx *resolutionContext) closestControlledSettlement(startID models.TerritoryID, ownerID models.PlayerID) models.TerritoryID {
	type queueItem struct {
		territoryID models.TerritoryID
		distance    int
	}
	queue := []queueItem{{territoryID: startID}}
	visited := map[models.TerritoryID]bool{startID: true}
	for len(queue) > 0 {
		distance := queue[0].distance
		level := make([]queueItem, 0)
		for len(queue) > 0 && queue[0].distance == distance {
			level = append(level, queue[0])
			queue = queue[1:]
		}
		candidates := make([]models.TerritoryID, 0)
		for _, item := range level {
			state := ctx.state.TerritoryStates[item.territoryID]
			if state.OwnerID != nil && *state.OwnerID == ownerID && (ctx.hasInfrastructure(item.territoryID, models.InfraTypeCastle) || ctx.hasInfrastructure(item.territoryID, models.InfraTypeVillage)) {
				candidates = append(candidates, item.territoryID)
			}
		}
		if len(candidates) > 0 {
			sort.Slice(candidates, func(i, j int) bool {
				left := ctx.territoriesByID[candidates[i]]
				right := ctx.territoriesByID[candidates[j]]
				if left.Code != right.Code {
					return left.Code < right.Code
				}
				return left.ID < right.ID
			})
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

func (ctx *resolutionContext) executeNobleStatusOrder(record *orderRecord, army *models.Army) {
	targetID := record.order.NobleTargetIDs[0]
	target := ctx.noblesByID[targetID]
	if target == nil || target.Status == models.NobleStatusFree || target.LocationID != army.TerritoryID || target.OwnerID == army.OwnerID {
		record.invalidate("noble_not_prisoner")
		return
	}
	previousStatus := target.Status
	if record.order.Type == models.OrderTypeHostage {
		target.Status = models.NobleStatusHostage
	} else {
		target.Status = models.NobleStatusDungeon
	}
	record.outcome = OutcomeSuccess
	record.reason = "noble_status_changed"
	ctx.events = append(ctx.events, Event{
		Type:           EventTypeCapture,
		Phase:          4,
		ArmyID:         army.ID,
		TerritoryID:    army.TerritoryID,
		NobleID:        target.ID,
		PreviousStatus: previousStatus,
		Status:         target.Status,
		Outcome:        OutcomeSuccess,
	})
}

func executeRetreats(ctx *resolutionContext) error {
	plans := make(map[models.ArmyID]*retreatPlan, len(ctx.dislodged))
	claims := make(map[models.TerritoryID][]*retreatPlan)
	for _, armyID := range sortedArmyMap(ctx.dislodged) {
		displaced := ctx.dislodged[armyID]
		candidates := retreatCandidates(ctx, displaced)
		plan := &retreatPlan{dislodged: displaced}
		plans[armyID] = plan
		if len(candidates) == 0 {
			plan.destroyReason = "no_retreat_destination"
			continue
		}
		for _, candidateID := range candidates {
			if len(claims[candidateID]) == 0 {
				plan.destinationID = candidateID
				break
			}
		}
		if plan.destinationID == "" {
			plan.destinationID = candidates[0]
		}
		claims[plan.destinationID] = append(claims[plan.destinationID], plan)
	}
	for _, territoryID := range sortedTerritoryMap(claims) {
		if len(claims[territoryID]) < 2 {
			continue
		}
		for _, plan := range claims[territoryID] {
			plan.destroyReason = "retreat_collision"
		}
	}
	for _, armyID := range sortedArmyMap(plans) {
		plan := plans[armyID]
		if plan.destroyReason != "" {
			ctx.destroyDislodgedArmy(plan)
			continue
		}
		army := plan.dislodged.army
		army.TerritoryID = plan.destinationID
		ctx.state.Armies = append(ctx.state.Armies, army)
		ctx.moveNobles(plan.dislodged.nobleIDs, plan.destinationID, army.ID)
		ctx.events = append(ctx.events, Event{
			Type:             EventTypeRetreat,
			Phase:            4,
			ArmyID:           army.ID,
			SourceID:         plan.dislodged.originID,
			DestinationID:    plan.destinationID,
			AttackerOriginID: plan.dislodged.attackerOriginID,
			Outcome:          OutcomeSuccess,
		})
	}
	if err := ctx.rebuildOccupancy(); err != nil {
		return err
	}
	for _, armyID := range sortedArmyMap(plans) {
		plan := plans[armyID]
		if plan.destroyReason != "" {
			ctx.captureNoblesAfterDestruction(plan)
		}
	}
	return nil
}

func retreatCandidates(ctx *resolutionContext, displaced *dislodgedArmy) []models.TerritoryID {
	candidates := make([]models.TerritoryID, 0)
	for _, territoryID := range ctx.sortedNeighbors(displaced.originID) {
		if ctx.currentArmyAt(territoryID) != nil || ctx.attackedTerritories[territoryID] || territoryID == displaced.attackerOriginID {
			continue
		}
		candidates = append(candidates, territoryID)
	}
	return candidates
}

func (ctx *resolutionContext) destroyDislodgedArmy(plan *retreatPlan) {
	army := plan.dislodged.army
	if record := ctx.records[army.ID]; record != nil {
		record.destroyed = true
	}
	ctx.events = append(ctx.events, Event{
		Type:             EventTypeArmyDestroyed,
		Phase:            4,
		ArmyID:           army.ID,
		TerritoryID:      plan.dislodged.originID,
		AttackerOriginID: plan.dislodged.attackerOriginID,
		Reason:           plan.destroyReason,
	})
}

func (ctx *resolutionContext) captureNoblesAfterDestruction(plan *retreatPlan) {
	occupier := ctx.currentArmyAt(plan.dislodged.originID)
	if occupier == nil {
		return
	}
	for _, nobleID := range plan.dislodged.nobleIDs {
		noble := ctx.noblesByID[nobleID]
		if noble == nil || noble.OwnerID == occupier.OwnerID {
			continue
		}
		previousStatus := noble.Status
		noble.Status = models.NobleStatusHostage
		ctx.events = append(ctx.events, Event{
			Type:           EventTypeCapture,
			Phase:          4,
			ArmyID:         occupier.ID,
			TerritoryID:    plan.dislodged.originID,
			NobleID:        noble.ID,
			PreviousStatus: previousStatus,
			Status:         noble.Status,
			CaptorPlayerID: occupier.OwnerID,
		})
	}
}

func fmtMissingArmy(armyID models.ArmyID) error {
	return fmt.Errorf("engine: expected live army %q during movement", armyID)
}
