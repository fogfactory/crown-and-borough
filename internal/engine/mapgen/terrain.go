package mapgen

import (
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type terrainSeed struct {
	site    int
	terrain models.Terrain
}

var allTerrains = []models.Terrain{
	models.TerrainPlain,
	models.TerrainForest,
	models.TerrainHill,
	models.TerrainMountain,
	models.TerrainSwamp,
}

// assignTerrains grows contiguous zones from several site seeds. The first
// five seeds receive a shuffled complete terrain set, guaranteeing presence.
func assignTerrains(rng *rand.Rand, sites []point) []models.Terrain {
	seedCount := len(sites) / 8
	if seedCount < len(allTerrains) {
		seedCount = len(allTerrains)
	}
	if seedCount > len(sites) {
		seedCount = len(sites)
	}

	indexes := make([]int, len(sites))
	for i := range indexes {
		indexes[i] = i
	}
	shuffle(rng, indexes)

	permutedTerrains := append([]models.Terrain(nil), allTerrains...)
	shuffle(rng, permutedTerrains)
	seeds := make([]terrainSeed, seedCount)
	for i := 0; i < seedCount; i++ {
		terrain := randomTerrain(rng)
		if i < len(permutedTerrains) {
			terrain = permutedTerrains[i]
		}
		seeds[i] = terrainSeed{site: indexes[i], terrain: terrain}
	}

	terrain := make([]models.Terrain, len(sites))
	for siteIndex, site := range sites {
		closest := seeds[0]
		closestDistance := squaredDistance(site.x, site.y, sites[closest.site].x, sites[closest.site].y)
		for _, candidate := range seeds[1:] {
			distance := squaredDistance(site.x, site.y, sites[candidate.site].x, sites[candidate.site].y)
			if distance < closestDistance || (distance == closestDistance && candidate.site < closest.site) {
				closest = candidate
				closestDistance = distance
			}
		}
		terrain[siteIndex] = closest.terrain
	}
	return terrain
}

func randomTerrain(rng *rand.Rand) models.Terrain {
	roll := rng.Float64()
	switch {
	case roll < 0.35:
		return models.TerrainPlain
	case roll < 0.60:
		return models.TerrainForest
	case roll < 0.80:
		return models.TerrainHill
	case roll < 0.90:
		return models.TerrainMountain
	default:
		return models.TerrainSwamp
	}
}
