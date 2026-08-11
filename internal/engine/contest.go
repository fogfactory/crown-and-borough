package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type contestState struct {
	active    map[models.ArmyID]bool
	dislodged map[models.ArmyID]bool
	vacated   map[models.ArmyID]bool
	cuts      map[models.ArmyID]bool
	results   map[models.TerritoryID]contestResult
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

func resolveContests(ctx *resolutionContext) error {
	if len(ctx.attacks) == 0 {
		ctx.contest = contestState{
			active:    map[models.ArmyID]bool{},
			dislodged: map[models.ArmyID]bool{},
			vacated:   map[models.ArmyID]bool{},
			cuts:      map[models.ArmyID]bool{},
			results:   map[models.TerritoryID]contestResult{},
		}
		return nil
	}

	active := make(map[models.ArmyID]bool, len(ctx.attacks))
	for armyID := range ctx.attacks {
		active[armyID] = true
	}
	dislodged := map[models.ArmyID]bool{}
	// Start optimistically so pure departure cycles, such as swaps, can resolve.
	vacated := make(map[models.ArmyID]bool, len(ctx.attacks))
	for armyID := range ctx.attacks {
		vacated[armyID] = true
	}
	var previous contestState
	hasPrevious := false
	moveOrderCount := len(ctx.attacks) + len(ctx.joins) + len(ctx.disperses)
	maxPasses := 2*moveOrderCount + 2
	for pass := 0; ; pass++ {
		current := ctx.evaluateContests(active, dislodged, vacated)
		if hasPrevious && sameContestState(previous, current) {
			ctx.contest = current
			break
		}
		if pass >= maxPasses {
			return fmt.Errorf("engine: combat resolution did not converge after %d passes", maxPasses+1)
		}
		previous = current
		hasPrevious = true
		active = current.active
		dislodged = current.dislodged
		vacated = current.vacated
	}

	for territoryID, result := range ctx.contest.results {
		if result.dislodgedArmyID == "" {
			continue
		}
		army, exists := ctx.startArmiesByID[result.dislodgedArmyID]
		if !exists {
			return fmt.Errorf("engine: combat dislodged unknown army %q", result.dislodgedArmyID)
		}
		ctx.dislodged[army.ID] = &dislodgedArmy{
			army:             army,
			originID:         territoryID,
			attackerOriginID: result.attackerOriginID,
		}
	}
	ctx.removeAlliedDestinationAttacks()
	ctx.clearSupportsForRemovedAttacks()
	ctx.emitCombatEvents()
	ctx.applyContestOutcomes()
	return nil
}

func (ctx *resolutionContext) evaluateContests(active, previouslyDislodged, vacated map[models.ArmyID]bool) contestState {
	excluded := ctx.alliedDestinationAttacks(vacated, previouslyDislodged)
	cuts := ctx.calculateSupportCuts(active, excluded)
	results := make(map[models.TerritoryID]contestResult)
	attacksByTarget := make(map[models.TerritoryID][]*attackIntent)
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		if !active[armyID] || excluded[armyID] {
			continue
		}
		attack := ctx.attacks[armyID]
		attacksByTarget[attack.target] = append(attacksByTarget[attack.target], attack)
	}
	for _, territoryID := range sortedTerritoryMap(attacksByTarget) {
		attacks := attacksByTarget[territoryID]
		result := ctx.resolveTerritoryContest(territoryID, attacks, cuts, previouslyDislodged, vacated)
		results[territoryID] = result
	}
	dislodged := make(map[models.ArmyID]bool)
	for _, result := range results {
		if result.dislodgedArmyID != "" {
			dislodged[result.dislodgedArmyID] = true
		}
	}
	nextActive := make(map[models.ArmyID]bool, len(ctx.attacks))
	for armyID := range ctx.attacks {
		nextActive[armyID] = !dislodged[armyID]
	}
	nextVacated := make(map[models.ArmyID]bool)
	for _, result := range results {
		if result.winnerID != "" && !dislodged[result.winnerID] {
			nextVacated[result.winnerID] = true
		}
	}
	return contestState{
		active:    nextActive,
		dislodged: dislodged,
		vacated:   nextVacated,
		cuts:      cuts,
		results:   results,
	}
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

func (ctx *resolutionContext) calculateSupportCuts(active map[models.ArmyID]bool, excluded map[models.ArmyID]bool) map[models.ArmyID]bool {
	cuts := make(map[models.ArmyID]bool, len(ctx.supports))
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		exemptOriginID := support.targetID
		if support.offensive {
			exemptOriginID = support.destinationID
		}
		for _, attackID := range sortedArmyMap(ctx.attacks) {
			if !active[attackID] || excluded[attackID] {
				continue
			}
			attack := ctx.attacks[attackID]
			if attack.target == support.source && attack.source != exemptOriginID {
				cuts[supportID] = true
				break
			}
		}
	}
	return cuts
}

