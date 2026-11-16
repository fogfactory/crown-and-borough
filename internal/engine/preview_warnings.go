package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Levels of a preview warning.
const (
	// WarningLevelWarning flags an order that will not run as written.
	WarningLevelWarning = "warning"
	// WarningLevelNote informs of a possibly intended situation.
	WarningLevelNote = "note"
	// WarningLevelError flags a line the engine will reject: the submission
	// stays accepted, but the line does nothing.
	WarningLevelError = "error"
)

// PreviewWarning is a non-blocking notice of an orders preview: the
// submission is accepted, but an order may not run as written. It reuses the
// shape of an InputError (code, localizable message, noble, line) and adds a
// level.
type PreviewWarning struct {
	InputError
	Level string `json:"level"`
}

// previewChainWarnings reports, for the armies of playerID, what the start of
// the turn already tells about the order each will execute: the chains of the
// submission and the chains under way that the submission leaves in place. It
// reads the start state and the player's own orders only, never another
// player's chain, so a warning cannot reveal an opponent's intentions.
//
// Only the current order of a chain is checked: the later lines depend on
// where the army will stand. A chain under way carries no source line.
func previewChainWarnings(game *models.GameState, balance assetgen.Balance, playerID models.PlayerID, input OrdersInput) []PreviewWarning {
	warnings := []PreviewWarning{}
	working := cloneGameState(game)

	submitted := make(map[models.ArmyID]models.NobleCode)
	openingLines := make(map[models.ArmyID]int)
	type receivable struct {
		chain models.Chain
		army  *models.Army
	}
	targets := make(map[models.ArmyID]int)
	parsed := make([]receivable, 0, len(input.Chains))
	for _, submission := range input.Chains {
		if submission.Player != playerID {
			continue
		}
		chain, parseErrors := orders.ParseChain(submission.Text, game)
		if len(parseErrors) != 0 || len(orders.ValidateChain(game, chain)) != 0 {
			continue
		}
		army := orders.ReceivingArmy(working, chain)
		parsed = append(parsed, receivable{chain: chain, army: army})
		if army != nil {
			targets[army.ID]++
		}
	}
	for _, entry := range parsed {
		chain, army := entry.chain, entry.army
		noble, found := findNoble(working.Nobles, chain.NobleID)
		if !found || noble.OwnerID != playerID || orders.ValidateReceivingArmyOwnership(working, chain, playerID) != nil {
			continue
		}
		if army == nil || targets[army.ID] != 1 {
			continue
		}
		var replaced bool
		if army.ChainID != nil {
			for _, stored := range working.Chains {
				if stored.ID == *army.ChainID && stored.CurrentIndex < len(stored.Orders) {
					replaced = true
				}
			}
		}
		if err := orders.AssignChain(working, chain); err != nil {
			continue
		}
		submitted[army.ID] = models.NobleCode(noble.Code)
		openingLines[army.ID] = chain.Orders[0].Line
		if replaced {
			warnings = append(warnings, newPreviewWarning(playerID, models.NobleCode(noble.Code), 0, "chain_replaced", i18n.WarningChainReplaced, string(army.TerritoryID)))
		}
	}

	ctx := newResolutionContext(working, balance)
	enumerateIntentions(ctx)
	for _, armyID := range sortedArmyMap(ctx.records) {
		army := ctx.startArmiesByID[armyID]
		if army.OwnerID != playerID {
			continue
		}
		record := ctx.records[armyID]
		submittedNoble, isSubmitted := submitted[armyID]
		warnings = append(warnings, orderWarnings(ctx, playerID, army, record, isSubmitted, submittedNoble, openingLines[armyID])...)
	}
	return warnings
}

