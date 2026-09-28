package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// foundFiefTestState builds a 7-territory line AAA-BBB-CCC-DDD-EEE-FFF-GGG,
// all controlled by P1, a castle on AAA (the intended capital), a free noble
// HUG at AAA, and enough stock on AAA to pay for any group.
func foundFiefTestState(t *testing.T) *models.GameState {
	t.Helper()
	territories := []models.Territory{
		territory("AAA", "AAA", "BBB"),
		territory("BBB", "BBB", "AAA", "CCC"),
		territory("CCC", "CCC", "BBB", "DDD"),
		territory("DDD", "DDD", "CCC", "EEE"),
		territory("EEE", "EEE", "DDD", "FFF"),
		territory("FFF", "FFF", "EEE", "GGG"),
		territory("GGG", "GGG", "FFF"),
	}
	state := winterTestState(t, territories, nil)
	for _, id := range []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG"} {
		setTerritoryOwner(state, id, "P1")
	}
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	addNoble(state, "N1", "HUG", "P1", "AAA")
	territoryState := state.TerritoryStates["AAA"]
	territoryState.Resources = 50
	state.TerritoryStates["AAA"] = territoryState
	return state
}

func placeArmyAt(state *models.GameState, armyID models.ArmyID, ownerID models.PlayerID, territoryID models.TerritoryID, size int) {
	state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: ownerID, TerritoryID: territoryID, Size: size})
	territoryState := state.TerritoryStates[territoryID]
	armyIDCopy := armyID
	territoryState.Army = &armyIDCopy
	state.TerritoryStates[territoryID] = territoryState
}

func applyFoundFief(state *models.GameState, order models.WinterOrder) (*resolutionContext, Event) {
	order.ID = "O1"
	order.Type = models.WinterOrderTypeFoundFief
	ctx := newResolutionContext(state, testBalance())
	foundFiefOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	return ctx, ctx.events[len(ctx.events)-1]
}

