package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type contestState struct {
	active           map[models.ArmyID]bool
	dislodged        map[models.ArmyID]bool
	vacated          map[models.ArmyID]bool
	disperseResidual map[models.ArmyID]int
	ghosted          map[models.ArmyID]bool
	retiredGhosts    map[models.ArmyID]bool
	cuts             map[models.ArmyID]bool
	voidedSupports   map[models.ArmyID]bool
	results          map[models.TerritoryID]contestResult
}

type peacefulDepartureState struct {
	vacated          map[models.ArmyID]bool
	disperseResidual map[models.ArmyID]int
}

type contestResult struct {
	territoryID      models.TerritoryID
	defenderID       models.ArmyID
	baseDefense      int
	defense          int
	castleBonus      int
	contenders       []CombatContender
	winnerID         models.ArmyID
	dislodgedArmyID  models.ArmyID
	attackerOriginID models.TerritoryID
	cutSupporterIDs  []models.ArmyID
	standoff         bool
}

type dislodgedArmy struct {
	army             models.Army
	originID         models.TerritoryID
	attackerOriginID models.TerritoryID
	nobleIDs         []models.NobleID
}

func (ctx *resolutionContext) predictPeacefulDepartures(current contestState, vacated map[models.ArmyID]bool, disperseResidual map[models.ArmyID]int) peacefulDepartureState {
	temp := ctx.cloneForDeparturePrediction()
	effectiveVacated := mergeBooleanMaps(vacated, current.vacated)
	temp.contest = current
	temp.contest.vacated = effectiveVacated
	temp.contest.disperseResidual = copyIntegerMap(disperseResidual)
	temp.dislodged = dislodgedFromContest(temp, current.results)
	temp.attackedTerritories = ctx.attackedTerritoriesFor(effectiveVacated, current.dislodged)
	temp.applyContestOutcomes()
	resolveDispersions(temp)
	resolveJoins(temp)
	resolveVacatedDisperseDestinations(temp)
	finalizeDispersions(temp)

	result := peacefulDepartureState{
		vacated:          make(map[models.ArmyID]bool),
		disperseResidual: make(map[models.ArmyID]int),
	}
	for armyID, residual := range disperseResidual {
		if current.dislodged[armyID] {
			result.disperseResidual[armyID] = residual
		}
	}
	for _, armyID := range sortedArmyMap(ctx.joins) {
		record := temp.records[armyID]
		if record != nil && record.outcome == OutcomeSuccess && !current.dislodged[armyID] {
			result.vacated[armyID] = true
		}
	}
	for _, recordArmyID := range sortedArmyMap(temp.disperseResults) {
		disperse := temp.disperseResults[recordArmyID]
		record := temp.records[recordArmyID]
		if record == nil || record.outcome == OutcomeInvalid || disperse.invalid {
			continue
		}
		if temp.disperseVacatesSource(disperse.intent.armyID, disperse.intent.source) {
			result.vacated[disperse.intent.armyID] = true
		} else if record.partialD {
			result.disperseResidual[disperse.intent.armyID] = disperse.remaining
		}
	}
	return result
}

func (ctx *resolutionContext) cloneForDeparturePrediction() *resolutionContext {
	temp := newResolutionContext(cloneGameState(ctx.state), ctx.balance)
	temp.famished = copyBooleanMap(ctx.famished)
	temp.records = make(map[models.ArmyID]*orderRecord, len(ctx.records))
	for armyID, record := range ctx.records {
		copyRecord := *record
		copyRecord.order.TargetIDs = append([]models.TerritoryID(nil), record.order.TargetIDs...)
		temp.records[armyID] = &copyRecord
	}
	temp.attacks = copyAttackIntents(ctx.attacks)
	temp.joins = copyJoinIntents(ctx.joins)
	temp.disperses = copyDisperseIntents(ctx.disperses)
	temp.supports = copySupportIntents(ctx.supports)
	temp.attackedTerritories = copyBooleanTerritories(ctx.attackedTerritories)
	return temp
}

func (ctx *resolutionContext) attackedTerritoriesFor(vacated, dislodged map[models.ArmyID]bool) map[models.TerritoryID]bool {
	attacked := make(map[models.TerritoryID]bool)
	excluded := ctx.alliedDestinationAttacks(vacated, dislodged)
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		if excluded[armyID] {
			continue
		}
		attacked[ctx.attacks[armyID].target] = true
	}
	return attacked
}

func dislodgedFromContest(ctx *resolutionContext, results map[models.TerritoryID]contestResult) map[models.ArmyID]*dislodgedArmy {
	dislodged := make(map[models.ArmyID]*dislodgedArmy)
	for territoryID, result := range results {
		if result.dislodgedArmyID == "" {
			continue
		}
		army, exists := ctx.startArmiesByID[result.dislodgedArmyID]
		if !exists {
			continue
		}
		dislodged[army.ID] = &dislodgedArmy{
			army:             army,
			originID:         territoryID,
			attackerOriginID: result.attackerOriginID,
		}
	}
	return dislodged
}

