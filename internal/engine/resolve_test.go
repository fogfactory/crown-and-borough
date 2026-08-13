package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveAttackIsPureAndUpdatesControl(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	validateTestState(t, state)
	before := cloneGameState(state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("Resolve mutated its input")
	}
	army := armyByID(t, resolution.State, "A1")
	if army.TerritoryID != "T02" || army.ChainID != nil {
		t.Fatalf("A1 = %+v, want moved to T02 without a chain", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T02" {
		t.Errorf("N1 location = %q, want T02", noble.LocationID)
	}
	owner := resolution.State.TerritoryStates["T02"].OwnerID
	if owner == nil || *owner != "P1" {
		t.Errorf("T02 owner = %v, want P1", owner)
	}
	if sourceOwner := resolution.State.TerritoryStates["T01"].OwnerID; sourceOwner == nil || *sourceOwner != "P1" {
		t.Errorf("T01 owner = %v, want P1 remanence after departure", sourceOwner)
	}
	if !containsEvent(resolution.Events, EventTypeMovement) || !containsEvent(resolution.Events, EventTypeControlChanged) {
		t.Errorf("events = %#v, want movement and control events", resolution.Events)
	}
}

func TestResolveSupportedCombatRetreatsDefender(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03", "T04"),
			territory("T03", "CCC", "T02"),
			territory("T04", "DDD", "T02"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N3", "THR", "P1", "T03")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" {
		t.Errorf("A1 territory = %q, want T02", army.TerritoryID)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T04" {
		t.Errorf("A2 retreat = %q, want T04", army.TerritoryID)
	}
	if !containsEvent(resolution.Events, EventTypeCombat) || !containsEvent(resolution.Events, EventTypeRetreat) {
		t.Errorf("events = %#v, want combat and retreat events", resolution.Events)
	}
}

func TestResolveCastleBlocksEqualAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
		t.Errorf("A1 territory = %q, want T01 after castle standoff", army.TerritoryID)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeCombat && event.TerritoryID == "T02" {
			if event.BaseDefense != 1 || event.Defense != 1 {
				t.Errorf("castle combat = %+v, want base/total defense 1", event)
			}
			return
		}
	}
	t.Fatal("missing castle combat event")
}

func TestResolveSupportCutAndDefensiveSupport(t *testing.T) {
	t.Run("cut", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02", "T04"),
				territory("T04", "DDD", "T03"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
				{ID: "A4", OwnerID: "P2", TerritoryID: "T04", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addNoble(state, "N4", "FOU", "P2", "T04")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}})
		addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T03"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
			t.Errorf("A1 territory = %q, want T01 after cut support", army.TerritoryID)
		}
		foundCut := false
		for _, event := range resolution.Events {
			if event.Type != EventTypeCombat || event.TerritoryID != "T02" {
				continue
			}
			for _, supporterID := range event.CutSupporterIDs {
				if supporterID == "A3" {
					foundCut = true
				}
			}
		}
		if !foundCut {
			t.Error("combat event for T02 did not report cut supporter A3")
		}
	})

	t.Run("defensive", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addNoble(state, "N3", "THR", "P2", "T03")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "T02"})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T02"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
			t.Errorf("A1 territory = %q, want T01 against defensive support", army.TerritoryID)
		}
	})
}

func TestResolveJoinPairAndCrossing(t *testing.T) {
	t.Run("pair", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T03"),
				territory("T02", "BBB", "T03"),
				territory("T03", "CCC", "T01", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T03"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !hasArmy(resolution.State, "A1") || hasArmy(resolution.State, "A2") {
			t.Fatalf("pair armies = %#v, want only A1", resolution.State.Armies)
		}
		army := armyByID(t, resolution.State, "A1")
		if army.TerritoryID != "T03" || army.Size != 2 || army.ChainID != nil {
			t.Errorf("merged army = %+v, want A1 size 2 at T03 without chain", army)
		}
	})

	t.Run("crossing", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 {
			t.Errorf("A1 = %+v, want separate arrival at T02", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" || army.Size != 1 {
			t.Errorf("A2 = %+v, want separate arrival at T01", army)
		}
	})
}

