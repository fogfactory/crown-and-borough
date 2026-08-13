package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestNobleCommandBonus(t *testing.T) {
	tests := []struct {
		name     string
		nobles   []models.Noble
		famished bool
		want     int
	}{
		{name: "no noble"},
		{
			name:   "one allied free noble",
			nobles: []models.Noble{testNoble("N1", "ONE", "P1", "T01", models.NobleStatusFree)},
			want:   1,
		},
		{
			name: "several allied free nobles do not stack",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "T01", models.NobleStatusFree),
				testNoble("N2", "TWO", "P1", "T01", models.NobleStatusFree),
			},
			want: 1,
		},
		{
			name: "captured enemy noble",
			nobles: []models.Noble{
				testNoble("N2", "TWO", "P2", "T01", models.NobleStatusHostage),
			},
		},
		{
			name: "allied detained noble",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "T01", models.NobleStatusDungeon),
			},
		},
		{
			name: "free noble on another territory",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "T02", models.NobleStatusFree),
			},
		},
		{
			name: "famine overrides the bonus",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "T01", models.NobleStatusFree),
			},
			famished: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("T01", "AAA", "T02"),
					territory("T02", "BBB", "T01"),
				},
				[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1}},
			)
			state.Nobles = append(state.Nobles, tt.nobles...)
			ctx := newResolutionContext(state, testBalance())
			ctx.famished["A1"] = tt.famished

			if got := nobleCommandBonus(ctx, state.Armies[0]); got != tt.want {
				t.Errorf("nobleCommandBonus = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveNobleCommandBonusInAttackAndReport(t *testing.T) {
	state := suppliedNobleCombatState(t,
		[]models.Territory{
			territory("T01", "AAA", "T02"),
			territory("T02", "BBB", "T01", "T03"),
			territory("T03", "CCC", "T02"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		})
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	contender := combatContenderForNobleBonus(t, resolution.Events, "T02", "A1")
	if contender.Force != 2 || contender.NobleBonus != 1 {
		t.Errorf("attacking contender = %#v, want force 2 and noble bonus 1", contender)
	}

	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	if len(report.Combats) != 1 || len(report.Combats[0].Contenders) != 2 {
		t.Fatalf("combat report = %#v, want one combat with two contenders", report.Combats)
	}
	reported := report.Combats[0].Contenders[1]
	if reported.ArmyID != "A1" || reported.NobleBonus != 1 {
		t.Errorf("reported contender = %#v, want A1 with noble bonus 1", reported)
	}
}

func TestResolveNobleCommandBonusInSupports(t *testing.T) {
	t.Run("offensive support", func(t *testing.T) {
		state := suppliedNobleCombatState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "T03", Size: 1},
			})
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N3", "THR", "P1", "T03")
		addChain(t, state, "A1", "N1", models.Order{
			Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"},
		})
		addChain(t, state, "A3", "N3", models.Order{
			Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T01", "T02"},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		contender := combatContenderForNobleBonus(t, resolution.Events, "T02", "A1")
		if contender.Force != 4 {
			t.Errorf("offensive contender = %#v, want attack plus two command bonuses", contender)
		}
	})

	t.Run("defensive support", func(t *testing.T) {
		state := suppliedNobleCombatState(t,
			[]models.Territory{
				territory("T01", "AAA", "T02"),
				territory("T02", "BBB", "T01", "T03"),
				territory("T03", "CCC", "T02"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "T03", Size: 1},
			})
		addNoble(state, "N1", "ONE", "P1", "T01")
		addNoble(state, "N2", "TWO", "P2", "T02")
		addNoble(state, "N3", "THR", "P2", "T03")
		addChain(t, state, "A1", "N1", models.Order{
			Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"},
		})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "T02"})
		addChain(t, state, "A3", "N3", models.Order{
			Type: models.OrderTypeSupport, PositionID: "T03", TargetIDs: []models.TerritoryID{"T02"},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		contender := combatContenderForNobleBonus(t, resolution.Events, "T02", "A2")
		if contender.Force != 4 || contender.NobleBonus != 1 {
			t.Errorf("defensive contender = %#v, want force 4 and noble bonus 1", contender)
		}
	})
}

func TestResolveFamishedNobleCommandHasZeroForce(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("T01", "AAA", models.TerrainMountain, "T02"),
			supplyTerritory("T02", "BBB", models.TerrainPlain, "T01"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "T01", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "T02", Size: 1},
		})
	addNoble(state, "N1", "ONE", "P1", "T01")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeAttack, PositionID: "T01", TargetIDs: []models.TerritoryID{"T02"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	contender := combatContenderForNobleBonus(t, resolution.Events, "T02", "A1")
	if contender.Force != 0 || contender.NobleBonus != 0 {
		t.Errorf("famished contender = %#v, want zero force and zero noble bonus", contender)
	}
}

func testNoble(id models.NobleID, code string, ownerID models.PlayerID, territoryID models.TerritoryID, status models.NobleStatus) models.Noble {
	return models.Noble{
		ID: id, Code: code, Name: string(code), OwnerID: ownerID,
		LocationID: territoryID, Status: status,
	}
}

func suppliedNobleCombatState(t *testing.T, territories []models.Territory, armies []models.Army) *models.GameState {
	t.Helper()
	state := testState(t, territories, armies)
	keepTestArmiesSupplied(state)
	return state
}

func combatContenderForNobleBonus(t *testing.T, events []Event, territoryID models.TerritoryID, armyID models.ArmyID) CombatContender {
	t.Helper()
	for _, event := range events {
		if event.Type != EventTypeCombat || event.TerritoryID != territoryID {
			continue
		}
		for _, contender := range event.Contenders {
			if contender.ArmyID == armyID {
				return contender
			}
		}
	}
	t.Fatalf("missing combat contender %q at %q in %#v", armyID, territoryID, events)
	return CombatContender{}
}
