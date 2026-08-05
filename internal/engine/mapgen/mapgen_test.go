package mapgen

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

var testConfig = Config{
	Width:        1000,
	Height:       700,
	SiteCount:    TerritoriesPerPlayer * 4,
	VillageCount: 5,
}

var minimumTestConfig = Config{
	Width:        100,
	Height:       100,
	SiteCount:    8,
	VillageCount: 1,
}

var testSeeds = []string{
	"crown-and-borough-dev",
	"alpha",
	"beta",
	"gamma",
	"delta",
	"epsilon",
	"zeta",
	"eta",
	"theta",
	"iota",
	"kappa",
	"lambda",
	"mu",
	"nu",
	"xi",
	"omicron",
	"pi",
	"rho",
	"sigma",
	"tau",
}

func TestDeterminism(t *testing.T) {
	if testConfig.SiteCount != 32 || testConfig.VillageCount != 5 {
		t.Fatalf("test config = %+v, want 32 territories and 5 villages", testConfig)
	}

	assets := loadTestAssets(t)
	first := generateTestMap(t, "crown-and-borough-dev", assets)
	second := generateTestMap(t, "crown-and-borough-dev", assets)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("Generate returned different map data for the same seed")
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first map: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second map: %v", err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("JSON output differs for the same seed")
	}
	if bytes.Contains(firstJSON, []byte(`"impassable":null`)) {
		t.Fatal("map JSON encodes an impassable list as null")
	}
	for _, territory := range first.Territories {
		if territory.Impassable == nil {
			t.Errorf("%s has a nil impassable list", territory.ID)
		}
	}

	different := generateTestMap(t, "beta", assets)
	differentJSON, err := json.Marshal(different)
	if err != nil {
		t.Fatalf("marshal different map: %v", err)
	}
	if bytes.Equal(firstJSON, differentJSON) {
		t.Fatal("different seeds returned identical JSON")
	}
}

func TestNoBorderPadding(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		assertPaddedGeometry(t, data, testConfig)
	})
}

func TestFrameAbsorbsBorder(t *testing.T) {
	configs := []struct {
		name string
		cfg  Config
	}{
		{name: "players-2", cfg: playerTestConfig(2)},
		{name: "players-3", cfg: playerTestConfig(3)},
		{name: "players-4", cfg: playerTestConfig(4)},
		{name: "players-5", cfg: playerTestConfig(5)},
		{name: "minimum-100x100", cfg: minimumTestConfig},
	}

	for _, test := range configs {
		test := test
		t.Run(test.name, func(t *testing.T) {
			forEachTestSeed(t, func(t *testing.T, seed string) {
				sites, grid := generateTestRaster(seed, test.cfg)
				if len(sites) <= test.cfg.SiteCount {
					t.Fatalf("generateSites returned %d sites, want interior sites plus a frame", len(sites))
				}
				if !rasterHasEveryRegion(grid, test.cfg.SiteCount) {
					t.Fatal("raster has an unassigned interior region")
				}
				assertFrameAbsorbsBorder(t, grid, test.cfg.SiteCount, len(sites))
			})
		})
	}
}

func TestSharedFrontierCoherence(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		geometry := extractTestFrontiers(t, seed, testConfig)
		if len(geometry.polygons) != len(data.Territories) {
			t.Fatalf("frontier polygon count = %d, want %d", len(geometry.polygons), len(data.Territories))
		}
		for index, polygon := range geometry.polygons {
			if !reflect.DeepEqual(polygon, data.Territories[index].Points) {
				t.Errorf("%s points differ from extracted frontier geometry", data.Territories[index].ID)
			}
		}

		assertSharedFrontierCoherence(t, data, geometry, collectDTOArcs(t, data).union)
	})
}

