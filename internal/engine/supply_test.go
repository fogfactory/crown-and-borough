package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestArmyCostAndRationDistribution(t *testing.T) {
	for _, test := range []struct {
		size int
		want int
	}{
		{size: 1, want: 1},
		{size: 2, want: 2},
		{size: 3, want: 4},
		{size: 4, want: 8},
	} {
		if got := armyCost(test.size, testBalance().CostBase); got != test.want {
			t.Errorf("armyCost(%d) = %d, want %d", test.size, got, test.want)
		}
	}

	armies := []models.Army{
		{ID: "A1", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		{ID: "A2", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
		{ID: "A3", OwnerID: "P3", TerritoryID: "CCC", Size: 2},
	}
	received := distributeRations(2, armies)
	if !reflect.DeepEqual(received, map[models.ArmyID]int{"A2": 1, "A3": 1}) {
		t.Errorf("ration distribution = %#v, want A3 then A2", received)
	}

	original := map[models.TerritoryID]int{"AAA": 1}
	cloned := cloneRations(original)
	original["AAA"] = 2
	if cloned["AAA"] != 1 {
		t.Errorf("cloneRations retained source map: %#v", cloned)
	}
}

func TestResolveSupplyRationsAndEvents(t *testing.T) {
	for _, test := range []struct {
		terrain    models.Terrain
		wantFamine bool
		name       string
	}{
		{terrain: models.TerrainPlain, name: "plain"},
		{terrain: models.TerrainForest, name: "forest"},
		{terrain: models.TerrainHill, name: "hill"},
		{terrain: models.TerrainMountain, wantFamine: true, name: "mountain"},
		{terrain: models.TerrainSwamp, wantFamine: true, name: "swamp"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := testState(t,
				[]models.Territory{supplyTerritory("AAA", "AAA", test.terrain)},
				[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
			)
			validateTestState(t, state)

			resolution, err := Resolve(state, testBalance())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got := hasFamineEvent(resolution.Events, "A1"); got != test.wantFamine {
				t.Errorf("famine = %t, want %t", got, test.wantFamine)
			}
			if test.wantFamine {
				event := famineEventForArmy(t, resolution.Events, "A1")
				if event.TroopsLost != 0 {
					t.Errorf("famine event = %#v, want no loss at the one-troop minimum", event)
				}
				if army := armyByID(t, resolution.State, "A1"); army.Size != 1 {
					t.Errorf("A1 size = %d, want the one-troop minimum", army.Size)
				}
			}
			if got := resolution.State.TerritoryStates["AAA"].Resources; got != 0 {
				t.Errorf("resources = %d, want rations never to change stock", got)
			}
		})
	}

	t.Run("neutral village stores production without supplying an army", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)
		clearTerritoryOwner(state, "AAA")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want no famine event", resolution.Events)
		}
		event := supplyEventForSource(t, resolution.Events, "AAA")
		if event.OwnerID != "" || event.Production != 1 || event.Demand != 0 || event.StockAfter != 1 {
			t.Errorf("neutral supply event = %#v, want neutral production and stock", event)
		}
		if got := resolution.State.TerritoryStates["AAA"].Resources; got != 1 {
			t.Errorf("neutral village resources = %d, want 1", got)
		}
	})

	t.Run("neutral supply event reflects auto-pillage in the same phase", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
		)
		clearTerritoryOwner(state, "AAA")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := supplyEventForSource(t, resolution.Events, "AAA")
		if event.StockAfter != 0 {
			t.Errorf("neutral supply event stock = %d, want 0 after auto-pillage", event.StockAfter)
		}
		if !hasFamineEvent(resolution.Events, "A1") || ctxInfrastructurePresent(resolution.State, "I1") {
			t.Errorf("events/state = %#v/%#v, want famine and destroyed village", resolution.Events, resolution.State)
		}
	})

	t.Run("assigned rations are reported by source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := supplyEventForSource(t, resolution.Events, "BBB")
		if event.Production != 1 || event.Demand != 1 || !reflect.DeepEqual(event.Rations, map[models.TerritoryID]int{"AAA": 1}) || event.StockConsumed != 0 {
			t.Errorf("supply event = %#v, want production 1, demand 1, and one ration at AAA", event)
		}
		if hasFamineEvent(resolution.Events, "A1") {
			t.Error("A1 should be supplied after its local ration")
		}
	})
}