func TestResolvePartialDisperseAllocatesStableArmyIDs(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01", "T04"),
			territory("T04", "DDD", "T03"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P2", "T04")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{
			"BBB": {"ONE"},
		},
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T03"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("carrier = %+v, want A1 at T02 size 1 with consumed single chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T01" || army.Size != 1 {
		t.Errorf("residual = %+v, want A3 at T01 size 1", army)
	}
	if got := resolution.State.NextArmyID; got != 4 {
		t.Errorf("NextArmyID = %d, want 4", got)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T02" {
		t.Errorf("N1 location = %q, want T02", noble.LocationID)
	}
}

func TestResolveRejectsConflictingDisperseDestinations(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T03"),
			territory("T02", "BBB", "T03"),
			territory("T03", "CCC", "T01", "T02"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P2", "T02")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"*"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T02",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"*"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolution.State.TerritoryStates["T03"].Army != nil {
		t.Errorf("T03 army = %q, want no army after conflicting D branches", *resolution.State.TerritoryStates["T03"].Army)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
		t.Errorf("A1 territory = %q, want T01", army.TerritoryID)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" {
		t.Errorf("A2 territory = %q, want T02", army.TerritoryID)
	}
}

func TestResolveJoinUsesFailedJoinerAsHost(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01", "T04"),
			territory("T04", "DDD", "T03"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T02")
	addNoble(state, "N3", "THR", "P2", "T04")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T03"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasArmy(resolution.State, "A2") {
		t.Fatal("A2 should fuse into A1 after A1's join fails")
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 2 {
		t.Errorf("A1 = %+v, want size 2 host at T01", army)
	}
}

func TestResolveJoinAndDisperseDependencies(t *testing.T) {
	t.Run("D bounce makes source a host", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "CCC", "T01", "T04"),
				territory("T04", "DDD", "T03"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "T04", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addNoble(state, "N3", "THR", "P2", "T04")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		addChain(t, state, "A3", "N3", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T04",
			TargetIDs:        []models.TerritoryID{"T03"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if hasArmy(resolution.State, "A2") {
			t.Fatal("A2 should fuse into stationary A1 after the D bounce")
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 2 {
			t.Errorf("A1 = %+v, want A1 host at T01", army)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T03" {
			t.Errorf("A3 = %+v, want D branch at T03", army)
		}
	})

	t.Run("D enters a successfully vacated J source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "CCC", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T02",
			TargetIDs:        []models.TerritoryID{"T01"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"AAA": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T03" {
			t.Errorf("A1 = %+v, want J arrival at T03", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" {
			t.Errorf("A2 = %+v, want D arrival at vacated T01", army)
		}
	})

	t.Run("D enters a completed D source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addChain(t, state, "A1", "N1", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T01",
			TargetIDs:        []models.TerritoryID{"T02"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"*"}},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T02",
			TargetIDs:        []models.TerritoryID{"T03"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" {
			t.Errorf("A1 = %+v, want D arrival at T02", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" {
			t.Errorf("A2 = %+v, want completed D branch at T03", army)
		}
	})

	t.Run("J enters a completed D source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "CCC", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T03")
		addChain(t, state, "A1", "N1", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T01",
			TargetIDs:        []models.TerritoryID{"T02"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"*"}},
		})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" {
			t.Errorf("A1 = %+v, want D arrival at T02", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" {
			t.Errorf("A2 = %+v, want J arrival at vacated T01", army)
		}
	})
}

