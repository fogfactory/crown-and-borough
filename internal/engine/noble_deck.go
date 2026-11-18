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

// bastardCardsPerPlayer is the number of bastard cards the noble deck holds
// for each player.
const bastardCardsPerPlayer = 2

// nobleDeckSize returns the size of the noble deck and how many of its cards
// are bastard cards (specs/succession.md § Deck de nobles): players ×
// (noble_limit_max + 1) noble cards, reduced until they can all take a free
// name, of which maleAvailable and femaleAvailable remain, and
// bastardCardsPerPlayer bastard cards per player. The noble cards are split
// evenly between the sexes; oddMale tells which sex gets the extra one when
// their number is odd. The size counts these two kinds of cards. Claim cards
// come on top of it: one card in 2 × max(players-1, 4) noble cards, at least
// one; one card of each lady dignity does the same. The size is zero when no
// noble card can be made.
func nobleDeckSize(players, limitMax, maleAvailable, femaleAvailable int, oddMale bool) (size, dignities, claims, males, females int) {
	dignities = bastardCardsPerPlayer * players
	for nobles := players * (limitMax + 1); nobles > 0; nobles-- {
		males, females = nobles/2, nobles/2
		if nobles%2 == 1 {
			if oddMale {
				males++
			} else {
				females++
			}
		}
		if males <= maleAvailable && females <= femaleAvailable {
			size = nobles + dignities
			claims = max(1, nobles/(2*max(players-1, 4)))
			return size, dignities, claims, males, females
		}
	}
	return 0, 0, 0, 0, 0
}

// buildNobleDeck generates the shared noble deck deterministically from the
// game seed. usedCodes are the codes already taken by the starting nobles; the
// dignity card codes are reserved too. cardinalCards is the number of cardinal
// cards added to the deck. One more male noble card is set aside in the deck's
// reserve, never dealt, when a free male name is left: the Chevalier d'Éon
// takes her male identity from it (see Reserve in models.NobleDeck). It returns
// nil when no noble card can be named.
func buildNobleDeck(seed string, players []models.Player, limitMax int, prenoms []assetgen.Asset, usedCodes map[string]bool, cardinalCards int) *models.NobleDeck {
	rng := newNobleDeckRNG(seed)
	oddMale := rng.IntN(2) == 0
	reserved := make(map[string]bool, len(usedCodes)+1)
	for code := range usedCodes {
		reserved[code] = true
	}
	reserved[models.DignityBastardCardCode] = true
	reserved[models.ClaimCardCode] = true
	reserved[models.DignityCardinalCardCode] = true
	for _, dignity := range models.LadyDignities {
		reserved[dignity.Effect().CardCode] = true
	}
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
	size, dignities, claims, males, females := nobleDeckSize(len(players), limitMax, len(male), len(female), oddMale)
	if size == 0 {
		return nil
	}
	shuffleAssets(rng, male)
	shuffleAssets(rng, female)

	cards := make([]models.NobleCard, 0, size+1)
	addNoble := func(asset assetgen.Asset, sex models.Sex) models.NobleCardID {
		card := models.NobleCard{
			ID:   nobleCardID(len(cards) + 1),
			Kind: models.NobleCardKindNoble,
			Code: asset.Code,
			Name: asset.Name,
			Sex:  sex,
		}
		cards = append(cards, card)
		return card.ID
	}
	// The reserved male card sits at a random place among the male cards, so
	// that nothing in the order of the cards tells it apart.
	maleBlock, reserveAt := male[:males], -1
	if len(male) > males {
		maleBlock = male[:males+1]
		reserveAt = rng.IntN(len(maleBlock))
	}
	var reserve []models.NobleCardID
	for index, asset := range maleBlock {
		if id := addNoble(asset, models.SexMale); index == reserveAt {
			reserve = append(reserve, id)
		}
	}
	for _, asset := range female[:females] {
		addNoble(asset, models.SexFemale)
	}
	for index := 0; index < dignities; index++ {
		cards = append(cards, models.NobleCard{
			ID:      nobleCardID(len(cards) + 1),
			Kind:    models.NobleCardKindDignity,
			Code:    models.DignityBastardCardCode,
			Dignity: models.DignityBastard,
		})
	}
	// One card of each lady dignity, on top of the base size like the claims.
	for _, dignity := range models.LadyDignities {
		cards = append(cards, models.NobleCard{
			ID:      nobleCardID(len(cards) + 1),
			Kind:    models.NobleCardKindDignity,
			Code:    dignity.Effect().CardCode,
			Dignity: dignity,
		})
	}
	// The cardinal cards come on top of the base size too: their number is the
	// cap of the cardinals obtained by card (religion.cardinal_card_*).
	for index := 0; index < cardinalCards; index++ {
		cards = append(cards, models.NobleCard{
			ID:      nobleCardID(len(cards) + 1),
			Kind:    models.NobleCardKindDignity,
			Code:    models.DignityCardinalCardCode,
			Dignity: models.DignityCardinal,
		})
	}
	for index := 0; index < claims; index++ {
		cards = append(cards, models.NobleCard{
			ID:   nobleCardID(len(cards) + 1),
			Kind: models.NobleCardKindClaim,
			Code: models.ClaimCardCode,
		})
	}
	drawPile := make([]models.NobleCardID, 0, len(cards))
	for _, card := range cards {
		if !slices.Contains(reserve, card.ID) {
			drawPile = append(drawPile, card.ID)
		}
	}
	for index := len(drawPile) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		drawPile[index], drawPile[swap] = drawPile[swap], drawPile[index]
	}
	deck := &models.NobleDeck{
		Cards:    cards,
		Reserve:  reserve,
		DrawPile: drawPile,
		Discard:  []models.NobleCardID{},
		Hands:    make(map[models.PlayerID][]models.NobleCardID, len(players)),
		Played:   []models.NobleCardPlay{},
		NamePool: []models.NobleName{},
	}
	// The names no card took feed the replacement cards, in a seeded order.
	for _, asset := range male[len(maleBlock):] {
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
		Reserve:    cloneSlice(source.Reserve),
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
// played cards (a starting noble) leaves nothing behind. A chevalier d'Éon
// carries two noble cards, the lady's and the male identity taken from the
// reserve: the second one is replaced inside the reserve.
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
		if card.Kind != models.NobleCardKindNoble {
			deck.Discard = append(deck.Discard, play.Card)
			continue
		}
		deck.Cards = slices.DeleteFunc(deck.Cards, func(candidate models.NobleCard) bool { return candidate.ID == play.Card })
		// The male identity of a chevalier d'Éon came from the reserve: its
		// replacement goes back there, so that the deck keeps a card set aside.
		ctx.addReplacementNobleCard(card.Sex, noble.SecretCode != "" && card.Code == noble.Code)
	}
	deck.Played = kept
}

// addReplacementNobleCard puts a new noble card of the given sex, built from
// the first free name of the pool, on the discard pile, or in the reserve when
// toReserve is set.
func (ctx *resolutionContext) addReplacementNobleCard(sex models.Sex, toReserve bool) {
	deck := ctx.state.NobleDeck
	for index, name := range deck.NamePool {
		if name.Sex != sex {
			continue
		}
		deck.NamePool = slices.Delete(deck.NamePool, index, index+1)
		card := models.NobleCard{ID: nextNobleCardID(deck), Kind: models.NobleCardKindNoble, Code: name.Code, Name: name.Name, Sex: name.Sex}
		deck.Cards = append(deck.Cards, card)
		if toReserve {
			deck.Reserve = append(deck.Reserve, card.ID)
		} else {
			deck.Discard = append(deck.Discard, card.ID)
		}
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
