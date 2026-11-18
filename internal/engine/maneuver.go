package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// The maneuver order (M) splits an army into groups sent to adjacent
// territories, or kept on the origin. It never takes part in the combat: it is
// planned when the turn starts (the troops that stay are the army's strength
// for the whole turn) and applied once combats and retreats are settled.

// maneuverGroup is the troops one maneuver sends to one territory.
type maneuverGroup struct {
	target models.TerritoryID
	size   int
	nobles []models.NobleID
}

// maneuverIntent is the plan of one maneuvering army.
type maneuverIntent struct {
	armyID models.ArmyID
	source models.TerritoryID
	// groups lists the groups that leave the origin, in order of appearance.
	groups []*maneuverGroup
	// remaining is the troops that stay on the origin: the unassigned troops
	// and the groups sent to the origin itself.
	remaining int
	// stayingNobles are the nobles planned to stay on the origin.
	stayingNobles map[models.NobleID]bool
	// plan is the split of the order (see PlanManeuver): its LastTarget is the
	// group that carries the chain, its Entry* slices mirror the destinations
	// of targets.
	plan ManeuverPlan
	// targets are the destinations of the stored order and blocked marks the
	// ones the weather removed.
	targets []models.TerritoryID
	blocked []bool
}

func (ctx *resolutionContext) enumerateManeuver(record *orderRecord, army models.Army, sourceAffected bool) {
	order := record.order
	if sourceAffected {
		ctx.emitBadWeatherBlocked(army, order, regionForTerritory(ctx, army.TerritoryID))
		record.invalidate("bad_weather")
		return
	}
	blocked := make([]bool, len(order.TargetIDs))
	liveEntries := 0
	blockedRegion := models.TerritoryID("")
	for index, targetID := range order.TargetIDs {
		if ctx.badWeatherRegions[regionForTerritory(ctx, targetID)] {
			blocked[index] = true
			if blockedRegion == "" {
				blockedRegion = regionForTerritory(ctx, targetID)
			}
			continue
		}
		liveEntries++
	}
	if liveEntries == 0 {
		ctx.emitBadWeatherBlocked(army, order, blockedRegion)
		record.invalidate("bad_weather")
		return
	}
	plan := PlanManeuver(order, army.Size, blocked, noblesCodesAt(ctx.state.Nobles, army.TerritoryID))
	if plan.Invalid {
		record.invalidate("invalid_maneuver_noble_assignment")
		return
	}
	toIDs := func(codes []models.NobleCode) []models.NobleID {
		ids := make([]models.NobleID, len(codes))
		for index, code := range codes {
			ids[index] = ctx.noblesByCode[code]
		}
		return ids
	}
	intent := &maneuverIntent{
		armyID:        army.ID,
		source:        army.TerritoryID,
		remaining:     plan.Stay.Troops,
		stayingNobles: make(map[models.NobleID]bool),
		plan:          plan,
		targets:       order.TargetIDs,
		blocked:       blocked,
	}
	for _, nobleID := range toIDs(plan.Stay.Nobles) {
		intent.stayingNobles[nobleID] = true
	}
	for _, leg := range plan.Groups {
		intent.groups = append(intent.groups, &maneuverGroup{target: leg.Target, size: leg.Troops, nobles: toIDs(leg.Nobles)})
	}
	ctx.maneuvers[army.ID] = intent
}

// maneuverGroupRef points at one group of one maneuver.
type maneuverGroupRef struct {
	intent *maneuverIntent
	group  *maneuverGroup
	owner  models.PlayerID
}

// maneuverChainCandidate is a chain that may carry on from an army after
// maneuver groups arrived on it.
type maneuverChainCandidate struct {
	chain        *models.Chain
	hasRemaining bool
	fromHost     bool
}

