package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// contestState is the settled combat of a turn, read by movement execution
// and chain progression.
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
			active:    map[models.ArmyID]bool{},
			dislodged: map[models.ArmyID]bool{},
			vacated:   map[models.ArmyID]bool{},
			cuts:      map[models.ArmyID]bool{},
			results:   map[models.TerritoryID]contestResult{},
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
	ctx.emitCombatEvents()
	ctx.applyContestOutcomes()
	return nil
}

// settledContest reads the contest from the settled decisions.
func (adj *adjudicator) settledContest() contestState {
	ctx := adj.ctx
	contest := contestState{
		active:    make(map[models.ArmyID]bool, len(ctx.attacks)),
		dislodged: make(map[models.ArmyID]bool),
		vacated:   make(map[models.ArmyID]bool),
		cuts:      make(map[models.ArmyID]bool),
		results:   adj.contestResults(),
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
		if present || !ctx.castleOwnedByAllAttackers(territoryID) {
			result.castleBonus = ctx.fortificationBonus(territoryID)
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
				Supported:  result.defense - result.baseDefense,
				NobleBonus: defenderNobleBonus, Defender: true,
			})
		}
		tunneled := false
		for _, armyID := range adj.attacksByTarget[territoryID] {
			if adj.retiredGhost(armyID) {
				continue
			}
			tunneled = tunneled || ctx.tunneledAttacks[armyID]
			force := adj.attackForce(armyID)
			attacker := ctx.startArmiesByID[armyID]
			result.contenders = append(result.contenders, CombatContender{
				ArmyID: armyID, OwnerID: attacker.OwnerID, Force: force, NobleBonus: nobleCommandBonus(ctx, attacker),
				Supported: force - ctx.attacks[armyID].size - nobleCommandBonus(ctx, attacker),
				Tunnel:    ctx.tunneledAttacks[armyID],
			})
			if adj.moves(armyID) {
				result.winnerID = armyID
				if result.defenderID != "" {
					result.dislodgedArmyID = result.defenderID
					result.attackerOriginID = ctx.attacks[armyID].source
				}
			}
		}
		if tunneled && result.castleBonus > 0 && len(result.contenders) > 0 && result.contenders[0].Defender {
			faced := result.contenders[0].Force - result.castleBonus
			result.contenders[0].TunnelForce = &faced
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

// nobleCommandBonus is the force the free nobles of the army owner add on
// its territory: the command bonus once, plus what their dignities grant
// (D'Arc, specs/dames.md § Dignités). A famished army gets neither.
func nobleCommandBonus(ctx *resolutionContext, army models.Army) int {
	if ctx.famished[army.ID] {
		return 0
	}
	commanded := false
	dignityBonus := 0
	for _, nobleID := range ctx.noblesAt(army.TerritoryID) {
		noble := ctx.noblesByID[nobleID]
		if noble != nil && noble.OwnerID == army.OwnerID && noble.Status == models.NobleStatusFree {
			commanded = true
			dignityBonus += noble.ArmyForceBonus()
		}
	}
	if !commanded {
		return 0
	}
	return ctx.balance.NobleCommandBonus + dignityBonus
}

func (ctx *resolutionContext) castleOwnedByAllAttackers(territoryID models.TerritoryID) bool {
	owner, controlled := ctx.controllerAtStart(territoryID)
	if !controlled {
		return false
	}
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
