package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type combatTable map[models.TerritoryID][]combatEntry

type combatEntry struct {
	armyID  models.ArmyID
	ownerID models.PlayerID
	force   int
	noHelp  int
	move    bool
}

type freedOrigin struct {
	headToHeadLoser models.ArmyID
}

func resolveContests(ctx *resolutionContext) error {
	if len(ctx.attacks) == 0 {
		ctx.contest = contestState{
			active:           map[models.ArmyID]bool{},
			dislodged:        map[models.ArmyID]bool{},
			vacated:          map[models.ArmyID]bool{},
			disperseResidual: map[models.ArmyID]int{},
			ghosted:          map[models.ArmyID]bool{},
			retiredGhosts:    map[models.ArmyID]bool{},
			cuts:             map[models.ArmyID]bool{},
			voidedSupports:   map[models.ArmyID]bool{},
			results:          map[models.TerritoryID]contestResult{},
		}
		return nil
	}

	peacefulVacated := initialPeacefulVacated(ctx)
	disperseResidual := map[models.ArmyID]int{}
	lockedDislodged := map[models.ArmyID]bool{}
	ghosted := map[models.ArmyID]bool{}
	retiredGhosts := map[models.ArmyID]bool{}
	seedVoidedSupports := map[models.ArmyID]bool{}
	var previous contestState
	hasPrevious := false
	moveOrderCount := len(ctx.attacks) + len(ctx.joins) + len(ctx.disperses)
	maxPasses := 4*(moveOrderCount+len(ctx.startArmiesByID)) + 10
	// Re-run combat after peaceful departures change origin defense. The inner
	// table solver handles attack bounces and dislodgements for each pass.
	for pass := 0; ; pass++ {
		current, err := resolveAttackTable(ctx, peacefulVacated, disperseResidual, lockedDislodged, ghosted, retiredGhosts, seedVoidedSupports)
		if err != nil {
			return err
		}
		effectiveVacated := mergeBooleanMaps(peacefulVacated, current.vacated)
		peaceful := ctx.predictPeacefulDepartures(current, effectiveVacated, disperseResidual)
		next := current
		next.vacated = mergeBooleanMaps(current.vacated, peaceful.vacated)
		next.disperseResidual = peaceful.disperseResidual
		next.dislodged = mergeBooleanMaps(current.dislodged, lockedDislodged)
		if hasPrevious && sameContestState(previous, next) {
			ctx.contest = next
			break
		}
		if pass >= maxPasses {
			return fmt.Errorf("engine: combat resolution did not converge after %d passes", maxPasses+1)
		}
		previous = next
		hasPrevious = true
		peacefulVacated = peaceful.vacated
		disperseResidual = peaceful.disperseResidual
		lockedDislodged = next.dislodged
		ghosted = next.ghosted
		retiredGhosts = next.retiredGhosts
		seedVoidedSupports = persistentVoidedSupports(ctx, next.voidedSupports, next.ghosted, next.retiredGhosts)
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
	ctx.clearSupportsForVoidedAttacks(ctx.contest.voidedSupports)
	ctx.emitCombatEvents()
	ctx.applyContestOutcomes()
	return nil
}

func initialPeacefulVacated(ctx *resolutionContext) map[models.ArmyID]bool {
	vacated := make(map[models.ArmyID]bool, len(ctx.joins)+len(ctx.disperses))
	for armyID := range ctx.joins {
		vacated[armyID] = true
	}
	for armyID := range ctx.disperses {
		vacated[armyID] = true
	}
	return vacated
}

func resolveAttackTable(
	ctx *resolutionContext,
	peacefulVacated map[models.ArmyID]bool,
	disperseResidual map[models.ArmyID]int,
	lockedDislodged map[models.ArmyID]bool,
	initialGhosted map[models.ArmyID]bool,
	initialRetiredGhosts map[models.ArmyID]bool,
	initialVoidedSupports map[models.ArmyID]bool,
) (contestState, error) {
	bounced := copyBooleanMap(initialGhosted)
	dislodged := copyBooleanMap(lockedDislodged)
	retiredGhosts := copyBooleanMap(initialRetiredGhosts)
	excluded := ctx.alliedDestinationAttacks(peacefulVacated, lockedDislodged)
	cuts := ctx.calculateSupportCuts(excluded)
	voidedSupports := copyBooleanMap(initialVoidedSupports)
	selfBounced := make(map[models.ArmyID]bool)
	freedOrigins := make(map[models.TerritoryID]freedOrigin)
	maxPasses := 4*(len(ctx.attacks)+len(ctx.startArmiesByID)) + 4

	var table combatTable
	converged := false
	for pass := 0; pass < maxPasses; pass++ {
		table = buildCombatTable(ctx, peacefulVacated, disperseResidual, bounced, selfBounced, dislodged, retiredGhosts, cuts, voidedSupports)
		changed := markSwapBounces(ctx, table, bounced, dislodged, voidedSupports)
		if changed {
			continue
		}
		changed = markSelfDislodgeBounces(ctx, table, peacefulVacated, bounced, selfBounced, dislodged, voidedSupports)
		if changed {
			continue
		}
		changed = markOutgunnedBounces(table, bounced, dislodged)
		if changed {
			continue
		}
		changed = markDislodgements(ctx, table, peacefulVacated, bounced, dislodged, retiredGhosts, cuts, freedOrigins)
		changed = unbounceSelfDislodgedTargets(ctx, selfBounced, bounced, dislodged, voidedSupports) || changed
		changed = unbounceFreedOrigins(ctx, table, freedOrigins, bounced, dislodged) || changed
		if changed {
			continue
		}
		converged = true
		break
	}
	if !converged {
		return contestState{}, fmt.Errorf("engine: combat table did not converge after %d passes", maxPasses)
	}

	table = buildCombatTable(ctx, peacefulVacated, disperseResidual, bounced, selfBounced, dislodged, retiredGhosts, cuts, voidedSupports)
	results := buildCombatResults(ctx, table, peacefulVacated, disperseResidual, bounced, dislodged, retiredGhosts, cuts, voidedSupports)
	vacated := make(map[models.ArmyID]bool)
	for _, result := range results {
		if result.winnerID == "" || dislodged[result.winnerID] {
			continue
		}
		vacated[result.winnerID] = true
	}
	active := make(map[models.ArmyID]bool, len(ctx.attacks))
	for armyID := range ctx.attacks {
		active[armyID] = !dislodged[armyID]
	}
	persistentGhosts := make(map[models.ArmyID]bool)
	for armyID := range bounced {
		if dislodged[armyID] && !retiredGhosts[armyID] {
			persistentGhosts[armyID] = true
		}
	}
	return contestState{
		active:           active,
		dislodged:        dislodged,
		vacated:          vacated,
		disperseResidual: map[models.ArmyID]int{},
		ghosted:          persistentGhosts,
		retiredGhosts:    retiredGhosts,
		cuts:             cuts,
		voidedSupports:   voidedSupports,
		results:          results,
	}, nil
}

func persistentVoidedSupports(ctx *resolutionContext, voidedSupports, ghosted, retiredGhosts map[models.ArmyID]bool) map[models.ArmyID]bool {
	persistent := make(map[models.ArmyID]bool)
	for supportID := range voidedSupports {
		support := ctx.supports[supportID]
		if support != nil && (ghosted[support.targetArmyID] || retiredGhosts[support.targetArmyID]) {
			persistent[supportID] = true
		}
	}
	return persistent
}

func buildCombatTable(
	ctx *resolutionContext,
	peacefulVacated map[models.ArmyID]bool,
	disperseResidual map[models.ArmyID]int,
	bounced, selfBounced, dislodged, retiredGhosts, cuts, voidedSupports map[models.ArmyID]bool,
) combatTable {
	targets := make(map[models.TerritoryID]bool)
	for _, attack := range ctx.attacks {
		targets[attack.target] = true
	}
	for armyID := range bounced {
		if attack := ctx.attacks[armyID]; attack != nil {
			targets[attack.source] = true
		}
	}
	table := make(combatTable, len(targets))
	for _, territoryID := range sortedTerritoryMap(targets) {
		defender := ctx.startArmyAt(territoryID)
		if defender != nil && armyDefendsOrigin(ctx, defender.ID, peacefulVacated, bounced, dislodged) {
			_, defense := defenseStrengthAt(ctx, defender, territoryID, peacefulVacated, disperseResidual, dislodged, cuts)
			table[territoryID] = append(table[territoryID], combatEntry{
				armyID:  defender.ID,
				ownerID: defender.OwnerID,
				force:   defense,
				move:    false,
			})
		} else if ctx.hasCastle(territoryID) {
			table[territoryID] = append(table[territoryID], combatEntry{
				force: ctx.balance.CastleDefenseBonus,
			})
		}
		for _, armyID := range sortedArmyMap(ctx.attacks) {
			attack := ctx.attacks[armyID]
			if attack.target != territoryID || (dislodged[armyID] && (!bounced[armyID] || retiredGhosts[armyID])) {
				continue
			}
			defenderOwnerID := models.PlayerID("")
			if defender != nil && !peacefulVacated[defender.ID] && !dislodged[defender.ID] {
				defenderOwnerID = defender.OwnerID
			}
			strength, noHelp := attackStrengthAt(ctx, attack, defenderOwnerID, cuts, dislodged)
			if bounced[armyID] {
				if retiredGhosts[armyID] {
					continue
				}
				// A bounced move still contests its destination while its origin
				// entry protects the bounce chain, as in Diplomacy's combat table.
				table[territoryID] = append(table[territoryID], combatEntry{armyID: armyID, ownerID: ctx.startArmiesByID[armyID].OwnerID, force: strength, noHelp: noHelp})
				continue
			}
			table[territoryID] = append(table[territoryID], combatEntry{
				armyID:  armyID,
				ownerID: ctx.startArmiesByID[armyID].OwnerID,
				force:   strength,
				noHelp:  noHelp,
				move:    true,
			})
		}
		if len(table[territoryID]) == 0 {
			delete(table, territoryID)
		}
	}
	return table
}

func armyDefendsOrigin(ctx *resolutionContext, armyID models.ArmyID, peacefulVacated, bounced, dislodged map[models.ArmyID]bool) bool {
	if dislodged[armyID] || peacefulVacated[armyID] {
		return false
	}
	if _, attacking := ctx.attacks[armyID]; attacking {
		return bounced[armyID]
	}
	return true
}

func defenseStrengthAt(
	ctx *resolutionContext,
	defender *models.Army,
	territoryID models.TerritoryID,
	peacefulVacated map[models.ArmyID]bool,
	disperseResidual map[models.ArmyID]int,
	dislodged, cuts map[models.ArmyID]bool,
) (int, int) {
	base := 0
	if ctx.hasCastle(territoryID) {
		base = ctx.balance.CastleDefenseBonus
	}
	strength := base
	defenderStrength := defender.Size
	if residual, exists := disperseResidual[defender.ID]; exists {
		defenderStrength = residual
	}
	if !ctx.famished[defender.ID] {
		base += defenderStrength
		strength += defenderStrength
	}
	bonus := nobleCommandBonus(ctx, *defender)
	base += bonus
	strength += bonus
	if record := ctx.records[defender.ID]; record != nil && holdsForDefense(record.order.Type) && !peacefulVacated[defender.ID] {
		defenseSupport := ctx.defensiveSupportStrength(defender.ID, cuts, dislodged)
		strength += defenseSupport
	}
	return base, strength
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

func attackStrengthAt(ctx *resolutionContext, attack *attackIntent, defenderOwnerID models.PlayerID, cuts, dislodged map[models.ArmyID]bool) (int, int) {
	attacker := ctx.startArmiesByID[attack.armyID]
	strength := attack.size + nobleCommandBonus(ctx, attacker)
	noHelp := 0
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if !support.applies || !support.offensive || support.targetArmyID != attack.armyID || cuts[supportID] || dislodged[supportID] || ctx.famished[supportID] {
			continue
		}
		supporter := ctx.startArmiesByID[supportID]
		supportStrength := supporter.Size + nobleCommandBonus(ctx, supporter)
		strength += supportStrength
		if defenderOwnerID != "" && supporter.OwnerID == defenderOwnerID {
			noHelp += supportStrength
		}
	}
	return strength, noHelp
}

func markSwapBounces(ctx *resolutionContext, table combatTable, bounced, dislodged, voidedSupports map[models.ArmyID]bool) bool {
	changed := false
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		attack := ctx.attacks[armyID]
		otherID, exists := ctx.startArmyAtTerritory[attack.target]
		other := ctx.attacks[otherID]
		if !exists || other == nil || other.target != attack.source || !lessArmyID(armyID, otherID) || bounced[armyID] || bounced[otherID] || dislodged[armyID] || dislodged[otherID] {
			continue
		}
		ourForce, ourNoHelp, ourOK := moveInfo(table, attack.target, armyID)
		theirForce, theirNoHelp, theirOK := moveInfo(table, other.target, otherID)
		if !ourOK || !theirOK {
			continue
		}
		ourOwner := ctx.startArmiesByID[armyID].OwnerID
		theirOwner := ctx.startArmiesByID[otherID].OwnerID
		effectiveOurForce := ourForce - ourNoHelp
		effectiveTheirForce := theirForce - theirNoHelp
		switch {
		case ourOwner == theirOwner || effectiveOurForce == effectiveTheirForce:
			bounced[armyID] = true
			bounced[otherID] = true
			voidNoHelpSupports(ctx, armyID, attack.target, voidedSupports)
			voidNoHelpSupports(ctx, otherID, other.target, voidedSupports)
			changed = true
		case effectiveOurForce < effectiveTheirForce:
			bounced[armyID] = true
			voidNoHelpSupports(ctx, armyID, attack.target, voidedSupports)
			changed = true
		default:
			bounced[otherID] = true
			voidNoHelpSupports(ctx, otherID, other.target, voidedSupports)
			changed = true
		}
	}
	return changed
}

