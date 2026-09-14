package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveJoinFusesWithAllyAttackAfterEnemyAttackFails(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "YYY"),
			territory("BBB", "BBB", "YYY"),
			territory("CCC", "CCC", "YYY"),
			territory("YYY", "YYY", "AAA", "BBB", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
			{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P2", "CCC")
	addChain(t, state, "A1", "N1", models.Order{
		ID:         "J1",
		Type:       models.OrderTypeJoin,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"YYY"},
	})
	addChainOrders(t, state, "A2", "N2",
		models.Order{
			ID:         "A1",
			Type:       models.OrderTypeAttack,
			PositionID: "BBB",
			TargetIDs:  []models.TerritoryID{"YYY"},
		},
		models.Order{ID: "H1", Type: models.OrderTypeHold, PositionID: "YYY"},
	)
	addChain(t, state, "A3", "N3", models.Order{
		ID:         "E1",
		Type:       models.OrderTypeAttack,
		PositionID: "CCC",
		TargetIDs:  []models.TerritoryID{"YYY"},
	})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasArmy(resolution.State, "A1") {
		t.Fatal("joining army should fuse into the allied attack winner")
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "YYY" || army.Size != 3 || army.ChainID == nil {
		t.Errorf("winning army = %+v, want size 3 at YYY with its chain", army)
	}
	if chain := chainOf(resolution.State, "A2"); chain == nil || chain.CurrentIndex != 1 {
		t.Errorf("winning army chain = %#v, want to continue at the hold order", chain)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "CCC" {
		t.Errorf("losing attacker = %+v, want to remain at CCC", army)
	}
	if outcome, found := findOutcome(resolution.Events, "J1"); !found || outcome.Reason != "join_attack_arrival" || outcome.Outcome != OutcomeSuccess {
		t.Errorf("join outcome = %#v, found=%t, want successful join_attack_arrival", outcome, found)
	}
	if outcome, found := findOutcome(resolution.Events, "J1"); !found || outcome.Progression != ProgressionConsumed {
		t.Errorf("join progression = %#v, found=%t, want consumed", outcome, found)
	}
	if outcome, found := findOutcome(resolution.Events, "A1"); !found || outcome.Reason != "attack_wins" || outcome.Outcome != OutcomeSuccess {
		t.Errorf("winning attack outcome = %#v, found=%t, want attack_wins", outcome, found)
	}
	if outcome, found := findOutcome(resolution.Events, "E1"); !found || outcome.Reason != "combat_lost" || outcome.Outcome != OutcomeFailure {
		t.Errorf("losing attack outcome = %#v, found=%t, want combat_lost", outcome, found)
	}
}
