package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// justiceTestState is assassinationTestState with a Justice card for P1 and a
// previous assassination: the P2 noble TWO commissioned the death of the dead
// noble DEAD.
func justiceTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := assassinationTestState(t)
	state.SpecialDeck.Cards = []models.SpecialCard{{ID: "C1", Kind: models.CardKindJustice}}
	state.RemovedNobles = append(state.RemovedNobles, models.RemovedNoble{
		ID: "N9", Code: "DED", Name: "Dead", Sex: models.SexMale, OwnerID: "P3", Cause: models.DeathCauseAssassination,
		Commissioner: "N2", Author: "P2", Turn: 1,
	})
	return state
}

func playJustice(target models.NobleID) map[models.PlayerID][]models.DeckOrder {
	return map[models.PlayerID][]models.DeckOrder{"P1": {{
		ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindJustice, TargetNobleID: target,
	}}}
}

func TestJusticeExecutesACommissioner(t *testing.T) {
	state := justiceTestState(t)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playJustice("N2"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if hasNoble(resolution.State, "N2") {
		t.Fatalf("the commissioner must be executed")
	}
	last := resolution.State.RemovedNobles[len(resolution.State.RemovedNobles)-1]
	if last.ID != "N2" || last.Cause != models.DeathCauseExecution {
		t.Fatalf("removed = %#v, want N2 executed", last)
	}
	events := eventsOfType(resolution.Events, EventTypeJustice)
	if len(events) != 1 || events[0].Reason != JusticeExecutedReason || events[0].NobleCode != "TWO" || events[0].OwnerID != "P1" {
		t.Fatalf("justice events = %#v", events)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 0 {
		t.Fatalf("hand = %#v, want the card consumed", got)
	}
}

func TestJusticeOnAnInnocentIsConsumedWithoutEffect(t *testing.T) {
	state := justiceTestState(t)
	validateTestState(t, state)
	// ONE, the player's own noble, commissioned nothing.
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playJustice("N1"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if !hasNoble(resolution.State, "N1") {
		t.Fatalf("an innocent noble must survive")
	}
	events := eventsOfType(resolution.Events, EventTypeJustice)
	if len(events) != 1 || events[0].Reason != JusticeNoEffectReason {
		t.Fatalf("justice events = %#v", events)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 0 {
		t.Fatalf("hand = %#v, want the card consumed", got)
	}
}

func TestJusticeNeverTargetsTheKing(t *testing.T) {
	state := justiceTestState(t)
	king := models.NobleID("N2")
	state.King = &king
	validateTestState(t, state)
	if _, err := ResolveWithDeckOrders(state, testBalance(), playJustice("N2")); err == nil {
		t.Fatalf("a justice card on the King must be rejected")
	}
}

func TestJusticeOnADeadNobleIsRejectedAndKeepsTheCard(t *testing.T) {
	state := justiceTestState(t)
	validateTestState(t, state)
	if _, err := ResolveWithDeckOrders(state, testBalance(), playJustice("N9")); err == nil {
		t.Fatalf("a dead target must be rejected at validation")
	}
}

func TestJusticeFollowsAnAssassinationOfTheSameTurn(t *testing.T) {
	state := justiceTestState(t)
	state.RemovedNobles = nil
	addNoble(state, "N3", "THR", "P3", "CCC")
	validateTestState(t, state)
	// Justice is submitted before the assassination but resolves after it: ONE
	// commissions the death of THR and is executed the same turn.
	orders := map[models.PlayerID][]models.DeckOrder{"P1": {
		{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindJustice, TargetNobleID: "N1"},
		{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindAssassination, CommissionerNobleID: "N1", TargetNobleID: "N3"},
	}}
	state.SpecialDeck.Cards = []models.SpecialCard{{ID: "C1", Kind: models.CardKindJustice}, {ID: "C2", Kind: models.CardKindAssassination}}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1", "C2"}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), orders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if hasNoble(resolution.State, "N1") || hasNoble(resolution.State, "N3") {
		t.Fatalf("both the victim and the commissioner must be dead")
	}
}

func TestWinterJusticeFollowsTheWinterAssassination(t *testing.T) {
	state := winterAssassinationState(t)
	state.SpecialDeck.Cards = []models.SpecialCard{{ID: "C1", Kind: models.CardKindJustice}, {ID: "C2", Kind: models.CardKindAssassination}}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1", "C2"}
	// Justice is listed first but applies after the assassination: HUG (N1)
	// commissions the death of ABE (N3) and is executed the same winter.
	deckOrders := map[models.PlayerID][]models.DeckOrder{"P1": {
		{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindJustice, TargetNobleID: "N1"},
		{ID: "O2", Type: models.DeckOrderTypePlay, Kind: models.CardKindAssassination, CommissionerNobleID: "N1", TargetNobleID: "N3"},
	}}
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, deckOrders)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if hasNoble(resolution.State, "N1") || hasNoble(resolution.State, "N3") {
		t.Fatalf("the victim and the commissioner must both be dead")
	}
	events := eventsOfType(resolution.Events, EventTypeJustice)
	if len(events) != 1 || events[0].Reason != JusticeExecutedReason {
		t.Fatalf("justice events = %#v", events)
	}
}

// winterJusticeState is winterTrialState with two Justice cards in P1's hand
// and a past assassination commissioned by ABE (N3, P3).
func winterJusticeState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterTrialState(t)
	state.RemovedNobles = append(state.RemovedNobles, models.RemovedNoble{
		ID: "N9", Code: "DED", Name: "Dead", Sex: models.SexMale, OwnerID: "P2", Cause: models.DeathCauseAssassination,
		Commissioner: "N3", Author: "P3", Turn: 1,
	})
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindJustice}, {ID: "C2", Kind: models.CardKindJustice}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1", "C2"}},
	}
	return state
}