func voidNoHelpSupports(ctx *resolutionContext, armyID models.ArmyID, destination models.TerritoryID, voidedSupports map[models.ArmyID]bool) {
	defender := ctx.startArmyAt(destination)
	if defender == nil {
		return
	}
	for supportID, support := range ctx.supports {
		if support.offensive && support.targetArmyID == armyID && ctx.startArmiesByID[supportID].OwnerID == defender.OwnerID {
			voidedSupports[supportID] = true
		}
	}
}

func markOutgunnedBounces(table combatTable, bounced, dislodged map[models.ArmyID]bool) bool {
	changed := false
	for _, entries := range table {
		maxForce, topCount := topForce(entries)
		for _, entry := range entries {
			if !entry.move || bounced[entry.armyID] || dislodged[entry.armyID] || (entry.force == maxForce && topCount == 1) {
				continue
			}
			bounced[entry.armyID] = true
			changed = true
		}
	}
	return changed
}

func markSelfDislodgeBounces(ctx *resolutionContext, table combatTable, peacefulVacated map[models.ArmyID]bool, bounced, selfBounced, dislodged, voidedSupports map[models.ArmyID]bool) bool {
	changed := false
	for territoryID, entries := range table {
		maxForce, topCount := topForce(entries)
		if topCount != 1 {
			continue
		}
		winner, found := uniqueMoveAt(entries, maxForce)
		if !found {
			continue
		}
		defender := ctx.startArmyAt(territoryID)
		if defender == nil || peacefulVacated[defender.ID] || dislodged[defender.ID] || !armyDefendsOrigin(ctx, defender.ID, peacefulVacated, bounced, dislodged) {
			continue
		}
		secondForce := secondForce(entries, winner.armyID)
		if defender.OwnerID != winner.ownerID && winner.force-winner.noHelp > secondForce {
			continue
		}
		bounced[winner.armyID] = true
		selfBounced[winner.armyID] = true
		for supportID, support := range ctx.supports {
			if support.offensive && support.targetArmyID == winner.armyID {
				voidedSupports[supportID] = true
			}
		}
		changed = true
	}
	return changed
}

