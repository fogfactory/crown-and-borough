package engine

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func deckTestPlayers(count int) []models.Player {
	players := make([]models.Player, count)
	for index := range players {
		players[index] = models.Player{ID: models.PlayerID(fmt.Sprintf("P%d", index+1))}
	}
	return players
}

func realPrenoms(t *testing.T) []assetgen.Asset {
	t.Helper()
	return loadGameTestAssets(t).Prenoms
}

// startingCodes returns the codes the first count prenoms would give the
// starting nobles of a game.
func startingCodes(prenoms []assetgen.Asset, count int) map[string]bool {
	used := make(map[string]bool, count)
	for _, prenom := range prenoms[:count] {
		used[prenom.Code] = true
	}
	return used
}

func countKinds(deck *models.NobleDeck) (males, females, dignities int) {
	for _, card := range deck.Cards {
		switch {
		case card.Kind == models.NobleCardKindDignity:
			dignities++
		case card.Sex == models.SexMale:
			males++
		default:
			females++
		}
	}
	return males, females, dignities
}

func TestBuildNobleDeckSizesParityAndBastards(t *testing.T) {
	prenoms := realPrenoms(t)
	for players := 2; players <= 16; players++ {
		t.Run(fmt.Sprintf("%d players", players), func(t *testing.T) {
			used := startingCodes(prenoms, 2*players)
			deck := buildNobleDeck("deck-sizes", deckTestPlayers(players), 6, prenoms, used)
			if deck == nil {
				t.Fatal("deck = nil, want a deck")
			}
			males, females, dignities := countKinds(deck)
			size := len(deck.Cards)
			if size > players*7 {
				t.Errorf("size = %d, want at most %d", size, players*7)
			}
			if players <= 6 && size != players*7 {
				t.Errorf("size = %d, want the unclamped %d", size, players*7)
			}
			if want := max(1, size/max(players-1, 4)); dignities != want || dignities < 1 {
				t.Errorf("dignity cards = %d, want %d (at least one)", dignities, want)
			}
			if diff := males - females; diff < -1 || diff > 1 {
				t.Errorf("males = %d, females = %d, want an even split", males, females)
			}
			if len(deck.DrawPile) != size || len(deck.Played) != 0 || len(deck.Discard) != 0 {
				t.Errorf("draw pile = %d, played = %d, want the whole deck in the pile", len(deck.DrawPile), len(deck.Played))
			}
			codes := map[string]bool{}
			for _, card := range deck.Cards {
				if card.Kind != models.NobleCardKindNoble {
					continue
				}
				if used[card.Code] || card.Code == models.DignityBastardCardCode || codes[card.Code] {
					t.Errorf("noble card %+v reuses a taken code", card)
				}
				codes[card.Code] = true
			}
			state := models.NewGameState()
			state.Players = deckTestPlayers(players)
			state.NobleDeck = deck
			if err := state.Validate(); err != nil {
				t.Errorf("deck is invalid: %v", err)
			}
		})
	}
}

func TestBuildNobleDeckFourPlayers(t *testing.T) {
	prenoms := realPrenoms(t)
	deck := buildNobleDeck("deck-four", deckTestPlayers(4), 6, prenoms, startingCodes(prenoms, 8))
	males, females, dignities := countKinds(deck)
	if len(deck.Cards) != 28 || dignities != 7 || males+females != 21 {
		t.Errorf("deck = %d cards (%d dignities, %d nobles), want 28 (7, 21)", len(deck.Cards), dignities, males+females)
	}
	for _, player := range deckTestPlayers(4) {
		if hand, exists := deck.Hands[player.ID]; !exists || len(hand) != 0 {
			t.Errorf("hand of %s = %v, want an empty hand", player.ID, hand)
		}
	}
}

func TestBuildNobleDeckIsDeterministicAndSeeded(t *testing.T) {
	prenoms := realPrenoms(t)
	build := func(seed string) *models.NobleDeck {
		return buildNobleDeck(seed, deckTestPlayers(4), 6, prenoms, startingCodes(prenoms, 8))
	}
	if !reflect.DeepEqual(build("same"), build("same")) {
		t.Error("the same seed built two different decks")
	}
	if reflect.DeepEqual(build("seed-a").DrawPile, build("seed-b").DrawPile) {
		t.Error("two seeds built the same shuffled draw pile")
	}
	// The seed decides which sex gets the odd card: 2 players make 11 noble
	// cards.
	sawMale, sawFemale := false, false
	for index := 0; index < 20; index++ {
		deck := buildNobleDeck(fmt.Sprintf("odd-%d", index), deckTestPlayers(2), 6, prenoms, startingCodes(prenoms, 4))
		males, females, _ := countKinds(deck)
		switch males - females {
		case 1:
			sawMale = true
		case -1:
			sawFemale = true
		default:
			t.Fatalf("males = %d, females = %d, want an odd split", males, females)
		}
	}
	if !sawMale || !sawFemale {
		t.Errorf("odd card went male=%v female=%v, want both across seeds", sawMale, sawFemale)
	}
}

func TestBuildNobleDeckClampsToAvailableNames(t *testing.T) {
	prenoms := []assetgen.Asset{
		{Code: "AAA", Name: "A", Sex: "male"}, {Code: "BBB", Name: "B", Sex: "male"},
		{Code: "CCC", Name: "C", Sex: "female"}, {Code: "DDD", Name: "D", Sex: "male"},
	}
	deck := buildNobleDeck("tiny", deckTestPlayers(2), 6, prenoms, map[string]bool{"DDD": true})
	males, females, dignities := countKinds(deck)
	if males > 2 || females > 1 || dignities < 1 {
		t.Errorf("deck = %d males, %d females, %d dignities, want it clamped to the three free names", males, females, dignities)
	}
	if deck := buildNobleDeck("none", deckTestPlayers(2), 6, nil, nil); deck != nil {
		t.Errorf("deck = %+v, want nil without any name", deck)
	}
}

func TestCreateGameBuildsNobleDeckWithoutStartingNames(t *testing.T) {
	assets := loadGameTestAssets(t)
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	players := []PlayerInit{{Name: "One"}, {Name: "Two"}, {Name: "Three"}, {Name: "Four"}}
	first, err := CreateGame("noble-deck-game", players, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	second, err := CreateGame("noble-deck-game", players, balance, assets)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if !reflect.DeepEqual(first.NobleDeck, second.NobleDeck) {
		t.Error("the same seed created two different noble decks")
	}
	deck := first.NobleDeck
	if deck == nil || len(deck.Cards) != 28 {
		t.Fatalf("deck = %+v, want 28 cards", deck)
	}
	starting := map[string]bool{}
	for _, noble := range first.Nobles {
		starting[noble.Code] = true
	}
	for _, card := range deck.Cards {
		if card.Kind == models.NobleCardKindNoble && starting[card.Code] {
			t.Errorf("noble card %+v collides with a starting noble", card)
		}
	}
}
