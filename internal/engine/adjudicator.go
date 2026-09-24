package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// The adjudicator resolves every attack, join and dispersion of a turn as a
// set of boolean decisions, each a pure function of other decisions: whether
// an attack moves, whether a join or a dispersion empties its origin, and
// whether an army that stays is dislodged. The move equation and the
// strengths (attack, hold, defend, prevent) follow Lucas Kruijswijk's "The
// Math of Adjudication" (Diplomatic Pouch, 2009), the reference behind the
// Diplomacy Adjudicator Test Cases, with Crown & Borough's own strengths.
//
// Kruijswijk resolves cycles of decisions by recursive guessing, which relies
// on a property of Diplomacy without convoys: a decision belongs to at most
// one cycle. Joins and dispersions break that property, since a departure
// depends on every combat around the territories it touches. The adjudicator
// therefore makes the cycles explicit instead:
//
//  1. It builds the static graph of which decision may read which other one,
//     and splits it into strongly connected components (Tarjan), in an order
//     where every component comes after the components it reads.
//  2. A component without a cycle is adjudicated directly from settled
//     decisions.
//  3. A cycle is searched for a consistent resolution, one where every
//     decision equals its adjudication: each attack and departure, in a fixed
//     order, succeeds whenever some consistent resolution allows it. A
//     circular movement, such as a rotation of attacks or a join crossing an
//     attack, therefore succeeds, as in Diplomacy.
//  4. A join or a dispersion whose origin is under attack, by anyone, is
//     cancelled before the graph is even built: none of its troops leaves.
//     This removes every cycle a departure's dependency on combat could
//     create, which leaves only Diplomacy's circular movements; a cycle
//     still without a consistent resolution is therefore structurally
//     unreachable, but the attacks then keep the status quo as a
//     defensive fallback, should one occur regardless.
//
// Components are finite, each is settled once, and the search of a cycle
// visits each assignment of its attacks and departures at most once, so
// resolution always terminates.
type decisionKind uint8

const (
	// decisionMove: an attack reaches its destination.
	decisionMove decisionKind = iota
	// decisionDeparture: a join or a dispersion empties its origin.
	decisionDeparture
	// decisionDislodged: an army that stays on its origin is dislodged.
	decisionDislodged
)

type decision struct {
	kind   decisionKind
	armyID models.ArmyID
}

type adjudicator struct {
	ctx *resolutionContext
	// settled holds the final value of every decision already resolved, and
	// assigned the working values of the cycle being solved.
	settled  map[decision]bool
	assigned map[decision]bool

	attacksByTarget map[models.TerritoryID][]models.ArmyID
	staticCuts      map[models.ArmyID]bool
	peaceful        map[models.ArmyID]*peacefulGroup
}

func newAdjudicator(ctx *resolutionContext) *adjudicator {
	adj := &adjudicator{
		ctx:             ctx,
		settled:         make(map[decision]bool),
		assigned:        make(map[decision]bool),
		attacksByTarget: make(map[models.TerritoryID][]models.ArmyID),
		staticCuts:      staticSupportCuts(ctx),
		peaceful:        peacefulGroups(ctx),
	}
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		target := ctx.attacks[armyID].target
		adj.attacksByTarget[target] = append(adj.attacksByTarget[target], armyID)
	}
	return adj
}

// resolveAll settles every decision of the turn, component by component.
func (adj *adjudicator) resolveAll() {
	for _, component := range adj.components() {
		adj.solve(component)
	}
}

func (adj *adjudicator) decisions() []decision {
	ctx := adj.ctx
	decisions := make([]decision, 0, len(ctx.attacks)+len(adj.peaceful)+len(ctx.startArmiesByID))
	for _, armyID := range sortedArmyMap(ctx.attacks) {
		decisions = append(decisions, decision{kind: decisionMove, armyID: armyID})
	}
	for _, armyID := range sortedArmyMap(adj.peaceful) {
		decisions = append(decisions, decision{kind: decisionDeparture, armyID: armyID})
	}
	for _, armyID := range sortedArmyMap(ctx.startArmiesByID) {
		decisions = append(decisions, decision{kind: decisionDislodged, armyID: armyID})
	}
	return decisions
}