func TestNoStaircase(t *testing.T) {
	if minimumChainSegment != 2 {
		t.Fatalf("minimum chain segment = %v, want 2 grid cells", minimumChainSegment)
	}

	forEachTestSeed(t, func(t *testing.T, seed string) {
		geometry := extractTestFrontiers(t, seed, testConfig)
		for chainIndex, chain := range geometry.chains {
			if len(chain.grid) != len(chain.points) || len(chain.grid) != len(chain.junctions) {
				t.Errorf("chain %d metadata lengths are grid=%d points=%d junctions=%d", chainIndex, len(chain.grid), len(chain.points), len(chain.junctions))
				continue
			}
			if len(chain.raw) < len(chain.grid) {
				t.Errorf("chain %d has %d simplified points from %d raw points", chainIndex, len(chain.grid), len(chain.raw))
			}
			for pointIndex := 1; pointIndex < len(chain.grid); pointIndex++ {
				if gridPointDistance(chain.grid[pointIndex-1], chain.grid[pointIndex]) < minimumChainSegment &&
					!chain.junctions[pointIndex-1] && !chain.junctions[pointIndex] {
					t.Errorf("chain %d retains a %.2f-cell staircase segment between %v and %v", chainIndex, gridPointDistance(chain.grid[pointIndex-1], chain.grid[pointIndex]), chain.grid[pointIndex-1], chain.grid[pointIndex])
				}
			}
		}
	})
}

func TestAllArcsGeometric(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		_, grid := generateTestRaster(seed, testConfig)
		want := extractAdjacency(grid, testConfig.SiteCount)
		got := collectDTOArcs(t, data)
		if !reflect.DeepEqual(got.union, want) {
			t.Errorf("DTO arc union = %v, want raster adjacency %v", got.union, want)
		}
	})
}

func TestGraphInvariants(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		assertGraphInvariants(t, generateTestMap(t, seed, assets))
	})
}

func TestVillageCount(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		count := 0
		for _, territory := range data.Territories {
			if territory.Village {
				count++
			}
		}
		if count != testConfig.VillageCount {
			t.Fatalf("village count = %d, want %d", count, testConfig.VillageCount)
		}
	})
}

func TestVillageSpread(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		bounds, ok := derivedBounds(data)
		if !ok {
			t.Fatal("map has no polygon points")
		}

		// Villages are spread by greedy max-min placement over the delivered,
		// re-anchored map rather than the raster-generation viewport.
		minDistance := 0.15 * math.Hypot(float64(bounds.width), float64(bounds.height))
		villages := make([]int, 0, testConfig.VillageCount)
		for index, territory := range data.Territories {
			if territory.Village {
				villages = append(villages, index)
			}
		}
		for firstIndex, first := range villages {
			for _, second := range villages[firstIndex+1:] {
				firstCentroid := polygonCentroid(data.Territories[first].Points)
				secondCentroid := polygonCentroid(data.Territories[second].Points)
				actual := math.Hypot(firstCentroid[0]-secondCentroid[0], firstCentroid[1]-secondCentroid[1])
				if actual < minDistance {
					t.Errorf("villages %s and %s are %v apart, want >= %v", data.Territories[first].ID, data.Territories[second].ID, actual, minDistance)
				}
			}
		}
	})
}

func TestCodes(t *testing.T) {
	assets := loadTestAssets(t)
	territoryCode := regexp.MustCompile(`^[A-Z]{3}$`)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		seen := make(map[string]bool, len(data.Territories))
		for _, territory := range data.Territories {
			if !territoryCode.MatchString(territory.Code) {
				t.Errorf("%s has invalid code %q", territory.ID, territory.Code)
			}
			if seen[territory.Code] {
				t.Errorf("duplicate code %q", territory.Code)
			}
			seen[territory.Code] = true
		}
	})
}

func TestNaming(t *testing.T) {
	assets := loadTestAssets(t)
	communesByName := make(map[string]assetgen.Commune, len(assets.Communes))
	for _, commune := range assets.Communes {
		communesByName[commune.Name] = commune
	}
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		seenTerrain := make(map[models.Terrain]bool, len(allTerrains))
		seenNames := make(map[string]bool, len(data.Territories))
		for _, territory := range data.Territories {
			if territory.Name == "" {
				t.Errorf("%s has an empty name", territory.ID)
			}
			seenTerrain[territory.Terrain] = true
			commune, ok := communesByName[territory.Name]
			if !ok {
				t.Errorf("%s has name %q outside communes.csv", territory.ID, territory.Name)
				continue
			}
			if territory.Code != commune.Code {
				t.Errorf("%s has code %q, want commune code %q", territory.ID, territory.Code, commune.Code)
			}
			if seenNames[territory.Name] {
				t.Errorf("duplicate territory name %q", territory.Name)
			}
			seenNames[territory.Name] = true
		}
		for _, terrain := range allTerrains {
			if !seenTerrain[terrain] {
				t.Errorf("terrain %q is absent", terrain)
			}
		}
	})
}

