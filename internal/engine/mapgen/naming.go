package mapgen

import (
	"fmt"
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type nameCode struct {
	name string
	code string
}

var affinityOrder = [...]string{
	"plain",
	"forest",
	"hill",
	"mountain",
	"swamp",
	"any",
}

func validateAssets(assets assetgen.Assets, siteCount int) error {
	if len(assets.Communes) == 0 {
		return fmt.Errorf("mapgen: no communes available for naming")
	}
	if len(assets.Communes) < siteCount {
		return fmt.Errorf("mapgen: need at least %d communes for territories, have %d", siteCount, len(assets.Communes))
	}

	communeCodes := make(map[string]bool, len(assets.Communes))
	communeNames := make(map[string]bool, len(assets.Communes))
	for _, commune := range assets.Communes {
		if !isTrigram(commune.Code) || commune.Name == "" || !isCommuneTerrain(commune.Terrain) {
			return fmt.Errorf("mapgen: invalid commune asset")
		}
		if communeCodes[commune.Code] {
			return fmt.Errorf("mapgen: duplicate commune code %q", commune.Code)
		}
		if communeNames[commune.Name] {
			return fmt.Errorf("mapgen: duplicate commune name %q", commune.Name)
		}
		communeCodes[commune.Code] = true
		communeNames[commune.Name] = true
	}
	return nil
}

func isTrigram(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func isCommuneTerrain(value string) bool {
	for _, terrain := range affinityOrder {
		if value == terrain {
			return true
		}
	}
	return false
}

// nameTerritories assigns each territory a distinct commune. It first ensures
// that every terrain present has a commune with the matching affinity, then
// prioritizes matching affinities, any-terrain communes, and a fixed fallback
// order. Each bucket is shuffled from the naming RNG for deterministic variety.
func nameTerritories(
	rng *rand.Rand,
	assets assetgen.Assets,
	terrain []models.Terrain,
) ([]nameCode, error) {
	for site, siteTerrain := range terrain {
		if !siteTerrain.IsValid() {
			return nil, fmt.Errorf("mapgen: invalid terrain %q on site %d", siteTerrain, site)
		}
	}

	buckets := make(map[string][]assetgen.Commune, len(affinityOrder))
	for _, commune := range assets.Communes {
		buckets[commune.Terrain] = append(buckets[commune.Terrain], commune)
	}
	for _, affinity := range affinityOrder {
		shuffle(rng, buckets[affinity])
	}

	names := make([]nameCode, len(terrain))
	named := make([]bool, len(terrain))
	for _, affinity := range affinityOrder[:len(affinityOrder)-1] {
		site := -1
		for index, siteTerrain := range terrain {
			if !named[index] && string(siteTerrain) == affinity {
				site = index
				break
			}
		}
		if site == -1 {
			continue
		}
		commune, ok := popCommune(buckets, affinity)
		if !ok {
			return nil, fmt.Errorf("mapgen: exhausted communes with terrain %q during affinity coverage", affinity)
		}
		names[site] = nameCode{name: commune.Name, code: commune.Code}
		named[site] = true
	}

	for site, siteTerrain := range terrain {
		if named[site] {
			continue
		}
		priorities := [...]string{string(siteTerrain), "any"}
		var commune assetgen.Commune
		assigned := false
		for _, affinity := range priorities {
			if commune, assigned = popCommune(buckets, affinity); assigned {
				break
			}
		}
		if !assigned {
			for _, affinity := range affinityOrder {
				if commune, assigned = popCommune(buckets, affinity); assigned {
					break
				}
			}
		}
		if !assigned {
			return nil, fmt.Errorf("mapgen: exhausted communes while naming territory %s", siteLabel(site))
		}
		names[site] = nameCode{name: commune.Name, code: commune.Code}
	}
	return names, nil
}

func popCommune(buckets map[string][]assetgen.Commune, affinity string) (assetgen.Commune, bool) {
	communes := buckets[affinity]
	if len(communes) == 0 {
		return assetgen.Commune{}, false
	}
	last := len(communes) - 1
	commune := communes[last]
	buckets[affinity] = communes[:last]
	return commune, true
}
