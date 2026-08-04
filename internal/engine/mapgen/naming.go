package mapgen

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type nameCode struct {
	name string
	code string
}

func validateAssets(assets assetgen.Assets, villageCount int) error {
	if len(assets.Communes) == 0 {
		return fmt.Errorf("mapgen: no communes available for naming")
	}
	if len(assets.Communes) < villageCount {
		return fmt.Errorf("mapgen: need at least %d communes for villages, have %d", villageCount, len(assets.Communes))
	}
	if len(assets.Qualificatifs) == 0 {
		return fmt.Errorf("mapgen: no qualifiers available for naming")
	}

	communeCodes := make(map[string]bool, len(assets.Communes))
	for _, commune := range assets.Communes {
		if !isTrigram(commune.Code) || commune.Name == "" {
			return fmt.Errorf("mapgen: invalid commune asset")
		}
		if communeCodes[commune.Code] {
			return fmt.Errorf("mapgen: duplicate commune code %q", commune.Code)
		}
		communeCodes[commune.Code] = true
	}

	prefixes := make(map[string]bool, len(assets.Qualificatifs))
	for _, qualifier := range assets.Qualificatifs {
		if !isPrefix(qualifier.Prefix) || qualifier.Name == "" || !isQualifierTerrain(qualifier.Terrain) {
			return fmt.Errorf("mapgen: invalid qualifier asset")
		}
		if prefixes[qualifier.Prefix] {
			return fmt.Errorf("mapgen: duplicate qualifier prefix %q", qualifier.Prefix)
		}
		prefixes[qualifier.Prefix] = true
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

func isPrefix(value string) bool {
	return len(value) == 1 && value[0] >= 'A' && value[0] <= 'Z'
}

func isQualifierTerrain(value string) bool {
	switch models.Terrain(value) {
	case models.TerrainPlain, models.TerrainForest, models.TerrainHill, models.TerrainMountain, models.TerrainSwamp:
		return true
	}
	return value == "any"
}

// assignVillages spreads count neutral villages over the map, maximizing the
// minimum distance between them (greedy max-min): the first village is drawn
// seeded at random, then each next village is the site whose distance to its
// nearest already-placed village is the largest. Every terrain is eligible.
// Ties break on the lowest site index for determinism.
func assignVillages(rng *rand.Rand, terrain []models.Terrain, centroids [][2]float64, count int) ([]bool, error) {
	n := len(terrain)
	if count < 1 || count > n {
		return nil, fmt.Errorf("mapgen: cannot place %d villages on %d sites", count, n)
	}
	if len(centroids) != n {
		return nil, fmt.Errorf("mapgen: internal village input length mismatch")
	}

	eligible := make([]int, 0, n)
	for site := 0; site < n; site++ {
		if !terrain[site].IsValid() {
			return nil, fmt.Errorf("mapgen: invalid terrain %q on site %d", terrain[site], site)
		}
		eligible = append(eligible, site)
	}
	if count > len(eligible) {
		return nil, fmt.Errorf("mapgen: need at least %d eligible sites for villages, have %d", count, len(eligible))
	}

	selected := make([]bool, n)
	first := eligible[rng.IntN(len(eligible))]
	selected[first] = true
	chosen := []int{first}

	for len(chosen) < count {
		bestSite := -1
		bestDistance := -1.0
		for _, site := range eligible {
			if selected[site] {
				continue
			}
			nearest := squaredDistanceToChosen(centroids, site, chosen)
			if bestSite == -1 || nearest > bestDistance ||
				(nearest == bestDistance && site < bestSite) {
				bestSite = site
				bestDistance = nearest
			}
		}
		selected[bestSite] = true
		chosen = append(chosen, bestSite)
	}
	return selected, nil
}

// squaredDistanceToChosen returns the squared centroid distance from site to
// its nearest already-chosen village site.
func squaredDistanceToChosen(centroids [][2]float64, site int, chosen []int) float64 {
	nearest := math.Inf(1)
	for _, other := range chosen {
		distance := centroidDistanceSquared(centroids, site, other)
		if distance < nearest {
			nearest = distance
		}
	}
	return nearest
}

// nameTerritories names territories. TRANSITIONAL naming (bridge to P1.2c,
// which switches every territory to a commune code): a village tile receives
// the name of a commune (code trigram), every other tile receives
// "{Qualificatif} de {Commune of the nearest/adjacent village}" (existing
// pattern). P1.2c replaces the whole system with "commune for all
// territories": do not go further here.
func nameTerritories(
	rng *rand.Rand,
	assets assetgen.Assets,
	villages []bool,
	terrain []models.Terrain,
	polygons [][][2]int,
	centroids [][2]float64,
	edges [][2]int,
	cfg Config,
) ([]nameCode, error) {
	n := len(villages)
	if len(terrain) != n || len(polygons) != n || len(centroids) != n {
		return nil, fmt.Errorf("mapgen: internal naming input length mismatch")
	}

	communes := append([]assetgen.Asset(nil), assets.Communes...)
	shuffle(rng, communes)
	names := make([]nameCode, n)
	communesBySite := make([]assetgen.Asset, n)
	villageSites := make([]int, 0)
	usedCodes := make(map[string]bool)
	nextCommune := 0
	for site, isVillage := range villages {
		if !isVillage {
			continue
		}
		if nextCommune >= len(communes) {
			return nil, fmt.Errorf("mapgen: exhausted communes while naming villages")
		}
		commune := communes[nextCommune]
		nextCommune++
		if usedCodes[commune.Code] {
			return nil, fmt.Errorf("mapgen: duplicate village code %q", commune.Code)
		}
		names[site] = nameCode{name: commune.Name, code: commune.Code}
		communesBySite[site] = commune
		usedCodes[commune.Code] = true
		villageSites = append(villageSites, site)
	}
	if len(villageSites) == 0 {
		return nil, fmt.Errorf("mapgen: cannot name territories without a village")
	}

	adjacency := edgeMatrix(n, edges)
	usedPairs := make(map[string]bool)
	for site, isVillage := range villages {
		if isVillage {
			continue
		}

		qualifiers := compatibleQualifiers(assets.Qualificatifs, terrain[site], isBorder(polygons[site], cfg))
		if len(qualifiers) == 0 {
			return nil, fmt.Errorf("mapgen: no qualifier available for terrain %q on territory %s", terrain[site], territoryID(site))
		}

		assign := func(qualifier assetgen.Qualificatif, commune assetgen.Asset) bool {
			pair := qualifier.Prefix + "\x00" + commune.Code
			code := qualifier.Prefix + commune.Code
			if usedPairs[pair] || usedCodes[code] {
				return false
			}
			names[site] = nameCode{
				name: qualifier.Name + " de " + commune.Name,
				code: code,
			}
			usedPairs[pair] = true
			usedCodes[code] = true
			return true
		}

		assigned := false
		for _, villageSite := range orderedVillages(site, villageSites, adjacency, centroids, n) {
			commune := communesBySite[villageSite]
			for _, qualifier := range qualifiers {
				if assign(qualifier, commune) {
					assigned = true
					break
				}
			}
			if assigned {
				break
			}
		}
		// Fallback (transitional, documented): with only a handful of
		// villages the (qualifier × village commune) pairs can be exhausted;
		// name the tile after an unused commune from the pool to keep codes
		// unique. P1.2c replaces this whole scheme with a commune for every
		// territory.
		if !assigned {
			for _, qualifier := range qualifiers {
				for nextCommune < len(communes) {
					commune := communes[nextCommune]
					nextCommune++
					if usedCodes[commune.Code] {
						continue
					}
					if assign(qualifier, commune) {
						assigned = true
						break
					}
				}
				if assigned {
					break
				}
			}
		}
		if !assigned {
			return nil, fmt.Errorf("mapgen: exhausted qualifier and commune combinations for territory %s", territoryID(site))
		}
	}

	return names, nil
}

func compatibleQualifiers(
	qualifiers []assetgen.Qualificatif,
	terrain models.Terrain,
	border bool,
) []assetgen.Qualificatif {
	compatible := make([]assetgen.Qualificatif, 0)
	for _, qualifier := range qualifiers {
		if qualifier.Terrain == string(terrain) || (border && qualifier.Terrain == "any") {
			compatible = append(compatible, qualifier)
		}
	}
	return compatible
}

func orderedVillages(
	site int,
	villageSites []int,
	adjacency []bool,
	centroids [][2]float64,
	n int,
) []int {
	adjacent := make([]int, 0)
	fallback := make([]int, 0)
	for _, villageSite := range villageSites {
		if matrixHasEdge(adjacency, n, site, villageSite) {
			adjacent = append(adjacent, villageSite)
		} else {
			fallback = append(fallback, villageSite)
		}
	}
	sortLieuSites(adjacent, site, centroids)
	sortLieuSites(fallback, site, centroids)
	return append(adjacent, fallback...)
}

func sortLieuSites(sites []int, from int, centroids [][2]float64) {
	sort.Slice(sites, func(first, second int) bool {
		firstDistance := centroidDistanceSquared(centroids, from, sites[first])
		secondDistance := centroidDistanceSquared(centroids, from, sites[second])
		if firstDistance == secondDistance {
			return sites[first] < sites[second]
		}
		return firstDistance < secondDistance
	})
}

func isBorder(points [][2]int, cfg Config) bool {
	for _, point := range points {
		if point[0] == 0 || point[0] == cfg.Width || point[1] == 0 || point[1] == cfg.Height {
			return true
		}
	}
	return false
}
