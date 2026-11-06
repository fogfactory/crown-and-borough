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

func TestResolveFamineDestroyedArmyDropsItsChain(t *testing.T) {
	balance := testBalance()
	balance.RationTerrain[models.TerrainMountain] = 0
	state := famineCapState(t, models.TerrainMountain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 2, Starving: true})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	hold := models.Order{Type: models.OrderTypeHold, PositionID: "AAA"}
	addChainOrders(t, state, "A1", "N1", hold, hold)
	validateTestState(t, state)

	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(resolution.State.Chains) != 0 {
		t.Errorf("chains = %+v, want the destroyed army's chain dropped", resolution.State.Chains)
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

func TestForecastFamineRiskExpectsNoLossWhenPillageSavesTheArmy(t *testing.T) {
	balance := testBalance()
	balance.PillageBonus = 10
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 4, Starving: true})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "AAA"})
	validateTestState(t, state)

	risks := ForecastFamineRisk(state, balance)["P1"].ArmiesAtRisk
	if len(risks) != 1 || risks[0].ExpectedLoss != 0 || !risks[0].SavedByPillage || risks[0].PillageInfrastructure != models.InfraTypeMill {
		t.Fatalf("risks = %#v, want the mill pillaged with no expected loss", risks)
	}
	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if event := famineEventForArmy(t, resolution.Events, "A1"); !event.SavedByPillage || event.TroopsLost != 0 {
		t.Errorf("famine event = %#v, want the army saved by pillage", event)
	}
}

func TestResolveShortPoolFeedsTheClosestStarvingArmyAtItsSuppliedStrength(t *testing.T) {
	// The pool holds 7 rations for B (4 troops, 8 rations, on the source)
	// and A (2 troops, 2 rations, one step away). B is served first, starves
	// and takes the 4 rations its supplied strength of 3 needs; A takes 2 and
	// the last ration stays in stock.
	balance := testBalance()
	balance.RationTerrain[models.TerrainMountain] = 0
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0
	state := testState(t,
		[]models.Territory{
			supplyTerritory("SSS", "SSS", models.TerrainMountain, "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainMountain, "SSS"),
		},
		[]models.Army{
			{ID: "B", OwnerID: "P1", TerritoryID: "SSS", Size: 4},
			{ID: "A", OwnerID: "P1", TerritoryID: "CCC", Size: 2},
		},
	)
	setTerritoryResources(state, "SSS", 7)
	validateTestState(t, state)

	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if b := armyByID(t, resolution.State, "B"); !b.Starving || b.Size != 4 || b.StarvingStrength != 2 {
		t.Errorf("B = %+v, want starving at size 4 with strength capped at 2", b)
	}
	if a := armyByID(t, resolution.State, "A"); a.Starving {
		t.Errorf("A = %+v, want it fed by the rations B left", a)
	}
	if stock := resolution.State.TerritoryStates["SSS"].Resources; stock != 1 {
		t.Errorf("SSS stock = %d, want the 1 ration nobody took", stock)
	}
	if event := famineEventForArmy(t, resolution.Events, "B"); event.SuppliedStrength != 3 {
		t.Errorf("B famine event = %#v, want a supplied strength of 3", event)
	}

	risks := ForecastFamineRisk(state, balance)["P1"]
	if len(risks.ArmiesAtRisk) != 1 || risks.ArmiesAtRisk[0].SuppliedStrength != 3 || risks.ArmiesAtRisk[0].Deficit != 1 || risks.NetConsumption != 6 {
		t.Errorf("forecast = %#v, want B alone at risk, supplied at 3, 1 R short, 6 R consumed", risks)
	}
}

func TestResolveCoveredPoolStoresOnlyItsNetExcess(t *testing.T) {
	// XXX's mill produces the 1 ration Y1 needs while Y1 is assigned to the
	// YYY stock: the pool's excess is zero, so neither source gains a ration.
	balance := testBalance()
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0
	state := testState(t,
		[]models.Territory{
			supplyTerritory("XXX", "XXX", models.TerrainMountain, "YYY"),
			supplyTerritory("YYY", "YYY", models.TerrainMountain, "XXX"),
		},
		[]models.Army{
			{ID: "X1", OwnerID: "P1", TerritoryID: "XXX", Size: 1},
			{ID: "Y1", OwnerID: "P1", TerritoryID: "YYY", Size: 2},
		},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeMill, Level: 1, TerritoryID: "XXX"})
	setTerritoryResources(state, "YYY", 1)
	validateTestState(t, state)

	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if source := supplyEventForSource(t, resolution.Events, "XXX"); source.Production != 1 {
		t.Fatalf("XXX supply event = %#v, want the mill's 1 ration in the pool", source)
	}
	for _, army := range resolution.State.Armies {
		if army.Starving {
			t.Errorf("%s = %+v, want every army fed", army.ID, army)
		}
	}
	total := resolution.State.TerritoryStates["XXX"].Resources + resolution.State.TerritoryStates["YYY"].Resources
	if total != 1 {
		t.Errorf("stocks XXX+YYY = %d, want 1: the mill's ration fed Y1", total)
	}
}