func TestResolveJoinPairWaitsForOutgoingJoinFailure(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03", "T05"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
			territory("T04", "DDD", "T05"),
			territory("T05", "EEE", "T01", "T04", "T06"),
			territory("T06", "FFF", "T05"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "T04", Size: 1},
			{ID: "A5", OwnerID: "P1", TerritoryID: "T06", Size: 1},
		},
	)
	for _, noble := range []struct {
		id        models.NobleID
		code      string
		territory models.TerritoryID
	}{
		{"N1", "ONE", "T01"},
		{"N2", "TWO", "T02"},
		{"N3", "THR", "T03"},
		{"N4", "FOU", "T04"},
		{"N5", "FIV", "T06"},
	} {
		addNoble(state, noble.id, noble.code, "P1", noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "T01", TargetIDs: []models.TerritoryID{"T05"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeJoin, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "T04", TargetIDs: []models.TerritoryID{"T05"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeJoin, PositionID: "T06", TargetIDs: []models.TerritoryID{"T05"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "T01"}, {"A2", "T02"}, {"A3", "T03"}, {"A4", "T04"}, {"A5", "T06"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s territory = %q, want %q", want.armyID, army.TerritoryID, want.territoryID)
		}
	}
}

func TestResolveLoopDisperseMovesResolvedBranchesAndRetriesResidual(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03", "T05"),
			territory("T02", "BBB", "T01", "T05"),
			territory("T03", "CCC", "T01", "T04"),
			territory("T04", "DDD", "T03"),
			territory("T05", "EEE", "T01", "T02"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P2", "T04")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{
			"BBB": {"ONE"},
		},
		Liaison: models.LiaisonModeLoop,
	})
	state.Chains[0].Orders = append(state.Chains[0].Orders, models.Order{
		ID:         "O2",
		Type:       models.OrderTypeHold,
		ArmyID:     "A1",
		PositionID: "T02",
		Liaison:    models.LiaisonModeSingle,
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T03"}})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 || army.ChainID == nil || *army.ChainID != "C1" {
		t.Errorf("A1 after partial loop D = %+v, want resolved carrier branch at T02", army)
	}
	residual := armyByID(t, first.State, "A3")
	if residual.TerritoryID != "T01" || residual.Size != 1 || residual.ChainID != nil {
		t.Errorf("residual = %+v, want A3 without the carrier chain at T01", residual)
	}
	if len(first.State.Chains) != 1 || first.State.Chains[0].ArmyID != "A1" || first.State.Chains[0].CurrentIndex != 0 || first.State.Chains[0].PendingDisperse == nil || first.State.Chains[0].PendingDisperse.ArmyID != "A3" || !reflect.DeepEqual(first.State.Chains[0].PendingDisperse.TargetIDs, []models.TerritoryID{"T03"}) {
		t.Errorf("pending D chain = %#v, want C1 on A1 with A3 retrying T03", first.State.Chains)
	}
	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if residual := armyByID(t, second.State, "A3"); residual.TerritoryID != "T01" || residual.ChainID != nil {
		t.Errorf("retry residual = %+v, want retry army at T01", residual)
	}
	if chain := second.State.Chains[0]; chain.PendingDisperse == nil || chain.PendingDisperse.ArmyID != "A3" {
		t.Errorf("second pending D chain = %#v, want A3 retry state", chain)
	}
	carrierDefeat := cloneGameState(second.State)
	attackerID := models.ArmyID("A4")
	attackerOwner := models.PlayerID("P2")
	carrierDefeat.Armies = append(carrierDefeat.Armies, models.Army{ID: attackerID, OwnerID: attackerOwner, TerritoryID: "T05", Size: 2})
	attackerState := carrierDefeat.TerritoryStates["T05"]
	attackerState.Army = &attackerID
	attackerState.OwnerID = &attackerOwner
	carrierDefeat.TerritoryStates["T05"] = attackerState
	addInfrastructure(carrierDefeat, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T05"})
	carrierDefeat.NextArmyID = 5
	addNoble(carrierDefeat, "N4", "FOU", "P2", "T05")
	addChain(t, carrierDefeat, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "T05", TargetIDs: []models.TerritoryID{"T02"}})
	validateTestState(t, carrierDefeat)
	cancelled, err := Resolve(carrierDefeat, testBalance())
	if err != nil {
		t.Fatalf("carrier defeat Resolve: %v", err)
	}
	if hasArmy(cancelled.State, "A1") || armyByID(t, cancelled.State, "A3").TerritoryID != "T01" || len(cancelled.State.Chains) != 0 {
		t.Errorf("carrier defeat state = armies:%#v chains:%#v, want destroyed carrier, idle residual, no chains", cancelled.State.Armies, cancelled.State.Chains)
	}
	residualJoin := cloneGameState(second.State)
	withoutAttacker := make([]models.Army, 0, len(residualJoin.Armies)-1)
	for _, army := range residualJoin.Armies {
		if army.ID != "A2" {
			withoutAttacker = append(withoutAttacker, army)
		}
	}
	residualJoin.Armies = withoutAttacker
	openDestination := residualJoin.TerritoryStates["T03"]
	openDestination.Army = nil
	residualJoin.TerritoryStates["T03"] = openDestination
	joiningID := models.ArmyID("A4")
	joiningOwner := models.PlayerID("P1")
	residualJoin.Armies = append(residualJoin.Armies, models.Army{ID: joiningID, OwnerID: joiningOwner, TerritoryID: "T05", Size: 1})
	joiningState := residualJoin.TerritoryStates["T05"]
	joiningState.Army = &joiningID
	joiningState.OwnerID = &joiningOwner
	residualJoin.TerritoryStates["T05"] = joiningState
	residualJoin.NextArmyID = 5
	addNoble(residualJoin, "N4", "FOU", "P1", "T05")
	addChain(t, residualJoin, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "T05", TargetIDs: []models.TerritoryID{"T01"}})
	validateTestState(t, residualJoin)
	joined, err := Resolve(residualJoin, testBalance())
	if err != nil {
		t.Fatalf("pending residual J Resolve: %v", err)
	}
	if army := armyByID(t, joined.State, "A4"); army.TerritoryID != "T01" {
		t.Errorf("A4 = %+v, want J arrival at T01 rather than D destination", army)
	}
	if noble := nobleByID(t, joined.State, "N4"); noble.LocationID != "T01" {
		t.Errorf("N4 location = %q, want T01 with joining army", noble.LocationID)
	}
	retry := cloneGameState(second.State)
	remainingArmies := make([]models.Army, 0, len(retry.Armies)-1)
	for _, army := range retry.Armies {
		if army.ID != "A2" {
			remainingArmies = append(remainingArmies, army)
		}
	}
	retry.Armies = remainingArmies
	cleared := retry.TerritoryStates["T03"]
	cleared.Army = nil
	retry.TerritoryStates["T03"] = cleared
	validateTestState(t, retry)
	completed, err := Resolve(retry, testBalance())
	if err != nil {
		t.Fatalf("completed pending D Resolve: %v", err)
	}
	if chain := completed.State.Chains[0]; chain.ArmyID != "A1" || chain.CurrentIndex != 1 || chain.PendingDisperse != nil {
		t.Errorf("completed chain = %#v, want original carrier advanced to H", chain)
	}
	final, err := Resolve(completed.State, testBalance())
	if err != nil {
		t.Fatalf("future H Resolve: %v", err)
	}
	if len(final.State.Chains) != 0 || armyByID(t, final.State, "A1").ChainID != nil {
		t.Errorf("future order state = %#v, want consumed H chain", final.State.Chains)
	}
}

