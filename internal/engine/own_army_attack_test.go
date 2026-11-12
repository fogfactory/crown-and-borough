package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ownArmyLine(t *testing.T, ownerX, ownerY models.PlayerID, sizeX, sizeY int, yOrder *models.Order) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("TXX", "TXX", "TYY"),
			territory("TYY", "TYY", "TXX", "TZZ"),
			territory("TZZ", "TZZ", "TYY"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: ownerX, TerritoryID: "TXX", Size: sizeX},
			{ID: "A2", OwnerID: ownerY, TerritoryID: "TYY", Size: sizeY},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", ownerX, "TXX")
	addNoble(state, "N2", "TWO", ownerY, "TYY")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "TXX", TargetIDs: []models.TerritoryID{"TYY"}})
	if yOrder != nil {
		addChain(t, state, "A2", "N2", *yOrder)
	}
	validateTestState(t, state)
	return state
}

func TestResolveAttackChainAdvancesWhateverTheOwners(t *testing.T) {
	forward := models.Order{Type: models.OrderTypeAttack, PositionID: "TYY", TargetIDs: []models.TerritoryID{"TZZ"}}
	for _, owner := range []models.PlayerID{"P1", "P2"} {
		resolution, err := Resolve(ownArmyLine(t, "P1", owner, 2, 2, &forward), testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TYY" {
			t.Errorf("owner %s: A1 = %+v, want TYY", owner, army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "TZZ" {
			t.Errorf("owner %s: A2 = %+v, want TZZ", owner, army)
		}
	}
}

func TestResolveAttackDislodgesStationaryOwnArmy(t *testing.T) {
	resolution, err := Resolve(ownArmyLine(t, "P1", "P1", 3, 2, nil), testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TYY" {
		t.Errorf("A1 = %+v, want TYY", army)
	}
	// The dislodged army retreats to the only free neighbour, TZZ.
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "TZZ" {
		t.Errorf("A2 = %+v, want a retreat to TZZ", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "attack_wins" {
		t.Errorf("A1 outcome = %#v, found=%t, want attack_wins", event, found)
	}
}

func TestResolveWeakerAttackOnOwnArmyFails(t *testing.T) {
	resolution, err := Resolve(ownArmyLine(t, "P1", "P1", 2, 3, nil), testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TXX" {
		t.Errorf("A1 = %+v, want TXX", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "TYY" {
		t.Errorf("A2 = %+v, want TYY", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "combat_lost" {
		t.Errorf("A1 outcome = %#v, found=%t, want combat_lost", event, found)
	}
}
