package models

// Territorial control is derived, never stored (titres.md, "Control and
// occupation"). A territory is controlled by, in priority order:
//
//  1. the owner of the fief it belongs to, capital or not;
//  2. the player whose capital castle stands on it;
//  3. the owner of the army stationed on it.
//
// Anything else is uncontrolled. A neutral (revolt) army occupies a territory
// but never controls it: it administers nothing.
//
// The GameState methods below read a state at rest, where TerritoryState.Army
// is consistent with the army list. The engine resolves on its own indexes
// mid-turn instead (engine.resolutionContext), since that pointer is only
// rebuilt at the end of a resolution.

// FiefOwnerAt returns the owner of the fief listing territoryID among its
// territories.
func (g *GameState) FiefOwnerAt(territoryID TerritoryID) (PlayerID, bool) {
	for i := range g.Fiefs {
		for _, member := range g.Fiefs[i].Territories {
			if member == territoryID {
				return g.Fiefs[i].OwnerID, true
			}
		}
	}
	return "", false
}

// CapitalOwnerAt returns the player whose capital castle stands on
// territoryID.
func (g *GameState) CapitalOwnerAt(territoryID TerritoryID) (PlayerID, bool) {
	for i := range g.Players {
		player := &g.Players[i]
		if player.CapitalCastleID == nil {
			continue
		}
		for j := range g.Infrastructures {
			if g.Infrastructures[j].ID == *player.CapitalCastleID {
				if g.Infrastructures[j].TerritoryID == territoryID {
					return player.ID, true
				}
				break
			}
		}
	}
	return "", false
}

// TerritoryController returns the player controlling territoryID, and false
// when nobody does.
func (g *GameState) TerritoryController(territoryID TerritoryID) (PlayerID, bool) {
	if owner, ok := g.FiefOwnerAt(territoryID); ok {
		return owner, true
	}
	if owner, ok := g.CapitalOwnerAt(territoryID); ok {
		return owner, true
	}
	state, exists := g.TerritoryStates[territoryID]
	if !exists || state.Army == nil {
		return "", false
	}
	for i := range g.Armies {
		if g.Armies[i].ID == *state.Army {
			if g.Armies[i].OwnerID == NeutralPlayerID {
				return "", false
			}
			return g.Armies[i].OwnerID, true
		}
	}
	return "", false
}

// TerritoryControllers returns the controller of every controlled territory
// in one pass, for callers that read many territories of the same state. It
// agrees with TerritoryController on every territory.
func (g *GameState) TerritoryControllers() map[TerritoryID]PlayerID {
	controllers := make(map[TerritoryID]PlayerID, len(g.Territories))
	for _, army := range g.Armies {
		if army.OwnerID != NeutralPlayerID {
			controllers[army.TerritoryID] = army.OwnerID
		}
	}
	infrastructureTerritory := make(map[InfraID]TerritoryID, len(g.Infrastructures))
	for _, infrastructure := range g.Infrastructures {
		infrastructureTerritory[infrastructure.ID] = infrastructure.TerritoryID
	}
	for _, player := range g.Players {
		if player.CapitalCastleID == nil {
			continue
		}
		if territoryID, exists := infrastructureTerritory[*player.CapitalCastleID]; exists {
			controllers[territoryID] = player.ID
		}
	}
	for _, fief := range g.Fiefs {
		for _, member := range fief.Territories {
			controllers[member] = fief.OwnerID
		}
	}
	return controllers
}