func TestResolveSupplyProductionAndStocks(t *testing.T) {
	t.Run("controlled source receives only connected mill production", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "DDD"),
				supplyTerritory("DDD", "DDD", models.TerrainPlain, "CCC"),
			},
			nil,
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "CCC"})
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeMill, Level: 3, TerritoryID: "DDD"})
		neutralState := state.TerritoryStates["CCC"]
		neutralState.Resources = 7
		state.TerritoryStates["CCC"] = neutralState
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got := resolution.State.TerritoryStates["AAA"].Resources; got != 3 {
			t.Errorf("controlled castle stock = %d, want base 1 plus adjacent mill level 2", got)
		}
		if got := resolution.State.TerritoryStates["CCC"].Resources; got != 11 {
			t.Errorf("neutral village stock = %d, want persisted 7 plus base and mill production", got)
		}
		event := supplyEventForSource(t, resolution.Events, "AAA")
		if event.Production != 3 || event.Demand != 0 {
			t.Errorf("source event = %#v, want production 3 and no demand", event)
		}
		neutralEvent := supplyEventForSource(t, resolution.Events, "CCC")
		if neutralEvent.OwnerID != "" || neutralEvent.Production != 4 || neutralEvent.StockAfter != 11 {
			t.Errorf("neutral event = %#v, want base plus adjacent mill production", neutralEvent)
		}
		if len(supplyEvents(resolution.Events)) != 2 {
			t.Errorf("supply events = %#v, want controlled and neutral village events", resolution.Events)
		}
	})

	t.Run("global deficit consumes the smallest stock then territory ID", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "BBB", models.TerrainPlain, "CCC"),
				supplyTerritory("BBB", "AAA", models.TerrainPlain, "DDD"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "AAA"),
				supplyTerritory("DDD", "DDD", models.TerrainPlain, "BBB"),
				supplyTerritory("EEE", "EEE", models.TerrainPlain),
				supplyTerritory("FFF", "FFF", models.TerrainPlain),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 2},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "EEE"})
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "FFF"})
		setTerritoryResources(state, "AAA", 1)
		setTerritoryResources(state, "BBB", 1)
		setTerritoryResources(state, "EEE", 7)
		setTerritoryResources(state, "FFF", 9)
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if got := resolution.State.TerritoryStates["AAA"].Resources; got != 0 {
			t.Errorf("AAA stock = %d, want 0 because AAA is consumed first on tie", got)
		}
		if got := resolution.State.TerritoryStates["BBB"].Resources; got != 1 {
			t.Errorf("BBB stock = %d, want 1", got)
		}
		if got := resolution.State.TerritoryStates["EEE"].Resources; got != 8 {
			t.Errorf("neutral stock = %d, want persisted 7 plus production", got)
		}
		if got := resolution.State.TerritoryStates["FFF"].Resources; got != 10 {
			t.Errorf("neutral non-source stock = %d, want persisted 9 plus production", got)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.StockConsumed != 1 {
			t.Errorf("AAA stock consumed = %d, want 1", event.StockConsumed)
		}
		if event := supplyEventForSource(t, resolution.Events, "BBB"); event.StockConsumed != 0 {
			t.Errorf("BBB stock consumed = %d, want 0", event.StockConsumed)
		}
		if hasFamineEvent(resolution.Events, "A1") || hasFamineEvent(resolution.Events, "A2") {
			t.Errorf("events = %#v, want stocks to cover the deficit", resolution.Events)
		}
	})

	t.Run("neutral village accumulates production on every action turn", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
			nil,
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		first, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("first Resolve: %v", err)
		}
		second, err := Resolve(first.State, testBalance())
		if err != nil {
			t.Fatalf("second Resolve: %v", err)
		}
		if got := first.State.TerritoryStates["AAA"].Resources; got != 1 {
			t.Errorf("first neutral stock = %d, want 1", got)
		}
		if got := second.State.TerritoryStates["AAA"].Resources; got != 2 {
			t.Errorf("second neutral stock = %d, want 2", got)
		}
	})

	t.Run("neutral village resolution is deterministic", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			nil,
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "BBB"})
		setTerritoryResources(state, "AAA", 7)
		validateTestState(t, state)

		first, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("first Resolve: %v", err)
		}
		second, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("second Resolve: %v", err)
		}
		if !reflect.DeepEqual(first.State, second.State) || !reflect.DeepEqual(first.Events, second.Events) {
			t.Fatalf("neutral village resolutions differ:\nfirst=%#v\nsecond=%#v", first, second)
		}
	})
}

