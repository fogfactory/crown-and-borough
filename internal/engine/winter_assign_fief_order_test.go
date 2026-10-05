package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// assignFiefTestState builds on foundFiefTestState with a vacant fief AAA/
// BBB/CCC owned by P1, and a second free noble ANN owned by P1 at BBB.
func assignFiefTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := foundFiefTestState(t)
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
	}, {
		// N1 heads the line and already holds a barony, so N2 may receive one.
		ID: "F0", Title: models.FiefTitleBarony, CapitalTerritoryID: "EEE",
		Territories: []models.TerritoryID{"EEE", "FFF", "GGG"}, OwnerID: "P1",
		HolderNobleID: nobleIDPtr("N1"),
	}}
	addNoble(state, "N2", "ANN", "P1", "BBB")
	return state
}

func TestAssignFiefOrderBlockedBySuccessionRank(t *testing.T) {
	cases := map[string]func(state *models.GameState){
		"head of line without title": func(state *models.GameState) { state.Fiefs[1].HolderNobleID = nil },
		"head of line lower title": func(state *models.GameState) {
			state.Fiefs[0].Title = models.FiefTitleCounty
			state.Fiefs[0].Territories = []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			state := assignFiefTestState(t)
			mutate(state)
			_, event := applyAssignFief(state, models.WinterOrder{NobleCode: "ANN", TerritoryID: "AAA"})
			if event.Type != EventTypeRejected || event.Reason != "succession_rank_blocked" {
				t.Fatalf("event = %#v, want rejected succession_rank_blocked", event)
			}
		})
	}
}

func applyAssignFief(state *models.GameState, order models.WinterOrder) (*resolutionContext, Event) {
	order.ID = "O1"
	order.Type = models.WinterOrderTypeAssignFief
	ctx := newResolutionContext(state, testBalance())
	assignFiefOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	return ctx, ctx.events[len(ctx.events)-1]
}

func TestAssignFiefOrderSuccess(t *testing.T) {
	state := assignFiefTestState(t)
	ctx, event := applyAssignFief(state, models.WinterOrder{NobleCode: "ANN", TerritoryID: "AAA"})
	if event.Type != EventTypeFiefAssigned {
		t.Fatalf("event = %#v, want fief_assigned", event)
	}
	if event.ResourceSpent != 0 {
		t.Errorf("ResourceSpent = %d, want 0", event.ResourceSpent)
	}
	if len(ctx.state.Fiefs) != 2 || ctx.state.Fiefs[0].HolderNobleID == nil || *ctx.state.Fiefs[0].HolderNobleID != "N2" {
		t.Fatalf("fiefs = %#v, want N2 as holder", ctx.state.Fiefs)
	}
	if err := state.Validate(); err != nil {
		t.Errorf("Validate() after assignment = %v", err)
	}
}

func TestAssignFiefOrderRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(state *models.GameState)
		order  models.WinterOrder
		reason string
	}{
		{
			name:   "fief not found",
			order:  models.WinterOrder{NobleCode: "ANN", TerritoryID: "ZZZ"},
			reason: "fief_not_found",
		},
		{
			name:   "capital not a fief",
			order:  models.WinterOrder{NobleCode: "ANN", TerritoryID: "DDD"},
			reason: "fief_not_found",
		},
		{
			name:   "fief not owned",
			mutate: func(state *models.GameState) { state.Fiefs[0].OwnerID = "P2" },
			order:  models.WinterOrder{NobleCode: "ANN", TerritoryID: "AAA"},
			reason: "fief_not_owned",
		},
		{
			name: "fief not vacant",
			mutate: func(state *models.GameState) {
				holder := models.NobleID("N1")
				state.Fiefs[0].HolderNobleID = &holder
			},
			order:  models.WinterOrder{NobleCode: "ANN", TerritoryID: "AAA"},
			reason: "fief_not_vacant",
		},
		{
			name:   "unknown noble",
			order:  models.WinterOrder{NobleCode: "ZZZ", TerritoryID: "AAA"},
			reason: "unknown_noble",
		},
		{
			name:   "holder not owned",
			mutate: func(state *models.GameState) { addNoble(state, "N3", "BOB", "P2", "BBB") },
			order:  models.WinterOrder{NobleCode: "BOB", TerritoryID: "AAA"},
			reason: "fief_holder_not_owned",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := assignFiefTestState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			_, event := applyAssignFief(state, tc.order)
			if event.Type != EventTypeRejected {
				t.Fatalf("event = %#v, want rejected", event)
			}
			if event.Reason != tc.reason {
				t.Errorf("reason = %q, want %q", event.Reason, tc.reason)
			}
			if event.ResourceSpent != 0 {
				t.Errorf("ResourceSpent = %d, want 0", event.ResourceSpent)
			}
		})
	}
}
