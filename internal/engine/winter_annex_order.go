package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type annexTerritoryOrder struct{ order models.WinterOrder }

// Apply annexes a territory to a fief (T X NNN XXX YYY, politique.md §
// Seigneurie et annexion). Every control runs before the payment, so a
// rejected order never spends resources. The fief's title is left untouched.
func (order annexTerritoryOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order
	annexedID := winterOrder.TargetID

	fief := resolution.fiefByCapital(winterOrder.TerritoryID)
	if fief == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_found")
		return
	}
	if fief.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_owned")
		return
	}
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.OwnerID != playerID || fief.HolderNobleID == nil || *fief.HolderNobleID != nobleID {
		resolution.rejectWinterOrder(playerID, winterOrder, "annex_not_titular")
		return
	}
	if !resolution.territoryExists(annexedID) {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "unknown_territory", annexedID)
		return
	}
	if resolution.fiefContaining(annexedID) == fief {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_already_member", annexedID)
		return
	}
	if !fiefAdjacentTo(resolution, fief, annexedID) {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_not_adjacent", annexedID)
		return
	}
	// The player must hold the territory with an army and share it with no
	// adverse or neutral one.
	if army := resolution.currentArmyAt(annexedID); army == nil || army.OwnerID != playerID {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_not_occupied", annexedID)
		return
	}
	if resolution.fiefByCapital(annexedID) != nil {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_capital", annexedID)
		return
	}
	donor := resolution.fiefContaining(annexedID)
	if donor != nil {
		if !resolution.hasAnnexAccord(fief.CapitalTerritoryID, annexedID) {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_royal_accord_required", annexedID)
			return
		}
		if len(donor.Territories)-1 < models.FiefMinTerritories {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "annex_donor_too_small", annexedID)
			return
		}
	}

	spent, paid := resolution.payWinterCost(playerID, annexedID, resolution.balance.Costs.AnnexPerTerritory)
	if !paid {
		resolution.rejectWinterOrderAt(playerID, winterOrder, "insufficient_resources", annexedID)
		return
	}

	var donorOwnerID models.PlayerID
	if donor != nil {
		donorOwnerID = donor.OwnerID
		donor.Territories = withoutTerritory(donor.Territories, annexedID)
	}
	fief.Territories = append(fief.Territories, annexedID)
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeFiefAnnexed,
		Phase:           winterPhase,
		OwnerID:         playerID,
		PreviousOwnerID: donorOwnerID,
		OrderID:         winterOrder.ID,
		TerritoryID:     annexedID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		NobleID:         nobleID,
		NobleCode:       models.NobleCode(noble.Code),
		NobleName:       resolution.state.NobleDisplayName(*noble),
		ResourceSpent:   spent,
	})
}

// fiefAdjacentTo reports whether territoryID borders a territory of fief
// through a crossable border.
func fiefAdjacentTo(ctx *resolutionContext, fief *models.Fief, territoryID models.TerritoryID) bool {
	for _, member := range fief.Territories {
		if ctx.isAdjacent(member, territoryID) {
			return true
		}
	}
	return false
}

func withoutTerritory(territories []models.TerritoryID, removed models.TerritoryID) []models.TerritoryID {
	kept := make([]models.TerritoryID, 0, len(territories))
	for _, territoryID := range territories {
		if territoryID != removed {
			kept = append(kept, territoryID)
		}
	}
	return kept
}