func TestNamingAffinity(t *testing.T) {
	assets := loadTestAssets(t)
	communesByName := make(map[string]assetgen.Commune, len(assets.Communes))
	for _, commune := range assets.Communes {
		communesByName[commune.Name] = commune
	}
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateTestMap(t, seed, assets)
		seenTerrain := make(map[models.Terrain]bool, len(allTerrains))
		usedAffinity := make(map[models.Terrain]bool, len(allTerrains))
		affine := 0
		for _, territory := range data.Territories {
			commune, ok := communesByName[territory.Name]
			if !ok {
				t.Errorf("%s has name %q outside communes.csv", territory.ID, territory.Name)
				continue
			}
			seenTerrain[territory.Terrain] = true
			if commune.Terrain == string(territory.Terrain) || commune.Terrain == "any" {
				affine++
			}
			if commune.Terrain == string(territory.Terrain) {
				usedAffinity[territory.Terrain] = true
			}
		}
		ratio := float64(affine) / float64(len(data.Territories))
		if ratio < 0.90 {
			t.Errorf("affinity ratio = %.2f, want >= 0.90", ratio)
		}
		for _, terrain := range allTerrains {
			if seenTerrain[terrain] && !usedAffinity[terrain] {
				t.Errorf("no %q-affinity commune used for present terrain", terrain)
			}
		}
	})
}

func TestNamingCollisions(t *testing.T) {
	assets := assetgen.Assets{
		Communes: []assetgen.Commune{
			{Code: "AAP", Name: "Aubepine", Terrain: "plain"},
			{Code: "BEP", Name: "Beaupre", Terrain: "plain"},
			{Code: "BOF", Name: "Boisfort", Terrain: "forest"},
			{Code: "BRH", Name: "Bruyeres", Terrain: "hill"},
			{Code: "MOM", Name: "Montdore", Terrain: "mountain"},
			{Code: "FOS", Name: "Fougeres", Terrain: "swamp"},
			{Code: "ANY", Name: "Belval", Terrain: "any"},
		},
	}
	terrain := []models.Terrain{
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainForest,
	}

	first, err := nameTerritories(newRNG("collision", "naming"), assets, terrain)
	if err != nil {
		t.Fatalf("first naming pass: %v", err)
	}
	second, err := nameTerritories(newRNG("collision", "naming"), assets, terrain)
	if err != nil {
		t.Fatalf("second naming pass: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("naming collisions are not deterministic")
	}

	seenNames := make(map[string]bool)
	seenCodes := make(map[string]bool)
	for _, name := range first {
		if seenNames[name.name] || seenCodes[name.code] {
			t.Fatalf("duplicate collision result for %q / %q", name.name, name.code)
		}
		seenNames[name.name] = true
		seenCodes[name.code] = true
	}
	for _, code := range []string{"BRH", "MOM", "FOS"} {
		if !seenCodes[code] {
			t.Fatalf("naming did not use fallback commune %q", code)
		}
	}

	exhaustedTerrain := append(append([]models.Terrain(nil), terrain...), models.TerrainPlain)
	if _, err := nameTerritories(newRNG("collision", "naming"), assets, exhaustedTerrain); err == nil {
		t.Fatal("naming with more sites than communes returned no error")
	}
}

func TestConfigValidation(t *testing.T) {
	assets := loadTestAssets(t)
	invalid := []Config{
		{Width: 99, Height: 700, SiteCount: 32, VillageCount: 5},
		{Width: 1000, Height: 99, SiteCount: 32, VillageCount: 5},
		{Width: 1000, Height: 700, SiteCount: 7, VillageCount: 5},
		{Width: 1000, Height: 700, SiteCount: gridW*gridH + 1, VillageCount: 1},
		{Width: 1000, Height: 700, SiteCount: 32, VillageCount: 0},
		{Width: 1000, Height: 700, SiteCount: 32, VillageCount: -1},
		{Width: 1000, Height: 700, SiteCount: 32, VillageCount: 33},
	}
	for _, cfg := range invalid {
		if _, err := Generate("invalid", assets, cfg); err == nil {
			t.Errorf("Generate(%+v) returned no validation error", cfg)
		}
	}
	if _, err := Generate("assets", assetgen.Assets{}, testConfig); err == nil {
		t.Error("Generate with empty assets returned no error")
	}
	if _, err := Generate("assets", assetgen.Assets{Communes: assets.Communes[:testConfig.SiteCount-1]}, testConfig); err == nil {
		t.Error("Generate with too few communes returned no error")
	}
}

func TestMinimumConfig(t *testing.T) {
	assets := loadTestAssets(t)
	forEachTestSeed(t, func(t *testing.T, seed string) {
		data := generateMap(t, seed, assets, minimumTestConfig)
		if len(data.Territories) != minimumTestConfig.SiteCount {
			t.Errorf("territory count = %d, want %d", len(data.Territories), minimumTestConfig.SiteCount)
		}
		count := 0
		for _, territory := range data.Territories {
			if territory.Village {
				count++
			}
		}
		if count != minimumTestConfig.VillageCount {
			t.Errorf("village count = %d, want %d", count, minimumTestConfig.VillageCount)
		}
		assertPaddedGeometry(t, data, minimumTestConfig)
	})
}

func loadTestAssets(t *testing.T) assetgen.Assets {
	t.Helper()
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load test assets: %v", err)
	}
	return assets
}

