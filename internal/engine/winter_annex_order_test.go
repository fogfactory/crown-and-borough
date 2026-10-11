package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// annexTestState builds on foundFiefTestState with a barony F1 AAA/BBB/CCC
// held by N1 (P1). DDD is next to CCC and held by a P1 army.
func annexTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := foundFiefTestState(t)
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "AAA",
		Territories: []models.TerritoryID{"AAA", "BBB", "CCC"}, OwnerID: "P1",
		HolderNobleID: nobleIDPtr("N1"),
	}}
	return state
}

// donorFief makes P2 own a fief of the given size ending on DDD, with the
// capital on GGG.
func donorFief(state *models.GameState, members ...models.TerritoryID) {
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "F2", Title: models.FiefTitleBarony, CapitalTerritoryID: members[0],
		Territories: members, OwnerID: "P2",
	})
}

func applyAnnex(state *models.GameState, order models.WinterOrder, accords royalAccords) (*resolutionContext, Event) {
	order.ID = "O1"
	order.Type = models.WinterOrderTypeAnnexTerritory
	ctx := newResolutionContext(state, testBalance())
	ctx.royalAccords = accords
	annexTerritoryOrder{order: order}.Apply(&ExecutionContext{resolution: ctx, playerID: "P1"})
	return ctx, ctx.events[len(ctx.events)-1]
}

func annexOrder() models.WinterOrder {
	return models.WinterOrder{NobleCode: "HUG", TerritoryID: "AAA", TargetID: "DDD"}
}

func TestAnnexTerritoryOutsideFief(t *testing.T) {
	state := annexTestState(t)
	ctx, event := applyAnnex(state, annexOrder(), royalAccords{})
	if event.Type != EventTypeFiefAnnexed {
		t.Fatalf("event = %#v, want fief_annexed", event)
	}
	fief := ctx.fiefByCapital("AAA")
	if want := []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"}; !slices.Equal(fief.Territories, want) {
		t.Errorf("territories = %v, want %v", fief.Territories, want)
	}
	if fief.Title != models.FiefTitleBarony {
		t.Errorf("title = %q, want barony: annexing never changes the title", fief.Title)
	}
	if want := ctx.balance.Costs.AnnexPerTerritory; event.ResourceSpent != want {
		t.Errorf("ResourceSpent = %d, want %d", event.ResourceSpent, want)
	}
	if event.TerritoryID != "DDD" || event.PreviousOwnerID != "" {
		t.Errorf("event = %#v, want territory DDD and no previous owner", event)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state invalid after annexation: %v", err)
	}
}

func TestAnnexTerritoryOfAnotherFiefWithAccord(t *testing.T) {
	state := annexTestState(t)
	donorFief(state, "GGG", "FFF", "EEE", "DDD")
	accords := royalAccords{annex: map[annexAccordKey]bool{{capital: "AAA", territory: "DDD"}: true}}
	ctx, event := applyAnnex(state, annexOrder(), accords)
	if event.Type != EventTypeFiefAnnexed || event.PreviousOwnerID != "P2" {
		t.Fatalf("event = %#v, want fief_annexed from P2", event)
	}
	donor := ctx.fiefByCapital("GGG")
	if want := []models.TerritoryID{"GGG", "FFF", "EEE"}; !slices.Equal(donor.Territories, want) {
		t.Errorf("donor territories = %v, want %v", donor.Territories, want)
	}
	if donor.Title != models.FiefTitleBarony {
		t.Errorf("donor title = %q, want it untouched", donor.Title)
	}
	if owner, _ := state.FiefOwnerAt("DDD"); owner != "P1" {
		t.Errorf("DDD controlled by %q, want P1", owner)
	}
}

