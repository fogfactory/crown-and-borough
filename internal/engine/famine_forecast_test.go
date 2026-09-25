package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// TestForecastFamineRiskFlagsIsolatedArmy covers the issue's first hotseat
// scenario: an army with no supply source at all within reach is flagged at
// risk, with its estimated ration shortfall as the deficit.
func TestForecastFamineRiskFlagsIsolatedArmy(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	validateTestState(t, state)

	forecasts := ForecastFamineRisk(state, testBalance())
	forecast := forecasts["P1"]
	// armyCost(2, 2) = 2, mountain terrain rations = 1, no reachable source at
	// all: the whole remaining demand (1) is the estimated deficit.
	if forecast.TotalDemand != 2 || len(forecast.ArmiesAtRisk) != 1 {
		t.Fatalf("forecast = %#v, want A1 flagged at risk", forecast)
	}
	risk := forecast.ArmiesAtRisk[0]
	if risk.ArmyID != "A1" || risk.TerritoryID != "AAA" || risk.Size != 2 || risk.Deficit != 1 {
		t.Errorf("risk = %#v, want A1 at AAA with a deficit of 1", risk)
	}
}

// TestForecastFamineRiskNoRiskWithSufficientLocalProduction covers the
// issue's second hotseat scenario: an army whose own territory produces
// enough rations is never flagged, and the projected consumption matches the
// sum of every army's demand.
func TestForecastFamineRiskNoRiskWithSufficientLocalProduction(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	validateTestState(t, state)

	forecasts := ForecastFamineRisk(state, testBalance())
	forecast := forecasts["P1"]
	// armyCost(1, 2) = 1, plain terrain rations = 3, well above demand.
	if forecast.TotalDemand != 1 || len(forecast.ArmiesAtRisk) != 0 {
		t.Fatalf("forecast = %#v, want no army at risk", forecast)
	}
}

// TestForecastFamineRiskCountsReachableUncontestedSource checks that an
// army short on local production is not flagged when a controlled source it
// can reach alone covers the remaining deficit, per the heuristic's
// uncontested-reach rule.
func TestForecastFamineRiskCountsReachableUncontestedSource(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
	)
	setTerritoryOwner(state, "BBB", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	setTerritoryResources(state, "BBB", 5)
	validateTestState(t, state)

	balance := testBalance()
	// Zeroed so territory/village income does not perturb BBB's stock: this
	// test is about the reachable-source heuristic, not income.
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0

	forecasts := ForecastFamineRisk(state, balance)
	forecast := forecasts["P1"]
	// armyCost(3, 2) = 4, mountain terrain rations = 1, remaining 3 is fully
	// covered by BBB's banked stock (5) reachable within supply range.
	if forecast.TotalDemand != 4 || len(forecast.ArmiesAtRisk) != 0 {
		t.Fatalf("forecast = %#v, want A1 covered by BBB's reachable stock", forecast)
	}
}

// TestForecastFamineRiskIgnoresDrawnCalamities checks that the forecast
// always uses the normal, unaffected ration production, even when a bad
// harvest calamity is already scheduled for the current season and region:
// the command post projection must never leak an undrawn card's effect.
func TestForecastFamineRiskIgnoresDrawnCalamities(t *testing.T) {
	state := effectTestState()
	army := models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}
	state.Armies = append(state.Armies, army)
	territoryState := state.TerritoryStates["AAA"]
	armyID := army.ID
	territoryState.Army = &armyID
	state.TerritoryStates["AAA"] = territoryState
	state.NextArmyID = nextArmyID(state.Armies)
	setCurrentCalamity(state, models.CardKindFamine, "AAA")

	forecasts := ForecastFamineRisk(state, testBalance())
	forecast := forecasts["P1"]
	// AAA is plain terrain (rations = 3), well above the size-1 army's
	// demand (1): a drawn but unresolved bad harvest must not suppress it.
	if forecast.TotalDemand != 1 || len(forecast.ArmiesAtRisk) != 0 {
		t.Fatalf("forecast = %#v, want the normal unsuppressed ration production, no risk", forecast)
	}
	if got := state.TerritoryStates["AAA"].Resources; got != 0 {
		t.Errorf("input state was mutated: AAA stock = %d, want 0", got)
	}
}

// TestForecastFamineRiskNilInWinter checks that the forecast is never
// computed in winter, since ravitaillement itself never happens then.
func TestForecastFamineRiskNilInWinter(t *testing.T) {
	state := winterTestState(t, []models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	validateTestState(t, state)

	if forecasts := ForecastFamineRisk(state, testBalance()); forecasts != nil {
		t.Fatalf("forecasts = %#v, want nil in winter", forecasts)
	}
}
