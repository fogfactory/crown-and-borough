package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Helpers shared by the cards played on an attack (Tunnel "P SO AAA - BBB",
// Ambush "P EM AAA BBB"): the validation of the designated attack, its match
// with the attack intents of the turn, and the report event.

// plannedAttack reports whether the player has an army on source whose current
// order is an attack on target.
func (ctx *resolutionContext) plannedAttack(playerID models.PlayerID, source, target models.TerritoryID) bool {
	army := ctx.startArmyAt(source)
	if army == nil || army.OwnerID != playerID || army.ChainID == nil {
		return false
	}
	chain := ctx.chainsByID[*army.ChainID]
	if chain == nil || chain.PendingDisperse != nil || chain.CurrentIndex < 0 || chain.CurrentIndex >= len(chain.Orders) {
		return false
	}
	order := chain.Orders[chain.CurrentIndex]
	return order.Type == models.OrderTypeAttack && order.PositionID == source && len(order.TargetIDs) > 0 && order.TargetIDs[0] == target
}

// designatedAttackPossible reports whether the player may designate the attack
// from source to target. An army of the player must have that attack as its
// current order. An army of another player is free to attack or not, since its
// orders are not known: the attack only has to be possible, between two
// adjacent territories (an advice that profits the attacker if it happens).
func (ctx *resolutionContext) designatedAttackPossible(playerID models.PlayerID, source, target models.TerritoryID) bool {
	army := ctx.startArmyAt(source)
	if army == nil {
		return false
	}
	if army.OwnerID == playerID {
		return ctx.plannedAttack(playerID, source, target)
	}
	return army.OwnerID != models.NeutralPlayerID && ctx.isAdjacent(source, target)
}

// designatedAttacker returns the army that carries out the attack designated
// by order among the intents enumerated for the turn, or nil when the attack
// does not take place (a canceled or invalid order).
func (ctx *resolutionContext) designatedAttacker(order models.DeckOrder) *models.Army {
	army := ctx.startArmyAt(order.SourceTerritoryID)
	if army == nil {
		return nil
	}
	if attack := ctx.attacks[army.ID]; attack == nil || attack.target != order.TargetTerritoryID {
		return nil
	}
	return army
}

// attackCardEvent builds the report event of a card played on an attack: the
// origin of the attack is the event's source and target, the attacked
// territory its territory.
func (ctx *resolutionContext) attackCardEvent(eventType EventType, kind models.CardKind, pending deckOrderIntent, reason string) Event {
	return Event{
		Type: eventType, Phase: phaseForSeason(ctx.state.Season), OwnerID: pending.playerID, OrderID: pending.order.ID,
		CardKind: kind, Reason: reason, SourceID: pending.order.SourceTerritoryID,
		TargetID: pending.order.SourceTerritoryID, TerritoryID: pending.order.TargetTerritoryID,
		Season: ctx.state.Season, Year: ctx.state.Year(),
	}
}
