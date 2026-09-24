package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// executionScenario builds a supplied board where every army has a free noble
// of its owner, named after the army (A1 -> N1, code NOB).
func executionScenario(t *testing.T, territories []models.Territory, armies []models.Army) *models.GameState {
	t.Helper()
	state := testState(t, territories, armies)
	keepTestArmiesSupplied(state)
	for _, army := range armies {
		code := "NO" + string(rune('A'+army.ID[len(army.ID)-1]-'0'))
		addNoble(state, nobleOf(army.ID), code, army.OwnerID, army.TerritoryID)
	}
	return state
}

func nobleOf(armyID models.ArmyID) models.NobleID {
	return models.NobleID("N" + string(armyID[1:]))
}

func armiesAt(state *models.GameState, territoryID models.TerritoryID) []models.Army {
	var armies []models.Army
	for _, army := range state.Armies {
		if army.TerritoryID == territoryID {
			armies = append(armies, army)
		}
	}
	return armies
}

func resolveScenario(t *testing.T, state *models.GameState) Resolution {
	t.Helper()
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return resolution
}

func TestResolveDisperseChainMovesWithItsOrders(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{territory("TAA", "TAA", "TBB"), territory("TBB", "TBB", "TAA")},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "TBB", Size: 1},
		},
	)
	// A1 leaves entirely and its chain goes on with A2, which has none.
	addChainOrders(t, state, "A1", nobleOf("A1"),
		models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TBB": {"*"}}},
		models.Order{Type: models.OrderTypeHold, PositionID: "TBB"},
	)

	resolution := resolveScenario(t, state)
	host := armyByID(t, resolution.State, "A2")
	chain := chainOf(resolution.State, "A2")
	if host.Size != 2 || host.ChainID == nil || chain == nil || chain.CurrentIndex != 1 {
		t.Fatalf("A2 = %+v, chain = %+v, want size 2 carrying A1's chain on its hold", host, chain)
	}
	for _, order := range chain.Orders {
		if order.ArmyID != "A2" {
			t.Errorf("order %s references %s, want A2", order.ID, order.ArmyID)
		}
	}
}

func TestResolveCrossingDispersesKeepEveryTroop(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{territory("TAA", "TAA", "TBB"), territory("TBB", "TBB", "TAA")},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "TBB", Size: 2},
		},
	)
	// Each army sends one troop onto the other's residual; A1's troop stacks
	// onto A2 before A2's own dispersion is applied.
	addChain(t, state, "A1", nobleOf("A1"), models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", nobleOf("A2"), models.Order{Type: models.OrderTypeDisperse, PositionID: "TBB", TargetIDs: []models.TerritoryID{"TAA"}})

	resolution := resolveScenario(t, state)
	for _, territoryID := range []models.TerritoryID{"TAA", "TBB"} {
		armies := armiesAt(resolution.State, territoryID)
		if len(armies) != 1 || armies[0].Size != 2 {
			t.Errorf("%s armies = %+v, want one army of 2", territoryID, armies)
		}
	}
}

func TestResolveDisperseFusesWithJoinHost(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{
			territory("TAA", "TAA", "TGG"),
			territory("TDD", "TDD", "TGG"),
			territory("TGG", "TGG", "TAA", "TDD"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "TDD", Size: 1},
			{ID: "A7", OwnerID: "P1", TerritoryID: "TGG", Size: 1},
		},
	)
	// A4 joins the holding A7; A1's troop fuses with A7 as well, not with A4,
	// which fuses away.
	addChainOrders(t, state, "A1", nobleOf("A1"),
		models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TGG"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TGG": {"*"}}},
		models.Order{Type: models.OrderTypeHold, PositionID: "TGG"},
	)
	addChain(t, state, "A4", nobleOf("A4"), models.Order{Type: models.OrderTypeJoin, PositionID: "TDD", TargetIDs: []models.TerritoryID{"TGG"}})
	addChain(t, state, "A7", nobleOf("A7"), models.Order{Type: models.OrderTypeHold, PositionID: "TGG"})

	resolution := resolveScenario(t, state)
	if armies := armiesAt(resolution.State, "TGG"); len(armies) != 1 || armies[0].ID != "A7" || armies[0].Size != 3 {
		t.Errorf("TGG armies = %+v, want A7 of 3", armies)
	}
}