// applyManeuvers settles every maneuver at the end of the turn, from the
// positions after combats and retreats.
func applyManeuvers(ctx *resolutionContext) error {
	if len(ctx.maneuvers) == 0 {
		return nil
	}
	live := make(map[models.ArmyID]models.Army, len(ctx.state.Armies))
	for _, army := range ctx.state.Armies {
		live[army.ID] = army
	}

	active := make(map[models.ArmyID]*maneuverIntent, len(ctx.maneuvers))
	for _, armyID := range sortedArmyMap(ctx.maneuvers) {
		record := ctx.records[armyID]
		if record == nil || record.outcome != "" {
			continue
		}
		if _, exists := live[armyID]; !exists {
			record.fail("army_destroyed")
			continue
		}
		active[armyID] = ctx.maneuvers[armyID]
	}

	activeIDs := sortedArmyMap(active)
	if len(activeIDs) == 0 {
		ctx.maneuvers = make(map[models.ArmyID]*maneuverIntent)
		return nil
	}
	var refs []*maneuverGroupRef
	for _, armyID := range activeIDs {
		intent := active[armyID]
		for _, group := range intent.groups {
			refs = append(refs, &maneuverGroupRef{intent: intent, group: group, owner: live[armyID].OwnerID})
		}
	}

	failed := make(map[*maneuverGroup]bool)
	for _, ref := range refs {
		if ref.group.size == 0 {
			failed[ref.group] = true
		}
	}
	// Troops that merged into a maneuvering army earlier in the turn (a
	// join, a dispersion, a retreat) stay on the origin with its remainder.
	arrived := make(map[*maneuverIntent]int, len(active))
	for _, armyID := range activeIDs {
		arrived[active[armyID]] = max(0, live[armyID].Size-ctx.startArmiesByID[armyID].Size)
	}
	stay := func(intent *maneuverIntent) int {
		total := intent.remaining + arrived[intent]
		for _, group := range intent.groups {
			if failed[group] {
				total += group.size
			}
		}
		return total
	}
	// The armies do not move while the plan settles: index them once (a
	// territory holds at most one army).
	armyAtTerritory := make(map[models.TerritoryID]models.ArmyID, len(live))
	for armyID, army := range live {
		armyAtTerritory[army.TerritoryID] = armyID
	}
	occupantAt := func(territoryID models.TerritoryID) (models.Army, bool) {
		armyID, exists := armyAtTerritory[territoryID]
		if !exists {
			return models.Army{}, false
		}
		if intent := active[armyID]; intent != nil && stay(intent) == 0 {
			return models.Army{}, false
		}
		return live[armyID], true
	}
	claimantsByTarget := func() map[models.TerritoryID][]*maneuverGroupRef {
		byTarget := make(map[models.TerritoryID][]*maneuverGroupRef)
		for _, ref := range refs {
			if !failed[ref.group] {
				byTarget[ref.group.target] = append(byTarget[ref.group.target], ref)
			}
		}
		return byTarget
	}
	// A group fails when its destination holds an enemy, is contested, or is
	// claimed by groups of several owners. A failed group returns to its
	// origin, which can stop that origin from being vacated: iterate until
	// the plan is stable.
	for changed := true; changed; {
		changed = false
		byTarget := claimantsByTarget()
		for _, targetID := range sortedTerritoryMap(byTarget) {
			claimants := byTarget[targetID]
			fail := func(ref *maneuverGroupRef) {
				failed[ref.group] = true
				changed = true
			}
			if occupant, occupied := occupantAt(targetID); occupied {
				for _, ref := range claimants {
					if ref.owner != occupant.OwnerID {
						fail(ref)
					}
				}
				continue
			}
			contested := false
			if result, exists := ctx.contest.results[targetID]; exists && result.winnerID == "" {
				contested = true
			}
			owners := make(map[models.PlayerID]bool)
			for _, ref := range claimants {
				owners[ref.owner] = true
			}
			if contested || len(owners) > 1 {
				for _, ref := range claimants {
					fail(ref)
				}
			}
		}
	}

	// Departures: each maneuvering army keeps the troops that stay.
	templates := make(map[models.ArmyID]models.Army, len(active))
	reusableIDs := make(map[models.ArmyID]bool)
	for _, armyID := range activeIDs {
		intent := active[armyID]
		army := live[armyID]
		templates[armyID] = army
		size := stay(intent)
		if size == 0 {
			delete(live, armyID)
			reusableIDs[armyID] = true
			continue
		}
		army.Size = size
		live[armyID] = army
	}

	// Arrivals, by destination.
	resulting := make(map[*maneuverGroup]models.ArmyID)
	byTarget := claimantsByTarget()
	for _, targetID := range sortedTerritoryMap(byTarget) {
		claimants := byTarget[targetID]
		total := 0
		for _, ref := range claimants {
			total += ref.group.size
		}
		var hostID models.ArmyID
		if armyID, occupied := armyAtTerritory[targetID]; occupied {
			// An ID freed by a departure may already name a new army elsewhere.
			if army, remains := live[armyID]; remains && army.TerritoryID == targetID {
				hostID = armyID
			}
		}
		if hostID != "" {
			host := live[hostID]
			host.Size += total
			live[hostID] = host
			ctx.events = append(ctx.events, Event{
				Type:        EventTypeFusion,
				Phase:       4,
				ArmyID:      hostID,
				ArmyIDs:     []models.ArmyID{hostID},
				TerritoryID: targetID,
				OrderType:   models.OrderTypeManeuver,
				Outcome:     OutcomeSuccess,
				Reason:      "maneuver_friendly_fusion",
			})
		} else {
			source := claimants[0]
			newID := models.ArmyID("")
			for _, ref := range claimants {
				if reusableIDs[ref.intent.armyID] {
					source, newID = ref, ref.intent.armyID
					delete(reusableIDs, newID)
					break
				}
			}
			if newID == "" {
				newID = ctx.allocateArmyID()
				if ctx.famished[source.intent.armyID] {
					ctx.famished[newID] = true
				}
				if ctx.plagueStruckArmies[source.intent.armyID] {
					// A group is not struck a second time by the plague its army
					// already suffered this turn.
					ctx.markPlagueStruckArmy(newID)
				}
			}
			created := templates[source.intent.armyID]
			created.ID = newID
			created.TerritoryID = targetID
			created.Size = total
			created.ChainID = nil
			if origin, reused := templates[newID]; reused {
				// The army keeps its identity, and so its chain.
				created.ChainID = origin.ChainID
			}
			live[newID] = created
			hostID = newID
		}
		for _, ref := range claimants {
			resulting[ref.group] = hostID
			ctx.events = append(ctx.events, Event{
				Type:          EventTypeMovement,
				Phase:         4,
				ArmyID:        hostID,
				SourceID:      ref.intent.source,
				DestinationID: targetID,
				OrderType:     models.OrderTypeManeuver,
				Outcome:       OutcomeSuccess,
			})
		}
	}

	// Nobles follow their group; those of a group that did not arrive stay
	// with the troops left on the origin. A noble cannot stay alone: when no
	// troop stays, the nobles still on the origin follow the group that
	// carries the chain, or the last group that arrived.
	vacated := make(map[models.ArmyID][]models.NobleID)
	for _, armyID := range activeIDs {
		if intent := active[armyID]; stay(intent) == 0 {
			vacated[armyID] = ctx.noblesAt(intent.source)
		}
	}
	for _, armyID := range activeIDs {
		intent := active[armyID]
		var follow *maneuverGroup
		for _, group := range intent.groups {
			destinationArmy, arrived := resulting[group]
			if !arrived {
				continue
			}
			ctx.moveNobles(group.nobles, group.target, destinationArmy)
			if follow == nil || follow.target != intent.plan.LastTarget {
				follow = group
			}
		}
		if follow == nil {
			continue
		}
		for _, nobleID := range vacated[armyID] {
			if noble := ctx.noblesByID[nobleID]; noble != nil && noble.LocationID == intent.source {
				ctx.moveNobles([]models.NobleID{nobleID}, follow.target, resulting[follow])
			}
		}
	}

	// Outcomes and chains.
	candidates := make(map[models.ArmyID][]maneuverChainCandidate)
	leaving := make(map[models.ChainID]bool)
	for _, armyID := range activeIDs {
		intent := active[armyID]
		record := ctx.records[armyID]
		chain := ctx.chainsByID[record.chainID]
		groupByTarget := make(map[models.TerritoryID]*maneuverGroup, len(intent.groups))
		for _, group := range intent.groups {
			groupByTarget[group.target] = group
		}
		delivered := func(index int) bool {
			if intent.blocked[index] || intent.plan.EntryEmpty[index] {
				return false
			}
			targetID := intent.targets[index]
			if targetID == intent.source {
				return true
			}
			group := groupByTarget[targetID]
			return group != nil && group.size > 0 && !failed[group]
		}
		complete := true
		for index := range intent.targets {
			// A "+" with no troop left to take is satisfied by definition.
			if !delivered(index) && !(intent.plan.EntryRest[index] && intent.plan.EntryEmpty[index] && !intent.blocked[index]) {
				complete = false
			}
		}
		lastDelivered := false
		carrierID := models.ArmyID("")
		if group := groupByTarget[intent.plan.LastTarget]; group != nil {
			if !failed[group] && group.size > 0 {
				lastDelivered, carrierID = true, resulting[group]
			}
		} else if intent.plan.LastTarget == intent.source && !intent.plan.EntryEmpty[intent.plan.LastEntry] && stay(intent) > 0 {
			lastDelivered, carrierID = true, armyID
		}
		loop := record.order.Liaison == models.LiaisonModeLoop

		switch {
		case loop && !complete:
			record.outcome = OutcomeFailure
			record.reason = "maneuver_partial"
			if stay(intent) > 0 && chain != nil {
				reduceManeuverOrder(chain, delivered)
			} else {
				// No troop stays on the origin: nothing is left to carry the
				// chain on.
				record.fused = true
				record.reason = "maneuver_chain_discarded"
			}
			continue
		case !lastDelivered:
			if loop {
				record.outcome = OutcomeSuccess
				record.reason = "maneuver_chain_discarded"
				record.fused = true
			} else {
				record.outcome = OutcomeFailure
				record.reason = "maneuver_failed"
			}
			continue
		}
		record.outcome = OutcomeSuccess
		record.reason = "maneuver_complete"
		if !complete {
			record.reason = "maneuver_partial"
		}
		if carrierID != armyID && chain != nil {
			leaving[chain.ID] = true
			candidates[carrierID] = append(candidates[carrierID], maneuverChainCandidate{
				chain:        chain,
				hasRemaining: chain.CurrentIndex < len(chain.Orders)-1,
			})
		}
	}

	for _, carrierID := range sortedArmyMap(candidates) {
		carrier, exists := live[carrierID]
		if !exists {
			continue
		}
		list := candidates[carrierID]
		// The chain an army already follows competes with the arriving ones,
		// unless it is itself leaving with a group of its own maneuver. A
		// chain that waits on a looped dispersion's residual cannot be
		// replaced.
		pending := ctx.pendingDisperseChainOf(carrierID)
		if pending != nil && carrier.ChainID == nil {
			list = append(list, maneuverChainCandidate{chain: pending, hasRemaining: true, fromHost: true})
		} else if carrier.ChainID != nil && !leaving[*carrier.ChainID] {
			if hostChain := ctx.chainsByID[*carrier.ChainID]; hostChain != nil {
				remaining := hostChain.CurrentIndex < len(hostChain.Orders)-1 ||
					hostChain.CurrentIndex >= 0 && hostChain.CurrentIndex < len(hostChain.Orders) && hostChain.Orders[hostChain.CurrentIndex].Liaison == models.LiaisonModeLoop
				list = append(list, maneuverChainCandidate{chain: hostChain, hasRemaining: remaining, fromHost: true})
			}
		}
		ctx.resolveManeuverChains(live, carrierID, list, pending != nil)
	}

	// Origin armies that handed their chain over no longer hold it.
	for _, armyID := range activeIDs {
		if army, exists := live[armyID]; exists && army.ChainID != nil && !chainBoundTo(ctx, *army.ChainID, armyID) {
			army.ChainID = nil
			live[armyID] = army
		}
	}

	ctx.state.Armies = armiesFromMap(live)
	ctx.maneuvers = make(map[models.ArmyID]*maneuverIntent)
	if err := ctx.rebuildOccupancy(); err != nil {
		return fmt.Errorf("engine: maneuver: %w", err)
	}
	return nil
}

