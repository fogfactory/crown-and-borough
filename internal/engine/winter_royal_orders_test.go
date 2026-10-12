package engine

import (
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// royalOrdersState builds on annexTestState: N1 (HUG, P1) is the king and holds
// the barony F1 (AAA, BBB, CCC); P2 owns the noble ANN.
func royalOrdersState(t *testing.T) *models.GameState {
	t.Helper()
	state := annexTestState(t)
	addNoble(state, "N2", "ANN", "P2", "GGG")
	king := models.NobleID("N1")
	state.King = &king
	return state
}

type royalSheets = map[models.PlayerID][]models.WinterOrder

func royalOrderOf(orderType models.WinterOrderType, id models.OrderID, mutate func(*models.WinterOrder)) models.WinterOrder {
	order := models.WinterOrder{ID: id, Type: orderType, NobleCode: "HUG", TerritoryID: "AAA"}
	if mutate != nil {
		mutate(&order)
	}
	return order
}

// resolveRoyal runs stage 2a on the sheets, then the management orders of
// the sheets, like the winter resolver does.
func resolveRoyal(state *models.GameState, sheets royalSheets) *resolutionContext {
	ctx := newResolutionContext(state, testBalance())
	ctx.resolveRoyalOrders(sheets, nil)
	for _, playerID := range sortedPlayerIDs(state.Players) {
		for _, order := range sheets[playerID] {
			if !order.Type.IsRoyal() {
				executeWinterOrder(ctx, playerID, order, nil)
			}
		}
	}
	return ctx
}

func eventOfOrder(t *testing.T, ctx *resolutionContext, id models.OrderID) Event {
	t.Helper()
	for _, event := range ctx.events {
		if event.OrderID == id {
			return event
		}
	}
	t.Fatalf("no event for order %s in %#v", id, ctx.events)
	return Event{}
}

func requireRejected(t *testing.T, ctx *resolutionContext, id models.OrderID, reason string) {
	t.Helper()
	event := eventOfOrder(t, ctx, id)
	if event.Type != EventTypeRejected || event.Reason != reason {
		t.Fatalf("order %s event = %#v, want rejected for %q", id, event, reason)
	}
}

func TestRoyalRankAccordMakesRankChangeFree(t *testing.T) {
	state := rankTestState(t)
	king := models.NobleID("N1")
	state.King = &king
	sheets := royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", nil),
		royalOrderOf(models.WinterOrderTypeChangeRank, "O2", nil),
	}}
	ctx := resolveRoyal(state, sheets)
	if event := eventOfOrder(t, ctx, "O1"); event.Type != EventTypeRoyalAccord || event.ResourceSpent != 0 {
		t.Fatalf("W R event = %#v, want a free royal_accord", event)
	}
	if event := eventOfOrder(t, ctx, "O2"); event.Type != EventTypeFiefRankChanged || event.ResourceSpent != 0 {
		t.Fatalf("T R event = %#v, want a free fief_rank_changed", event)
	}
	if state.TerritoryStates["AAA"].Resources != 50 {
		t.Errorf("stock = %d, want it untouched", state.TerritoryStates["AAA"].Resources)
	}
}

