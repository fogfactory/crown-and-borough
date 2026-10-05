package engine

import (
	"reflect"
	"slices"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func deckOrdersState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterTestState(t, []models.Territory{territory("AAA", "Aaa")}, []models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "AAA", Size: 1}})
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeVillage, Level: 1, TerritoryID: "AAA"})
	setTerritoryOwner(state, "AAA", "P1")
	state.TerritoryStates["AAA"] = models.TerritoryState{Army: state.TerritoryStates["AAA"].Army, Infrastructures: infraPointer("I1"), Resources: 4}
	return state
}

func resolveNobleDeckWinter(t *testing.T, state *models.GameState, winter map[models.PlayerID][]models.WinterOrder) Resolution {
	t.Helper()
	validateTestState(t, state)
	resolution, err := ResolveWinter(state, testBalance(), winter)
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	return resolution
}

func rejectionReasons(events []Event) []string {
	reasons := []string{}
	for _, event := range eventsOfType(events, EventTypeRejected) {
		reasons = append(reasons, event.Reason)
	}
	return reasons
}

func TestDrawNobleOrderDrawsTopCardOncePerPlayerAndWinter(t *testing.T) {
	state := deckOrdersState(t)
	first := pileNobleCard(state, "ELE", "Eleonore", models.SexFemale)
	second := pileNobleCard(state, "GUI", "Guy", models.SexMale)
	third := pileNobleCard(state, "MAH", "Mahaut", models.SexFemale)
	draw := func(id models.OrderID) models.WinterOrder {
		return models.WinterOrder{ID: id, Type: models.WinterOrderTypeDrawNoble}
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {draw("O1"), draw("O2")},
		"P2": {draw("O1")},
	})
	deck := resolution.State.NobleDeck
	if hand := deck.Hands["P1"]; len(hand) != 1 || hand[0] != first {
		t.Errorf("P1 hand = %v, want the top card %s", hand, first)
	}
	if hand := deck.Hands["P2"]; len(hand) != 1 || hand[0] != second {
		t.Errorf("P2 hand = %v, want the next card %s", hand, second)
	}
	if len(deck.DrawPile) != 1 || deck.DrawPile[0] != third {
		t.Errorf("draw pile = %v, want only %s", deck.DrawPile, third)
	}
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "noble_draw_already_used" {
		t.Errorf("rejections = %v, want one noble_draw_already_used", reasons)
	}
	for _, event := range eventsOfType(resolution.Events, EventTypeNobleDraw) {
		if event.CardID != "" || event.NobleCode != "" {
			t.Errorf("draw event %+v names the drawn card, want it left out of the public report", event)
		}
	}
	if len(state.NobleDeck.DrawPile) != 3 {
		t.Error("ResolveWinter mutated its input deck")
	}

	// A new winter allows a new draw.
	next := cloneGameState(resolution.State)
	next.Turn += 4
	again, err := ResolveWinter(next, testBalance(), map[models.PlayerID][]models.WinterOrder{"P1": {draw("O1")}})
	if err != nil {
		t.Fatalf("ResolveWinter: %v", err)
	}
	if hand := again.State.NobleDeck.Hands["P1"]; len(hand) != 2 {
		t.Errorf("P1 hand = %v, want a second card the next winter", hand)
	}
}

func TestDrawNobleOrderRejectsEmptyDeck(t *testing.T) {
	for name, prepare := range map[string]func(*models.GameState){
		"no deck":    func(*models.GameState) {},
		"empty pile": func(state *models.GameState) { ensureNobleDeck(state) },
	} {
		t.Run(name, func(t *testing.T) {
			state := deckOrdersState(t)
			prepare(state)
			resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
				"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}},
			})
			if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "noble_deck_empty" {
				t.Errorf("rejections = %v, want noble_deck_empty", reasons)
			}
		})
	}
}

func TestDrawThenPlayNextWinter(t *testing.T) {
	state := deckOrdersState(t)
	pileNobleCard(state, "ELE", "Eleonore", models.SexFemale)
	drawn := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}},
	}).State
	drawn.Turn += 4
	played := resolveNobleDeckWinter(t, drawn, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, CardCode: "ELE", TerritoryID: "AAA"}},
	})
	if len(played.State.Nobles) != 1 || played.State.Nobles[0].Code != "ELE" {
		t.Fatalf("nobles = %+v, want the drawn noble recruited", played.State.Nobles)
	}
}