func generateTestMap(t *testing.T, seed string, assets assetgen.Assets) MapData {
	t.Helper()
	return generateMap(t, seed, assets, testConfig)
}

func generateMap(t *testing.T, seed string, assets assetgen.Assets, cfg Config) MapData {
	t.Helper()
	data, err := Generate(seed, assets, cfg)
	if err != nil {
		t.Fatalf("Generate(%q): %v", seed, err)
	}
	return data
}

func playerTestConfig(players int) Config {
	return Config{
		Width:        testConfig.Width,
		Height:       testConfig.Height,
		SiteCount:    TerritoriesPerPlayer * players,
		VillageCount: players + 1,
	}
}

func forEachTestSeed(t *testing.T, run func(t *testing.T, seed string)) {
	t.Helper()
	if len(testSeeds) < 20 {
		t.Fatalf("test seed count = %d, want at least 20", len(testSeeds))
	}
	seen := make(map[string]bool, len(testSeeds))
	hasDevSeed := false
	for _, seed := range testSeeds {
		if seen[seed] {
			t.Fatalf("duplicate test seed %q", seed)
		}
		seen[seed] = true
		if seed == "crown-and-borough-dev" {
			hasDevSeed = true
		}
	}
	if !hasDevSeed {
		t.Fatal("test seeds do not include crown-and-borough-dev")
	}
	for _, seed := range testSeeds {
		seed := seed
		t.Run(seed, func(t *testing.T) {
			run(t, seed)
		})
	}
}

func generateTestRaster(seed string, cfg Config) ([]point, []int) {
	sites := generateSites(newRNG(seed, "sites"), cfg)
	return sites, assignRaster(sites, cfg)
}

func extractTestFrontiers(t *testing.T, seed string, cfg Config) frontierGeometry {
	t.Helper()
	sites, grid := generateTestRaster(seed, cfg)
	geometry, err := extractInteriorFrontiers(grid, len(sites), cfg.SiteCount, cfg, seed)
	if err != nil {
		t.Fatalf("extract interior frontiers for %q: %v", seed, err)
	}
	return geometry
}

func assertFrameAbsorbsBorder(t *testing.T, grid []int, interiorCount, siteCount int) {
	t.Helper()
	if len(grid) != gridW*gridH {
		t.Fatalf("raster length = %d, want %d", len(grid), gridW*gridH)
	}
	for x := 0; x < gridW; x++ {
		assertFrameOwner(t, grid[x], x, 0, interiorCount, siteCount)
		assertFrameOwner(t, grid[(gridH-1)*gridW+x], x, gridH-1, interiorCount, siteCount)
	}
	for y := 0; y < gridH; y++ {
		assertFrameOwner(t, grid[y*gridW], 0, y, interiorCount, siteCount)
		assertFrameOwner(t, grid[y*gridW+gridW-1], gridW-1, y, interiorCount, siteCount)
	}
}