func chainBoundTo(ctx *resolutionContext, chainID models.ChainID, armyID models.ArmyID) bool {
	chain := ctx.chainsByID[chainID]
	return chain != nil && chain.ArmyID == armyID
}

// resolveManeuverChains keeps one chain on the army that maneuver groups
// arrived on. Only chains with orders left for the next turn matter.
func (ctx *resolutionContext) resolveManeuverChains(live map[models.ArmyID]models.Army, carrierID models.ArmyID, candidates []maneuverChainCandidate, hostPending bool) {
	remaining := make([]maneuverChainCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.hasRemaining {
			remaining = append(remaining, candidate)
		}
	}
	turn := ctx.state.Turn
	var winner *maneuverChainCandidate
	switch {
	case hostPending:
		// The pending chain stays; every arriving chain with orders left is
		// abandoned.
		for index := range remaining {
			if remaining[index].fromHost {
				winner = &remaining[index]
			}
		}
	case len(remaining) == 1:
		winner = &remaining[0]
	case len(remaining) > 1:
		sort.SliceStable(remaining, func(i, j int) bool {
			return remaining[i].chain.EffectiveEmittedTurn(turn) > remaining[j].chain.EffectiveEmittedTurn(turn)
		})
		if remaining[0].chain.EffectiveEmittedTurn(turn) != remaining[1].chain.EffectiveEmittedTurn(turn) {
			winner = &remaining[0]
		}
	}
	for _, candidate := range remaining {
		if winner != nil && candidate.chain.ID == winner.chain.ID {
			continue
		}
		ctx.discardManeuverChain(candidate.chain.ID)
	}
	if winner == nil {
		if len(remaining) > 0 {
			carrier := live[carrierID]
			carrier.ChainID = nil
			live[carrierID] = carrier
		}
		return
	}
	if winner.fromHost {
		return
	}
	carrier := live[carrierID]
	chainID := winner.chain.ID
	carrier.ChainID = &chainID
	live[carrierID] = carrier
	rebindChain(winner.chain, carrierID)
}

