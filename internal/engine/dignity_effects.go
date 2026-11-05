package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// dignityBeneficiary reports whether the player profits from the dignities of
// the noble: its owner while it is not a prisoner, and whoever holds it
// hostage (specs/dames.md § Dignités: a hostage keeps its bonuses and passes
// them on to its holder). holder is the owner of the army standing with it.
func dignityBeneficiary(noble models.Noble, holder models.PlayerID, player models.PlayerID) bool {
	if !noble.DignityActive() {
		return false
	}
	if noble.OwnerID == player {
		return true
	}
	return noble.Status == models.NobleStatusHostage && holder == player
}

// DignityHolder returns the owner of the army that holds the hostage noble,
// or "" when none does.
func dignityHolder(armyAt func(models.TerritoryID) *models.Army, noble models.Noble) models.PlayerID {
	if noble.Status != models.NobleStatusHostage {
		return ""
	}
	if army := armyAt(noble.LocationID); army != nil && army.OwnerID != noble.OwnerID {
		return army.OwnerID
	}
	return ""
}

func (ctx *resolutionContext) beneficiaryOf(noble models.Noble, player models.PlayerID) bool {
	return dignityBeneficiary(noble, dignityHolder(ctx.currentArmyAt, noble), player)
}

// armyDemand is the ration cost of the army: its size cost, lowered by the
// dignities spare it (Herboriste) and raised by those that burden a rival
// army in their region (Sorcière), never below zero.
func armyDemand(ctx *resolutionContext, army models.Army) int {
	demand := armyCost(army.Size, ctx.balance.CostBase)
	if army.OwnerID == models.NeutralPlayerID {
		return demand
	}
	region := regionForTerritory(ctx, army.TerritoryID)
	for _, noble := range ctx.state.Nobles {
		if len(noble.Dignities) == 0 {
			continue
		}
		if noble.LocationID == army.TerritoryID && ctx.beneficiaryOf(noble, army.OwnerID) {
			demand -= noble.RationDiscount()
		}
		if noble.OwnerID != army.OwnerID && regionForTerritory(ctx, noble.LocationID) == region {
			demand += noble.RivalConsumption()
		}
	}
	return max(0, demand)
}

// plagueProtected reports whether a plague-proof noble (Herboriste) that
// serves the player stands on the territory or on an adjacent one.
func plagueProtected(ctx *resolutionContext, player models.PlayerID, territoryID models.TerritoryID) bool {
	for _, noble := range ctx.state.Nobles {
		if !noble.ProtectsFromPlague() || !ctx.beneficiaryOf(noble, player) {
			continue
		}
		if noble.LocationID == territoryID || ctx.isAdjacent(noble.LocationID, territoryID) {
			return true
		}
	}
	return false
}