func TestResolvePillageCoversOnlyTheDemandThePoolLeftUnfed(t *testing.T) {
	// B (4 troops, 8 rations) starves on a castle holding 4 rations: it takes
	// those 4 for its supplied strength of 3, and the pillage bonus of 4
	// covers the remaining 4, which saves it at full size.
	balance := testBalance()
	balance.RationTerrain[models.TerrainMountain] = 0
	balance.TerritoryIncome = 0
	balance.VillageIncome = 0
	balance.PillageBonus = 4
	state := famineCapState(t, models.TerrainMountain, models.Army{ID: "B", OwnerID: "P1", TerritoryID: "AAA", Size: 4, Starving: true})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setTerritoryResources(state, "AAA", 4)
	validateTestState(t, state)

	risks := ForecastFamineRisk(state, balance)["P1"].ArmiesAtRisk
	if len(risks) != 1 || !risks[0].SavedByPillage || risks[0].ExpectedLoss != 0 {
		t.Errorf("risks = %#v, want the pillage forecast to save B", risks)
	}
	resolution, err := Resolve(state, balance)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if event := famineEventForArmy(t, resolution.Events, "B"); !event.SavedByPillage || event.TroopsLost != 0 {
		t.Errorf("famine event = %#v, want B saved by pillage", event)
	}
	if b := armyByID(t, resolution.State, "B"); b.Size != 4 || b.Starving {
		t.Errorf("B = %+v, want it saved at size 4", b)
	}
}

func TestResolveDispersedStarvingArmySplintersKeepTheSecondDeficit(t *testing.T) {
	// Plain gives 3 rations: each three-troop half (4 rations) is supplied at
	// 2. Both halves, including the one that gets a fresh army ID, are on
	// their second consecutive deficit and lose the troop they cannot feed.
	state := famineCapState(t, models.TerrainPlain, models.Army{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 6, Starving: true, StarvingStrength: 1})
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addChain(t, state, "A1", "N1", models.Order{
		Type: models.OrderTypeDisperse, PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB", "BBB", "BBB"},
		NobleAssignments: map[models.TerritoryID][]models.NobleCode{"BBB": {"*"}},
	})
	validateTestState(t, state)

	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	for _, territoryID := range []models.TerritoryID{"AAA", "BBB"} {
		found := false
		for _, army := range resolution.State.Armies {
			if army.TerritoryID != territoryID {
				continue
			}
			found = true
			if army.Size != 2 {
				t.Errorf("army on %s = %+v, want it cut down to its supplied 2", territoryID, army)
			}
		}
		if !found {
			t.Errorf("no army on %s, want one half of the dispersion", territoryID)
		}
	}
}
