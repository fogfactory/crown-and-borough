// Package mapgen generates the static world map served by the map.json
// contract. It deliberately keeps geometry outside the business models package.
package mapgen

import (
	"fmt"
	"math"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	gridW = 256
	gridH = 160
)

// Config controls the generated map viewport and population.
type Config struct {
	Width, Height int
	SiteCount     int
	LieuDitRatio  float64
}

// Territory is the static map representation of a territory. Geometry belongs
// here rather than in models.Territory, which represents the game domain.
type Territory struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Terrain     models.Terrain `json:"terrain"`
	LieuDit     bool           `json:"lieuDit"`
	Points      [][2]int       `json:"points"`
	Adjacencies []string       `json:"adjacencies"`
}

// MapData is the complete static map document exposed by the development API.
type MapData struct {
	Territories []Territory `json:"territories"`
}

// Generate builds a deterministic map from seed, assets and cfg. Randomness is
// isolated by phase so changes in one generation step do not perturb another.
func Generate(seed string, assets assetgen.Assets, cfg Config) (MapData, error) {
	if err := validateConfig(cfg); err != nil {
		return MapData{}, err
	}

	lieuDitCount := int(math.Round(float64(cfg.SiteCount) * cfg.LieuDitRatio))
	if lieuDitCount == 0 {
		return MapData{}, fmt.Errorf("mapgen: configuration produces no lieu-dits")
	}
	if err := validateAssets(assets, lieuDitCount); err != nil {
		return MapData{}, err
	}

	sites := generateSites(newRNG(seed, "sites"), cfg)
	grid := assignRaster(sites, cfg)
	if !rasterHasEveryRegion(grid, len(sites)) {
		return MapData{}, fmt.Errorf("mapgen: raster left at least one site without a region")
	}

	polygons, centroids := extractPolygons(grid, sites, cfg)
	if err := validateGeometry(polygons, cfg); err != nil {
		return MapData{}, err
	}

	geometricEdges := extractAdjacency(grid)
	terrain := assignTerrains(newRNG(seed, "terrain"), sites)
	passableEdges := filterFrontiers(newRNG(seed, "frontiers"), geometricEdges, terrain)
	finalEdges := repairGraphWithGeometry(passableEdges, geometricEdges, centroids, len(sites))
	if err := validateGraph(finalEdges, len(sites)); err != nil {
		return MapData{}, err
	}

	lieuDits := assignLieuDits(newRNG(seed, "lieudit"), len(sites), cfg.LieuDitRatio)
	names, err := nameTerritories(
		newRNG(seed, "naming"),
		assets,
		lieuDits,
		terrain,
		polygons,
		centroids,
		finalEdges,
		cfg,
	)
	if err != nil {
		return MapData{}, err
	}

	adjacency := adjacencyIDs(finalEdges, len(sites))
	territories := make([]Territory, len(sites))
	for i := range territories {
		territories[i] = Territory{
			ID:          territoryID(i),
			Code:        names[i].code,
			Name:        names[i].name,
			Terrain:     terrain[i],
			LieuDit:     lieuDits[i],
			Points:      polygons[i],
			Adjacencies: adjacency[i],
		}
	}

	return MapData{Territories: territories}, nil
}

func validateConfig(cfg Config) error {
	if cfg.Width < 100 {
		return fmt.Errorf("mapgen: width must be at least 100")
	}
	if cfg.Height < 100 {
		return fmt.Errorf("mapgen: height must be at least 100")
	}
	if cfg.SiteCount < 8 {
		return fmt.Errorf("mapgen: site count must be at least 8")
	}
	if cfg.SiteCount > gridW*gridH {
		return fmt.Errorf("mapgen: site count must not exceed raster capacity %d", gridW*gridH)
	}
	if !(cfg.LieuDitRatio > 0 && cfg.LieuDitRatio <= 0.5) {
		return fmt.Errorf("mapgen: lieu-dit ratio must be in (0, 0.5]")
	}
	return nil
}

func validateGeometry(polygons [][][2]int, cfg Config) error {
	for i, points := range polygons {
		if len(points) < 3 {
			return fmt.Errorf("mapgen: territory %s has fewer than three polygon points", territoryID(i))
		}
		if polygonAreaTwice(points) <= 0 {
			return fmt.Errorf("mapgen: territory %s has a degenerate polygon", territoryID(i))
		}
		for _, point := range points {
			if point[0] < 0 || point[0] > cfg.Width || point[1] < 0 || point[1] > cfg.Height {
				return fmt.Errorf("mapgen: territory %s has a polygon point outside the viewport", territoryID(i))
			}
		}
	}
	return nil
}

func territoryID(index int) string {
	return fmt.Sprintf("T%02d", index+1)
}

func adjacencyIDs(edges [][2]int, n int) [][]string {
	matrix := edgeMatrix(n, edges)
	adjacency := make([][]string, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if matrixHasEdge(matrix, n, i, j) {
				adjacency[i] = append(adjacency[i], territoryID(j))
			}
		}
	}
	return adjacency
}