func TestRoyalRankAccordWithoutRankChangeIsConsumedWithoutEffect(t *testing.T) {
	state := royalOrdersState(t)
	ctx := resolveRoyal(state, royalSheets{"P1": {royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", nil)}})
	requireRejected(t, ctx, "O1", "royal_accord_unused")
	if ctx.hasRankAccord("AAA") {
		t.Error("an unused W R must not leave an accord behind")
	}
}

func TestRoyalRankAccordIgnoresTheRankChangeOfAnotherFief(t *testing.T) {
	state := royalOrdersState(t)
	sheets := royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", nil),
		royalOrderOf(models.WinterOrderTypeChangeRank, "O2", func(o *models.WinterOrder) { o.TerritoryID = "BBB" }),
	}}
	requireRejected(t, resolveRoyal(state, sheets), "O1", "royal_accord_unused")
}

func TestRoyalAnnexAccordLetsAFiefAnnexAnotherFiefTerritory(t *testing.T) {
	state := royalOrdersState(t)
	donorFief(state, "GGG", "FFF", "EEE", "DDD")
	sheets := royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalAnnex, "O1", func(o *models.WinterOrder) { o.TargetID = "DDD" }),
		royalOrderOf(models.WinterOrderTypeAnnexTerritory, "O2", func(o *models.WinterOrder) { o.TargetID = "DDD" }),
	}}
	ctx := resolveRoyal(state, sheets)
	if event := eventOfOrder(t, ctx, "O1"); event.Type != EventTypeRoyalAccord || event.TerritoryID != "AAA" {
		t.Fatalf("W X event = %#v, want a royal_accord on AAA", event)
	}
	if event := eventOfOrder(t, ctx, "O2"); event.Type != EventTypeFiefAnnexed || event.PreviousOwnerID != "P2" {
		t.Fatalf("T X event = %#v, want fief_annexed from P2", event)
	}
	if want := []models.TerritoryID{"AAA", "BBB", "CCC", "DDD"}; !slices.Equal(ctx.fiefByCapital("AAA").Territories, want) {
		t.Errorf("territories = %v, want %v", ctx.fiefByCapital("AAA").Territories, want)
	}
}

func TestRoyalAnnexAccordWithoutAnnexationIsConsumedWithoutEffect(t *testing.T) {
	state := royalOrdersState(t)
	donorFief(state, "GGG", "FFF", "EEE", "DDD")
	sheets := royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalAnnex, "O1", func(o *models.WinterOrder) { o.TargetID = "DDD" }),
		// The annexation targets another territory: the accord does not cover it.
		royalOrderOf(models.WinterOrderTypeAnnexTerritory, "O2", func(o *models.WinterOrder) { o.TargetID = "EEE" }),
	}}
	ctx := resolveRoyal(state, sheets)
	requireRejected(t, ctx, "O1", "royal_accord_unused")
	if event := eventOfOrder(t, ctx, "O2"); event.Reason == "" {
		t.Errorf("T X event = %#v, want a rejection", event)
	}
}

func TestRoyalFoundAccordMakesFoundingFree(t *testing.T) {
	cases := []struct {
		name     string
		royal    bool
		wantCost int
	}{
		{name: "with accord", royal: true, wantCost: 0},
		{name: "without accord", royal: false, wantCost: 3 * 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := foundFiefTestState(t)
			king := models.NobleID("N1")
			state.King = &king
			sheet := []models.WinterOrder{}
			if tc.royal {
				sheet = append(sheet, royalOrderOf(models.WinterOrderTypeRoyalFound, "O1", nil))
			}
			sheet = append(sheet, models.WinterOrder{
				ID: "O2", Type: models.WinterOrderTypeFoundFief, NobleCode: "HUG",
				TerritoryID: "AAA", TerritoryIDs: []models.TerritoryID{"AAA", "BBB", "CCC"},
			})
			ctx := resolveRoyal(state, royalSheets{"P1": sheet})
			event := eventOfOrder(t, ctx, "O2")
			if event.Type != EventTypeFiefFounded || event.ResourceSpent != tc.wantCost {
				t.Fatalf("T F event = %#v, want fief_founded costing %d", event, tc.wantCost)
			}
		})
	}
}

func TestRoyalFoundAccordWithoutFoundingIsConsumedWithoutEffect(t *testing.T) {
	state := royalOrdersState(t)
	ctx := resolveRoyal(state, royalSheets{"P1": {royalOrderOf(models.WinterOrderTypeRoyalFound, "O1", nil)}})
	requireRejected(t, ctx, "O1", "royal_accord_unused")
}

