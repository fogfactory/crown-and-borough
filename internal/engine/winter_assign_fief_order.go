package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type assignFiefOrder struct{ order models.WinterOrder }

// Apply attributes a vacant fief to a free noble of its owner (T A, titres.md,
// #194). It is free (ResourceSpent stays 0): only constitution costs R.
func (order assignFiefOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order

	fief := resolution.fiefByCapital(winterOrder.TerritoryID)
	if fief == nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_found")
		return
	}
	if fief.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_owned")
		return
	}
	if fief.HolderNobleID != nil {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_vacant")
		return
	}
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.OwnerID != playerID {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_holder_not_owned")
		return
	}
	if noble.Status != models.NobleStatusFree {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_holder_not_free")
		return
	}

	holderNobleID := nobleID
	fief.HolderNobleID = &holderNobleID
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeFiefAssigned,
		Phase:           winterPhase,
		OwnerID:         playerID,
		OrderID:         winterOrder.ID,
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		NobleID:         nobleID,
		NobleCode:       models.NobleCode(noble.Code),
		NobleName:       noble.Name,
		ResourceSpent:   0,
	})
}
