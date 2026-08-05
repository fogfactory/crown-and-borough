// Package mapgen generates the static world map served by the map.json
// contract. It deliberately keeps geometry outside the business models package.
package mapgen

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	gridW = 256
	gridH = 160
	// TerritoriesPerPlayer is the fixed development map scale.
	TerritoriesPerPlayer = 8
)

// Config controls the raster-generation viewport and delivered population.
// Width and Height are not serialized: final dimensions are derived from the
// re-anchored interior polygons. SiteCount is the number of delivered interior
// territories; sacrificial frame sites exist only during raster generation.
type Config struct {
	Width, Height int
	SiteCount     int
	VillageCount  int
}

// Territory is the static map representation of a territory. Geometry belongs
// here rather than in models.Territory, which represents the game domain.
// Village is STATIC SEED DATA: game creation (NewGame, P1.6/P1.7, P1.2f)
// materializes each flag as an ownerless village Infrastructure on the
// uncontrolled territory. The flag never changes: a tile stays "village" in
// map.json even when a castle replaces the village (the state layer is
// authoritative for the real situation).
type Territory struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Terrain     models.Terrain `json:"terrain"`
	Village     bool           `json:"village"`
	Points      [][2]int       `json:"points"`
	Adjacencies []string       `json:"adjacencies"`
	Impassable  []string       `json:"impassable"`
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

	if err := validateAssets(assets, cfg.SiteCount); err != nil {
		return MapData{}, err
	}

	sites := generateSites(newRNG(seed, "sites"), cfg)
	grid := assignRaster(sites, cfg)
	if !rasterHasEveryRegion(grid, cfg.SiteCount) {
		return MapData{}, fmt.Errorf("mapgen: raster left at least one interior site without a region")
	}

	geometry, err := extractInteriorFrontiers(grid, len(sites), cfg.SiteCount, cfg, seed)
	if err != nil {
		return MapData{}, err
	}
	if err := validateGeometry(geometry.polygons, geometry.padding); err != nil {
		return MapData{}, err
	}

	geometricEdges := extractAdjacency(grid, cfg.SiteCount)
	terrain := assignTerrains(newRNG(seed, "terrain"), sites[:cfg.SiteCount])
	passableEdges, impassableEdges := pruneFrontiers(newRNG(seed, "frontiers"), geometricEdges, terrain)
	passableEdges, impassableEdges, err = enforceDegreeCaps(passableEdges, impassableEdges, terrain, geometry.centroids)
	if err != nil {
		return MapData{}, err
	}
	if err := validateGraph(passableEdges, terrain, cfg.SiteCount); err != nil {
		return MapData{}, err
	}

	villages, err := assignVillages(newRNG(seed, "village"), terrain, geometry.centroids, cfg.VillageCount)
	if err != nil {
		return MapData{}, err
	}
	names, err := nameTerritories(newRNG(seed, "naming"), assets, terrain)
	if err != nil {
		return MapData{}, err
	}

	adjacency := adjacencyIDs(passableEdges, cfg.SiteCount)
	impassable := adjacencyIDs(impassableEdges, cfg.SiteCount)
	territories := make([]Territory, cfg.SiteCount)
	for i := range territories {
		territories[i] = Territory{
			ID:          territoryID(i),
			Code:        names[i].code,
			Name:        names[i].name,
			Terrain:     terrain[i],
			Village:     villages[i],
			Points:      geometry.polygons[i],
			Adjacencies: adjacency[i],
			Impassable:  impassable[i],
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
	if cfg.VillageCount < 1 {
		return fmt.Errorf("mapgen: village count must be at least 1")
	}
	if cfg.VillageCount > cfg.SiteCount {
		return fmt.Errorf("mapgen: village count %d exceeds site count %d", cfg.VillageCount, cfg.SiteCount)
	}
	return nil
}

func validateGeometry(polygons [][][2]int, padding int) error {
	maxX, maxY := padding, padding
	for _, points := range polygons {
		for _, point := range points {
			if point[0] > maxX {
				maxX = point[0]
			}
			if point[1] > maxY {
				maxY = point[1]
			}
		}
	}
	width := maxX + padding
	height := maxY + padding
	for i, points := range polygons {
		if len(points) < 3 {
			return fmt.Errorf("mapgen: territory %s has fewer than three polygon points", territoryID(i))
		}
		if polygonAreaTwice(points) <= 0 {
			return fmt.Errorf("mapgen: territory %s has a degenerate polygon", territoryID(i))
		}
		if !isSimplePolygon(points) {
			return fmt.Errorf("mapgen: territory %s has a self-intersecting polygon", territoryID(i))
		}
		for _, point := range points {
			if point[0] < padding || point[0] > width-padding || point[1] < padding || point[1] > height-padding {
				return fmt.Errorf("mapgen: territory %s has a polygon point outside the derived viewport", territoryID(i))
			}
		}
	}
	return nil
}

func isSimplePolygon(points [][2]int) bool {
	if len(points) < 3 {
		return false
	}
	for first := range points {
		firstNext := (first + 1) % len(points)
		if points[first] == points[firstNext] {
			return false
		}
		for second := first + 1; second < len(points); second++ {
			secondNext := (second + 1) % len(points)
			if firstNext == second || secondNext == first {
				continue
			}
			if segmentsIntersect(points[first], points[firstNext], points[second], points[secondNext]) {
				return false
			}
		}
	}
	return true
}

func segmentsIntersect(firstStart, firstEnd, secondStart, secondEnd [2]int) bool {
	first := orientation(firstStart, firstEnd, secondStart)
	second := orientation(firstStart, firstEnd, secondEnd)
	third := orientation(secondStart, secondEnd, firstStart)
	fourth := orientation(secondStart, secondEnd, firstEnd)
	if first == 0 && pointOnSegment(firstStart, firstEnd, secondStart) {
		return true
	}
	if second == 0 && pointOnSegment(firstStart, firstEnd, secondEnd) {
		return true
	}
	if third == 0 && pointOnSegment(secondStart, secondEnd, firstStart) {
		return true
	}
	if fourth == 0 && pointOnSegment(secondStart, secondEnd, firstEnd) {
		return true
	}
	return (first > 0) != (second > 0) && (third > 0) != (fourth > 0)
}

func orientation(first, second, third [2]int) int64 {
	return int64(second[0]-first[0])*int64(third[1]-first[1]) -
		int64(second[1]-first[1])*int64(third[0]-first[0])
}

func pointOnSegment(first, second, point [2]int) bool {
	return point[0] >= min(first[0], second[0]) && point[0] <= max(first[0], second[0]) &&
		point[1] >= min(first[1], second[1]) && point[1] <= max(first[1], second[1])
}

func territoryID(index int) string {
	return fmt.Sprintf("T%02d", index+1)
}

func adjacencyIDs(edges [][2]int, n int) [][]string {
	matrix := edgeMatrix(n, edges)
	adjacency := make([][]string, n)
	for i := 0; i < n; i++ {
		adjacency[i] = make([]string, 0)
		for j := 0; j < n; j++ {
			if matrixHasEdge(matrix, n, i, j) {
				adjacency[i] = append(adjacency[i], territoryID(j))
			}
		}
	}
	return adjacency
}
