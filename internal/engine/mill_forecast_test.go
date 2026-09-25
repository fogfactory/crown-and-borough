package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestForecastMillIncomeCreditsAdjacentSettlement checks the basic case: a
// controlled castle adjacent to a level-2 mill under the same control is
// forecast to receive its full production.
func TestForecastMillIncomeCreditsAdjacentSettlement(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "MIL"),
			supplyTerritory("MIL", "MIL", models.TerrainMountain, "AAA"),
		},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "MIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	validateTestState(t, state)

	totals := ForecastMillIncome(state, testBalance())
	if totals["P1"] != 2 {
		t.Fatalf("mill income = %#v, want 2 for P1", totals)
	}
}

// TestForecastMillIncomeCreditsSingleDestination checks #195's fix: a mill
// adjacent to both a controlled castle and a controlled village under the
// same control credits its production to the castle only, never doubling up.
func TestForecastMillIncomeCreditsSingleDestination(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "MIL"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "MIL"),
			supplyTerritory("MIL", "MIL", models.TerrainMountain, "AAA", "BBB"),
		},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "BBB", "P1")
	setTerritoryOwner(state, "MIL", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 3, TerritoryID: "MIL"})
	validateTestState(t, state)

	totals := ForecastMillIncome(state, testBalance())
	if totals["P1"] != 3 {
		t.Fatalf("mill income = %#v, want 3 credited once, to the castle only", totals)
	}
}

// TestForecastMillIncomeRequiresMatchingController checks #195's fix: a mill
// controlled by a different player than an adjacent castle never credits
// that castle; instead, having no eligible neighbor, it stocks itself.
func TestForecastMillIncomeRequiresMatchingController(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "MIL"),
			supplyTerritory("MIL", "MIL", models.TerrainMountain, "AAA"),
		},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "MIL", "P2")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "MIL"})
	validateTestState(t, state)

	totals := ForecastMillIncome(state, testBalance())
	if totals["P1"] != 0 {
		t.Fatalf("mill income = %#v, want 0 for P1: the mill's controller does not match the castle's", totals)
	}
	if totals["P2"] != 1 {
		t.Fatalf("mill income = %#v, want 1 for P2: the mill stocks itself instead", totals)
	}
}

// TestForecastMillIncomeNilInWinter checks that the forecast is never
// computed in winter, since ravitaillement itself never happens then.
func TestForecastMillIncomeNilInWinter(t *testing.T) {
	state := winterTestState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "MIL"),
			supplyTerritory("MIL", "MIL", models.TerrainMountain, "AAA"),
		},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "MIL"})
	validateTestState(t, state)

	if totals := ForecastMillIncome(state, testBalance()); totals != nil {
		t.Fatalf("totals = %#v, want nil in winter", totals)
	}
}
