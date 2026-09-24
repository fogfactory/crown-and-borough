package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// contestState is the settled combat of a turn, read by movement execution
// and chain progression.
type contestState struct {
	active         map[models.ArmyID]bool
	dislodged      map[models.ArmyID]bool
	vacated        map[models.ArmyID]bool
	cuts           map[models.ArmyID]bool
	voidedSupports map[models.ArmyID]bool
	results        map[models.TerritoryID]contestResult
}

type contestResult struct {
	territoryID      models.TerritoryID
	defenderID       models.ArmyID
	baseDefense      int
	defense          int
	castleBonus      int
	contenders       []CombatContender
	supporterIDs     []models.ArmyID
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

// resolveContests adjudicates every attack, join and dispersion of the turn
// (see adjudicator), then records the combats, their events and the outcome
// of every attack. Joins and dispersions are executed later from the settled
// contest.
func resolveContests(ctx *resolutionContext) (err error) {
	ctx.facts = settledFacts{ctx: ctx}
	if len(ctx.attacks) == 0 {
		ctx.contest = contestState{
			active:         map[models.ArmyID]bool{},
			dislodged:      map[models.ArmyID]bool{},
			vacated:        map[models.ArmyID]bool{},
			cuts:           map[models.ArmyID]bool{},
			voidedSupports: map[models.ArmyID]bool{},
			results:        map[models.TerritoryID]contestResult{},
		}
		return nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			unsettled, isUnsettled := recovered.(unsettledDecision)
			if !isUnsettled {
				panic(recovered)
			}
			err = unsettled
		}
	}()
	adj := newAdjudicator(ctx)
	adj.resolveAll()
	ctx.contest = adj.settledContest()
	for _, territoryID := range sortedTerritoryMap(ctx.contest.results) {
		result := ctx.contest.results[territoryID]
		if result.dislodgedArmyID == "" {
			continue
		}
		ctx.dislodged[result.dislodgedArmyID] = &dislodgedArmy{
			army:             ctx.startArmiesByID[result.dislodgedArmyID],
			originID:         territoryID,
			attackerOriginID: result.attackerOriginID,
		}
	}
	ctx.removeAlliedDestinationAttacks()
	ctx.clearSupportsForRemovedAttacks()
	ctx.clearSupportsForVoidedAttacks(ctx.contest.voidedSupports)
	ctx.emitCombatEvents()
	ctx.applyContestOutcomes()
	return nil
}

// settledContest reads the contest from the settled decisions.
func (adj *adjudicator) settledContest() contestState {
	ctx := adj.ctx
	contest := contestState{
		active:         make(map[models.ArmyID]bool, len(ctx.attacks)),
		dislodged:      make(map[models.ArmyID]bool),
		vacated:        make(map[models.ArmyID]bool),
		cuts:           make(map[models.ArmyID]bool),
		voidedSupports: adj.voidedSupports(),
		results:        adj.contestResults(),
	}
	for _, armyID := range sortedArmyMap(ctx.startArmiesByID) {
		if adj.dislodged(armyID) {
			contest.dislodged[armyID] = true
		}
	}
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		contest.active[armyID] = !contest.dislodged[armyID]
		if adj.moves(armyID) {
			contest.vacated[armyID] = true
		}
	}
	for _, armyID := range sortedArmyMap(adj.peaceful) {
		if adj.departs(armyID) {
			contest.vacated[armyID] = true
		}
	}
	for _, supportID := range sortedArmyMap(ctx.supports) {
		if adj.staticCuts[supportID] || contest.dislodged[supportID] {
			contest.cuts[supportID] = true
		}
	}
	return contest
}

// contestResults describes the combat on every attacked territory: the army
// or castle defending it, every attack with its strength, and the winner.
// An attack that lost its head-to-head battle is not listed at its
// destination, which it no longer contests.
func (adj *adjudicator) contestResults() map[models.TerritoryID]contestResult {
	ctx := adj.ctx
	results := make(map[models.TerritoryID]contestResult, len(adj.attacksByTarget))
	for _, territoryID := range sortedTerritoryMap(adj.attacksByTarget) {
		defender := ctx.startArmyAt(territoryID)
		present := defender != nil && adj.stays(defender.ID)
		result := contestResult{territoryID: territoryID}
		if ctx.hasCastle(territoryID) && (present || !ctx.castleOwnedByAllAttackers(territoryID)) {
			result.castleBonus = ctx.balance.CastleDefenseBonus
		}
		result.baseDefense, result.defense = result.castleBonus, result.castleBonus
		defenderOwnerID := models.PlayerID("")
		if defender != nil && !adj.departs(defender.ID) && !adj.dislodged(defender.ID) {
			defenderOwnerID = defender.OwnerID
		}
		defenderNobleBonus := 0
		if present {
			result.defenderID = defender.ID
			defenderOwnerID = defender.OwnerID
			base, supported := adj.defenseStrength(*defender)
			result.baseDefense, result.defense = base, base+supported
			defenderNobleBonus = nobleCommandBonus(ctx, *defender)
		}
		if result.defenderID != "" || result.castleBonus > 0 {
			result.contenders = append(result.contenders, CombatContender{
				ArmyID: result.defenderID, OwnerID: defenderOwnerID, Force: result.defense,
				NobleBonus: defenderNobleBonus, Defender: true,
			})
		}
		for _, armyID := range adj.attacksByTarget[territoryID] {
			if adj.retiredGhost(armyID) {
				continue
			}
			force, _ := adj.attackForce(armyID, "")
			attacker := ctx.startArmiesByID[armyID]
			result.contenders = append(result.contenders, CombatContender{
				ArmyID: armyID, OwnerID: attacker.OwnerID, Force: force, NobleBonus: nobleCommandBonus(ctx, attacker),
			})
			if adj.moves(armyID) {
				result.winnerID = armyID
				if result.defenderID != "" {
					result.dislodgedArmyID = result.defenderID
					result.attackerOriginID = ctx.attacks[armyID].source
				}
			}
		}
		result.standoff = result.winnerID == "" && result.hasAttackStandoff()
		for _, supportID := range sortedArmyMap(ctx.supports) {
			support := ctx.supports[supportID]
			if !supportRelevantToTerritory(ctx, support, territoryID) {
				continue
			}
			if support.applies {
				result.supporterIDs = append(result.supporterIDs, supportID)
			}
			if adj.staticCuts[supportID] || adj.dislodged(supportID) {
				result.cutSupporterIDs = append(result.cutSupporterIDs, supportID)
			}
		}
		results[territoryID] = result
	}
	return results
}