// orderWarnings checks the current order of one army of the player.
func orderWarnings(ctx *resolutionContext, playerID models.PlayerID, army models.Army, record *orderRecord, isSubmitted bool, submittedNoble models.NobleCode, line int) []PreviewWarning {
	order := record.order
	noble := submittedNoble
	if !isSubmitted {
		noble, line = "", 0
		if chain := ctx.chainsByID[record.chainID]; chain != nil {
			if emitter, found := findNoble(ctx.state.Nobles, chain.NobleID); found {
				noble = models.NobleCode(emitter.Code)
			}
		}
	}
	territory := string(army.TerritoryID)
	inapplicable := func(code, key string, args ...any) []PreviewWarning {
		return inapplicableOrder(playerID, noble, line, isSubmitted, army, order, code, key, args...)
	}

	if record.outcome == OutcomeFailure || record.outcome == OutcomeInvalid {
		switch record.reason {
		case "position_mismatch":
			return inapplicable("position", "")
		case "famished_attacker":
			return inapplicable("starving_attack", i18n.WarningStarvingAttack, territory)
		case "no_infrastructure":
			return inapplicable("pillage_no_infrastructure", i18n.WarningPillageNoInfra, territory)
		case "ruined_infrastructure":
			return inapplicable("pillage_ruin", i18n.WarningPillageRuin, territory)
		case "transfer_source_not_controlled":
			return inapplicable("transfer_source_lost", i18n.WarningTransferSourceLost, territory)
		case "invalid_transfer_destination", "transfer_target_occupied":
			return inapplicable("transfer_destination", i18n.WarningTransferDestination, territory)
		case "transfer_path_blocked":
			if isSubmitted {
				// Already refused at submission.
				return nil
			}
			return inapplicable("transfer_blocked", i18n.WarningTransferBlocked, territory)
		case "invalid_disperse_noble_assignment":
			return inapplicable("disperse_nobles", i18n.WarningDisperseNobles, territory)
		}
		return nil
	}

	switch order.Type {
	case models.OrderTypeAttack:
		targetID := order.TargetIDs[0]
		if target := ctx.startArmyAt(targetID); target != nil && target.OwnerID == playerID && !armyMoves(ctx.records[target.ID]) {
			return []PreviewWarning{newPreviewWarning(playerID, noble, line, "attack_own_army", i18n.WarningAttackOwnArmy, territory, string(targetID))}
		}
	case models.OrderTypeTransfer:
		if ctx.famished[army.ID] {
			return inapplicable("starving_transfer", i18n.WarningStarvingTransfer, territory)
		}
	case models.OrderTypeDisperse:
		if order.Liaison == models.LiaisonModeLoop && len(order.TargetIDs) > army.Size {
			return inapplicable("disperse_too_many", i18n.WarningDisperseTooMany, territory, len(order.TargetIDs), army.Size)
		}
	case models.OrderTypeSupport:
		return supportWarnings(ctx, playerID, army, record, isSubmitted, noble, line)
	}
	return nil
}

// inapplicableOrder is the warning of an order that cannot run. On a submitted
// line it says why. For a chain under way, which the player did not just write,
// it says that the chain is at risk: sending a new order to the army is the way
// out, otherwise the chain is lost.
func inapplicableOrder(playerID models.PlayerID, noble models.NobleCode, line int, isSubmitted bool, army models.Army, order models.Order, code, key string, args ...any) []PreviewWarning {
	if isSubmitted && key != "" {
		return []PreviewWarning{newPreviewWarning(playerID, noble, line, code, key, args...)}
	}
	if isSubmitted {
		return nil
	}
	return []PreviewWarning{newPreviewWarning(playerID, noble, 0, "running_chain_"+code, i18n.WarningRunningChainInapplied, string(army.TerritoryID), orderSymbol(order.Type))}
}

// supportWarnings checks a support. The army it supports is judged by its own
// order only when it belongs to the player, since that order is known; an army
// of another player only earns a note.
func supportWarnings(ctx *resolutionContext, playerID models.PlayerID, army models.Army, record *orderRecord, isSubmitted bool, noble models.NobleCode, line int) []PreviewWarning {
	order := record.order
	territory := string(army.TerritoryID)
	warnings := []PreviewWarning{}
	if ctx.famished[army.ID] {
		warnings = append(warnings, inapplicableOrder(playerID, noble, line, isSubmitted, army, order, "starving_support", i18n.WarningStarvingSupport, territory)...)
	}
	supportedID := order.TargetIDs[0]
	supported := ctx.startArmyAt(supportedID)
	if supported == nil {
		return warnings
	}
	supportedOrder, known := knownOrder(ctx, playerID, *supported)
	if !known {
		owner := string(supported.OwnerID)
		for _, player := range ctx.state.Players {
			if player.ID == supported.OwnerID {
				owner = player.Name
			}
		}
		return append(warnings, newPreviewWarningLevel(WarningLevelNote, playerID, noble, line, "support_opposing", i18n.WarningSupportOpposing, string(supportedID), owner))
	}
	attacks := supportedOrder != nil && supportedOrder.Type == models.OrderTypeAttack
	if len(order.TargetIDs) == 1 {
		if attacks {
			warnings = append(warnings, newPreviewWarning(playerID, noble, line, "support_attacker", i18n.WarningSupportAttacker, string(supportedID), string(supportedID)))
		}
		return warnings
	}
	destination := order.TargetIDs[1]
	if !attacks || supportedOrder.TargetIDs[0] != destination {
		// A submitted chain opens on this order, and submission already
		// refuses it when the army is the player's own (support_no_attack):
		// no need to say it twice.
		if !(isSubmitted && supported.OwnerID == playerID) {
			warnings = append(warnings, newPreviewWarning(playerID, noble, line, "support_no_attack", i18n.WarningSupportNoAttack, string(supportedID), string(destination), territory))
		}
	}
	return warnings
}