func TestWinterJusticeExecutesAndSpendsTheCard(t *testing.T) {
	state := winterJusticeState(t)
	validateTestState(t, state)
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, playJustice("N3"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if hasNoble(resolution.State, "N3") {
		t.Fatalf("the commissioner must be executed")
	}
	events := eventsOfType(resolution.Events, EventTypeJustice)
	if len(events) != 1 || events[0].Reason != JusticeExecutedReason || events[0].Phase != winterPhase {
		t.Fatalf("justice events = %#v", events)
	}
}

func TestWinterJusticeOnTheKingIsRejectedAndKeepsTheCard(t *testing.T) {
	state := winterJusticeState(t)
	king := models.NobleID("N3")
	state.King = &king
	validateTestState(t, state)
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, playJustice("N3"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !hasNoble(resolution.State, "N3") {
		t.Fatalf("the King must survive")
	}
	if reasons := electionRejections(resolution.Events); reasons["O1"] != JusticeKingReason {
		t.Fatalf("rejections = %#v, want %q", reasons, JusticeKingReason)
	}
	if len(eventsOfType(resolution.Events, EventTypeDeckOrderPlayed)) != 0 {
		t.Fatalf("a rejected play must not consume the card")
	}
}

// A sheet that plays two Justice cards on the same noble must not tell, through
// the rejection of the second one, that the first executed: the draft preview
// would otherwise answer for free the question the card gambles on.
func TestWinterPreviewDoesNotRevealWhetherAJusticeExecuted(t *testing.T) {
	for name, commissioner := range map[string]bool{"commissioner": true, "innocent": false} {
		t.Run(name, func(t *testing.T) {
			state := winterJusticeState(t)
			if !commissioner {
				state.RemovedNobles = nil
			}
			validateTestState(t, state)
			preview, err := PreviewOrders(state, winterDeckBalance(), "P1", OrdersInput{
				Winter: []WinterSubmission{{Player: "P1", Lines: "P JU ABE\nP JU ABE"}},
			})
			if err != nil {
				t.Fatalf("PreviewOrders: %v", err)
			}
			if len(preview.Winter) != 2 {
				t.Fatalf("winter lines = %#v", preview.Winter)
			}
			for _, line := range preview.Winter {
				if !line.Applied || line.Reason != "" {
					t.Fatalf("line %d = applied %t, reason %q: the preview must look the same whoever ABE is", line.Line, line.Applied, line.Reason)
				}
			}
		})
	}
}

// The supply projection resolves the viewer's drafted cards on a copy of the
// game: a Justice card must stay out of it, like the revolt roll, or the
// projected rations would tell whether its target is a commissioner.
func TestSupplyProjectionLeavesJusticeOut(t *testing.T) {
	state := justiceTestState(t)
	validateTestState(t, state)
	ctx, err := supplyQueryContext(state, testBalance(), "AAA", map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindJustice, TargetNobleID: "N2"}},
	})
	if err != nil {
		t.Fatalf("supplyQueryContext: %v", err)
	}
	if ctx.noblesByID["N2"] == nil {
		t.Fatalf("the projection executed the target of a Justice card")
	}
}
