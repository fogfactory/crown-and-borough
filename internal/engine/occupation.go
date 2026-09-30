package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// Territorial control is derived, never stored (titres.md, "Control and
// occupation"; models.GameState.TerritoryController): the owner of the fief a
// territory belongs to, else the player whose capital castle stands on it,
// else the owner of the army stationed on it. A neutral (revolt) army occupies
// a territory but never controls it.
//
// The engine reads it at two moments of a resolution, which are not
// interchangeable because armies move, fiefs dissolve and capitals change hands
// while a turn resolves:
//
//   - controllerAtStart is the frozen snapshot of the control the previous
//     resolution left behind (fiefs, capitals and armies as they were when the
//     context was created). Everything that runs before the control pass of
//     phase 5 (intentions, contests, movement, retreats, pillage credit) and
//     the whole winter resolution read it: a territory an army just vacated or
//     a fief a pillage just dissolved keeps its start-of-turn controller until
//     the control pass settles it.
//   - controllerNow derives control from the current fiefs, capitals and
//     armies. It is what the end-of-turn ravitaillement reads, once phase 5
//     has settled positions and control.

// controlView reads the controller of a territory at one moment of a
// resolution: controllerAtStart or controllerNow.
type controlView func(models.TerritoryID) (models.PlayerID, bool)

// controllerAtStart returns the controller of territoryID in the snapshot taken
// when the context was created.
func (ctx *resolutionContext) controllerAtStart(territoryID models.TerritoryID) (models.PlayerID, bool) {
	controller, controlled := ctx.startControl[territoryID]
	return controller, controlled
}

// controllerNow derives the controller of territoryID from the current fiefs,
// capitals and armies.
func (ctx *resolutionContext) controllerNow(territoryID models.TerritoryID) (models.PlayerID, bool) {
	if ownerID, anchored := ctx.anchorOwner(territoryID); anchored {
		return ownerID, true
	}
	if army := ctx.currentArmyAt(territoryID); army != nil && army.OwnerID != models.NeutralPlayerID {
		return army.OwnerID, true
	}
	return "", false
}

// controllerSettled returns the controller of territoryID in the snapshot
// resolveSupply takes when it starts, i.e. the control phase 5 settled: what
// the ravitaillement reports and credits read once a famine auto-pillage may
// have changed the fiefs and capitals underneath.
func (ctx *resolutionContext) controllerSettled(territoryID models.TerritoryID) (models.PlayerID, bool) {
	controller, controlled := ctx.settledControl[territoryID]
	return controller, controlled
}

// snapshotControlNow freezes controllerNow for every territory.
func (ctx *resolutionContext) snapshotControlNow() map[models.TerritoryID]models.PlayerID {
	snapshot := make(map[models.TerritoryID]models.PlayerID, len(ctx.state.Territories))
	for _, territory := range ctx.state.Territories {
		if controller, controlled := ctx.controllerNow(territory.ID); controlled {
			snapshot[territory.ID] = controller
		}
	}
	return snapshot
}

// controlledBy reports whether playerID controls territoryID in view.
func controlledBy(view controlView, playerID models.PlayerID, territoryID models.TerritoryID) bool {
	controller, controlled := view(territoryID)
	return controlled && controller == playerID
}

// occupiedAgainst reports whether territoryID is occupied against its
// controller in view: the territory has a controller and army sits on it
// whose owner differs from that controller. A NEUTRAL revolt army counts as
// an occupier like any other player's army (titres.md "Contrôle et
// occupation"). Only an anchored territory (fief member or capital) can be
// occupied: outside them the army on the territory is its controller.
func occupiedAgainst(view controlView, territoryID models.TerritoryID, army *models.Army) bool {
	if army == nil {
		return false
	}
	controller, controlled := view(territoryID)
	return controlled && army.OwnerID != controller
}

// occupiedAgainstController is occupiedAgainst on the current control. It is
// the single seam every end-of-turn "occupied cell" rule (supply source, depot
// range bonus) goes through, so they all agree on what "occupied" means.
// Uncontrolled territory is never occupied against a controller: it has none.
func (ctx *resolutionContext) occupiedAgainstController(territoryID models.TerritoryID, army *models.Army) bool {
	return occupiedAgainst(ctx.controllerNow, territoryID, army)
}

// occupiedAgainstStartController is occupiedAgainst on the start-of-resolution
// control: what the phases before the control pass and the winter rules (transfer,
// investment, payment reserve, stock repatriation) read.
func (ctx *resolutionContext) occupiedAgainstStartController(territoryID models.TerritoryID, army *models.Army) bool {
	return occupiedAgainst(ctx.controllerAtStart, territoryID, army)
}

// rejectIfOccupied rejects order with "territory_occupied_by_other_player"
// when territoryID is occupied against its controller (titres.md "Contrôle
// et occupation"), and reports whether it did so the caller can return
// immediately. Every single-territory winter order that must reject an
// occupied territory (build, elect capital, recruit troop, transfer's
// source) shares this reason and this attribution, unlike found_fief, whose
// own reason and per-member attribution keep it a direct
// occupiedAgainstStartController call.
func (ctx *resolutionContext) rejectIfOccupied(playerID models.PlayerID, order models.WinterOrder, territoryID models.TerritoryID) bool {
	if !ctx.occupiedAgainstStartController(territoryID, ctx.currentArmyAt(territoryID)) {
		return false
	}
	ctx.rejectWinterOrder(playerID, order, "territory_occupied_by_other_player")
	return true
}

// territoryOccupiedAgainstController is the resolutionContext-free variant of
// occupiedAgainstController for callers, such as preview.go, that only carry
// a GameState at rest. It reads the army currently stationed on territoryID
// directly off TerritoryState.
func territoryOccupiedAgainstController(game *models.GameState, territoryID models.TerritoryID) bool {
	state := game.TerritoryStates[territoryID]
	if state.Army == nil {
		return false
	}
	controller, controlled := game.TerritoryController(territoryID)
	if !controlled {
		return false
	}
	for _, army := range game.Armies {
		if army.ID == *state.Army {
			return army.OwnerID != controller
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

// emitAbandonedControl reports the territories that lost their controller
// since before was taken: control outside a fief is ephemeral (titres.md,
// "Control and occupation"), so a territory controlled in before that is no
// longer a fief member, its controller's own capital, or held by one of the
// controller's armies is now free, to be retaken positionally by the next army
// that stops there. Reported by a control_changed event, reason "abandoned",
// only when the territory carries an infrastructure, so an empty cell losing
// its controller does not clutter the report. Nothing is written back: control
// is derived, so the territory is already uncontrolled.
func (ctx *resolutionContext) emitAbandonedControl(before map[models.TerritoryID]models.PlayerID) {
	phase := 5
	if ctx.state.Season == models.SeasonWinter {
		phase = winterPhase
	}
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		previousOwnerID, controlled := before[territoryID]
		if !controlled {
			continue
		}
		if controller, stillControlled := ctx.controllerNow(territoryID); stillControlled && controller == previousOwnerID {
			continue
		}
		if ctx.state.TerritoryStates[territoryID].Infrastructures != nil {
			ctx.events = append(ctx.events, Event{
				Type:            EventTypeControlChanged,
				Phase:           phase,
				TerritoryID:     territoryID,
				PreviousOwnerID: previousOwnerID,
				OwnerID:         "",
				Reason:          "abandoned",
			})
		}
	}
}
