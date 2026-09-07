package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand/v2"
	"strconv"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

var specialCalamityKinds = []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindFamine}
var specialBonusKinds = []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt}

func buildSpecialDeck(seed string, balance assetgen.Balance) (*models.SpecialDeck, error) {
	config := balance.SpecialOrders
	if config.DeckSize <= 0 {
		return nil, nil
	}
	calamityCount := config.DeckSize * config.CalamityPercentage / 100
	counts := assetgen.WeightedCardCounts(calamityCount, config.CalamityWeights, specialCalamityKinds)
	bonusCounts := assetgen.WeightedCardCounts(config.DeckSize-calamityCount, config.BonusWeights, specialBonusKinds)
	cards := make([]models.SpecialCard, 0, config.DeckSize)
	for _, kind := range specialCalamityKinds {
		for index := 0; index < counts[kind]; index++ {
			cards = append(cards, models.SpecialCard{ID: models.SpecialCardID("C" + formatCardNumber(len(cards)+1)), Kind: kind})
		}
	}
	for _, kind := range specialBonusKinds {
		for index := 0; index < bonusCounts[kind]; index++ {
			cards = append(cards, models.SpecialCard{ID: models.SpecialCardID("C" + formatCardNumber(len(cards)+1)), Kind: kind})
		}
	}
	drawPile := make([]models.SpecialCardID, len(cards))
	for index, card := range cards {
		drawPile[index] = card.ID
	}
	shuffleSpecialCards(newSpecialDeckRNG(seed), drawPile)
	return &models.SpecialDeck{Cards: cards, DrawPile: drawPile, Discard: []models.SpecialCardID{}, Hands: map[models.PlayerID][]models.SpecialCardID{}}, nil
}

func formatCardNumber(number int) string {
	return strconv.Itoa(number)
}

func newSpecialDeckRNG(seed string) *rand.Rand {
	digest := sha256.Sum256([]byte(seed + "|special-deck|initial"))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}

func shuffleSpecialCards(rng *rand.Rand, values []models.SpecialCardID) {
	for index := len(values) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		values[index], values[swap] = values[swap], values[index]
	}
}
