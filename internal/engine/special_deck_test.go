package engine

import (
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func specialDeckTestBalance() assetgen.Balance {
	balance := testBalance()
	balance.SpecialOrders = assetgen.SpecialOrdersBalance{
		DeckSize:           30,
		CalamityPercentage: 30,
		CalamityWeights: map[models.CardKind]int{
			models.CardKindPlague:     1,
			models.CardKindBadWeather: 6,
			models.CardKindFamine:     6,
		},
		BonusWeights: map[models.CardKind]int{
			models.CardKindFairWeather:     3,
			models.CardKindAbundantHarvest: 3,
			models.CardKindRevolt:          1,
		},
	}
	return balance
}

func TestBuildSpecialDeckIsDeterministicAndWeighted(t *testing.T) {
	balance := specialDeckTestBalance()
	first, err := buildSpecialDeck("deck-seed", balance)
	if err != nil {
		t.Fatalf("buildSpecialDeck = %v", err)
	}
	second, err := buildSpecialDeck("deck-seed", balance)
	if err != nil {
		t.Fatalf("second buildSpecialDeck = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same seed produced different decks")
	}
	if len(first.Cards) != balance.SpecialOrders.DeckSize || len(first.DrawPile) != len(first.Cards) {
		t.Fatalf("deck sizes = %d/%d, want %d", len(first.Cards), len(first.DrawPile), balance.SpecialOrders.DeckSize)
	}
	counts := map[models.CardKind]int{}
	for _, card := range first.Cards {
		counts[card.Kind]++
	}
	if counts[models.CardKindPlague] != 1 || counts[models.CardKindBadWeather] != 4 || counts[models.CardKindFamine] != 4 {
		t.Fatalf("calamity counts = %#v", counts)
	}
	if counts[models.CardKindFairWeather] != 9 || counts[models.CardKindAbundantHarvest] != 9 || counts[models.CardKindRevolt] != 3 {
		t.Fatalf("bonus counts = %#v, want fair 9, harvest 9, revolt 3", counts)
	}
	for index, card := range first.Cards {
		wantID := models.SpecialCardID("C" + formatCardNumber(index+1))
		if card.ID != wantID {
			t.Errorf("card[%d].ID = %q, want %q", index, card.ID, wantID)
		}
	}
}

func TestCreateGameInitializesSpecialDeck(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance := specialDeckTestBalance()
	game, err := CreateGame("deck-game", []PlayerInit{{}, {}}, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame = %v", err)
	}
	if game.SpecialDeck == nil || len(game.SpecialDeck.Cards) != balance.SpecialOrders.DeckSize {
		t.Fatalf("special deck = %#v", game.SpecialDeck)
	}
	if len(game.SpecialDeck.Hands["P1"]) != 0 || len(game.SpecialDeck.Hands["P2"]) != 0 {
		t.Fatalf("initial hands = %#v, want empty", game.SpecialDeck.Hands)
	}
}

func TestResolveActionAppliesDeckOrderWithoutNoble(t *testing.T) {
	state := models.NewGameState()
	state.ID = "deck-action"
	state.Seed = "deck-action"
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Territories = []models.Territory{{ID: "ROS", Name: "ROS", Terrain: models.TerrainPlain}}
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"ROS": {Infrastructures: []models.InfraID{}}}
	state.SpecialDeck = &models.SpecialDeck{
		Cards:    []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}},
		DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {"C1"}},
	}
	resolution, err := ResolveWithDeckOrders(state, testBalance(), map[models.PlayerID][]models.DeckOrder{
		"P1": {{ID: "O1", Type: models.DeckOrderTypePlay, Kind: models.CardKindFairWeather, RegionSeed: "ROS"}},
	})
	if err != nil {
		t.Fatalf("ResolveWithDeckOrders = %v", err)
	}
	if len(resolution.State.SpecialDeck.Hands["P1"]) != 0 || len(resolution.State.SpecialDeck.Discard) != 1 {
		t.Fatalf("deck after action = %#v, want consumed card", resolution.State.SpecialDeck)
	}
}