func (ctx *resolutionContext) resolveTerritoryContest(
	territoryID models.TerritoryID,
	attacks []*attackIntent,
	cuts map[models.ArmyID]bool,
	previouslyDislodged map[models.ArmyID]bool,
	vacated map[models.ArmyID]bool,
) contestResult {
	defender := ctx.startArmyAt(territoryID)
	if defender != nil && vacated[defender.ID] {
		defender = nil
	}
	castleBonus := 0
	if ctx.hasCastle(territoryID) {
		castleBonus = ctx.balance.CastleDefenseBonus
	}
	baseDefense := castleBonus
	defense := baseDefense
	defenderID := models.ArmyID("")
	defenderOwnerID := models.PlayerID("")
	if defender != nil {
		if !ctx.famished[defender.ID] {
			baseDefense += defender.Size
			defense += defender.Size
		}
		defenderID = defender.ID
		defenderOwnerID = defender.OwnerID
		defense += ctx.defensiveSupportStrength(defender.ID, cuts, previouslyDislodged)
	}
	result := contestResult{
		territoryID: territoryID,
		defenderID:  defenderID,
		baseDefense: baseDefense,
		defense:     defense,
		castleBonus: castleBonus,
	}
	if defender != nil || castleBonus > 0 {
		result.contenders = append(result.contenders, CombatContender{
			ArmyID:   defenderID,
			OwnerID:  defenderOwnerID,
			Force:    defense,
			Defender: true,
		})
	}

	bestForce := defense
	bestArmyID := models.ArmyID("")
	tied := false
	singleAttackForce := -1
	for _, attack := range attacks {
		attacker := ctx.startArmiesByID[attack.armyID]
		force := attack.size + ctx.offensiveSupportStrength(attack.armyID, cuts, previouslyDislodged)
		if len(attacks) == 1 {
			singleAttackForce = force
		}
		result.contenders = append(result.contenders, CombatContender{
			ArmyID:  attack.armyID,
			OwnerID: attacker.OwnerID,
			Force:   force,
		})
		if force > bestForce {
			bestForce = force
			bestArmyID = attack.armyID
			tied = false
		} else if force == bestForce {
			tied = true
		}
	}
	if defender == nil && castleBonus == 0 && len(attacks) == 1 && singleAttackForce == 0 {
		result.winnerID = attacks[0].armyID
	} else if !tied && bestArmyID != "" {
		result.winnerID = bestArmyID
		if defender != nil {
			result.dislodgedArmyID = defender.ID
			result.attackerOriginID = ctx.attacks[bestArmyID].source
		}
	} else if tied {
		result.standoff = true
	}
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if cuts[supportID] && supportRelevantToTerritory(ctx, support, territoryID) {
			result.cutSupporterIDs = append(result.cutSupporterIDs, supportID)
		}
	}
	return result
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
		}
	}
}

func (ctx *resolutionContext) clearSupportsForRemovedAttacks() {
	for _, support := range ctx.supports {
		if support.targetArmyID == "" {
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

func (ctx *resolutionContext) offensiveSupportStrength(armyID models.ArmyID, cuts, previouslyDislodged map[models.ArmyID]bool) int {
	strength := 0
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if !support.applies || !support.offensive || support.targetArmyID != armyID || cuts[supportID] || previouslyDislodged[supportID] {
			continue
		}
		if !ctx.famished[supportID] {
			strength += ctx.startArmiesByID[supportID].Size
		}
	}
	return strength
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
	if !sameBooleanMap(left.active, right.active) || !sameBooleanMap(left.dislodged, right.dislodged) || !sameBooleanMap(left.vacated, right.vacated) || !sameBooleanMap(left.cuts, right.cuts) || len(left.results) != len(right.results) {
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
