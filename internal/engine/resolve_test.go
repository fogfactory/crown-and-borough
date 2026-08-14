package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveAttackIsPureAndUpdatesControl(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
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
	if army.TerritoryID != "BBB" || army.ChainID != nil {
		t.Fatalf("A1 = %+v, want moved to BBB without a chain", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "BBB" {
		t.Errorf("N1 location = %q, want BBB", noble.LocationID)
	}
	owner := resolution.State.TerritoryStates["BBB"].OwnerID
	if owner == nil || *owner != "P1" {
		t.Errorf("BBB owner = %v, want P1", owner)
	}
	if sourceOwner := resolution.State.TerritoryStates["AAA"].OwnerID; sourceOwner == nil || *sourceOwner != "P1" {
		t.Errorf("AAA owner = %v, want P1 remanence after departure", sourceOwner)
	}
	if !containsEvent(resolution.Events, EventTypeMovement) || !containsEvent(resolution.Events, EventTypeControlChanged) {
		t.Errorf("events = %#v, want movement and control events", resolution.Events)
	}
}

func TestResolveSupportedCombatRetreatsDefender(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC", "DDD"),
			territory("CCC", "CCC", "BBB"),
			territory("DDD", "DDD", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
		t.Errorf("A1 territory = %q, want BBB", army.TerritoryID)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "DDD" {
		t.Errorf("A2 retreat = %q, want DDD", army.TerritoryID)
	}
	if !containsEvent(resolution.Events, EventTypeCombat) || !containsEvent(resolution.Events, EventTypeRetreat) {
		t.Errorf("events = %#v, want combat and retreat events", resolution.Events)
	}
}

func TestResolveCastleBlocksEqualAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setNobleStatus(state, "N1", models.NobleStatusHostage)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Errorf("A1 territory = %q, want AAA after castle standoff", army.TerritoryID)
	}
	for _, event := range resolution.Events {
		if event.Type == EventTypeCombat && event.TerritoryID == "BBB" {
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB", "DDD"),
				territory("DDD", "DDD", "CCC"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
				{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addNoble(state, "N4", "FOU", "P2", "DDD")
		setNobleStatus(state, "N1", models.NobleStatusHostage)
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
		addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
			t.Errorf("A1 territory = %q, want AAA after cut support", army.TerritoryID)
		}
		foundCut := false
		for _, event := range resolution.Events {
			if event.Type != EventTypeCombat || event.TerritoryID != "BBB" {
				continue
			}
			for _, supporterID := range event.CutSupporterIDs {
				if supporterID == "A3" {
					foundCut = true
				}
			}
		}
		if !foundCut {
			t.Error("combat event for BBB did not report cut supporter A3")
		}
	})

	t.Run("defensive", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addNoble(state, "N3", "THR", "P2", "CCC")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "BBB"})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
			t.Errorf("A1 territory = %q, want AAA against defensive support", army.TerritoryID)
		}
	})
}

func TestResolveJoinPairAndCrossing(t *testing.T) {
	t.Run("pair", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "CCC"),
				territory("BBB", "BBB", "CCC"),
				territory("CCC", "CCC", "AAA", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"CCC"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !hasArmy(resolution.State, "A1") || hasArmy(resolution.State, "A2") {
			t.Fatalf("pair armies = %#v, want only A1", resolution.State.Armies)
		}
		army := armyByID(t, resolution.State, "A1")
		if army.TerritoryID != "CCC" || army.Size != 2 || army.ChainID != nil {
			t.Errorf("merged army = %+v, want A1 size 2 at CCC without chain", army)
		}
	})

	t.Run("crossing", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 {
			t.Errorf("A1 = %+v, want separate arrival at BBB", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" || army.Size != 1 {
			t.Errorf("A2 = %+v, want separate arrival at AAA", army)
		}
	})
}

func TestResolvePartialDisperseAllocatesStableArmyIDs(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "DDD")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{
			"BBB": {"ONE"},
		},
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("carrier = %+v, want A1 at BBB size 1 with consumed single chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "AAA" || army.Size != 1 {
		t.Errorf("residual = %+v, want A3 at AAA size 1", army)
	}
	if got := resolution.State.NextArmyID; got != 4 {
		t.Errorf("NextArmyID = %d, want 4", got)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "BBB" {
		t.Errorf("N1 location = %q, want BBB", noble.LocationID)
	}
}

func TestResolveRejectsConflictingDisperseDestinations(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "CCC"),
			territory("BBB", "BBB", "CCC"),
			territory("CCC", "CCC", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"*"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "BBB",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"*"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolution.State.TerritoryStates["CCC"].Army != nil {
		t.Errorf("CCC army = %q, want no army after conflicting D branches", *resolution.State.TerritoryStates["CCC"].Army)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Errorf("A1 territory = %q, want AAA", army.TerritoryID)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("A2 territory = %q, want BBB", army.TerritoryID)
	}
}

