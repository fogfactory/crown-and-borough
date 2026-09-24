package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func revoltTestState(t *testing.T, territories []models.Territory, armies []models.Army) *models.GameState {
	t.Helper()
	state := testState(t, territories, armies)
	state.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}}}
	state.Auguries[1] = models.YearAugury{
		Year:       1,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
		Calamities: []models.Calamity{{CardID: "C9", Kind: models.CardKindFamine, Season: models.SeasonSpring, Year: 1, RegionSeed: "ROS"}},
	}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindRevolt}, {ID: "C9", Kind: models.CardKindFamine}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	return state
}

func neutralArmyAt(t *testing.T, state *models.GameState, territoryID models.TerritoryID) *models.Army {
	t.Helper()
	for index := range state.Armies {
		army := &state.Armies[index]
		if army.OwnerID == models.NeutralPlayerID && army.TerritoryID == territoryID {
			return army
		}
	}
	t.Fatalf("armies = %#v, want a neutral army at %q", state.Armies, territoryID)
	return nil
}

func TestRevoltPlacesNeutralArmyOnUnownedTerritory(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	validateTestState(t, state)
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyMinSize = 1
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = 1
	resolution, err := ResolveWithDeckOrders(state, balance, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	rebels := neutralArmyAt(t, resolution.State, "BBB")
	if rebels.Size != 1 {
		t.Fatalf("rebel size = %d, want the fixed one-troop roll", rebels.Size)
	}
	if occupant := resolution.State.TerritoryStates["BBB"].Army; occupant == nil || *occupant != rebels.ID {
		t.Fatalf("BBB occupancy = %#v, want the rebel army", resolution.State.TerritoryStates["BBB"].Army)
	}
	rollFound := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeNeutralArmy && event.TerritoryID == "BBB" && event.Troops == rebels.Size {
			rollFound = true
		}
	}
	if !rollFound {
		t.Fatalf("events = %#v, want a neutral army event carrying the one-troop roll", resolution.Events)
	}
}

func TestRevoltResolvesAsCombatOnOccupiedTerritory(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 2},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	validateTestState(t, state)
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyMinSize = 3
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = 3
	resolution, err := ResolveWithDeckOrders(state, balance, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	rebels := neutralArmyAt(t, resolution.State, "BBB")
	if rebels.Size != 3 {
		t.Fatalf("rebel size = %d, want the fixed roll of 3", rebels.Size)
	}
	if occupant := resolution.State.TerritoryStates["BBB"].Army; occupant == nil || *occupant != rebels.ID {
		t.Fatalf("BBB occupancy = %#v, want the rebels in control", resolution.State.TerritoryStates["BBB"].Army)
	}
	retreated := false
	for _, army := range resolution.State.Armies {
		if army.ID == "A2" {
			if army.TerritoryID != "CCC" {
				t.Fatalf("A2 = %#v, want it to retreat to the empty CCC", army)
			}
			retreated = true
		}
	}
	if !retreated {
		t.Fatal("A2 disappeared instead of retreating")
	}
	combatFound := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeCombat && event.TerritoryID == "BBB" && event.WinnerArmyID == rebels.ID {
			combatFound = true
		}
	}
	if !combatFound {
		t.Fatalf("events = %#v, want a revolt combat won by the rebels at BBB", resolution.Events)
	}
}

func TestRevoltRefusedInWinter(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainPlain)},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	ctx := newResolutionContext(state, testBalance())
	definition := cardDefinitions[models.CardKindRevolt]
	if ok, reason := definition.CanPlay(&ExecutionContext{resolution: ctx, season: models.SeasonWinter}, models.DeckOrder{TargetTerritoryID: "AAA"}); ok || reason != "deck_order_out_of_season" {
		t.Fatalf("winter revolt = %t/%q, want false/deck_order_out_of_season", ok, reason)
	}
}

func TestNeutralArmyStarvesWithoutLocalRations(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	state.Armies = []models.Army{{ID: "A1", OwnerID: models.NeutralPlayerID, TerritoryID: "AAA", Size: 3}}
	state.NextArmyID = 2
	armyID := models.ArmyID("A1")
	aaaState := state.TerritoryStates["AAA"]
	aaaState.Army = &armyID
	state.TerritoryStates["AAA"] = aaaState
	validateTestState(t, state)
	resolution, err := Resolve(state, testBalance())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	starved := neutralArmyAt(t, resolution.State, "AAA")
	if starved.Size != 2 {
		t.Fatalf("starving neutral army = %d troops, want 2 after one loss", starved.Size)
	}
	report := BuildTurnReport(state, resolution.State, resolution.Events, nil)
	found := false
	for _, effect := range report.SeasonEffects {
		if effect.Kind == EventTypeFamine && effect.Territory == "AAA" && effect.SizeBefore == 3 && effect.SizeAfter == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("season effects = %#v, want a neutral famine line at AAA from 3 to 2 troops", report.SeasonEffects)
	}
}