func TestDignityOrderConfersBastard(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "AAA")
	giveDignityCard(state, "P1")
	giveDignityCard(state, "P1")
	play := func(id models.OrderID, noble models.NobleCode) models.WinterOrder {
		return models.WinterOrder{ID: id, Type: models.WinterOrderTypeDignity, NobleCode: noble, CardCode: "BAS"}
	}
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {play("O1", "ONE"), play("O2", "ONE"), play("O3", "TWO")},
	})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 2 || reasons[0] != "noble_already_bastard" || reasons[1] != "noble_not_owned" {
		t.Errorf("rejections = %v, want noble_already_bastard then noble_not_owned", reasons)
	}
	for _, noble := range resolution.State.Nobles {
		if want := noble.ID == "N1"; noble.IsBastard() != want {
			t.Errorf("noble %s bastard = %v, want %v", noble.ID, noble.IsBastard(), want)
		}
	}
	deck := resolution.State.NobleDeck
	if len(deck.Hands["P1"]) != 1 || len(deck.Played) != 1 {
		t.Errorf("hand = %v, played = %v, want one card consumed", deck.Hands["P1"], deck.Played)
	}
	if events := eventsOfType(resolution.Events, EventTypeDignity); len(events) != 1 || events[0].Dignity != models.DignityBastard || events[0].NobleCode != "ONE" {
		t.Errorf("dignity events = %+v, want one bastard on ONE", events)
	}
	if state.Nobles[0].IsBastard() {
		t.Error("ResolveWinter mutated its input noble")
	}
	scores := ComputeScores(resolution.State)
	if scores["P1"].Titles != 1 {
		t.Errorf("P1 titles = %d, want the bastard dignity to count as one title", scores["P1"].Titles)
	}
}

func TestDignityOrderRejectsCardNotInHand(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	giveDignityCard(state, "P2")
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {
			{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "ONE", CardCode: "BAS"},
			{ID: "O2", Type: models.WinterOrderTypeDignity, NobleCode: "ONE", CardCode: "ELE"},
		},
	})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 2 || reasons[0] != "card_not_in_hand" || reasons[1] != "card_not_in_hand" {
		t.Errorf("rejections = %v, want two card_not_in_hand", reasons)
	}
	if resolution.State.Nobles[0].IsBastard() {
		t.Error("a rejected dignity order conferred the dignity")
	}
}

func TestNobleLimitBastardBonus(t *testing.T) {
	limitFor := func(mutate func(*models.GameState), limit, limitMax int) int {
		state := deckOrdersState(t)
		for _, code := range []string{"ONE", "TWO", "THR", "FOU"} {
			addNoble(state, models.NobleID("N"+code), code, "P1", "AAA")
		}
		mutate(state)
		balance := testBalance()
		balance.NobleLimit, balance.NobleLimitMax = limit, limit+limitMax
		return newResolutionContext(state, balance).nobleLimit("P1")
	}
	bastard := func(index int) func(*models.GameState) {
		return func(state *models.GameState) { state.Nobles[index].Dignities = []models.Dignity{models.DignityBastard} }
	}
	if got := limitFor(func(*models.GameState) {}, 4, 2); got != 4 {
		t.Errorf("limit without bastard = %d, want 4", got)
	}
	if got := limitFor(bastard(0), 4, 2); got != 5 {
		t.Errorf("limit with a bastard = %d, want 5", got)
	}
	if got := limitFor(func(state *models.GameState) { bastard(0)(state); bastard(1)(state) }, 4, 2); got != 6 {
		t.Errorf("limit with two bastards = %d, want +1 per bastard", got)
	}
	// Whatever the status or marital state of the carrier.
	if got := limitFor(func(state *models.GameState) {
		bastard(2)(state)
		state.Nobles[2].Status = models.NobleStatusDungeon
	}, 4, 2); got != 5 {
		t.Errorf("limit with a dungeon bastard = %d, want 5", got)
	}
	if got := limitFor(func(state *models.GameState) {
		bastard(0)(state)
		state.Nobles[0].Status = models.NobleStatusHostage
	}, 4, 2); got != 5 {
		t.Errorf("limit with a hostage bastard = %d, want 5", got)
	}
	if got := limitFor(func(state *models.GameState) { bastard(0)(state); bastard(1)(state); bastard(2)(state) }, 4, 2); got != 6 {
		t.Errorf("limit with three bastards = %d, want it clamped to noble_limit_max", got)
	}
	if got := limitFor(bastard(0), 6, 0); got != 6 {
		t.Errorf("limit at the maximum = %d, want it clamped to noble_limit_max", got)
	}
	if got := limitFor(func(state *models.GameState) {
		bastard(0)(state)
		state.Nobles = append(state.Nobles, models.Noble{ID: "N9", Code: "NIN", Sex: models.SexMale, OwnerID: "P2", LocationID: "AAA", Status: models.NobleStatusFree})
	}, 4, 2); got != 5 {
		t.Errorf("limit of P1 = %d, want another player's nobles ignored", got)
	}
}

