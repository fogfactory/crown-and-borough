package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// occupiedAgainstController reports whether territoryID is occupied against
// its controller: the territory has a controller (OwnerID) and army sits on
// it whose owner differs from that controller. A NEUTRAL revolt army counts
// as an occupier like any other player's army (titres.md "Contrôle et
// occupation"). It is the single seam every "occupied cell" rule (supply
// source, depot range bonus, transfer, winter investment, payment reserve,
// stock repatriation) goes through, so they all agree on what "occupied"
// means. Uncontrolled territory (OwnerID == nil) is never occupied against a
// controller: it has none.
func (ctx *resolutionContext) occupiedAgainstController(territoryID models.TerritoryID, army *models.Army) bool {
	if army == nil {
		return false
	}
	state := ctx.state.TerritoryStates[territoryID]
	return state.OwnerID != nil && army.OwnerID != *state.OwnerID
}

// rejectIfOccupied rejects order with "territory_occupied_by_other_player"
// when territoryID is occupied against its controller (titres.md "Contrôle
// et occupation"), and reports whether it did so the caller can return
// immediately. Every single-territory winter order that must reject an
// occupied territory (build, elect capital, recruit troop, transfer's
// source) shares this reason and this attribution, unlike found_fief, whose
// own reason and per-member attribution keep it a direct
// occupiedAgainstController call.
func (ctx *resolutionContext) rejectIfOccupied(playerID models.PlayerID, order models.WinterOrder, territoryID models.TerritoryID) bool {
	if !ctx.occupiedAgainstController(territoryID, ctx.currentArmyAt(territoryID)) {
		return false
	}
	ctx.rejectWinterOrder(playerID, order, "territory_occupied_by_other_player")
	return true
}

// territoryOccupiedAgainstController is the resolutionContext-free variant of
// occupiedAgainstController for callers, such as preview.go, that only carry
// a GameState. It reads the army currently stationed on territoryID directly
// off TerritoryState.
func territoryOccupiedAgainstController(game *models.GameState, territoryID models.TerritoryID) bool {
	state := game.TerritoryStates[territoryID]
	if state.OwnerID == nil || state.Army == nil {
		return false
	}
	for _, army := range game.Armies {
		if army.ID == *state.Army {
			return army.OwnerID != *state.OwnerID
		}
	}
	return false
}