func TestResolveJoinPairKeepsInheritedDisperseChain(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{
			territory("TAA", "TAA", "TCC"),
			territory("TBB", "TBB", "TCC"),
			territory("TCC", "TCC", "TAA", "TBB", "TDD"),
			territory("TDD", "TDD", "TCC"),
		},
		[]models.Army{
			{ID: "A2", OwnerID: "P1", TerritoryID: "TBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "TCC", Size: 2},
			{ID: "A4", OwnerID: "P1", TerritoryID: "TDD", Size: 1},
		},
	)
	// The join pair arrives where A3 keeps a troop with its chain: the merged
	// army drops the joins' chains but carries A3's on.
	addChain(t, state, "A2", nobleOf("A2"), models.Order{Type: models.OrderTypeJoin, PositionID: "TBB", TargetIDs: []models.TerritoryID{"TCC"}})
	addChainOrders(t, state, "A3", nobleOf("A3"),
		models.Order{Type: models.OrderTypeDisperse, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TCC", "TAA"}},
		models.Order{Type: models.OrderTypeHold, PositionID: "TCC"},
	)
	addChain(t, state, "A4", nobleOf("A4"), models.Order{Type: models.OrderTypeJoin, PositionID: "TDD", TargetIDs: []models.TerritoryID{"TCC"}})

	resolution := resolveScenario(t, state)
	armies := armiesAt(resolution.State, "TCC")
	if len(armies) != 1 || armies[0].Size != 3 || armies[0].ChainID == nil {
		t.Fatalf("TCC armies = %+v, want one army of 3 carrying A3's chain", armies)
	}
	if chain := chainOf(resolution.State, armies[0].ID); chain == nil || chain.Orders[chain.CurrentIndex].Type != models.OrderTypeHold {
		t.Errorf("chain = %+v, want A3's chain on its hold", chain)
	}
}

func TestResolvePendingDisperseResidualKeepsItsChain(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{
			territory("TAA", "TAA", "TCC", "TDD", "TFF"),
			territory("TCC", "TCC", "TAA"),
			territory("TDD", "TDD", "TAA"),
			territory("TFF", "TFF", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 2},
			{ID: "A3", OwnerID: "P1", TerritoryID: "TCC", Size: 1},
			{ID: "A5", OwnerID: "P2", TerritoryID: "TFF", Size: 1},
		},
	)
	// A1's loop dispersion leaves a residual on TAA to retry TFF; A3's troop
	// stacks onto it, and its own chain gives way to the pending dispersion.
	addChain(t, state, "A1", nobleOf("A1"), models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TDD", "TFF"}, Liaison: models.LiaisonModeLoop})
	addChainOrders(t, state, "A3", nobleOf("A3"),
		models.Order{Type: models.OrderTypeDisperse, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TAA"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TAA": {"*"}}},
		models.Order{Type: models.OrderTypeHold, PositionID: "TAA"},
	)
	addChain(t, state, "A5", nobleOf("A5"), models.Order{Type: models.OrderTypeHold, PositionID: "TFF"})

	resolution := resolveScenario(t, state)
	residual := armiesAt(resolution.State, "TAA")
	if len(residual) != 1 || residual[0].Size != 2 || residual[0].ChainID != nil {
		t.Fatalf("TAA armies = %+v, want a chainless residual of 2", residual)
	}
	chain := chainOf(resolution.State, "A1")
	if chain == nil || chain.PendingDisperse == nil || chain.PendingDisperse.ArmyID != residual[0].ID {
		t.Errorf("A1 chain = %+v, want its dispersion pending on the residual", chain)
	}
}

func TestResolveJoinIgnoresDislodgedAllyAttackOnItsDestination(t *testing.T) {
	state := executionScenario(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "TDD"),
			territory("TBB", "TBB", "TAA", "TFF"),
			territory("TDD", "TDD", "TAA", "TEE"),
			territory("TEE", "TEE", "TDD"),
			territory("TFF", "TFF", "TBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TBB", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "TDD", Size: 1},
			{ID: "A5", OwnerID: "P3", TerritoryID: "TEE", Size: 3},
			{ID: "A6", OwnerID: "P3", TerritoryID: "TFF", Size: 1},
		},
	)
	// A4 only attacks its ally A1, which stays, and is dislodged: TAA is not
	// contested, A2 joins A1 and A6 enters the TBB it left. A4 then retreats
	// onto A1 as well.
	addChain(t, state, "A1", nobleOf("A1"), models.Order{Type: models.OrderTypeHold, PositionID: "TAA"})
	addChain(t, state, "A2", nobleOf("A2"), models.Order{Type: models.OrderTypeJoin, PositionID: "TBB", TargetIDs: []models.TerritoryID{"TAA"}})
	addChain(t, state, "A4", nobleOf("A4"), models.Order{Type: models.OrderTypeAttack, PositionID: "TDD", TargetIDs: []models.TerritoryID{"TAA"}})
	addChain(t, state, "A5", nobleOf("A5"), models.Order{Type: models.OrderTypeAttack, PositionID: "TEE", TargetIDs: []models.TerritoryID{"TDD"}})
	addChain(t, state, "A6", nobleOf("A6"), models.Order{Type: models.OrderTypeAttack, PositionID: "TFF", TargetIDs: []models.TerritoryID{"TBB"}})

	resolution := resolveScenario(t, state)
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TAA" || army.Size != 3 {
		t.Errorf("A1 = %+v, want A2 and the retreating A4 fused into it at TAA", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "join_host" {
		t.Errorf("A2 outcome = %#v, found=%t, want join_host", event, found)
	}
	if army := armyByID(t, resolution.State, "A6"); army.TerritoryID != "TBB" {
		t.Errorf("A6 = %+v, want TBB", army)
	}
}
