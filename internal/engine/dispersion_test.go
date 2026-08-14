package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveDisperseRepeatedDestinationStacksTroops(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{
			"BBB": {"ONE", "TWO"},
		},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("stacked army = %+v, want A1 size 2 at BBB without a chain", army)
	}
	if hasArmy(resolution.State, "A2") {
		t.Error("repeated destination should stack into one army")
	}
	for _, nobleID := range []models.NobleID{"N1", "N2"} {
		if noble := nobleByID(t, resolution.State, nobleID); noble.LocationID != "BBB" {
			t.Errorf("%s location = %q, want BBB", nobleID, noble.LocationID)
		}
	}
}

func TestResolveDisperseOccupiedDestinationDoesNotConsumeUnit(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"ONE"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("residual = %+v, want one unit left at AAA without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "CCC" || army.Size != 1 {
		t.Errorf("branch = %+v, want one unit at CCC", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "CCC" {
		t.Errorf("N1 location = %q, want CCC", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveDisperseShortListLeavesUnitsAndUnassignedNobles(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("arrival = %+v, want A1 size 1 at BBB without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("residual = %+v, want A2 size 2 at AAA without a chain", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "AAA" {
		t.Errorf("unassigned noble location = %q, want AAA", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Outcome != OutcomeSuccess {
		t.Errorf("D outcome = %#v, want success", outcome)
	}
}

func TestResolveDisperseLongListSingleDropsExtraDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("arrival = %+v, want A1 at BBB without a chain", army)
	}
	if hasArmy(resolution.State, "A2") || resolution.State.TerritoryStates["CCC"].Army != nil {
		t.Error("extra destination should not create an army")
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveDisperseLongListLoopBreaksWithoutMovement(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("army after invalid loop D = %+v, want stationary A1 without a chain", army)
	}
	if resolution.State.TerritoryStates["BBB"].Army != nil || resolution.State.TerritoryStates["CCC"].Army != nil {
		t.Error("invalid loop D should not move any branch")
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Outcome != OutcomeInvalid || outcome.Reason != "disperse_no_residual" {
		t.Errorf("D outcome = %#v, want invalid disperse_no_residual", outcome)
	}
}

func TestResolveDisperseUnassignedNobleInvalidatesAfterPreviousOrder(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}},
		models.Order{Type: models.OrderTypeDisperse, PositionID: "BBB", TargetIDs: []models.TerritoryID{"CCC"}},
	)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "BBB" || army.ChainID == nil {
		t.Fatalf("after previous order = %+v, want A1 at BBB carrying its chain", army)
	}

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if army := armyByID(t, second.State, "A1"); army.TerritoryID != "BBB" || army.ChainID != nil {
		t.Errorf("after invalid D = %+v, want A1 to remain at BBB without a chain", army)
	}
	if army := second.State.TerritoryStates["CCC"].Army; army != nil {
		t.Errorf("invalid D created army %q at CCC", *army)
	}
	if outcome, found := findOutcome(second.Events, "O2"); !found || outcome.Outcome != OutcomeInvalid || outcome.Reason != "disperse_noble_left_behind" {
		t.Errorf("D outcome = %#v, want invalid disperse_noble_left_behind", outcome)
	}
}

func TestResolveLoopDisperseRetriesRepeatedDestinationAndStacks(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 || army.ChainID == nil {
		t.Errorf("first residual = %+v, want A1 size 2 at AAA carrying the loop chain", army)
	}
	if chain := chainOf(first.State, "A1"); chain == nil || chain.PendingDisperse == nil || len(chain.PendingDisperse.TargetIDs) != 2 {
		t.Fatalf("first pending chain = %#v, want two repeated BBB targets", chain)
	}

	removeArmy(first.State, "A2")
	validateTestState(t, first.State)
	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if army := armyByID(t, second.State, "A1"); army.TerritoryID != "BBB" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("completed residual = %+v, want stacked A1 size 2 at BBB without a chain", army)
	}
	if chain := chainOf(second.State, "A1"); chain != nil {
		t.Errorf("completed loop chain = %#v, want consumed chain", chain)
	}
	if noble := nobleByID(t, second.State, "N1"); noble.LocationID != "BBB" {
		t.Errorf("N1 location = %q, want BBB", noble.LocationID)
	}
}

func TestResolveLoopDispersePreservesWildcardInPendingState(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"*"}},
		Liaison:          models.LiaisonModeLoop,
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	chain := chainOf(resolution.State, "A1")
	if chain == nil || chain.PendingDisperse == nil {
		t.Fatalf("pending chain = %#v, want a pending loop dispersion", chain)
	}
	if got := chain.PendingDisperse.NobleAssignments["BBB"]; len(got) != 1 || got[0] != "*" {
		t.Errorf("pending wildcard = %#v, want [*]", got)
	}
}

func TestResolveInvalidDisperseDoesNotVacateOrigin(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
			territory("DDD", "DDD", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "DDD")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "DDD",
		TargetIDs:        []models.TerritoryID{"AAA"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"AAA": {"TWO"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("invalid D carrier = %+v, want stationary A1 at AAA", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "DDD" || army.ChainID != nil {
		t.Errorf("blocked arrival = %+v, want A2 to remain at DDD", army)
	}
	if resolution.State.TerritoryStates["AAA"].Army == nil || *resolution.State.TerritoryStates["AAA"].Army != "A1" {
		t.Error("AAA should remain occupied by A1")
	}
}

func TestResolveInvalidDisperseDoesNotBlockJoinArrival(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA", "DDD"),
			territory("CCC", "CCC", "AAA"),
			territory("DDD", "DDD", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "DDD")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
		Liaison:    models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "DDD",
		TargetIDs:  []models.TerritoryID{"BBB"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.ChainID != nil {
		t.Errorf("invalid D carrier = %+v, want stationary A1 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" || army.ChainID != nil {
		t.Errorf("join arrival = %+v, want A2 at BBB without a chain", army)
	}
}

func TestResolveShortDisperseResidualBlocksArrivals(t *testing.T) {
	t.Run("dispersion", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA"),
				territory("CCC", "CCC", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{
			Type:       models.OrderTypeDisperse,
			PositionID: "AAA",
			TargetIDs:  []models.TerritoryID{"BBB"},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "CCC",
			TargetIDs:        []models.TerritoryID{"AAA"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"AAA": {"TWO"}},
		})
		keepTestArmiesSupplied(state)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 {
			t.Errorf("arrival = %+v, want A1 size 1 at BBB", army)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "AAA" || army.Size != 1 {
			t.Errorf("residual = %+v, want A3 size 1 at AAA", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" {
			t.Errorf("blocked D = %+v, want A2 to remain at CCC", army)
		}
	})

	t.Run("join", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA"),
				territory("CCC", "CCC", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{
			Type:       models.OrderTypeDisperse,
			PositionID: "AAA",
			TargetIDs:  []models.TerritoryID{"BBB"},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:       models.OrderTypeJoin,
			PositionID: "CCC",
			TargetIDs:  []models.TerritoryID{"AAA"},
		})
		keepTestArmiesSupplied(state)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "AAA" || army.Size != 1 {
			t.Errorf("residual = %+v, want A3 size 1 at AAA", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" || army.ChainID != nil {
			t.Errorf("blocked J = %+v, want A2 to remain at CCC without a chain", army)
		}
		if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_complete" {
			t.Errorf("D outcome = %#v, want disperse_complete", outcome)
		}
	})
}

func TestResolveDisperseSourceTargetKeepsSourceGroup(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"AAA", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{
			"AAA": {"ONE"},
			"BBB": {"TWO"},
		},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("source group = %+v, want A1 size 1 at AAA without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" || army.Size != 1 {
		t.Errorf("outgoing group = %+v, want A2 size 1 at BBB", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "AAA" {
		t.Errorf("N1 location = %q, want AAA", noble.LocationID)
	}
	if noble := nobleByID(t, resolution.State, "N2"); noble.LocationID != "BBB" {
		t.Errorf("N2 location = %q, want BBB", noble.LocationID)
	}
}

func TestResolveConflictingDispersesDoNotEnterVacatedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "CCC"),
			territory("BBB", "BBB", "CCC"),
			territory("CCC", "CCC", "AAA", "BBB", "DDD"),
			territory("DDD", "DDD", "CCC"),
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
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "BBB",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "CCC",
		TargetIDs:  []models.TerritoryID{"DDD"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Errorf("first claimant = %+v, want A1 at AAA", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("second claimant = %+v, want A2 at BBB", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "DDD" {
		t.Errorf("vacating army = %+v, want A3 at DDD", army)
	}
	if resolution.State.TerritoryStates["CCC"].Army != nil {
		t.Error("conflicting disperses should leave the vacated CCC empty")
	}
}

func TestResolveJoinArrivalBlocksDisperseIntoVacatedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "CCC"),
			territory("BBB", "BBB", "CCC"),
			territory("CCC", "CCC", "AAA", "BBB", "DDD"),
			territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "BBB",
		TargetIDs:  []models.TerritoryID{"CCC"},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "CCC",
		TargetIDs:  []models.TerritoryID{"DDD"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.ChainID != nil {
		t.Errorf("blocked D = %+v, want A1 at AAA without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" || army.ChainID != nil {
		t.Errorf("join arrival = %+v, want A2 at CCC without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "DDD" || army.ChainID != nil {
		t.Errorf("outgoing join = %+v, want A3 at DDD without a chain", army)
	}
}

func TestResolveLateVacatedDestinationCannotOrphanNoble(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "EEE"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA", "DDD"),
			territory("DDD", "DDD", "CCC"),
			territory("EEE", "EEE", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "CCC")
	addNoble(state, "N3", "THR", "P1", "EEE")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB", "CCC"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "CCC",
		TargetIDs:        []models.TerritoryID{"DDD"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"DDD": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "EEE",
		TargetIDs:        []models.TerritoryID{"AAA"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"AAA": {"THR"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("first branch = %+v, want A1 size 1 at BBB", army)
	}
	if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "AAA" || army.Size != 1 {
		t.Errorf("protected residual = %+v, want A4 size 1 at AAA", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "DDD" {
		t.Errorf("vacating army = %+v, want A2 at DDD", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "EEE" || army.ChainID != nil {
		t.Errorf("blocked origin arrival = %+v, want A3 at EEE", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "AAA" {
		t.Errorf("unassigned noble location = %q, want AAA", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("A1 D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveIntrinsicallyInvalidLoopDisperseDoesNotClaimDestination(t *testing.T) {
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
		TargetIDs:        []models.TerritoryID{"CCC", "CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "BBB",
		TargetIDs:        []models.TerritoryID{"CCC"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"CCC": {"TWO"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.ChainID != nil {
		t.Errorf("invalid loop carrier = %+v, want A1 at AAA without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "CCC" || army.ChainID != nil {
		t.Errorf("valid competing D = %+v, want A2 at CCC without a chain", army)
	}
}

func TestResolveDislodgedDisperseDoesNotBlockVacatedArrival(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "CCC", "EEE", "GGG"),
			territory("BBB", "BBB", "EEE"),
			territory("CCC", "CCC", "AAA"),
			territory("EEE", "EEE", "AAA", "BBB", "FFF"),
			territory("FFF", "FFF", "EEE"),
			territory("GGG", "GGG", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "EEE", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "CCC", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P2", "EEE")
	addNoble(state, "N4", "FOU", "P2", "CCC")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"EEE"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "BBB",
		TargetIDs:        []models.TerritoryID{"EEE"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"EEE": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "EEE",
		TargetIDs:  []models.TerritoryID{"FFF"},
	})
	addChain(t, state, "A4", "N4", models.Order{
		Type:       models.OrderTypeAttack,
		PositionID: "CCC",
		TargetIDs:  []models.TerritoryID{"AAA"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "EEE" || army.ChainID != nil {
		t.Errorf("vacated arrival = %+v, want A2 at EEE without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "FFF" {
		t.Errorf("vacating join = %+v, want A3 at FFF", army)
	}
	if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "AAA" {
		t.Errorf("attacker = %+v, want A4 at AAA", army)
	}
}

func removeArmy(state *models.GameState, armyID models.ArmyID) {
	armies := make([]models.Army, 0, len(state.Armies)-1)
	for _, army := range state.Armies {
		if army.ID == armyID {
			territoryState := state.TerritoryStates[army.TerritoryID]
			territoryState.Army = nil
			state.TerritoryStates[army.TerritoryID] = territoryState
			continue
		}
		armies = append(armies, army)
	}
	state.Armies = armies
}