// voidedSupports returns the offensive supports that are reported void
// because the attack they support could not use them:
//   - in a head-to-head battle, the supports given by the opponent's owner to
//     the side that loses it (both sides on a tie or between allies);
//   - every support of an attack that would win its destination but is
//     turned back because it cannot dislodge the army staying there, an ally
//     or an army it outmatches only with the help of that army's owner.
//
// The head-to-head rule is reported while both armies stay in place, or when
// the losing side is itself dislodged.
func (adj *adjudicator) voidedSupports() map[models.ArmyID]bool {
	ctx := adj.ctx
	voided := make(map[models.ArmyID]bool)
	voidSupports := func(armyID models.ArmyID, ownerID models.PlayerID) {
		for _, supportID := range sortedArmyMap(ctx.supports) {
			support := ctx.supports[supportID]
			if support.offensive && support.targetArmyID == armyID && (ownerID == "" || ctx.startArmiesByID[supportID].OwnerID == ownerID) {
				voided[supportID] = true
			}
		}
	}
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		attack := ctx.attacks[armyID]
		defender := ctx.startArmyAt(attack.target)
		lostHeadToHead := false
		if defender != nil && adj.headToHead(armyID) && (adj.dislodged(armyID) || !adj.dislodged(defender.ID)) {
			owner := ctx.startArmiesByID[armyID].OwnerID
			ourForce, ourHelp := adj.attackForce(armyID, defender.OwnerID)
			theirForce, theirHelp := adj.attackForce(defender.ID, owner)
			lostHeadToHead = defender.OwnerID == owner || ourForce-ourHelp <= theirForce-theirHelp
		}
		if lostHeadToHead {
			voidSupports(armyID, defender.OwnerID)
			continue
		}
		if adj.moves(armyID) || defender == nil || !adj.stays(defender.ID) || adj.dislodged(defender.ID) {
			continue
		}
		force, _ := adj.attackForce(armyID, "")
		if force <= adj.holdStrength(*defender) {
			continue
		}
		outmatched := true
		for _, otherID := range adj.attacksByTarget[attack.target] {
			if otherID == armyID || adj.retiredGhost(otherID) {
				continue
			}
			if prevent, _ := adj.attackForce(otherID, ""); force <= prevent {
				outmatched = false
				break
			}
		}
		if outmatched {
			voidSupports(armyID, "")
		}
	}
	return voided
}

func (result contestResult) hasAttackStandoff() bool {
	if len(result.contenders) < 2 {
		return false
	}
	maxForce, count := -1, 0
	attackAtTop := false
	for _, contender := range result.contenders {
		if contender.Force > maxForce {
			maxForce = contender.Force
			count = 1
			attackAtTop = !contender.Defender
		} else if contender.Force == maxForce {
			count++
			attackAtTop = attackAtTop || !contender.Defender
		}
	}
	return count > 1 && attackAtTop
}

func nobleCommandBonus(ctx *resolutionContext, army models.Army) int {
	if ctx.famished[army.ID] || ctx.balance.NobleCommandBonus == 0 {
		return 0
	}
	for _, nobleID := range ctx.noblesAt(army.TerritoryID) {
		noble := ctx.noblesByID[nobleID]
		if noble != nil && noble.OwnerID == army.OwnerID && noble.Status == models.NobleStatusFree {
			return ctx.balance.NobleCommandBonus
		}
	}
	return 0
}

func (ctx *resolutionContext) castleOwnedByAllAttackers(territoryID models.TerritoryID) bool {
	state := ctx.state.TerritoryStates[territoryID]
	if state.OwnerID == nil {
		return false
	}
	owner := *state.OwnerID
	hasAttacker := false
	for _, attack := range ctx.attacks {
		if attack.target != territoryID {
			continue
		}
		hasAttacker = true
		attacker := ctx.startArmiesByID[attack.armyID]
		if attacker.OwnerID != owner {
			return false
		}
	}
	return hasAttacker
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

func (ctx *resolutionContext) clearSupportsForVoidedAttacks(voided map[models.ArmyID]bool) {
	for supportID, support := range ctx.supports {
		if voided[supportID] {
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
			SupporterIDs:    append([]models.ArmyID(nil), result.supporterIDs...),
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
		if ctx.cancelledPeaceful[armyID] {
			record.fail("attacked_origin")
			if record.pendingDisperse {
				ctx.clearPendingDisperse(record)
			}
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
