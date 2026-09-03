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
			models.CardKindRevolt:     4,
			models.CardKindFamine:     4,
		},
		BonusWeights: map[models.CardKind]int{
			models.CardKindFairWeather:     1,
			models.CardKindAbundantHarvest: 1,
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
	if counts[models.CardKindPlague] != 1 || counts[models.CardKindBadWeather] != 4 || counts[models.CardKindRevolt] != 2 || counts[models.CardKindFamine] != 2 {
		t.Fatalf("calamity counts = %#v", counts)
	}
	if counts[models.CardKindFairWeather]+counts[models.CardKindAbundantHarvest] != 21 {
		t.Fatalf("bonus count = %d, want 5", counts[models.CardKindFairWeather]+counts[models.CardKindAbundantHarvest])
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