func unbounceSelfDislodgedTargets(ctx *resolutionContext, selfBounced, bounced, dislodged, voidedSupports map[models.ArmyID]bool) bool {
	changed := false
	for armyID := range selfBounced {
		if !bounced[armyID] {
			delete(selfBounced, armyID)
			continue
		}
		attack := ctx.attacks[armyID]
		if attack == nil {
			continue
		}
		defender := ctx.startArmyAt(attack.target)
		if defender == nil || !dislodged[defender.ID] {
			continue
		}
		delete(bounced, armyID)
		delete(selfBounced, armyID)
		for supportID, support := range ctx.supports {
			if support.offensive && support.targetArmyID == armyID {
				delete(voidedSupports, supportID)
			}
		}
		changed = true
	}
	return changed
}

func markDislodgements(ctx *resolutionContext, table combatTable, peacefulVacated map[models.ArmyID]bool, bounced, dislodged, retiredGhosts, cuts map[models.ArmyID]bool, freedOrigins map[models.TerritoryID]freedOrigin) bool {
	changed := false
	for territoryID, entries := range table {
		maxForce, topCount := topForce(entries)
		if topCount != 1 {
			continue
		}
		winner, found := uniqueMoveAt(entries, maxForce)
		if !found {
			continue
		}
		defender := ctx.startArmyAt(territoryID)
		if defender == nil || peacefulVacated[defender.ID] || dislodged[defender.ID] || !armyDefendsOrigin(ctx, defender.ID, peacefulVacated, bounced, dislodged) {
			continue
		}
		dislodged[defender.ID] = true
		loserID := models.ArmyID("")
		if loserAttack := ctx.attacks[defender.ID]; loserAttack != nil && loserAttack.target == ctx.attacks[winner.armyID].source {
			loserID = defender.ID
			retiredGhosts[loserID] = true
		}
		freedOrigins[ctx.attacks[winner.armyID].source] = freedOrigin{headToHeadLoser: loserID}
		changed = true
		if support := ctx.supports[defender.ID]; support != nil {
			cuts[defender.ID] = true
		}
		_ = winner
	}
	return changed
}