func TestNeutralVillageCapturePreservesStockAndDelaysSupply(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I0", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I0")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "BBB"})
	setTerritoryResources(state, "BBB", 3)
	addChain(t, state, "A1", "N1", models.Order{
		Type:       models.OrderTypeAttack,
		PositionID: "AAA",
		TargetIDs:  []models.TerritoryID{"BBB"},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve capture: %v", err)
	}
	target := resolution.State.TerritoryStates["BBB"]
	if target.OwnerID == nil || *target.OwnerID != "P1" {
		t.Errorf("captured village owner = %v, want P1", target.OwnerID)
	}
	if target.Resources != 4 {
		t.Errorf("captured village stock = %d, want preloaded 3 plus neutral production", target.Resources)
	}
	neutralEvent := supplyEventForSource(t, resolution.Events, "BBB")
	if neutralEvent.OwnerID != "" || neutralEvent.Demand != 0 || neutralEvent.StockAfter != 4 {
		t.Errorf("capture-turn supply event = %#v, want neutral stock before capture", neutralEvent)
	}

	next, err := Resolve(resolution.State, testBalance())
	if err != nil {
		t.Fatalf("Resolve after capture: %v", err)
	}
	controlledEvent := supplyEventForSource(t, next.Events, "BBB")
	if controlledEvent.OwnerID != "P1" || controlledEvent.Production != 1 || controlledEvent.StockAfter != 5 {
		t.Errorf("post-capture supply event = %#v, want controlled production on next turn", controlledEvent)
	}
	if got := next.State.TerritoryStates["BBB"].Resources; got != 5 {
		t.Errorf("post-capture village stock = %d, want 5", got)
	}
	if got := next.State.TerritoryStates["AAA"].Resources; got != 2 {
		t.Errorf("capital stock before winter = %d, want 2", got)
	}

	winter := cloneGameState(next.State)
	winter.Turn = 4
	winter.Season = models.SeasonWinter
	addNoble(winter, "N2", "TWO", "P1", "AAA")
	winterResolution, err := ResolveWinter(winter, testBalance(), map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "W1", Type: models.WinterOrderTypeRecruitTroop, TerritoryID: "BBB"}},
	})
	if err != nil {
		t.Fatalf("ResolveWinter after capture: %v", err)
	}
	if got := winterResolution.State.TerritoryStates["BBB"].Resources; got != 1 {
		t.Errorf("captured village stock after winter = %d, want 1 after payment and conservation", got)
	}
	if got := winterResolution.State.TerritoryStates["AAA"].Resources; got != 2 {
		t.Errorf("capital stock after winter = %d, want 2 after repatriation", got)
	}
}

func TestResolveSupplyNetworks(t *testing.T) {
	t.Run("range reaches three edges but no farther without a depot", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB", "DDD"),
				supplyTerritory("DDD", "DDD", models.TerrainMountain, "CCC", "EEE"),
				supplyTerritory("EEE", "EEE", models.TerrainMountain, "DDD"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "EEE", Size: 1},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 1 {
			t.Errorf("source demand = %d, want only the range-three army", event.Demand)
		}
		if hasFamineEvent(resolution.Events, "A1") || !hasFamineEvent(resolution.Events, "A2") {
			t.Errorf("events = %#v, want only out-of-range A2 to enter famine", resolution.Events)
		}
	})

	t.Run("source ties use territory ID order", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("ZZZ", "ZZZ", models.TerrainPlain, "MMM"),
				supplyTerritory("MMM", "MMM", models.TerrainMountain, "ZZZ", "AAA"),
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "MMM"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "MMM", Size: 1}},
		)
		setTerritoryOwner(state, "ZZZ", "P1")
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		events := supplyEvents(resolution.Events)
		if len(events) != 2 || events[0].SourceID != "AAA" || events[1].SourceID != "ZZZ" {
			t.Fatalf("supply event order = %#v, want AAA then ZZZ", events)
		}
		if events[0].Demand != 1 || events[1].Demand != 0 {
			t.Errorf("source demands = %#v, want source AAA to feed A1", events)
		}
	})

	t.Run("controlled depots extend and cumulatively relay a source", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB", "DDD"),
				supplyTerritory("DDD", "DDD", models.TerrainPlain, "CCC", "EEE"),
				supplyTerritory("EEE", "EEE", models.TerrainPlain, "DDD", "FFF"),
				supplyTerritory("FFF", "FFF", models.TerrainPlain, "EEE", "GGG"),
				supplyTerritory("GGG", "GGG", models.TerrainPlain, "FFF", "HHH"),
				supplyTerritory("HHH", "HHH", models.TerrainMountain, "GGG"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "HHH", Size: 1}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "DDD", "P1")
		setTerritoryOwner(state, "FFF", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "DDD"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "FFF"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 1 {
			t.Errorf("source demand = %d, want relay network to reach HHH", event.Demand)
		}
		if hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want A1 supplied through both depots", resolution.Events)
		}
	})

	t.Run("enemy territory blocks the supply network", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 1}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 0 {
			t.Errorf("blocked source demand = %d, want 0", event.Demand)
		}
		if !hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want direct famine behind enemy territory", resolution.Events)
		}
	})
}

