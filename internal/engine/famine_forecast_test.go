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
	// armyCost(2, 2) = 2, mountain terrain rations = 1: with no reachable
	// source at all, A1 draws nothing from the network (net consumption 0)
	// and starves outright for the full remaining demand of 1.
	if forecast.NetConsumption != 0 || len(forecast.ArmiesAtRisk) != 1 {
		t.Fatalf("forecast = %#v, want A1 flagged at risk", forecast)
	}
	risk := forecast.ArmiesAtRisk[0]
	if risk.ArmyID != "A1" || risk.TerritoryID != "AAA" || risk.Size != 2 || risk.Deficit != 1 {
		t.Errorf("risk = %#v, want A1 at AAA with a deficit of 1", risk)
	}
}

// TestForecastFamineRiskNoRiskWithSufficientLocalProduction covers the
// issue's second hotseat scenario: an army whose own territory produces
// enough rations is never flagged, and contributes nothing to the projected
// consumption since none of its demand is drawn from stock or the network.
func TestForecastFamineRiskNoRiskWithSufficientLocalProduction(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	validateTestState(t, state)

	forecasts := ForecastFamineRisk(state, testBalance())
	forecast := forecasts["P1"]
	// armyCost(1, 2) = 1, plain terrain rations = 3, well above demand: 0 is
	// drawn from stock or the network.
	if forecast.NetConsumption != 0 || len(forecast.ArmiesAtRisk) != 0 {
		t.Fatalf("forecast = %#v, want no army at risk", forecast)
	}
}

// TestForecastFamineRiskCountsReachableUncontestedSource checks that an
// army short on local production is not flagged when it is the only one
// drawing on a controlled source that covers its remaining deficit.
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
	// test is about reachable-source coverage, not income.
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0

	forecasts := ForecastFamineRisk(state, balance)
	forecast := forecasts["P1"]
	// armyCost(3, 2) = 4, mountain terrain rations = 1: the remaining 3 is
	// what will be drawn from BBB's banked stock (5), reachable within
	// supply range, so it is not a deficit.
	if forecast.NetConsumption != 3 || len(forecast.ArmiesAtRisk) != 0 {
		t.Fatalf("forecast = %#v, want A1 covered by BBB's reachable stock", forecast)
	}
}

// TestForecastFamineRiskFlagsSharedSourceContention checks that when two
// armies each reach the same source, and each looks fine considered alone,
// but the source cannot feed both, at least one of them is correctly
// flagged: the forecast must not silently under-report the risk by crediting
// the same stock to every claimant.
func TestForecastFamineRiskFlagsSharedSourceContention(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "CCC"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain, "AAA", "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 3},
		},
	)
	setTerritoryOwner(state, "CCC", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "CCC"})
	setTerritoryResources(state, "CCC", 3)
	validateTestState(t, state)

	balance := testBalance()
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0

	forecasts := ForecastFamineRisk(state, balance)
	forecast := forecasts["P1"]
	// armyCost(3, 2) = 4, mountain terrain rations = 1: each army needs 3
	// from the network. Considered alone, either looks covered by CCC's
	// stock of 3 - but together they need 6, CCC's stock only covers 3, and
	// the tie-break (equal distance and size) picks AAA to starve first.
	if forecast.NetConsumption != 3 || len(forecast.ArmiesAtRisk) != 1 {
		t.Fatalf("forecast = %#v, want exactly one army flagged", forecast)
	}
	risk := forecast.ArmiesAtRisk[0]
	if risk.ArmyID != "A1" || risk.TerritoryID != "AAA" || risk.Deficit != 3 {
		t.Errorf("risk = %#v, want A1 at AAA with a deficit of 3", risk)
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
	// demand (1): a drawn but unresolved bad harvest must not suppress it,
	// so nothing is drawn from stock or the network.
	if forecast.NetConsumption != 0 || len(forecast.ArmiesAtRisk) != 0 {
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
