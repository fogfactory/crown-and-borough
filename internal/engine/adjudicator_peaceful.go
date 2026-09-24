package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// combatFacts is what the join and dispersion rules read from combat. The
// adjudicator answers with decisions while it resolves the turn
// (decisionFacts); movement execution answers with the settled contest
// (settledFacts).
type combatFacts interface {
	// attackMoves reports whether an army's attack reaches its destination.
	attackMoves(armyID models.ArmyID) bool
	// dislodged reports whether an army is dislodged from its origin.
	dislodged(armyID models.ArmyID) bool
	// attacked reports whether a territory is the destination of an attack
	// that is not an ally's attack on an army staying there.
	attacked(territoryID models.TerritoryID) bool
	// winnerAt returns the attack that wins the combat on a territory.
	winnerAt(territoryID models.TerritoryID) models.ArmyID
	// defenderLeaves reports whether the army starting on a territory leaves
	// it, by moving or by being dislodged.
	defenderLeaves(territoryID models.TerritoryID) bool
}

// settledFacts reads the contest settled by the adjudicator.
type settledFacts struct {
	ctx *resolutionContext
}

func (facts settledFacts) attackMoves(armyID models.ArmyID) bool {
	record := facts.ctx.records[armyID]
	return facts.ctx.attacks[armyID] != nil && record != nil && record.outcome == OutcomeSuccess
}

func (facts settledFacts) dislodged(armyID models.ArmyID) bool {
	return facts.ctx.dislodged[armyID] != nil
}

func (facts settledFacts) attacked(territoryID models.TerritoryID) bool {
	return facts.ctx.attackedTerritories[territoryID]
}

func (facts settledFacts) winnerAt(territoryID models.TerritoryID) models.ArmyID {
	return facts.ctx.contest.results[territoryID].winnerID
}

func (facts settledFacts) defenderLeaves(territoryID models.TerritoryID) bool {
	defender := facts.ctx.startArmyAt(territoryID)
	return defender == nil || facts.ctx.contest.vacated[defender.ID] || facts.ctx.contest.dislodged[defender.ID]
}

// decisionFacts answers the join and dispersion rules with the adjudicator's
// decisions, settled or being searched, while the outcome of subject is being
// adjudicated. The subject is
// taken as not dislodged: a dislodged army cannot leave, so its departure only
// matters when it is not dislodged, and this keeps a departure from depending
// on its own dislodgement. When leaving is set, the other movements also take
// the subject as gone from its origin, which is how the troops a partial
// dispersion leaves behind are measured.
type decisionFacts struct {
	adj     *adjudicator
	subject models.ArmyID
	leaving bool
}

func (facts decisionFacts) attackMoves(armyID models.ArmyID) bool {
	return facts.adj.moves(armyID)
}

// stays reports whether an army stays on its origin for the join and
// dispersion rules.
func (facts decisionFacts) stays(armyID models.ArmyID) bool {
	if facts.leaving && armyID == facts.subject {
		return false
	}
	return facts.adj.stays(armyID)
}

func (facts decisionFacts) dislodged(armyID models.ArmyID) bool {
	return armyID != facts.subject && facts.adj.dislodged(armyID)
}

func (facts decisionFacts) attacked(territoryID models.TerritoryID) bool {
	ctx := facts.adj.ctx
	defender := ctx.startArmyAt(territoryID)
	for _, armyID := range facts.adj.attacksByTarget[territoryID] {
		if defender == nil || ctx.startArmiesByID[armyID].OwnerID != defender.OwnerID {
			return true
		}
		if !facts.stays(defender.ID) || facts.dislodged(defender.ID) {
			return true
		}
	}
	return false
}

func (facts decisionFacts) winnerAt(territoryID models.TerritoryID) models.ArmyID {
	for _, armyID := range facts.adj.attacksByTarget[territoryID] {
		if facts.adj.moves(armyID) {
			return armyID
		}
	}
	return ""
}

func (facts decisionFacts) defenderLeaves(territoryID models.TerritoryID) bool {
	defender := facts.adj.ctx.startArmyAt(territoryID)
	return defender == nil || !facts.stays(defender.ID) || facts.dislodged(defender.ID)
}

// peacefulGroup gathers the joins and dispersions whose origins or
// destinations overlap: the join and dispersion rules of one group never read
// the orders of another, so a departure is adjudicated on its group only.
// Members are keyed by record army, as in ctx.joins and ctx.disperses.
type peacefulGroup struct {
	joins     map[models.ArmyID]bool
	disperses map[models.ArmyID]bool
}

// armies lists the armies whose dislodgement the group's rules read: the army
// of every record and the army executing every dispersion.
func (group *peacefulGroup) armies(ctx *resolutionContext) []models.ArmyID {
	armies := make(map[models.ArmyID]bool)
	for armyID := range group.joins {
		armies[armyID] = true
	}
	for armyID := range group.disperses {
		armies[armyID] = true
		armies[ctx.disperses[armyID].armyID] = true
	}
	return sortedArmyMap(armies)
}

// territories lists the origins and destinations of the group.
func (group *peacefulGroup) territories(ctx *resolutionContext) []models.TerritoryID {
	territories := make(map[models.TerritoryID]bool)
	for armyID := range group.joins {
		territories[ctx.joins[armyID].source] = true
		territories[ctx.joins[armyID].target] = true
	}
	for armyID := range group.disperses {
		disperse := ctx.disperses[armyID]
		territories[disperse.source] = true
		for _, targetID := range disperse.targets {
			territories[targetID] = true
		}
	}
	return sortedTerritoryMap(territories)
}

