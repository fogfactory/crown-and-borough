package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveAttackEntersDestinationFreedByHeadToHeadWinner(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TCC", "TCC", "TDD", "TFF"),
			territory("TDD", "TDD", "TCC"),
			territory("TFF", "TFF", "TCC", "TGG"),
			territory("TGG", "TGG", "TFF"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "TCC", Size: 2},
			{ID: "A2", OwnerID: "P3", TerritoryID: "TDD", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "TFF", Size: 1},
			{ID: "A4", OwnerID: "P3", TerritoryID: "TGG", Size: 2},
		},
	)
	keepTestArmiesSupplied(state)
	for _, noble := range []struct {
		id        models.NobleID
		code      string
		owner     models.PlayerID
		territory models.TerritoryID
	}{
		{"N1", "ONE", "P2", "TCC"}, {"N2", "TWO", "P3", "TDD"}, {"N3", "THR", "P1", "TFF"}, {"N4", "FOU", "P3", "TGG"},
	} {
		addNoble(state, noble.id, noble.code, noble.owner, noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TDD"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TDD", TargetIDs: []models.TerritoryID{"TCC"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "TFF", TargetIDs: []models.TerritoryID{"TCC"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "TGG", TargetIDs: []models.TerritoryID{"TFF"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// A1 wins the head-to-head and dislodges A2, whose attack then no longer
	// contests TCC: A3 enters it, and A4 enters the TFF it left.
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "TDD"}, {"A3", "TCC"}, {"A4", "TFF"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s = %+v, want %s", want.armyID, army, want.territoryID)
		}
	}
	if event, found := outcomeForArmy(resolution.Events, "A3"); !found || event.Reason != "attack_wins" {
		t.Errorf("A3 outcome = %#v, found=%t, want attack_wins", event, found)
	}
	if event, found := combatAt(resolution.Events, "TCC"); !found || event.WinnerArmyID != "A3" || hasContender(event, "A2") {
		t.Errorf("TCC contest = %#v, found=%t, want A3 winning without the retired A2", event, found)
	}
}

func TestResolveCancelsPeacefulCrossingAttackedOrigins(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB", "THH"),
			territory("TBB", "TBB", "TAA"),
			territory("THH", "THH", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "TAA", Size: 2},
			{ID: "A2", OwnerID: "P3", TerritoryID: "TBB", Size: 2},
			{ID: "A3", OwnerID: "P2", TerritoryID: "THH", Size: 2},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P2", "TAA")
	addNoble(state, "N2", "TWO", "P3", "TBB")
	addNoble(state, "N3", "THR", "P2", "THH")
	// TAA is under attack from A3, so A1's join out of it is cancelled
	// outright and it stays. A2's join is not cancelled, since TBB is not
	// attacked, but TAA is now held by A1, an enemy to A2, so it is turned
	// back. A3 only attacks its ally A1, which stays, so its own attack is
	// deferred as allied_destination.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "TBB", TargetIDs: []models.TerritoryID{"TAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "THH", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
		reason      string
	}{
		{"A1", "TAA", "attacked_origin"},
		{"A2", "TBB", "enemy_destination"},
		{"A3", "THH", "allied_destination"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s = %+v, want %s", want.armyID, army, want.territoryID)
		}
		if event, found := outcomeForArmy(resolution.Events, want.armyID); !found || event.Reason != want.reason {
			t.Errorf("%s outcome = %#v, found=%t, want %s", want.armyID, event, found, want.reason)
		}
	}
}

func TestResolveTiedHelpAgainstOwnArmyIsApplied(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TFF", "TGG"),
			territory("TFF", "TFF", "TAA", "TGG"),
			territory("TGG", "TGG", "TAA", "TFF"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P3", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "TFF", Size: 1},
			{ID: "A3", OwnerID: "P3", TerritoryID: "TGG", Size: 2},
		},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "TGG"})
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P3", "TAA")
	addNoble(state, "N2", "TWO", "P1", "TFF")
	addNoble(state, "N3", "THR", "P3", "TGG")
	// A1 helps A2 against its own owner's army A3, which attacks A1, its ally,
	// and therefore stays. A2's 4 ties A3's defense of 4: A2 never wins TGG, so
	// the help it received is applied, not void.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeSupport, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TFF", "TGG"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TFF", TargetIDs: []models.TerritoryID{"TGG"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "TGG", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if event, found := outcomeForArmy(first.Events, "A1"); !found || event.Reason != "support_applied" {
		t.Errorf("A1 outcome = %#v, found=%t, want support_applied", event, found)
	}
	if event, found := combatAt(first.Events, "TGG"); !found || event.Reason != "standoff" {
		t.Errorf("TGG contest = %#v, found=%t, want standoff", event, found)
	}
	for attempt := 0; attempt < 50; attempt++ {
		again, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("resolution %d differs from the first one", attempt)
		}
	}
}

func TestResolveJoinCrossesAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB"),
			territory("TBB", "TBB", "TAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TBB", Size: 1},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P2", "TBB")
	// TAA is under attack from A2, so A1's join out of it is cancelled
	// outright: it stays, and its presence now defends TAA against A2.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "TBB", TargetIDs: []models.TerritoryID{"TAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TAA" {
		t.Errorf("A1 = %+v, want TAA", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "TBB" {
		t.Errorf("A2 = %+v, want TBB", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "attacked_origin" {
		t.Errorf("A1 outcome = %#v, found=%t, want attacked_origin", event, found)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
		t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
	}
}

func TestResolveDisperseJoinsAlliedAttackWinner(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB"),
			territory("TBB", "TBB", "TAA", "TCC"),
			territory("TCC", "TCC", "TBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "TAA", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "TCC", Size: 2},
			{ID: "A3", OwnerID: "P2", TerritoryID: "TBB", Size: 1},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "TAA")
	addNoble(state, "N2", "TWO", "P1", "TCC")
	addNoble(state, "N3", "THR", "P2", "TBB")
	// A1 dislodges A3 from TBB; A2's dispersion arrives there as an allied
	// join would, and its troop fuses with the winner.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeDisperse, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TBB"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "TBB"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "TBB" || army.Size != 3 {
		t.Errorf("A1 = %+v, want the winner fused to size 3 at TBB", army)
	}
	residual := false
	for _, army := range resolution.State.Armies {
		residual = residual || army.TerritoryID == "TCC" && army.OwnerID == "P1" && army.Size == 1
	}
	if !residual {
		t.Errorf("armies = %+v, want A2's residual troop at TCC", resolution.State.Armies)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "disperse_complete" {
		t.Errorf("A2 outcome = %#v, found=%t, want disperse_complete", event, found)
	}
}

func TestResolveDisperseFusesWithWinnerWhileEnemyDisperses(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("TAA", "TAA", "TBB"),
			territory("TBB", "TBB", "TAA", "TCC"),
			territory("TCC", "TCC", "TBB"),
			territory("TEE", "TEE", "TGG"),
			territory("TGG", "TGG", "TEE"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P3", TerritoryID: "TAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "TEE", Size: 1},
			{ID: "A3", OwnerID: "P3", TerritoryID: "TCC", Size: 2},
		},
	)
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P3", "TAA")
	addNoble(state, "N2", "TWO", "P2", "TEE")
	addNoble(state, "N3", "THR", "P3", "TCC")
	// TBB is empty, so A3's attack wins it uncontested and A1's dispersion
	// arrives there as an allied join would, fusing with the winner. A2, an
	// unrelated enemy, disperses at the same time from an origin under no
	// attack at all, unaffected by TBB's combat.
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeDisperse, PositionID: "TAA", TargetIDs: []models.TerritoryID{"TBB"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TBB": {"*"}}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeDisperse, PositionID: "TEE", TargetIDs: []models.TerritoryID{"TGG"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"TGG": {"*"}}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "TCC", TargetIDs: []models.TerritoryID{"TBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "TBB" || army.Size != 3 {
		t.Errorf("A3 = %+v, want the winner fused to size 3 at TBB", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "TGG" {
		t.Errorf("A2 = %+v, want TGG", army)
	}
}
