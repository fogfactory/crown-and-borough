package engine

import (
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// assassinationTestState is P1's free noble ONE (the commissioner) on AAA, the
// P2 noble TWO held hostage by a P1 army on BBB, and one Assassination card in
// P1's hand.
func assassinationTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := testState(t,
		[]models.Territory{
			territory("AAA", "AAA", "BBB"), territory("BBB", "BBB", "AAA", "CCC"),
			territory("CCC", "CCC", "BBB", "DDD"), territory("DDD", "DDD", "CCC"),
		},
		[]models.Army{{ID: "A1", OwnerID: "P1", TerritoryID: "BBB", Size: 1}},
	)
	addInfrastructure(state, models.Infrastructure{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "AAA"})
	setCapital(state, "P1", "I1")
	keepTestArmiesSupplied(state)
	addNoble(state, "N1", "ONE", "P1", "AAA")
	addNoble(state, "N2", "TWO", "P2", "BBB")
	setNobleStatus(state, "N2", models.NobleStatusHostage)
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindAssassination}},
		DrawPile: []models.SpecialCardID{},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	return state
}

func playAssassination(commissioner, target models.NobleID) map[models.PlayerID][]models.DeckOrder {
	return map[models.PlayerID][]models.DeckOrder{"P1": {{
		ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindAssassination,
		CommissionerNobleID: commissioner, TargetNobleID: target,
	}}}
}

func TestAssassinationKillsTheVictimAndRecordsTheCommissioner(t *testing.T) {
	state := assassinationTestState(t)
	// The victim holds a fief and a Claim names it; the fief is vacated.
	state.Fiefs = []models.Fief{{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: "BBB",
		Territories: []models.TerritoryID{"BBB", "CCC", "DDD"}, OwnerID: "P2", HolderNobleID: nobleIDPtr("N2"),
	}}
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N2"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	for _, noble := range resolution.State.Nobles {
		if noble.ID == "N2" {
			t.Fatalf("victim still alive: %#v", noble)
		}
	}
	if len(resolution.State.RemovedNobles) != 1 {
		t.Fatalf("removed nobles = %#v", resolution.State.RemovedNobles)
	}
	removed := resolution.State.RemovedNobles[0]
	if removed.ID != "N2" || removed.Cause != models.DeathCauseAssassination || removed.Commissioner != "N1" || removed.Author != "P1" || removed.OwnerID != "P2" {
		t.Fatalf("removed = %#v, want N2 assassinated, commissioned by N1", removed)
	}
	if len(resolution.State.Fiefs) != 1 || resolution.State.Fiefs[0].HolderNobleID != nil {
		t.Fatalf("fiefs = %#v, want the fief vacated", resolution.State.Fiefs)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 0 {
		t.Fatalf("hand = %#v, want the card consumed", got)
	}
	events := eventsOfType(resolution.Events, EventTypeAssassination)
	if len(events) != 1 || events[0].OwnerID != "P1" || events[0].PreviousOwnerID != "P2" ||
		events[0].NobleCode != "TWO" || events[0].CommissionerCode != "ONE" {
		t.Fatalf("assassination events = %#v", events)
	}
	report := BuildTurnReportWithBalance(state, resolution.State, resolution.Events, nil, 3, testBalance())
	if len(report.SeasonEffects) != 1 || report.SeasonEffects[0].Kind != EventTypeAssassination || report.SeasonEffects[0].Commissioner != "ONE" {
		t.Fatalf("season effects = %#v", report.SeasonEffects)
	}
	rumored := false
	for _, rumor := range report.Rumors {
		rumored = rumored || rumor.Key == "rumor.assassination.killing"
	}
	if !rumored {
		t.Fatalf("rumors = %#v, want the assassination rumor", report.Rumors)
	}
}

func TestAssassinationOfTheKingVacatesTheCrown(t *testing.T) {
	state := assassinationTestState(t)
	king := models.NobleID("N2")
	state.King = &king
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N2"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if resolution.State.King != nil {
		t.Fatalf("king = %v, want the crown vacant", *resolution.State.King)
	}
}

func TestAssassinationDropsTheChainEmittedThisTurn(t *testing.T) {
	state := assassinationTestState(t)
	// The victim emits an order this turn from the P2 army it commands.
	addAnchorArmy(t, state, "A2", "P2", "CCC")
	addNoble(state, "N3", "TRE", "P2", "CCC")
	state.Nobles[len(state.Nobles)-1].LastEmissionTurn = state.Turn
	addChain(t, state, "A2", "N3", models.Order{Type: models.OrderTypeHold, PositionID: "CCC"})
	keepTestArmiesSupplied(state)
	validateTestState(t, state)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N3"))
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if chainOf(resolution.State, "A2") != nil {
		t.Fatalf("chains = %#v, want the victim's chain dropped", resolution.State.Chains)
	}
}

