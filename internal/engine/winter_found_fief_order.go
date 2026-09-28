package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

type foundFiefOrder struct{ order models.WinterOrder }

// Apply constitutes a fief (T F, titres.md, #194). Every control below runs
// before any payment, so a rejected order never spends resources.
func (order foundFiefOrder) Apply(ctx *ExecutionContext) {
	resolution := ctx.resolution
	playerID := ctx.playerID
	winterOrder := order.order

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

	territories := winterOrder.TerritoryIDs
	seen := make(map[models.TerritoryID]bool, len(territories))
	for _, territoryID := range territories {
		if seen[territoryID] {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "fief_duplicate_territory", territoryID)
			return
		}
		seen[territoryID] = true
	}
	if len(territories) < models.FiefMinTerritories {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_too_small")
		return
	}
	title, ok := models.FiefTitleForSize(len(territories))
	if !ok {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_too_small")
		return
	}

	for _, territoryID := range territories {
		if !resolution.territoryExists(territoryID) {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "unknown_territory", territoryID)
			return
		}
		if !resolution.controlsTerritory(playerID, territoryID) {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "territory_not_controlled", territoryID)
			return
		}
		if resolution.fiefContaining(territoryID) != nil {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "fief_territory_already_in_fief", territoryID)
			return
		}
		// Other castles than the capital's are tolerated in the group; only an
		// enemy or neutral (revolt) army stationed there blocks constitution.
		if army := resolution.currentArmyAt(territoryID); army != nil && army.OwnerID != playerID {
			resolution.rejectWinterOrderAt(playerID, winterOrder, "fief_territory_occupied_by_other_player", territoryID)
			return
		}
	}

	capitalID := territories[0]
	capitalInfrastructure := resolution.infrastructureAt(capitalID)
	if capitalInfrastructure == nil || capitalInfrastructure.Type != models.InfraTypeCastle {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_capital_requires_castle")
		return
	}
	if !fiefGroupContiguous(resolution, territories) {
		resolution.rejectWinterOrder(playerID, winterOrder, "fief_not_contiguous")
		return
	}

	cost := resolution.balance.Costs.FiefPerTerritory * len(territories)
	spent, paid := resolution.payWinterCost(playerID, capitalID, cost)
	if !paid {
		resolution.rejectWinterOrder(playerID, winterOrder, "insufficient_resources")
		return
	}

	holderNobleID := nobleID
	fief := models.Fief{
		ID:                 nextFiefID(resolution.state.Fiefs),
		Title:              title,
		CapitalTerritoryID: capitalID,
		Territories:        append([]models.TerritoryID(nil), territories...),
		OwnerID:            playerID,
		HolderNobleID:      &holderNobleID,
	}
	resolution.state.Fiefs = append(resolution.state.Fiefs, fief)
	resolution.events = append(resolution.events, Event{
		Type:            EventTypeFiefFounded,
		Phase:           winterPhase,
		OwnerID:         playerID,
		OrderID:         winterOrder.ID,
		TerritoryID:     capitalID,
		FiefID:          fief.ID,
		FiefTitle:       fief.Title,
		FiefTerritories: fief.Territories,
		NobleID:         nobleID,
		NobleCode:       models.NobleCode(noble.Code),
		NobleName:       noble.Name,
		ResourceSpent:   spent,
	})
}

// fiefGroupContiguous reports whether every territory of the group is
// reachable from the others crossing only crossable borders (Territory.
// Adjacencies already excludes impassable geometric edges) without leaving
// the group.
func fiefGroupContiguous(ctx *resolutionContext, territories []models.TerritoryID) bool {
	if len(territories) == 0 {
		return true
	}
	inGroup := make(map[models.TerritoryID]bool, len(territories))
	for _, territoryID := range territories {
		inGroup[territoryID] = true
	}
	visited := map[models.TerritoryID]bool{territories[0]: true}
	queue := []models.TerritoryID{territories[0]}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, neighborID := range ctx.sortedNeighbors(current) {
			if !inGroup[neighborID] || visited[neighborID] {
				continue
			}
			visited[neighborID] = true
			queue = append(queue, neighborID)
		}
	}
	return len(visited) == len(inGroup)
}