func TestFoundFiefOrderTitlesAndCosts(t *testing.T) {
	cases := []struct {
		territories []models.TerritoryID
		wantTitle   models.FiefTitle
		wantCost    int
	}{
		{[]models.TerritoryID{"AAA", "BBB", "CCC"}, models.FiefTitleBarony, 6},
		{[]models.TerritoryID{"AAA", "BBB", "CCC", "DDD"}, models.FiefTitleCounty, 8},
		{[]models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE"}, models.FiefTitleMarquisate, 10},
		{[]models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF"}, models.FiefTitleDuchy, 12},
		{[]models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG"}, models.FiefTitleDuchy, 14},
	}
	for _, tc := range cases {
		state := foundFiefTestState(t)
		ctx, event := applyFoundFief(state, models.WinterOrder{NobleCode: "HUG", TerritoryID: tc.territories[0], TerritoryIDs: tc.territories})
		if event.Type != EventTypeFiefFounded {
			t.Fatalf("territories %v: last event = %#v, want fief_founded", tc.territories, event)
		}
		if event.ResourceSpent != tc.wantCost {
			t.Errorf("territories %v: spent = %d, want %d", tc.territories, event.ResourceSpent, tc.wantCost)
		}
		if len(ctx.state.Fiefs) != 1 {
			t.Fatalf("territories %v: fiefs = %#v, want one", tc.territories, ctx.state.Fiefs)
		}
		fief := ctx.state.Fiefs[0]
		if fief.Title != tc.wantTitle {
			t.Errorf("territories %v: title = %q, want %q", tc.territories, fief.Title, tc.wantTitle)
		}
		if fief.CapitalTerritoryID != "AAA" || fief.OwnerID != "P1" {
			t.Errorf("territories %v: fief = %#v", tc.territories, fief)
		}
		if fief.HolderNobleID == nil || *fief.HolderNobleID != "N1" {
			t.Errorf("territories %v: holder = %v, want N1", tc.territories, fief.HolderNobleID)
		}
		if err := state.Validate(); err != nil {
			t.Errorf("territories %v: Validate() after founding = %v", tc.territories, err)
		}
	}
}

func TestFoundFiefOrderTolerateOtherCastleInGroup(t *testing.T) {
	state := foundFiefTestState(t)
	addInfrastructure(state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "CCC"})
	_, event := applyFoundFief(state, models.WinterOrder{
		NobleCode: "HUG", TerritoryID: "AAA",
		TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"},
	})
	if event.Type != EventTypeFiefFounded {
		t.Fatalf("event = %#v, want fief_founded", event)
	}
}

func TestFoundFiefOrderSameNobleMultipleTitles(t *testing.T) {
	state := foundFiefTestState(t)
	ctx := newResolutionContext(state, testBalance())
	foundFiefOrder{order: models.WinterOrder{
		ID: "O1", Type: models.WinterOrderTypeFoundFief, NobleCode: "HUG", TerritoryID: "AAA",
		TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"},
	}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	// The second group is disjoint from the first and needs its own castle
	// and capital: build one on EEE before assigning HUG a second title.
	addInfrastructure(ctx.state, models.Infrastructure{ID: "I2", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "EEE"})
	ctx.rebuildIndexes()
	foundFiefOrder{order: models.WinterOrder{
		ID: "O2", Type: models.WinterOrderTypeFoundFief, NobleCode: "HUG", TerritoryID: "EEE",
		TerritoryIDs: []models.TerritoryID{"EEE", "FFF", "GGG"},
	}}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	founded := eventsOfType(ctx.events, EventTypeFiefFounded)
	if len(founded) != 2 {
		t.Fatalf("founded events = %#v, want 2", founded)
	}
	if len(ctx.state.Fiefs) != 2 {
		t.Fatalf("fiefs = %#v, want 2", ctx.state.Fiefs)
	}
	for _, fief := range ctx.state.Fiefs {
		if fief.HolderNobleID == nil || *fief.HolderNobleID != "N1" {
			t.Errorf("fief %#v: want holder N1", fief)
		}
	}
}

// TestFoundFiefOrderRejectsFortifiedVillageAsCapital verifies that a
// fortified village can never be a fief's capital: T F still requires an
// actual castle there (#193).
func TestFoundFiefOrderRejectsFortifiedVillageAsCapital(t *testing.T) {
	state := foundFiefTestState(t)
	// Replace AAA's castle with a fortified village: the only structure on
	// the intended capital is now a village, fortified or not.
	for index := range state.Infrastructures {
		if state.Infrastructures[index].ID == "I1" {
			state.Infrastructures[index] = models.Infrastructure{
				ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA", Fortified: true,
			}
		}
	}
	_, event := applyFoundFief(state, models.WinterOrder{
		NobleCode: "HUG", TerritoryID: "AAA",
		TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"},
	})
	if event.Type != EventTypeRejected || event.Reason != "fief_capital_requires_castle" {
		t.Fatalf("event = %#v, want fief_capital_requires_castle", event)
	}
}

func TestFoundFiefOrderNeutralArmyBlocks(t *testing.T) {
	state := foundFiefTestState(t)
	placeArmyAt(state, "A9", models.NeutralPlayerID, "CCC", 2)
	_, event := applyFoundFief(state, models.WinterOrder{
		NobleCode: "HUG", TerritoryID: "AAA",
		TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"},
	})
	if event.Type != EventTypeRejected || event.Reason != "fief_territory_occupied_by_other_player" {
		t.Fatalf("event = %#v, want fief_territory_occupied_by_other_player", event)
	}
}

func TestFoundFiefOrderRejections(t *testing.T) {
	cases := []struct {
		name string
		// wantTerritory is the offending territory a rejection reason tied
		// to one specific group member should be attributed to, so a map
		// marker lands there rather than always on the capital. Left empty
		// for reasons that are not about one particular territory.
		wantTerritory models.TerritoryID
		mutate        func(state *models.GameState)
		order         models.WinterOrder
		reason        string
	}{
		{
			name:   "unknown noble",
			order:  models.WinterOrder{NobleCode: "ZZZ", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason: "unknown_noble",
		},
		{
			name:   "holder not owned",
			mutate: func(state *models.GameState) { addNoble(state, "N2", "BOB", "P2", "AAA") },
			order:  models.WinterOrder{NobleCode: "BOB", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason: "fief_holder_not_owned",
		},
		{
			name:   "holder not free",
			mutate: func(state *models.GameState) { setNobleStatus(state, "N1", models.NobleStatusHostage) },
			order:  models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason: "fief_holder_not_free",
		},
		{
			name:          "duplicate territory",
			order:         models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "BBB"}},
			reason:        "fief_duplicate_territory",
			wantTerritory: "BBB",
		},
		{
			name:   "too small",
			order:  models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB"}},
			reason: "fief_too_small",
		},
		{
			name:          "unknown territory",
			order:         models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "ZZZ"}},
			reason:        "unknown_territory",
			wantTerritory: "ZZZ",
		},
		{
			name:          "territory not controlled",
			mutate:        func(state *models.GameState) { setTerritoryOwner(state, "CCC", "P2") },
			order:         models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason:        "territory_not_controlled",
			wantTerritory: "CCC",
		},
		{
			name:   "capital requires castle",
			order:  models.WinterOrder{NobleCode: "HUG", TerritoryID: "BBB", TerritoryIDs: []models.TerritoryID{"BBB", "AAA", "CCC"}},
			reason: "fief_capital_requires_castle",
		},
		{
			name: "territory already in fief",
			mutate: func(state *models.GameState) {
				state.Fiefs = []models.Fief{{
					ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "EEE",
					Territories: []models.TerritoryID{"EEE", "FFF", "GGG"}, OwnerID: "P1",
				}}
			},
			order:         models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "FFF"}},
			reason:        "fief_territory_already_in_fief",
			wantTerritory: "FFF",
		},
		{
			name:          "territory occupied by another player",
			mutate:        func(state *models.GameState) { placeArmyAt(state, "A9", "P2", "CCC", 2) },
			order:         models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason:        "fief_territory_occupied_by_other_player",
			wantTerritory: "CCC",
		},
		{
			name:   "not contiguous",
			order:  models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "CCC", "DDD"}},
			reason: "fief_not_contiguous",
		},
		{
			name: "insufficient resources",
			mutate: func(state *models.GameState) {
				territoryState := state.TerritoryStates["AAA"]
				territoryState.Resources = 0
				state.TerritoryStates["AAA"] = territoryState
			},
			order:  models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"}},
			reason: "insufficient_resources",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := foundFiefTestState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			_, event := applyFoundFief(state, tc.order)
			if event.Type != EventTypeRejected {
				t.Fatalf("event = %#v, want rejected", event)
			}
			if event.Reason != tc.reason {
				t.Errorf("reason = %q, want %q", event.Reason, tc.reason)
			}
			if event.TerritoryID != tc.wantTerritory {
				t.Errorf("TerritoryID = %q, want %q", event.TerritoryID, tc.wantTerritory)
			}
			if event.ResourceSpent != 0 {
				t.Errorf("ResourceSpent = %d, want 0", event.ResourceSpent)
			}
			if len(state.Fiefs) != 0 && tc.name != "territory already in fief" {
				t.Errorf("Fiefs = %#v, want none founded", state.Fiefs)
			}
		})
	}
}

func TestFiefGroupContiguousUsesCrossableBordersOnly(t *testing.T) {
	state := foundFiefTestState(t)
	ctx := newResolutionContext(state, testBalance())
	if !fiefGroupContiguous(ctx, []models.TerritoryID{"AAA", "BBB", "CCC"}) {
		t.Error("AAA-BBB-CCC should be contiguous")
	}
	if fiefGroupContiguous(ctx, []models.TerritoryID{"AAA", "CCC", "DDD"}) {
		t.Error("AAA-CCC-DDD should not be contiguous without BBB")
	}
}
