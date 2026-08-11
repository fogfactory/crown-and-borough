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
					territory("T01", "SVM", "T02", "T03"),
					territory("T02", "BOM", "T01"),
					territory("T03", "THE", "T01"),
				},
				[]models.Army{
					{ID: "A1", OwnerID: test.defenderOwner, TerritoryID: "T01", Size: 1},
					{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
				},
			)
			addNoble(state, "N1", "ONE", test.defenderOwner, "T01")
			addNoble(state, "N2", "TWO", "P1", "T02")
			addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
			addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
			validateTestState(t, state)

			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T03" {
				t.Errorf("A1 = %+v, want T03", army)
			}
			if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" {
				t.Errorf("A2 = %+v, want T01", army)
			}
			for _, armyID := range []models.ArmyID{"A1", "A2"} {
				event, found := outcomeForArmy(resolution.Events, armyID)
				if !found || event.Outcome != OutcomeSuccess || event.Reason != "attack_wins" {
					t.Errorf("%s outcome = %#v, found=%t, want successful attack_wins", armyID, event, found)
				}
			}
			event, found := combatAt(resolution.Events, "T01")
			if !found {
				t.Fatal("missing T01 contest event")
			}
			if event.DislodgedArmyID != "" || event.WinnerArmyID != "A2" || event.Defense != 0 {
				t.Errorf("T01 contest = %#v, want an empty vacated destination won by A2", event)
			}
		})
	}
}

func TestResolveCombatRemainsWhenOccupantStaysOrFailsToMove(t *testing.T) {
	t.Run("holding defender", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "SVM", "T02"),
				territory("T02", "BOM", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "T01"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" {
			t.Errorf("A2 = %+v, want T02", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
			t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
		}
	})

	t.Run("failed departure", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "SVM", "T02", "T03"),
				territory("T02", "BOM", "T01"),
				territory("T03", "THE", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P3", TerritoryID: "T03", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addNoble(state, "N3", "THR", "P3", "T03")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "T03"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		for _, armyID := range []models.ArmyID{"A1", "A2"} {
			if army := armyByID(t, resolution.State, armyID); army.TerritoryID != map[models.ArmyID]models.TerritoryID{"A1": "T01", "A2": "T02"}[armyID] {
				t.Errorf("%s = %+v, want to remain in its origin", armyID, army)
			}
			if event, found := outcomeForArmy(resolution.Events, armyID); !found || event.Reason != "combat_lost" {
				t.Errorf("%s outcome = %#v, found=%t, want combat_lost", armyID, event, found)
			}
		}
	})
}

func TestResolveAlliedDestinationFailureIsDeferredAndDoesNotBlockJoin(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("T01", "SVM", "T02", "T03", "T04"),
			territory("T02", "BOM", "T01"),
			territory("T03", "THE", "T01"),
			territory("T04", "ATL", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "T04", Size: 1},
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
	} {
		addNoble(state, noble.id, noble.code, "P1", noble.territory)
	}
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "T01"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeJoin, PositionID: "T04", TargetIDs: []models.TerritoryID{"T01"}})
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
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T01" || army.Size != 2 {
		t.Errorf("host = %+v, want fused size 2 at T01", army)
	}
	if hasArmy(resolution.State, "A4") {
		t.Error("A4 should fuse into the stationary host")
	}
	if _, found := combatAt(resolution.Events, "T01"); found {
		t.Error("allied destination failures should not create a T01 combat event")
	}
}

func TestResolveAttackCyclesSucceed(t *testing.T) {
	t.Run("swap", func(t *testing.T) {
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
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "T02" {
			t.Errorf("A1 = %+v, want T02", army)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" {
			t.Errorf("A2 = %+v, want T01", army)
		}
	})

	t.Run("rotation", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02", "T03"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T01", "T02"),
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
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		for _, want := range []struct {
			armyID      models.ArmyID
			territoryID models.TerritoryID
		}{
			{"A1", "T02"}, {"A2", "T03"}, {"A3", "T01"},
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
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02", "T04"),
			territory("T04", "DDD", "T03"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
			{ID: "A3", OwnerID: "P3", TerritoryID: "T03", Size: 1},
			{ID: "A4", OwnerID: "P1", TerritoryID: "T04", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "T01")
	addNoble(state, "N2", "TWO", "P2", "T02")
	addNoble(state, "N3", "THR", "P3", "T03")
	addNoble(state, "N4", "FOU", "P1", "T04")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T03"}})
	addChain(t, state, "A3", "N3", models.Order{Type: models.OrderTypeAttack, PositionID: "T03", TargetIDs: []models.TerritoryID{"T04"}})
	addChain(t, state, "A4", "N4", models.Order{Type: models.OrderTypeHold, PositionID: "T04"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, want := range []struct {
		armyID      models.ArmyID
		territoryID models.TerritoryID
	}{
		{"A1", "T01"}, {"A2", "T02"}, {"A3", "T03"}, {"A4", "T04"},
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
			territory("T01", "SVM", "T02", "T03"),
			territory("T02", "BOM", "T01"),
			territory("T03", "THE", "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
		},
	)
	addNoble(state, "N1", "ONE", "P2", "T01")
	addNoble(state, "N2", "TWO", "P1", "T02")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "T01"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" {
		t.Errorf("A2 = %+v, want T02 after castle standoff", army)
	}
	if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "combat_lost" {
		t.Errorf("A2 outcome = %#v, found=%t, want combat_lost", event, found)
	}
	if event, found := combatAt(resolution.Events, "T01"); !found || event.Defense != 1 || event.WinnerArmyID != "" {
		t.Errorf("T01 contest = %#v, found=%t, want castle-only standoff", event, found)
	}
}

func TestResolveJoinEntersVacatedEnemyDestination(t *testing.T) {
	t.Run("vacated", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "SVM", "T02", "T03"),
				territory("T02", "BOM", "T01"),
				territory("T03", "THE", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T03"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T01" {
			t.Errorf("A2 = %+v, want T01", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "join_move" {
			t.Errorf("A2 outcome = %#v, found=%t, want join_move", event, found)
		}
	})

	t.Run("staying enemy", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				territory("T01", "SVM", "T02"),
				territory("T02", "BOM", "T01"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P2", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "T02", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P2", "T01")
		addNoble(state, "N2", "TWO", "P1", "T02")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "T01"})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeJoin, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "T02" {
			t.Errorf("A2 = %+v, want T02", army)
		}
		if event, found := outcomeForArmy(resolution.Events, "A2"); !found || event.Reason != "enemy_destination" {
			t.Errorf("A2 outcome = %#v, found=%t, want enemy_destination", event, found)
		}
	})
}

func TestResolveVacatedAttackIsDeterministic(t *testing.T) {
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
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"}})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "T02", TargetIDs: []models.TerritoryID{"T01"}})
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