// crownFiefState makes the fief F1 a crown fief: vacant, and attached to a
// player (P3) with no noble left.
func crownFiefState(t *testing.T) *models.GameState {
	t.Helper()
	state := royalOrdersState(t)
	state.Fiefs[0].HolderNobleID = nil
	state.Fiefs[0].OwnerID = "P3"
	// The fief now controls P1's former capital: P1 holds no capital castle.
	state.Players[0].CapitalCastleID = nil
	return state
}

func grantOrder(mutate func(*models.WinterOrder)) models.WinterOrder {
	return royalOrderOf(models.WinterOrderTypeRoyalGrant, "O1", func(o *models.WinterOrder) {
		o.TargetCode = "ANN"
		if mutate != nil {
			mutate(o)
		}
	})
}

func TestRoyalGrantGivesACrownFiefToANobleOfAnotherPlayer(t *testing.T) {
	state := crownFiefState(t)
	ctx := resolveRoyal(state, royalSheets{"P1": {grantOrder(nil)}})
	event := eventOfOrder(t, ctx, "O1")
	if event.Type != EventTypeFiefGranted || event.NobleCode != "ANN" || event.PreviousOwnerID != "P3" || event.ResourceSpent != 0 {
		t.Fatalf("event = %#v, want a free fief_granted to ANN", event)
	}
	fief := ctx.fiefByCapital("AAA")
	if fief.OwnerID != "P2" || fief.HolderNobleID == nil || *fief.HolderNobleID != "N2" {
		t.Errorf("fief = %#v, want it held by N2 of P2", fief)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state invalid after the grant: %v", err)
	}
}

func TestRoyalGrantToANobleOfTheKing(t *testing.T) {
	state := crownFiefState(t)
	ctx := resolveRoyal(state, royalSheets{"P1": {grantOrder(func(o *models.WinterOrder) { o.TargetCode = "HUG" })}})
	fief := ctx.fiefByCapital("AAA")
	if event := eventOfOrder(t, ctx, "O1"); event.Type != EventTypeFiefGranted || fief.OwnerID != "P1" || *fief.HolderNobleID != "N1" {
		t.Fatalf("event = %#v, fief = %#v, want the fief granted to HUG of P1", event, fief)
	}
}

func TestRoyalGrantRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(state *models.GameState)
		order  func(order *models.WinterOrder)
		reason string
	}{
		{name: "unknown fief", order: func(o *models.WinterOrder) { o.TerritoryID = "BBB" }, reason: "fief_not_found"},
		{name: "fief held by a titleholder", mutate: func(s *models.GameState) { s.Fiefs[0].HolderNobleID = nobleIDPtr("N1") }, reason: "royal_fief_not_crown"},
		{name: "vacant fief of a player with a noble", mutate: func(s *models.GameState) { s.Fiefs[0].OwnerID = "P2" }, reason: "royal_fief_not_crown"},
		{name: "unknown grantee", order: func(o *models.WinterOrder) { o.TargetCode = "ZZZ" }, reason: "unknown_noble"},
		{
			// HUG ranks above BOB in the line of P1 and holds no fief.
			name:   "grantee blocked by the line of succession",
			mutate: func(s *models.GameState) { addNoble(s, "N3", "BOB", "P1", "AAA") },
			order:  func(o *models.WinterOrder) { o.TargetCode = "BOB" },
			reason: "succession_rank_blocked",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := crownFiefState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			ctx := resolveRoyal(state, royalSheets{"P1": {grantOrder(tc.order)}})
			requireRejected(t, ctx, "O1", tc.reason)
			if fief := ctx.fiefByCapital("AAA"); fief != nil && fief.HolderNobleID != nil && tc.name != "fief held by a titleholder" {
				t.Errorf("fief = %#v, want it left vacant", fief)
			}
		})
	}
}

