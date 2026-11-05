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
	if resolution.rejectIfOccupied(playerID, winterOrder, winterOrder.TerritoryID) {
		return
	}
	if !isBuildableInfrastructure(winterOrder.InfraType) {
		resolution.rejectWinterOrder(playerID, winterOrder, "invalid_infrastructure")
		return
	}
	existing := resolution.infrastructureAt(winterOrder.TerritoryID)
	// A ruin counts as an empty tile: any build replaces it, except C C on a
	// ruined castle, which rebuilds it in place. A ruined fief capital can only
	// be rebuilt: replacing it would dissolve the fief.
	rebuildsCastle := existing != nil && existing.Type == models.InfraTypeCastle && existing.Ruined && winterOrder.InfraType == models.InfraTypeCastle
	var ruin *models.Infrastructure
	if existing != nil && existing.IsRuined() && !rebuildsCastle {
		if resolution.fiefByCapital(winterOrder.TerritoryID) != nil {
			resolution.rejectWinterOrder(playerID, winterOrder, "ruined_fief_capital_rebuild_only")
			return
		}
		ruin, existing = existing, nil
	}
	upgradeCost := 0
	if existing != nil && existing.Type == models.InfraTypeMill && winterOrder.InfraType == models.InfraTypeMill {
		var exists bool
		upgradeCost, exists = millCostForLevel(resolution.balance.Costs, existing.Level+1)
		if !exists {
			resolution.rejectWinterOrder(playerID, winterOrder, i18n.WinterMillMaxLevelReached)
			return
		}
	}
	// The productive-neighbor requirement only gates building a new mill
	// (existing == nil): upgrading one already standing is always allowed,
	// even in isolation, since it can pay for itself (see #195).
	if existing == nil && winterOrder.InfraType == models.InfraTypeMill && !resolution.millCanBeBuiltAt(winterOrder.TerritoryID) {
		resolution.rejectWinterOrder(playerID, winterOrder, "mill_requires_productive_neighbor")
		return
	}
	if existing != nil {
		if existing.Type == models.InfraTypeMill && winterOrder.InfraType == models.InfraTypeMill {
			nextLevel := existing.Level + 1
			sources := resolution.millUpgradePaymentSources(playerID, winterOrder.TerritoryID)
			spent, paid := resolution.payFromSources(sources, upgradeCost)
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
		if existing.Type == models.InfraTypeVillage && winterOrder.InfraType == models.InfraTypeCastle {
			// C C on a village fortifies it instead of replacing it with a
			// castle: it keeps its InfraID and every village behavior, and
			// additionally gains a castle's defensive bonus (#193).
			if existing.Fortified {
				resolution.rejectWinterOrder(playerID, winterOrder, "village_already_fortified")
				return
			}
			fortifyCost, exists := infrastructureCost(resolution.balance.Costs, models.InfraTypeCastle)
			if !exists {
				resolution.rejectWinterOrder(playerID, winterOrder, "invalid_infrastructure")
				return
			}
			spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, fortifyCost)
			if !paid {
				resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
				return
			}
			existing.Fortified = true
			resolution.events = append(resolution.events, Event{
				Type:               EventTypeFortify,
				Phase:              winterPhase,
				OwnerID:            playerID,
				OrderID:            winterOrder.ID,
				TerritoryID:        winterOrder.TerritoryID,
				InfrastructureID:   existing.ID,
				InfrastructureType: models.InfraTypeVillage,
				ResourceSpent:      spent,
			})
			return
		}
		if rebuildsCastle {
			// C C on a ruined castle rebuilds it in place at the normal
			// castle cost.
			rebuildCost, exists := infrastructureCost(resolution.balance.Costs, models.InfraTypeCastle)
			if !exists {
				resolution.rejectWinterOrder(playerID, winterOrder, "invalid_infrastructure")
				return
			}
			spent, paid := resolution.payWinterCost(playerID, winterOrder.TerritoryID, rebuildCost)
			if !paid {
				resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
				return
			}
			existing.Ruined = false
			resolution.events = append(resolution.events, Event{
				Type:               EventTypeBuild,
				Phase:              winterPhase,
				OwnerID:            playerID,
				OrderID:            winterOrder.ID,
				TerritoryID:        winterOrder.TerritoryID,
				InfrastructureID:   existing.ID,
				InfrastructureType: existing.Type,
				Level:              existing.Level,
				ResourceSpent:      spent,
			})
			resolution.assignCapitalIfMissing(playerID, winterOrder, existing)
			return
		}
		resolution.rejectWinterOrder(playerID, winterOrder, "structure_present")
		return
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
	if ruin != nil {
		resolution.removeInfrastructurePreservingStock(ruin.ID)
	}
	infrastructure := resolution.addWinterInfrastructure(winterOrder.InfraType, winterOrder.TerritoryID)
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
	resolution.assignCapitalIfMissing(playerID, winterOrder, infrastructure)
}

// assignCapitalIfMissing makes infrastructure, just built or rebuilt, the
// capital of a player who has none when it is a castle.
func (ctx *resolutionContext) assignCapitalIfMissing(playerID models.PlayerID, winterOrder models.WinterOrder, infrastructure *models.Infrastructure) {
	if infrastructure.Type != models.InfraTypeCastle {
		return
	}
	if _, _, hasCapital := ctx.capitalTerritory(playerID); hasCapital {
		return
	}
	ctx.electCapital(playerID, winterOrder, infrastructure, true)
}

// electCapital makes castle the capital of playerID and reports it; automatic
// marks a capital the engine assigned rather than one an elect-capital order
// chose.
func (ctx *resolutionContext) electCapital(playerID models.PlayerID, winterOrder models.WinterOrder, castle *models.Infrastructure, automatic bool) {
	ctx.setCapital(playerID, castle.ID)
	ctx.events = append(ctx.events, Event{
		Type:               EventTypeCapitalElected,
		Phase:              winterPhase,
		OwnerID:            playerID,
		OrderID:            winterOrder.ID,
		TerritoryID:        winterOrder.TerritoryID,
		InfrastructureID:   castle.ID,
		InfrastructureType: castle.Type,
		ResourceSpent:      0,
		Automatic:          automatic,
	})
}
