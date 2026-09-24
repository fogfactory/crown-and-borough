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
	nobles       []models.NobleID
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

type transferIntent struct {
	armyID          models.ArmyID
	sourceID        models.TerritoryID
	targetID        models.TerritoryID
	recipientArmyID models.ArmyID
	recipientPlayer models.PlayerID
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
		nobles:       ctx.noblesAt(army.TerritoryID),
		pending:      true,
	}
}

// cancelAttackedOriginPeaceful cancels every join and dispersion whose origin
// is the target of an attack, whoever the attacker is and whatever it wins:
// an ally, an enemy, a starving attack at strength zero, or one that later
// fails as allied_destination all count, since the rule looks at the attack,
// not its outcome. A cancelled army's intent is removed before the
// adjudicator ever sees it, so it settles like a hold: none of its troops
// leaves, and applyContestOutcomes records why.
func cancelAttackedOriginPeaceful(ctx *resolutionContext) {
	attackedOrigins := make(map[models.TerritoryID]bool, len(ctx.attacks))
	for _, attack := range ctx.attacks {
		attackedOrigins[attack.target] = true
	}
	for _, armyID := range sortedArmyMap(ctx.joins) {
		if attackedOrigins[ctx.joins[armyID].source] {
			ctx.cancelledPeaceful[armyID] = true
			delete(ctx.joins, armyID)
		}
	}
	for _, armyID := range sortedArmyMap(ctx.disperses) {
		if attackedOrigins[ctx.disperses[armyID].source] {
			ctx.cancelledPeaceful[armyID] = true
			delete(ctx.disperses, armyID)
		}
	}
}

// emitBadWeatherBlocked reports one army whose order could not run because of
// the bad weather calamity, for the season-effects section of the report.
func (ctx *resolutionContext) emitBadWeatherBlocked(army models.Army, order models.Order, regionSeed models.TerritoryID) {
	ctx.events = append(ctx.events, Event{
		Type:        EventTypeBadWeatherBlocked,
		Phase:       phaseForSeason(ctx.state.Season),
		ArmyID:      army.ID,
		OwnerID:     army.OwnerID,
		RegionSeed:  regionSeed,
		TerritoryID: army.TerritoryID,
		TargetID:    firstOrderTarget(order),
		Season:      ctx.state.Season,
		Year:        ctx.state.Year(),
	})
}

