package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// nobleDeckSize returns the size of the noble deck and how many of its cards
// are dignity cards (specs/succession.md § Deck de nobles): players ×
// (noble_limit_max + 1) cards, reduced until the noble cards can all take a
// free name, of which maleAvailable and femaleAvailable remain. Dignity
// cards are one card in max(players-1, 4), at least one. The noble cards are
// split evenly between the sexes; oddMale tells which sex gets the extra one
// when their number is odd. The size is zero when no noble card can be made.
func nobleDeckSize(players, limitMax, maleAvailable, femaleAvailable int, oddMale bool) (size, dignities, males, females int) {
	for size = players * (limitMax + 1); size > 0; size-- {
		dignities = max(1, size/max(players-1, 4))
		nobles := size - dignities
		if nobles <= 0 {
			continue
		}
		males, females = nobles/2, nobles/2
		if nobles%2 == 1 {
			if oddMale {
				males++
			} else {
				females++
			}
		}
		if males <= maleAvailable && females <= femaleAvailable {
			return size, dignities, males, females
		}
	}
	return 0, 0, 0, 0
}

// buildNobleDeck generates the shared noble deck deterministically from the
// game seed. usedCodes are the codes already taken by the starting nobles; the
// dignity card codes are reserved too. It returns nil when no noble card
// can be named.
func buildNobleDeck(seed string, players []models.Player, limitMax int, prenoms []assetgen.Asset, usedCodes map[string]bool) *models.NobleDeck {
	rng := newNobleDeckRNG(seed)
	oddMale := rng.IntN(2) == 0
	reserved := make(map[string]bool, len(usedCodes)+1)
	for code := range usedCodes {
		reserved[code] = true
	}
	reserved[models.DignityBastardCardCode] = true
	var male, female []assetgen.Asset
	for _, prenom := range prenoms {
		if reserved[prenom.Code] {
			continue
		}
		if models.Sex(prenom.Sex) == models.SexFemale {
			female = append(female, prenom)
		} else {
			male = append(male, prenom)
		}
	}
	size, dignities, males, females := nobleDeckSize(len(players), limitMax, len(male), len(female), oddMale)
	if size == 0 {
		return nil
	}
	shuffleAssets(rng, male)
	shuffleAssets(rng, female)

	cards := make([]models.NobleCard, 0, size)
	addNobles := func(assets []assetgen.Asset, sex models.Sex) {
		for _, asset := range assets {
			cards = append(cards, models.NobleCard{
				ID:   nobleCardID(len(cards) + 1),
				Kind: models.NobleCardKindNoble,
				Code: asset.Code,
				Name: asset.Name,
				Sex:  sex,
			})
		}
	}
	addNobles(male[:males], models.SexMale)
	addNobles(female[:females], models.SexFemale)
	for index := 0; index < dignities; index++ {
		cards = append(cards, models.NobleCard{
			ID:      nobleCardID(len(cards) + 1),
			Kind:    models.NobleCardKindDignity,
			Code:    models.DignityBastardCardCode,
			Dignity: models.DignityBastard,
		})
	}
	drawPile := make([]models.NobleCardID, len(cards))
	for index, card := range cards {
		drawPile[index] = card.ID
	}
	for index := len(drawPile) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		drawPile[index], drawPile[swap] = drawPile[swap], drawPile[index]
	}
	deck := &models.NobleDeck{
		Cards:    cards,
		DrawPile: drawPile,
		Discard:  []models.NobleCardID{},
		Hands:    make(map[models.PlayerID][]models.NobleCardID, len(players)),
		Played:   []models.NobleCardPlay{},
		NamePool: []models.NobleName{},
	}
	// The names no card took feed the replacement cards, in a seeded order.
	for _, asset := range male[males:] {
		deck.NamePool = append(deck.NamePool, models.NobleName{Code: asset.Code, Name: asset.Name, Sex: models.SexMale})
	}
	for _, asset := range female[females:] {
		deck.NamePool = append(deck.NamePool, models.NobleName{Code: asset.Code, Name: asset.Name, Sex: models.SexFemale})
	}
	for index := len(deck.NamePool) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		deck.NamePool[index], deck.NamePool[swap] = deck.NamePool[swap], deck.NamePool[index]
	}
	for _, player := range players {
		deck.Hands[player.ID] = []models.NobleCardID{}
	}
	return deck
}

func nobleCardID(number int) models.NobleCardID {
	return models.NobleCardID(fmt.Sprintf("K%d", number))
}

func newNobleDeckRNG(seed string) *rand.Rand {
	digest := sha256.Sum256([]byte(seed + "|noble-deck|initial"))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func shuffleAssets(rng *rand.Rand, values []assetgen.Asset) {
	for index := len(values) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		values[index], values[swap] = values[swap], values[index]
	}
}