func TestResolveSupplyIsolatedByOwner(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB", "CCC"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain, "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	clearTerritoryOwner(state, "BBB")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 1 {
		t.Errorf("source demand = %d, want only P1's army", event.Demand)
	}
	if hasFamineEvent(resolution.Events, "A1") || !hasFamineEvent(resolution.Events, "A2") {
		t.Errorf("events = %#v, want P1 supplied and P2 unfed despite neutral reachability", resolution.Events)
	}
}

func TestResolveSupplyFamineAndAutoPillage(t *testing.T) {
	t.Run("direct famine can pillage, recover, and credit a controlled settlement", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
		)
		setTerritoryOwner(state, "BBB", "P2")
		setTerritoryOwner(state, "CCC", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := famineEventForArmy(t, resolution.Events, "A1")
		if !event.SavedByPillage || event.InfrastructureID != "I1" || event.InfrastructureType != models.InfraTypeMill || event.ResourceCredit != 1 || event.CreditTerritoryID != "CCC" {
			t.Errorf("famine event = %#v, want saved mill pillage credited to CCC", event)
		}
		if len(resolution.State.Infrastructures) != 1 || resolution.State.Infrastructures[0].ID != "I2" {
			t.Errorf("infrastructures = %#v, want only the castle left", resolution.State.Infrastructures)
		}
		if got := resolution.State.TerritoryStates["CCC"].Resources; got != 2 {
			t.Errorf("credited source resources = %d, want local production plus pillage credit", got)
		}
		if event := supplyEventForSource(t, resolution.Events, "CCC"); event.StockAfter != 2 {
			t.Errorf("credited source event stock = %d, want 2", event.StockAfter)
		}
	})

	t.Run("the farthest assigned army is evaluated first and can recover at zero gain", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB", "DDD"),
				supplyTerritory("DDD", "DDD", models.TerrainMountain, "CCC"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 2},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 1, TerritoryID: "DDD"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := famineEventForArmy(t, resolution.Events, "A2")
		if !event.SavedByPillage || event.ResourceCredit != 0 || event.InfrastructureID != "I3" || event.SourceID != "AAA" {
			t.Errorf("A2 famine event = %#v, want zero-gain recovery through AAA", event)
		}
		if hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want A2 to consume all remaining deficit before A1", resolution.Events)
		}
		if ctxInfrastructurePresent(resolution.State, "I3") {
			t.Error("auto-pillage should remove I3")
		}
	})

	t.Run("negative pillage gain starves a size-three army down to two", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "AAA"})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := famineEventForArmy(t, resolution.Events, "A1")
		if event.SavedByPillage || event.InfrastructureID != "I1" || event.InfrastructureType != models.InfraTypeMill || event.Troops != 3 || event.TroopsLost != 1 {
			t.Errorf("famine event = %#v, want an unsaved three-troop army with one troop lost", event)
		}
		if army := armyByID(t, resolution.State, "A1"); army.Size != 2 {
			t.Errorf("A1 size = %d, want 2 after one famine turn", army.Size)
		}
		report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
		if len(report.Famines) != 1 || report.Famines[0].Troops != 3 || report.Famines[0].TroopsLost != 1 {
			t.Errorf("famine report = %#v, want initial size 3 and one lost troop", report.Famines)
		}
		if len(resolution.State.Infrastructures) != 0 {
			t.Errorf("infrastructures = %#v, want auto-pillage to remove I1", resolution.State.Infrastructures)
		}
	})
}

