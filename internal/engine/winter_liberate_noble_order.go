package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type liberateNobleOrder struct{ order models.WinterOrder }

func (order liberateNobleOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.Status == models.NobleStatusFree {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_prisoner")
		return
	}
	holder := resolution.currentArmyAt(noble.LocationID)
	if holder == nil || holder.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_not_held")
		return
	}
	capitalTerritoryID, _, hasCapital := resolution.capitalTerritory(noble.OwnerID)
	if !hasCapital {
		resolution.rejectWinterOrder(playerID, winterOrder, "no_capital")
		return
	}
	capitalArmy := resolution.currentArmyAt(capitalTerritoryID)
	if capitalArmy == nil || capitalArmy.OwnerID != noble.OwnerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "no_army_at_capital")
		return
	}
	paymentTargetID := noble.LocationID
	if holderCapitalTerritoryID, _, holderHasCapital := resolution.capitalTerritory(playerID); holderHasCapital {
		paymentTargetID = holderCapitalTerritoryID
	}
	spent, paid := resolution.payWinterCost(playerID, paymentTargetID, resolution.balance.Costs.Liberation)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	previousStatus := noble.Status
	noble.Status = models.NobleStatusFree
	noble.LocationID = capitalTerritoryID
	resolution.events = append(resolution.events, Event{
		Type:           EventTypeLiberation,
		Phase:          winterPhase,
		OwnerID:        playerID,
		OrderID:        winterOrder.ID,
		NobleID:        noble.ID,
		NobleCode:      models.NobleCode(noble.Code),
		NobleName:      noble.Name,
		PreviousStatus: previousStatus,
		Status:         noble.Status,
		TerritoryID:    capitalTerritoryID,
		ResourceSpent:  spent,
	})
}
