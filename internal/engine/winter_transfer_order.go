package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type transferOrder struct{ order models.WinterOrder }

func (order transferOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	if !resolution.territoryExists(winterOrder.SourceID) || !resolution.territoryExists(winterOrder.TargetID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_territory")
		return
	}
	if winterOrder.SourceID == winterOrder.TargetID {
		resolution.rejectWinterOrder(playerID, winterOrder, "transfer_same_territory")
		return
	}
	if winterOrder.Amount < 1 {
		resolution.rejectWinterOrder(playerID, winterOrder, "invalid_transfer_amount")
		return
	}
	if !resolution.controlsTerritory(playerID, winterOrder.SourceID) || !resolution.hasSettlement(winterOrder.SourceID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "transfer_source_not_settlement")
		return
	}
	if resolution.rejectIfOccupied(playerID, winterOrder, winterOrder.SourceID) {
		return
	}
	targetState := resolution.state.TerritoryStates[winterOrder.TargetID]
	targetControllerID, targetControlled := resolution.controllerAtStart(winterOrder.TargetID)
	if !targetControlled || targetControllerID == playerID || !PlayerAlive(resolution.state, targetControllerID) || !resolution.hasSettlement(winterOrder.TargetID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "transfer_target_not_settlement")
		return
	}
	spent, paid := resolution.payWinterCost(playerID, winterOrder.SourceID, winterOrder.Amount)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	targetState.Resources += winterOrder.Amount
	resolution.state.TerritoryStates[winterOrder.TargetID] = targetState
	orderCopy := winterOrder
	resolution.events = append(resolution.events, Event{
		Type:           EventTypeTransfer,
		Phase:          winterPhase,
		OwnerID:        playerID,
		OrderID:        winterOrder.ID,
		SourceID:       winterOrder.SourceID,
		TargetID:       winterOrder.TargetID,
		ResourceAmount: winterOrder.Amount,
		ResourceSpent:  spent,
		Outcome:        OutcomeSuccess,
		WinterOrder:    &orderCopy,
	})
}