func TestResolveJoinUsesFailedJoinerAsHost(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P2", "DDD")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasArmy(resolution.State, "A2") {
		t.Fatal("A2 should fuse into A1 after A1's join fails")
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 {
		t.Errorf("A1 = %+v, want size 2 host at AAA", army)
	}
}

func TestResolveJoinAndDisperseDependencies(t *testing.T) {
	t.Run("D bounce makes source a host", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA"),
				territory("CCC", "CCC", "AAA", "DDD"),
				territory("DDD", "DDD", "CCC"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addNoble(state, "N3", "THR", "P2", "DDD")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		addChain(t, state, "A3", "N3", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "DDD",
			TargetIDs:        []models.TerritoryID{"CCC"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if hasArmy(resolution.State, "A2") {
			t.Fatal("A2 should fuse into stationary A1 after the D bounce")
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 {
			t.Errorf("A1 = %+v, want A1 host at AAA", army)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "CCC" {
			t.Errorf("A3 = %+v, want D branch at CCC", army)
		}
	})

	t.Run("D enters a successfully vacated J source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA"),
				territory("CCC", "CCC", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "BBB",
			TargetIDs:        []models.TerritoryID{"AAA"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"AAA": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "CCC" {
			t.Errorf("A1 = %+v, want J arrival at CCC", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
			t.Errorf("A2 = %+v, want D arrival at vacated AAA", army)
		}
	})

	t.Run("D enters a completed D source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addChain(t, state, "A1", "N1", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "AAA",
			TargetIDs:        []models.TerritoryID{"BBB"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"*"}},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "BBB",
			TargetIDs:        []models.TerritoryID{"CCC"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"*"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want D arrival at BBB", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" {
			t.Errorf("A2 = %+v, want completed D branch at CCC", army)
		}
	})

	t.Run("J enters a completed D source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA"),
				territory("CCC", "CCC", "AAA"),
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
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"*"}},
		})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want D arrival at BBB", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
			t.Errorf("A2 = %+v, want J arrival at vacated AAA", army)
		}
	})
}

