package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Reasons carried by a tunnel event: the escape of a detained noble, an attack
// that ignores the fortifications of its target, and a designated attack that
// did not take place.
const (
	TunnelEscapeReason = "tunnel_escape"
	TunnelBypassReason = "tunnel_bypass"
	TunnelVoidReason   = "tunnel_void"
)

// tunnelCardDefinition is the tunnel card (specs/ordres-speciaux.md
// § Souterrain, #413), with two uses told apart by the order:
//
//   - "P SO NNN" (TargetNobleID) frees the player's noble NNN, detained by
//     another player's army;
//   - "P SO AAA - BBB" (SourceTerritoryID, TargetTerritoryID) designates the
//     attack the player plans from AAA to BBB, which ignores the fortifications
//     of BBB.
//
// The card is consumed when it is played. An escape applies on the spot, before
// supply and the army orders; an attack is only matched with the enumerated
// attack intents afterwards (applyPendingTunnels).
type tunnelCardDefinition struct{}

type tunnelOrder struct {
	playerID models.PlayerID
	order    models.DeckOrder
}

func (tunnelCardDefinition) Kind() models.CardKind { return models.CardKindTunnel }

func (tunnelCardDefinition) CanPlay(ctx *ExecutionContext, order models.DeckOrder) (bool, string) {
	if ctx.season == models.SeasonWinter {
		return false, "deck_order_out_of_season"
	}
	resolution := ctx.resolution
	if order.SourceTerritoryID != "" {
		if !resolution.plannedAttack(ctx.playerID, order.SourceTerritoryID, order.TargetTerritoryID) {
			return false, "tunnel_no_attack"
		}
		return true, ""
	}
	noble := resolution.noblesByID[order.TargetNobleID]
	if noble == nil || resolution.tunnelDetainer(ctx.playerID, noble) == nil {
		return false, "noble_not_held"
	}
	if _, found := resolution.tunnelRefuge(ctx.playerID); !found {
		return false, "tunnel_no_refuge"
	}
	return true, ""
}

func (tunnelCardDefinition) NewOrder(playerID models.PlayerID, order models.DeckOrder) ExecutableOrder {
	return tunnelOrder{playerID: playerID, order: order}
}

func (order tunnelOrder) Apply(ctx *ExecutionContext) {
	definition := tunnelCardDefinition{}
	if applicable, reason := definition.CanPlay(ctx, order.order); !applicable {
		ctx.resolution.rejectDeckOrderReason(order.playerID, order.order, reason)
		return
	}
	resolution := ctx.resolution
	if !resolution.applyDeckCardOrder(order.playerID, order.order) {
		return
	}
	if order.order.SourceTerritoryID != "" {
		resolution.pendingTunnels = append(resolution.pendingTunnels, deckOrderIntent{playerID: order.playerID, order: order.order})
		return
	}
	resolution.tunnelEscape(order.playerID, order.order)
}

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

// applyPendingTunnels matches each tunnel card played on an attack with the
// attack intents enumerated for the turn: the attack then ignores the
// fortifications of its target. An attack that did not materialize (a canceled
// or invalid order) leaves the card consumed with no effect.
func applyPendingTunnels(ctx *resolutionContext) {
	for _, pending := range ctx.pendingTunnels {
		event := Event{
			Type: EventTypeTunnel, Phase: phaseForSeason(ctx.state.Season), OwnerID: pending.playerID, OrderID: pending.order.ID,
			CardKind: models.CardKindTunnel, Reason: TunnelVoidReason, SourceID: pending.order.SourceTerritoryID,
			TargetID: pending.order.SourceTerritoryID, TerritoryID: pending.order.TargetTerritoryID,
			Season: ctx.state.Season, Year: ctx.state.Year(),
		}
		if army := ctx.startArmyAt(pending.order.SourceTerritoryID); army != nil && army.OwnerID == pending.playerID {
			if attack := ctx.attacks[army.ID]; attack != nil && attack.target == pending.order.TargetTerritoryID {
				ctx.tunneledAttacks[army.ID] = true
				event.Reason, event.ArmyID = TunnelBypassReason, army.ID
			}
		}
		ctx.events = append(ctx.events, event)
	}
}

// tunnelDetainer returns the army of another player that holds the player's
// own noble hostage or in a dungeon, or nil when the noble is not detained.
func (ctx *resolutionContext) tunnelDetainer(playerID models.PlayerID, noble *models.Noble) *models.Army {
	if noble.OwnerID != playerID || noble.Status == models.NobleStatusFree {
		return nil
	}
	army := ctx.currentArmyAt(noble.LocationID)
	if army == nil || army.OwnerID == playerID || army.OwnerID == models.NeutralPlayerID {
		return nil
	}
	return army
}

// tunnelRefuge returns where an escaped noble of the player reappears: their
// capital while they still control it, else their controlled castle of smallest
// code. A territory held by an army of another player is skipped, so that the
// noble never ends up riding a foreign army.
func (ctx *resolutionContext) tunnelRefuge(playerID models.PlayerID) (models.TerritoryID, bool) {
	for index := range ctx.state.Players {
		player := &ctx.state.Players[index]
		if player.ID != playerID || player.CapitalCastleID == nil {
			continue
		}
		if capital := ctx.infrastructuresByID[*player.CapitalCastleID]; capital != nil && ctx.tunnelRefugeUsable(playerID, capital.TerritoryID) {
			return capital.TerritoryID, true
		}
	}
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		if ctx.hasCastle(territoryID) && ctx.tunnelRefugeUsable(playerID, territoryID) {
			return territoryID, true
		}
	}
	return "", false
}

func (ctx *resolutionContext) tunnelRefugeUsable(playerID models.PlayerID, territoryID models.TerritoryID) bool {
	if !controlledBy(ctx.controllerAtStart, playerID, territoryID) {
		return false
	}
	army := ctx.currentArmyAt(territoryID)
	return army == nil || army.OwnerID == playerID
}

// tunnelEscape frees the detained noble and puts them back on their refuge.
func (ctx *resolutionContext) tunnelEscape(playerID models.PlayerID, order models.DeckOrder) {
	noble := ctx.noblesByID[order.TargetNobleID]
	detainer := ctx.tunnelDetainer(playerID, noble)
	refuge, _ := ctx.tunnelRefuge(playerID)
	previousStatus, source := noble.Status, noble.LocationID
	noble.Status = models.NobleStatusFree
	noble.LocationID = refuge
	ctx.events = append(ctx.events, Event{
		Type: EventTypeTunnel, Phase: phaseForSeason(ctx.state.Season), OwnerID: playerID, OrderID: order.ID,
		CardKind: models.CardKindTunnel, Reason: TunnelEscapeReason,
		NobleID: noble.ID, NobleCode: models.NobleCode(noble.Code), NobleName: ctx.state.NobleDisplayName(*noble),
		PreviousStatus: previousStatus, Status: noble.Status, SourceID: source, TargetID: source,
		TerritoryID: refuge, CaptorPlayerID: detainer.OwnerID, ArmyID: detainer.ID,
		Season: ctx.state.Season, Year: ctx.state.Year(),
	})
}
