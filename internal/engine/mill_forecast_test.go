package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestForecastMillIncomeCreditsAdjacentSettlement checks the basic case: a
// controlled castle adjacent to a level-2 mill is forecast to receive its
// full production.
func TestForecastMillIncomeCreditsAdjacentSettlement(t *testing.T) {
	state := testState(t,
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

	totals := ForecastMillIncome(state, testBalance())
	if totals["P1"] != 2 {
		t.Fatalf("mill income = %#v, want 2 for P1", totals)
	}
}

// TestForecastMillIncomeCreditsEachAdjacentSettlement documents the current
// (pre-#195) mechanic: a mill credits every adjacent controlled settlement
// independently, so a player with two settlements next to the same mill is
// forecast to receive its production twice over.
func TestForecastMillIncomeCreditsEachAdjacentSettlement(t *testing.T) {
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
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 3, TerritoryID: "MIL"})
	validateTestState(t, state)

	totals := ForecastMillIncome(state, testBalance())
	if totals["P1"] != 6 {
		t.Fatalf("mill income = %#v, want 6 (3 credited to each of AAA and BBB)", totals)
	}
}

// TestForecastMillIncomeIgnoresMillOwner checks that a mill's production is
// credited regardless of who controls the mill's own territory.
func TestForecastMillIncomeIgnoresMillOwner(t *testing.T) {
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
	if totals["P1"] != 1 {
		t.Fatalf("mill income = %#v, want 1 for P1 regardless of the mill's owner", totals)
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
