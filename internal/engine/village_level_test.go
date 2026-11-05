package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// villageIncomeState builds a player owning AAA (castle capital) and BBB
// (village of the given level), both anchored under one fief.
func villageIncomeState(t *testing.T, level int) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA", "CCC"), territory("CCC", "CCC", "BBB")},
		nil,
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: level, TerritoryID: "BBB"})
	setCapital(state, "P1", "I1")
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}}
	validateTestState(t, state)
	return state
}

func TestVillageIncomeScalesWithLevel(t *testing.T) {
	balance := testBalance()
	balance.VillageIncome = 2
	for level, want := range map[int]int{0: 3, 1: 5, 2: 7, 3: 9} {
		// Three territories at territory_income 1, plus 2 R per village level.
		state := villageIncomeState(t, level)
		resolution, err := Resolve(state, balance)
		if err != nil {
			t.Fatalf("level %d: Resolve: %v", level, err)
		}
		event := incomeEventFor(t, resolution.Events, "P1", "AAA")
		if event.Production != want || event.VillageCount != 1 {
			t.Errorf("level %d: income event = %#v, want %d R and 1 village", level, event, want)
		}
		forecast := ForecastTerritoryIncome(villageIncomeState(t, level), balance)
		if got := forecast["BBB"].Amount; got != 1+2*level {
			t.Errorf("level %d: BBB forecast = %d, want %d", level, got, 1+2*level)
		}
	}
}

func TestLevelZeroVillageProducesNothing(t *testing.T) {
	t.Run("controlled village yields only territory income", func(t *testing.T) {
		resolution, err := Resolve(villageIncomeState(t, 0), testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got := incomeEventFor(t, resolution.Events, "P1", "AAA").Production; got != 3 {
			t.Errorf("income = %d, want 3 (territory income only)", got)
		}
	})

	t.Run("neutral village stores nothing, a leveled one stores per level", func(t *testing.T) {
		for level, want := range map[int]int{0: 0, 1: 1, 3: 3} {
			state := testState(t,
				[]models.Territory{
					supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
					supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
				},
				[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 1}},
			)
			addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: level, TerritoryID: "AAA"})
			validateTestState(t, state)
			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("level %d: Resolve: %v", level, err)
			}
			if got := resolution.State.TerritoryStates["AAA"].Resources; got != want {
				t.Errorf("level %d: neutral village stock = %d, want %d", level, got, want)
			}
		}
	})
}

func TestLevelZeroVillageKeepsWinterConservation(t *testing.T) {
	state := winterTestState(t,
		[]models.Territory{territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA")},
		nil,
	)
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 0, TerritoryID: "BBB"})
	setCapital(state, "P1", "I1")
	setTerritoryResources(state, "BBB", 10)
	validateTestState(t, state)

	resolution, err := ResolveWinter(state, testBalance(), nil)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if got := resolution.State.TerritoryStates["BBB"].Resources; got != 1 {
		t.Errorf("level-0 village stock = %d, want 1 (10 halved to 5, capped at 1)", got)
	}
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 4 {
		t.Errorf("capital stock = %d, want the 4 R surplus", got)
	}
}
