package engine

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type recruitNobleOrder struct{ order models.WinterOrder }

func (order recruitNobleOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	if !resolution.territoryExists(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_territory")
		return
	}
	if !resolution.controlsTerritory(playerID, winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "territory_not_controlled")
		return
	}
	if !resolution.hasSettlement(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_requires_settlement")
		return
	}
	army := resolution.currentArmyAt(winterOrder.TerritoryID)
	if army == nil || army.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "noble_requires_owned_army")
		return
	}
	if !resolution.hasAvailableFirstName(resolution.balance.FirstNames) {
		resolution.rejectWinterOrder(playerID, winterOrder, "no_available_first_name")
		return
	}
	spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, resolution.balance.Costs.Noble)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	firstName := resolution.drawFirstName(ctx.firstNameRNG, resolution.balance.FirstNames)
	territory := resolution.territoriesByID[winterOrder.TerritoryID]
	noble := models.Noble{
		ID:               nextNobleID(resolution.state.Nobles),
		Code:             firstName.Code,
		Name:             fmt.Sprintf("%s de %s", firstName.Name, territory.Name),
		OwnerID:          playerID,
		LocationID:       winterOrder.TerritoryID,
		Status:           models.NobleStatusFree,
		LastEmissionTurn: 0,
	}
	resolution.state.Nobles = append(resolution.state.Nobles, noble)
	resolution.rebuildIndexes()
	resolution.events = append(resolution.events, Event{
		Type:          EventTypeRecruit,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       winterOrder.ID,
		TerritoryID:   winterOrder.TerritoryID,
		NobleID:       noble.ID,
		NobleCode:     models.NobleCode(noble.Code),
		NobleName:     noble.Name,
		ResourceSpent: spent,
	})
}
