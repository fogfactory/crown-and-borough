package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ManeuverLeg is what one destination of a maneuver receives: the troops, the
// nobles that travel with them, and whether the order sheet (the chain)
// follows this group.
type ManeuverLeg struct {
	Target models.TerritoryID `json:"target"`
	Troops int                `json:"troops"`
	Nobles []models.NobleCode `json:"nobles,omitempty"`
	// AnyNobles marks a "*" whose nobles are not known (see PlanManeuver).
	AnyNobles    bool `json:"anyNobles,omitempty"`
	CarriesChain bool `json:"carriesChain,omitempty"`
}

// ManeuverPlan splits an army the way a maneuver order says. It is the one
// place where the placement rules live: the resolution executes it and the
// API projects it for the map.
type ManeuverPlan struct {
	// Stay is what stays on the origin (its Target is the origin): the troops
	// not sent anywhere and the groups sent to the origin itself.
	Stay ManeuverLeg
	// StayListed reports that the order names its own origin.
	StayListed bool
	// Groups are the legs that leave the origin, one per distinct destination,
	// in order of appearance. A group may receive no troops.
	Groups []ManeuverLeg
	// LastTarget is the destination of the last entry the weather lets
	// through: the group that carries the chain.
	LastTarget models.TerritoryID
	// LastEntry is the index of that entry in the order.
	LastEntry int
	// EntryEmpty and EntryRest tell, per entry of the order, that it received
	// no troops and that it is a "+".
	EntryEmpty []bool
	EntryRest  []bool
	// Invalid reports a noble assignment that cannot be honored: a noble named
	// twice, or one that does not stand with the army (when known).
	Invalid bool
}

// Legs lists the legs for display: the origin first, when something stays or
// the order names it, then the groups.
func (plan ManeuverPlan) Legs() []ManeuverLeg {
	legs := make([]ManeuverLeg, 0, len(plan.Groups)+1)
	if plan.Stay.Troops > 0 || plan.StayListed {
		legs = append(legs, plan.Stay)
	}
	return append(legs, plan.Groups...)
}

// PlanManeuver plans the order for an army of armySize troops standing with
// noblesHere. blocked marks the entries the weather removes (nil for none).
//
// Troops go to the entries in order of appearance, a "+" taking what is left.
// A noble leaves with the group it is assigned to, or with the group that
// carries the chain when no destination claims it (a "*" takes every
// unclaimed noble instead); a noble whose group gets no troops, was removed
// by the weather or is the origin stays with the troops of the origin, unless
// no troop stays: a noble cannot stay alone, so it follows the group that
// carries the chain, or else the last group that receives troops. The sheet
// follows the last group only when that group receives troops. With noblesHere
// nil the unclaimed nobles are not known: a "*" then marks its leg AnyNobles.
// A destination without a count takes one troop.
func PlanManeuver(order models.Order, armySize int, blocked []bool, noblesHere []models.NobleCode) ManeuverPlan {
	count := len(order.TargetIDs)
	plan := ManeuverPlan{
		Stay:       ManeuverLeg{Target: order.PositionID},
		EntryEmpty: make([]bool, count),
		EntryRest:  make([]bool, count),
	}
	groupAt := make(map[models.TerritoryID]int)
	available := armySize
	for index, targetID := range order.TargetIDs {
		if targetID == order.PositionID {
			plan.StayListed = true
		}
		if index < len(blocked) && blocked[index] {
			continue
		}
		plan.LastTarget, plan.LastEntry = targetID, index
		troops := 1
		if index < len(order.Counts) {
			troops = order.Counts[index]
		}
		if troops == models.ManeuverRest {
			plan.EntryRest[index] = true
			troops = available
		}
		troops = min(troops, available)
		available -= troops
		plan.EntryEmpty[index] = troops == 0
		if targetID == order.PositionID {
			plan.Stay.Troops += troops
			continue
		}
		groupIndex, exists := groupAt[targetID]
		if !exists {
			groupIndex = len(plan.Groups)
			groupAt[targetID] = groupIndex
			plan.Groups = append(plan.Groups, ManeuverLeg{Target: targetID})
		}
		plan.Groups[groupIndex].Troops += troops
	}
	plan.Stay.Troops += available

	// The leg that receives a noble sent to targetID: the group when it gets
	// troops, the origin otherwise.
	receiver := func(targetID models.TerritoryID) *ManeuverLeg {
		if groupIndex, exists := groupAt[targetID]; exists && plan.Groups[groupIndex].Troops > 0 {
			return &plan.Groups[groupIndex]
		}
		return &plan.Stay
	}
	here := make(map[models.NobleCode]bool, len(noblesHere))
	for _, code := range noblesHere {
		here[code] = true
	}
	claimed := make(map[models.NobleCode]bool)
	wildcardTarget := models.TerritoryID("")
	for _, targetID := range sortedTerritoryMap(order.NobleAssignments) {
		for _, code := range order.NobleAssignments[targetID] {
			if code == "*" {
				wildcardTarget = targetID
				continue
			}
			plan.Invalid = plan.Invalid || claimed[code] || (noblesHere != nil && !here[code])
			claimed[code] = true
			leg := receiver(targetID)
			leg.Nobles = append(leg.Nobles, code)
		}
	}
	leftoverTarget := plan.LastTarget
	if wildcardTarget != "" {
		leftoverTarget = wildcardTarget
		if noblesHere == nil {
			receiver(wildcardTarget).AnyNobles = true
		}
	}
	for _, code := range noblesHere {
		if !claimed[code] {
			leg := receiver(leftoverTarget)
			leg.Nobles = append(leg.Nobles, code)
		}
	}
	// A noble cannot stay alone on an emptied origin.
	if plan.Stay.Troops == 0 && (len(plan.Stay.Nobles) > 0 || plan.Stay.AnyNobles) {
		if follower := plan.follower(groupAt); follower != nil {
			follower.Nobles = append(follower.Nobles, plan.Stay.Nobles...)
			follower.AnyNobles = follower.AnyNobles || plan.Stay.AnyNobles
			plan.Stay.Nobles, plan.Stay.AnyNobles = nil, false
		}
	}
	// The sheet follows the last group, if it receives troops.
	if groupIndex, exists := groupAt[plan.LastTarget]; exists {
		plan.Groups[groupIndex].CarriesChain = plan.Groups[groupIndex].Troops > 0
	} else {
		plan.Stay.CarriesChain = count > 0 && plan.LastTarget == order.PositionID &&
			plan.Stay.Troops > 0 && !plan.EntryEmpty[plan.LastEntry]
	}
	return plan
}

