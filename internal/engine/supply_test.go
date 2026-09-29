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
	received := distributeRations(2, armies, testBalance().CostBase)
	if !reflect.DeepEqual(received, map[models.ArmyID]int{"A3": 2}) {
		t.Errorf("ration distribution = %#v, want A3 to receive its full demand first", received)
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
		{terrain: models.TerrainMountain, name: "mountain"},
		{terrain: models.TerrainSwamp, name: "swamp"},
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
		// A1 stands on a separate, self-sufficient plain territory rather
		// than on AAA itself: an army standing directly on AAA would capture
		// it before ravitaillement now runs (control resolves first, #208),
		// which is exactly what the capture-feeds-the-same-turn scenario
		// below is about, not this one.
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 1}},
		)
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
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3}},
		)
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
		// No army garrisons BBB: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can use it as a source.
		setCapital(state, "P1", "I1")
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		event := supplyEventForSource(t, resolution.Events, "BBB")
		// BBB has no mill, so its mill-only production is 0; its territory
		// income (from itself and AAA, both routed to BBB, the only castle)
		// is credited to stock beforehand and covers A1's deficit instead.
		if event.Production != 0 || event.Demand != 1 || !reflect.DeepEqual(event.Rations, map[models.TerritoryID]int{"AAA": 3}) || event.StockConsumed != 1 {
			t.Errorf("supply event = %#v, want no mill production, demand 1, three local rations at AAA, and stock consumed 1", event)
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
			// P1's army holds BBB's mill: outside every fief and capital, a
			// mill only produces while occupied (#215). Its local rations (3,
			// plain terrain) comfortably cover its own demand (1), so it never
			// starves and never pillages its own mill.
			[]models.Army{{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 1}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 2, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "CCC"})
		// DDD's mill is never claimed: outside every fief and capital, with no
		// army on it, it is inert (#215) and no longer grandfathers production
		// into CCC's neutral village.
		addInfrastructure(state, models.Infrastructure{ID: "I4", Type: models.InfraTypeMill, Level: 5, TerritoryID: "DDD"})
		// No army garrisons AAA: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can use it as a source.
		setCapital(state, "P1", "I1")
		neutralState := state.TerritoryStates["CCC"]
		neutralState.Resources = 7
		state.TerritoryStates["CCC"] = neutralState
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		// AAA's own territory income (1) plus BBB's (1, BBB has no
		// settlement of its own so it flows to AAA, its closest controlled
		// castle) plus BBB's adjacent mill level 2.
		if got := resolution.State.TerritoryStates["AAA"].Resources; got != 4 {
			t.Errorf("controlled castle stock = %d, want territory income 2 plus adjacent mill level 2", got)
		}
		if got := resolution.State.TerritoryStates["CCC"].Resources; got != 8 {
			t.Errorf("neutral village stock = %d, want persisted 7 plus its own base production only (DDD's inert mill contributes nothing)", got)
		}
		event := supplyEventForSource(t, resolution.Events, "AAA")
		if event.Production != 2 || event.Demand != 0 {
			t.Errorf("source event = %#v, want mill production 2 (territory income is a separate event) and no demand", event)
		}
		neutralEvent := supplyEventForSource(t, resolution.Events, "CCC")
		if neutralEvent.OwnerID != "" || neutralEvent.Production != 1 || neutralEvent.StockAfter != 8 {
			t.Errorf("neutral event = %#v, want base production only, DDD's inert mill excluded", neutralEvent)
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
				{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 3},
				{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
				// A3 and A4 anchor BBB and AAA against control resolution's
				// unanchored release, now ahead of ravitaillement (#208):
				// neither carries a fief or a player capital (BBB is a
				// village and could not be one anyway), which the no-capital
				// fallback routing this scenario is about requires to stay
				// absent. Both are fully fed by their own local terrain
				// rations and never touch pooled stock, leaving every
				// assertion below unchanged.
				{ID: "A3", OwnerID: "P1", TerritoryID: "BBB", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
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
		// BBB is P1's own closest settlement for both itself and DDD (its
		// component has no castle), so it collects both territories' income
		// on top of its preset stock, and its lone army is fully fed by
		// local terrain rations, leaving that income untouched.
		if got := resolution.State.TerritoryStates["BBB"].Resources; got != 4 {
			t.Errorf("BBB stock = %d, want preset 1 plus territory income 3", got)
		}
		if got := resolution.State.TerritoryStates["EEE"].Resources; got != 8 {
			t.Errorf("neutral stock = %d, want persisted 7 plus production", got)
		}
		if got := resolution.State.TerritoryStates["FFF"].Resources; got != 10 {
			t.Errorf("neutral non-source stock = %d, want persisted 9 plus production", got)
		}
		// AAA's preset stock (1) plus its own and CCC's territory income (2)
		// is entirely consumed by A1's deficit at CCC.
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.StockConsumed != 3 {
			t.Errorf("AAA stock consumed = %d, want 3", event.StockConsumed)
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

// TestNeutralVillageCaptureFeedsSupplyTheSameTurn mirrors #208's acceptance
// case: a village captured this turn is already a controlled source, its
// territory income already routed to the capital, by the end of this same
// turn -- ravitaillement now resolves after control (progressChainsAndControl),
// not before it.
func TestNeutralVillageCaptureFeedsSupplyTheSameTurn(t *testing.T) {
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
	// Once captured, BBB's own stock only carries its preloaded 3: as a
	// controlled source (not a neutral one any more) its territory income
	// routes to the still-valid capital at AAA instead, this same turn.
	if target.Resources != 3 {
		t.Errorf("captured village stock = %d, want preloaded 3 unchanged", target.Resources)
	}
	capturedEvent := supplyEventForSource(t, resolution.Events, "BBB")
	if capturedEvent.OwnerID != "P1" || capturedEvent.Demand != 0 || capturedEvent.Production != 0 || capturedEvent.StockAfter != 3 {
		t.Errorf("capture-turn supply event = %#v, want P1's controlled stock, no local production", capturedEvent)
	}
	// AAA's own income (1) plus BBB's (2, territory + village), both credited
	// to the capital the same turn BBB changes hands.
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != 3 {
		t.Errorf("capital stock after capture turn = %d, want its own income 1 plus BBB's 2", got)
	}

	next, err := Resolve(resolution.State, testBalance())
	if err != nil {
		t.Fatalf("Resolve after capture: %v", err)
	}
	controlledEvent := supplyEventForSource(t, next.Events, "BBB")
	if controlledEvent.OwnerID != "P1" || controlledEvent.Production != 0 || controlledEvent.StockAfter != 3 {
		t.Errorf("post-capture supply event = %#v, want no local production once controlled", controlledEvent)
	}
	if got := next.State.TerritoryStates["BBB"].Resources; got != 3 {
		t.Errorf("post-capture village stock = %d, want unchanged 3", got)
	}
	if got := next.State.TerritoryStates["AAA"].Resources; got != 6 {
		t.Errorf("capital stock before winter = %d, want turn 1's 3 plus a second turn's own income 1 plus BBB's 2", got)
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
	if got := winterResolution.State.TerritoryStates["AAA"].Resources; got != 3 {
		t.Errorf("capital stock after winter = %d, want 3 after repatriation", got)
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
				{ID: "A1", OwnerID: "P1", TerritoryID: "DDD", Size: 2},
				{ID: "A2", OwnerID: "P1", TerritoryID: "EEE", Size: 2},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		// No army garrisons AAA: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can use it as a source.
		setCapital(state, "P1", "I1")
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
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "MMM", Size: 2},
				// A2 anchors AAA (a village, so it cannot be a capital like
				// ZZZ below) against control resolution's unanchored release,
				// now ahead of ravitaillement (#208); it is fully fed by
				// AAA's own local terrain rations and never competes for the
				// tie-break this scenario is about.
				{ID: "A2", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			},
		)
		setTerritoryOwner(state, "ZZZ", "P1")
		setTerritoryOwner(state, "AAA", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
		setCapital(state, "P1", "I1")
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
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "HHH", Size: 2},
				// A2, A3 and A4 anchor AAA, DDD and FFF against control
				// resolution's unanchored release, now ahead of
				// ravitaillement (#208), so the castle and both depots stay
				// controlled sources for the relay this scenario is about.
				// All three are fully fed by their own local terrain rations.
				{ID: "A2", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A3", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
				{ID: "A4", OwnerID: "P1", TerritoryID: "FFF", Size: 1},
			},
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

	t.Run("enemy army blocks the supply network", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		// No army garrisons AAA: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can use it as a source.
		setCapital(state, "P1", "I1")
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 0 {
			t.Errorf("blocked source demand = %d, want 0", event.Demand)
		}
		if !hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want direct famine behind enemy army", resolution.Events)
		}
	})

	t.Run("enemy territory without an army does not block the supply network", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "BBB"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2}},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P2")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		// No army garrisons AAA: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can use it as a source. BBB's own,
		// unanchored P2 ownership is released the same way, but that does not
		// change this scenario: supplyNetwork only blocks on an army, never
		// on bare ownership.
		setCapital(state, "P1", "I1")
		validateTestState(t, state)

		resolution, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if event := supplyEventForSource(t, resolution.Events, "AAA"); event.Demand != 1 {
			t.Errorf("source demand = %d, want the army beyond the enemy territory to be reached", event.Demand)
		}
		if hasFamineEvent(resolution.Events, "A1") {
			t.Errorf("events = %#v, want A1 supplied through enemy territory without an army", resolution.Events)
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
			{ID: "A1", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	clearTerritoryOwner(state, "BBB")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	// No army garrisons AAA: anchor it as P1's capital so control resolution
	// (now ahead of ravitaillement, #208) does not release it as unanchored
	// before supply can use it as a source.
	setCapital(state, "P1", "I1")
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
			[]models.Army{
				// Size 3 (not 2): A1's own presence now captures AAA before
				// ravitaillement runs (control resolves first, #208), turning
				// its mill into a legitimate self-supplied source (no
				// same-control neighbor, BBB belongs to P2) instead of the
				// inert neutral one the old turn order left it as. A size-3
				// demand still outstrips that mill's level-1 production, so
				// the deficit -- and the pillage-and-credit point -- survive.
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
				{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1},
			},
		)
		setTerritoryOwner(state, "BBB", "P2")
		setTerritoryOwner(state, "CCC", "P1")
		clearTerritoryOwner(state, "AAA")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
		// No army garrisons CCC: anchor it as P1's capital so control
		// resolution (now ahead of ravitaillement, #208) does not release it
		// as unanchored before supply can credit the pillage gain to it.
		setCapital(state, "P1", "I2")
		validateTestState(t, state)

		// This scenario is about direct famine and pillage, not territory
		// income: zeroing it out keeps CCC's pooled stock at 0, so AAA's
		// deficit (see above) cannot be covered from elsewhere in P1's
		// network before the pillage-and-credit point this test is about. A
		// higher-than-default pillage bonus is still needed to make saving a
		// size-3 army's full deficit worth a positive credit.
		balance := testBalance()
		balance.TerritoryIncome = 0
		balance.VillageIncome = 0
		balance.PillageBonus = 4
		resolution, err := Resolve(state, balance)
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
		// CCC carries no territory income in this scenario (zeroed out), so
		// its only resources are the pillage credit.
		if got := resolution.State.TerritoryStates["CCC"].Resources; got != 1 {
			t.Errorf("credited source resources = %d, want only the pillage credit", got)
		}
		if event := supplyEventForSource(t, resolution.Events, "CCC"); event.StockAfter != 1 {
			t.Errorf("credited source event stock = %d, want 1", event.StockAfter)
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
				{ID: "A2", OwnerID: "P1", TerritoryID: "DDD", Size: 3},
			},
		)
		setTerritoryOwner(state, "AAA", "P1")
		setTerritoryOwner(state, "BBB", "P1")
		addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
		addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "BBB"})
		addInfrastructure(state, models.Infrastructure{ID: "I3", Type: models.InfraTypeMill, Level: 1, TerritoryID: "DDD"})
		// AAA, BBB and CCC form a fief: BBB carries no army of its own, and
		// outside every fief and capital an unoccupied mill is inert (#215),
		// so its production needs this anchor to keep routing to AAA.
		state.Fiefs = []models.Fief{{
			ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
			Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
		}}
		validateTestState(t, state)

		balance := testBalance()
		balance.PillageBonus = 3
		// This scenario is about the famine deficit order, not territory
		// income: zeroing it out avoids AAA's, CCC's and DDD's income
		// cascading into the deficit.
		balance.TerritoryIncome = 0
		balance.VillageIncome = 0
		resolution, err := Resolve(state, balance)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		// A2's own presence now captures DDD before ravitaillement runs
		// (control resolves first, #208), turning its mill into a legitimate
		// self-supplied source (CCC carries no settlement of its own):
		// DDD -- not AAA -- becomes the closest source for both armies (0
		// for A2 standing right on it, 1 for A1 at neighboring CCC). A1 is
		// therefore now the farther-assigned army evaluated first
		// (sortAssignedFamine): it exhausts DDD's single level of mill
		// production and, with no infrastructure of its own at CCC to
		// pillage, loses a troop outright. A2, evaluated second, recovers
		// through its own mill at exactly zero gain (PillageBonus 3 against
		// its own demand 3).
		a1Event := famineEventForArmy(t, resolution.Events, "A1")
		if a1Event.SavedByPillage || a1Event.SourceID != "DDD" || a1Event.TroopsLost != 1 {
			t.Errorf("A1 famine event = %#v, want an unsaved deficit through DDD", a1Event)
		}
		a2Event := famineEventForArmy(t, resolution.Events, "A2")
		if !a2Event.SavedByPillage || a2Event.ResourceCredit != 0 || a2Event.InfrastructureID != "I3" || a2Event.SourceID != "DDD" {
			t.Errorf("A2 famine event = %#v, want zero-gain recovery through its own mill", a2Event)
		}
		if ctxInfrastructurePresent(resolution.State, "I3") {
			t.Error("auto-pillage should remove I3")
		}
	})

	t.Run("negative pillage gain starves a size-three army down to two", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
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
		var consumption *ConsumptionReport
		for index := range report.Consumption {
			if report.Consumption[index].Army == "A1" {
				consumption = &report.Consumption[index]
				break
			}
		}
		if consumption == nil || !consumption.Famine || consumption.Size != 3 || consumption.TroopsLost != 1 {
			t.Errorf("consumption report = %#v, want a famine line for A1 with initial size 3 and one lost troop", report.Consumption)
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
			{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 3},
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	// No army garrisons AAA: anchor it as P1's capital so control resolution
	// (now ahead of ravitaillement, #208) does not release it as unanchored
	// before supply can use it as a source.
	setCapital(state, "P1", "I1")
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

// TestResolveAssignedFamineTieBreaks mirrors the previous
// TestResolveAssignedFamineTieBreaksAndHasZeroStrength, minus its
// zero-strength-attack half: famine no longer weakens this same turn's
// combat (it only sets models.Army.Starving for next turn, see #208), so a
// famished attacker's force is covered by TestResolveFamineCombatEffects's
// two-turn scenario instead. A2 stays put (no order) here purely to isolate
// the assigned-famine tie-break between AAA and BBB.
func TestResolveAssignedFamineTieBreaks(t *testing.T) {
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
			// A4 holds DDD's mill: outside every fief and capital, a mill
			// only produces while occupied (#215). Its local rations (3,
			// plain terrain) cover its own demand (1), so it never competes
			// for ZZZ's pooled deficit below.
			{ID: "A4", OwnerID: "P1", TerritoryID: "DDD", Size: 1},
		},
	)
	setTerritoryOwner(state, "ZZZ", "P1")
	setTerritoryOwner(state, "DDD", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "ZZZ"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "DDD"})
	// No army garrisons ZZZ: anchor it as P1's capital so control resolution
	// (now ahead of ravitaillement, #208) does not release it as unanchored
	// before supply can use it as a source.
	setCapital(state, "P1", "I1")
	validateTestState(t, state)

	// This scenario is about the assigned-famine tie-break, not territory
	// income: zeroing it out avoids AAA's and BBB's income cascading into
	// ZZZ (the only castle) and perturbing the deficit. ZZZ's baseline
	// production instead comes from the adjacent mill at DDD, exactly
	// matching the old base-production deficit of 1.
	balance := testBalance()
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0
	resolution, err := Resolve(state, balance)
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
	if army := armyByID(t, resolution.State, "A2"); army.TerritoryID != "AAA" || army.Size != 1 {
		t.Errorf("A2 = %+v, want it to stay at AAA with one troop lost", army)
	}
}