func (ctx *resolutionContext) enumerateOrder(record *orderRecord, army models.Army, isLastOrder bool) {
	order := record.order
	sourceAffected := ctx.badWeatherRegions[regionForTerritory(ctx, army.TerritoryID)]
	if order.Type == models.OrderTypeDisperse {
		if sourceAffected {
			ctx.emitBadWeatherBlocked(army, order, regionForTerritory(ctx, army.TerritoryID))
			record.invalidate("bad_weather")
			return
		}
		blockedRegion := models.TerritoryID("")
		if len(order.TargetIDs) > 0 {
			blockedRegion = regionForTerritory(ctx, order.TargetIDs[0])
		}
		order.TargetIDs = ctx.filterBadWeatherDisperseTargets(order.TargetIDs)
		order.NobleAssignments = ctx.filterBadWeatherDisperseAssignments(order.NobleAssignments, order.TargetIDs)
		record.order.TargetIDs = order.TargetIDs
		record.order.NobleAssignments = order.NobleAssignments
		if len(order.TargetIDs) == 0 {
			ctx.emitBadWeatherBlocked(army, order, blockedRegion)
			record.invalidate("bad_weather")
			return
		}
	} else if order.Type != models.OrderTypeHold && (sourceAffected || ctx.badWeatherTarget(order.TargetIDs)) {
		blockedRegion := regionForTerritory(ctx, army.TerritoryID)
		if !sourceAffected && len(order.TargetIDs) > 0 {
			blockedRegion = regionForTerritory(ctx, order.TargetIDs[0])
		}
		ctx.emitBadWeatherBlocked(army, order, blockedRegion)
		record.invalidate("bad_weather")
		return
	}
	switch order.Type {
	case models.OrderTypeAttack:
		targetID := order.TargetIDs[0]
		strength := army.Size
		if ctx.famished[army.ID] {
			strength = 0
		}
		ctx.attacks[army.ID] = &attackIntent{armyID: army.ID, source: army.TerritoryID, target: targetID, size: strength}
		ctx.attackedTerritories[targetID] = true
	case models.OrderTypeJoin:
		targetID := order.TargetIDs[0]
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
			nobles:       ctx.noblesAt(army.TerritoryID),
		}
	case models.OrderTypeSupport, models.OrderTypeHold:
		// Supports are computed from the stored order once every intention
		// is known; a hold has nothing to enumerate.
	case models.OrderTypePillage:
		if ctx.state.TerritoryStates[army.TerritoryID].Infrastructures == nil {
			record.invalidate("no_infrastructure")
		}
	case models.OrderTypeTransfer:
		targetID := order.TargetIDs[0]
		sourceState := ctx.state.TerritoryStates[army.TerritoryID]
		if sourceState.OwnerID == nil || *sourceState.OwnerID != army.OwnerID {
			record.invalidate("transfer_source_not_controlled")
			return
		}
		intent := &transferIntent{armyID: army.ID, sourceID: army.TerritoryID, targetID: targetID}
		if targetArmy := ctx.startArmyAt(targetID); targetArmy != nil {
			if targetArmy.OwnerID == army.OwnerID || !PlayerAlive(ctx.state, targetArmy.OwnerID) {
				record.invalidate("invalid_transfer_destination")
				return
			}
			intent.recipientArmyID = targetArmy.ID
			intent.recipientPlayer = targetArmy.OwnerID
		} else {
			targetState := ctx.state.TerritoryStates[targetID]
			if targetState.OwnerID == nil || *targetState.OwnerID == army.OwnerID || !PlayerAlive(ctx.state, *targetState.OwnerID) || !ctx.hasSettlement(targetID) {
				record.invalidate("invalid_transfer_destination")
				return
			}
			intent.recipientPlayer = *targetState.OwnerID
		}
		reachable := transferNetwork(ctx, army.TerritoryID, army.OwnerID, targetID)
		if _, reachable := reachable[targetID]; !reachable {
			record.invalidate("transfer_path_blocked")
			return
		}
		ctx.transfers[army.ID] = intent
	}
}

func (ctx *resolutionContext) badWeatherTarget(targets []models.TerritoryID) bool {
	for _, targetID := range targets {
		if ctx.badWeatherRegions[regionForTerritory(ctx, targetID)] {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) filterBadWeatherDisperseTargets(targets []models.TerritoryID) []models.TerritoryID {
	filtered := make([]models.TerritoryID, 0, len(targets))
	for _, targetID := range targets {
		if !ctx.badWeatherRegions[regionForTerritory(ctx, targetID)] {
			filtered = append(filtered, targetID)
		}
	}
	return filtered
}

func (ctx *resolutionContext) filterBadWeatherDisperseAssignments(assignments map[models.TerritoryID][]models.NobleCode, targets []models.TerritoryID) map[models.TerritoryID][]models.NobleCode {
	filtered := make(map[models.TerritoryID][]models.NobleCode)
	for _, targetID := range targets {
		if nobles, exists := assignments[targetID]; exists {
			filtered[targetID] = append([]models.NobleCode(nil), nobles...)
		}
	}
	return filtered
}

func (ctx *resolutionContext) validateDisperse(record *orderRecord, army models.Army) (map[models.TerritoryID][]models.NobleID, bool) {
	order := record.order
	coLocated := ctx.noblesAt(army.TerritoryID)
	assignments := make(map[models.TerritoryID][]models.NobleID, len(order.NobleAssignments))
	assigned := make(map[models.NobleID]bool, len(coLocated))
	wildcardDestination := models.TerritoryID("")
	assignmentDestinations := make([]models.TerritoryID, 0, len(order.NobleAssignments))
	for destination := range order.NobleAssignments {
		assignmentDestinations = append(assignmentDestinations, destination)
	}
	sort.Slice(assignmentDestinations, func(i, j int) bool { return assignmentDestinations[i] < assignmentDestinations[j] })
	for _, destinationID := range assignmentDestinations {
		for _, nobleCode := range order.NobleAssignments[destinationID] {
			if nobleCode == "*" {
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