func TestRoyalOrderRejections(t *testing.T) {
	rank := func(mutate func(*models.WinterOrder)) models.WinterOrder {
		return royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", mutate)
	}
	cases := []struct {
		name   string
		mutate func(state *models.GameState)
		order  models.WinterOrder
		reason string
	}{
		{name: "unknown noble", order: rank(func(o *models.WinterOrder) { o.NobleCode = "ZZZ" }), reason: "unknown_noble"},
		{name: "no king", mutate: func(s *models.GameState) { s.King = nil }, order: rank(nil), reason: "royal_not_issuer"},
		{name: "king in a dungeon without queen", mutate: func(s *models.GameState) { setNobleStatus(s, "N1", models.NobleStatusDungeon) }, order: rank(nil), reason: "royal_not_issuer"},
		{name: "noble of the player who is not the king", mutate: func(s *models.GameState) { addNoble(s, "N3", "BOB", "P1", "AAA") }, order: rank(func(o *models.WinterOrder) { o.NobleCode = "BOB" }), reason: "royal_not_issuer"},
		{name: "king of another player", mutate: func(s *models.GameState) { k := models.NobleID("N2"); s.King = &k }, order: rank(nil), reason: "royal_not_issuer"},
		{name: "unknown fief", order: rank(func(o *models.WinterOrder) { o.TerritoryID = "BBB" }), reason: "fief_not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := royalOrdersState(t)
			if tc.mutate != nil {
				tc.mutate(state)
			}
			ctx := resolveRoyal(state, royalSheets{"P1": {tc.order}})
			requireRejected(t, ctx, "O1", tc.reason)
		})
	}
}

func TestRoyalOrdersAreLimitedPerTypeAndWinter(t *testing.T) {
	state := rankTestState(t)
	king := models.NobleID("N1")
	state.King = &king
	sheets := royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", nil),
		royalOrderOf(models.WinterOrderTypeRoyalRank, "O2", nil),
		royalOrderOf(models.WinterOrderTypeChangeRank, "O3", nil),
	}}
	ctx := resolveRoyal(state, sheets)
	if event := eventOfOrder(t, ctx, "O1"); event.Type != EventTypeRoyalAccord {
		t.Fatalf("first W R event = %#v, want a royal_accord", event)
	}
	requireRejected(t, ctx, "O2", "royal_order_limit")

	// Another type is counted apart.
	state = royalOrdersState(t)
	state.Fiefs[0].HolderNobleID = nil
	ctx = resolveRoyal(state, royalSheets{"P1": {
		royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", nil),
		grantOrder(func(o *models.WinterOrder) {}),
	}})
	if event := eventOfOrder(t, ctx, "O1"); event.Type != EventTypeRejected || event.Reason != "royal_accord_unused" {
		t.Errorf("W R event = %#v, want it unused", event)
	}
}

func TestRegentIssuesTheRoyalOrders(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(state *models.GameState)
		want   bool
	}{
		{name: "queen of a king in a dungeon", mutate: func(s *models.GameState) { setNobleStatus(s, "N1", models.NobleStatusDungeon) }, want: true},
		{name: "dowager queen of a vacant throne", mutate: func(s *models.GameState) { s.King = nil }, want: true},
		{name: "queen in a dungeon too", mutate: func(s *models.GameState) {
			setNobleStatus(s, "N1", models.NobleStatusDungeon)
			setNobleStatus(s, "Q1", models.NobleStatusDungeon)
		}, want: false},
		{name: "king reigning", mutate: func(s *models.GameState) {}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := royalOrdersState(t)
			addNoble(state, "Q1", "MAH", "P2", "GGG")
			state.Marriages = []models.Marriage{{NobleA: "N1", NobleB: "Q1", Turn: 1}}
			queen := models.NobleID("Q1")
			state.Queen = &queen
			tc.mutate(state)
			sheets := royalSheets{"P2": {royalOrderOf(models.WinterOrderTypeRoyalRank, "O1", func(o *models.WinterOrder) { o.NobleCode = "MAH" })}}
			ctx := newResolutionContext(state, testBalance())
			ctx.resolveRoyalOrders(sheets, nil)
			event := eventOfOrder(t, ctx, "O1")
			if tc.want != (event.Reason != "royal_not_issuer") {
				t.Fatalf("event = %#v, regent allowed = %v", event, tc.want)
			}
		})
	}
}
