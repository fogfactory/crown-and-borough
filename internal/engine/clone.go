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
	clone.Privacy = clonePrivacy(source.Privacy)
	clone.Infrastructures = cloneSlice(source.Infrastructures)
	clone.Regions = cloneSlice(source.Regions)
	for i, region := range source.Regions {
		clone.Regions[i] = region
		clone.Regions[i].Territories = cloneSlice(region.Territories)
	}
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
	clone.SpecialDeck = cloneSpecialDeck(source.SpecialDeck)
	if source.Auguries != nil {
		clone.Auguries = make(map[int]models.YearAugury, len(source.Auguries))
		for year, augury := range source.Auguries {
			clone.Auguries[year] = cloneYearAugury(augury)
		}
	}
	return &clone
}

func cloneSpecialDeck(source *models.SpecialDeck) *models.SpecialDeck {
	if source == nil {
		return nil
	}
	clone := &models.SpecialDeck{
		Cards:    cloneSlice(source.Cards),
		DrawPile: cloneSlice(source.DrawPile),
		Discard:  cloneSlice(source.Discard),
	}
	if source.Hands != nil {
		clone.Hands = make(map[models.PlayerID][]models.SpecialCardID, len(source.Hands))
		for playerID, hand := range source.Hands {
			clone.Hands[playerID] = cloneSlice(hand)
		}
	}
	return clone
}

func cloneYearAugury(source models.YearAugury) models.YearAugury {
	clone := source
	if source.Capacities != nil {
		clone.Capacities = make(map[models.Season]int, len(source.Capacities))
		for season, capacity := range source.Capacities {
			clone.Capacities[season] = capacity
		}
	}
	clone.Calamities = cloneSlice(source.Calamities)
	return clone
}

func clonePrivacy(source *models.PrivacyMeta) *models.PrivacyMeta {
	if source == nil {
		return nil
	}
	clone := &models.PrivacyMeta{
		ChainKnowledge:      make(map[models.PlayerID]map[models.ChainID]models.ChainSnapshot, len(source.ChainKnowledge)),
		CombatParticipation: make(map[models.PlayerID]map[string]bool, len(source.CombatParticipation)),
	}
	for playerID, snapshots := range source.ChainKnowledge {
		copySnapshots := make(map[models.ChainID]models.ChainSnapshot, len(snapshots))
		for chainID, snapshot := range snapshots {
			copySnapshot := snapshot
			copySnapshot.Orders = make([]models.Order, len(snapshot.Orders))
			for index, order := range snapshot.Orders {
				copySnapshot.Orders[index] = cloneOrder(order)
			}
			copySnapshots[chainID] = copySnapshot
		}
		clone.ChainKnowledge[playerID] = copySnapshots
	}
	for playerID, combats := range source.CombatParticipation {
		copyCombats := make(map[string]bool, len(combats))
		for combatID, participating := range combats {
			copyCombats[combatID] = participating
		}
		clone.CombatParticipation[playerID] = copyCombats
	}
	return clone
}

func cloneOrder(source models.Order) models.Order {
	clone := source
	clone.TargetIDs = cloneSlice(source.TargetIDs)
	clone.NobleAssignments = cloneNobleAssignments(source.NobleAssignments)
	return clone
}

func cloneChain(source models.Chain) models.Chain {
	clone := source
	clone.Orders = make([]models.Order, len(source.Orders))
	for i, order := range source.Orders {
		clone.Orders[i] = cloneOrder(order)
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

func cloneNobleAssignments(source map[models.TerritoryID][]models.NobleCode) map[models.TerritoryID][]models.NobleCode {
	if source == nil {
		return nil
	}
	clone := make(map[models.TerritoryID][]models.NobleCode, len(source))
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