func unbounceFreedOrigins(ctx *resolutionContext, table combatTable, freedOrigins map[models.TerritoryID]freedOrigin, bounced, dislodged map[models.ArmyID]bool) bool {
	changed := false
	for originID, freed := range freedOrigins {
		entries := table[originID]
		filtered := make([]combatEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.armyID != freed.headToHeadLoser {
				filtered = append(filtered, entry)
			}
		}
		maxForce, topCount := topForce(filtered)
		if topCount != 1 {
			delete(freedOrigins, originID)
			continue
		}
		winner, found := uniqueBouncedAt(filtered, maxForce, bounced, dislodged)
		if found && bounced[winner.armyID] {
			delete(bounced, winner.armyID)
			changed = true
		}
		delete(freedOrigins, originID)
	}
	return changed
}

func uniqueBouncedAt(entries []combatEntry, force int, bounced, dislodged map[models.ArmyID]bool) (combatEntry, bool) {
	var winner combatEntry
	found := false
	for _, entry := range entries {
		if entry.force != force || !bounced[entry.armyID] || dislodged[entry.armyID] {
			continue
		}
		if found {
			return combatEntry{}, false
		}
		winner = entry
		found = true
	}
	return winner, found
}

func moveInfo(table combatTable, territoryID models.TerritoryID, armyID models.ArmyID) (int, int, bool) {
	for _, entry := range table[territoryID] {
		if entry.move && entry.armyID == armyID {
			return entry.force, entry.noHelp, true
		}
	}
	return 0, 0, false
}

