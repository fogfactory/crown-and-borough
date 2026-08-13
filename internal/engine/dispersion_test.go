package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveDisperseRepeatedDestinationStacksTroops(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T02"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{
			"BBB": {"ONE", "TWO"},
		},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("stacked army = %+v, want A1 size 2 at T02 without a chain", army)
	}
	if hasArmy(resolution.State, "A2") {
		t.Error("repeated destination should stack into one army")
	}
	for _, nobleID := range []models.NobleID{"N1", "N2"} {
		if noble := nobleByID(t, resolution.State, nobleID); noble.LocationID != "T02" {
			t.Errorf("%s location = %q, want T02", nobleID, noble.LocationID)
		}
	}
}

func TestResolveDisperseOccupiedDestinationDoesNotConsumeUnit(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"ONE"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("residual = %+v, want one unit left at T01 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T03" || army.Size != 1 {
		t.Errorf("branch = %+v, want one unit at T03", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T03" {
		t.Errorf("N1 location = %q, want T03", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveDisperseShortListLeavesUnitsAndUnassignedNobles(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 3}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02"},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("arrival = %+v, want A1 size 1 at T02 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("residual = %+v, want A2 size 2 at T01 without a chain", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T01" {
		t.Errorf("unassigned noble location = %q, want T01", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Outcome != OutcomeSuccess {
		t.Errorf("D outcome = %#v, want success", outcome)
	}
}

func TestResolveDisperseLongListSingleDropsExtraDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"ONE"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("arrival = %+v, want A1 at T02 without a chain", army)
	}
	if hasArmy(resolution.State, "A2") || resolution.State.TerritoryStates["T03"].Army != nil {
		t.Error("extra destination should not create an army")
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveDisperseLongListLoopBreaksWithoutMovement(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("army after invalid loop D = %+v, want stationary A1 without a chain", army)
	}
	if resolution.State.TerritoryStates["T02"].Army != nil || resolution.State.TerritoryStates["T03"].Army != nil {
		t.Error("invalid loop D should not move any branch")
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Outcome != OutcomeInvalid || outcome.Reason != "disperse_no_residual" {
		t.Errorf("D outcome = %#v, want invalid disperse_no_residual", outcome)
	}
}

func TestResolveDisperseUnassignedNobleInvalidatesAfterPreviousOrder(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChainOrders(t, state, "A1", "N1",
		models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}},
		models.Order{Type: models.OrderTypeDisperse, PositionID: "T02", TargetIDs: []models.TerritoryID{"T03"}},
	)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "T02" || army.ChainID == nil {
		t.Fatalf("after previous order = %+v, want A1 at T02 carrying its chain", army)
	}

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if army := armyByID(t, second.State, "A1"); army.TerritoryID != "T02" || army.ChainID != nil {
		t.Errorf("after invalid D = %+v, want A1 to remain at T02 without a chain", army)
	}
	if army := second.State.TerritoryStates["T03"].Army; army != nil {
		t.Errorf("invalid D created army %q at T03", *army)
	}
	if outcome, found := findOutcome(second.Events, "O2"); !found || outcome.Outcome != OutcomeInvalid || outcome.Reason != "disperse_noble_left_behind" {
		t.Errorf("D outcome = %#v, want invalid disperse_noble_left_behind", outcome)
	}
}

func TestResolveLoopDisperseRetriesRepeatedDestinationAndStacks(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T02", "T02"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "T01" || army.Size != 2 || army.ChainID == nil {
		t.Errorf("first residual = %+v, want A1 size 2 at T01 carrying the loop chain", army)
	}
	if chain := chainOf(first.State, "A1"); chain == nil || chain.PendingDisperse == nil || len(chain.PendingDisperse.TargetIDs) != 2 {
		t.Fatalf("first pending chain = %#v, want two repeated T02 targets", chain)
	}

	removeArmy(first.State, "A2")
	validateTestState(t, first.State)
	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if army := armyByID(t, second.State, "A1"); army.TerritoryID != "T02" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("completed residual = %+v, want stacked A1 size 2 at T02 without a chain", army)
	}
	if chain := chainOf(second.State, "A1"); chain != nil {
		t.Errorf("completed loop chain = %#v, want consumed chain", chain)
	}
	if noble := nobleByID(t, second.State, "N1"); noble.LocationID != "T02" {
		t.Errorf("N1 location = %q, want T02", noble.LocationID)
	}
}

func TestResolveLoopDispersePreservesWildcardInPendingState(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T02", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"BBB": {"*"}},
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
			territory("T01", "AAA", "T02", "T03", "T04"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01"),
			territory("T04", "DDD", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T04")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T03"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T04",
		TargetIDs:        []models.TerritoryID{"T01"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"AAA": {"TWO"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 2 || army.ChainID != nil {
		t.Errorf("invalid D carrier = %+v, want stationary A1 at T01", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T04" || army.ChainID != nil {
		t.Errorf("blocked arrival = %+v, want A2 to remain at T04", army)
	}
	if resolution.State.TerritoryStates["T01"].Army == nil || *resolution.State.TerritoryStates["T01"].Army != "A1" {
		t.Error("T01 should remain occupied by A1")
	}
}

func TestResolveInvalidDisperseDoesNotBlockJoinArrival(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03"),
			territory("T02", "BBB", "T01", "T04"),
			territory("T03", "CCC", "T01"),
			territory("T04", "DDD", "T02"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T04")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T03"},
		Liaison:    models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "T04",
		TargetIDs:  []models.TerritoryID{"T02"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.ChainID != nil {
		t.Errorf("invalid D carrier = %+v, want stationary A1 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" || army.ChainID != nil {
		t.Errorf("join arrival = %+v, want A2 at T02 without a chain", army)
	}
}

func TestResolveShortDisperseResidualBlocksArrivals(t *testing.T) {
	t.Run("dispersion", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "CCC", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T03")
		addChain(t, state, "A1", "N1", models.Order{
			Type:       models.OrderTypeDisperse,
			PositionID: "T01",
			TargetIDs:  []models.TerritoryID{"T02"},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "T03",
			TargetIDs:        []models.TerritoryID{"T01"},
			NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"AAA": {"TWO"}},
		})
		keepTestArmiesSupplied(state)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 {
			t.Errorf("arrival = %+v, want A1 size 1 at T02", army)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T01" || army.Size != 1 {
			t.Errorf("residual = %+v, want A3 size 1 at T01", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" {
			t.Errorf("blocked D = %+v, want A2 to remain at T03", army)
		}
	})

	t.Run("join", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01"),
				territory("T03", "CCC", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P1", "T03")
		addChain(t, state, "A1", "N1", models.Order{
			Type:       models.OrderTypeDisperse,
			PositionID: "T01",
			TargetIDs:  []models.TerritoryID{"T02"},
		})
		addChain(t, state, "A2", "N2", models.Order{
			Type:       models.OrderTypeJoin,
			PositionID: "T03",
			TargetIDs:  []models.TerritoryID{"T01"},
		})
		keepTestArmiesSupplied(state)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T01" || army.Size != 1 {
			t.Errorf("residual = %+v, want A3 size 1 at T01", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" || army.ChainID != nil {
			t.Errorf("blocked J = %+v, want A2 to remain at T03 without a chain", army)
		}
		if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_complete" {
			t.Errorf("D outcome = %#v, want disperse_complete", outcome)
		}
	})
}

func TestResolveDisperseSourceTargetKeepsSourceGroup(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T01", "T02"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{
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
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("source group = %+v, want A1 size 1 at T01 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" || army.Size != 1 {
		t.Errorf("outgoing group = %+v, want A2 size 1 at T02", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T01" {
		t.Errorf("N1 location = %q, want T01", noble.LocationID)
	}
	if noble := nobleByID(t, resolution.State, "N2"); noble.LocationID != "T02" {
		t.Errorf("N2 location = %q, want T02", noble.LocationID)
	}
}

func TestResolveConflictingDispersesDoNotEnterVacatedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T03"),
			territory("T02", "BBB", "T03"),
			territory("T03", "CCC", "T01", "T02", "T04"),
			territory("T04", "DDD", "T03"),
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
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T02",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "T03",
		TargetIDs:  []models.TerritoryID{"T04"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" {
		t.Errorf("first claimant = %+v, want A1 at T01", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" {
		t.Errorf("second claimant = %+v, want A2 at T02", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T04" {
		t.Errorf("vacating army = %+v, want A3 at T04", army)
	}
	if resolution.State.TerritoryStates["T03"].Army != nil {
		t.Error("conflicting disperses should leave the vacated T03 empty")
	}
}

func TestResolveJoinArrivalBlocksDisperseIntoVacatedDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T03"),
			territory("T02", "BBB", "T03"),
			territory("T03", "CCC", "T01", "T02", "T04"),
			territory("T04", "DDD", "T03"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T02")
	addNoble(state, "N3", "THR", "P1", "T03")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T01",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"ONE"}},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "T02",
		TargetIDs:  []models.TerritoryID{"T03"},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "T03",
		TargetIDs:  []models.TerritoryID{"T04"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.ChainID != nil {
		t.Errorf("blocked D = %+v, want A1 at T01 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" || army.ChainID != nil {
		t.Errorf("join arrival = %+v, want A2 at T03 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T04" || army.ChainID != nil {
		t.Errorf("outgoing join = %+v, want A3 at T04 without a chain", army)
	}
}

func TestResolveLateVacatedDestinationCannotOrphanNoble(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02", "T03", "T05"),
			territory("T02", "BBB", "T01"),
			territory("T03", "CCC", "T01", "T04"),
			territory("T04", "DDD", "T03"),
			territory("T05", "EEE", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T05", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T03")
	addNoble(state, "N3", "THR", "P1", "T05")
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T02", "T03"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T03",
		TargetIDs:        []models.TerritoryID{"T04"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"DDD": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T05",
		TargetIDs:        []models.TerritoryID{"T01"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"AAA": {"THR"}},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" || army.Size != 1 || army.ChainID != nil {
		t.Errorf("first branch = %+v, want A1 size 1 at T02", army)
	}
	if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "T01" || army.Size != 1 {
		t.Errorf("protected residual = %+v, want A4 size 1 at T01", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T04" {
		t.Errorf("vacating army = %+v, want A2 at T04", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T05" || army.ChainID != nil {
		t.Errorf("blocked origin arrival = %+v, want A3 at T05", army)
	}
	if noble := nobleByID(t, resolution.State, "N1"); noble.LocationID != "T01" {
		t.Errorf("unassigned noble location = %q, want T01", noble.LocationID)
	}
	if outcome, found := findOutcome(resolution.Events, "O1"); !found || outcome.Reason != "disperse_partial" {
		t.Errorf("A1 D outcome = %#v, want disperse_partial", outcome)
	}
}

func TestResolveIntrinsicallyInvalidLoopDisperseDoesNotClaimDestination(t *testing.T) {
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
		TargetIDs:        []models.TerritoryID{"T03", "T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T02",
		TargetIDs:        []models.TerritoryID{"T03"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"CCC": {"TWO"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.ChainID != nil {
		t.Errorf("invalid loop carrier = %+v, want A1 at T01 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T03" || army.ChainID != nil {
		t.Errorf("valid competing D = %+v, want A2 at T03 without a chain", army)
	}
}

func TestResolveDislodgedDisperseDoesNotBlockVacatedArrival(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "AAA", "T03", "T05", "T07"),
			territory("T02", "BBB", "T05"),
			territory("T03", "CCC", "T01"),
			territory("T05", "EEE", "T01", "T02", "T06"),
			territory("T06", "FFF", "T05"),
			territory("T07", "GGG", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "T05", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "T03", Size: 2},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P1", "T02")
	addNoble(state, "N3", "THR", "P2", "T05")
	addNoble(state, "N4", "FOU", "P2", "T03")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T03"})
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeDisperse,
		PositionID: "T01",
		TargetIDs:  []models.TerritoryID{"T05"},
	})
	addChain(t, state, "A2", "N2", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "T02",
		TargetIDs:        []models.TerritoryID{"T05"},
		NobleAssignments: map[models.TerritoryCode][]models.NobleCode{"EEE": {"TWO"}},
	})
	addChain(t, state, "A3", "N3", models.Order{
		Type:       models.OrderTypeJoin,
		PositionID: "T05",
		TargetIDs:  []models.TerritoryID{"T06"},
	})
	addChain(t, state, "A4", "N4", models.Order{
		Type:       models.OrderTypeAttack,
		PositionID: "T03",
		TargetIDs:  []models.TerritoryID{"T01"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T05" || army.ChainID != nil {
		t.Errorf("vacated arrival = %+v, want A2 at T05 without a chain", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "T06" {
		t.Errorf("vacating join = %+v, want A3 at T06", army)
	}
	if army := armyByID(t, resolution.State, "A4"); army.TerritoryID != "T01" {
		t.Errorf("attacker = %+v, want A4 at T01", army)
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
