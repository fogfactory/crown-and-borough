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

func validateAssets(assets assetgen.Assets, lieuDitCount int) error {
	if len(assets.Communes) == 0 {
		return fmt.Errorf("mapgen: no communes available for naming")
	}
	if len(assets.Communes) < lieuDitCount {
		return fmt.Errorf("mapgen: need at least %d communes for lieu-dits, have %d", lieuDitCount, len(assets.Communes))
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

// assignLieuDits selects the rounded requested count without changing terrain.
func assignLieuDits(rng *rand.Rand, n int, ratio float64) []bool {
	indexes := make([]int, n)
	for i := range indexes {
		indexes[i] = i
	}
	shuffle(rng, indexes)

	selected := make([]bool, n)
	count := int(math.Round(float64(n) * ratio))
	for _, index := range indexes[:count] {
		selected[index] = true
	}
	return selected
}

func nameTerritories(
	rng *rand.Rand,
	assets assetgen.Assets,
	lieuDits []bool,
	terrain []models.Terrain,
	polygons [][][2]int,
	centroids [][2]float64,
	edges [][2]int,
	cfg Config,
) ([]nameCode, error) {
	n := len(lieuDits)
	if len(terrain) != n || len(polygons) != n || len(centroids) != n {
		return nil, fmt.Errorf("mapgen: internal naming input length mismatch")
	}

	communes := append([]assetgen.Asset(nil), assets.Communes...)
	shuffle(rng, communes)
	names := make([]nameCode, n)
	communesBySite := make([]assetgen.Asset, n)
	lieuSites := make([]int, 0)
	usedCodes := make(map[string]bool)
	nextCommune := 0
	for site, isLieuDit := range lieuDits {
		if !isLieuDit {
			continue
		}
		if nextCommune >= len(communes) {
			return nil, fmt.Errorf("mapgen: exhausted communes while naming lieu-dits")
		}
		commune := communes[nextCommune]
		nextCommune++
		if usedCodes[commune.Code] {
			return nil, fmt.Errorf("mapgen: duplicate lieu-dit code %q", commune.Code)
		}
		names[site] = nameCode{name: commune.Name, code: commune.Code}
		communesBySite[site] = commune
		usedCodes[commune.Code] = true
		lieuSites = append(lieuSites, site)
	}
	if len(lieuSites) == 0 {
		return nil, fmt.Errorf("mapgen: cannot name territories without a lieu-dit")
	}

	adjacency := edgeMatrix(n, edges)
	usedPairs := make(map[string]bool)
	for site, isLieuDit := range lieuDits {
		if isLieuDit {
			continue
		}

		qualifiers := compatibleQualifiers(assets.Qualificatifs, terrain[site], isBorder(polygons[site], cfg))
		if len(qualifiers) == 0 {
			return nil, fmt.Errorf("mapgen: no qualifier available for terrain %q on territory %s", terrain[site], territoryID(site))
		}

		assigned := false
		for _, lieuSite := range orderedLieuDits(site, lieuSites, adjacency, centroids, n) {
			commune := communesBySite[lieuSite]
			for _, qualifier := range qualifiers {
				pair := qualifier.Prefix + "\x00" + commune.Code
				code := qualifier.Prefix + commune.Code
				if usedPairs[pair] || usedCodes[code] {
					continue
				}
				names[site] = nameCode{
					name: qualifier.Name + " de " + commune.Name,
					code: code,
				}
				usedPairs[pair] = true
				usedCodes[code] = true
				assigned = true
				break
			}
			if assigned {
				break
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

func orderedLieuDits(
	site int,
	lieuSites []int,
	adjacency []bool,
	centroids [][2]float64,
	n int,
) []int {
	adjacent := make([]int, 0)
	fallback := make([]int, 0)
	for _, lieuSite := range lieuSites {
		if matrixHasEdge(adjacency, n, site, lieuSite) {
			adjacent = append(adjacent, lieuSite)
		} else {
			fallback = append(fallback, lieuSite)
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
