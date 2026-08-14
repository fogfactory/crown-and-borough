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
			nobles: []models.Noble{testNoble("N1", "ONE", "P1", "AAA", models.NobleStatusFree)},
			want:   1,
		},
		{
			name: "several allied free nobles do not stack",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "AAA", models.NobleStatusFree),
				testNoble("N2", "TWO", "P1", "AAA", models.NobleStatusFree),
			},
			want: 1,
		},
		{
			name: "captured enemy noble",
			nobles: []models.Noble{
				testNoble("N2", "TWO", "P2", "AAA", models.NobleStatusHostage),
			},
		},
		{
			name: "allied detained noble",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "AAA", models.NobleStatusDungeon),
			},
		},
		{
			name: "free noble on another territory",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "BBB", models.NobleStatusFree),
			},
		},
		{
			name: "famine overrides the bonus",
			nobles: []models.Noble{
				testNoble("N1", "ONE", "P1", "AAA", models.NobleStatusFree),
			},
			famished: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{
					territory("AAA", "AAA", "BBB"),
					territory("BBB", "BBB", "AAA"),
				},
				[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
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
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	contender := combatContenderForNobleBonus(t, resolution.Events, "BBB", "A1")
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
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N3", "THR", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{
			Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
		})
		addChain(t, state, "A3", "N3", models.Order{
			Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		contender := combatContenderForNobleBonus(t, resolution.Events, "BBB", "A1")
		if contender.Force != 4 {
			t.Errorf("offensive contender = %#v, want attack plus two command bonuses", contender)
		}
	})

	t.Run("defensive support", func(t *testing.T) {
		state := suppliedNobleCombatState(t,
			[]models.Territory{
				territory("AAA", "AAA", "BBB"),
				territory("BBB", "BBB", "AAA", "CCC"),
				territory("CCC", "CCC", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
				{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
			})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addNoble(state, "N3", "THR", "P2", "CCC")
		addChain(t, state, "A1", "N1", models.Order{
			Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
		})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeHold, PositionID: "BBB"})
		addChain(t, state, "A3", "N3", models.Order{
			Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"BBB"},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		contender := combatContenderForNobleBonus(t, resolution.Events, "BBB", "A2")
		if contender.Force != 4 || contender.NobleBonus != 1 {
			t.Errorf("defensive contender = %#v, want force 4 and noble bonus 1", contender)
		}
	})
}

func TestResolveFamishedNobleCommandHasZeroForce(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	contender := combatContenderForNobleBonus(t, resolution.Events, "BBB", "A1")
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