func topForce(entries []combatEntry) (int, int) {
	maxForce := -1
	topCount := 0
	for _, entry := range entries {
		if entry.force > maxForce {
			maxForce = entry.force
			topCount = 1
		} else if entry.force == maxForce {
			topCount++
		}
	}
	return maxForce, topCount
}

func uniqueMoveAt(entries []combatEntry, force int) (combatEntry, bool) {
	var winner combatEntry
	found := false
	for _, entry := range entries {
		if entry.force != force || !entry.move {
			continue
		}
		if found {
			return combatEntry{}, false
		}
		winner = entry
		found = true
	}
	return winner, found
}

func secondForce(entries []combatEntry, excluded models.ArmyID) int {
	second := -1
	for _, entry := range entries {
		if entry.armyID == excluded || entry.force <= second {
			continue
		}
		second = entry.force
	}
	return second
}

func buildCombatResults(ctx *resolutionContext, table combatTable, peacefulVacated map[models.ArmyID]bool, disperseResidual map[models.ArmyID]int, bounced, dislodged, retiredGhosts, cuts, voidedSupports map[models.ArmyID]bool) map[models.TerritoryID]contestResult {
	results := make(map[models.TerritoryID]contestResult)
	targets := make(map[models.TerritoryID]bool)
	for _, attack := range ctx.attacks {
		targets[attack.target] = true
	}
	for _, territoryID := range sortedTerritoryMap(targets) {
		defender := ctx.startArmyAt(territoryID)
		castleBonus := 0
		if ctx.hasCastle(territoryID) {
			castleBonus = ctx.balance.CastleDefenseBonus
		}
		baseDefense := castleBonus
		defense := castleBonus
		defenderID := models.ArmyID("")
		defenderOwnerID := models.PlayerID("")
		defenderNobleBonus := 0
		if defender != nil && !peacefulVacated[defender.ID] && !dislodged[defender.ID] {
			defenderOwnerID = defender.OwnerID
		}
		defenderPresent := defender != nil && !peacefulVacated[defender.ID] && (armyDefendsOrigin(ctx, defender.ID, peacefulVacated, bounced, dislodged) || dislodged[defender.ID])
		if defenderPresent {
			defenderID = defender.ID
			defenderOwnerID = defender.OwnerID
			base, total := defenseStrengthAt(ctx, defender, territoryID, peacefulVacated, disperseResidual, dislodged, cuts)
			baseDefense = base
			defense = total
			defenderNobleBonus = nobleCommandBonus(ctx, *defender)
		}
		result := contestResult{
			territoryID: territoryID,
			defenderID:  defenderID,
			baseDefense: baseDefense,
			defense:     defense,
			castleBonus: castleBonus,
		}
		if defenderID != "" || castleBonus > 0 {
			result.contenders = append(result.contenders, CombatContender{
				ArmyID: defenderID, OwnerID: defenderOwnerID, Force: defense,
				NobleBonus: defenderNobleBonus, Defender: true,
			})
		}
		for _, armyID := range sortedArmyMap(ctx.attacks) {
			attack := ctx.attacks[armyID]
			if attack.target != territoryID || retiredGhosts[armyID] {
				continue
			}
			owner := defenderOwnerID
			force, _ := attackStrengthAt(ctx, attack, owner, cuts, dislodged)
			result.contenders = append(result.contenders, CombatContender{
				ArmyID: armyID, OwnerID: ctx.startArmiesByID[armyID].OwnerID,
				Force: force, NobleBonus: nobleCommandBonus(ctx, ctx.startArmiesByID[armyID]),
			})
		}
		entries := table[territoryID]
		maxForce, topCount := topForce(entries)
		if topCount == 1 {
			if winner, found := uniqueMoveAt(entries, maxForce); found && !bounced[winner.armyID] && !dislodged[winner.armyID] {
				result.winnerID = winner.armyID
				if defenderID != "" {
					result.dislodgedArmyID = defenderID
					result.attackerOriginID = ctx.attacks[winner.armyID].source
				}
			}
		}
		if result.winnerID == "" && result.hasAttackStandoff() {
			result.standoff = true
		}
		for _, supportID := range sortedArmyMap(ctx.supports) {
			support := ctx.supports[supportID]
			if cuts[supportID] && supportRelevantToTerritory(ctx, support, territoryID) {
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

func (ctx *resolutionContext) clearSupportsForVoidedAttacks(voided map[models.ArmyID]bool) {
	for supportID, support := range ctx.supports {
		if voided[supportID] {
			support.applies = false
		}
	}
}