func TestResolveInvalidPillageBreaksLoopWhenDislodged(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P2", "T02")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "T01", Liaison: models.LiaisonModeLoop})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T03" || army.ChainID != nil {
		t.Errorf("A1 = %+v, want retreat to T03 with broken invalid P loop", army)
	}
}

func TestResolvePillageCreditsNearestControlledSettlement(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "T01"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
	owner := models.PlayerID("P1")
	castleState := state.TerritoryStates["T02"]
	castleState.OwnerID = &owner
	state.TerritoryStates["T02"] = castleState
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "T01"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolution.State.Infrastructures) != 1 || resolution.State.Infrastructures[0].ID != "I2" {
		t.Errorf("infrastructures = %#v, want only I2", resolution.State.Infrastructures)
	}
	wantResources := testBalance().PillageBonus + testBalance().BaseProduction + 1
	if got := resolution.State.TerritoryStates["T02"].Resources; got != wantResources {
		t.Errorf("castle resources = %d, want %d", got, wantResources)
	}
	if !containsEvent(resolution.Events, EventTypePillage) {
		t.Errorf("events = %#v, want a pillage event", resolution.Events)
	}
}

func TestResolveNobleCapture(t *testing.T) {
	t.Run("capture", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if hasArmy(resolution.State, "A1") {
			t.Error("A1 should be destroyed without a retreat")
		}
		if noble := nobleByID(t, resolution.State, "N1"); noble.Status != models.NobleStatusHostage || noble.LocationID != "T01" {
			t.Errorf("N1 = %+v, want hostage at T01", noble)
		}
		if !containsEvent(resolution.Events, EventTypeCapture) {
			t.Errorf("events = %#v, want capture event", resolution.Events)
		}
	})
}

