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

// capitalOwnerAt returns the player whose capital castle sits at territoryID,
// if any.
func (ctx *resolutionContext) capitalOwnerAt(territoryID models.TerritoryID) (models.PlayerID, bool) {
	for i := range ctx.state.Players {
		player := &ctx.state.Players[i]
		if player.CapitalCastleID == nil {
			continue
		}
		infrastructure := ctx.infrastructuresByID[*player.CapitalCastleID]
		if infrastructure != nil && infrastructure.TerritoryID == territoryID {
			return player.ID, true
		}
	}
	return "", false
}

// anchorOwner returns the player a territory stays controlled by even without
// any army standing on it: a fief's owner for one of its members (titres.md,
// "Control and occupation"), or a player for their own capital, a permanent
// anchor regardless of fief membership. It reports false for every other
// territory, whose control is otherwise ephemeral and only lasts as long as
// one of the controller's armies occupies it (#215).
func (ctx *resolutionContext) anchorOwner(territoryID models.TerritoryID) (models.PlayerID, bool) {
	if fief := ctx.fiefContaining(territoryID); fief != nil {
		return fief.OwnerID, true
	}
	if ownerID, ok := ctx.capitalOwnerAt(territoryID); ok {
		return ownerID, true
	}
	return "", false
}

// territoryAnchored reports whether territoryID is anchored to any player at
// all, regardless of who: used where the identity of the anchor does not
// matter, only whether the territory stands on its own without an army (for
// instance a castle's defensive bonus, or a retreat destination).
func (ctx *resolutionContext) territoryAnchored(territoryID models.TerritoryID) bool {
	_, anchored := ctx.anchorOwner(territoryID)
	return anchored
}

// releaseUnanchoredControl normalizes every territory's OwnerID at the end of
// a turn's control changes (updateTerritorialControl for an action turn,
// ResolveWinter after repatriateWinterStocks): control outside a fief is
// ephemeral (titres.md, "Control and occupation"), so a controlled territory
// keeps its OwnerID only while it is a fief member, its controller's own
// capital, or currently held by one of the controller's armies. Anything else
// reverts to neutral (OwnerID nil), free to be retaken positionally by the
// next army that stops there. Idempotent: a territory already neutral, or
// still anchored, is left untouched. Reported by a control_changed event,
// reason "abandoned", only when the released territory carries an
// infrastructure, so an empty cell losing a stale OwnerID does not clutter
// the report.
func (ctx *resolutionContext) releaseUnanchoredControl() {
	phase := 5
	if ctx.state.Season == models.SeasonWinter {
		phase = winterPhase
	}
	// Indexed once for every territory in the loop below, instead of each
	// iteration re-scanning every fief's member list via fiefContaining (the
	// per-call cost anchorOwner pays elsewhere, where it is only ever called
	// for a single, already-known territory).
	fiefOwnerAt := make(map[models.TerritoryID]models.PlayerID, len(ctx.state.Fiefs))
	for _, fief := range ctx.state.Fiefs {
		for _, member := range fief.Territories {
			fiefOwnerAt[member] = fief.OwnerID
		}
	}
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID == nil {
			continue
		}
		playerID := *state.OwnerID
		if fiefOwner, anchored := fiefOwnerAt[territoryID]; anchored && fiefOwner == playerID {
			continue
		}
		if capitalOwner, anchored := ctx.capitalOwnerAt(territoryID); anchored && capitalOwner == playerID {
			continue
		}
		if army := ctx.currentArmyAt(territoryID); army != nil && army.OwnerID == playerID {
			continue
		}
		state.OwnerID = nil
		ctx.state.TerritoryStates[territoryID] = state
		if state.Infrastructures != nil {
			ctx.events = append(ctx.events, Event{
				Type:            EventTypeControlChanged,
				Phase:           phase,
				TerritoryID:     territoryID,
				PreviousOwnerID: playerID,
				OwnerID:         "",
				Reason:          "abandoned",
			})
		}
	}
}
