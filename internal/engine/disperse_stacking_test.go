package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveFriendlyDispersesStackOnEmptyDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "CCC",
		TargetIDs:        []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"TWO"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasArmy(resolution.State, "A2") {
		t.Fatal("second friendly disperse carrier should fuse into the first arrival")
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("stacked arrival = %+v, want A1 size 2 at BBB without a chain", army)
	}
	if event, found := fusionForArmy(resolution.Events, "A1"); !found || event.OtherArmyID != "A2" || event.TerritoryID != "BBB" {
		t.Errorf("disperse fusion = %#v, found=%t, want A1/A2 at BBB", event, found)
	}
}

func TestResolveFriendlyDisperseFusesWithJoinArrival(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "CCC",
		TargetIDs:  []models.TerritoryID{"BBB"},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("friendly J/D arrival = %+v, want A2 size 2 at BBB without a chain", army)
	}
	if hasArmy(resolution.State, "A1") {
		t.Fatal("disperse carrier should fuse into the friendly join arrival")
	}
}

func TestResolveFriendlyDispersesFuseWithStationaryFriendlyArmy(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "CCC",
		TargetIDs:        []models.TerritoryID{"BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"THR"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" || army.Size != 4 {
		t.Errorf("friendly host = %+v, want size 4 at BBB", army)
	}
	if hasArmy(resolution.State, "A1") || hasArmy(resolution.State, "A3") {
		t.Fatal("friendly disperse carriers should fuse into the stationary host")
	}
}

func fusionForArmy(events []Event, armyID models.ArmyID) (Event, bool) {
	for _, event := range events {
		if event.Type == EventTypeFusion && event.ArmyID == armyID {
			return event, true
		}
	}
	return Event{}, false
}