func TestAssassinationIsRejectedWithItsReasonAndKeepsTheCard(t *testing.T) {
	cases := map[string]struct {
		prepare              func(*models.GameState)
		commissioner, target models.NobleID
		reason               string
	}{
		"commissioner held hostage": {
			prepare:      func(state *models.GameState) { setNobleStatus(state, "N1", models.NobleStatusHostage) },
			commissioner: "N1", target: "N2", reason: "assassination_commissioner_invalid",
		},
		"commissioner of another player": {
			commissioner: "N2", target: "N2", reason: "assassination_commissioner_invalid",
		},
		"unknown commissioner": {
			commissioner: "N9", target: "N2", reason: "assassination_commissioner_invalid",
		},
		"own victim": {
			prepare: func(state *models.GameState) {
				addNoble(state, "N3", "TRE", "P1", "AAA")
			},
			commissioner: "N1", target: "N3", reason: "assassination_own_noble",
		},
		"unknown victim": {
			commissioner: "N1", target: "N9", reason: "assassination_unknown_noble",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			state := assassinationTestState(t)
			if test.prepare != nil {
				test.prepare(state)
			}
			validateTestState(t, state)
			_, err := ResolveWithDeckOrders(state, testBalance(), playAssassination(test.commissioner, test.target))
			if err == nil || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("error = %v, want the reason %q", err, test.reason)
			}
			if got := state.SpecialDeck.Hands["P1"]; len(got) != 1 {
				t.Fatalf("hand = %#v, want the card kept", got)
			}
		})
	}
}

// A second assassination of an already dead victim is rejected at resolution
// and keeps its card, while the first one is applied.
func TestSecondAssassinationOfTheSameVictimIsRejected(t *testing.T) {
	state := assassinationTestState(t)
	state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, models.SpecialCard{ID: "C2", Kind: models.CardKindAssassination})
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1", "C2"}
	validateTestState(t, state)
	deckOrders := playAssassination("N1", "N2")
	second := deckOrders["P1"][0]
	second.ID = "O2"
	deckOrders["P1"] = append(deckOrders["P1"], second)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), deckOrders)
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders: %v", err)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 {
		t.Fatalf("hand = %#v, want one card kept", got)
	}
	rejected := eventsOfType(resolution.Events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "assassination_unknown_noble" {
		t.Fatalf("rejected events = %#v", rejected)
	}
}

func TestAssassinationIsParsed(t *testing.T) {
	state := assassinationTestState(t)
	parsed, parseErrors := orders.ParseDeckOrders("P AS ONE TWO\n", state)
	if len(parseErrors) != 0 {
		t.Fatalf("ParseDeckOrders: %v", parseErrors)
	}
	if len(parsed) != 1 || parsed[0].Kind != models.CardKindAssassination ||
		parsed[0].CommissionerNobleID != "N1" || parsed[0].TargetNobleID != "N2" {
		t.Fatalf("orders = %#v", parsed)
	}
	for _, line := range []string{"P AS ONE", "P AS ONE ZZZ", "P AS ZZZ TWO", "P AS ONE TWO ONE"} {
		if _, parseErrors := orders.ParseDeckOrders(line+"\n", state); len(parseErrors) == 0 {
			t.Errorf("%q must be rejected", line)
		}
	}
}

func TestAssassinationIsPlayableInEverySeason(t *testing.T) {
	for _, season := range []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn, models.SeasonWinter} {
		t.Run(string(season), func(t *testing.T) {
			state := assassinationTestState(t)
			state.Turn = map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 2, models.SeasonAutumn: 3, models.SeasonWinter: 4}[season]
			state.Season = season
			var resolution Resolution
			var err error
			if season == models.SeasonWinter {
				resolution, err = ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, playAssassination("N1", "N2"))
			} else {
				resolution, err = ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N2"))
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if len(eventsOfType(resolution.Events, EventTypeAssassination)) != 1 {
				t.Fatalf("events = %#v, want one assassination", resolution.Events)
			}
			if len(resolution.State.RemovedNobles) != 1 || resolution.State.RemovedNobles[0].Commissioner != "N1" {
				t.Fatalf("removed = %#v, want N2 recorded with its commissioner", resolution.State.RemovedNobles)
			}
		})
	}
}

func TestWinterAssassinationIsRejectedAloneAndKeepsTheCard(t *testing.T) {
	state := assassinationTestState(t)
	state.Turn = 4
	state.Season = models.SeasonWinter
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, playAssassination("N2", "N2"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	rejected := eventsOfType(resolution.Events, EventTypeRejected)
	if len(rejected) != 1 || rejected[0].Reason != "assassination_commissioner_invalid" {
		t.Fatalf("rejections = %#v", rejected)
	}
	if len(resolution.State.RemovedNobles) != 0 {
		t.Fatalf("removed = %#v, want nobody dead", resolution.State.RemovedNobles)
	}
}

func TestWinterSheetParsesTheAssassinationCard(t *testing.T) {
	state := assassinationTestState(t)
	winter, deck, parseErrors := orders.ParseWinterOrdersWithDeckOrders("P AS ONE TWO", state)
	if len(parseErrors) != 0 || len(winter) != 0 || len(deck) != 1 || deck[0].Kind != models.CardKindAssassination || deck[0].CommissionerNobleID != "N1" || deck[0].TargetNobleID != "N2" {
		t.Fatalf("winter = %#v, deck = %#v, errors = %#v", winter, deck, parseErrors)
	}
}

func TestAssassinationCardEntryNamesTheVictimAndTheCommissioner(t *testing.T) {
	state := assassinationTestState(t)
	resolution, err := ResolveWithDeckOrders(state, testBalance(), playAssassination("N1", "N2"))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	report := BuildTurnReportWithBalance(state, resolution.State, resolution.Events, nil, 4, testBalance())
	for _, card := range report.Cards {
		if card.EventType == EventTypeDeckOrderPlayed {
			if card.Noble != "TWO" || card.Commissioner != "ONE" || card.NobleName == "" {
				t.Fatalf("card entry = %#v, want victim TWO and commissioner ONE", card)
			}
			return
		}
	}
	t.Fatalf("no card entry in %#v", report.Cards)
}
