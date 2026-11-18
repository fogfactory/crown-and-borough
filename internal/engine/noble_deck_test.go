package engine

import (
	"fmt"
	"reflect"
	"slices"
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

// countKinds counts the male noble cards, the female noble cards and the
// bastard cards that are dealt: the reserve is left out.
func countKinds(deck *models.NobleDeck) (males, females, dignities int) {
	for _, card := range deck.Cards {
		switch {
		case slices.Contains(deck.Reserve, card.ID):
		case card.Kind == models.NobleCardKindDignity && card.Dignity == models.DignityBastard:
			dignities++
		case card.Kind == models.NobleCardKindDignity:
		case card.Kind == models.NobleCardKindClaim:
		case card.Sex == models.SexMale:
			males++
		default:
			females++
		}
	}
	return males, females, dignities
}

// ladyDignityCards counts the lady dignity cards, one per dignity, which come
// on top of the base deck size.
func ladyDignityCards(deck *models.NobleDeck) int {
	count := 0
	for _, card := range deck.Cards {
		if card.Kind == models.NobleCardKindDignity && card.Dignity != models.DignityBastard {
			count++
		}
	}
	return count
}

func claimsOf(deck *models.NobleDeck) int {
	claims := 0
	for _, card := range deck.Cards {
		if card.Kind == models.NobleCardKindClaim {
			claims++
		}
	}
	return claims
}

func TestBuildNobleDeckSizesParityAndBastards(t *testing.T) {
	prenoms := realPrenoms(t)
	for players := models.MinimumGamePlayers; players <= models.MaximumGamePlayers; players++ {
		t.Run(fmt.Sprintf("%d players", players), func(t *testing.T) {
			used := startingCodes(prenoms, 2*players)
			deck := buildNobleDeck("deck-sizes", deckTestPlayers(players), 6, prenoms, used, 0)
			if deck == nil {
				t.Fatal("deck = nil, want a deck")
			}
			males, females, dignities := countKinds(deck)
			claims := claimsOf(deck)
			if ladies := ladyDignityCards(deck); ladies != len(models.LadyDignities) {
				t.Errorf("lady dignity cards = %d, want %d", ladies, len(models.LadyDignities))
			}
			nobles := males + females
			if want := max(1, nobles/(2*max(players-1, 4))); claims != want {
				t.Errorf("claim cards = %d, want %d on top of the deck", claims, want)
			}
			if nobles > players*7 || (players <= 9 && nobles != players*7) {
				t.Errorf("noble cards = %d, want %d (7 per player)", nobles, players*7)
			}
			if want := 2 * players; dignities != want {
				t.Errorf("bastard cards = %d, want %d (2 per player)", dignities, want)
			}
			if diff := males - females; diff < -1 || diff > 1 {
				t.Errorf("males = %d, females = %d, want an even split", males, females)
			}
			if len(deck.DrawPile) != len(deck.Cards)-len(deck.Reserve) || len(deck.Played) != 0 || len(deck.Discard) != 0 {
				t.Errorf("draw pile = %d, played = %d, want the whole deck in the pile", len(deck.DrawPile), len(deck.Played))
			}
			codes := map[string]bool{}
			for _, card := range deck.Cards {
				if card.Kind != models.NobleCardKindNoble {
					continue
				}
				if used[card.Code] || card.Code == models.DignityBastardCardCode || card.Code == models.ClaimCardCode || codes[card.Code] {
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
	deck := buildNobleDeck("deck-four", deckTestPlayers(4), 6, prenoms, startingCodes(prenoms, 8), 0)
	males, females, dignities := countKinds(deck)
	claims := claimsOf(deck)
	if len(deck.Cards) != 50 || dignities != 8 || claims != 3 || males+females != 28 {
		t.Errorf("deck = %d cards (%d bastards, %d claims, %d nobles), want 50 (8 bastards, 10 lady dignities, 3 claims, 28 nobles, 1 reserved)", len(deck.Cards), dignities, claims, males+females)
	}
	for _, player := range deckTestPlayers(4) {
		if hand, exists := deck.Hands[player.ID]; !exists || len(hand) != 0 {
			t.Errorf("hand of %s = %v, want an empty hand", player.ID, hand)
		}
	}
}

// One more male noble card is set aside: it is a card of the deck, never in
// the draw pile, and its name is out of the name pool.
func TestBuildNobleDeckSetsAsideAMaleCard(t *testing.T) {
	prenoms := realPrenoms(t)
	positions := map[int]bool{}
	for players := models.MinimumGamePlayers; players <= models.MaximumGamePlayers; players++ {
		deck := buildNobleDeck("deck-reserve", deckTestPlayers(players), 6, prenoms, startingCodes(prenoms, 2*players), 0)
		if len(deck.Reserve) != 1 {
			t.Fatalf("%d players: reserve = %v, want one card", players, deck.Reserve)
		}
		card, found := deck.Card(deck.Reserve[0])
		if !found || card.Kind != models.NobleCardKindNoble || card.Sex != models.SexMale {
			t.Fatalf("%d players: reserved card = %+v, want a male noble card", players, card)
		}
		if slices.Contains(deck.DrawPile, card.ID) || slices.ContainsFunc(deck.NamePool, func(name models.NobleName) bool { return name.Code == card.Code }) {
			t.Errorf("%d players: reserved card %+v is dealt or its name is in the pool", players, card)
		}
	}
	for index := 0; index < 30; index++ {
		deck := buildNobleDeck(fmt.Sprintf("reserve-%d", index), deckTestPlayers(4), 6, prenoms, startingCodes(prenoms, 8), 0)
		males, _, _ := countKinds(deck)
		position := slices.IndexFunc(deck.Cards, func(card models.NobleCard) bool { return card.ID == deck.Reserve[0] })
		positions[position] = true
		if position < 0 || position > males {
			t.Fatalf("reserved card at %d, want it among the %d male cards", position, males+1)
		}
	}
	if len(positions) < 3 {
		t.Errorf("reserved card positions = %v, want them to vary with the seed", positions)
	}
}

func TestBuildNobleDeckIsDeterministicAndSeeded(t *testing.T) {
	prenoms := realPrenoms(t)
	build := func(seed string) *models.NobleDeck {
		return buildNobleDeck(seed, deckTestPlayers(4), 6, prenoms, startingCodes(prenoms, 8), 0)
	}
	if !reflect.DeepEqual(build("same"), build("same")) {
		t.Error("the same seed built two different decks")
	}
	if reflect.DeepEqual(build("seed-a").DrawPile, build("seed-b").DrawPile) {
		t.Error("two seeds built the same shuffled draw pile")
	}
	// The seed decides which sex gets the odd card: 5 players make 35 noble
	// cards, an odd number.
	sawMale, sawFemale := false, false
	for index := 0; index < 20; index++ {
		deck := buildNobleDeck(fmt.Sprintf("odd-%d", index), deckTestPlayers(5), 6, prenoms, startingCodes(prenoms, 10), 0)
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
	deck := buildNobleDeck("tiny", deckTestPlayers(2), 6, prenoms, map[string]bool{"DDD": true}, 0)
	males, females, dignities := countKinds(deck)
	if males > 2 || females > 1 || dignities < 1 {
		t.Errorf("deck = %d males, %d females, %d dignities, want it clamped to the three free names", males, females, dignities)
	}
	if deck := buildNobleDeck("none", deckTestPlayers(2), 6, nil, nil, 0); deck != nil {
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
	if deck == nil || len(deck.Cards) != 52 {
		t.Fatalf("deck = %+v, want 52 cards", deck)
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
