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
	SiteCount:    64,
	VillageCount: 5,
}

var testSeeds = []string{"alpha", "beta", "gamma", "delta", "epsilon"}

func TestDeterminism(t *testing.T) {
	assets := loadTestAssets(t)
	first := generateTestMap(t, "alpha", assets)
	second := generateTestMap(t, "alpha", assets)
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
	if bytes.Contains(firstJSON, []byte("lieuDit")) {
		t.Fatal("map JSON still contains the removed lieuDit flag")
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

func TestGraphInvariants(t *testing.T) {
	assets := loadTestAssets(t)
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			assertGraphInvariants(t, generateTestMap(t, seed, assets))
		})
	}
}

func TestVillageCount(t *testing.T) {
	assets := loadTestAssets(t)
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
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
}

func TestVillageSpread(t *testing.T) {
	assets := loadTestAssets(t)
	diagonal := math.Hypot(float64(testConfig.Width), float64(testConfig.Height))
	// Villages are spread by a greedy max-min placement: no two villages may
	// sit closer than 15% of the viewport diagonal.
	minDistance := 0.15 * diagonal
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data := generateTestMap(t, seed, assets)
			var villages []int
			for i, territory := range data.Territories {
				if territory.Village {
					villages = append(villages, i)
				}
			}
			for a, first := range villages {
				for _, second := range villages[a+1:] {
					c1 := polygonCentroid(data.Territories[first].Points)
					c2 := polygonCentroid(data.Territories[second].Points)
					actual := math.Hypot(c1[0]-c2[0], c1[1]-c2[1])
					if actual < minDistance {
						t.Errorf("villages %s and %s are %v apart, want >= %v", data.Territories[first].ID, data.Territories[second].ID, actual, minDistance)
					}
				}
			}
		})
	}
}

func TestCodes(t *testing.T) {
	assets := loadTestAssets(t)
	territoryCode := regexp.MustCompile(`^[A-Z]{3}$`)
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
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
}

func TestNaming(t *testing.T) {
	assets := loadTestAssets(t)
	communesByName := make(map[string]assetgen.Commune, len(assets.Communes))
	for _, commune := range assets.Communes {
		communesByName[commune.Name] = commune
	}
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
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
}

func TestNamingAffinity(t *testing.T) {
	assets := loadTestAssets(t)
	communesByName := make(map[string]assetgen.Commune, len(assets.Communes))
	for _, commune := range assets.Communes {
		communesByName[commune.Name] = commune
	}
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data := generateTestMap(t, seed, assets)
			seenTerrain := make(map[models.Terrain]bool, len(allTerrains))
			usedAffinity := make(map[models.Terrain]bool, len(allTerrains))
			affine := 0
			for _, territory := range data.Territories {
				commune := communesByName[territory.Name]
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

func TestGeometry(t *testing.T) {
	assets := loadTestAssets(t)
	data := generateTestMap(t, "alpha", assets)
	for _, territory := range data.Territories {
		if len(territory.Points) < 3 {
			t.Errorf("%s has %d points, want at least 3", territory.ID, len(territory.Points))
		}
		if polygonAreaTwice(territory.Points) <= 0 {
			t.Errorf("%s has non-positive polygon area", territory.ID)
		}
		for _, point := range territory.Points {
			if point[0] < 0 || point[0] > testConfig.Width || point[1] < 0 || point[1] > testConfig.Height {
				t.Errorf("%s point %v is outside the viewport", territory.ID, point)
			}
		}
	}

	sites := generateSites(newRNG("alpha", "sites"), testConfig)
	grid := assignRaster(sites, testConfig)
	if len(grid) != gridW*gridH {
		t.Fatalf("raster length = %d, want %d", len(grid), gridW*gridH)
	}
	if !rasterHasEveryRegion(grid, len(sites)) {
		t.Fatal("raster has an unassigned cell or empty region")
	}
	counts := make([]int, len(sites))
	for _, owner := range grid {
		if owner < 0 || owner >= len(sites) {
			t.Fatalf("raster owner %d is outside site range", owner)
		}
		counts[owner]++
	}
	for site, count := range counts {
		if count == 0 {
			t.Errorf("site %d has no raster cells", site)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	assets := loadTestAssets(t)
	invalid := []Config{
		{Width: 99, Height: 700, SiteCount: 64, VillageCount: 5},
		{Width: 1000, Height: 99, SiteCount: 64, VillageCount: 5},
		{Width: 1000, Height: 700, SiteCount: 7, VillageCount: 5},
		{Width: 1000, Height: 700, SiteCount: 64, VillageCount: 0},
		{Width: 1000, Height: 700, SiteCount: 64, VillageCount: -1},
		{Width: 1000, Height: 700, SiteCount: 64, VillageCount: 65},
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
	cfg := Config{Width: 100, Height: 100, SiteCount: 8, VillageCount: 1}
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data, err := Generate(seed, assets, cfg)
			if err != nil {
				t.Fatalf("Generate(%q) with minimum config: %v", seed, err)
			}
			if len(data.Territories) != cfg.SiteCount {
				t.Errorf("territory count = %d, want %d", len(data.Territories), cfg.SiteCount)
			}
		})
	}
}

func TestRepairGraphUsesNonGeometricRoutes(t *testing.T) {
	centroids := [][2]float64{
		{0, 0},
		{10, 0},
		{20, 0},
		{30, 0},
		{40, 0},
		{50, 0},
	}
	geometric := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}, {0, 5}}
	repaired := repairGraphWithGeometry(nil, geometric, centroids, len(centroids))
	if err := validateGraph(repaired, len(centroids)); err != nil {
		t.Fatalf("repaired graph violates invariants: %v", err)
	}
	for _, edge := range repaired {
		if containsEdge(geometric, edge) {
			t.Errorf("route %v is a geometric adjacency", edge)
		}
	}
}

