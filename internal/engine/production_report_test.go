package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func productionLineFor(t *testing.T, report TurnReport, territoryID models.TerritoryID) ProductionReport {
	t.Helper()
	for _, line := range report.Production {
		if line.Territory == territoryID {
			return line
		}
	}
	t.Fatalf("production lines = %#v, want a line for %q", report.Production, territoryID)
	return ProductionReport{}
}

func consumptionLineFor(t *testing.T, report TurnReport, armyID models.ArmyID) ConsumptionReport {
	t.Helper()
	for _, line := range report.Consumption {
		if line.Army == armyID {
			return line
		}
	}
	t.Fatalf("consumption lines = %#v, want a line for %q", report.Consumption, armyID)
	return ConsumptionReport{}
}

func TestProductionReportBreaksDownSourceAndBonus(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "BBB"})
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	validateTestState(t, state)
	balance := testBalance()
	resolution, err := ResolveWithDeckOrders(state, balance, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindFairWeather, RegionSeed: "AAA"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	line := productionLineFor(t, report, "AAA")
	if line.TerrainRations != 3 || line.BonusRations != 0 {
		t.Fatalf("ration breakdown = %#v, want terrain 3 without fair weather bonus", line)
	}
	if line.BaseProduction != 0 || line.MillProduction != 2 || line.BonusProduction != 2 {
		t.Fatalf("production breakdown = %#v, want no base production, mills 2 doubled by fair weather", line)
	}
	if line.Produced != 7 {
		t.Fatalf("produced = %d, want 7", line.Produced)
	}
	consumption := consumptionLineFor(t, report, "A1")
	if consumption.Demand != 1 || consumption.ReceivedLocal != 1 || consumption.ReceivedTransfer != 0 || consumption.Missing != 0 {
		t.Fatalf("A1 consumption = %#v, want demand 1 fully covered locally", consumption)
	}
}

func TestConsumptionReportSplitsLocalAndTransfer(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}}}
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	home := consumptionLineFor(t, report, "A1")
	if home.Demand != 1 || home.ReceivedLocal != 1 || home.ReceivedTransfer != 0 {
		t.Fatalf("A1 consumption = %#v, want demand 1 covered locally", home)
	}
	remote := consumptionLineFor(t, report, "A2")
	if remote.Demand != 2 || remote.ReceivedLocal != 1 || remote.ReceivedTransfer != 1 || remote.TotalReceived != 2 || remote.Missing != 0 {
		t.Fatalf("A2 consumption = %#v, want demand 2 split into local 1 and transfer 1", remote)
	}
	if remote.Source != "AAA" {
		t.Fatalf("A2 source = %q, want AAA", remote.Source)
	}
	source := productionLineFor(t, report, "AAA")
	if len(source.SentToRations) != 1 || source.SentToRations["CCC"] != 1 {
		t.Fatalf("AAA sent rations = %#v, want 1 ration dispatched to CCC", source.SentToRations)
	}
}

func TestProductionReportTracesDispatchToMultipleArmies(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA", "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 3},
			{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 3, TerritoryID: "BBB"})
	state.Regions = []models.Region{{ID: "AAA", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}}}
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	source := productionLineFor(t, report, "AAA")
	if source.BaseProduction != 0 || source.MillProduction != 3 || source.Produced != 3 {
		t.Fatalf("AAA production = %#v, want no base production, mills 3", source)
	}
	if len(source.SentToRations) != 2 || source.SentToRations["BBB"] != 3 || source.SentToRations["CCC"] != 1 {
		t.Fatalf("AAA dispatch = %#v, want 3 rations to BBB and 1 to CCC", source.SentToRations)
	}
	first := consumptionLineFor(t, report, "A1")
	if first.Source != "AAA" || first.ReceivedLocal != 1 || first.ReceivedTransfer != 3 || first.Missing != 0 {
		t.Fatalf("A1 consumption = %#v, want 1 local plus 3 from AAA without famine", first)
	}
	second := consumptionLineFor(t, report, "A2")
	if second.Source != "AAA" || second.ReceivedLocal != 1 || second.ReceivedTransfer != 1 || second.Missing != 0 {
		t.Fatalf("A2 consumption = %#v, want 1 local plus 1 from AAA without famine", second)
	}
}

func TestProductionReportShowsFamineSuppression(t *testing.T) {
	state := effectTestState()
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "BBB"})
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}}
	aaaState := state.TerritoryStates["AAA"]
	aaaState.Army = armyPointer("A1")
	state.TerritoryStates["AAA"] = aaaState
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "BBB", "P1")
	setCurrentCalamity(state, models.CardKindFamine, "AAA")
	ctx := newResolutionContext(state, testBalance())
	resolveSeasonEffects(ctx)
	resolveSupply(ctx)
	report := BuildTurnReport(state, ctx.state, ctx.events, nil)
	line := productionLineFor(t, report, "AAA")
	if line.TerrainRations != 0 || line.SuppressedRations != 3 {
		t.Fatalf("ration suppression = %#v, want all 3 terrain rations suppressed", line)
	}
	if line.BaseProduction != 0 || line.MillProduction != 2 || line.SuppressedProduction != 0 {
		t.Fatalf("production suppression = %#v, want the mill intact and no base production line", line)
	}
	if line.Produced != 2 {
		t.Fatalf("produced = %d, want the mill's 2 R only", line.Produced)
	}
	consumption := consumptionLineFor(t, report, "A1")
	if consumption.Famine || consumption.Demand != 1 || consumption.ReceivedLocal != 0 || consumption.ReceivedTransfer != 1 {
		t.Fatalf("A1 consumption = %#v, want demand 1 covered by the castle stock", consumption)
	}
}
