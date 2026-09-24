package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// These tests cover the general rule that closes the combat-adjudication
// paradox family (issue #180): a join or a dispersion whose origin is under
// attack, whoever the attacker is, is cancelled outright before the
// adjudicator even runs. TestResolveJoinCrossesAttack,
// TestResolveJoinDepartureVacatesOriginBeforeAttack and
// TestResolveCancelsPeacefulCrossingAttackedOrigins already exercise the
// enemy and cyclic cases; these add the allied and partial-dispersion
// families, and the boundary cases around dislodgement and starvation.

func TestResolveAlliedAttackCancelsDispersionOrigin(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "TCC"),
			territory("TBB", "TBB", "TAA"),
			territory("TCC", "TCC", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "TCC", Size: 1},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P1", "TCC")
	// A2 attacks its own ally's territory TAA: the attack itself fails as
	// allied_destination, but it still targets TAA, so A1's dispersion out of
	// it is cancelled regardless of who attacks or whether the attack wins.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TBB": {"ONE"}}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TAA" {
		t.Errorf("A1 = %+v, want TAA, its dispersion cancelled", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "attacked_origin" {
		t.Errorf("A1 outcome = %#v, found=%t, want attacked_origin", event, found)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "allied_destination" {
		t.Errorf("A2 outcome = %#v, found=%t, want allied_destination", event, found)
	}
}

func TestResolveEnemyAttackCancelsPartialDispersionOrigin(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "TCC", "TDD"),
			territory("TBB", "TBB", "TAA"),
			territory("TCC", "TCC", "TAA"),
			territory("TDD", "TDD", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TDD", Size: 1},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P2", "TDD")
	// A1's dispersion to two different targets is a "partial" order in shape,
	// but its origin TAA is attacked, so it is cancelled just the same: none
	// of its troops leaves for either target.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB", "TCC"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TBB": {"ONE"}}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TDD", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TAA" || army.Size != 2 {
		t.Errorf("A1 = %+v, want its full 2 troops at TAA, its dispersion cancelled", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "attacked_origin" {
		t.Errorf("A1 outcome = %#v, found=%t, want attacked_origin", event, found)
	}
	for _, territoryID := range []models.TerritoryID{"TBB", "TCC"} {
		if state := resolution.State.TerritoryStates[territoryID]; state.Army != nil {
			t.Errorf("%s army = %v, want no troop left behind by the cancelled dispersion", territoryID, *state.Army)
		}
	}
}

func TestResolveEnemyAttackCancelsJoinOriginOutsideAnyCycle(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "TCC"),
			territory("TBB", "TBB", "TAA"),
			territory("TCC", "TCC", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TCC", Size: 1},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P2", "TCC")
	// A1's join target TBB has nothing to do with A2's attack, so the two
	// orders never form a cycle: the cancellation is a plain, direct
	// consequence of TAA being attacked.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TAA" {
		t.Errorf("A1 = %+v, want TAA, its join cancelled", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "attacked_origin" {
		t.Errorf("A1 outcome = %#v, found=%t, want attacked_origin", event, found)
	}
}

// TestCancelAttackedOriginPeacefulCountsStarvingAttack exercises the
// enumeration rule directly: an attack at strength zero, because its army is
// famished, still targets its destination and therefore still cancels a
// join or dispersion leaving from there.
func TestCancelAttackedOriginPeacefulCountsStarvingAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "TCC"),
			territory("TBB", "TBB", "TAA"),
			territory("TCC", "TCC", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P2", "TCC")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	ctx := newResolutionContext(state, testBalance())
	// A2 is famished before its order is enumerated, so its attack is built
	// with strength zero, the same as resolveSeasonEffects would leave it.
	ctx.famished["A2"] = true
	enumerateIntentions(ctx)
	if attack := ctx.attacks["A2"]; attack == nil || attack.size != 0 || attack.target != "TAA" {
		t.Fatalf("A2 attack = %+v, want a starving attack on TAA", attack)
	}
	cancelAttackedOriginPeaceful(ctx)

	if !ctx.cancelledPeaceful["A1"] {
		t.Error("A1's join should be cancelled by the starving attack on its origin")
	}
	if _, exists := ctx.joins["A1"]; exists {
		t.Error("A1's join should be removed from ctx.joins once cancelled")
	}
}