func TestResolveLoopProgression(t *testing.T) {
	t.Run("hold", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{territory("T01", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "T01", Liaison: models.LiaisonModeLoop})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if len(resolution.State.Chains) != 1 || resolution.State.Chains[0].CurrentIndex != 0 {
			t.Errorf("chains = %#v, want held loop at index 0", resolution.State.Chains)
		}
	})

	t.Run("support while attack continues", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}, Liaison: models.LiaisonModeLoop})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if len(resolution.State.Chains) != 2 {
			t.Fatalf("chains = %#v, want attacker and support loops retained", resolution.State.Chains)
		}
		for _, chain := range resolution.State.Chains {
			if chain.CurrentIndex != 0 {
				t.Errorf("chain %q index = %d, want 0", chain.ID, chain.CurrentIndex)
			}
		}
	})
}

func TestResolveLoopSupport(t *testing.T) {
	t.Run("offensive advances when attack succeeds", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T03"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" {
			t.Errorf("A1 territory = %q, want T02 after the supported attack", army.TerritoryID)
		}
		if chain := chainOf(resolution.State, "A3"); chain == nil || chain.CurrentIndex != 1 {
			t.Errorf("support chain = %#v, want index 1 after attack success", chain)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionAdvanced {
			t.Errorf("S1 event = %#v, found=%t, want advanced", event, found)
		}

		second, err := Resolve(resolution.State, testBalance())
		if err != nil {
			t.Fatalf("second Resolve: %v", err)
		}
		if _, replayed := findOutcome(second.Events, "S1"); replayed {
			t.Errorf("S1 replayed after attack success: %#v", second.Events)
		}
		if event, found := findOutcome(second.Events, "S2"); !found || event.Outcome != OutcomeSuccess {
			t.Errorf("S2 event = %#v, found=%t, want suffix executed", event, found)
		}
	})

	t.Run("offensive retries while attack repelled", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T03"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
			t.Errorf("A1 territory = %q, want T01 after the repelled attack", army.TerritoryID)
		}
		if chain := chainOf(resolution.State, "A3"); chain == nil || chain.CurrentIndex != 0 {
			t.Errorf("support chain = %#v, want index 0 while attack repelled", chain)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionRetried {
			t.Errorf("S1 event = %#v, found=%t, want retried", event, found)
		}
	})

	t.Run("offensive retries while attacker dislodged", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T05", "T04"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
				territory("T04", "DDD", "T01"),
				territory("T05", "EEE", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
				{ID: "A4", OwnerID: "P2", TerritoryID: "T04", Size: 2},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addNoble(state, "N4", "FOU", "P2", "T04")
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T02"})
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T04"})
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T01"}})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T03"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T05" {
			t.Errorf("A1 territory = %q, want T05 after retreat", army.TerritoryID)
		}
		if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "T01" {
			t.Errorf("A4 territory = %q, want T01 after dislodging A1", army.TerritoryID)
		}
		if chain := chainOf(resolution.State, "A3"); chain == nil || chain.CurrentIndex != 0 {
			t.Errorf("support chain = %#v, want index 0 while attack failed by dislodgement", chain)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionRetried {
			t.Errorf("S1 event = %#v, found=%t, want retried", event, found)
		}
	})

	t.Run("defensive retries while supported army holds", func(t *testing.T) {
		t.Run("under attack", func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("T01", "AAA", "T02"),
					territory("T02", "BBB", "T01", "T03"),
					territory("T03", "CCC", "T02"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
					{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
					{ID: "A3", OwnerID: "P3", TerritoryID: "T03", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", "P1", "T01")
			addNoble(state, "N2", "TWO", "P2", "T02")
			addNoble(state, "N3", "THR", "P3", "T03")
			addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "T02"})
			addChain(t, state, "A3", "N3", models.Order{ID: "A3O", Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T02"}})
			addChainOrders(t, state, "A1", "N1",
				models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop},
				models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T01"},
			)
			validateTestState(t, state)

			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if chain := chainOf(resolution.State, "A1"); chain == nil || chain.CurrentIndex != 0 {
				t.Errorf("support chain = %#v, want index 0 while army holds", chain)
			}
			if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionRetried {
				t.Errorf("S1 event = %#v, found=%t, want retried", event, found)
			}
		})

		t.Run("unattacked", func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("T01", "AAA", "T02"),
					territory("T02", "BBB", "T01"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
					{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", "P1", "T01")
			addNoble(state, "N2", "TWO", "P2", "T02")
			addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "T02"})
			addChainOrders(t, state, "A1", "N1",
				models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop},
				models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T01"},
			)
			validateTestState(t, state)

			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if chain := chainOf(resolution.State, "A1"); chain == nil || chain.CurrentIndex != 0 {
				t.Errorf("support chain = %#v, want index 0 while army holds", chain)
			}
			if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionRetried {
				t.Errorf("S1 event = %#v, found=%t, want retried", event, found)
			}
		})
	})

	t.Run("defensive advances when supported army moves away", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03", "T04"),
				territory("T03", "CCC", "T02"),
				territory("T04", "DDD", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addNoble(state, "N4", "FOU", "P1", "T03")
		addChainOrders(t, state, "A1", "N1",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T01"},
		)
		addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T04"}})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T02"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T04" {
			t.Errorf("A2 territory = %q, want T04 after moving away", army.TerritoryID)
		}
		if chain := chainOf(resolution.State, "A1"); chain == nil || chain.CurrentIndex != 1 {
			t.Errorf("support chain = %#v, want index 1 after supported army left", chain)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionAdvanced {
			t.Errorf("S1 event = %#v, found=%t, want advanced", event, found)
		}
	})

	t.Run("defensive advances when supported army dislodged", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03", "T04", "T05"),
				territory("T03", "CCC", "T02"),
				territory("T04", "DDD", "T02", "T05"),
				territory("T05", "EEE", "T02", "T04"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "T04", Size: 2},
				{ID: "A5", OwnerID: "P1", TerritoryID: "T05", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addNoble(state, "N4", "FOU", "P1", "T04")
		addNoble(state, "N5", "FIV", "P1", "T05")
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T04"})
		addChainOrders(t, state, "A1", "N1",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "T01"},
		)
		addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "T02"})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "T04", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A5", "N5", models.Order{ID: "A5O", Type: models.OrderTypeSupport, PositionID: "T05", TargetIDs: []models.TerritoryID{"T04", "T02"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" {
			t.Errorf("A2 territory = %q, want T03 after retreat", army.TerritoryID)
		}
		if chain := chainOf(resolution.State, "A1"); chain == nil || chain.CurrentIndex != 1 {
			t.Errorf("support chain = %#v, want index 1 after supported army dislodged", chain)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Progression != ProgressionAdvanced {
			t.Errorf("S1 event = %#v, found=%t, want advanced", event, found)
		}
		if event, found := findOutcome(resolution.Events, "S1"); !found || event.Reason != "support_applied" {
			t.Errorf("S1 event = %#v, found=%t, want support_applied", event, found)
		}
	})
}

