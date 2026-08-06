package orders

import "github.com/fogfactory/crown-and-borough/internal/models"

type gameIndexes struct {
	territoriesByCode map[string]models.TerritoryID
	territoriesByID   map[models.TerritoryID]*models.Territory
	noblesByCode      map[string]models.NobleID
	noblesByID        map[models.NobleID]*models.Noble
	armiesByID        map[models.ArmyID]*models.Army
}

func indexGame(game *models.GameState) gameIndexes {
	indexes := gameIndexes{
		territoriesByCode: map[string]models.TerritoryID{},
		territoriesByID:   map[models.TerritoryID]*models.Territory{},
		noblesByCode:      map[string]models.NobleID{},
		noblesByID:        map[models.NobleID]*models.Noble{},
		armiesByID:        map[models.ArmyID]*models.Army{},
	}
	if game == nil {
		return indexes
	}
	for i := range game.Territories {
		territory := &game.Territories[i]
		if _, exists := indexes.territoriesByID[territory.ID]; !exists {
			indexes.territoriesByID[territory.ID] = territory
		}
		if _, exists := indexes.territoriesByCode[territory.Code]; !exists {
			indexes.territoriesByCode[territory.Code] = territory.ID
		}
	}
	for i := range game.Nobles {
		noble := &game.Nobles[i]
		if _, exists := indexes.noblesByID[noble.ID]; !exists {
			indexes.noblesByID[noble.ID] = noble
		}
		if _, exists := indexes.noblesByCode[noble.Code]; !exists {
			indexes.noblesByCode[noble.Code] = noble.ID
		}
	}
	for i := range game.Armies {
		army := &game.Armies[i]
		if _, exists := indexes.armiesByID[army.ID]; !exists {
			indexes.armiesByID[army.ID] = army
		}
	}
	return indexes
}

func isCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, character := range code {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func adjacent(indexes gameIndexes, from, to models.TerritoryID) bool {
	territory := indexes.territoriesByID[from]
	if territory == nil {
		return false
	}
	for _, adjacentID := range territory.Adjacencies {
		if adjacentID == to {
			return true
		}
	}
	return false
}

// armyAtTerritory resolves both storage indexes: the TerritoryState pointer
// and the Army record must agree before an army can receive a chain.
func armyAtTerritory(game *models.GameState, indexes gameIndexes, territoryID models.TerritoryID) *models.Army {
	if game == nil {
		return nil
	}
	state, exists := game.TerritoryStates[territoryID]
	if !exists || state.Army == nil {
		return nil
	}
	army := indexes.armiesByID[*state.Army]
	if army == nil || army.TerritoryID != territoryID {
		return nil
	}
	return army
}

func cloneChain(chain models.Chain) models.Chain {
	clone := chain
	clone.Orders = make([]models.Order, len(chain.Orders))
	for i, order := range chain.Orders {
		copyOrder := order
		copyOrder.TargetIDs = append([]models.TerritoryID(nil), order.TargetIDs...)
		copyOrder.NobleTargetIDs = append([]models.NobleID(nil), order.NobleTargetIDs...)
		if order.NobleAssignments != nil {
			copyOrder.NobleAssignments = make(map[models.TerritoryCode][]models.NobleCode, len(order.NobleAssignments))
			for destination, nobleCodes := range order.NobleAssignments {
				copyOrder.NobleAssignments[destination] = append([]models.NobleCode(nil), nobleCodes...)
			}
		}
		clone.Orders[i] = copyOrder
	}
	return clone
}