func cloneNobleDeck(source *models.NobleDeck) *models.NobleDeck {
	if source == nil {
		return nil
	}
	clone := &models.NobleDeck{
		Cards:      cloneSlice(source.Cards),
		DrawPile:   cloneSlice(source.DrawPile),
		Discard:    cloneSlice(source.Discard),
		Played:     cloneSlice(source.Played),
		Reshuffles: source.Reshuffles,
		NamePool:   cloneSlice(source.NamePool),
	}
	if source.Hands != nil {
		clone.Hands = make(map[models.PlayerID][]models.NobleCardID, len(source.Hands))
		for playerID, hand := range source.Hands {
			clone.Hands[playerID] = cloneSlice(hand)
		}
	}
	return clone
}

func newNobleReshuffleRNG(seed string, count int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|noble-deck|reshuffle|%d", seed, count)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

// drawNobleCard takes the top card of the draw pile. When the pile is empty,
// the discard pile is first shuffled into it, from a seed derived from the
// game seed and the reshuffle counter. It reports false when both piles are
// empty.
func (ctx *resolutionContext) drawNobleCard() (models.NobleCardID, bool) {
	deck := ctx.state.NobleDeck
	if deck == nil {
		return "", false
	}
	if len(deck.DrawPile) == 0 {
		if len(deck.Discard) == 0 {
			return "", false
		}
		deck.Reshuffles++
		deck.DrawPile = deck.Discard
		deck.Discard = []models.NobleCardID{}
		rng := newNobleReshuffleRNG(ctx.state.Seed, deck.Reshuffles)
		for index := len(deck.DrawPile) - 1; index > 0; index-- {
			swap := rng.IntN(index + 1)
			deck.DrawPile[index], deck.DrawPile[swap] = deck.DrawPile[swap], deck.DrawPile[index]
		}
	}
	cardID := deck.DrawPile[0]
	deck.DrawPile = deck.DrawPile[1:]
	return cardID, true
}

// removeDignity takes a dignity away from a noble, whatever the game effect
// that does it. The dignity card that conferred it, if any, goes back to the
// discard pile. Every effect that removes a dignity goes through here.
func (ctx *resolutionContext) removeDignity(noble *models.Noble, dignity models.Dignity) {
	noble.Dignities = slices.DeleteFunc(noble.Dignities, func(carried models.Dignity) bool { return carried == dignity })
	deck := ctx.state.NobleDeck
	if deck == nil {
		return
	}
	for index, play := range deck.Played {
		if play.Noble != noble.ID {
			continue
		}
		if card, exists := deck.Card(play.Card); exists && card.Kind == models.NobleCardKindDignity && card.Dignity == dignity {
			deck.Played = slices.Delete(deck.Played, index, index+1)
			deck.Discard = append(deck.Discard, play.Card)
			return
		}
	}
}

// releaseNobleCards settles the cards played on a noble that has just died
// or left play for good. Its noble card leaves the deck and a new card of the
// same sex, with a name no card, noble or dead noble uses, goes to the
// discard pile (none when no name is left); the code of the dead noble stays
// reserved. Its dignity cards go back to the discard pile. A noble without
// played cards (a starting noble) leaves nothing behind.
func (ctx *resolutionContext) releaseNobleCards(noble models.Noble) {
	deck := ctx.state.NobleDeck
	if deck == nil {
		return
	}
	kept := deck.Played[:0]
	for _, play := range deck.Played {
		if play.Noble != noble.ID {
			kept = append(kept, play)
			continue
		}
		card, _ := deck.Card(play.Card)
		if card.Kind == models.NobleCardKindDignity {
			deck.Discard = append(deck.Discard, play.Card)
			continue
		}
		deck.Cards = slices.DeleteFunc(deck.Cards, func(candidate models.NobleCard) bool { return candidate.ID == play.Card })
		ctx.addReplacementNobleCard(card.Sex)
	}
	deck.Played = kept
}

// addReplacementNobleCard puts a new noble card of the given sex, built from
// the first free name of the pool, on the discard pile.
func (ctx *resolutionContext) addReplacementNobleCard(sex models.Sex) {
	deck := ctx.state.NobleDeck
	for index, name := range deck.NamePool {
		if name.Sex != sex {
			continue
		}
		deck.NamePool = slices.Delete(deck.NamePool, index, index+1)
		card := models.NobleCard{ID: nextNobleCardID(deck), Kind: models.NobleCardKindNoble, Code: name.Code, Name: name.Name, Sex: name.Sex}
		deck.Cards = append(deck.Cards, card)
		deck.Discard = append(deck.Discard, card.ID)
		return
	}
}

// nextNobleCardID returns the first card ID above every ID in use.
func nextNobleCardID(deck *models.NobleDeck) models.NobleCardID {
	highest := 0
	for _, card := range deck.Cards {
		var number int
		if _, err := fmt.Sscanf(string(card.ID), "K%d", &number); err == nil {
			highest = max(highest, number)
		}
	}
	return nobleCardID(highest + 1)
}