// components returns the strongly connected components of the dependency
// graph, each listed after every component it depends on (Tarjan's
// algorithm emits them in that order).
func (adj *adjudicator) components() [][]decision {
	index := make(map[decision]int)
	low := make(map[decision]int)
	onStack := make(map[decision]bool)
	var stack []decision
	var components [][]decision
	var visit func(decision)
	visit = func(d decision) {
		index[d] = len(index)
		low[d] = index[d]
		stack = append(stack, d)
		onStack[d] = true
		for _, dependency := range adj.dependencies(d) {
			if _, visited := index[dependency]; !visited {
				visit(dependency)
				low[d] = min(low[d], low[dependency])
			} else if onStack[dependency] {
				low[d] = min(low[d], index[dependency])
			}
		}
		if low[d] != index[d] {
			return
		}
		var component []decision
		for {
			member := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[member] = false
			component = append(component, member)
			if member == d {
				break
			}
		}
		components = append(components, adj.ordered(component))
	}
	for _, d := range adj.decisions() {
		if _, visited := index[d]; !visited {
			visit(d)
		}
	}
	return components
}

// ordered sorts the decisions of a component as decisions() lists them, so
// that searching a cycle is deterministic.
func (adj *adjudicator) ordered(component []decision) []decision {
	rank := make(map[decision]int, len(component))
	for position, d := range adj.decisions() {
		rank[d] = position
	}
	ordered := append([]decision(nil), component...)
	sort.Slice(ordered, func(i, j int) bool { return rank[ordered[i]] < rank[ordered[j]] })
	return ordered
}

// dependencies lists every decision that adjudicating d may read, whatever
// the values involved.
func (adj *adjudicator) dependencies(d decision) []decision {
	ctx := adj.ctx
	var dependencies []decision
	add := func(kind decisionKind, armyID models.ArmyID) {
		dependencies = append(dependencies, decision{kind: kind, armyID: armyID})
	}
	stays := func(armyID models.ArmyID) {
		if ctx.attacks[armyID] != nil {
			add(decisionMove, armyID)
		} else if adj.peaceful[armyID] != nil {
			add(decisionDeparture, armyID)
		}
	}
	supporters := func(armyID models.ArmyID, offensive bool) {
		for _, supportID := range sortedArmyMap(ctx.supports) {
			support := ctx.supports[supportID]
			if support.offensive == offensive && support.targetArmyID == armyID && support.applies && !adj.staticCuts[supportID] && !ctx.famished[supportID] {
				add(decisionDislodged, supportID)
			}
		}
	}
	peacefulGroup := func(group *peacefulGroup) {
		for _, armyID := range group.armies(ctx) {
			add(decisionDislodged, armyID)
		}
		for _, territoryID := range group.territories(ctx) {
			if occupant := ctx.startArmyAt(territoryID); occupant != nil {
				stays(occupant.ID)
				add(decisionDislodged, occupant.ID)
			}
			for _, armyID := range adj.attacksByTarget[territoryID] {
				add(decisionMove, armyID)
			}
		}
	}

	switch d.kind {
	case decisionMove:
		attack := ctx.attacks[d.armyID]
		supporters(d.armyID, true)
		if defender := ctx.startArmyAt(attack.target); defender != nil {
			stays(defender.ID)
			if adj.headToHead(d.armyID) {
				supporters(defender.ID, true)
			}
			supporters(defender.ID, false)
			if group := adj.peaceful[defender.ID]; group != nil {
				add(decisionDislodged, defender.ID)
				peacefulGroup(group)
			}
		}
		for _, otherID := range adj.attacksByTarget[attack.target] {
			supporters(otherID, true)
		}
	case decisionDeparture:
		peacefulGroup(adj.peaceful[d.armyID])
	case decisionDislodged:
		if army, exists := ctx.startArmiesByID[d.armyID]; exists && len(adj.attacksByTarget[army.TerritoryID]) != 0 {
			stays(d.armyID)
			for _, armyID := range adj.attacksByTarget[army.TerritoryID] {
				add(decisionMove, armyID)
			}
		}
	}
	return dependencies
}