// follower is the group the nobles of an emptied origin go with: the one that
// carries the chain, otherwise the last group that receives troops.
func (plan *ManeuverPlan) follower(groupAt map[models.TerritoryID]int) *ManeuverLeg {
	if groupIndex, exists := groupAt[plan.LastTarget]; exists && plan.Groups[groupIndex].Troops > 0 {
		return &plan.Groups[groupIndex]
	}
	for index := len(plan.Groups) - 1; index >= 0; index-- {
		if plan.Groups[index].Troops > 0 {
			return &plan.Groups[index]
		}
	}
	return nil
}

// PlanChainManeuvers plans the maneuvers of a chain carried by army, one
// entry per order (nil for the other orders, and for those already carried
// out), or nil when the chain has no maneuver left. Only the current order
// knows who stands with the army; the later ones start from where it will be,
// so their unclaimed nobles are not placed and their troops are indicative.
func PlanChainManeuvers(game *models.GameState, army models.Army, orders []models.Order, current int) [][]ManeuverLeg {
	var legs [][]ManeuverLeg
	for index := max(current, 0); index < len(orders); index++ {
		order := orders[index]
		if order.Type != models.OrderTypeManeuver {
			continue
		}
		var here []models.NobleCode
		if index == current {
			here = noblesCodesAt(game.Nobles, army.TerritoryID)
		}
		if legs == nil {
			legs = make([][]ManeuverLeg, len(orders))
		}
		legs[index] = PlanManeuver(order, army.Size, nil, here).Legs()
	}
	return legs
}

// noblesCodesAt lists the codes of the nobles standing on a territory, in the
// order of their IDs (as the resolution sees them).
func noblesCodesAt(nobles []models.Noble, territoryID models.TerritoryID) []models.NobleCode {
	standing := make([]models.Noble, 0)
	for _, noble := range nobles {
		if noble.LocationID == territoryID {
			standing = append(standing, noble)
		}
	}
	sort.Slice(standing, func(i, j int) bool { return standing[i].ID < standing[j].ID })
	codes := make([]models.NobleCode, len(standing))
	for index, noble := range standing {
		codes[index] = models.NobleCode(noble.Code)
	}
	return codes
}