func TestResolveSupplyFamineEventOrder(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	famines := famineEvents(resolution.Events)
	if len(famines) != 2 || famines[0].ArmyID != "A2" || famines[1].ArmyID != "A1" {
		t.Errorf("famine event order = %#v, want direct famine A2 before assigned famine A1", famines)
	}
}

func TestResolveAssignedFamineTieBreaksAndHasZeroStrength(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("ZZZ", "ZZZ", models.TerrainPlain, "BBB", "AAA", "DDD"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "ZZZ"),
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "ZZZ", "CCC"),
			supplyTerritory("DDD", "DDD", models.TerrainPlain, "ZZZ"),
			supplyTerritory("CCC", "CCC", models.TerrainPlain, "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
			{ID: "A2", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
			{ID: "A3", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		},
	)
	setTerritoryOwner(state, "ZZZ", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "DDD"})
	addNoble(state, "N2", "TWO", "P1", "AAA")
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"CCC"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	event := famineEventForArmy(t, resolution.Events, "A2")
	if event.SavedByPillage || event.SourceID != "ZZZ" {
		t.Errorf("A2 famine event = %#v, want an unsaved source-assigned famine", event)
	}
	if hasFamineEvent(resolution.Events, "A1") {
		t.Errorf("events = %#v, want AAA selected before BBB", resolution.Events)
	}
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" {
		t.Errorf("A2 = %+v, want zero-strength attack to lose", army)
	}
	if got := combatContenderForce(t, resolution.Events, "CCC", "A2"); got != 0 {
		t.Errorf("A2 attack force = %d, want source-assigned famine to have zero strength", got)
	}
}

func TestResolveFamineCombatEffects(t *testing.T) {
	t.Run("one-troop swamp army has zero famine force", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainSwamp, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{
			Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		famine := famineEventForArmy(t, resolution.Events, "A1")
		if famine.Troops != 1 || famine.TroopsLost != 0 || !hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("famine event = %#v, want one troop, no physical loss, and famine", famine)
		}
		if army := armyByID(t, resolution.State, "A1"); army.Size != 1 || army.TerritoryID != "AAA" {
			t.Errorf("A1 = %+v, want one troop remaining at AAA", army)
		}
		if got := combatContenderForce(t, resolution.Events, "BBB", "A1"); got != 0 {
			t.Errorf("A1 attack force = %d, want famine force 0", got)
		}
	})

	t.Run("famine removes attack strength", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 1 {
			t.Errorf("A1 = %+v, want a zero-strength attack and one lost troop", army)
		}
		if got := combatContenderForce(t, resolution.Events, "BBB", "A1"); got != 0 {
			t.Errorf("A1 combat force = %d, want 0", got)
		}
	})

	t.Run("famine removes defense strength but preserves retreat", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB", "CCC"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "AAA"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N2", "TWO", "P2", "BBB")
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "CCC" || army.Size != 1 {
			t.Errorf("A1 = %+v, want normal retreat after one lost troop", army)
		}
		if got := combatContenderForce(t, resolution.Events, "AAA", "A1"); got != 0 {
			t.Errorf("A1 defense force = %d, want 0", got)
		}
	})

	t.Run("famished support remains valid but adds no force", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB", "CCC"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "AAA", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
				{ID: "A3", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addNoble(state, "N2", "TWO", "P1", "CCC")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want the free noble's bonus to win", army)
		}
		if got := combatContenderForce(t, resolution.Events, "BBB", "A1"); got != 2 {
			t.Errorf("A1 attack force = %d, want army plus noble bonus while famished support contributes nothing", got)
		}
	})

	t.Run("zero-strength attack can move to an empty territory", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 {
			t.Errorf("A1 = %+v, want famished army to move after losing one troop", army)
		}
	})

	t.Run("famished join remains a peaceful movement", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeJoin, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Size != 1 {
			t.Errorf("A1 = %+v, want famished join to move after losing one troop", army)
		}
	})

	t.Run("famished dispersion still executes", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{
			Type:             models.OrderTypeDisperse,
			PositionID:       "AAA",
			TargetIDs:        []models.TerritoryID{"AAA", "BBB"},
			NobleAssignments: map[models.TerritoryID][]models.NobleCode{"AAA": {"ONE"}},
		})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "AAA" || army.Size != 1 {
			t.Errorf("carrier = %+v, want the famine minimum at AAA", army)
		}
		if hasArmy(resolution.State, "A2") {
			t.Error("famine attrition should leave only one troop to disperse")
		}
	})

	t.Run("moving onto infrastructure does not auto-pillage it", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "BBB"})
		addNoble(state, "N1", "ONE", "P1", "AAA")
		addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want movement onto BBB", army)
		}
		if !ctxInfrastructurePresent(resolution.State, "I1") {
			t.Error("I1 should survive because auto-pillage only uses the start position")
		}
	})
}

