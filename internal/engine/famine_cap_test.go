package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestSuppliedStrength(t *testing.T) {
	for _, test := range []struct {
		size, missing, want int
	}{
		{size: 1, missing: 1, want: 0},
		{size: 2, missing: 1, want: 1},
		{size: 2, missing: 2, want: 0},
		{size: 3, missing: 1, want: 2},
		{size: 3, missing: 2, want: 2},
		{size: 3, missing: 3, want: 1},
		{size: 4, missing: 5, want: 2},
		{size: 4, missing: 7, want: 1},
		{size: 4, missing: 8, want: 0},
	} {
		if got := suppliedStrength(test.size, test.missing, 2); got != test.want {
			t.Errorf("suppliedStrength(size %d, missing %d) = %d, want %d", test.size, test.missing, got, test.want)
		}
	}
}

func famineCapState(t *testing.T, terrain models.Terrain, army models.Army) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", terrain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{army},
	)
	validateTestState(t, state)
	return state
}

func TestResolveFirstFamineTurnCapsStrengthAtSuppliedMinusOne(t *testing.T) {
	// Plain gives 3 rations: a three-troop army (4 rations) is supplied at 2.
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3})
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	army := armyByID(t, resolution.State, "A1")
	if !army.Starving || army.Size != 3 || army.StarvingStrength != 1 {
		t.Fatalf("A1 = %+v, want starving, size 3 kept and strength capped at 1", army)
	}
}

func TestResolveStarvingDefenderFightsAtCap(t *testing.T) {
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3, Starving: true, StarvingStrength: 1})
	addNoble(state, "N2", "TWO", "P2", "BBB")
	state.Armies = append(state.Armies, models.Army{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 1})
	territoryState := state.TerritoryStates["BBB"]
	armyID := models.ArmyID("A2")
	territoryState.Army = &armyID
	state.TerritoryStates["BBB"] = territoryState
	state.NextArmyID = nextArmyID(state.Armies)
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := combatContenderForce(t, resolution.Events, "AAA", "A1"); got != 1 {
		t.Errorf("A1 defense force = %d, want the cap of 1", got)
	}
}

func TestResolveRepeatedFamineRemovesTroopsDownToSuppliedStrength(t *testing.T) {
	// Four troops cost 8 rations, plain gives 3: supplied at 2, two troops go.
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4, Starving: true, StarvingStrength: 1})
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	army := armyByID(t, resolution.State, "A1")
	if army.Size != 2 || army.Starving || army.StarvingStrength != 0 {
		t.Fatalf("A1 = %+v, want size 2 and no longer starving", army)
	}
	event := famineEventForArmy(t, resolution.Events, "A1")
	if event.TroopsLost != 2 || event.SuppliedStrength != 2 || event.Destroyed {
		t.Errorf("famine event = %#v, want two troops lost down to a supplied strength of 2", event)
	}
}

func TestResolveFamineWithoutRationsDestroysArmyAndKillsItsNobles(t *testing.T) {
	balance := testBalance()
	balance.RationTerrain[models.TerrainMountain] = 0
	state := famineCapState(t, models.TerrainMountain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2, Starving: true})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	validateTestState(t, state)

	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if hasArmy(resolution.State, "A1") {
		t.Fatal("A1 should disappear when no ration reaches it")
	}
	for _, noble := range resolution.State.Nobles {
		if noble.ID == "N1" {
			t.Errorf("N1 = %+v, want it dead with its army", noble)
		}
	}
	died := false
	for _, removed := range resolution.State.RemovedNobles {
		if removed.ID == "N1" && removed.Cause == models.DeathCauseFamine {
			died = true
		}
	}
	if !died {
		t.Error("N1 should be recorded as dead of famine")
	}
	event := famineEventForArmy(t, resolution.Events, "A1")
	if !event.Destroyed || event.TroopsLost != 2 {
		t.Errorf("famine event = %#v, want a destroyed army", event)
	}
	destroyed := false
	for _, e := range resolution.Events {
		if e.Type == EventTypeArmyDestroyed && e.ArmyID == "A1" && e.Reason == "famine" {
			destroyed = true
		}
	}
	if !destroyed {
		t.Error("missing army_destroyed event with the famine reason")
	}
}

func TestResolveFedStarvingArmyLeavesFamine(t *testing.T) {
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2, Starving: true, StarvingStrength: 1})
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if army := armyByID(t, resolution.State, "A1"); army.Starving || army.StarvingStrength != 0 || army.Size != 2 {
		t.Errorf("A1 = %+v, want a fed army out of famine", army)
	}
}

func TestResolveRetreatOntoUnsuppliedHostStarvesIt(t *testing.T) {
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB", "EEE"),
			territory("BBB", "BBB", "AAA"),
			territory("EEE", "EEE", "AAA"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 3},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 4},
			{ID: "A5", OwnerID: "P1", TerritoryID: "EEE", Size: 3},
		},
	)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	addNoble(state, "N5", "FIV", "P1", "EEE")
	addChain(t, state, "A1", "N1", models.Order{Type: models.OrderTypeHold, PositionID: "AAA"})
	addChain(t, state, "A2", "N2", models.Order{Type: models.OrderTypeAttack, PositionID: "BBB", TargetIDs: []models.TerritoryID{"AAA"}})
	addChain(t, state, "A5", "N5", models.Order{Type: models.OrderTypeHold, PositionID: "EEE"})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	host := armyByID(t, resolution.State, "A5")
	if host.Size <= 3 {
		t.Fatalf("A5 = %+v, want it grown by the retreat merge", host)
	}
	if !host.Starving {
		t.Errorf("A5 = %+v, want the merged host starving on its unsupplied cell", host)
	}
}

func TestForecastFamineRiskReportsSuppliedStrengthAndExpectedLoss(t *testing.T) {
	for _, test := range []struct {
		name     string
		starving bool
		wantLoss int
	}{
		{name: "first deficit only caps the strength", wantLoss: 0},
		{name: "second deficit removes the excess", starving: true, wantLoss: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4, Starving: test.starving})
			risks := ForecastFamineRisk(state, testBalance())["P1"].ArmiesAtRisk
			if len(risks) != 1 || risks[0].SuppliedStrength != 2 || risks[0].ExpectedLoss != test.wantLoss {
				t.Fatalf("risks = %#v, want supplied strength 2 and loss %d", risks, test.wantLoss)
			}
		})
	}
}
