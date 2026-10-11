package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type changeRankOrder struct{ order models.WinterOrder }

// Apply raises a fief's title by one rank (T R NNN XXX, politique.md §
// Seigneurie et annexion), at most once per winter and per fief. A royal
// accord makes it free; otherwise it costs rank_per_territory for each
// territory of the fief. Every control runs before the payment.
func (order changeRankOrder) Apply(ctx *ExecutionContext) {
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
	nobleID, exists := resolution.noblesByCode[winterOrder.NobleCode]
	if !exists {
		resolution.rejectWinterOrder(playerID, winterOrder, "unknown_noble")
		return
	}
	noble := resolution.noblesByID[nobleID]
	if noble == nil || noble.OwnerID != playerID || fief.HolderNobleID == nil || *fief.HolderNobleID != nobleID {
		resolution.rejectWinterOrder(playerID, winterOrder, "rank_not_titular")
		return
	}
	if resolution.rankChangedFiefs[fief.ID] {
		resolution.rejectWinterOrder(playerID, winterOrder, "rank_already_changed")
		return
	}
	title, ok := fief.Title.Next()
	if !ok {
		resolution.rejectWinterOrder(playerID, winterOrder, "rank_max")
		return
	}
	if !resolution.state.CanReceiveTitle(nobleID, title) {
		resolution.rejectWinterOrder(playerID, winterOrder, "succession_rank_blocked")
		return
	}

	spent := 0
	if !resolution.hasRankAccord(fief.CapitalTerritoryID) {
		cost := resolution.balance.Costs.RankPerTerritory * len(fief.Territories)
		var paid bool
		spent, paid = resolution.payWinterCost(playerID, fief.CapitalTerritoryID, cost)
		if !paid {
			resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
			return
		}
	}

	fief.Title = title
	resolution.rankChangedFiefs[fief.ID] = true
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeFiefRankChanged,
		Phase:           winterPhase,
		OwnerID:         playerID,
		OrderID:         winterOrder.ID,
		TerritoryID:     fief.CapitalTerritoryID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: append([]models.TerritoryID(nil), fief.Territories...),
		NobleID:         nobleID,
		NobleCode:       models.NobleCode(noble.Code),
		NobleName:       resolution.state.NobleDisplayName(*noble),
		ResourceSpent:   spent,
	})
}
