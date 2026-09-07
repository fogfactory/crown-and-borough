package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type hostageOrder struct{ order models.WinterOrder }

type dungeonOrder struct{ order models.WinterOrder }

func (order hostageOrder) Apply(ctx *ExecutionContext) {
	applyNobleStatusOrder(ctx, order.order, models.NobleStatusHostage)
}

func (order dungeonOrder) Apply(ctx *ExecutionContext) {
	applyNobleStatusOrder(ctx, order.order, models.NobleStatusDungeon)
}

func applyNobleStatusOrder(ctx *ExecutionContext, order models.WinterOrder, status models.NobleStatus) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	nobleID, exists := resolution.noblesByCode[order.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, order, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.Status == models.NobleStatusFree {
		resolution.rejectWinterOrder(playerID, order, "noble_not_prisoner")
		return
	}
	holder := resolution.currentArmyAt(noble.LocationID)
	if holder == nil || holder.OwnerID != playerID || noble.OwnerID == playerID {
		resolution.rejectWinterOrder(playerID, order, "noble_not_held")
		return
	}
	previousStatus := noble.Status
	noble.Status = status
	orderCopy := order
	resolution.events = append(resolution.events, Event{
		Type:           EventTypeCapture,
		Phase:          winterPhase,
		OwnerID:        playerID,
		ArmyID:         holder.ID,
		OrderID:        order.ID,
		TerritoryID:    noble.LocationID,
		NobleID:        noble.ID,
		NobleCode:      models.NobleCode(noble.Code),
		NobleName:      noble.Name,
		PreviousStatus: previousStatus,
		Status:         noble.Status,
		CaptorPlayerID: playerID,
		WinterOrder:    &orderCopy,
	})
}