func TestRecruitAboveRaisedLimitWithBastard(t *testing.T) {
	state := deckOrdersState(t)
	for _, code := range []string{"ONE", "TWO", "THR", "FOU"} {
		addNoble(state, models.NobleID("N"+code), code, "P1", "AAA")
	}
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	recruit := models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, CardCode: "ELE", TerritoryID: "AAA"}
	rejected := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {recruit}})
	if reasons := rejectionReasons(rejected.Events); len(reasons) != 1 || reasons[0] != "noble_limit_reached" {
		t.Fatalf("rejections = %v, want noble_limit_reached at the base limit", reasons)
	}
	state.Nobles[0].Dignities = []models.Dignity{models.DignityBastard}
	accepted := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {recruit}})
	if len(accepted.State.Nobles) != 5 || len(rejectionReasons(accepted.Events)) != 0 {
		t.Fatalf("nobles = %d, rejections = %v, want a fifth noble with a bastard in the family", len(accepted.State.Nobles), rejectionReasons(accepted.Events))
	}
}

func TestBastardKeepsTitleOnRecruitAndOnCardPlay(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	state.Nobles[0].Dignities = []models.Dignity{models.DignityBastard}
	holdAsFiefMember(state, "AAA", "P1")
	holder := models.NobleID("N1")
	state.Fiefs[0].HolderNobleID = &holder
	giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	// Recruiting a non-bastard noble leaves the bastard's title alone.
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeRecruitNoble, CardCode: "ELE", TerritoryID: "AAA"}},
	})
	if fief := resolution.State.Fiefs[0]; fief.HolderNobleID == nil || *fief.HolderNobleID != "N1" {
		t.Fatalf("holder = %v, want the bastard N1 to keep its title", fief.HolderNobleID)
	}
	if events := eventsOfType(resolution.Events, EventTypeFiefAssigned); len(events) != 0 {
		t.Errorf("fief events = %+v, want none", events)
	}
	// Making the holder a bastard leaves its title alone too.
	state = deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	holdAsFiefMember(state, "AAA", "P1")
	holder = "N2"
	state.Fiefs[0].HolderNobleID = &holder
	giveDignityCard(state, "P1")
	resolution = resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "TWO", CardCode: "BAS"}},
	})
	if fief := resolution.State.Fiefs[0]; fief.HolderNobleID == nil || *fief.HolderNobleID != "N2" {
		t.Fatalf("holder = %v, want the new bastard N2 to keep its title", fief.HolderNobleID)
	}
	if events := eventsOfType(resolution.Events, EventTypeFiefAssigned); len(events) != 0 {
		t.Errorf("fief events = %+v, want none", events)
	}
}

func TestAssignFiefRejectsBastardWhileNonBastardLives(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P1", "AAA")
	state.Nobles[1].Dignities = []models.Dignity{models.DignityBastard}
	holdAsFiefMember(state, "AAA", "P1")
	capital := state.Fiefs[0].CapitalTerritoryID
	assign := func(code models.NobleCode) models.WinterOrder {
		return models.WinterOrder{ID: "O1", Type: models.WinterOrderTypeAssignFief, NobleCode: code, TerritoryID: capital}
	}
	rejected := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{"P1": {assign("TWO")}})
	if reasons := rejectionReasons(rejected.Events); len(reasons) != 1 || reasons[0] != "succession_rank_blocked" {
		t.Fatalf("rejections = %v, want succession_rank_blocked for the bastard", reasons)
	}
}