// solve settles a component. A single decision that does not read itself is
// adjudicated directly. A cycle is searched depth first over its attacks and
// departures, in a fixed order and trying success before failure, while its
// dislodgements follow from them; every decision is checked as soon as all
// the decisions it reads are fixed. The first consistent resolution found is
// the one that lets each order succeed, in turn, whenever some consistent
// resolution allows it. Cancelling every join and dispersion whose origin is
// under attack, before components are even built, removes the only cycles a
// departure could create, so a cycle without a consistent resolution should
// no longer occur; should one occur regardless, the attacks keep the status
// quo, as cheap insurance against a case the rule missed.
func (adj *adjudicator) solve(component []decision) {
	if len(component) == 1 && !adj.readsItself(component[0]) {
		adj.settled[component[0]] = adj.adjudicate(component[0])
		return
	}
	if !adj.newCycleSearch(component).run(0) {
		adj.statusQuo(component)
	}
	for _, d := range component {
		if value, assigned := adj.assigned[d]; assigned {
			adj.settled[d] = value
			delete(adj.assigned, d)
		}
	}
}

// cycleSearch is the depth-first search of a cycle's resolution. Attacks and
// departures are chosen; a dislodgement only reads attacks and departures, so
// it is derived once they are fixed.
type cycleSearch struct {
	adj    *adjudicator
	chosen []decision
	// derivedAt and checkedAt list, for each depth, the dislodgements that can
	// be derived and the choices that can be checked once the first depth
	// choices are fixed.
	derivedAt [][]decision
	checkedAt [][]decision
}

func (adj *adjudicator) newCycleSearch(component []decision) *cycleSearch {
	search := &cycleSearch{adj: adj}
	members := make(map[decision]bool, len(component))
	for _, d := range component {
		members[d] = true
		if d.kind != decisionDislodged {
			search.chosen = append(search.chosen, d)
		}
	}
	depth := make(map[decision]int, len(component))
	for index, d := range search.chosen {
		depth[d] = index + 1
	}
	search.derivedAt = make([][]decision, len(search.chosen)+1)
	search.checkedAt = make([][]decision, len(search.chosen)+1)
	for _, d := range component {
		if d.kind == decisionDislodged {
			depth[d] = search.readyAt(d, members, depth)
			search.derivedAt[depth[d]] = append(search.derivedAt[depth[d]], d)
		}
	}
	for _, d := range search.chosen {
		ready := max(depth[d], search.readyAt(d, members, depth))
		search.checkedAt[ready] = append(search.checkedAt[ready], d)
	}
	return search
}

// readyAt returns the depth at which every member of the cycle that d reads
// is fixed.
func (search *cycleSearch) readyAt(d decision, members map[decision]bool, depth map[decision]int) int {
	ready := 0
	for _, dependency := range search.adj.dependencies(d) {
		if members[dependency] {
			ready = max(ready, depth[dependency])
		}
	}
	return ready
}

func (search *cycleSearch) run(fixed int) bool {
	adj := search.adj
	for _, d := range search.derivedAt[fixed] {
		adj.assigned[d] = adj.adjudicate(d)
	}
	for _, d := range search.checkedAt[fixed] {
		if adj.adjudicate(d) != adj.assigned[d] {
			search.undo(fixed)
			return false
		}
	}
	if fixed == len(search.chosen) {
		return true
	}
	d := search.chosen[fixed]
	for _, value := range []bool{true, false} {
		adj.assigned[d] = value
		if search.run(fixed + 1) {
			return true
		}
	}
	delete(adj.assigned, d)
	return false
}

func (search *cycleSearch) undo(fixed int) {
	for _, d := range search.derivedAt[fixed] {
		delete(search.adj.assigned, d)
	}
}

// statusQuo resolves a cycle that has no consistent resolution: none of its
// attacks succeeds, and the dislodgements follow from that.
func (adj *adjudicator) statusQuo(component []decision) {
	for _, d := range component {
		if d.kind != decisionDislodged {
			adj.assigned[d] = false
		}
	}
	for _, d := range component {
		if d.kind == decisionDislodged {
			adj.assigned[d] = adj.adjudicate(d)
		}
	}
}

func (adj *adjudicator) readsItself(d decision) bool {
	for _, dependency := range adj.dependencies(d) {
		if dependency == d {
			return true
		}
	}
	return false
}

// value returns a decision read while adjudicating another one: it is either
// settled or part of the cycle being solved, since components are solved
// after every component they depend on. Reading any other decision means
// dependencies missed a read, which resolveContests reports as an error.
func (adj *adjudicator) value(d decision) bool {
	if value, settled := adj.settled[d]; settled {
		return value
	}
	if value, assigned := adj.assigned[d]; assigned {
		return value
	}
	panic(unsettledDecision{d})
}

