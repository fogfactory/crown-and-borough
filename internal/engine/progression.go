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
	if record.order.Type == models.OrderTypeTransfer {
		if record.outcome == OutcomeFailure && record.reason == "insufficient_resources" {
			if record.order.Liaison == models.LiaisonModeLoop {
				return before, before, ProgressionRetried
			}
			return ctx.advanceChain(record.chainID, before)
		}
		if record.outcome == OutcomeSuccess && record.order.Liaison == models.LiaisonModeLoop {
			if state, exists := ctx.state.TerritoryStates[record.order.PositionID]; exists && state.Resources > 0 {
				return before, before, ProgressionRetried
			}
		}
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
		// Bad weather is a temporary circumstance, not a player error: the
		// chain pauses on the same order and re-attempts it next season.
		if record.reason == "bad_weather" {
			return before, before, ProgressionRetried
		}
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
		if attack == nil || attack.target != support.destinationID {
			return false
		}
		record := ctx.records[attack.armyID]
		return record == nil || record.outcome != OutcomeSuccess
	}
	if support.targetArmyID == "" {
		return false
	}
	army := ctx.armiesByID[support.targetArmyID]
	return army != nil && army.TerritoryID == support.targetID
}

// updateTerritorialControl runs the two deterministic passes titres.md
// describes: control, then occupation. A fief makes control transitive
// (titres.md "Contrôle et occupation"): a non-capital member stays controlled
// by the fief's owner regardless of which army stops on it, and only the
// capital's capture transfers the whole fief (transferFiefOnCapitalCapture).
// Both passes iterate armies sorted by ID, which is also a stable per-army,
// per-territory order since exactly one army can occupy a territory.
func updateTerritorialControl(ctx *resolutionContext) {
	startOwnerAt := make(map[models.TerritoryID]models.PlayerID, len(ctx.startArmyAtTerritory))
	for territoryID, armyID := range ctx.startArmyAtTerritory {
		startOwnerAt[territoryID] = ctx.startArmiesByID[armyID].OwnerID
	}
	startFiefOwnerAt := make(map[models.FiefID]models.PlayerID, len(ctx.state.Fiefs))
	for _, fief := range ctx.state.Fiefs {
		startFiefOwnerAt[fief.ID] = fief.OwnerID
	}
	// fiefAt indexes every fief by each of its member territories once, up
	// front: neither pass appends to or removes from ctx.state.Fiefs (only
	// transferFiefOnCapitalCapture mutates a fief's OwnerID/HolderNobleID in
	// place), so the *models.Fief pointers below stay valid and up to date
	// across both passes instead of each re-scanning all fiefs per army.
	fiefAt := make(map[models.TerritoryID]*models.Fief, len(ctx.state.Fiefs))
	for i := range ctx.state.Fiefs {
		fief := &ctx.state.Fiefs[i]
		for _, member := range fief.Territories {
			fiefAt[member] = fief
		}
	}

	// Pass 1: control. A non-capital fief member never changes OwnerID under
	// a visiting army: it is occupied, not conquered.
	for _, armyID := range sortedArmyMap(ctx.armiesByID) {
		army := ctx.armiesByID[armyID]
		if army.OwnerID == models.NeutralPlayerID {
			// Rebel armies occupy territory without administering it: control
			// stays with the previous owner until an army of a player stops.
			continue
		}
		if fief := fiefAt[army.TerritoryID]; fief != nil && fief.CapitalTerritoryID != army.TerritoryID && fief.OwnerID != army.OwnerID {
			continue
		}
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
		ctx.transferFiefOnCapitalCapture(army.TerritoryID, ownerID)
		ctx.events = append(ctx.events, Event{
			Type:            EventTypeControlChanged,
			Phase:           5,
			TerritoryID:     army.TerritoryID,
			PreviousOwnerID: previousOwnerID,
			OwnerID:         ownerID,
		})
	}

	// Pass 2: occupation. Any army (including a NEUTRAL revolt) stationed on
	// a non-capital fief member whose owner differs from the fief's owner
	// occupies it against its controller. Report it once, the turn it
	// starts: a garrison that merely stays under an unchanged controller does
	// not repeat the event. A controller that changed this same turn (the
	// capital just fell, per pass 1's transferFiefOnCapitalCapture) always
	// reports its now-occupied members, even to an army that was already
	// sitting there under the previous controller.
	for _, armyID := range sortedArmyMap(ctx.armiesByID) {
		army := ctx.armiesByID[armyID]
		fief := fiefAt[army.TerritoryID]
		if fief == nil || fief.CapitalTerritoryID == army.TerritoryID || fief.OwnerID == army.OwnerID {
			continue
		}
		startOwnerID, armyWasPresent := startOwnerAt[army.TerritoryID]
		controllerUnchanged := startFiefOwnerAt[fief.ID] == fief.OwnerID
		if armyWasPresent && startOwnerID == army.OwnerID && controllerUnchanged {
			continue
		}
		ctx.events = append(ctx.events, Event{
			Type:           EventTypeFiefMemberOccupied,
			Phase:          5,
			ArmyID:         army.ID,
			TerritoryID:    army.TerritoryID,
			DestinationID:  fief.CapitalTerritoryID,
			FiefID:         fief.ID,
			FiefTitle:      fief.Title,
			OwnerID:        fief.OwnerID,
			CaptorPlayerID: army.OwnerID,
		})
	}

	// A fief dissolved this same turn (its capital's castle pillaged) already
	// lost its membership above, in removeInfrastructureWithStock, before
	// either pass ran: a former member without a fresh army of its own is
	// released here, in the same pass that releases every other unanchored
	// territory (#215).
	ctx.releaseUnanchoredControl()
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
