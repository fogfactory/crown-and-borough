package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestResolveAttacksEnterSameTurnVacatedDestination(t *testing.T) {
	for _, test := range []struct {
		name          string
		defenderOwner models.PlayerID
	}{
		{name: "allied departure", defenderOwner: "P1"},
		{name: "enemy departure", defenderOwner: "P2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("AAA", "SVM", "BBB", "CCC"),
					territory("BBB", "BOM", "AAA"),
					territory("CCC", "THE", "AAA"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: test.defenderOwner, TerritoryID: "AAA", Size: 1},
					{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", test.defenderOwner, "AAA")
			addNoble(state, "N2", "TWO", "P1", "BBB")
			addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
			addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
			validateTestState(t, state)

			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "CCC" {
				t.Errorf("A1 = %+v, want CCC", army)
			}
			if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
				t.Errorf("A2 = %+v, want AAA", army)
			}
			for _, armyID := range []models.ArmyID{"A1", "A2"} {
				event, found := outcomeForArmy(resolution.Events, armyID)
				if !found || event.Outcome != OutcomeSuccess || event.Reason != "attack_wins" {
					t.Errorf("%s outcome = %#v, found=%t, want successful attack_wins", armyID, event, found)
				}
			}
			event, found := combatAt(resolution.Events, "AAA")
			if !found {
				t.Fatal("missing AAA contest event")
			}
			if event.DislodgedArmyID != "" || event.WinnerArmyID != "A2" || event.Defense != 0 || len(event.Contenders) != 1 || event.Contenders[0].ArmyID != "A2" {
				t.Errorf("AAA contest = %#v, want an empty vacated destination won by A2", event)
			}
		})
	}
}