// peacefulGroups returns the group of every army that leaves its origin with
// a join or a dispersion, keyed by the army on the origin.
func peacefulGroups(ctx *resolutionContext) map[models.ArmyID]*peacefulGroup {
	parent := make(map[models.TerritoryID]models.TerritoryID)
	var find func(models.TerritoryID) models.TerritoryID
	find = func(territoryID models.TerritoryID) models.TerritoryID {
		if parent[territoryID] == "" || parent[territoryID] == territoryID {
			parent[territoryID] = territoryID
			return territoryID
		}
		root := find(parent[territoryID])
		parent[territoryID] = root
		return root
	}
	union := func(source models.TerritoryID, targets []models.TerritoryID) {
		root := find(source)
		for _, targetID := range targets {
			if other := find(targetID); other != root {
				parent[other] = root
			}
		}
	}
	for _, armyID := range sortedArmyMap(ctx.joins) {
		join := ctx.joins[armyID]
		union(join.source, []models.TerritoryID{join.target})
	}
	for _, armyID := range sortedArmyMap(ctx.disperses) {
		disperse := ctx.disperses[armyID]
		union(disperse.source, disperse.targets)
	}

	byRoot := make(map[models.TerritoryID]*peacefulGroup)
	groupAt := func(territoryID models.TerritoryID) *peacefulGroup {
		root := find(territoryID)
		if byRoot[root] == nil {
			byRoot[root] = &peacefulGroup{joins: make(map[models.ArmyID]bool), disperses: make(map[models.ArmyID]bool)}
		}
		return byRoot[root]
	}
	groups := make(map[models.ArmyID]*peacefulGroup, len(ctx.joins)+len(ctx.disperses))
	for _, armyID := range sortedArmyMap(ctx.joins) {
		group := groupAt(ctx.joins[armyID].source)
		group.joins[armyID] = true
		groups[armyID] = group
	}
	for _, armyID := range sortedArmyMap(ctx.disperses) {
		disperse := ctx.disperses[armyID]
		group := groupAt(disperse.source)
		group.disperses[armyID] = true
		groups[disperse.armyID] = group
	}
	return groups
}

// peacefulOutcome is what combat needs from a join or a dispersion: whether
// its army empties its origin and, for a partial dispersion, how many troops
// remain there to defend it.
type peacefulOutcome struct {
	departs     bool
	residual    int
	hasResidual bool
}

// peacefulOutcome runs the join and dispersion rules on the group of an army
// against the current decisions and returns that army's outcome. When leaving
// is set, the other movements take the army as gone from its origin.
func (adj *adjudicator) peacefulOutcome(armyID models.ArmyID, leaving bool) peacefulOutcome {
	group := adj.peaceful[armyID]
	if group == nil {
		return peacefulOutcome{}
	}
	scratch := adj.ctx.peacefulScratch(group, decisionFacts{adj: adj, subject: armyID, leaving: leaving})
	resolvePeacefulMovements(scratch)

	if join := scratch.joins[armyID]; join != nil {
		return peacefulOutcome{departs: scratch.records[armyID].outcome == OutcomeSuccess}
	}
	for _, recordArmyID := range sortedArmyMap(scratch.disperseResults) {
		disperse := scratch.disperseResults[recordArmyID]
		if disperse.intent.armyID != armyID {
			continue
		}
		record := scratch.records[recordArmyID]
		if record == nil || record.outcome == OutcomeInvalid || disperse.invalid {
			return peacefulOutcome{}
		}
		if scratch.disperseVacatesSource(armyID, disperse.intent.source) {
			return peacefulOutcome{departs: true}
		}
		if record.partialD {
			return peacefulOutcome{residual: disperse.remaining, hasResidual: true}
		}
		return peacefulOutcome{}
	}
	return peacefulOutcome{}
}

// peacefulScratch returns a copy of the resolution context restricted to the
// joins and dispersions of group, with private copies of their records so
// that the join and dispersion rules can run on it without touching the turn.
// Records of dislodged armies are failed first, as applyContestOutcomes does.
func (ctx *resolutionContext) peacefulScratch(group *peacefulGroup, facts combatFacts) *resolutionContext {
	scratch := *ctx
	scratch.facts = facts
	scratch.joins = make(map[models.ArmyID]*joinIntent, len(group.joins))
	scratch.disperses = make(map[models.ArmyID]*disperseIntent, len(group.disperses))
	scratch.disperseResults = make(map[models.ArmyID]*disperseResolution)
	scratch.joinResults = make(map[models.ArmyID]*joinResolution)
	scratch.records = make(map[models.ArmyID]*orderRecord, len(ctx.records))
	for armyID, record := range ctx.records {
		scratch.records[armyID] = record
	}
	private := func(armyID models.ArmyID) {
		copyRecord := *ctx.records[armyID]
		switch {
		case copyRecord.outcome != "":
		case facts.dislodged(copyRecord.armyID):
			copyRecord.fail("dislodged")
		case copyRecord.executionArmyID != copyRecord.armyID && facts.dislodged(copyRecord.executionArmyID):
			if copyRecord.pendingDisperse {
				copyRecord.invalidate("disperse_residual_dislodged")
			} else {
				copyRecord.fail("dislodged")
			}
		}
		scratch.records[armyID] = &copyRecord
	}
	for _, armyID := range sortedArmyMap(group.joins) {
		scratch.joins[armyID] = ctx.joins[armyID]
		private(armyID)
	}
	for _, armyID := range sortedArmyMap(group.disperses) {
		scratch.disperses[armyID] = ctx.disperses[armyID]
		private(armyID)
	}
	return &scratch
}

// resolvePeacefulMovements applies the join and dispersion rules once combat
// is known through ctx.facts.
func resolvePeacefulMovements(ctx *resolutionContext) {
	resolveDispersions(ctx)
	resolveJoins(ctx)
	resolveVacatedDisperseDestinations(ctx)
	finalizeDispersions(ctx)
}
