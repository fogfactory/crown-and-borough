package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type orderRecord struct {
	armyID          models.ArmyID
	executionArmyID models.ArmyID
	chainID         models.ChainID
	order           models.Order
	pendingDisperse bool
	outcome         Outcome
	reason          string
	progression     Progression
	partialD        bool
	destroyed       bool
	fused           bool
}

type attackIntent struct {
	armyID models.ArmyID
	source models.TerritoryID
	target models.TerritoryID
	size   int
}

type joinIntent struct {
	armyID models.ArmyID
	source models.TerritoryID
	target models.TerritoryID
}

type disperseIntent struct {
	armyID       models.ArmyID
	recordArmyID models.ArmyID
	source       models.TerritoryID
	targets      []models.TerritoryID
	assignments  map[models.TerritoryID][]models.NobleID
	pending      bool
}

type supportIntent struct {
	armyID        models.ArmyID
	source        models.TerritoryID
	targetArmyID  models.ArmyID
	targetID      models.TerritoryID
	destinationID models.TerritoryID
	offensive     bool
	applies       bool
}

func enumerateIntentions(ctx *resolutionContext) {
	armyIDs := make([]models.ArmyID, 0, len(ctx.startArmiesByID))
	for armyID := range ctx.startArmiesByID {
		armyIDs = append(armyIDs, armyID)
	}
	sortArmyIDs(armyIDs)
	for _, armyID := range armyIDs {
		army := ctx.startArmiesByID[armyID]
		if army.ChainID == nil {
			continue
		}
		chain := ctx.chainsByID[*army.ChainID]
		if chain == nil || chain.CurrentIndex < 0 || chain.CurrentIndex >= len(chain.Orders) {
			continue
		}
		record := &orderRecord{
			armyID:          army.ID,
			executionArmyID: army.ID,
			chainID:         chain.ID,
			order:           chain.Orders[chain.CurrentIndex],
		}
		ctx.records[army.ID] = record
		if chain.PendingDisperse != nil {
			ctx.enumeratePendingDisperse(record, chain)
			continue
		}
		if record.order.PositionID != army.TerritoryID {
			record.fail("position_mismatch")
			continue
		}
		ctx.enumerateOrder(record, army, chain.CurrentIndex == len(chain.Orders)-1)
	}
}

func (ctx *resolutionContext) enumeratePendingDisperse(record *orderRecord, chain *models.Chain) {
	pending := chain.PendingDisperse
	if pending == nil {
		record.invalidate("missing_pending_disperse")
		return
	}
	army, exists := ctx.startArmiesByID[pending.ArmyID]
	if !exists || army.TerritoryID != pending.SourceID {
		record.invalidate("missing_disperse_residual")
		return
	}
	record.executionArmyID = army.ID
	record.pendingDisperse = true
	record.order.PositionID = pending.SourceID
	record.order.TargetIDs = append([]models.TerritoryID(nil), pending.TargetIDs...)
	record.order.NobleAssignments = cloneNobleAssignments(pending.NobleAssignments)
	assignments, valid := ctx.validateDisperse(record, army)
	if !valid {
		return
	}
	ctx.disperses[record.armyID] = &disperseIntent{
		armyID:       army.ID,
		recordArmyID: record.armyID,
		source:       army.TerritoryID,
		targets:      append([]models.TerritoryID(nil), record.order.TargetIDs...),
		assignments:  assignments,
		pending:      true,
	}
}

func (ctx *resolutionContext) enumerateOrder(record *orderRecord, army models.Army, isLastOrder bool) {
	order := record.order
	switch order.Type {
	case models.OrderTypeAttack:
		targetID, valid := ctx.singleAdjacentTarget(record, army, false)
		if !valid {
			return
		}
		if defender := ctx.startArmyAt(targetID); defender != nil && defender.OwnerID == army.OwnerID {
			record.fail("allied_destination")
			return
		}
		ctx.attacks[army.ID] = &attackIntent{armyID: army.ID, source: army.TerritoryID, target: targetID, size: army.Size}
		ctx.attackedTerritories[targetID] = true
	case models.OrderTypeJoin:
		if !isLastOrder {
			record.invalidate("join_not_terminal")
			return
		}
		targetID, valid := ctx.singleAdjacentTarget(record, army, false)
		if !valid {
			return
		}
		if defender := ctx.startArmyAt(targetID); defender != nil && defender.OwnerID != army.OwnerID {
			record.fail("enemy_destination")
			return
		}
		ctx.joins[army.ID] = &joinIntent{armyID: army.ID, source: army.TerritoryID, target: targetID}
	case models.OrderTypeDisperse:
		assignments, valid := ctx.validateDisperse(record, army)
		if !valid {
			return
		}
		ctx.disperses[army.ID] = &disperseIntent{
			armyID:       army.ID,
			recordArmyID: army.ID,
			source:       army.TerritoryID,
			targets:      append([]models.TerritoryID(nil), order.TargetIDs...),
			assignments:  assignments,
		}
	case models.OrderTypeSupport:
		ctx.validateSupport(record, army)
	case models.OrderTypeHold:
		if len(order.TargetIDs) != 0 || len(order.NobleTargetIDs) != 0 || len(order.NobleAssignments) != 0 {
			record.invalidate("invalid_hold_shape")
		}
	case models.OrderTypePillage:
		if len(order.TargetIDs) != 0 || len(order.NobleTargetIDs) != 0 || len(order.NobleAssignments) != 0 {
			record.invalidate("invalid_pillage_shape")
			return
		}
		if len(ctx.state.TerritoryStates[army.TerritoryID].Infrastructures) == 0 {
			record.invalidate("no_infrastructure")
		}
	case models.OrderTypeHostage, models.OrderTypeDungeon:
		if len(order.TargetIDs) != 0 || len(order.NobleTargetIDs) != 1 || len(order.NobleAssignments) != 0 {
			record.invalidate("invalid_noble_order_shape")
		}
	default:
		record.invalidate("unknown_order_type")
	}
}

