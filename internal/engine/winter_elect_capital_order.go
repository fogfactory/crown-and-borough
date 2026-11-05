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
	if resolution.rejectIfOccupied(playerID, winterOrder, winterOrder.TerritoryID) {
		return
	}
	infrastructure := resolution.infrastructureAt(winterOrder.TerritoryID)
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle {
		resolution.rejectWinterOrder(playerID, winterOrder, "capital_requires_controlled_castle")
		return
	}
	if infrastructure.Ruined {
		resolution.rejectWinterOrder(playerID, winterOrder, "capital_castle_ruined")
		return
	}
	resolution.electCapital(playerID, winterOrder, infrastructure, false)
}
