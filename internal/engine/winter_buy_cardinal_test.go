package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func buyCardinal(id models.OrderID, noble models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeBuyCardinal, NobleCode: noble}
}

// cardinalTestState gives P1 two bishops (HUG, OTO) and 20 R on AAA; the cap is
// 1 cardinal for 3 players.
func cardinalTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := electionTestState(t)
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N1"}, {Region: "R2", Noble: "N4"}}
	addInfrastructure(state, models.Infrastructure{ID: "VAAA", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	territoryState := state.TerritoryStates["AAA"]
	territoryState.Resources = 20
	state.TerritoryStates["AAA"] = territoryState
	return state
}

func TestBuyCardinalPaysAndPromotesAtInvestiture(t *testing.T) {
	state := cardinalTestState(t)
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {buyCardinal("O1", "HUG")}})
	if !resolution.State.IsCardinal("N1") {
		t.Fatalf("cardinals = %v, want N1", resolution.State.Cardinals)
	}
	purchases := eventsOfType(resolution.Events, EventTypeCardinalPurchase)
	if len(purchases) != 1 || purchases[0].ResourceSpent != 8 {
		t.Fatalf("purchase events = %#v, want one costing 8", purchases)
	}
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
}

func TestBuyCardinalRejections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.GameState)
		orders []models.WinterOrder
		want   string
	}{
		{"not a bishop", func(s *models.GameState) {}, []models.WinterOrder{buyCardinal("O1", "LEO")}, "noble_not_owned"},
		{"free noble", func(s *models.GameState) { s.Bishops = nil }, []models.WinterOrder{buyCardinal("O1", "HUG")}, "cardinal_requires_bishop"},
		{"already cardinal", func(s *models.GameState) { s.Cardinals = []models.NobleID{"N1"} }, []models.WinterOrder{buyCardinal("O1", "HUG")}, "already_cardinal"},
		{"cap counts pending purchases", func(s *models.GameState) {}, []models.WinterOrder{buyCardinal("O0", "HUG"), buyCardinal("O1", "OTO")}, "cardinal_cap_reached"},
		{"duplicate order", func(s *models.GameState) {}, []models.WinterOrder{buyCardinal("O0", "HUG"), buyCardinal("O1", "HUG")}, "already_cardinal"},
		{"insufficient resources", func(s *models.GameState) {
			ts := s.TerritoryStates["AAA"]
			ts.Resources = 7
			s.TerritoryStates["AAA"] = ts
		}, []models.WinterOrder{buyCardinal("O1", "HUG")}, "insufficient_resources"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := cardinalTestState(t)
			test.mutate(state)
			resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": test.orders})
			if got := electionRejections(resolution.Events)["O1"]; got != test.want {
				t.Fatalf("reason = %q, want %q", got, test.want)
			}
			if resolution.State.IsCardinal("N4") {
				t.Fatalf("N4 must not be promoted")
			}
		})
	}
}

func cardinalCardOrder(id models.OrderID, noble models.NobleCode) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeDignity, NobleCode: noble, CardCode: models.DignityCardinalCardCode}
}

func giveCardinalCard(state *models.GameState, playerID models.PlayerID) models.NobleCardID {
	return giveCard(state, playerID, models.NobleCard{Kind: models.NobleCardKindDignity, Code: models.DignityCardinalCardCode, Dignity: models.DignityCardinal})
}

func TestCardinalCardIsFreeAndIndependentFromPurchaseCap(t *testing.T) {
	state := cardinalTestState(t)
	cardID := giveCardinalCard(state, "P1")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {buyCardinal("O1", "OTO"), cardinalCardOrder("O2", "HUG")},
	})
	// 3 players: one purchase (cap 1) and one card are both allowed.
	if !resolution.State.IsCardinal("N1") || !resolution.State.IsCardinal("N4") {
		t.Fatalf("cardinals = %v, want N1 (card) and N4 (purchase)", resolution.State.Cardinals)
	}
	if reasons := electionRejections(resolution.Events); len(reasons) != 0 {
		t.Fatalf("rejections = %#v", reasons)
	}
	if !resolution.State.Nobles[0].Has(models.DignityCardinal) || resolution.State.Nobles[3].Has(models.DignityCardinal) {
		t.Fatalf("only the card cardinal carries the dignity")
	}
	deck := resolution.State.NobleDeck
	if len(deck.Played) != 1 || deck.Played[0].Card != cardID || len(deck.Hands["P1"]) != 0 {
		t.Fatalf("deck = %+v, want the card lying on N1", deck)
	}
	if got := PurchasedCardinals(resolution.State); got != 1 {
		t.Fatalf("purchased cardinals = %d, want 1", got)
	}
	if spent := eventsOfType(resolution.Events, EventTypeCardinalPurchase); len(spent) != 1 || spent[0].ResourceSpent != 8 {
		t.Fatalf("only the purchase is paid: %#v", spent)
	}
}