func assertFrameOwner(t *testing.T, owner, x, y, interiorCount, siteCount int) {
	t.Helper()
	if owner < interiorCount || owner >= siteCount {
		t.Fatalf("border raster cell (%d, %d) belongs to site %d, want a frame site in [%d, %d)", x, y, owner, interiorCount, siteCount)
	}
}

type mapBounds struct {
	minX, minY    int
	maxX, maxY    int
	width, height int
}

func derivedBounds(data MapData) (mapBounds, bool) {
	bounds := mapBounds{}
	set := false
	for _, territory := range data.Territories {
		for _, point := range territory.Points {
			if !set {
				bounds.minX, bounds.maxX = point[0], point[0]
				bounds.minY, bounds.maxY = point[1], point[1]
				set = true
				continue
			}
			if point[0] < bounds.minX {
				bounds.minX = point[0]
			}
			if point[0] > bounds.maxX {
				bounds.maxX = point[0]
			}
			if point[1] < bounds.minY {
				bounds.minY = point[1]
			}
			if point[1] > bounds.maxY {
				bounds.maxY = point[1]
			}
		}
	}
	if !set {
		return mapBounds{}, false
	}
	bounds.width = bounds.maxX + bounds.minX
	bounds.height = bounds.maxY + bounds.minY
	return bounds, true
}

func assertPaddedGeometry(t *testing.T, data MapData, cfg Config) {
	t.Helper()
	if len(data.Territories) != cfg.SiteCount {
		t.Errorf("territory count = %d, want %d", len(data.Territories), cfg.SiteCount)
	}
	bounds, ok := derivedBounds(data)
	if !ok {
		t.Fatal("map has no polygon points")
	}
	padding := mapPadding(cfg)
	if bounds.minX != padding || bounds.minY != padding {
		t.Errorf("polygon minimum = (%d, %d), want padding (%d, %d)", bounds.minX, bounds.minY, padding, padding)
	}
	for _, territory := range data.Territories {
		if len(territory.Points) < 3 {
			t.Errorf("%s has %d points, want at least 3", territory.ID, len(territory.Points))
			continue
		}
		if polygonAreaTwice(territory.Points) <= 0 {
			t.Errorf("%s has non-positive polygon area", territory.ID)
		}
		for _, point := range territory.Points {
			if point[0] < padding || point[0] > bounds.width-padding || point[1] < padding || point[1] > bounds.height-padding {
				t.Errorf("%s point %v is outside the derived padded viewport %dx%d", territory.ID, point, bounds.width, bounds.height)
			}
		}
	}
}

type dtoArcs struct {
	passable   [][2]int
	impassable [][2]int
	union      [][2]int
}

