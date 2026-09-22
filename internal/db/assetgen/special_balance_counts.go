package assetgen

import "github.com/fogfactory/crown-and-borough/internal/models"

func WeightedCardCounts(total int, weights map[models.CardKind]int, canonical []models.CardKind) map[models.CardKind]int {
	return weightedCounts(total, weights, canonical)
}