func TestResolveJoinPairWaitsForOutgoingJoinFailure(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "EEE"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
			territory("DDD", "DDD", "EEE"),
			territory("EEE", "EEE", "AAA", "DDD", "FFF"),
			territory("FFF", "FFF", "EEE"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
			{ID: "A5", OwnerID: "P1", TerritoryID: "FFF", Size: 1},
		},
	)
	for _, noble := range []struct {
		id        models.NobleID
		code      string
		territory models.TerritoryID
	}{
		{"N1", "ONE", "AAA"},
		{"N2", "TWO", "BBB"},
		{"N3", "THR", "CCC"},
		{"N4", "FOU", "DDD"},
		{"N5", "FIV", "FFF"},
	} {
		addNoble(state, noble.id, noble.code, "P1", noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"EEE"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeJoin, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "DDD", TargetIDs: []models.TerritoryID{"EEE"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeJoin, PositionID: "FFF", TargetIDs: []models.TerritoryID{"EEE"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "AAA"}, {"A2", "BBB"}, {"A3", "CCC"}, {"A4", "DDD"}, {"A5", "FFF"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s territory = %q, want %q", want.armyID, army.TerritoryID, want.territoryID)
		}
	}
}

func TestResolveLoopDisperseMovesResolvedBranchesAndRetriesResidual(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "EEE"),
			territory("BBB", "BBB", "AAA", "EEE"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "DDD")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{
			"BBB": {"ONE"},
		},
		Liaison: models.LiaisonModeLoop,
	})
	state.Chains[0].Orders = append(state.Chains[0].Orders, models.Order{
		ID:         "O2",
		Type:       models.OrderTypeHold,
		ArmyID:     "A1",
		PositionID: "BBB",
		Liaison:    models.LiaisonModeSingle,
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC"}})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 || army.ChainID == nil || *army.ChainID != "C1" {
		t.Errorf("A1 after partial loop D = %+v, want resolved carrier branch at BBB", army)
	}
	residual := armyByID(t, first.State, "A3")
	if residual.TerritoryID != "AAA" || residual.Size != 1 || residual.ChainID != nil {
		t.Errorf("residual = %+v, want A3 without the carrier chain at AAA", residual)
	}
	if len(first.State.Chains) != 1 || first.State.Chains[0].ArmyID != "A1" || first.State.Chains[0].CurrentIndex != 0 || first.State.Chains[0].PendingDisperse == nil || first.State.Chains[0].PendingDisperse.ArmyID != "A3" || !reflect.DeepEqual(first.State.Chains[0].PendingDisperse.TargetIDs, []models.TerritoryID{"CCC"}) {
		t.Errorf("pending D chain = %#v, want C1 on A1 with A3 retrying CCC", first.State.Chains)
	}
	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if residual := armyByID(t, second.State, "A3"); residual.TerritoryID != "AAA" || residual.ChainID != nil {
		t.Errorf("retry residual = %+v, want retry army at AAA", residual)
	}
	if chain := second.State.Chains[0]; chain.PendingDisperse == nil || chain.PendingDisperse.ArmyID != "A3" {
		t.Errorf("second pending D chain = %#v, want A3 retry state", chain)
	}
	carrierDefeat := cloneGameState(second.State)
	attackerID := models.ArmyID("A4")
	attackerOwner := models.PlayerID("P2")
	carrierDefeat.Armies = append(carrierDefeat.Armies, models.Army{ID: attackerID, OwnerID: attackerOwner, TerritoryID: "EEE", Size: 2})
	attackerState := carrierDefeat.TerritoryStates["EEE"]
	attackerState.Army = &attackerID
	attackerState.OwnerID = &attackerOwner
	carrierDefeat.TerritoryStates["EEE"] = attackerState
	addInfrastructure(carrierDefeat, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "EEE"})
	carrierDefeat.NextArmyID = 5
	addNoble(carrierDefeat, "N4", "FOU", "P2", "EEE")
	addChain(t, carrierDefeat, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "EEE", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, carrierDefeat)
	cancelled, err := Resolve(carrierDefeat, testBalance())
	if err != nil {
		t.Fatalf("carrier defeat Resolve: %v", err)
	}
	if hasArmy(cancelled.State, "A1") || armyByID(t, cancelled.State, "A3").TerritoryID != "AAA" || len(cancelled.State.Chains) != 0 {
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
	openDestination := residualJoin.TerritoryStates["CCC"]
	openDestination.Army = nil
	residualJoin.TerritoryStates["CCC"] = openDestination
	joiningID := models.ArmyID("A4")
	joiningOwner := models.PlayerID("P1")
	residualJoin.Armies = append(residualJoin.Armies, models.Army{ID: joiningID, OwnerID: joiningOwner, TerritoryID: "EEE", Size: 1})
	joiningState := residualJoin.TerritoryStates["EEE"]
	joiningState.Army = &joiningID
	joiningState.OwnerID = &joiningOwner
	residualJoin.TerritoryStates["EEE"] = joiningState
	residualJoin.NextArmyID = 5
	addNoble(residualJoin, "N4", "FOU", "P1", "EEE")
	addChain(t, residualJoin, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "EEE", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, residualJoin)
	joined, err := Resolve(residualJoin, testBalance())
	if err != nil {
		t.Fatalf("pending residual J Resolve: %v", err)
	}
	if army := armyByID(t, joined.State, "A4"); army.TerritoryID != "AAA" {
		t.Errorf("A4 = %+v, want J arrival at AAA rather than D destination", army)
	}
	if noble := nobleByID(t, joined.State, "N4"); noble.LocationID != "AAA" {
		t.Errorf("N4 location = %q, want AAA with joining army", noble.LocationID)
	}
	retry := cloneGameState(second.State)
	remainingArmies := make([]models.Army, 0, len(retry.Armies)-1)
	for _, army := range retry.Armies {
		if army.ID != "A2" {
			remainingArmies = append(remainingArmies, army)
		}
	}
	retry.Armies = remainingArmies
	cleared := retry.TerritoryStates["CCC"]
	cleared.Army = nil
	retry.TerritoryStates["CCC"] = cleared
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
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "AAA", Liaison: models.LiaisonModeLoop})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "CCC" || army.ChainID != nil {
		t.Errorf("A1 = %+v, want retreat to CCC with broken invalid P loop", army)
	}
}

func TestResolvePillageCreditsNearestControlledSettlement(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	owner := models.PlayerID("P1")
	castleState := state.TerritoryStates["BBB"]
	castleState.OwnerID = &owner
	state.TerritoryStates["BBB"] = castleState
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypePillage, PositionID: "AAA"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolution.State.Infrastructures) != 1 || resolution.State.Infrastructures[0].ID != "I2" {
		t.Errorf("infrastructures = %#v, want only I2", resolution.State.Infrastructures)
	}
	wantResources := testBalance().PillageBonus + testBalance().BaseProduction + 1
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != wantResources {
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if hasArmy(resolution.State, "A1") {
			t.Error("A1 should be destroyed without a retreat")
		}
		if noble := nobleByID(t, resolution.State, "N1"); noble.Status != models.NobleStatusHostage || noble.LocationID != "AAA" {
			t.Errorf("N1 = %+v, want hostage at AAA", noble)
		}
		if !containsEvent(resolution.Events, EventTypeCapture) {
			t.Errorf("events = %#v, want capture event", resolution.Events)
		}
	})
}