func TestParseNobleDeckOrders(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	parsed, parseErrors := orders.ParseWinterOrders("t n\nr n ele aaa\nd n one bas", state)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseWinterOrders errors = %#v", parseErrors)
	}
	want := []models.WinterOrder{
		{ID: "O1", Type: models.WinterOrderTypeDrawNoble},
		{ID: "O2", Type: models.WinterOrderTypeRecruitNoble, CardCode: "ELE", TerritoryID: "AAA"},
		{ID: "O3", Type: models.WinterOrderTypeDignity, NobleCode: "ONE", CardCode: "BAS"},
	}
	if !reflect.DeepEqual(parsed, want) {
		t.Errorf("parsed = %#v, want %#v", parsed, want)
	}
	for _, line := range []string{"T N 3", "R N AAA", "R N ELE AAA ZZZ", "R N ELE ZZZ", "R N el AAA", "D N ONE", "D N XXX BAS", "D N ONE ba", "D N ONE BAS AAA", "T X"} {
		if _, parseErrors := orders.ParseWinterOrders(line, state); len(parseErrors) == 0 {
			t.Errorf("ParseWinterOrders(%q) accepted a malformed order", line)
		}
	}
}

func cardByCode(deck *models.NobleDeck, code string) (models.NobleCard, bool) {
	for _, card := range deck.Cards {
		if card.Code == code {
			return card, true
		}
	}
	return models.NobleCard{}, false
}

func TestDrawNobleReshufflesDiscardWhenPileIsEmpty(t *testing.T) {
	build := func() *models.GameState {
		state := deckOrdersState(t)
		ensureNobleDeck(state)
		for _, code := range []string{"AAB", "AAC", "AAD", "AAE", "AAF"} {
			id := pileNobleCard(state, code, code, models.SexMale)
			state.NobleDeck.DrawPile = state.NobleDeck.DrawPile[:len(state.NobleDeck.DrawPile)-1]
			state.NobleDeck.Discard = append(state.NobleDeck.Discard, id)
		}
		state.Seed = "reshuffle-seed"
		return state
	}
	draw := map[models.PlayerID][]models.WinterOrder{"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}}}
	first := resolveNobleDeckWinter(t, build(), draw)
	second := resolveNobleDeckWinter(t, build(), draw)
	deck := first.State.NobleDeck
	if len(deck.Hands["P1"]) != 1 || len(deck.DrawPile) != 4 || len(deck.Discard) != 0 || deck.Reshuffles != 1 {
		t.Fatalf("deck = %+v, want one card drawn from the reshuffled discard", deck)
	}
	if !reflect.DeepEqual(deck, second.State.NobleDeck) {
		t.Error("the same state reshuffled differently")
	}
	if reasons := rejectionReasons(first.Events); len(reasons) != 0 {
		t.Errorf("rejections = %v, want none", reasons)
	}
	// The counter changes the next shuffle.
	other := build()
	other.NobleDeck.Reshuffles = 7
	third := resolveNobleDeckWinter(t, other, draw)
	if third.State.NobleDeck.Reshuffles != 8 {
		t.Errorf("reshuffles = %d, want the counter to advance to 8", third.State.NobleDeck.Reshuffles)
	}
}

func TestDrawNobleRejectsWhenBothPilesAreEmpty(t *testing.T) {
	state := deckOrdersState(t)
	ensureNobleDeck(state)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}},
	})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "noble_deck_empty" {
		t.Errorf("rejections = %v, want noble_deck_empty", reasons)
	}
}

func plagueNobleState(t *testing.T) *models.GameState {
	t.Helper()
	state := effectTestState()
	state.Players = []models.Player{{ID: "P1"}}
	setCurrentCalamity(state, models.CardKindPlague, "AAA")
	ensureNobleDeck(state)
	state.NobleDeck.NamePool = []models.NobleName{
		{Code: "MAH", Name: "Mahaut", Sex: models.SexFemale},
		{Code: "GUI", Name: "Guy", Sex: models.SexMale},
		{Code: "ISA", Name: "Isabeau", Sex: models.SexFemale},
	}
	return state
}

