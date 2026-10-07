package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// transferNobleOrder is H N NNN XXX (specs/succession.md § Transfert de
// noble): the player hands a noble under their control to the army standing
// on XXX. The noble becomes a hostage of that army's owner (or keeps its
// hostage or dungeon status, or takes the optional O/P one); when the recipient
// is the noble's own owner, the noble is released free of charge.
type transferNobleOrder struct{ order models.WinterOrder }

func (order transferNobleOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || !resolution.controlsNoble(playerID, noble) {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_controlled")
		return
	}
	recipient := resolution.currentArmyAt(winterOrder.TerritoryID)
	if recipient == nil || recipient.OwnerID == models.NeutralPlayerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "no_army_at_destination")
		return
	}
	if recipient.OwnerID == playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "transfer_to_self")
		return
	}
	released := recipient.OwnerID == noble.OwnerID
	if released && winterOrder.Status != "" {
		resolution.rejectWinterOrder(playerID, winterOrder, "transfer_status_to_owner")
		return
	}
	previousStatus := noble.Status
	sourceID := noble.LocationID
	noble.LocationID = winterOrder.TerritoryID
	switch {
	case released:
		noble.Status = models.NobleStatusFree
	case winterOrder.Status != "":
		noble.Status = winterOrder.Status
	case previousStatus == models.NobleStatusFree:
		noble.Status = models.NobleStatusHostage
	}
	orderCopy := winterOrder
	resolution.events = append(resolution.events, Event{
		Type:           EventTypeNobleTransfer,
		Phase:          winterPhase,
		OwnerID:        playerID,
		OrderID:        winterOrder.ID,
		ArmyID:         recipient.ID,
		NobleID:        noble.ID,
		NobleCode:      models.NobleCode(noble.Code),
		NobleName:      resolution.state.NobleDisplayName(*noble),
		PreviousStatus: previousStatus,
		Status:         noble.Status,
		SourceID:       sourceID,
		TerritoryID:    winterOrder.TerritoryID,
		CaptorPlayerID: recipient.OwnerID,
		WinterOrder:    &orderCopy,
	})
}

// controlsNoble reports whether the player can hand the noble over: their own
// free noble, or a noble (hostage or prisoner) held by one of their armies.
func (ctx *resolutionContext) controlsNoble(playerID models.PlayerID, noble *models.Noble) bool {
	if noble.Status == models.NobleStatusFree {
		return noble.OwnerID == playerID
	}
	holder := ctx.currentArmyAt(noble.LocationID)
	return holder != nil && holder.OwnerID == playerID && noble.OwnerID != playerID
}
