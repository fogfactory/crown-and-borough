package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// rankTestState builds on annexTestState with F1 grown to five territories
// (AAA..EEE) while still a barony; the title is independent of the size.
func rankTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := annexTestState(t)
	state.Fiefs[0].Territories = []models.TerritoryID{"AAA", "BBB", "CCC", "DDD", "EEE"}
	return state
}

func applyChangeRank(state *models.GameState, order models.WinterOrder, accords royalAccords) (*resolutionContext, Event) {
	order.ID = "O1"
	order.Type = models.WinterOrderTypeChangeRank
	ctx := newResolutionContext(state, testBalance())
	ctx.royalAccords = accords
	changeRankOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	return ctx, ctx.events[len(ctx.events)-1]
}

func TestChangeRankPaysEveryTerritory(t *testing.T) {
	state := rankTestState(t)
	ctx, event := applyChangeRank(state, models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA"}, royalAccords{})
	if event.Type != EventTypeFiefRankChanged || event.FiefTitle != models.FiefTitleCounty {
		t.Fatalf("event = %#v, want fief_rank_changed to county", event)
	}
	if want := 5 * ctx.balance.Costs.RankPerTerritory; event.ResourceSpent != want {
		t.Errorf("ResourceSpent = %d, want %d", event.ResourceSpent, want)
	}
	if got := ctx.fiefByCapital("AAA").Title; got != models.FiefTitleCounty {
		t.Errorf("title = %q, want county", got)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state invalid after rank change: %v", err)
	}
}

func TestChangeRankIsFreeWithRoyalAccord(t *testing.T) {
	state := rankTestState(t)
	_, event := applyChangeRank(state, models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA"},
		royalAccords{rank: map[models.TerritoryID]bool{"AAA": true}})
	if event.Type != EventTypeFiefRankChanged || event.ResourceSpent != 0 {
		t.Fatalf("event = %#v, want a free fief_rank_changed", event)
	}
	if state.TerritoryStates["AAA"].Resources != 50 {
		t.Errorf("stock = %d, want it untouched", state.TerritoryStates["AAA"].Resources)
	}
}

func TestChangeRankRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(state *models.GameState)
		order  func(order *models.WinterOrder)
		reason string
	}{
		{name: "unknown fief", order: func(o *models.WinterOrder) { o.TerritoryID = "BBB" }, reason: "fief_not_found"},
		{name: "fief of another player", mutate: func(s *models.GameState) { s.Fiefs[0].OwnerID = "P2" }, reason: "fief_not_owned"},
		{name: "unknown noble", order: func(o *models.WinterOrder) { o.NobleCode = "ZZZ" }, reason: "unknown_noble"},
		{name: "not the titleholder", mutate: func(s *models.GameState) { addNoble(s, "N2", "ANN", "P1", "AAA") }, order: func(o *models.WinterOrder) { o.NobleCode = "ANN" }, reason: "rank_not_titular"},
		{name: "already a duchy", mutate: func(s *models.GameState) { s.Fiefs[0].Title = models.FiefTitleDuchy }, reason: "rank_max"},
		{
			name: "insufficient resources",
			mutate: func(s *models.GameState) {
				territoryState := s.TerritoryStates["AAA"]
				territoryState.Resources = 14
				s.TerritoryStates["AAA"] = territoryState
			},
			reason: "insufficient_resources",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := rankTestState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			order := models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA"}
			if tc.order != nil {
				tc.order(&order)
			}
			title := state.Fiefs[0].Title
			stock := state.TerritoryStates["AAA"].Resources
			_, event := applyChangeRank(state, order, royalAccords{})
			if event.Type != EventTypeRejected || event.Reason != tc.reason {
				t.Fatalf("event = %#v, want rejected %s", event, tc.reason)
			}
			if state.Fiefs[0].Title != title || state.TerritoryStates["AAA"].Resources != stock {
				t.Errorf("a rejected rank change must change and spend nothing")
			}
		})
	}
}

func TestChangeRankBlockedBySuccessionRank(t *testing.T) {
	state := rankTestState(t)
	// HUG heads the line without any fief: ANN, who holds F1, may not rise.
	addNoble(state, "N2", "ANN", "P1", "AAA")
	state.Fiefs[0].HolderNobleID = nobleIDPtr("N2")
	_, event := applyChangeRank(state, models.WinterOrder{NobleCode: "ANN", TerritoryID: "AAA"}, royalAccords{})
	if event.Type != EventTypeRejected || event.Reason != "succession_rank_blocked" {
		t.Fatalf("event = %#v, want rejected succession_rank_blocked", event)
	}
}

func TestChangeRankOnlyOncePerWinter(t *testing.T) {
	state := rankTestState(t)
	order := models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeChangeRank, NobleCode: "HUG", TerritoryID: "AAA"}
	ctx := newResolutionContext(state, testBalance())
	ctx.royalAccords = royalAccords{rank: map[models.TerritoryID]bool{"AAA": true}}
	changeRankOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	changeRankOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	event := ctx.events[len(ctx.events)-1]
	if event.Type != EventTypeRejected || event.Reason != "rank_already_changed" {
		t.Fatalf("second order event = %#v, want rejected rank_already_changed", event)
	}
	if got := ctx.fiefByCapital("AAA").Title; got != models.FiefTitleCounty {
		t.Errorf("title = %q, want a single step to county", got)
	}
}

func TestChangeRankStepsOneRankWhateverTheSize(t *testing.T) {
	state := annexTestState(t)
	// A three-territory barony rises to a county although its size is 3.
	_, event := applyChangeRank(state, models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA"}, royalAccords{})
	if event.Type != EventTypeFiefRankChanged || event.FiefTitle != models.FiefTitleCounty {
		t.Fatalf("event = %#v, want county", event)
	}
	if want := 3 * testBalance().Costs.RankPerTerritory; event.ResourceSpent != want {
		t.Errorf("ResourceSpent = %d, want %d", event.ResourceSpent, want)
	}
}