func killNobles(state *models.GameState) *resolutionContext {
	balance := testBalance()
	balance.SpecialOrders.Effects.PlagueNobleMortalityPercentage = 100
	ctx := newResolutionContext(state, balance)
	resolveSeasonEffects(ctx)
	return ctx
}

func TestDeadNobleFromCardLeavesReplacementCardOfSameSexInDiscard(t *testing.T) {
	state := plagueNobleState(t)
	addNoble(state, "N1", "ELE", "P1", "AAA")
	cardID := giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	state.NobleDeck.Hands["P1"] = nil
	state.NobleDeck.Played = []models.NobleCardPlay{{Card: cardID, Noble: "N1"}}
	state.Nobles[0].Sex = models.SexFemale
	validateTestState(t, state)

	ctx := killNobles(state)
	deck := ctx.state.NobleDeck
	if len(ctx.state.Nobles) != 0 || len(deck.Played) != 0 {
		t.Fatalf("nobles = %d, played = %v, want the noble dead and untracked", len(ctx.state.Nobles), deck.Played)
	}
	if len(deck.Discard) != 1 {
		t.Fatalf("discard = %v, want one replacement card", deck.Discard)
	}
	replacement, _ := deck.Card(deck.Discard[0])
	if replacement.Kind != models.NobleCardKindNoble || replacement.Sex != models.SexFemale || replacement.Code != "MAH" {
		t.Errorf("replacement = %+v, want a new female noble card from the pool", replacement)
	}
	if _, found := cardByCode(deck, "ELE"); found {
		t.Error("the dead noble's code is still on a card, want it only reserved by the lineage")
	}
	if len(deck.NamePool) != 2 {
		t.Errorf("pool = %v, want the used name taken out", deck.NamePool)
	}
	validateTestState(t, ctx.state)
}

func TestDeadNobleWithoutCardLeavesNothing(t *testing.T) {
	state := plagueNobleState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	ctx := killNobles(state)
	if deck := ctx.state.NobleDeck; len(deck.Discard) != 0 || len(deck.NamePool) != 3 {
		t.Errorf("deck = %+v, want a starting noble to leave no card", deck)
	}
}

func TestDeadNobleWithoutFreeNameLeavesNoReplacement(t *testing.T) {
	state := plagueNobleState(t)
	state.NobleDeck.NamePool = []models.NobleName{{Code: "GUI", Name: "Guy", Sex: models.SexMale}}
	addNoble(state, "N1", "ELE", "P1", "AAA")
	cardID := giveNobleCard(state, "P1", "ELE", "Eleonore", models.SexFemale)
	state.NobleDeck.Hands["P1"] = nil
	state.NobleDeck.Played = []models.NobleCardPlay{{Card: cardID, Noble: "N1"}}
	ctx := killNobles(state)
	if deck := ctx.state.NobleDeck; len(deck.Discard) != 0 || len(deck.NamePool) != 1 {
		t.Errorf("deck = %+v, want no replacement when no female name is free", deck)
	}
}

func TestDeadBastardReturnsDignityCardToDiscard(t *testing.T) {
	state := plagueNobleState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	state.Nobles[0].Dignities = []models.Dignity{models.DignityBastard}
	cardID := giveDignityCard(state, "P1")
	state.NobleDeck.Hands["P1"] = nil
	state.NobleDeck.Played = []models.NobleCardPlay{{Card: cardID, Noble: "N1"}}
	validateTestState(t, state)

	ctx := killNobles(state)
	deck := ctx.state.NobleDeck
	if len(deck.Discard) != 1 || deck.Discard[0] != cardID || len(deck.Played) != 0 {
		t.Errorf("discard = %v, played = %v, want the dignity card back in the discard", deck.Discard, deck.Played)
	}
	validateTestState(t, ctx.state)
}

func TestRemoveDignityReturnsCardToDiscard(t *testing.T) {
	state := deckOrdersState(t)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	giveDignityCard(state, "P1")
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDignity, NobleCode: "ONE", CardCode: "BAS"}},
	})
	next := resolution.State
	if len(next.NobleDeck.Played) != 1 || !next.Nobles[0].IsBastard() {
		t.Fatalf("setup: played = %v, nobles = %+v", next.NobleDeck.Played, next.Nobles)
	}
	ctx := newResolutionContext(next, testBalance())
	ctx.removeDignity(&ctx.state.Nobles[0], models.DignityBastard)
	deck := ctx.state.NobleDeck
	if ctx.state.Nobles[0].IsBastard() || len(deck.Played) != 0 || len(deck.Discard) != 1 {
		t.Errorf("noble = %+v, deck = %+v, want the dignity gone and its card discarded", ctx.state.Nobles[0], deck)
	}
	validateTestState(t, ctx.state)
}

