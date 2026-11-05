package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

func copySeasonCapacities(source map[models.Season]int) map[models.Season]int {
	copy := make(map[models.Season]int, len(source))
	for season, capacity := range source {
		copy[season] = capacity
	}
	return copy
}