func mergeBooleanMaps(left, right map[models.ArmyID]bool) map[models.ArmyID]bool {
	merged := make(map[models.ArmyID]bool, len(left)+len(right))
	for key, value := range left {
		if value {
			merged[key] = true
		}
	}
	for key, value := range right {
		if value {
			merged[key] = true
		}
	}
	return merged
}

func copyBooleanMap(source map[models.ArmyID]bool) map[models.ArmyID]bool {
	copyMap := make(map[models.ArmyID]bool, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}

func copyBooleanTerritories(source map[models.TerritoryID]bool) map[models.TerritoryID]bool {
	copyMap := make(map[models.TerritoryID]bool, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}

func copyIntegerMap(source map[models.ArmyID]int) map[models.ArmyID]int {
	copyMap := make(map[models.ArmyID]int, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}

func copyAttackIntents(source map[models.ArmyID]*attackIntent) map[models.ArmyID]*attackIntent {
	copyMap := make(map[models.ArmyID]*attackIntent, len(source))
	for key, intent := range source {
		copyIntent := *intent
		copyMap[key] = &copyIntent
	}
	return copyMap
}

func copyJoinIntents(source map[models.ArmyID]*joinIntent) map[models.ArmyID]*joinIntent {
	copyMap := make(map[models.ArmyID]*joinIntent, len(source))
	for key, intent := range source {
		copyIntent := *intent
		copyMap[key] = &copyIntent
	}
	return copyMap
}

func copyDisperseIntents(source map[models.ArmyID]*disperseIntent) map[models.ArmyID]*disperseIntent {
	copyMap := make(map[models.ArmyID]*disperseIntent, len(source))
	for key, intent := range source {
		copyIntent := *intent
		copyIntent.targets = append([]models.TerritoryID(nil), intent.targets...)
		copyIntent.nobles = append([]models.NobleID(nil), intent.nobles...)
		copyIntent.assignments = copyNobleAssignments(intent.assignments)
		copyMap[key] = &copyIntent
	}
	return copyMap
}

func copyNobleAssignments(source map[models.TerritoryID][]models.NobleID) map[models.TerritoryID][]models.NobleID {
	copyMap := make(map[models.TerritoryID][]models.NobleID, len(source))
	for territoryID, nobleIDs := range source {
		copyMap[territoryID] = append([]models.NobleID(nil), nobleIDs...)
	}
	return copyMap
}

func copySupportIntents(source map[models.ArmyID]*supportIntent) map[models.ArmyID]*supportIntent {
	copyMap := make(map[models.ArmyID]*supportIntent, len(source))
	for key, intent := range source {
		copyIntent := *intent
		copyMap[key] = &copyIntent
	}
	return copyMap
}

func (ctx *resolutionContext) alliedDestinationAttacks(vacated, previouslyDislodged map[models.ArmyID]bool) map[models.ArmyID]bool {
	excluded := make(map[models.ArmyID]bool)
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		attack := ctx.attacks[armyID]
		defender := ctx.startArmyAt(attack.target)
		if defender == nil || vacated[defender.ID] || previouslyDislodged[defender.ID] {
			continue
		}
		attacker := ctx.startArmiesByID[armyID]
		if attacker.OwnerID == defender.OwnerID {
			excluded[armyID] = true
		}
	}
	return excluded
}

func (ctx *resolutionContext) calculateSupportCuts(excluded map[models.ArmyID]bool) map[models.ArmyID]bool {
	cuts := make(map[models.ArmyID]bool, len(ctx.supports))
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		exemptOriginID := support.targetID
		if support.offensive {
			exemptOriginID = support.destinationID
		}
		for _, attackID := range sortedArmyMap(ctx.attacks) {
			if excluded[attackID] {
				continue
			}
			attack := ctx.attacks[attackID]
			attacker := ctx.startArmiesByID[attackID]
			supporter := ctx.startArmiesByID[supportID]
			if attack.target == support.source && attack.source != exemptOriginID && attacker.OwnerID != supporter.OwnerID {
				cuts[supportID] = true
				break
			}
		}
	}
	return cuts
}

func (ctx *resolutionContext) removeAlliedDestinationAttacks() {
	removedTargets := make(map[models.TerritoryID]bool)
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		attack := ctx.attacks[armyID]
		defender := ctx.startArmyAt(attack.target)
		if defender == nil || ctx.contest.vacated[defender.ID] || ctx.contest.dislodged[defender.ID] {
			continue
		}
		attacker := ctx.startArmiesByID[armyID]
		if attacker.OwnerID != defender.OwnerID || ctx.contest.dislodged[armyID] {
			continue
		}
		record := ctx.records[armyID]
		if record == nil || record.outcome != "" {
			continue
		}
		record.fail("allied_destination")
		delete(ctx.attacks, armyID)
		removedTargets[attack.target] = true
	}
	for targetID := range removedTargets {
		if !ctx.hasAttackTarget(targetID) {
			delete(ctx.attackedTerritories, targetID)
			delete(ctx.contest.results, targetID)
		}
	}
}