// unsettledDecision is raised when adjudication reads a decision that is
// neither settled nor in the cycle being solved.
type unsettledDecision struct {
	decision decision
}

func (err unsettledDecision) Error() string {
	return fmt.Sprintf("engine: adjudication read unsettled decision %d of army %q", err.decision.kind, err.decision.armyID)
}

func (adj *adjudicator) adjudicate(d decision) bool {
	switch d.kind {
	case decisionMove:
		return adj.adjudicateMove(d.armyID)
	case decisionDeparture:
		return adj.peacefulOutcome(d.armyID, false).departs
	default:
		return adj.adjudicateDislodged(d.armyID)
	}
}

func (adj *adjudicator) moves(armyID models.ArmyID) bool {
	if adj.ctx.attacks[armyID] == nil {
		return false
	}
	return adj.value(decision{kind: decisionMove, armyID: armyID})
}

func (adj *adjudicator) departs(armyID models.ArmyID) bool {
	if adj.peaceful[armyID] == nil {
		return false
	}
	return adj.value(decision{kind: decisionDeparture, armyID: armyID})
}

func (adj *adjudicator) dislodged(armyID models.ArmyID) bool {
	return adj.value(decision{kind: decisionDislodged, armyID: armyID})
}

// stays reports whether an army is still on its origin once movements are
// settled: it neither won its attack nor completed a join or a dispersion.
func (adj *adjudicator) stays(armyID models.ArmyID) bool {
	if adj.ctx.attacks[armyID] != nil {
		return !adj.moves(armyID)
	}
	return !adj.departs(armyID)
}

func (adj *adjudicator) adjudicateDislodged(armyID models.ArmyID) bool {
	army, exists := adj.ctx.startArmiesByID[armyID]
	if !exists || len(adj.attacksByTarget[army.TerritoryID]) == 0 || !adj.stays(armyID) {
		return false
	}
	for _, attackerID := range adj.attacksByTarget[army.TerritoryID] {
		if adj.moves(attackerID) {
			return true
		}
	}
	return false
}

// adjudicateMove applies Kruijswijk's move equation with Crown & Borough
// strengths. A head-to-head opponent must be outmatched on the forces that do
// not come from its own owner; an army that stays must be outmatched by the
// supports that do not come from its owner, and never by an ally; every other
// attack on the destination keeps its full strength to prevent the move,
// unless it was dislodged by the head-to-head winner coming from that
// destination.
func (adj *adjudicator) adjudicateMove(armyID models.ArmyID) bool {
	ctx := adj.ctx
	attack := ctx.attacks[armyID]
	owner := ctx.startArmiesByID[armyID].OwnerID
	defender := ctx.startArmyAt(attack.target)
	if defender != nil && adj.headToHead(armyID) {
		if defender.OwnerID == owner {
			return false
		}
		ourForce, ourHelp := adj.attackForce(armyID, defender.OwnerID)
		theirForce, theirHelp := adj.attackForce(defender.ID, owner)
		if ourForce-ourHelp <= theirForce-theirHelp {
			return false
		}
	}

	strength := 0
	threshold := -1
	if defender != nil && adj.stays(defender.ID) {
		if defender.OwnerID == owner {
			return false
		}
		force, help := adj.attackForce(armyID, defender.OwnerID)
		strength = force - help
		threshold = adj.holdStrength(*defender)
	} else {
		strength, _ = adj.attackForce(armyID, "")
		if ctx.hasCastle(attack.target) && !ctx.castleOwnedByAllAttackers(attack.target) {
			threshold = ctx.balance.CastleDefenseBonus
		}
	}
	if strength <= threshold {
		return false
	}
	for _, otherID := range adj.attacksByTarget[attack.target] {
		if otherID == armyID || adj.retiredGhost(otherID) {
			continue
		}
		if prevent, _ := adj.attackForce(otherID, ""); strength <= prevent {
			return false
		}
	}
	return true
}