func (ctx *resolutionContext) singleAdjacentTarget(record *orderRecord, army models.Army, allowSource bool) (models.TerritoryID, bool) {
	if len(record.order.TargetIDs) != 1 || len(record.order.NobleTargetIDs) != 0 || len(record.order.NobleAssignments) != 0 {
		record.invalidate("invalid_target_shape")
		return "", false
	}
	targetID := record.order.TargetIDs[0]
	if ctx.territoriesByID[targetID] == nil || (!allowSource || targetID != army.TerritoryID) && !ctx.isAdjacent(army.TerritoryID, targetID) {
		record.invalidate("non_adjacent_destination")
		return "", false
	}
	return targetID, true
}

func (ctx *resolutionContext) validateSupport(record *orderRecord, army models.Army) {
	order := record.order
	if len(order.NobleTargetIDs) != 0 || len(order.NobleAssignments) != 0 || len(order.TargetIDs) < 1 || len(order.TargetIDs) > 2 {
		record.invalidate("invalid_support_shape")
		return
	}
	targetID := order.TargetIDs[0]
	if ctx.territoriesByID[targetID] == nil {
		record.invalidate("unknown_support_target")
		return
	}
	if len(order.TargetIDs) == 1 {
		if targetID == army.TerritoryID || !ctx.isAdjacent(army.TerritoryID, targetID) {
			record.invalidate("invalid_defensive_support")
		}
		return
	}
	destinationID := order.TargetIDs[1]
	if ctx.territoriesByID[destinationID] == nil || !ctx.isAdjacent(army.TerritoryID, destinationID) || !ctx.isAdjacent(targetID, destinationID) {
		record.invalidate("invalid_offensive_support")
	}
}

func (ctx *resolutionContext) validateDisperse(record *orderRecord, army models.Army) (map[models.TerritoryID][]models.NobleID, bool) {
	order := record.order
	if len(order.NobleTargetIDs) != 0 || len(order.TargetIDs) != army.Size {
		record.invalidate("invalid_disperse_size")
		return nil, false
	}
	targetSet := make(map[models.TerritoryID]bool, len(order.TargetIDs))
	for _, targetID := range order.TargetIDs {
		if ctx.territoriesByID[targetID] == nil || (targetID != army.TerritoryID && !ctx.isAdjacent(army.TerritoryID, targetID)) {
			record.invalidate("non_adjacent_disperse_destination")
			return nil, false
		}
		if targetSet[targetID] {
			record.invalidate("duplicate_disperse_destination")
			return nil, false
		}
		targetSet[targetID] = true
	}

	coLocated := ctx.noblesAt(army.TerritoryID)
	assignments := make(map[models.TerritoryID][]models.NobleID, len(order.NobleAssignments))
	assigned := make(map[models.NobleID]bool, len(coLocated))
	wildcardDestination := models.TerritoryID("")
	assignmentDestinations := make([]models.TerritoryCode, 0, len(order.NobleAssignments))
	for destination := range order.NobleAssignments {
		assignmentDestinations = append(assignmentDestinations, destination)
	}
	sort.Slice(assignmentDestinations, func(i, j int) bool { return assignmentDestinations[i] < assignmentDestinations[j] })
	for _, destinationCode := range assignmentDestinations {
		destinationID, exists := ctx.territoriesByCode[destinationCode]
		if !exists || !targetSet[destinationID] {
			record.invalidate("invalid_disperse_assignment_destination")
			return nil, false
		}
		for _, nobleCode := range order.NobleAssignments[destinationCode] {
			if nobleCode == "*" {
				if wildcardDestination != "" {
					record.invalidate("duplicate_disperse_wildcard")
					return nil, false
				}
				wildcardDestination = destinationID
				continue
			}
			nobleID, exists := ctx.noblesByCode[nobleCode]
			if !exists || assigned[nobleID] || !containsNobleID(coLocated, nobleID) {
				record.invalidate("invalid_disperse_noble_assignment")
				return nil, false
			}
			assigned[nobleID] = true
			assignments[destinationID] = append(assignments[destinationID], nobleID)
		}
	}
	if wildcardDestination != "" {
		for _, nobleID := range coLocated {
			if !assigned[nobleID] {
				assigned[nobleID] = true
				assignments[wildcardDestination] = append(assignments[wildcardDestination], nobleID)
			}
		}
	}
	for _, nobleID := range coLocated {
		if !assigned[nobleID] {
			record.invalidate("incomplete_disperse_noble_assignment")
			return nil, false
		}
	}
	return assignments, true
}

func containsNobleID(ids []models.NobleID, target models.NobleID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func (record *orderRecord) fail(reason string) {
	record.outcome = OutcomeFailure
	record.reason = reason
}

func (record *orderRecord) invalidate(reason string) {
	record.outcome = OutcomeInvalid
	record.reason = reason
}
