package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func winterDeckBalance() assetgen.Balance {
	balance := testBalance()
	balance.SpecialOrders.HandLimit = 4
	balance.SpecialOrders.DrawOrdersLimit = 2
	balance.SpecialOrders.CalamitySlots = map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonWinter: 1}
	return balance
}

func winterDeckState() *models.GameState {
	state := models.NewGameState()
	state.Turn = 4
	state.Season = models.SeasonWinter
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Regions = []models.Region{{ID: "ROS", Seed: "ROS", Territories: []models.TerritoryID{"ROS"}}}
	state.Territories = []models.Territory{
		{ID: "ROS", Name: "ROS", Terrain: models.TerrainPlain},
		{ID: "BOI", Name: "BOI", Terrain: models.TerrainPlain},
	}
	p1, p2 := models.PlayerID("P1"), models.PlayerID("P2")
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{
		"ROS": {OwnerID: &p1, Infrastructures: []models.InfraID{}},
		"BOI": {OwnerID: &p2, Infrastructures: []models.InfraID{}},
	}
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{
			{ID: "C1", Kind: models.CardKindPlague},
			{ID: "C2", Kind: models.CardKindFairWeather},
		},
		DrawPile: []models.SpecialCardID{"C1", "C2"},
		Discard:  []models.SpecialCardID{},
		Hands:    map[models.PlayerID][]models.SpecialCardID{"P1": {}},
	}
	return state
}

func TestResolveWinterDrawsThroughCalamityIntoBonus(t *testing.T) {
	state := winterDeckState()
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypeDraw}},
	})
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("hand = %#v, want [C2]", got)
	}
	if len(resolution.State.SpecialDeck.Discard) != 0 {
		t.Fatalf("discard = %#v, want empty programmed calamity", resolution.State.SpecialDeck.Discard)
	}
	augury := resolution.State.Auguries[2]
	if len(augury.Calamities) != 1 || augury.Calamities[0].CardID != "C1" || augury.Calamities[0].RegionSeed != "ROS" {
		t.Fatalf("augury = %#v, want programmed C1 at ROS", augury)
	}
}

func TestResolveTurnValidationDoesNotApplyWinterDeckOrders(t *testing.T) {
	state := winterDeckState()
	before := cloneGameState(state)
	report, err := ResolveTurn(state, winterDeckBalance(), OrdersInput{
		Special: []DeckSubmission{{Player: "P1", Text: "T C"}},
	})
	if err != nil {
		t.Fatalf("ResolveTurn = %v", err)
	}
	if got := report.State.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("resolved hand = %#v, want [C2]", got)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("winter deck validation mutated the submitted game state")
	}
}

func TestResolveWinterDeckOrdersRejectsThirdDrawAtomically(t *testing.T) {
	state := winterDeckState()
	before := cloneGameState(state)
	_, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {
			{ID: "O1", Type: models.DeckOrderTypeDraw},
			{ID: "O2", Type: models.DeckOrderTypeDraw},
			{ID: "O3", Type: models.DeckOrderTypeDraw},
		},
	})
	if err == nil {
		t.Fatal("ResolveWinterWithDeckOrders = nil, want draw limit error")
	}
	if got := state.SpecialDeck.Hands["P1"]; len(got) != 0 || len(state.SpecialDeck.DrawPile) != len(before.SpecialDeck.DrawPile) {
		t.Fatalf("input state changed after rejected submission: %#v", state.SpecialDeck)
	}
}

func TestWinterRumorsRequireMultiplePlayersAndAreDeterministic(t *testing.T) {
	state := winterDeckState()
	first := newResolutionContext(state, winterDeckBalance())
	first.deckDraws["P1"] = []models.CardKind{models.CardKindFairWeather}
	emitWinterRumors(first)
	if len(first.events) != 0 {
		t.Fatalf("single-player rumors = %#v, want none", first.events)
	}
	first.deckDraws["P2"] = []models.CardKind{models.CardKindAbundantHarvest}
	emitWinterRumors(first)
	second := newResolutionContext(state, winterDeckBalance())
	second.deckDraws["P1"] = []models.CardKind{models.CardKindFairWeather}
	second.deckDraws["P2"] = []models.CardKind{models.CardKindAbundantHarvest}
	emitWinterRumors(second)
	if len(first.events) != len(second.events) {
		t.Fatalf("rumor counts = %d/%d, want deterministic", len(first.events), len(second.events))
	}
	for index := range first.events {
		if first.events[index].Type != EventTypeRumor || first.events[index].CardKind != second.events[index].CardKind || first.events[index].RumorKey != second.events[index].RumorKey || first.events[index].OwnerID != "" {
			t.Fatalf("rumor events = %#v/%#v, want public deterministic events", first.events, second.events)
		}
	}
}