// knownOrder is the order the army executes this turn, when the player can
// know it: an army of the player, or one whose chain under way the player has
// learned (a hostage's chain read by its captor, even once the hostage is
// freed, or a chain revealed by an information dignity), as the privacy
// knowledge of the state records. The order is nil for a known army without an
// applicable order, which stays where it is. known is false for any other army.
func knownOrder(ctx *resolutionContext, playerID models.PlayerID, army models.Army) (order *models.Order, known bool) {
	if army.OwnerID != playerID && !chainKnownTo(ctx.state, playerID, army) {
		return nil, false
	}
	record := ctx.records[army.ID]
	if record == nil || record.outcome == OutcomeInvalid || record.outcome == OutcomeFailure {
		return nil, true
	}
	return &record.order, true
}

// armyMoves reports whether the order of an army of the player moves it: an
// attack, a join or a dispersion that can run. An army without an order, or
// holding, supporting, pillaging or transferring, stays on its territory.
func armyMoves(record *orderRecord) bool {
	if record == nil || record.outcome == OutcomeInvalid || record.outcome == OutcomeFailure {
		return false
	}
	switch record.order.Type {
	case models.OrderTypeAttack, models.OrderTypeJoin, models.OrderTypeDisperse:
		return true
	}
	return false
}

// chainKnownTo reports whether playerID has learned the chain under way of
// army.
func chainKnownTo(state *models.GameState, playerID models.PlayerID, army models.Army) bool {
	if army.ChainID == nil || state.Privacy == nil {
		return false
	}
	_, known := state.Privacy.ChainKnowledge[playerID][*army.ChainID]
	return known
}

func newPreviewWarning(player models.PlayerID, noble models.NobleCode, line int, code, key string, args ...any) PreviewWarning {
	return newPreviewWarningLevel(WarningLevelWarning, player, noble, line, code, key, args...)
}

func newPreviewWarningLevel(level string, player models.PlayerID, noble models.NobleCode, line int, code, key string, args ...any) PreviewWarning {
	warning := PreviewWarning{InputError: newInputError(player, noble, line, code, key, args...), Level: level}
	warning.Source = InputSourceChain
	return warning
}

func orderSymbol(orderType models.OrderType) string {
	switch orderType {
	case models.OrderTypeAttack:
		return "A"
	case models.OrderTypeSupport:
		return "S"
	case models.OrderTypeHold:
		return "H"
	case models.OrderTypeJoin:
		return "J"
	case models.OrderTypePillage:
		return "P"
	case models.OrderTypeDisperse:
		return "D"
	case models.OrderTypeTransfer:
		return "T"
	}
	return string(orderType)
}

// rejectedWinterWarnings turns the winter lines the simulation rejects into
// warnings. The code is the engine's rejection reason, which has no server-side
// message: the client words it. A line the player cannot pay yet is a plain
// warning, since the reserves can still change before the sheet resolves.
func rejectedWinterWarnings(playerID models.PlayerID, lines []WinterLinePreview) []PreviewWarning {
	warnings := []PreviewWarning{}
	for _, line := range lines {
		if line.Applied || line.ParseError != nil || line.Discard != nil || (line.Order == nil && line.Play == nil) {
			continue
		}
		// A line rejected without a reason comes from a payment that fell
		// short: the sheet reads it as insufficient resources.
		reason := line.Reason
		if reason == "" {
			reason = "insufficient_resources"
		}
		level := WarningLevelError
		if reason == "insufficient_resources" {
			level = WarningLevelWarning
		}
		warning := PreviewWarning{
			InputError: InputError{Player: playerID, Line: line.Line, Source: InputSourceWinter, Code: reason, Message: reason},
			Level:      level,
		}
		warnings = append(warnings, warning)
	}
	return warnings
}