// reduceManeuverOrder rewrites the looped maneuver a chain retries so that
// it only lists the destinations not carried out yet.
func reduceManeuverOrder(chain *models.Chain, delivered func(int) bool) {
	if chain.CurrentIndex < 0 || chain.CurrentIndex >= len(chain.Orders) {
		return
	}
	order := chain.Orders[chain.CurrentIndex]
	targets := make([]models.TerritoryID, 0, len(order.TargetIDs))
	counts := make([]int, 0, len(order.TargetIDs))
	for index, targetID := range order.TargetIDs {
		if delivered(index) {
			continue
		}
		targets = append(targets, targetID)
		counts = append(counts, order.Counts[index])
	}
	if len(targets) == 0 {
		return
	}
	kept := make(map[models.TerritoryID]bool, len(targets))
	for _, targetID := range targets {
		kept[targetID] = true
	}
	assignments := make(map[models.TerritoryID][]models.NobleCode)
	for targetID, codes := range order.NobleAssignments {
		if kept[targetID] {
			assignments[targetID] = codes
		}
	}
	order.TargetIDs = targets
	order.Counts = counts
	order.NobleAssignments = assignments
	chain.Orders[chain.CurrentIndex] = order
}

// discardManeuverChain makes chain progression remove a chain that lost its
// army to another one, and reports why on its order.
func (ctx *resolutionContext) discardManeuverChain(chainID models.ChainID) {
	for _, armyID := range sortedArmyMap(ctx.records) {
		if record := ctx.records[armyID]; record.chainID == chainID {
			record.reason = "maneuver_chain_discarded"
			record.fused = true
		}
	}
}
