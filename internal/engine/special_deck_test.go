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
		BonusCardsBase:         6,
		BonusCardsPerPlayer:    6,
		CalamityCardsBase:      2,
		CalamityCardsPerPlayer: 1,
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
	first, err := buildSpecialDeck("deck-seed", 2, balance)
	if err != nil {
		t.Fatalf("buildSpecialDeck = %v", err)
	}
	second, err := buildSpecialDeck("deck-seed", 2, balance)
	if err != nil {
		t.Fatalf("second buildSpecialDeck = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same seed produced different decks")
	}
	if want := balance.SpecialOrders.DeckSize(2); len(first.Cards) != want || len(first.DrawPile) != len(first.Cards) {
		t.Fatalf("deck sizes = %d/%d, want %d", len(first.Cards), len(first.DrawPile), want)
	}
	counts := map[models.CardKind]int{}
	for _, card := range first.Cards {
		counts[card.Kind]++
	}
	// 2 + 1 x 2 calamities, weighted 1/6/6; 6 + 6 x 2 bonus cards, weighted 3/3/1.
	if counts[models.CardKindPlague] != 0 || counts[models.CardKindBadWeather] != 2 || counts[models.CardKindFamine] != 2 {
		t.Fatalf("calamity counts = %#v, want 4 calamities: no plague, 2 bad weather, 2 famine", counts)
	}
	if counts[models.CardKindFairWeather] != 8 || counts[models.CardKindAbundantHarvest] != 8 || counts[models.CardKindRevolt] != 2 {
		t.Fatalf("bonus counts = %#v, want fair 8, harvest 8, revolt 2", counts)
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
	if game.SpecialDeck == nil || len(game.SpecialDeck.Cards) != balance.SpecialOrders.DeckSize(2) {
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
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"ROS": {}}
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
	played := 0
	discarded := 0
	for _, event := range resolution.Events {
		if event.Type == EventTypeDeckOrderPlayed {
			played++
		}
		if event.Type == EventTypeDeckDiscard {
			discarded++
		}
	}
	if played != 1 || discarded != 0 {
		t.Fatalf("card events = played %d/discarded %d, want one played event only", played, discarded)
	}
}

// The deck is built from the same kind lists as the rules rendering, so every
// weighted kind enters it with the announced count..
func TestBuildSpecialDeckContainsEveryWeightedKind(t *testing.T) {
	balance := specialDeckTestBalance()
	balance.SpecialOrders.CalamityWeights = map[models.CardKind]int{models.CardKindPlague: 1, models.CardKindBadWeather: 6, models.CardKindFamine: 6}
	balance.SpecialOrders.BonusWeights = map[models.CardKind]int{
		models.CardKindFairWeather: 3, models.CardKindAbundantHarvest: 3, models.CardKindRevolt: 1,
		models.CardKindSeigneurialTax: 2, models.CardKindTunnel: 1,
	}
	deck, err := buildSpecialDeck("deck-seed", 2, balance)
	if err != nil {
		t.Fatalf("buildSpecialDeck = %v", err)
	}
	counts := map[models.CardKind]int{}
	for _, card := range deck.Cards {
		counts[card.Kind]++
	}
	config := balance.SpecialOrders
	want := assetgen.WeightedCardCounts(config.CalamityCardCount(2), config.CalamityWeights, assetgen.CalamityCardKinds())
	for kind, count := range assetgen.WeightedCardCounts(config.BonusCardCount(2), config.BonusWeights, assetgen.BonusCardKinds()) {
		want[kind] = count
	}
	for kind, count := range want {
		if counts[kind] != count {
			t.Errorf("deck holds %d %s cards, want %d", counts[kind], kind, count)
		}
	}
	if counts[models.CardKindTunnel] == 0 {
		t.Errorf("no Tunnel card in the deck: %#v", counts)
	}
}

// The deck grows with the number of players: the bonus cards and the
// calamities follow balance.special_orders.
func TestBuildSpecialDeckGrowsWithPlayers(t *testing.T) {
	balance := specialDeckTestBalance()
	for _, players := range []int{2, 4, 8} {
		deck, err := buildSpecialDeck("deck-seed", players, balance)
		if err != nil {
			t.Fatalf("buildSpecialDeck(%d players) = %v", players, err)
		}
		bonus, calamities := 0, 0
		for _, card := range deck.Cards {
			if card.Kind.IsCalamity() {
				calamities++
			} else {
				bonus++
			}
		}
		if bonus != 6+6*players || calamities != 2+players {
			t.Errorf("%d players: %d bonus and %d calamities, want %d and %d", players, bonus, calamities, 6+6*players, 2+players)
		}
		if len(deck.DrawPile) != len(deck.Cards) {
			t.Errorf("%d players: draw pile = %d cards, want %d", players, len(deck.DrawPile), len(deck.Cards))
		}
	}
}