// headToHead reports whether the army on an attack's destination attacks the
// attack's origin.
func (adj *adjudicator) headToHead(armyID models.ArmyID) bool {
	attack := adj.ctx.attacks[armyID]
	defender := adj.ctx.startArmyAt(attack.target)
	if defender == nil {
		return false
	}
	opposing := adj.ctx.attacks[defender.ID]
	return opposing != nil && opposing.target == attack.source
}

// retiredGhost reports whether an attack lost its head-to-head battle: the
// opposing army moved into its origin, so it no longer prevents anything.
func (adj *adjudicator) retiredGhost(armyID models.ArmyID) bool {
	if !adj.headToHead(armyID) {
		return false
	}
	opponent := adj.ctx.startArmyAt(adj.ctx.attacks[armyID].target)
	return adj.moves(opponent.ID)
}

// attackForce returns the strength of an attack and the part of it given by
// supports of against, the owner of the army it attacks.
func (adj *adjudicator) attackForce(armyID models.ArmyID, against models.PlayerID) (int, int) {
	ctx := adj.ctx
	attacker := ctx.startArmiesByID[armyID]
	force := ctx.attacks[armyID].size + nobleCommandBonus(ctx, attacker)
	help := 0
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if !support.offensive || support.targetArmyID != armyID || !adj.supportCounts(supportID) {
			continue
		}
		supporter := ctx.startArmiesByID[supportID]
		strength := supporter.Size + nobleCommandBonus(ctx, supporter)
		force += strength
		if against != "" && supporter.OwnerID == against {
			help += strength
		}
	}
	return force, help
}

// supportCounts reports whether a support adds its strength: it applies, it
// is not cut by an enemy attack nor by the dislodgement of its army, and the
// army is not famished.
func (adj *adjudicator) supportCounts(supportID models.ArmyID) bool {
	support := adj.ctx.supports[supportID]
	return support.applies && !adj.staticCuts[supportID] && !adj.ctx.famished[supportID] && !adj.dislodged(supportID)
}

// holdStrength returns the defense of an army that stays on its origin.
func (adj *adjudicator) holdStrength(defender models.Army) int {
	base, supported := adj.defenseStrength(defender)
	return base + supported
}

// defenseStrength returns the defense of an army that stays on its origin
// without defensive supports, and the strength those supports add. Only
// armies that hold, support or pillage receive defensive supports. A partial
// dispersion defends with the troops it leaves behind: measured against the
// other movements as resolved when it holds its origin, and, when it is
// dislodged, which cancels it, as if it had left.
func (adj *adjudicator) defenseStrength(defender models.Army) (int, int) {
	ctx := adj.ctx
	base := 0
	if ctx.hasCastle(defender.TerritoryID) {
		base = ctx.balance.CastleDefenseBonus
	}
	if !ctx.famished[defender.ID] {
		size := defender.Size
		if outcome := adj.peacefulOutcome(defender.ID, adj.peaceful[defender.ID] != nil && adj.dislodged(defender.ID)); outcome.hasResidual {
			size = outcome.residual
		}
		base += size
	}
	base += nobleCommandBonus(ctx, defender)
	record := ctx.records[defender.ID]
	if record == nil || !holdsForDefense(record.order.Type) {
		return base, 0
	}
	supported := 0
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		if support.offensive || support.targetArmyID != defender.ID || !adj.supportCounts(supportID) {
			continue
		}
		supporter := ctx.startArmiesByID[supportID]
		supported += supporter.Size + nobleCommandBonus(ctx, supporter)
	}
	return base, supported
}

// staticSupportCuts marks the supports cut by an enemy attack on the
// supporting army that does not come from the territory the support is
// directed at, whatever the outcome of that attack.
func staticSupportCuts(ctx *resolutionContext) map[models.ArmyID]bool {
	cuts := make(map[models.ArmyID]bool, len(ctx.supports))
	for _, supportID := range sortedArmyMap(ctx.supports) {
		support := ctx.supports[supportID]
		exemptOriginID := support.targetID
		if support.offensive {
			exemptOriginID = support.destinationID
		}
		supporter := ctx.startArmiesByID[supportID]
		for _, attackID := range sortedArmyMap(ctx.attacks) {
			attack := ctx.attacks[attackID]
			if attack.target == support.source && attack.source != exemptOriginID && ctx.startArmiesByID[attackID].OwnerID != supporter.OwnerID {
				cuts[supportID] = true
				break
			}
		}
	}
	return cuts
}