// TestResolveFamineCombatEffects mirrors #208's F2 redesign: famine now
// resolves at the end of the turn, on the army's post-combat position, so it
// can no longer weaken this same turn's own combat. A starving army fights
// at full strength the turn it starves (its combat already resolved before
// ravitaillement runs) and only fights at strength 0 the turn after, if its
// carried-over models.Army.Starving flag is still true by then (unmet demand
// again). Every subtest below therefore spans two Resolve calls: turn 1
// starves the army (no orders, so nothing but ravitaillement happens), turn
// 2 exercises the combat effect.
func TestResolveFamineCombatEffects(t *testing.T) {
	t.Run("attacker fights at full strength the turn it starves, zero the turn after", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		validateTestState(t, state)

		turn1, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("turn 1 Resolve: %v", err)
		}
		army := armyByID(t, turn1.State, "A1")
		if !army.Starving || army.Size != 1 {
			t.Fatalf("A1 after turn 1 = %+v, want starving with one troop lost", army)
		}

		state2 := cloneGameState(turn1.State)
		addNoble(state2, "N1", "ONE", "P1", "AAA")
		state2.Armies = append([]models.Army(nil), state2.Armies...)
		state2.Armies = append(state2.Armies, models.Army{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
		territoryState := state2.TerritoryStates["BBB"]
		armyID := models.ArmyID("A2")
		ownerID := models.PlayerID("P2")
		territoryState.Army = &armyID
		territoryState.OwnerID = &ownerID
		state2.TerritoryStates["BBB"] = territoryState
		state2.NextArmyID = nextArmyID(state2.Armies)
		addChain(t, state2, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state2)

		turn2, err := Resolve(state2, testBalance())
		if err != nil {
			t.Fatalf("turn 2 Resolve: %v", err)
		}
		if got := combatContenderForce(t, turn2.Events, "BBB", "A1"); got != 0 {
			t.Errorf("A1 attack force in turn 2 = %d, want famine force 0", got)
		}
	})

	t.Run("defender loses force but still retreats, only from the turn after it starves", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB", "CCC"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
				supplyTerritory("CCC", "CCC", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		validateTestState(t, state)

		turn1, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("turn 1 Resolve: %v", err)
		}
		if army := armyByID(t, turn1.State, "A1"); !army.Starving || army.Size != 1 {
			t.Fatalf("A1 after turn 1 = %+v, want starving with one troop lost", army)
		}

		state2 := cloneGameState(turn1.State)
		addNoble(state2, "N2", "TWO", "P2", "BBB")
		state2.Armies = append([]models.Army(nil), state2.Armies...)
		state2.Armies = append(state2.Armies, models.Army{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
		territoryState := state2.TerritoryStates["BBB"]
		armyID := models.ArmyID("A2")
		ownerID := models.PlayerID("P2")
		territoryState.Army = &armyID
		territoryState.OwnerID = &ownerID
		state2.TerritoryStates["BBB"] = territoryState
		state2.NextArmyID = nextArmyID(state2.Armies)
		addChain(t, state2, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
		validateTestState(t, state2)

		turn2, err := Resolve(state2, testBalance())
		if err != nil {
			t.Fatalf("turn 2 Resolve: %v", err)
		}
		if got := combatContenderForce(t, turn2.Events, "AAA", "A1"); got != 0 {
			t.Errorf("A1 defense force in turn 2 = %d, want 0", got)
		}
		if army := armyByID(t, turn2.State, "A1"); army.TerritoryID != "CCC" {
			t.Errorf("A1 = %+v, want a normal retreat despite zero defense force", army)
		}
	})

	t.Run("support from a starving army remains valid but adds no force, only from the turn after it starves", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB", "CCC"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
				supplyTerritory("CCC", "CCC", models.TerrainMountain, "AAA", "BBB"),
			},
			[]models.Army{
				{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
				{ID: "A2", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
			},
		)
		validateTestState(t, state)

		turn1, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("turn 1 Resolve: %v", err)
		}
		// CCC's mountain terrain cannot feed a two-troop army and P1 has no
		// other source anywhere in this scenario, so A2 (the supporter)
		// starves; A1 (fed by AAA's plain terrain alone) does not.
		if army := armyByID(t, turn1.State, "A2"); !army.Starving {
			t.Fatalf("A2 after turn 1 = %+v, want starving", army)
		}
		if army := armyByID(t, turn1.State, "A1"); army.Starving {
			t.Fatalf("A1 after turn 1 = %+v, want fed by its own plain terrain", army)
		}

		state2 := cloneGameState(turn1.State)
		addNoble(state2, "N1", "ONE", "P1", "AAA")
		addNoble(state2, "N2", "TWO", "P1", "CCC")
		state2.Armies = append([]models.Army(nil), state2.Armies...)
		state2.Armies = append(state2.Armies, models.Army{ID: "A3", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
		territoryState := state2.TerritoryStates["BBB"]
		armyID := models.ArmyID("A3")
		ownerID := models.PlayerID("P2")
		territoryState.Army = &armyID
		territoryState.OwnerID = &ownerID
		state2.TerritoryStates["BBB"] = territoryState
		state2.NextArmyID = nextArmyID(state2.Armies)
		addChain(t, state2, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		addChain(t, state2, "A2", "N2", models.Order{Type: models.OrderTypeSupport, PositionID: "CCC", TargetIDs: []models.TerritoryID{"AAA", "BBB"}})
		validateTestState(t, state2)

		turn2, err := Resolve(state2, testBalance())
		if err != nil {
			t.Fatalf("turn 2 Resolve: %v", err)
		}
		if army := armyByID(t, turn2.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want the free noble's bonus alone to still win", army)
		}
		if got := combatContenderForce(t, turn2.Events, "BBB", "A1"); got != 2 {
			t.Errorf("A1 attack force = %d, want army plus noble bonus while the starving support contributes nothing", got)
		}
	})

	t.Run("a zero-force attack can still move to an empty territory, only from the turn after it starves", func(t *testing.T) {
		state := testState(t,
			[]models.Territory{
				supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
				supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
			},
			[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
		)
		validateTestState(t, state)

		turn1, err := Resolve(state, testBalance())
		if err != nil {
			t.Fatalf("turn 1 Resolve: %v", err)
		}
		if army := armyByID(t, turn1.State, "A1"); !army.Starving || army.Size != 1 {
			t.Fatalf("A1 after turn 1 = %+v, want starving with one troop lost", army)
		}

		state2 := cloneGameState(turn1.State)
		addNoble(state2, "N1", "ONE", "P1", "AAA")
		addChain(t, state2, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
		validateTestState(t, state2)

		turn2, err := Resolve(state2, testBalance())
		if err != nil {
			t.Fatalf("turn 2 Resolve: %v", err)
		}
		if army := armyByID(t, turn2.State, "A1"); army.TerritoryID != "BBB" {
			t.Errorf("A1 = %+v, want the zero-force attack to still move onto the empty territory", army)
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
			{ID: "A2", OwnerID: "P2", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeMill, Level: 1, TerritoryID: "CCC"})
	// No army garrisons AAA: anchor it as P1's capital so control resolution
	// (now ahead of ravitaillement, #208) does not release it as unanchored
	// before supply can use it as a source.
	setCapital(state, "P1", "I1")
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
	if event := supplyEventForSource(t, first.Events, "AAA"); !reflect.DeepEqual(event.Rations, map[models.TerritoryID]int(nil)) {
		t.Errorf("supply event rations = %#v, want no BBB ration", event.Rations)
	}
	// CCC has no eligible neighbor for its mill (it has none at all), so
	// under #195 it self-supplies: its level-1 production exactly covers
	// A2's deficit, and no famine or auto-pillage occurs.
	if event := supplyEventForSource(t, first.Events, "CCC"); event.Production != 1 || event.Demand != 1 || event.StockConsumed != 0 {
		t.Errorf("CCC supply event = %#v, want the isolated mill to self-supply A2", event)
	}
	if hasFamineEvent(first.Events, "A2") {
		t.Errorf("events = %#v, want the isolated mill to prevent A2's famine", first.Events)
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

// TestControlledSupplySourcesExcludesOccupiedTerritory verifies that a
// controlled settlement occupied against its controller is unusable as a
// supply source by either the controller or the occupant (titres.md,
// economie.md#portée-de-ravitaillement, #196).
func TestControlledSupplySourcesExcludesOccupiedTerritory(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	ctx := newResolutionContext(cloneGameState(state), testBalance())
	if sources := controlledSupplySources(ctx, "P1"); len(sources) != 0 {
		t.Fatalf("P1 sources = %#v, want none: AAA is occupied against its controller", sources)
	}
	if sources := controlledSupplySources(ctx, "P2"); len(sources) != 0 {
		t.Fatalf("P2 sources = %#v, want none: P2 only occupies AAA, it does not control it", sources)
	}
}

// TestAbandonedVillageKeepsNeutralProduction checks #215's decision 3: once
// released for lack of an anchor, a village keeps its status quo neutral
// production (village_income into its own stock) exactly like one that was
// never claimed -- unlike a mill (see TestUnanchoredMillProducesNothing),
// villages are deliberately not made inert.
func TestAbandonedVillageKeepsNeutralProduction(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if owner := resolution.State.TerritoryStates["AAA"].OwnerID; owner != nil {
		t.Fatalf("AAA owner = %v, want nil after A1 departed (test setup drifted)", owner)
	}

	balance := testBalance()
	// Control resolution (now ahead of ravitaillement, #208) already
	// releases AAA as unanchored within this same turn 1, before income
	// runs: AAA is neutral by then, so it only earns the neutral village's
	// own base production (VillageIncome), not the owned territory income
	// (TerritoryIncome plus VillageIncome) it would have kept under A1.
	stockAfterAbandonment := balance.VillageIncome
	if got := resolution.State.TerritoryStates["AAA"].Resources; got != stockAfterAbandonment {
		t.Fatalf("AAA stock after turn 1 = %d, want %d (test setup drifted)", got, stockAfterAbandonment)
	}

	next, err := Resolve(resolution.State, balance)
	if err != nil {
		t.Fatalf("Resolve after abandonment: %v", err)
	}
	if got, want := next.State.TerritoryStates["AAA"].Resources, stockAfterAbandonment+balance.VillageIncome; got != want {
		t.Errorf("abandoned village stock = %d, want %d (turn 1's stock plus this turn's neutral production)", got, want)
	}
	event := supplyEventForSource(t, next.Events, "AAA")
	if event.OwnerID != "" || event.Production != balance.VillageIncome {
		t.Errorf("abandoned village supply event = %#v, want neutral base production %d", event, balance.VillageIncome)
	}
}

// TestIsControlledDepotUnusableWhenOccupied verifies that a depot on a
// territory occupied against its controller extends nobody's supply range
// (titres.md, economie.md#portée-de-ravitaillement, #196).
func TestIsControlledDepotUnusableWhenOccupied(t *testing.T) {
	state := testState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 2}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	ctx := newResolutionContext(cloneGameState(state), testBalance())
	if ctx.isControlledDepot("AAA", "P1") {
		t.Errorf("isControlledDepot(P1) = true, want false: occupied against its controller")
	}
	if ctx.isControlledDepot("AAA", "P2") {
		t.Errorf("isControlledDepot(P2) = true, want false: P2 only occupies, it does not control")
	}
}

// TestIsControlledDepotUnusableWhenUnanchored checks #215: a depot outside
// every fief and capital, once its owner's army leaves and no anchor keeps
// it, no longer extends anyone's supply range -- isControlledDepot already
// requires an exact OwnerID match, so the ordinary release below is enough.
func TestIsControlledDepotUnusableWhenUnanchored(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"),
			territory("BBB", "BBB", "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeSupplyDepot, Level: 1, TerritoryID: "AAA"})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if owner := resolution.State.TerritoryStates["AAA"].OwnerID; owner != nil {
		t.Fatalf("AAA owner = %v, want nil after A1 departed (test setup drifted)", owner)
	}

	ctx := newResolutionContext(cloneGameState(resolution.State), testBalance())
	if ctx.isControlledDepot("AAA", "P1") {
		t.Errorf("isControlledDepot(P1) = true, want false: released, no fief/capital/army anchor left")
	}
}

// TestResolveFleeingArmyIsFedBySourceTheSameTurn covers #208's acceptance
// case: an army that flees a starving cell into one with an accessible
// source is fed there this same turn -- ravitaillement resolves on
// post-movement positions, after control.
func TestResolveFleeingArmyIsFedBySourceTheSameTurn(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2}},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"})
	setTerritoryResources(state, "BBB", 10)
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeAttack, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.TerritoryID != "BBB" || army.Starving {
		t.Errorf("A1 = %+v, want it to have moved to BBB and not be starving", army)
	}
	if hasFamineEvent(resolution.Events, "A1") {
		t.Errorf("events = %#v, want no famine: A1's new position BBB has its own source", resolution.Events)
	}
}

// TestResolveDispersionReducesDemandTheSameTurn covers #208's acceptance
// case: a dispersion executed this turn already reduces the pooled demand
// this same turn's ravitaillement resolves, since it runs on post-movement
// positions.
func TestResolveDispersionReducesDemandTheSameTurn(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainMountain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
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
	// Split into two one-troop armies (cost 1 each, not the size-2 army's
	// cost 2), each fed by its own mountain terrain's single ration: the
	// dispersion's lower pooled demand already applies this same turn.
	if hasFamineEvent(resolution.Events, "A1") {
		t.Errorf("events = %#v, want the carrier fed after dispersing", resolution.Events)
	}
	if hasFamineEvent(resolution.Events, "A2") {
		t.Errorf("events = %#v, want the dispersed splinter fed too", resolution.Events)
	}
	if !hasArmy(resolution.State, "A2") {
		t.Fatal("dispersion should have created a second one-troop army at BBB")
	}
}

// TestResolveTransferFeedsRecipientTheSameTurn covers #208's acceptance
// case: a transfer submitted this turn already feeds its recipient this same
// turn's ravitaillement, since transfers execute during movement, ahead of
// ravitaillement.
func TestResolveTransferFeedsRecipientTheSameTurn(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainMountain, "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P1", TerritoryID: "BBB", Size: 3},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	setTerritoryResources(state, "AAA", 10)
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeTransfer, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}, Amount: 1,
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasFamineEvent(resolution.Events, "A2") {
		t.Errorf("events = %#v, want A2 fed by this same turn's transfer", resolution.Events)
	}
}

// TestResolveFaminePillageDissolvesFiefAndReleasesControl covers #208's
// acceptance case: the auto-pillage a same-turn famine triggers can destroy a
// fief capital's castle, dissolving the fief; releaseUnanchoredControl then
// runs again after ravitaillement (idempotently) to release the fief's other
// members, which lost their only anchor along with it.
func TestResolveFaminePillageDissolvesFiefAndReleasesControl(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			supplyTerritory("CAP", "CAP", models.TerrainMountain, "MEM"),
			supplyTerritory("MEM", "MEM", models.TerrainPlain, "CAP", "OTH"),
			supplyTerritory("OTH", "OTH", models.TerrainPlain, "MEM"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "CAP", Size: 2}},
	)
	setTerritoryOwner(state, "MEM", "P1")
	setTerritoryOwner(state, "OTH", "P1")
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CAP"})
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "CAP",
		Territories: []models.TerritoryID{"CAP", "MEM", "OTH"}, OwnerID: "P1",
	}}
	validateTestState(t, state)

	// This scenario is about the auto-pillage and its fallout, not territory
	// income: zeroing it out keeps the fief's own income from covering A1's
	// deficit before the famine this test is about. A1's demand
	// (armyCost(2)=2) outstrips CAP's single mountain ration, and the
	// balance's default pillage bonus (2) covers the resulting deficit (1)
	// with a non-negative gain, so CAP's castle is auto-pillaged and saves
	// A1 from losing a troop.
	balance := testBalance()
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0
	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	famine := famineEventForArmy(t, resolution.Events, "A1")
	if !famine.SavedByPillage || famine.InfrastructureType != models.InfraTypeCastle {
		t.Fatalf("famine event = %#v, want the capital's castle auto-pillaged and A1 saved", famine)
	}
	if len(resolution.State.Fiefs) != 0 {
		t.Errorf("fiefs = %#v, want F1 dissolved with its capital's castle", resolution.State.Fiefs)
	}
	if owner := resolution.State.TerritoryStates["MEM"].OwnerID; owner != nil {
		t.Errorf("MEM owner = %v, want nil: released with the fief that anchored it", owner)
	}
	if owner := resolution.State.TerritoryStates["OTH"].OwnerID; owner != nil {
		t.Errorf("OTH owner = %v, want nil: released with the fief that anchored it", owner)
	}
}

// TestResolveFirstTurnHasNoFamineBeforeMovement covers #208's acceptance
// case: the very first action turn of a game carries no models.Army.Starving
// from any previous turn, so ctx.famished starts empty and an army destined
// to starve by this same turn's own end still attacks at full strength.
func TestResolveFirstTurnHasNoFamineBeforeMovement(t *testing.T) {
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
	// Nothing carried into this very first turn from a previous one, so
	// A1's attack lands at full force despite AAA (its starting position)
	// being unable to feed it: only the position it ends this same turn on
	// (BBB, won and fully self-sufficient) matters for this turn's own
	// ravitaillement.
	if got := combatContenderForce(t, resolution.Events, "BBB", "A1"); got != 3 {
		t.Errorf("A1 attack force = %d, want army (2) plus noble bonus (1)", got)
	}
}
