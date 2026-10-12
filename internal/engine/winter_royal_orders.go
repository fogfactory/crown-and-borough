package engine

import (
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type royalOrder struct{ order models.WinterOrder }

// resolveRoyalOrders runs stage 2a of the winter (hiver.md § Ordre de
// résolution): the royal orders of every sheet, players by identifier then
// sheet order, before the management orders that read the accords they fill.
func (ctx *resolutionContext) resolveRoyalOrders(orders map[models.PlayerID][]models.WinterOrder, firstNameRNG *rand.Rand) {
	ctx.winterSheets = orders
	for _, playerID := range sortedPlayerIDs(ctx.state.Players) {
		for _, order := range orders[playerID] {
			if order.Type.IsRoyal() {
				executeWinterOrder(ctx, playerID, order, firstNameRNG)
			}
		}
	}
}

// Apply executes a royal order (W X, W R, W F, W G; politique.md § Pouvoirs
// royaux). They are free. The issuer must be the king or the regent, and a
// king issues at most royal.orders_per_winter orders of each type per winter.
// An accord (W X, W R, W F) whose management order is absent from this
// winter's sheets is consumed without effect and reported as rejected.
func (order royalOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order

	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	issuer, reigns := resolution.state.RoyalIssuer()
	if noble == nil || noble.OwnerID != playerID || !reigns || issuer.ID != nobleID {
		resolution.rejectWinterOrder(playerID, winterOrder, "royal_not_issuer")
		return
	}
	if resolution.royalOrdersIssued[winterOrder.Type] >= resolution.balance.Royal.OrdersPerWinter {
		resolution.rejectWinterOrder(playerID, winterOrder, "royal_order_limit")
		return
	}

	capitalID := winterOrder.TerritoryID
	if winterOrder.Type == models.WinterOrderTypeRoyalFound {
		// The fief does not exist yet: the T F that founds it is the order authorized.
		if !resolution.hasSheetOrder(func(other models.WinterOrder, _ models.PlayerID) bool {
			return other.Type == models.WinterOrderTypeFoundFief && other.TerritoryID == capitalID
		}) {
			resolution.rejectWinterOrder(playerID, winterOrder, "royal_accord_unused")
			return
		}
		resolution.royalAccords.grantFound(capitalID)
		resolution.emitRoyalOrder(EventTypeRoyalAccord, playerID, winterOrder, nil, noble)
		return
	}

	fief := resolution.fiefByCapital(capitalID)
	if fief == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_found")
		return
	}
	switch winterOrder.Type {
	case models.WinterOrderTypeRoyalAnnex:
		if !resolution.hasSheetOrder(func(other models.WinterOrder, sheetPlayer models.PlayerID) bool {
			return sheetPlayer == fief.OwnerID && other.Type == models.WinterOrderTypeAnnexTerritory &&
				other.TerritoryID == capitalID && other.TargetID == winterOrder.TargetID
		}) {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "royal_accord_unused", winterOrder.TargetID)
			return
		}
		resolution.royalAccords.grantAnnex(capitalID, winterOrder.TargetID)
		resolution.emitRoyalOrder(EventTypeRoyalAccord, playerID, winterOrder, fief, noble)
	case models.WinterOrderTypeRoyalRank:
		if !resolution.hasSheetOrder(func(other models.WinterOrder, sheetPlayer models.PlayerID) bool {
			return sheetPlayer == fief.OwnerID && other.Type == models.WinterOrderTypeChangeRank && other.TerritoryID == capitalID
		}) {
			resolution.rejectWinterOrder(playerID, winterOrder, "royal_accord_unused")
			return
		}
		resolution.royalAccords.grantRank(capitalID)
		resolution.emitRoyalOrder(EventTypeRoyalAccord, playerID, winterOrder, fief, noble)
	case models.WinterOrderTypeRoyalGrant:
		order.grantCrownFief(resolution, playerID, fief, noble)
	}
}

// grantCrownFief gives a crown fief to a noble of any player (W G): the fief
// must be vacant and marked as a crown fief (or, for a state that predates the
// mark, belong to a player with no noble left alive), and the grantee must be
// able to receive its title in his line of succession.
func (order royalOrder) grantCrownFief(resolution *resolutionContext, playerID models.PlayerID, fief *models.Fief, issuer *models.Noble) {
	winterOrder := order.order
	if fief.HolderNobleID != nil || (!fief.Crown && resolution.playerHasNoble(fief.OwnerID)) {
		resolution.rejectWinterOrder(playerID, winterOrder, "royal_fief_not_crown")
		return
	}
	granteeID, exists := resolution.noblesByCode[winterOrder.TargetCode]
	grantee := resolution.noblesByID[granteeID]
	if !exists || grantee == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	if !resolution.state.CanReceiveTitle(granteeID, fief.Title) {
		resolution.rejectWinterOrder(playerID, winterOrder, "succession_rank_blocked")
		return
	}

	previousOwnerID := fief.OwnerID
	if grantee.OwnerID != previousOwnerID {
		for _, memberID := range fief.Territories {
			resolution.clearCapitalOnControlLoss(previousOwnerID, memberID)
		}
		fief.OwnerID = grantee.OwnerID
	}
	holderID := granteeID
	fief.HolderNobleID = &holderID
	fief.Crown = false
	resolution.royalOrdersIssued[winterOrder.Type]++
	orderCopy := winterOrder
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeFiefGranted,
		Phase:           winterPhase,
		OwnerID:         playerID,
		PreviousOwnerID: previousOwnerID,
		OrderID:         winterOrder.ID,
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		NobleID:         granteeID,
		NobleCode:       models.NobleCode(grantee.Code),
		NobleName:       resolution.state.NobleDisplayName(*grantee),
		WinterOrder:     &orderCopy,
	})
}

// emitRoyalOrder records an accord granted by a royal order and counts it
// against the winter limit. fief is nil for a W F, whose fief does not exist
// yet.
func (ctx *resolutionContext) emitRoyalOrder(eventType EventType, playerID models.PlayerID, order models.WinterOrder, fief *models.Fief, issuer *models.Noble) {
	ctx.royalOrdersIssued[order.Type]++
	orderCopy := order
	event := Event{
		Type:        eventType,
		Phase:       winterPhase,
		OwnerID:     playerID,
		OrderID:     order.ID,
		TerritoryID: order.TerritoryID,
		NobleID:     issuer.ID,
		NobleCode:   models.NobleCode(issuer.Code),
		NobleName:   ctx.state.NobleDisplayName(*issuer),
		WinterOrder: &orderCopy,
	}
	if fief != nil {
		event.FiefID = fief.ID
		event.FiefTitle = fief.Title
		event.FiefTerritories = append([]models.TerritoryID(nil), fief.Territories...)
	}
	ctx.events = append(ctx.events, event)
}

// hasSheetOrder reports whether any winter sheet holds an order matching
// match, called with the order and the player whose sheet holds it.
func (ctx *resolutionContext) hasSheetOrder(match func(models.WinterOrder, models.PlayerID) bool) bool {
	for playerID, sheet := range ctx.winterSheets {
		for _, order := range sheet {
			if match(order, playerID) {
				return true
			}
		}
	}
	return false
}

// playerHasNoble reports whether the player has a living noble.
func (ctx *resolutionContext) playerHasNoble(playerID models.PlayerID) bool {
	for _, noble := range ctx.state.Nobles {
		if noble.OwnerID == playerID {
			return true
		}
	}
	return false
}