// TestResolveLoopDispersePendingCancelledRetriesNextChainStep exercises a
// pending loop dispersion whose residual origin comes under attack, without
// being dislodged: it is cancelled with reason attacked_origin, its
// PendingDisperse is cleared rather than left stale, and the loop retries the
// same order next chain step.
func TestResolveLoopDispersePendingCancelledRetriesNextChainStep(t *testing.T) {
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
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "BBB"})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	if army := armyByID(t, first.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 {
		t.Fatalf("first residual = %+v, want A1 size 2 at AAA, blocked by A2", army)
	}
	chain := chainOf(first.State, "A1")
	if chain == nil || chain.PendingDisperse == nil {
		t.Fatalf("first chain = %#v, want a pending loop dispersion", chain)
	}

	// A third player now attacks the residual's origin AAA, weakly enough
	// that it does not dislodge it.
	first.State.Armies = append(first.State.Armies, models.Army{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 2})
	first.State.NextArmyID = 4
	territoryState := first.State.TerritoryStates["CCC"]
	army3 := models.ArmyID("A3")
	owner3 := models.PlayerID("P3")
	territoryState.Army = &army3
	territoryState.OwnerID = &owner3
	first.State.TerritoryStates["CCC"] = territoryState
	addNoble(first.State, "N3", "THR", "P3", "CCC")
	addChain(t, first.State, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	keepTestArmiesSupplied(first.State)
	validateTestState(t, first.State)

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if army := armyByID(t, second.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 {
		t.Errorf("A1 = %+v, want it to stay at AAA, cancelled rather than dislodged", army)
	}
	if event, found := outcomeForArmy(second.Events, "A1"); !found || event.Reason != "attacked_origin" {
		t.Errorf("A1 outcome = %#v, found=%t, want attacked_origin", event, found)
	}
	if chain := chainOf(second.State, "A1"); chain == nil || chain.PendingDisperse != nil || chain.CurrentIndex != 0 {
		t.Errorf("A1 chain = %#v, want its pending dispersion cleared and the loop retrying the same order", chain)
	}
}

// TestResolveLoopDispersePendingCancelledAndDislodgedReportsDislodged is the
// same setup, but the attack is strong enough to dislodge the residual: the
// outcome must report dislodged, not attacked_origin, and PendingDisperse
// must still be cleared rather than left stale.
func TestResolveLoopDispersePendingCancelledAndDislodgedReportsDislodged(t *testing.T) {
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
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addChain(t, state, "A1", "N1", models.Order{
		Type:             models.OrderTypeDisperse,
		PositionID:       "AAA",
		TargetIDs:        []models.TerritoryID{"BBB", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"ONE"}},
		Liaison:          models.LiaisonModeLoop,
	})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "BBB"})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	chain := chainOf(first.State, "A1")
	if chain == nil || chain.PendingDisperse == nil {
		t.Fatalf("first chain = %#v, want a pending loop dispersion", chain)
	}

	// This time the third player's attack is strong enough to dislodge A1.
	first.State.Armies = append(first.State.Armies, models.Army{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 4})
	first.State.NextArmyID = 4
	territoryState := first.State.TerritoryStates["CCC"]
	army3 := models.ArmyID("A3")
	owner3 := models.PlayerID("P3")
	territoryState.Army = &army3
	territoryState.OwnerID = &owner3
	first.State.TerritoryStates["CCC"] = territoryState
	addNoble(first.State, "N3", "THR", "P3", "CCC")
	addChain(t, first.State, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	keepTestArmiesSupplied(first.State)
	validateTestState(t, first.State)

	second, err := Resolve(first.State, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if event, found := outcomeForArmy(second.Events, "A1"); !found || event.Reason != "dislodged" {
		t.Errorf("A1 outcome = %#v, found=%t, want dislodged", event, found)
	}
	if chain := chainOf(second.State, "A1"); chain != nil && chain.PendingDisperse != nil {
		t.Errorf("A1 chain = %#v, want its pending dispersion cleared, not left stale", chain)
	}
}
