package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

func cloneGameState(source *models.GameState) *models.GameState {
	clone := *source
	clone.Players = cloneSlice(source.Players)
	for i, player := range source.Players {
		if player.CapitalCastleID != nil {
			capitalCastleID := *player.CapitalCastleID
			clone.Players[i].CapitalCastleID = &capitalCastleID
		}
	}
	clone.Territories = make([]models.Territory, len(source.Territories))
	for i, territory := range source.Territories {
		clone.Territories[i] = territory
		clone.Territories[i].Adjacencies = cloneSlice(territory.Adjacencies)
	}
	clone.Nobles = cloneSlice(source.Nobles)
	clone.Armies = make([]models.Army, len(source.Armies))
	for i, army := range source.Armies {
		clone.Armies[i] = army
		if army.ChainID != nil {
			chainID := *army.ChainID
			clone.Armies[i].ChainID = &chainID
		}
	}
	clone.Chains = make([]models.Chain, len(source.Chains))
	for i, chain := range source.Chains {
		clone.Chains[i] = cloneChain(chain)
	}
	clone.Infrastructures = cloneSlice(source.Infrastructures)
	clone.TerritoryStates = make(map[models.TerritoryID]models.TerritoryState, len(source.TerritoryStates))
	for territoryID, state := range source.TerritoryStates {
		copyState := state
		if state.OwnerID != nil {
			ownerID := *state.OwnerID
			copyState.OwnerID = &ownerID
		}
		if state.Army != nil {
			armyID := *state.Army
			copyState.Army = &armyID
		}
		copyState.Infrastructures = cloneSlice(state.Infrastructures)
		clone.TerritoryStates[territoryID] = copyState
	}
	return &clone
}

func cloneChain(source models.Chain) models.Chain {
	clone := source
	clone.Orders = make([]models.Order, len(source.Orders))
	for i, order := range source.Orders {
		clone.Orders[i] = order
		clone.Orders[i].TargetIDs = cloneSlice(order.TargetIDs)
		clone.Orders[i].NobleTargetIDs = cloneSlice(order.NobleTargetIDs)
		if order.NobleAssignments != nil {
			clone.Orders[i].NobleAssignments = make(map[models.TerritoryCode][]models.NobleCode, len(order.NobleAssignments))
			for destination, codes := range order.NobleAssignments {
				clone.Orders[i].NobleAssignments[destination] = cloneSlice(codes)
			}
		}
	}
	if source.PendingDisperse != nil {
		clone.PendingDisperse = &models.PendingDisperse{
			ArmyID:           source.PendingDisperse.ArmyID,
			SourceID:         source.PendingDisperse.SourceID,
			TargetIDs:        cloneSlice(source.PendingDisperse.TargetIDs),
			NobleAssignments: cloneNobleAssignments(source.PendingDisperse.NobleAssignments),
		}
	}
	return clone
}

func cloneNobleAssignments(source map[models.TerritoryCode][]models.NobleCode) map[models.TerritoryCode][]models.NobleCode {
	if source == nil {
		return nil
	}
	clone := make(map[models.TerritoryCode][]models.NobleCode, len(source))
	for destination, codes := range source {
		clone[destination] = cloneSlice(codes)
	}
	return clone
}

func cloneSlice[T any](source []T) []T {
	if source == nil {
		return nil
	}
	clone := make([]T, len(source))
	copy(clone, source)
	return clone
}