func TestResolveLoopProgression(t *testing.T) {
	t.Run("hold", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{territory("AAA", "AAA")},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA", Liaison: models.LiaisonModeLoop})
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		setNobleStatus(state, "N1", models.NobleStatusHostage)
		setNobleStatus(state, "N3", models.NobleStatusHostage)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}, Liaison: models.LiaisonModeLoop})
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "CCC"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 territory = %q, want BBB after the supported attack", army.TerritoryID)
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		setNobleStatus(state, "N1", models.NobleStatusHostage)
		setNobleStatus(state, "N3", models.NobleStatusHostage)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "CCC"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
			t.Errorf("A1 territory = %q, want AAA after the repelled attack", army.TerritoryID)
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
				territory("AAA", "AAA", "BBB", "EEE", "DDD"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
				territory("DDD", "DDD", "AAA"),
				territory("EEE", "EEE", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
				{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 2},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addNoble(state, "N4", "FOU", "P2", "DDD")
		setNobleStatus(state, "N1", models.NobleStatusHostage)
		setNobleStatus(state, "N3", models.NobleStatusHostage)
		setNobleStatus(state, "N4", models.NobleStatusHostage)
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "DDD"})
		addChain(t, state, "A1", "N1", models.Order{ID: "A1O", Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"AAA"}})
		addChainOrders(t, state, "A3", "N3",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "CCC"},
		)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "EEE" {
			t.Errorf("A1 territory = %q, want EEE after retreat", army.TerritoryID)
		}
		if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "AAA" {
			t.Errorf("A4 territory = %q, want AAA after dislodging A1", army.TerritoryID)
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
					territory("AAA", "AAA", "BBB"),
					territory("BBB", "BBB", "AAA", "CCC"),
					territory("CCC", "CCC", "BBB"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
					{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
					{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", "P1", "AAA")
			addNoble(state, "N2", "TWO", "P2", "BBB")
			addNoble(state, "N3", "THR", "P3", "CCC")
			addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "BBB"})
			addChain(t, state, "A3", "N3", models.Order{ID: "A3O", Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"}})
			addChainOrders(t, state, "A1", "N1",
				models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop},
				models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "AAA"},
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
					territory("AAA", "AAA", "BBB"),
					territory("BBB", "BBB", "AAA"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
					{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", "P1", "AAA")
			addNoble(state, "N2", "TWO", "P2", "BBB")
			addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "BBB"})
			addChainOrders(t, state, "A1", "N1",
				models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop},
				models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "AAA"},
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC", "DDD"),
				territory("CCC", "CCC", "BBB"),
				territory("DDD", "DDD", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addNoble(state, "N4", "FOU", "P1", "CCC")
		addChainOrders(t, state, "A1", "N1",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "AAA"},
		)
		addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"DDD"}})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "DDD" {
			t.Errorf("A2 territory = %q, want DDD after moving away", army.TerritoryID)
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC", "DDD", "EEE"),
				territory("CCC", "CCC", "BBB"),
				territory("DDD", "DDD", "BBB", "EEE"),
				territory("EEE", "EEE", "BBB", "DDD"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 2},
				{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addNoble(state, "N4", "FOU", "P1", "DDD")
		addNoble(state, "N5", "FIV", "P1", "EEE")
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "DDD"})
		addChainOrders(t, state, "A1", "N1",
			models.Order{ID: "S1", Type: models.OrderTypeSupport, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Liaison: models.LiaisonModeLoop},
			models.Order{ID: "S2", Type: models.OrderTypeHold, PositionID: "AAA"},
		)
		addChain(t, state, "A2", "N2", models.Order{ID: "A2O", Type: models.OrderTypeHold, PositionID: "BBB"})
		addChain(t, state, "A4", "N4", models.Order{ID: "A4O", Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A5", "N5", models.Order{ID: "A5O", Type: models.OrderTypeSupport, PositionID: "EEE", TargetIDs: []models.TerritoryID{"DDD", "BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" {
			t.Errorf("A2 territory = %q, want CCC after retreat", army.TerritoryID)
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
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A10", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addChain(t, state, "A10", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"}})
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
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB", "DDD", "EEE"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "CCC"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}},
		models.Order{Type: models.OrderTypeHold, PositionID: "BBB"},
		models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"DDD"}},
		models.Order{Type: models.OrderTypePillage, PositionID: "BBB"},
	)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "BBB" {
		t.Fatalf("A1 territory = %q, want BBB after O1", army.TerritoryID)
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
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB", "DDD", "EEE"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "CCC"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "BBB")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"DDD"}, Liaison: models.LiaisonModeLoop},
		models.Order{Type: models.OrderTypePillage, PositionID: "BBB"},
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