func collectDTOArcs(t *testing.T, data MapData) dtoArcs {
	t.Helper()
	indexes := make(map[string]int, len(data.Territories))
	for index, territory := range data.Territories {
		if territory.ID != territoryID(index) {
			t.Errorf("territory at index %d has ID %q, want %q", index, territory.ID, territoryID(index))
		}
		if _, exists := indexes[territory.ID]; exists {
			t.Errorf("duplicate territory ID %q", territory.ID)
		}
		indexes[territory.ID] = index
	}

	passableCounts := make(map[[2]int]int)
	impassableCounts := make(map[[2]int]int)
	for owner, territory := range data.Territories {
		if territory.Impassable == nil {
			t.Errorf("%s has a nil impassable list", territory.ID)
		}
		collect := func(field string, values []string, counts map[[2]int]int) {
			for valueIndex, adjacentID := range values {
				if valueIndex > 0 && values[valueIndex-1] >= adjacentID {
					t.Errorf("%s %s are not strictly sorted", territory.ID, field)
				}
				adjacent, exists := indexes[adjacentID]
				if !exists {
					t.Errorf("%s %s references unknown territory %s", territory.ID, field, adjacentID)
					continue
				}
				if adjacent == owner {
					t.Errorf("%s has a self %s", territory.ID, field)
					continue
				}
				first, second := orderedPair(owner, adjacent)
				counts[[2]int{first, second}]++
			}
		}
		collect("adjacencies", territory.Adjacencies, passableCounts)
		collect("impassable", territory.Impassable, impassableCounts)
	}

	var arcs dtoArcs
	for first := 0; first < len(data.Territories); first++ {
		for second := first + 1; second < len(data.Territories); second++ {
			edge := [2]int{first, second}
			passableCount := passableCounts[edge]
			impassableCount := impassableCounts[edge]
			if passableCount > 0 && impassableCount > 0 {
				t.Errorf("%s/%s is classified as both passable and impassable", data.Territories[first].ID, data.Territories[second].ID)
			}
			if passableCount > 0 {
				assertArcSymmetry(t, data, edge, false, passableCount)
				arcs.passable = append(arcs.passable, edge)
			}
			if impassableCount > 0 {
				assertArcSymmetry(t, data, edge, true, impassableCount)
				arcs.impassable = append(arcs.impassable, edge)
			}
			if passableCount > 0 || impassableCount > 0 {
				arcs.union = append(arcs.union, edge)
			}
		}
	}
	return arcs
}

func assertArcSymmetry(t *testing.T, data MapData, edge [2]int, impassable bool, count int) {
	t.Helper()
	classification := "adjacency"
	firstValues := data.Territories[edge[0]].Adjacencies
	secondValues := data.Territories[edge[1]].Adjacencies
	if impassable {
		classification = "impassable arc"
		firstValues = data.Territories[edge[0]].Impassable
		secondValues = data.Territories[edge[1]].Impassable
	}
	if count != 2 {
		t.Errorf("%s between %s and %s appears %d times, want twice", classification, data.Territories[edge[0]].ID, data.Territories[edge[1]].ID, count)
	}
	if !containsID(firstValues, data.Territories[edge[1]].ID) || !containsID(secondValues, data.Territories[edge[0]].ID) {
		t.Errorf("%s between %s and %s is not symmetric", classification, data.Territories[edge[0]].ID, data.Territories[edge[1]].ID)
	}
}

func assertSharedFrontierCoherence(t *testing.T, data MapData, geometry frontierGeometry, arcs [][2]int) {
	t.Helper()
	deltaX, deltaY, ok := frontierReanchorOffset(geometry)
	if !ok {
		t.Fatal("frontier geometry has no chain points")
	}
	for _, edge := range arcs {
		if !hasSharedPolygonEdge(data.Territories[edge[0]].Points, data.Territories[edge[1]].Points) {
			t.Errorf("geometric arc %s/%s has no exact shared polygon edge", data.Territories[edge[0]].ID, data.Territories[edge[1]].ID)
		}
		qualifyingChainCount := 0
		for chainIndex, chain := range geometry.chains {
			if chain.owners != edge || len(chain.raw)-1 < minSharedEdges {
				continue
			}
			qualifyingChainCount++
			points := translatedChainPoints(chain, deltaX, deltaY)
			if len(points) < 2 {
				t.Errorf("chain %d for %s/%s has %d distinct points", chainIndex, data.Territories[edge[0]].ID, data.Territories[edge[1]].ID, len(points))
				continue
			}
			if !hasCyclicPath(data.Territories[edge[0]].Points, points) {
				t.Errorf("chain %d for %s/%s is absent from %s", chainIndex, data.Territories[edge[0]].ID, data.Territories[edge[1]].ID, data.Territories[edge[0]].ID)
			}
			if !hasCyclicPath(data.Territories[edge[1]].Points, points) {
				t.Errorf("chain %d for %s/%s is absent from %s", chainIndex, data.Territories[edge[0]].ID, data.Territories[edge[1]].ID, data.Territories[edge[1]].ID)
			}
		}
		if qualifyingChainCount == 0 {
			t.Errorf("geometric arc %s/%s has no frontier chain with at least %d raster edges", data.Territories[edge[0]].ID, data.Territories[edge[1]].ID, minSharedEdges)
		}
	}
}

