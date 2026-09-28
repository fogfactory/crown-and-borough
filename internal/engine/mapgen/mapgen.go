// Package mapgen generates the static world map served by the map.json
// contract. It deliberately keeps geometry outside the business models package.
package mapgen

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	gridW = 256
	gridH = 160
	// TerritoriesPerPlayer is the fixed development map scale.
	TerritoriesPerPlayer = 8
	// TerritoriesPerSeat reserves four additional territories for each
	// chef-lieu (region seed): StartCount+1 of them in a game map.
	TerritoriesPerSeat = 4
)

// Config controls the raster-generation viewport and delivered population.
// Width and Height are not serialized: final dimensions are derived from the
// re-anchored interior polygons. SiteCount is the number of delivered interior
// territories; sacrificial frame sites exist only during raster generation.
// StartCount is the number of starting positions to place, each with its own
// dedicated home village. SeatCount is the number of chefs-lieux (region
// seeds) to place; a game map sets it to StartCount+1.
type Config struct {
	Width, Height int
	SiteCount     int
	StartCount    int
	SeatCount     int
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
	Name        string         `json:"name"`
	Terrain     models.Terrain `json:"terrain"`
	Village     bool           `json:"village"`
	Points      [][2]int       `json:"points"`
	Adjacencies []string       `json:"adjacencies"`
	Impassable  []string       `json:"impassable"`
}

// MapData is the complete static map document exposed by the development API.
// Starts holds the generated starting-territory IDs, in the deterministic
// order CreateGame assigns to players; it is internal to the engine and never
// published in map.json.
type MapData struct {
	Territories []Territory     `json:"territories"`
	Regions     []models.Region `json:"regions"`
	Starts      []string        `json:"-"`
}

// maxGenerateAttempts bounds the derived sub-seeds tried when a random draw
// yields a map that fails a structural constraint (degree caps, region
// balance, ...). Such draws are rare, so a handful of attempts suffices.
const maxGenerateAttempts = 16

// Generate builds a deterministic map from seed, assets and cfg. Randomness is
// isolated by phase so changes in one generation step do not perturb another.
// When the draw for seed violates a structural constraint, generation retries
// with derived sub-seeds; the first attempt uses seed itself, so maps that
// already generated successfully are unchanged.
func Generate(seed string, assets assetgen.Assets, cfg Config) (MapData, error) {
	if err := validateConfig(cfg); err != nil {
		return MapData{}, err
	}

	if err := validateAssets(assets, cfg.SiteCount); err != nil {
		return MapData{}, err
	}

	var err error
	for attempt := range maxGenerateAttempts {
		var data MapData
		data, err = generateAttempt(attemptSeed(seed, attempt), assets, cfg)
		if err == nil {
			return data, nil
		}
	}
	return MapData{}, fmt.Errorf("mapgen: no valid map after %d attempts: %w", maxGenerateAttempts, err)
}

// attemptSeed derives the seed of one generation attempt. Attempt zero keeps
// the caller's seed so the retry loop is invisible for successful draws.
func attemptSeed(seed string, attempt int) string {
	if attempt == 0 {
		return seed
	}
	return fmt.Sprintf("%s#retry-%d", seed, attempt)
}

func generateAttempt(seed string, assets assetgen.Assets, cfg Config) (MapData, error) {
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

	distances := siteDistances(passableEdges, cfg.SiteCount)
	starts, homeVillages, err := selectStarts(newRNG(seed, "starts"), distances, cfg.StartCount)
	if err != nil {
		return MapData{}, err
	}
	seats, err := placeSeats(newRNG(seed, "seats"), geometry.centroids, distances, starts, homeVillages, cfg.SeatCount)
	if err != nil {
		return MapData{}, err
	}

	names, err := nameTerritories(newRNG(seed, "naming"), assets, terrain)
	if err != nil {
		return MapData{}, err
	}

	territoryIDs := make([]string, len(names))
	for index, name := range names {
		territoryIDs[index] = name.code
	}
	adjacency := adjacencyIDs(passableEdges, territoryIDs)
	impassable := adjacencyIDs(impassableEdges, territoryIDs)

	village := make([]bool, cfg.SiteCount)
	for _, home := range homeVillages {
		village[home] = true
	}
	for _, seat := range seats {
		village[seat] = true
	}

	territories := make([]Territory, cfg.SiteCount)
	for i := range territories {
		territories[i] = Territory{
			ID:          names[i].code,
			Name:        names[i].name,
			Terrain:     terrain[i],
			Village:     village[i],
			Points:      geometry.polygons[i],
			Adjacencies: adjacency[i],
			Impassable:  impassable[i],
		}
	}

	seatIDs := make([]models.TerritoryID, len(seats))
	for index, seat := range seats {
		seatIDs[index] = models.TerritoryID(names[seat].code)
	}
	regions, err := generateRegions(territories, seatIDs)
	if err != nil {
		return MapData{}, err
	}

	startIDs := make([]string, len(starts))
	for index, start := range starts {
		startIDs[index] = names[start].code
	}
	sort.Strings(startIDs)

	return MapData{Territories: territories, Regions: regions, Starts: startIDs}, nil
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
	if cfg.StartCount < 1 {
		return fmt.Errorf("mapgen: start count must be at least 1")
	}
	if cfg.SeatCount < 1 {
		return fmt.Errorf("mapgen: seat count must be at least 1")
	}
	if needed := 2*cfg.StartCount + cfg.SeatCount; needed > cfg.SiteCount {
		return fmt.Errorf("mapgen: site count %d cannot fit %d starts, %d home villages and %d seats", cfg.SiteCount, cfg.StartCount, cfg.StartCount, cfg.SeatCount)
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
			return fmt.Errorf("mapgen: territory %d has fewer than three polygon points", i)
		}
		if polygonAreaTwice(points) <= 0 {
			return fmt.Errorf("mapgen: territory %d has a degenerate polygon", i)
		}
		if !isSimplePolygon(points) {
			return fmt.Errorf("mapgen: territory %d has a self-intersecting polygon", i)
		}
		for _, point := range points {
			if point[0] < padding || point[0] > width-padding || point[1] < padding || point[1] > height-padding {
				return fmt.Errorf("mapgen: territory %d has a polygon point outside the derived viewport", i)
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

func adjacencyIDs(edges [][2]int, ids []string) [][]string {
	n := len(ids)
	matrix := edgeMatrix(n, edges)
	adjacency := make([][]string, n)
	for i := 0; i < n; i++ {
		adjacency[i] = make([]string, 0)
		for j := 0; j < n; j++ {
			if matrixHasEdge(matrix, n, i, j) {
				adjacency[i] = append(adjacency[i], ids[j])
			}
		}
		sort.Strings(adjacency[i])
	}
	return adjacency
}

func siteLabel(index int) string {
	return fmt.Sprintf("site-%d", index+1)
}
