package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type recruitTroopOrder struct{ order models.WinterOrder }

func (order recruitTroopOrder) Apply(ctx *ExecutionContext) {
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
	army := resolution.currentArmyAt(winterOrder.TerritoryID)
	if army != nil && army.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "territory_occupied_by_other_player")
		return
	}
	if !resolution.hasEligibleTroopNoble(playerID, winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "troop_requires_adjacent_noble")
		return
	}
	spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, resolution.balance.Costs.Troop)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	if army != nil {
		army.Size++
		resolution.events = append(resolution.events, Event{
			Type:          EventTypeRecruit,
			Phase:         winterPhase,
			OwnerID:       playerID,
			OrderID:       winterOrder.ID,
			TerritoryID:   winterOrder.TerritoryID,
			ArmyID:        army.ID,
			Troops:        1,
			ResourceSpent: spent,
		})
		return
	}
	newArmy := models.Army{
		ID:          resolution.allocateArmyID(),
		OwnerID:     playerID,
		TerritoryID: winterOrder.TerritoryID,
		Size:        1,
	}
	resolution.state.Armies = append(resolution.state.Armies, newArmy)
	armyID := newArmy.ID
	state := resolution.state.TerritoryStates[winterOrder.TerritoryID]
	state.Army = &armyID
	resolution.state.TerritoryStates[winterOrder.TerritoryID] = state
	resolution.rebuildIndexes()
	resolution.events = append(resolution.events, Event{
		Type:          EventTypeRecruit,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       winterOrder.ID,
		TerritoryID:   winterOrder.TerritoryID,
		ArmyID:        newArmy.ID,
		Troops:        1,
		ResourceSpent: spent,
	})
}

func (ctx *resolutionContext) hasEligibleTroopNoble(playerID models.PlayerID, targetID models.TerritoryID) bool {
	for _, noble := range ctx.state.Nobles {
		if noble.OwnerID != playerID || noble.Status != models.NobleStatusFree {
			continue
		}
		if noble.LocationID == targetID || ctx.isAdjacent(noble.LocationID, targetID) {
			return true
		}
	}
	return false
}
