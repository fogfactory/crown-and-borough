package engine

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func testState(t *testing.T, territories []models.Territory, armies []models.Army) *models.GameState {
	t.Helper()
	state := models.NewGameState()
	state.ID = "resolution-test"
	state.Seed = "resolution-test"
	state.Players = []models.Player{
		{ID: "P1", Name: "One", Color: "red"},
		{ID: "P2", Name: "Two", Color: "blue"},
		{ID: "P3", Name: "Three", Color: "green"},
	}
	state.Territories = append([]models.Territory(nil), territories...)
	state.Armies = append([]models.Army(nil), armies...)
	state.TerritoryStates = make(map[models.TerritoryID]models.TerritoryState, len(territories))
	for _, territory := range territories {
		state.TerritoryStates[territory.ID] = models.TerritoryState{Infrastructures: []models.InfraID{}}
	}
	for _, army := range armies {
		territoryState := state.TerritoryStates[army.TerritoryID]
		armyID := army.ID
		ownerID := army.OwnerID
		territoryState.Army = &armyID
		territoryState.OwnerID = &ownerID
		state.TerritoryStates[army.TerritoryID] = territoryState
	}
	state.NextArmyID = nextArmyID(armies)
	return state
}

func addNoble(state *models.GameState, id models.NobleID, code string, ownerID models.PlayerID, territoryID models.TerritoryID) {
	state.Nobles = append(state.Nobles, models.Noble{
		ID:         id,
		Code:       code,
		Name:       string(code),
		OwnerID:    ownerID,
		LocationID: territoryID,
		Status:     models.NobleStatusFree,
	})
}

func addChain(t *testing.T, state *models.GameState, armyID models.ArmyID, nobleID models.NobleID, order models.Order) {
	t.Helper()
	chainID := models.ChainID(fmt.Sprintf("C%d", state.NextChainID))
	order.ArmyID = armyID
	if order.ID == "" {
		order.ID = "O1"
	}
	if order.Liaison == "" {
		order.Liaison = models.LiaisonModeSingle
	}
	state.Chains = append(state.Chains, models.Chain{
		ID:           chainID,
		NobleID:      nobleID,
		ArmyID:       armyID,
		Orders:       []models.Order{order},
		CurrentIndex: 0,
	})
	for index := range state.Armies {
		if state.Armies[index].ID == armyID {
			state.Armies[index].ChainID = &chainID
			break
		}
	}
	state.NextChainID++
}

func addInfrastructure(state *models.GameState, infrastructure models.Infrastructure) {
	state.Infrastructures = append(state.Infrastructures, infrastructure)
	territoryState := state.TerritoryStates[infrastructure.TerritoryID]
	territoryState.Infrastructures = append(territoryState.Infrastructures, infrastructure.ID)
	state.TerritoryStates[infrastructure.TerritoryID] = territoryState
}

func validateTestState(t *testing.T, state *models.GameState) {
	t.Helper()
	if err := state.Validate(); err != nil {
		t.Fatalf("invalid test state: %v", err)
	}
}

func armyByID(t *testing.T, state *models.GameState, armyID models.ArmyID) models.Army {
	t.Helper()
	for _, army := range state.Armies {
		if army.ID == armyID {
			return army
		}
	}
	t.Fatalf("army %q not found", armyID)
	return models.Army{}
}

func hasArmy(state *models.GameState, armyID models.ArmyID) bool {
	for _, army := range state.Armies {
		if army.ID == armyID {
			return true
		}
	}
	return false
}

func nobleByID(t *testing.T, state *models.GameState, nobleID models.NobleID) models.Noble {
	t.Helper()
	for _, noble := range state.Nobles {
		if noble.ID == nobleID {
			return noble
		}
	}
	t.Fatalf("noble %q not found", nobleID)
	return models.Noble{}
}

func containsEvent(events []Event, eventType EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func nextArmyID(armies []models.Army) int {
	next := 1
	for _, army := range armies {
		value := string(army.ID)
		if len(value) < 2 || value[0] != 'A' {
			continue
		}
		sequence, err := strconv.Atoi(value[1:])
		if err == nil && sequence >= next {
			next = sequence + 1
		}
	}
	return next
}

func territory(id, code string, neighbors ...models.TerritoryID) models.Territory {
	return models.Territory{
		ID:          models.TerritoryID(id),
		Code:        code,
		Name:        code,
		Terrain:     models.TerrainPlain,
		Adjacencies: neighbors,
	}
}
