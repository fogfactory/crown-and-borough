package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

func progressChainsAndControl(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.records) {
		record := ctx.records[armyID]
		if record.outcome == "" {
			record.invalidate("unresolved_order")
		}
		before, after, progression := ctx.progressRecord(record)
		record.progression = progression
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeOrderOutcome,
			Phase:       5,
			ArmyID:      record.armyID,
			ChainID:     record.chainID,
			OrderID:     record.order.ID,
			OrderType:   record.order.Type,
			Outcome:     record.outcome,
			Reason:      record.reason,
			Progression: progression,
			SourceID:    record.order.PositionID,
			TargetID:    firstOrderTarget(record.order),
		})
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeChainProgression,
			Phase:       5,
			ArmyID:      record.armyID,
			ChainID:     record.chainID,
			OrderID:     record.order.ID,
			Outcome:     record.outcome,
			Progression: progression,
			IndexBefore: before,
			IndexAfter:  after,
		})
	}
	updateTerritorialControl(ctx)
}

func firstOrderTarget(order models.Order) models.TerritoryID {
	if len(order.TargetIDs) == 0 {
		return ""
	}
	return order.TargetIDs[0]
}

func (ctx *resolutionContext) progressRecord(record *orderRecord) (int, int, Progression) {
	chain := ctx.chainsByID[record.chainID]
	if chain == nil {
		return 0, 0, ProgressionBroken
	}
	before := chain.CurrentIndex
	if record.destroyed {
		ctx.removeChain(record.chainID)
		return before, before, ProgressionBroken
	}
	if record.fused {
		ctx.removeChain(record.chainID)
		return before, len(chain.Orders), ProgressionConsumed
	}

	switch record.outcome {
	case OutcomeSuccess:
		if record.order.Liaison == models.LiaisonModeLoop && record.order.Type == models.OrderTypeHold {
			return before, before, ProgressionRetried
		}
		if record.order.Liaison == models.LiaisonModeLoop && record.order.Type == models.OrderTypeSupport && ctx.freezeLoopSupport(record.armyID) {
			return before, before, ProgressionRetried
		}
		return ctx.advanceChain(record.chainID, before)
	case OutcomeFailure:
		if record.partialD {
			if record.order.Liaison == models.LiaisonModeSingle {
				return ctx.advanceChain(record.chainID, before)
			}
			return before, before, ProgressionRetried
		}
		if record.order.Liaison == models.LiaisonModeLoop {
			return before, before, ProgressionRetried
		}
		ctx.removeChain(record.chainID)
		return before, before, ProgressionBroken
	case OutcomeInvalid:
		ctx.removeChain(record.chainID)
		return before, before, ProgressionBroken
	default:
		ctx.removeChain(record.chainID)
		return before, before, ProgressionBroken
	}
}

func (ctx *resolutionContext) advanceChain(chainID models.ChainID, before int) (int, int, Progression) {
	chain := ctx.chainsByID[chainID]
	if chain == nil {
		return before, before, ProgressionBroken
	}
	chain.CurrentIndex++
	after := chain.CurrentIndex
	if after >= len(chain.Orders) {
		ctx.removeChain(chainID)
		return before, after, ProgressionConsumed
	}
	return before, after, ProgressionAdvanced
}

func (ctx *resolutionContext) removeChain(chainID models.ChainID) {
	chains := make([]models.Chain, 0, len(ctx.state.Chains)-1)
	for _, chain := range ctx.state.Chains {
		if chain.ID != chainID {
			chains = append(chains, chain)
		}
	}
	ctx.state.Chains = chains
	for index := range ctx.state.Armies {
		army := &ctx.state.Armies[index]
		if army.ChainID != nil && *army.ChainID == chainID {
			army.ChainID = nil
		}
	}
	ctx.rebuildIndexes()
}

func (ctx *resolutionContext) freezeLoopSupport(armyID models.ArmyID) bool {
	support := ctx.supports[armyID]
	if support == nil {
		return false
	}
	if support.offensive {
		attack := ctx.attacks[support.targetArmyID]
		return attack != nil && attack.target == support.destinationID && ctx.contest.active[attack.armyID]
	}
	return ctx.attackedTerritories[support.targetID]
}

func updateTerritorialControl(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.armiesByID) {
		army := ctx.armiesByID[armyID]
		state := ctx.state.TerritoryStates[army.TerritoryID]
		if state.OwnerID != nil && *state.OwnerID == army.OwnerID {
			continue
		}
		previousOwnerID := models.PlayerID("")
		if state.OwnerID != nil {
			previousOwnerID = *state.OwnerID
		}
		ownerID := army.OwnerID
		state.OwnerID = &ownerID
		ctx.state.TerritoryStates[army.TerritoryID] = state
		ctx.clearCapitalOnControlLoss(previousOwnerID, army.TerritoryID)
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeControlChanged,
			Phase:           5,
			TerritoryID:     army.TerritoryID,
			PreviousOwnerID: previousOwnerID,
			OwnerID:         ownerID,
		})
	}
}

func (ctx *resolutionContext) clearCapitalOnControlLoss(previousOwnerID models.PlayerID, territoryID models.TerritoryID) {
	if previousOwnerID == "" {
		return
	}
	for index := range ctx.state.Players {
		player := &ctx.state.Players[index]
		if player.ID != previousOwnerID || player.CapitalCastleID == nil {
			continue
		}
		infrastructure := ctx.infrastructuresByID[*player.CapitalCastleID]
		if infrastructure != nil && infrastructure.TerritoryID == territoryID {
			player.CapitalCastleID = nil
		}
	}
}