func TestAnnexTerritoryRejections(t *testing.T) {
	grant := royalAccords{annex: map[annexAccordKey]bool{{capital: "AAA", territory: "DDD"}: true}}
	cases := []struct {
		name    string
		mutate  func(state *models.GameState)
		order   func(order *models.WinterOrder)
		accords royalAccords
		reason  string
	}{
		{name: "unknown fief", order: func(o *models.WinterOrder) { o.TerritoryID = "BBB" }, reason: "fief_not_found"},
		{name: "fief of another player", mutate: func(s *models.GameState) { s.Fiefs[0].OwnerID = "P2" }, reason: "fief_not_owned"},
		{name: "unknown noble", order: func(o *models.WinterOrder) { o.NobleCode = "ZZZ" }, reason: "unknown_noble"},
		{name: "noble of another player", mutate: func(s *models.GameState) { addNoble(s, "N2", "BOB", "P2", "AAA") }, order: func(o *models.WinterOrder) { o.NobleCode = "BOB" }, reason: "annex_not_titular"},
		{name: "not the titleholder", mutate: func(s *models.GameState) { addNoble(s, "N2", "ANN", "P1", "AAA") }, order: func(o *models.WinterOrder) { o.NobleCode = "ANN" }, reason: "annex_not_titular"},
		{name: "vacant fief", mutate: func(s *models.GameState) { s.Fiefs[0].HolderNobleID = nil }, reason: "annex_not_titular"},
		{name: "unknown territory", order: func(o *models.WinterOrder) { o.TargetID = "ZZZ" }, reason: "unknown_territory"},
		{name: "already a member", order: func(o *models.WinterOrder) { o.TargetID = "BBB" }, reason: "annex_already_member"},
		{name: "not adjacent", order: func(o *models.WinterOrder) { o.TargetID = "EEE" }, reason: "annex_not_adjacent"},
		{name: "not occupied", mutate: func(s *models.GameState) { removeAnchorArmy(s, "DDD") }, reason: "annex_not_occupied"},
		{name: "occupied by another player", mutate: func(s *models.GameState) { placeArmyAt(s, "A9", "P2", "DDD", 2) }, reason: "annex_not_occupied"},
		{name: "occupied by a neutral army", mutate: func(s *models.GameState) { placeArmyAt(s, "A9", models.NeutralPlayerID, "DDD", 2) }, reason: "annex_not_occupied"},
		{
			name:   "capital of a fief",
			mutate: func(s *models.GameState) { donorFief(s, "DDD", "EEE", "FFF") },
			reason: "annex_capital", accords: grant,
		},
		{
			name:   "member of a fief without accord",
			mutate: func(s *models.GameState) { donorFief(s, "GGG", "FFF", "EEE", "DDD") },
			reason: "annex_royal_accord_required",
		},
		{
			name:    "accord for another territory",
			mutate:  func(s *models.GameState) { donorFief(s, "GGG", "FFF", "EEE", "DDD") },
			accords: royalAccords{annex: map[annexAccordKey]bool{{capital: "AAA", territory: "EEE"}: true}},
			reason:  "annex_royal_accord_required",
		},
		{
			name:    "donor fief too small",
			mutate:  func(s *models.GameState) { donorFief(s, "GGG", "FFF", "DDD") },
			accords: grant,
			reason:  "annex_donor_too_small",
		},
		{
			name: "insufficient resources",
			mutate: func(s *models.GameState) {
				territoryState := s.TerritoryStates["AAA"]
				territoryState.Resources = 1
				s.TerritoryStates["AAA"] = territoryState
			},
			reason: "insufficient_resources",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := annexTestState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			order := annexOrder()
			if tc.order != nil {
				tc.order(&order)
			}
			before := slices.Clone(state.Fiefs[0].Territories)
			stock := state.TerritoryStates["AAA"].Resources
			_, event := applyAnnex(state, order, tc.accords)
			if event.Type != EventTypeRejected || event.Reason != tc.reason {
				t.Fatalf("event = %#v, want rejected %s", event, tc.reason)
			}
			if event.ResourceSpent != 0 || state.TerritoryStates["AAA"].Resources != stock {
				t.Errorf("a rejected annexation must not spend resources")
			}
			if !slices.Equal(state.Fiefs[0].Territories, before) {
				t.Errorf("fief territories changed to %v on a rejection", state.Fiefs[0].Territories)
			}
		})
	}
}

// TestAnnexTerritoryThroughWinterResolution runs T X then T R as a whole
// winter: the annexation keeps the barony, the rank change then raises it by
// one rank.
func TestAnnexTerritoryThroughWinterResolution(t *testing.T) {
	state := annexTestState(t)
	state.Fiefs[0].Territories = []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"}
	setTerritoryOwner(state, "EEE", "P1")
	resolution, err := ResolveWinter(state, testBalance(), map[models.PlayerID][]models.WinterOrder{"P1": {
		{ID: "O1", Type: models.WinterOrderTypeAnnexTerritory, NobleCode: "HUG", TerritoryID: "AAA", TargetID: "EEE"},
		{ID: "O2", Type: models.WinterOrderTypeChangeRank, NobleCode: "HUG", TerritoryID: "AAA"},
	}})
	if err != nil {
		t.Fatalf("ResolveWinter() = %v", err)
	}
	fief := resolution.State.Fiefs[0]
	if len(fief.Territories) != 5 || fief.Title != models.FiefTitleCounty {
		t.Errorf("fief = %#v, want 5 territories and the county", fief)
	}
	var kinds []EventType
	for _, event := range resolution.Events {
		if event.OrderID != "" {
			kinds = append(kinds, event.Type)
		}
	}
	if want := []EventType{EventTypeFiefAnnexed, EventTypeFiefRankChanged}; !slices.Equal(kinds, want) {
		t.Errorf("order events = %v, want %v", kinds, want)
	}
	report := BuildReport(state, resolution.State, resolution.Events, nil)
	if report.Winter == nil || len(report.Winter.Investments) < 2 {
		t.Fatalf("winter report = %#v, want both fief orders listed", report.Winter)
	}
}