func TestDrawNobleOrderRejectedWhenSharedHandIsFull(t *testing.T) {
	state := deckOrdersState(t)
	pileNobleCard(state, "ELE", "Eleonore", models.SexFemale)
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{
			{ID: "C1", Kind: models.CardKindFairWeather}, {ID: "C2", Kind: models.CardKindFairWeather},
		},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1", "C2"}},
	}
	held := []models.NobleCardID{pileNobleCard(state, "GUI", "Guy", models.SexMale), pileNobleCard(state, "MAH", "Mahaut", models.SexFemale)}
	state.NobleDeck.DrawPile = slicesWithout(state.NobleDeck.DrawPile, held)
	state.NobleDeck.Hands["P1"] = held
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}},
	})
	if reasons := rejectionReasons(resolution.Events); len(reasons) != 1 || reasons[0] != "hand_limit_reached" {
		t.Errorf("rejections = %v, want one hand_limit_reached", reasons)
	}
	if len(resolution.State.NobleDeck.DrawPile) != 1 || len(resolution.State.NobleDeck.Hands["P1"]) != 2 {
		t.Errorf("noble deck = %+v, want the pile untouched", resolution.State.NobleDeck)
	}
}

func slicesWithout(pile, removed []models.NobleCardID) []models.NobleCardID {
	kept := []models.NobleCardID{}
	for _, id := range pile {
		if !slices.Contains(removed, id) {
			kept = append(kept, id)
		}
	}
	return kept
}

// refillState gives P1 a special deck full of bonus cards and the given
// noble-hand cards.
func refillState(t *testing.T, nobleHand int) *models.GameState {
	t.Helper()
	state := deckOrdersState(t)
	pileNobleCard(state, "ELE", "Eleonore", models.SexFemale)
	for index := 0; index < nobleHand; index++ {
		id := pileNobleCard(state, []string{"HAA", "HAB", "HAC"}[index], "Held", models.SexMale)
		state.NobleDeck.DrawPile = slicesWithout(state.NobleDeck.DrawPile, []models.NobleCardID{id})
		state.NobleDeck.Hands["P1"] = append(state.NobleDeck.Hands["P1"], id)
	}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {}},
	}
	for index := 1; index <= 6; index++ {
		id := models.SpecialCardID("C" + string(rune('0'+index)))
		state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: id, Kind: models.CardKindFairWeather})
		state.SpecialDeck.DrawPile = append(state.SpecialDeck.DrawPile, id)
	}
	return state
}

func TestSpecialRefillRespectsNobleHandCards(t *testing.T) {
	state := refillState(t, 3)
	resolution := resolveNobleDeckWinter(t, state, nil)
	if got := len(resolution.State.SpecialDeck.Hands["P1"]); got != 1 {
		t.Errorf("special hand = %d cards, want 1 free slot filled (3 noble cards held, limit 4)", got)
	}
}

func TestSpecialRefillWithoutNobleDrawFillsTwo(t *testing.T) {
	state := refillState(t, 0)
	resolution := resolveNobleDeckWinter(t, state, nil)
	if got := len(resolution.State.SpecialDeck.Hands["P1"]); got != 2 {
		t.Errorf("special hand = %d cards, want 2", got)
	}
}

func TestNobleDrawAndSpecialRefillShareTwoDrawsPerWinter(t *testing.T) {
	state := refillState(t, 0)
	resolution := resolveNobleDeckWinter(t, state, map[models.PlayerID][]models.WinterOrder{
		"P1": {{ID: "O1", Type: models.WinterOrderTypeDrawNoble}},
	})
	if got := len(resolution.State.NobleDeck.Hands["P1"]); got != 1 {
		t.Errorf("noble hand = %d cards, want 1", got)
	}
	if got := len(resolution.State.SpecialDeck.Hands["P1"]); got != 1 {
		t.Errorf("special hand = %d cards, want 1 (the noble draw uses one of the two draws)", got)
	}
}