func TestResolveIsDeterministic(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02"),
		},
		[]models.Army{
			{ID: "A10", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N3", "THR", "P1", "T03")
	addChain(t, state, "A10", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T02"}})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	second, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if !reflect.DeepEqual(first.State, second.State) {
		t.Fatalf("states differ:\nfirst=%+v\nsecond=%+v", first.State, second.State)
	}
	if !reflect.DeepEqual(first.Events, second.Events) {
		t.Fatalf("events differ:\nfirst=%#v\nsecond=%#v", first.Events, second.Events)
	}
}

func TestResolveDeferredNonAdjacencyPreservesEarlierOrders(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02", "T04", "T05"),
			territory("T04", "DDD", "T03"),
			territory("T05", "EEE", "T03"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}},
		models.Order{Type: models.OrderTypeHold, PositionID: "T02"},
		models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T04"}},
		models.Order{Type: models.OrderTypePillage, PositionID: "T02"},
	)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "T02" {
		t.Fatalf("A1 territory = %q, want T02 after O1", army.TerritoryID)
	}
	if chain := chainOf(first.State, "A1"); chain == nil || chain.CurrentIndex != 1 {
		t.Fatalf("chain after O1 = %#v, want index 1", chain)
	}
	if event, found := findOutcome(first.Events, "O1"); !found || event.Outcome != OutcomeSuccess {
		t.Fatalf("O1 events = %#v, want success", first.Events)
	}

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if event, found := findOutcome(second.Events, "O2"); !found || event.Outcome != OutcomeSuccess {
		t.Fatalf("O2 events = %#v, want success", second.Events)
	}
	if chain := chainOf(second.State, "A1"); chain == nil || chain.CurrentIndex != 2 {
		t.Fatalf("chain after O2 = %#v, want index 2", chain)
	}

	third, err := Resolve(second.State, testBalance())
	if err != nil {
		t.Fatalf("third Resolve: %v", err)
	}
	event, found := findOutcome(third.Events, "O3")
	if !found {
		t.Fatalf("missing O3 outcome event in %#v", third.Events)
	}
	if event.Outcome != OutcomeInvalid || event.Reason != "non_adjacent_destination" || event.Progression != ProgressionBroken {
		t.Fatalf("O3 event = %#v, want invalid non_adjacent_destination broken", event)
	}
	if chain := chainOf(third.State, "A1"); chain != nil {
		t.Errorf("chain after O3 = %#v, want removed", chain)
	}
	if army := armyByID(t, third.State, "A1"); army.ChainID != nil {
		t.Errorf("A1 chain link = %#v, want nil", army.ChainID)
	}

	fourth, err := Resolve(third.State, testBalance())
	if err != nil {
		t.Fatalf("fourth Resolve: %v", err)
	}
	if _, executed := findOutcome(fourth.Events, "O4"); executed {
		t.Errorf("O4 was executed after the broken chain: %#v", fourth.Events)
	}
	if containsEvent(fourth.Events, EventTypePillage) {
		t.Errorf("suffix O4 pillage executed: %#v", fourth.Events)
	}
}

func TestResolveLoopNonAdjacentBreaksChainWithoutRetry(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02", "T04", "T05"),
			territory("T04", "DDD", "T03"),
			territory("T05", "EEE", "T03"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T02", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "T02")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T04"}, Liaison: models.LiaisonModeLoop},
		models.Order{Type: models.OrderTypePillage, PositionID: "T02"},
	)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event, found := findOutcome(resolution.Events, "O1")
	if !found {
		t.Fatalf("missing O1 outcome event in %#v", resolution.Events)
	}
	if event.Outcome != OutcomeInvalid || event.Progression != ProgressionBroken {
		t.Fatalf("O1 event = %#v, want invalid broken (not retried)", event)
	}
	if chain := chainOf(resolution.State, "A1"); chain != nil {
		t.Errorf("chain after break = %#v, want removed", chain)
	}
	if _, executed := findOutcome(resolution.Events, "O2"); executed {
		t.Errorf("O2 suffix executed after broken loop order: %#v", resolution.Events)
	}
}