func frontierReanchorOffset(geometry frontierGeometry) (int, int, bool) {
	set := false
	minX, minY := 0, 0
	for _, chain := range geometry.chains {
		for _, point := range chain.points {
			if !set {
				minX, minY = point[0], point[1]
				set = true
				continue
			}
			if point[0] < minX {
				minX = point[0]
			}
			if point[1] < minY {
				minY = point[1]
			}
		}
	}
	if !set {
		return 0, 0, false
	}
	return geometry.padding - minX, geometry.padding - minY, true
}

func translatedChainPoints(chain frontierChain, deltaX, deltaY int) [][2]int {
	points := make([][2]int, len(chain.points))
	for index, point := range chain.points {
		points[index] = [2]int{point[0] + deltaX, point[1] + deltaY}
	}
	return removeConsecutiveDuplicates(points)
}

func hasCyclicPath(polygon, path [][2]int) bool {
	if len(path) < 2 || len(path) > len(polygon) {
		return false
	}
	for start, point := range polygon {
		if point != path[0] {
			continue
		}
		forward, reverse := true, true
		for offset := 1; offset < len(path); offset++ {
			if polygon[(start+offset)%len(polygon)] != path[offset] {
				forward = false
			}
			if polygon[(start-offset+len(polygon))%len(polygon)] != path[offset] {
				reverse = false
			}
		}
		if forward || reverse {
			return true
		}
	}
	return false
}

func hasSharedPolygonEdge(first, second [][2]int) bool {
	for firstIndex, firstPoint := range first {
		firstNext := first[(firstIndex+1)%len(first)]
		for secondIndex, secondPoint := range second {
			secondNext := second[(secondIndex+1)%len(second)]
			if (firstPoint == secondPoint && firstNext == secondNext) ||
				(firstPoint == secondNext && firstNext == secondPoint) {
				return true
			}
		}
	}
	return false
}

func assertGraphInvariants(t *testing.T, data MapData) {
	t.Helper()
	if len(data.Territories) == 0 {
		t.Fatal("map has no territories")
	}
	indexes := make(map[string]int, len(data.Territories))
	for index, territory := range data.Territories {
		if _, exists := indexes[territory.ID]; exists {
			t.Fatalf("duplicate territory ID %q", territory.ID)
		}
		indexes[territory.ID] = index
	}

	for _, territory := range data.Territories {
		maximum := maxDegree(territory.Terrain)
		if maximum == 0 {
			t.Errorf("%s has terrain %q without a degree cap", territory.ID, territory.Terrain)
		}
		if len(territory.Adjacencies) < 2 || len(territory.Adjacencies) > maximum {
			t.Errorf("%s has degree %d, want [2, %d] for %q", territory.ID, len(territory.Adjacencies), maximum, territory.Terrain)
		}
		seen := make(map[string]bool, len(territory.Adjacencies))
		for index, adjacentID := range territory.Adjacencies {
			if index > 0 && territory.Adjacencies[index-1] >= adjacentID {
				t.Errorf("%s adjacencies are not strictly sorted", territory.ID)
			}
			if adjacentID == territory.ID {
				t.Errorf("%s has a self adjacency", territory.ID)
			}
			if seen[adjacentID] {
				t.Errorf("%s has duplicate adjacency %s", territory.ID, adjacentID)
			}
			seen[adjacentID] = true
			adjacent, exists := indexes[adjacentID]
			if !exists {
				t.Errorf("%s references unknown territory %s", territory.ID, adjacentID)
				continue
			}
			if !containsID(data.Territories[adjacent].Adjacencies, territory.ID) {
				t.Errorf("%s -> %s is not symmetric", territory.ID, adjacentID)
			}
		}
	}

	seen := make([]bool, len(data.Territories))
	seen[0] = true
	queue := []int{0}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, adjacentID := range data.Territories[current].Adjacencies {
			adjacent, exists := indexes[adjacentID]
			if exists && !seen[adjacent] {
				seen[adjacent] = true
				queue = append(queue, adjacent)
			}
		}
	}
	for territory, reached := range seen {
		if !reached {
			t.Errorf("BFS from %s does not reach %s", data.Territories[0].ID, data.Territories[territory].ID)
		}
	}
}

func containsID(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
