package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type electCapitalOrder struct{ order models.WinterOrder }

func (order electCapitalOrder) Apply(ctx *ExecutionContext) {
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
	infrastructure := resolution.infrastructureAt(winterOrder.TerritoryID)
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle {
		resolution.rejectWinterOrder(playerID, winterOrder, "capital_requires_controlled_castle")
		return
	}
	resolution.setCapital(playerID, infrastructure.ID)
	resolution.events = append(resolution.events, Event{
		Type:               EventTypeCapitalElected,
		Phase:              winterPhase,
		OwnerID:            playerID,
		OrderID:            winterOrder.ID,
		TerritoryID:        winterOrder.TerritoryID,
		InfrastructureID:   infrastructure.ID,
		InfrastructureType: infrastructure.Type,
		ResourceSpent:      0,
	})
}