func TestRepairGraphWithoutGeometry(t *testing.T) {
	centroids := [][2]float64{
		{0, 0},
		{10, 0},
		{20, 0},
		{30, 0},
		{40, 0},
		{50, 0},
	}
	edges := [][2]int{{0, 1}, {1, 2}, {3, 4}}
	repaired := repairGraph(edges, centroids, len(centroids))
	if err := validateGraph(repaired, len(centroids)); err != nil {
		t.Fatalf("geometry-agnostic repaired graph violates invariants: %v", err)
	}
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
	data, err := Generate(seed, assets, testConfig)
	if err != nil {
		t.Fatalf("Generate(%q): %v", seed, err)
	}
	return data
}

func assertGraphInvariants(t *testing.T, data MapData) {
	t.Helper()
	index := make(map[string]int, len(data.Territories))
	for i, territory := range data.Territories {
		if _, exists := index[territory.ID]; exists {
			t.Fatalf("duplicate territory ID %q", territory.ID)
		}
		index[territory.ID] = i
	}

	for _, territory := range data.Territories {
		seen := make(map[string]bool, len(territory.Adjacencies))
		if len(territory.Adjacencies) < 2 {
			t.Errorf("%s has degree %d, want at least 2", territory.ID, len(territory.Adjacencies))
		}
		for i, adjacentID := range territory.Adjacencies {
			if i > 0 && territory.Adjacencies[i-1] > adjacentID {
				t.Errorf("%s adjacencies are not sorted", territory.ID)
			}
			if adjacentID == territory.ID {
				t.Errorf("%s has a self adjacency", territory.ID)
			}
			if seen[adjacentID] {
				t.Errorf("%s has duplicate adjacency %s", territory.ID, adjacentID)
			}
			seen[adjacentID] = true
			adjacentIndex, exists := index[adjacentID]
			if !exists {
				t.Errorf("%s references unknown territory %s", territory.ID, adjacentID)
				continue
			}
			if !containsID(data.Territories[adjacentIndex].Adjacencies, territory.ID) {
				t.Errorf("%s -> %s is not symmetric", territory.ID, adjacentID)
			}
		}
	}

	for start := range data.Territories {
		seen := make([]bool, len(data.Territories))
		seen[start] = true
		queue := []int{start}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, adjacentID := range data.Territories[current].Adjacencies {
				adjacent := index[adjacentID]
				if !seen[adjacent] {
					seen[adjacent] = true
					queue = append(queue, adjacent)
				}
			}
		}
		for territory, reached := range seen {
			if !reached {
				t.Errorf("BFS from %s does not reach %s", data.Territories[start].ID, data.Territories[territory].ID)
			}
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

func containsEdge(edges [][2]int, wanted [2]int) bool {
	for _, edge := range edges {
		if edge == wanted {
			return true
		}
	}
	return false
}
