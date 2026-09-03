package engine

import (
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
	state.Territories = []models.Territory{{ID: "ROS", Name: "ROS", Terrain: models.TerrainPlain}}
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"ROS": {Infrastructures: []models.InfraID{}}}
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