func TestResolveSupplyIsPureAndDeterministic(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 1},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "CCC"})
	validateTestState(t, state)
	before := cloneGameState(state)

	first, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	second, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("Resolve mutated its input during supply")
	}
	if !reflect.DeepEqual(first.State, second.State) || !reflect.DeepEqual(first.Events, second.Events) {
		t.Fatalf("supply resolutions differ:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if event := supplyEventForSource(t, first.Events, "AAA"); !reflect.DeepEqual(event.Rations, map[models.TerritoryID]int{"BBB": 1}) {
		t.Errorf("supply event rations = %#v, want BBB ration", event.Rations)
	}
	if event := famineEventForArmy(t, first.Events, "A2"); !event.SavedByPillage || event.InfrastructureID != "I2" {
		t.Errorf("A2 famine event = %#v, want saved auto-pillage", event)
	}
}

func supplyTerritory(id, code string, terrain models.Terrain, neighbors ...models.TerritoryID) models.Territory {
	territory := territory(id, code, neighbors...)
	territory.Terrain = terrain
	return territory
}

func setTerritoryOwner(state *models.GameState, territoryID models.TerritoryID, ownerID models.PlayerID) {
	territoryState := state.TerritoryStates[territoryID]
	owner := ownerID
	territoryState.OwnerID = &owner
	state.TerritoryStates[territoryID] = territoryState
}

func clearTerritoryOwner(state *models.GameState, territoryID models.TerritoryID) {
	territoryState := state.TerritoryStates[territoryID]
	territoryState.OwnerID = nil
	state.TerritoryStates[territoryID] = territoryState
}

func setTerritoryResources(state *models.GameState, territoryID models.TerritoryID, resources int) {
	territoryState := state.TerritoryStates[territoryID]
	territoryState.Resources = resources
	state.TerritoryStates[territoryID] = territoryState
}

func supplyEvents(events []Event) []Event {
	result := make([]Event, 0)
	for _, event := range events {
		if event.Type == EventTypeSupply {
			result = append(result, event)
		}
	}
	return result
}

func famineEvents(events []Event) []Event {
	result := make([]Event, 0)
	for _, event := range events {
		if event.Type == EventTypeFamine {
			result = append(result, event)
		}
	}
	return result
}

func supplyEventForSource(t *testing.T, events []Event, sourceID models.TerritoryID) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeSupply && event.SourceID == sourceID {
			return event
		}
	}
	t.Fatalf("missing supply event for %q in %#v", sourceID, events)
	return Event{}
}

func famineEventForArmy(t *testing.T, events []Event, armyID models.ArmyID) Event {
	t.Helper()
	for _, event := range events {
		if event.Type == EventTypeFamine && event.ArmyID == armyID {
			return event
		}
	}
	t.Fatalf("missing famine event for %q in %#v", armyID, events)
	return Event{}
}

func hasFamineEvent(events []Event, armyID models.ArmyID) bool {
	for _, event := range events {
		if event.Type == EventTypeFamine && event.ArmyID == armyID {
			return true
		}
	}
	return false
}

func ctxInfrastructurePresent(state *models.GameState, infrastructureID models.InfraID) bool {
	for _, infrastructure := range state.Infrastructures {
		if infrastructure.ID == infrastructureID {
			return true
		}
	}
	return false
}

func combatContenderForce(t *testing.T, events []Event, territoryID models.TerritoryID, armyID models.ArmyID) int {
	t.Helper()
	for _, event := range events {
		if event.Type != EventTypeCombat || event.TerritoryID != territoryID {
			continue
		}
		for _, contender := range event.Contenders {
			if contender.ArmyID == armyID {
				return contender.Force
			}
		}
	}
	t.Fatalf("missing combat contender %q at %q in %#v", armyID, territoryID, events)
	return 0
}