func (ctx *resolutionContext) clearSupportsForRemovedAttacks() {
	for _, support := range ctx.supports {
		if support.targetArmyID == "" || !support.offensive {
			continue
		}
		if _, exists := ctx.attacks[support.targetArmyID]; !exists {
			support.applies = false
		}
	}
}

func (ctx *resolutionContext) hasAttackTarget(targetID models.TerritoryID) bool {
	for _, attack := range ctx.attacks {
		if attack.target == targetID {
			return true
		}
	}
	return false
}

func supportRelevantToTerritory(ctx *resolutionContext, support *supportIntent, territoryID models.TerritoryID) bool {
	if support.offensive {
		attack := ctx.attacks[support.targetArmyID]
		return attack != nil && attack.target == territoryID
	}
	return support.targetID == territoryID
}

func (ctx *resolutionContext) defensiveSupportStrength(armyID models.ArmyID, cuts, previouslyDislodged map[models.ArmyID]bool) int {
	strength := 0
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if !support.applies || support.offensive || support.targetArmyID != armyID || cuts[supportID] || previouslyDislodged[supportID] {
			continue
		}
		if !ctx.famished[supportID] {
			strength += ctx.startArmiesByID[supportID].Size
		}
	}
	return strength
}

func (ctx *resolutionContext) emitCombatEvents() {
	for _, territoryID := range sortedTerritoryMap(ctx.contest.results) {
		result := ctx.contest.results[territoryID]
		reason := "defense_holds"
		if result.standoff {
			reason = "standoff"
		} else if result.winnerID != "" {
			reason = "attack_wins"
		}
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeCombat,
			Phase:           3,
			TerritoryID:     territoryID,
			BaseDefense:     result.baseDefense,
			Defense:         result.defense,
			CastleBonus:     result.castleBonus,
			Contenders:      append([]CombatContender(nil), result.contenders...),
			WinnerArmyID:    result.winnerID,
			DislodgedArmyID: result.dislodgedArmyID,
			CutSupporterIDs: append([]models.ArmyID(nil), result.cutSupporterIDs...),
			Reason:          reason,
		})
	}
}

func (ctx *resolutionContext) applyContestOutcomes() {
	for _, armyID := range sortedArmyMap(ctx.records) {
		record := ctx.records[armyID]
		if record.outcome != "" {
			continue
		}
		if ctx.contest.dislodged[record.armyID] {
			record.fail("dislodged")
			if record.pendingDisperse {
				ctx.clearPendingDisperse(record)
			}
			continue
		}
		executionArmyID := record.executionArmyID
		if ctx.contest.dislodged[executionArmyID] {
			if record.pendingDisperse {
				record.invalidate("disperse_residual_dislodged")
				ctx.clearPendingDisperse(record)
				continue
			}
			record.fail("dislodged")
			continue
		}
		attack := ctx.attacks[armyID]
		if attack == nil {
			continue
		}
		result := ctx.contest.results[attack.target]
		if ctx.contest.active[armyID] && result.winnerID == armyID {
			record.outcome = OutcomeSuccess
			record.reason = "attack_wins"
		} else {
			record.fail("combat_lost")
		}
	}
}

func (ctx *resolutionContext) clearPendingDisperse(record *orderRecord) {
	if chain := ctx.chainsByID[record.chainID]; chain != nil {
		chain.PendingDisperse = nil
	}
}

func sameContestState(left, right contestState) bool {
	if !sameBooleanMap(left.active, right.active) || !sameBooleanMap(left.dislodged, right.dislodged) || !sameBooleanMap(left.vacated, right.vacated) || !sameIntegerMap(left.disperseResidual, right.disperseResidual) || !sameBooleanMap(left.ghosted, right.ghosted) || !sameBooleanMap(left.retiredGhosts, right.retiredGhosts) || !sameBooleanMap(left.cuts, right.cuts) || !sameBooleanMap(left.voidedSupports, right.voidedSupports) || len(left.results) != len(right.results) {
		return false
	}
	for territoryID, leftResult := range left.results {
		rightResult, exists := right.results[territoryID]
		if !exists || !sameContestResult(leftResult, rightResult) {
			return false
		}
	}
	return true
}

func sameBooleanMap(left, right map[models.ArmyID]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func sameIntegerMap(left, right map[models.ArmyID]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func sameContestResult(left, right contestResult) bool {
	if left.territoryID != right.territoryID || left.defenderID != right.defenderID || left.baseDefense != right.baseDefense || left.defense != right.defense || left.castleBonus != right.castleBonus || left.winnerID != right.winnerID || left.dislodgedArmyID != right.dislodgedArmyID || left.attackerOriginID != right.attackerOriginID || left.standoff != right.standoff || len(left.contenders) != len(right.contenders) || len(left.cutSupporterIDs) != len(right.cutSupporterIDs) {
		return false
	}
	for index := range left.contenders {
		if left.contenders[index] != right.contenders[index] {
			return false
		}
	}
	for index := range left.cutSupporterIDs {
		if left.cutSupporterIDs[index] != right.cutSupporterIDs[index] {
			return false
		}
	}
	return true
}