func TestBuildWinterReportRumors(t *testing.T) {
	before := winterDeckState()
	after := cloneGameState(before)
	report := BuildTurnReport(before, after, []Event{{Type: EventTypeRumor, Phase: winterPhase, CardKind: models.CardKindFairWeather, RumorKey: "rumor.fair_weather"}}, nil)
	if report.Winter == nil || len(report.Winter.Rumors) != 1 || report.Winter.Rumors[0].Key != "rumor.fair_weather" {
		t.Fatalf("winter rumors = %#v, want one public rumor", report.Winter)
	}
}

func TestResolveWinterDiscardsCardInHandOrder(t *testing.T) {
	state := winterDeckState()
	state.SpecialDeck.Cards = []models.SpecialCard{
		{ID: "C1", Kind: models.CardKindFairWeather},
		{ID: "C2", Kind: models.CardKindFairWeather},
	}
	state.SpecialDeck.DrawPile = []models.SpecialCardID{}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1", "C2"}
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypeDiscard, Kind: models.CardKindFairWeather}},
	})
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	if got := resolution.State.SpecialDeck.Hands["P1"]; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("hand = %#v, want [C2]", got)
	}
	if got := resolution.State.SpecialDeck.Discard; len(got) != 1 || got[0] != "C1" {
		t.Fatalf("discard = %#v, want [C1]", got)
	}
}

func TestResolveWinterFullHandMakesDrawNoOp(t *testing.T) {
	state := winterDeckState()
	state.SpecialDeck.Cards = []models.SpecialCard{
		{ID: "C1", Kind: models.CardKindFairWeather},
		{ID: "C2", Kind: models.CardKindFairWeather},
	}
	state.SpecialDeck.DrawPile = []models.SpecialCardID{"C2"}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{"C1"}
	balance := winterDeckBalance()
	balance.SpecialOrders.HandLimit = 1
	resolution, err := ResolveWinterWithDeckOrders(state, balance, nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypeDraw}},
	})
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	if got := resolution.State.SpecialDeck.DrawPile; len(got) != 1 || got[0] != "C2" {
		t.Fatalf("draw pile = %#v, want [C2]", got)
	}
}

func TestResolveWinterRemixesDiscardBeforeDrawing(t *testing.T) {
	state := winterDeckState()
	state.SpecialDeck.DrawPile = []models.SpecialCardID{}
	state.SpecialDeck.Discard = []models.SpecialCardID{"C1", "C2"}
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypeDraw}},
	})
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	if len(resolution.State.SpecialDeck.Discard) != 0 || len(resolution.State.SpecialDeck.Hands["P1"]) != 1 {
		t.Fatalf("deck after remix = %#v, want one card in hand and empty discard", resolution.State.SpecialDeck)
	}
}

func TestResolveWinterDiscardsCalamityWhenSlotsAreFull(t *testing.T) {
	state := winterDeckState()
	state.SpecialDeck.Cards = []models.SpecialCard{{ID: "C1", Kind: models.CardKindPlague}}
	state.SpecialDeck.DrawPile = []models.SpecialCardID{"C1"}
	state.SpecialDeck.Hands["P1"] = []models.SpecialCardID{}
	state.Auguries[2] = models.YearAugury{Year: 2, Capacities: map[models.Season]int{models.SeasonSpring: 0, models.SeasonSummer: 0, models.SeasonWinter: 0}, Calamities: []models.Calamity{}}
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), nil, map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypeDraw}},
	})
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	if got := resolution.State.SpecialDeck.Discard; len(got) != 1 || got[0] != "C1" {
		t.Fatalf("discard = %#v, want [C1]", got)
	}
}
