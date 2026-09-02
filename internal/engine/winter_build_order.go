package engine

import (
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type buildOrder struct{ order models.WinterOrder }

func (order buildOrder) Apply(ctx *ExecutionContext) {
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
	if !isBuildableInfrastructure(winterOrder.InfraType) {
		resolution.rejectWinterOrder(playerID, winterOrder, "invalid_infrastructure")
		return
	}
	existing := resolution.infrastructureAt(winterOrder.TerritoryID)
	upgradeCost := 0
	if existing != nil && existing.Type == models.InfraTypeMill && winterOrder.InfraType == models.InfraTypeMill {
		var exists bool
		upgradeCost, exists = millCostForLevel(resolution.balance.Costs, existing.Level+1)
		if !exists {
			resolution.rejectWinterOrder(playerID, winterOrder, i18n.WinterMillMaxLevelReached)
			return
		}
	}
	if winterOrder.InfraType == models.InfraTypeMill && !resolution.millCanBeBuiltAt(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "mill_requires_productive_neighbor")
		return
	}
	if existing != nil {
		if existing.Type == models.InfraTypeMill && winterOrder.InfraType == models.InfraTypeMill {
			nextLevel := existing.Level + 1
			spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, upgradeCost)
			if !paid {
				resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
				return
			}
			existing.Level = nextLevel
			resolution.events = append(resolution.events, Event{
				Type:               EventTypeUpgrade,
				Phase:              winterPhase,
				OwnerID:            playerID,
				OrderID:            winterOrder.ID,
				TerritoryID:        winterOrder.TerritoryID,
				InfrastructureID:   existing.ID,
				InfrastructureType: existing.Type,
				Level:              existing.Level,
				ResourceSpent:      spent,
			})
			return
		}
		if existing.Type != models.InfraTypeVillage || winterOrder.InfraType != models.InfraTypeCastle {
			resolution.rejectWinterOrder(playerID, winterOrder, "structure_present")
			return
		}
	}
	cost, exists := infrastructureCost(resolution.balance.Costs, winterOrder.InfraType)
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "invalid_infrastructure")
		return
	}
	spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, cost)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}
	if existing != nil {
		resolution.removeInfrastructurePreservingStock(existing.ID)
	}
	infrastructure := resolution.addWinterInfrastructure(winterOrder.InfraType, winterOrder.TerritoryID)
	capitalAssigned := false
	if winterOrder.InfraType == models.InfraTypeCastle {
		if _, _, hasCapital := resolution.capitalTerritory(playerID); !hasCapital {
			resolution.setCapital(playerID, infrastructure.ID)
			capitalAssigned = true
		}
	}
	resolution.events = append(resolution.events, Event{
		Type:               EventTypeBuild,
		Phase:              winterPhase,
		OwnerID:            playerID,
		OrderID:            winterOrder.ID,
		TerritoryID:        winterOrder.TerritoryID,
		InfrastructureID:   infrastructure.ID,
		InfrastructureType: infrastructure.Type,
		Level:              infrastructure.Level,
		ResourceSpent:      spent,
	})
	if capitalAssigned {
		resolution.events = append(resolution.events, Event{
			Type:               EventTypeCapitalElected,
			Phase:              winterPhase,
			OwnerID:            playerID,
			OrderID:            winterOrder.ID,
			TerritoryID:        winterOrder.TerritoryID,
			InfrastructureID:   infrastructure.ID,
			InfrastructureType: infrastructure.Type,
			ResourceSpent:      0,
			Automatic:          true,
		})
	}
}
