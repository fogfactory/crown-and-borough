package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

func calculateSupports(ctx *resolutionContext) {
	for _, armyID := range sortedArmyMap(ctx.records) {
		record := ctx.records[armyID]
		if record.order.Type != models.OrderTypeSupport || record.outcome != "" {
			continue
		}
		order := record.order
		targetID := order.TargetIDs[0]
		support := &supportIntent{
			armyID:   armyID,
			source:   order.PositionID,
			targetID: targetID,
		}
		if supported := ctx.startArmyAt(targetID); supported != nil {
			support.targetArmyID = supported.ID
		}
		if len(order.TargetIDs) == 2 {
			support.offensive = true
			support.destinationID = order.TargetIDs[1]
			attack := ctx.attacks[support.targetArmyID]
			support.applies = attack != nil && attack.target == support.destinationID
		} else {
			targetRecord := ctx.records[support.targetArmyID]
			support.applies = targetRecord != nil && targetRecord.outcome == "" && holdsForDefense(targetRecord.order.Type)
		}
		ctx.supports[armyID] = support
	}
}

func holdsForDefense(orderType models.OrderType) bool {
	return orderType == models.OrderTypeHold || orderType == models.OrderTypeSupport || orderType == models.OrderTypePillage
}