func TestResolveCombatRemainsWhenOccupantStaysOrFailsToMove(t *testing.T) {
	t.Run("holding defender", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("SVM", "SVM", "BOM"),
				territory("BOM", "BOM", "SVM"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "SVM", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BOM", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "SVM")
		addNoble(state, "N2", "TWO", "P1", "BOM")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "SVM"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BOM", TargetIDs: []models.TerritoryID{"SVM"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BOM" {
			t.Errorf("A2 = %+v, want BOM", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
			t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
		}
	})

	t.Run("failed departure", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("SVM", "SVM", "BOM", "THE"),
				territory("BOM", "BOM", "SVM"),
				territory("THE", "THE", "SVM"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "SVM", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BOM", Size: 1},
				{ID: "A3", OwnerID: "P3", TerritoryID: "THE", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "SVM")
		addNoble(state, "N2", "TWO", "P1", "BOM")
		addNoble(state, "N3", "THR", "P3", "THE")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "SVM", TargetIDs: []models.TerritoryID{"THE"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BOM", TargetIDs: []models.TerritoryID{"SVM"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "THE"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		for _, armyID := range []models.ArmyID{"A1", "A2"} {
			if army := armyByID(t, resolution.State, armyID); army.TerritoryID != map[models.ArmyID]models.TerritoryID{"A1": "SVM", "A2": "BOM"}[armyID] {
				t.Errorf("%s = %+v, want to remain in its origin", armyID, army)
			}
			if event, found := outcomeForArmy(resolution.Events, armyID); !found || event.Reason != "combat_lost" {
				t.Errorf("%s outcome = %#v, found=%t, want combat_lost", armyID, event, found)
			}
		}
	})
}

func TestResolveDefenderOwnedSupportCannotDislodgeItsOwner(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC"),
			territory("BBB", "BOM", "AAA", "CCC"),
			territory("CCC", "THE", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Errorf("A1 = %+v, want AAA", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("A2 = %+v, want BBB", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
		t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
	}
}

func TestResolveNoHelpSupportAffectsHeadToHeadSwap(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD", "EEE"),
			territory("BBB", "BBB", "AAA", "CCC", "DDD", "EEE"),
			territory("CCC", "CCC", "AAA", "BBB"),
			territory("DDD", "DDD", "AAA", "BBB"),
			territory("EEE", "EEE", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			{ID: "A5", OwnerID: "P2", TerritoryID: "EEE", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")
	addNoble(state, "N5", "FIV", "P2", "EEE")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "DDD", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeSupport, PositionID: "EEE", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "AAA"}, {"A2", "BBB"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s = %+v, want %s after no-help head-to-head bounce", want.armyID, army, want.territoryID)
		}
	}
	if event, found := outcomeForArmy(resolution.Events, "A4"); !found || event.Reason != "support_void" {
		t.Errorf("A4 outcome = %#v, found=%t, want support_void for no-help swap support", event, found)
	}
}

func TestResolveNoHelpSupportOfHeadToHeadWinnerIsApplied(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		},
	)
	state.Territories[1].Terrain = models.TerrainMountain
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P2", "CCC")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
		t.Errorf("A1 = %+v, want BBB", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A3"); !found || event.Reason != "support_applied" {
		t.Errorf("A3 outcome = %#v, found=%t, want support_applied for the winning move", event, found)
	}
}

func TestResolveNoHelpSupportOfRetiredSwapLoserRemainsVoid(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD"),
			territory("BBB", "BBB", "AAA", "CCC", "DDD"),
			territory("CCC", "CCC", "AAA", "BBB"),
			territory("DDD", "DDD", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P1", "DDD")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "DDD", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
		t.Errorf("A1 = %+v, want BBB", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A4"); !found || event.Reason != "support_void" {
		t.Errorf("A4 outcome = %#v, found=%t, want persistent support_void", event, found)
	}
}

func TestResolveDislodgedHeadToHeadLoserUnblocksThirdAttacker(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG"),
			territory("BBB", "BBB", "AAA", "DDD", "EEE", "FFF"),
			territory("CCC", "CCC", "AAA", "GGG"),
			territory("DDD", "DDD", "AAA", "BBB"),
			territory("EEE", "EEE", "AAA", "BBB"),
			territory("FFF", "FFF", "AAA", "BBB"),
			territory("GGG", "GGG", "AAA", "CCC"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
			{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
			{ID: "A6", OwnerID: "P2", TerritoryID: "FFF", Size: 1},
			{ID: "A7", OwnerID: "P3", TerritoryID: "GGG", Size: 1},
		},
	)
	for _, noble := range []struct {
		id        models.NobleID
		code      string
		owner     models.PlayerID
		territory models.TerritoryID
	}{
		{"N1", "ONE", "P1", "AAA"}, {"N2", "TWO", "P2", "BBB"}, {"N3", "THR", "P3", "CCC"},
		{"N4", "FOU", "P1", "DDD"}, {"N5", "FIV", "P1", "EEE"}, {"N6", "SIX", "P2", "FFF"}, {"N7", "SEV", "P3", "GGG"},
	} {
		addNoble(state, noble.id, noble.code, noble.owner, noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "DDD", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeSupport, PositionID: "EEE", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
	addChain(t, state, "A6", "N6", models.Order{Type: models.OrderTypeSupport, PositionID: "FFF", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	addChain(t, state, "A7", "N7", models.Order{Type: models.OrderTypeSupport, PositionID: "GGG", TargetIDs: []models.TerritoryID{"CCC", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
		t.Errorf("A1 = %+v, want BBB", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "AAA" {
		t.Errorf("A3 = %+v, want AAA after the head-to-head loser's ghost is retired", army)
	}
	if hasArmy(resolution.State, "A2") {
		t.Errorf("A2 should be dislodged and destroyed, state = %#v", resolution.State.Armies)
	}
}

func TestResolveBouncedSwapAttackBlocksThirdAttacker(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "CCC"),
			territory("BBB", "BBB", "AAA"),
			territory("CCC", "CCC", "AAA"),
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
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "AAA"}, {"A2", "BBB"}, {"A3", "CCC"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s = %+v, want %s because the bounced swap attack retains its destination force", want.armyID, army, want.territoryID)
		}
	}
}

func TestResolveDislodgedAttackerStillCutsSupport(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC", "DDD", "FFF"),
			territory("BBB", "BOM", "AAA", "CCC"),
			territory("CCC", "THE", "AAA", "BBB"),
			territory("DDD", "ATL", "AAA", "EEE", "FFF"),
			territory("EEE", "NOR", "DDD", "FFF"),
			territory("FFF", "PIC", "AAA", "DDD", "EEE"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
			{ID: "A6", OwnerID: "P1", TerritoryID: "FFF", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P3", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")
	addNoble(state, "N5", "FIV", "P1", "EEE")
	addNoble(state, "N6", "SIX", "P1", "FFF")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeSupport, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB", "CCC"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "CCC"})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeAttack, PositionID: "DDD", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeAttack, PositionID: "EEE", TargetIDs: []models.TerritoryID{"DDD"}})
	addChain(t, state, "A6", "N6", models.Order{Type: models.OrderTypeSupport, PositionID: "FFF", TargetIDs: []models.TerritoryID{"EEE", "DDD"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("A2 = %+v, want BBB after cut support", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A4"); !found || event.Reason != "dislodged" {
		t.Errorf("A4 outcome = %#v, found=%t, want dislodged", event, found)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "support_cut" {
		t.Errorf("A1 outcome = %#v, found=%t, want support_cut", event, found)
	}
}

func TestResolveSelfDislodgeBounceProtectsAlliedDefender(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC", "DDD", "EEE", "FFF"),
			territory("BBB", "BOM", "AAA", "EEE", "FFF"),
			territory("CCC", "THE", "AAA", "DDD"),
			territory("DDD", "ATL", "AAA", "CCC"),
			territory("EEE", "NOR", "AAA", "BBB"),
			territory("FFF", "PIC", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P2", TerritoryID: "DDD", Size: 1},
			{ID: "A5", OwnerID: "P2", TerritoryID: "EEE", Size: 1},
			{ID: "A6", OwnerID: "P2", TerritoryID: "FFF", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P2", "CCC")
	addNoble(state, "N4", "FOU", "P2", "DDD")
	addNoble(state, "N5", "FIV", "P2", "EEE")
	addNoble(state, "N6", "SIX", "P2", "FFF")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "DDD", TargetIDs: []models.TerritoryID{"CCC", "AAA"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeSupport, PositionID: "EEE", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	addChain(t, state, "A6", "N6", models.Order{Type: models.OrderTypeSupport, PositionID: "FFF", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
		t.Errorf("A1 = %+v, want AAA protected by the self-dislodge bounce", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("A2 = %+v, want self-dislodge bounce at BBB", army)
	}
	if army := armyByID(t, resolution.State, "A3"); army.TerritoryID != "CCC" {
		t.Errorf("A3 = %+v, want third attacker to bounce at CCC", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "allied_destination" {
		t.Errorf("A2 outcome = %#v, found=%t, want allied_destination", event, found)
	}
	if event, found := outcomeForArmy(resolution.Events, "A3"); !found || event.Reason != "combat_lost" {
		t.Errorf("A3 outcome = %#v, found=%t, want combat_lost", event, found)
	}
	if event, found := combatAt(resolution.Events, "AAA"); !found || event.WinnerArmyID != "" || event.DislodgedArmyID != "" || !hasContender(event, "A2") || !hasContender(event, "A3") {
		t.Errorf("AAA contest = %#v, found=%t, want protected defender with both bounced attackers", event, found)
	}
}

func TestResolveJoinDepartureVacatesOriginBeforeAttack(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC"),
			territory("BBB", "BOM", "AAA"),
			territory("CCC", "THE", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P2", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "CCC" {
		t.Errorf("A1 = %+v, want join destination CCC", army)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
		t.Errorf("A2 = %+v, want attack entry into freed AAA", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A1"); !found || event.Reason != "join_move" {
		t.Errorf("A1 outcome = %#v, found=%t, want join_move", event, found)
	}
}

func TestResolveDisperseDepartureChangesOriginDefense(t *testing.T) {
	t.Run("full disperse frees origin", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("SVM", "SVM", "BOM", "THE"),
				territory("BOM", "BOM", "SVM"),
				territory("THE", "THE", "SVM"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "SVM", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BOM", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "SVM")
		addNoble(state, "N2", "TWO", "P1", "BOM")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeDisperse, PositionID: "SVM", TargetIDs: []models.TerritoryID{"THE"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"THE": {"ONE"}}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BOM", TargetIDs: []models.TerritoryID{"SVM"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "THE" {
			t.Errorf("A1 = %+v, want disperse destination THE", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "SVM" {
			t.Errorf("A2 = %+v, want entry into freed SVM", army)
		}
	})

	t.Run("partial disperse leaves residual defense", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("SVM", "SVM", "BOM", "THE", "ATL", "NOR"),
				territory("BOM", "BOM", "SVM", "NOR"),
				territory("THE", "THE", "SVM"),
				territory("ATL", "ATL", "SVM"),
				territory("NOR", "NOR", "SVM", "BOM"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "SVM", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BOM", Size: 1},
				{ID: "A3", OwnerID: "P3", TerritoryID: "ATL", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "NOR", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "SVM")
		addNoble(state, "N2", "TWO", "P1", "BOM")
		addNoble(state, "N3", "THR", "P3", "ATL")
		addNoble(state, "N4", "FOU", "P1", "NOR")
		setTerritoryOwner(state, "SVM", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "SVM"})
		state.TerritoryStates["SVM"] = models.TerritoryState{OwnerID: state.TerritoryStates["SVM"].OwnerID, Infrastructures: []models.InfraID{"I1"}, Resources: 1, Army: state.TerritoryStates["SVM"].Army}
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeDisperse, PositionID: "SVM", TargetIDs: []models.TerritoryID{"THE", "ATL"}, NobleAssignments: map[models.TerritoryID][]models.NobleCode{"THE": {"ONE"}}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BOM", TargetIDs: []models.TerritoryID{"SVM"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "ATL"})
		addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "NOR", TargetIDs: []models.TerritoryID{"BOM", "SVM"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "SVM" {
			t.Errorf("A2 = %+v, want attack through residual defense", army)
		}
		if event, found := combatAt(resolution.Events, "SVM"); !found || event.Defense != 2 || event.WinnerArmyID != "A2" {
			t.Errorf("SVM contest = %#v, found=%t, want residual defense 2 including the command bonus and A2 winner", event, found)
		}
	})
}

func TestResolveAlliedDestinationFailureIsDeferredAndDoesNotBlockJoin(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC", "DDD"),
			territory("BBB", "BOM", "AAA"),
			territory("CCC", "THE", "AAA"),
			territory("DDD", "ATL", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
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
	} {
		addNoble(state, noble.id, noble.code, "P1", noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "DDD", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, armyID := range []models.ArmyID{"A2", "A3"} {
		if event, found := outcomeForArmy(resolution.Events, armyID); !found || event.Reason != "allied_destination" {
			t.Errorf("%s outcome = %#v, found=%t, want allied_destination", armyID, event, found)
		}
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 2 {
		t.Errorf("host = %+v, want fused size 2 at AAA", army)
	}
	if hasArmy(resolution.State, "A4") {
		t.Error("A4 should fuse into the stationary host")
	}
	if _, found := combatAt(resolution.Events, "AAA"); found {
		t.Error("allied destination failures should not create a AAA combat event")
	}
}

func TestResolveSupportToAlliedFailedAttackIsVoid(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC"),
			territory("BBB", "BOM", "AAA"),
			territory("CCC", "THE", "AAA"),
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
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if event, found := outcomeForArmy(resolution.Events, "A3"); !found || event.Reason != "support_void" {
		t.Errorf("A3 outcome = %#v, found=%t, want support_void", event, found)
	}
}

func TestResolveAttackCycles(t *testing.T) {
	t.Run("swap", func(t *testing.T) {
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
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" {
			t.Errorf("A1 = %+v, want AAA after head-to-head bounce", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
			t.Errorf("A2 = %+v, want BBB after head-to-head bounce", army)
		}
		for _, armyID := range []models.ArmyID{"A1", "A2"} {
			if event, found := outcomeForArmy(resolution.Events, armyID); !found || event.Reason != "combat_lost" {
				t.Errorf("%s outcome = %#v, found=%t, want combat_lost", armyID, event, found)
			}
		}
	})

	t.Run("stronger head-to-head dislodges weaker", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "AAA", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want BBB", army)
		}
		if hasArmy(resolution.State, "A2") {
			t.Errorf("A2 should be dislodged from BBB, state = %#v", resolution.State.Armies)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "dislodged" {
			t.Errorf("A2 outcome = %#v, found=%t, want dislodged", event, found)
		}
	})

	t.Run("rotation", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB", "CCC"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "AAA", "BBB"),
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
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"CCC"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		for _, want := range []struct {
			armyID      models.ArmyID
			territoryID models.TerritoryID
		}{
			{"A1", "BBB"}, {"A2", "CCC"}, {"A3", "AAA"},
		} {
			if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
				t.Errorf("%s = %+v, want %s", want.armyID, army, want.territoryID)
			}
		}
	})
}

func TestResolveFailedDepartureBouncesBackThroughChain(t *testing.T) {
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
			{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N3", "THR", "P3", "CCC")
	addNoble(state, "N4", "FOU", "P1", "DDD")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "CCC", TargetIDs: []models.TerritoryID{"DDD"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeHold, PositionID: "DDD"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "AAA"}, {"A2", "BBB"}, {"A3", "CCC"}, {"A4", "DDD"},
	} {
		if army := armyByID(t, resolution.State, want.armyID); army.TerritoryID != want.territoryID {
			t.Errorf("%s = %+v, want %s", want.armyID, army, want.territoryID)
		}
	}
	for _, armyID := range []models.ArmyID{"A1", "A2", "A3"} {
		if event, found := outcomeForArmy(resolution.Events, armyID); !found || event.Reason != "combat_lost" {
			t.Errorf("%s outcome = %#v, found=%t, want combat_lost", armyID, event, found)
		}
	}
}

func TestResolveVacatedCastleStillDefends(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC"),
			territory("BBB", "BOM", "AAA"),
			territory("CCC", "THE", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P2", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	setNobleStatus(state, "N2", models.NobleStatusHostage)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
		t.Errorf("A2 = %+v, want BBB after castle standoff", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
		t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
	}
	if event, found := combatAt(resolution.Events, "AAA"); !found || event.Defense != 1 || event.WinnerArmyID != "" {
		t.Errorf("AAA contest = %#v, found=%t, want castle-only standoff", event, found)
	}
}

func TestResolveJoinEntersVacatedEnemyDestination(t *testing.T) {
	t.Run("vacated", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "SVM", "BBB", "CCC"),
				territory("BBB", "BOM", "AAA"),
				territory("CCC", "THE", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
			t.Errorf("A2 = %+v, want AAA", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "join_move" {
			t.Errorf("A2 outcome = %#v, found=%t, want join_move", event, found)
		}
	})

	t.Run("staying enemy", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("AAA", "SVM", "BBB"),
				territory("BBB", "BOM", "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "AAA")
		addNoble(state, "N2", "TWO", "P1", "BBB")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "BBB" {
			t.Errorf("A2 = %+v, want BBB", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "enemy_destination" {
			t.Errorf("A2 outcome = %#v, found=%t, want enemy_destination", event, found)
		}
	})
}

func TestResolveJoinFusesWithWinnerAfterDefenderIsDislodged(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "SVM", "BBB", "CCC", "DDD"),
			territory("BBB", "BOM", "AAA", "DDD"),
			territory("CCC", "THE", "AAA"),
			territory("DDD", "ATL", "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P2", "AAA")
	addNoble(state, "N2", "TWO", "P1", "BBB")
	addNoble(state, "N3", "THR", "P1", "CCC")
	addNoble(state, "N4", "FOU", "P1", "DDD")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeJoin, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeSupport, PositionID: "DDD", TargetIDs: []models.TerritoryID{"BBB", "AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" || army.Size != 2 {
		t.Errorf("A2 = %+v, want fused winner size 2 at AAA", army)
	}
	if hasArmy(resolution.State, "A3") {
		t.Error("A3 should fuse into A2")
	}
	if event, found := outcomeForArmy(resolution.Events, "A3"); !found || event.Reason != "join_attack_arrival" {
		t.Errorf("A3 outcome = %#v, found=%t, want join_attack_arrival", event, found)
	}
}

func TestResolveVacatedAttackIsDeterministic(t *testing.T) {
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
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
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

func outcomeForArmy(events []Event, armyID models.ArmyID) (Event, bool) {
	for _, event := range events {
		if event.Type == EventTypeOrderOutcome && event.ArmyID == armyID {
			return event, true
		}
	}
	return Event{}, false
}

func combatAt(events []Event, territoryID models.TerritoryID) (Event, bool) {
	for _, event := range events {
		if event.Type == EventTypeCombat && event.TerritoryID == territoryID {
			return event, true
		}
	}
	return Event{}, false
}

func hasContender(event Event, armyID models.ArmyID) bool {
	for _, contender := range event.Contenders {
		if contender.ArmyID == armyID {
			return true
		}
	}
	return false
}