func TestCardinalCardRejections(t *testing.T) {
	state := cardinalTestState(t)
	giveCardinalCard(state, "P1")
	giveCardinalCard(state, "P1")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {cardinalCardOrder("O1", "LEO"), cardinalCardOrder("O2", "HUG"), cardinalCardOrder("O3", "HUG")},
	})
	reasons := electionRejections(resolution.Events)
	if reasons["O1"] != "noble_not_owned" || reasons["O3"] != "already_cardinal" || len(reasons) != 2 {
		t.Fatalf("rejections = %#v", reasons)
	}
	if len(resolution.State.NobleDeck.Hands["P1"]) != 1 {
		t.Fatalf("a rejected card must stay in hand")
	}
}

func TestExcommunicatedCardCardinalGivesCardBackAndPurchasedFreesItsPlace(t *testing.T) {
	state := cardinalTestState(t)
	cardID := giveCardinalCard(state, "P1")
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {buyCardinal("O1", "OTO"), cardinalCardOrder("O2", "HUG")},
	})
	next := resolution.State
	next.Turn += 4
	ctx := newResolutionContext(next, testBalance())
	ctx.excommunicate(models.Excommunication{Noble: "N1", Reason: models.ExcommunicationExOfficio, Turn: next.Turn})
	ctx.excommunicate(models.Excommunication{Noble: "N4", Reason: models.ExcommunicationExOfficio, Turn: next.Turn})
	deck := ctx.state.NobleDeck
	if len(ctx.state.Cardinals) != 0 || ctx.state.Nobles[0].Has(models.DignityCardinal) {
		t.Fatalf("cardinals = %v, dignities = %v, want both titles gone", ctx.state.Cardinals, ctx.state.Nobles[0].Dignities)
	}
	if len(deck.Played) != 0 || len(deck.Discard) != 1 || deck.Discard[0] != cardID {
		t.Fatalf("deck = %+v, want the card back in the discard", deck)
	}
	if got := PurchasedCardinals(ctx.state); got != 0 {
		t.Fatalf("purchased cardinals = %d, want the place free again", got)
	}
	validateTestState(t, ctx.state)
}

// castFor runs the vacant R2 election with P1 backing OTO and returns the
// voices P1 cast, after the given extra P1 orders.
func castFor(t *testing.T, extra ...models.WinterOrder) int {
	t.Helper()
	state := cardinalTestState(t)
	state.Bishops = []models.Bishop{{Region: "R1", Noble: "N1"}}
	giveCardinalCard(state, "P1")
	orders := append(extra, candidacy("K1", "OTO", "DDD"), ballot("K2", "OTO", "DDD"))
	resolution := resolveElectionSheets(t, state, map[models.PlayerID][]models.WinterOrder{"P1": orders})
	for _, event := range eventsOfType(resolution.Events, EventTypeElectionResult) {
		if event.Election.Region == "R2" {
			return event.Election.Cast
		}
	}
	t.Fatalf("no result for R2")
	return 0
}

func TestCardinalCardVotesInTheSameWinterButPurchaseDoesNot(t *testing.T) {
	base := castFor(t)
	if got := castFor(t, buyCardinal("O1", "HUG")); got != base {
		t.Fatalf("voices after a purchase = %d, want unchanged %d (title conferred at the investiture)", got, base)
	}
	// A bishop weighs 1 voice, a cardinal 2.
	if got := castFor(t, cardinalCardOrder("O1", "HUG")); got != base+1 {
		t.Fatalf("voices after a card = %d, want %d (cardinal at once)", got, base+1)
	}
}