func TestRevoltIsCanceledWhenFamineIsCountered(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}},
	)
	setTerritoryOwner(state, "AAA", "P1")
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C2", Kind: models.CardKindAbundantHarvest})
	state.SpecialDeck.Hands["P1"] = append(state.SpecialDeck.Hands["P1"], "C2")
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {
			{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindAbundantHarvest, RegionSeed: "ROS"},
			{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"},
		},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	for _, army := range resolution.State.Armies {
		if army.OwnerID == models.NeutralPlayerID {
			t.Fatalf("armies = %#v, want no rebel army after the bad harvest was countered", resolution.State.Armies)
		}
	}
	canceled := false
	restored := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeCardCanceled && event.CardKind == models.CardKindRevolt && event.TerritoryID == "BBB" {
			canceled = true
		}
		if event.Type == EventTypeDeckRestore && event.OwnerID == "P1" && event.CardKind == models.CardKindRevolt {
			restored = true
		}
	}
	if !canceled {
		t.Fatalf("events = %#v, want a canceled revolt card at BBB", resolution.Events)
	}
	if !restored {
		t.Fatalf("events = %#v, want the revolt card restored to its player", resolution.Events)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C1" {
		t.Fatalf("P1 hand = %#v, want the canceled revolt card back", got)
	}
	if got := resolution.State.SpecialDeck.Discard; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("discard = %#v, want only the abundant harvest", got)
	}
}

func TestNeutralRebelRetreatsWhenDefeated(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{
			supplyTerritory("AAA", "AAA", models.TerrainPlain, "BBB"),
			supplyTerritory("BBB", "BBB", models.TerrainPlain, "AAA", "CCC"),
			supplyTerritory("CCC", "CCC", models.TerrainPlain, "BBB"),
		},
		[]models.Army{
			{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1},
			{ID: "A2", OwnerID: "P2", TerritoryID: "BBB", Size: 3},
		},
	)
	setTerritoryOwner(state, "AAA", "P1")
	setTerritoryOwner(state, "CCC", "P2")
	validateTestState(t, state)
	balance := testBalance()
	balance.SpecialOrders.Effects.RevoltArmyMinSize = 1
	balance.SpecialOrders.Effects.RevoltArmyMaxSize = 1
	resolution, err := ResolveWithDeckOrders(state, balance, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindRevolt, TargetTerritoryID: "BBB"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if occupant := resolution.State.TerritoryStates["BBB"].Army; occupant == nil || *occupant != "A2" {
		t.Fatalf("BBB occupancy = %#v, want the holder to keep the territory", resolution.State.TerritoryStates["BBB"].Army)
	}
	rebels := neutralArmyAt(t, resolution.State, "CCC")
	if rebels.Size != 1 {
		t.Fatalf("rebel size = %d, want the crushed rebellion to retreat with one troop", rebels.Size)
	}
	combatFound := false
	for _, event := range resolution.Events {
		if event.Type == EventTypeCombat && event.TerritoryID == "BBB" && event.Reason == "defense_holds" {
			combatFound = true
		}
	}
	if !combatFound {
		t.Fatalf("events = %#v, want a revolt combat held by the defense at BBB", resolution.Events)
	}
}

func TestFindSupplyServesNeutralArmy(t *testing.T) {
	state := revoltTestState(t,
		[]models.Territory{supplyTerritory("AAA", "AAA", models.TerrainMountain)},
		nil,
	)
	setTerritoryOwner(state, "AAA", "P1")
	state.Armies = []models.Army{{ID: "A1", OwnerID: models.NeutralPlayerID, TerritoryID: "AAA", Size: 3}}
	state.NextArmyID = 2
	armyID := models.ArmyID("A1")
	aaaState := state.TerritoryStates["AAA"]
	aaaState.Army = &armyID
	state.TerritoryStates["AAA"] = aaaState
	validateTestState(t, state)

	line, err := FindSupply(state, testBalance(), "AAA")
	if err != nil {
		t.Fatalf("FindSupply: %v", err)
	}
	if line.ArmyOwner != models.NeutralPlayerID || line.ArmySize != 3 {
		t.Fatalf("supply line = %#v, want the neutral three-troop army", line)
	}
	if line.TotalDemand != 4 || line.FamineRations != 1 || line.LocalProduction != 0 || line.Rations != 0 {
		t.Fatalf("supply line = %#v, want demand 4 with the only local ration lost to the famine", line)
	}
	if line.SelfSupplied || line.Source != nil {
		t.Fatalf("supply line = %#v, want an unfed neutral army without any source", line)
	}
}
