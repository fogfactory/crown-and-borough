package mapgen

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

var testConfig = Config{
	Width:        1000,
	Height:       700,
	SiteCount:    64,
	LieuDitRatio: 0.25,
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

func TestLieuDitRatio(t *testing.T) {
	assets := loadTestAssets(t)
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data := generateTestMap(t, seed, assets)
			count := 0
			for _, territory := range data.Territories {
				if territory.LieuDit {
					count++
				}
			}
			ratio := float64(count) / float64(len(data.Territories))
			if ratio < 0.20 || ratio > 0.30 {
				t.Fatalf("lieu-dit ratio = %.2f, want between 0.20 and 0.30", ratio)
			}
		})
	}
}

func TestCodes(t *testing.T) {
	assets := loadTestAssets(t)
	lieuDitCode := regexp.MustCompile(`^[A-Z]{3}$`)
	territoryCode := regexp.MustCompile(`^[A-Z]{4}$`)
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data := generateTestMap(t, seed, assets)
			seen := make(map[string]bool, len(data.Territories))
			for _, territory := range data.Territories {
				valid := territoryCode.MatchString(territory.Code)
				if territory.LieuDit {
					valid = lieuDitCode.MatchString(territory.Code)
				}
				if !valid {
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
	for _, seed := range testSeeds {
		t.Run(seed, func(t *testing.T) {
			data := generateTestMap(t, seed, assets)
			seenTerrain := make(map[models.Terrain]bool, len(allTerrains))
			for _, territory := range data.Territories {
				if territory.Name == "" {
					t.Errorf("%s has an empty name", territory.ID)
				}
				seenTerrain[territory.Terrain] = true
				if !territory.LieuDit && !validTerritoryName(territory, assets, testConfig) {
					t.Errorf("%s has an invalid generated name %q", territory.ID, territory.Name)
				}
			}
			for _, terrain := range allTerrains {
				if !seenTerrain[terrain] {
					t.Errorf("terrain %q is absent", terrain)
				}
			}
		})
	}
}

func TestNamingCollisions(t *testing.T) {
	assets := assetgen.Assets{
		Communes: []assetgen.Asset{
			{Code: "AAA", Name: "Aubeterre"},
			{Code: "BBB", Name: "Bellac"},
		},
		Qualificatifs: []assetgen.Qualificatif{
			{Prefix: "F", Name: "Forêt", Terrain: "plain"},
			{Prefix: "B", Name: "Bois", Terrain: "plain"},
		},
	}
	lieuDits := []bool{true, true, false, false, false, false}
	terrain := []models.Terrain{
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
		models.TerrainPlain,
	}
	polygons := make([][][2]int, len(lieuDits))
	centroids := make([][2]float64, len(lieuDits))
	for i := range polygons {
		polygons[i] = [][2]int{{10, 10}, {20, 10}, {20, 20}, {10, 20}}
		centroids[i] = [2]float64{float64(i * 10), 0}
	}
	edges := [][2]int{{0, 2}, {1, 3}, {0, 4}, {1, 5}}
	cfg := Config{Width: 100, Height: 100}

	first, err := nameTerritories(
		newRNG("collision", "naming"),
		assets,
		lieuDits,
		terrain,
		polygons,
		centroids,
		edges,
		cfg,
	)
	if err != nil {
		t.Fatalf("first naming pass: %v", err)
	}
	second, err := nameTerritories(
		newRNG("collision", "naming"),
		assets,
		lieuDits,
		terrain,
		polygons,
		centroids,
		edges,
		cfg,
	)
	if err != nil {
		t.Fatalf("second naming pass: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("naming collisions are not deterministic")
	}

	seenNames := make(map[string]bool)
	seenCodes := make(map[string]bool)
	for site := 2; site < len(first); site++ {
		if seenNames[first[site].name] || seenCodes[first[site].code] {
			t.Fatalf("duplicate collision result for %q / %q", first[site].name, first[site].code)
		}
		seenNames[first[site].name] = true
		seenCodes[first[site].code] = true
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
		{Width: 99, Height: 700, SiteCount: 64, LieuDitRatio: 0.25},
		{Width: 1000, Height: 99, SiteCount: 64, LieuDitRatio: 0.25},
		{Width: 1000, Height: 700, SiteCount: 7, LieuDitRatio: 0.25},
		{Width: 1000, Height: 700, SiteCount: 64, LieuDitRatio: 0},
		{Width: 1000, Height: 700, SiteCount: 64, LieuDitRatio: 0.51},
		{Width: 1000, Height: 700, SiteCount: 64, LieuDitRatio: math.NaN()},
	}
	for _, cfg := range invalid {
		if _, err := Generate("invalid", assets, cfg); err == nil {
			t.Errorf("Generate(%+v) returned no validation error", cfg)
		}
	}
	if _, err := Generate("assets", assetgen.Assets{}, testConfig); err == nil {
		t.Error("Generate with empty assets returned no error")
	}
}

func TestMinimumConfig(t *testing.T) {
	assets := loadTestAssets(t)
	cfg := Config{Width: 100, Height: 100, SiteCount: 8, LieuDitRatio: 0.5}
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

func validTerritoryName(territory Territory, assets assetgen.Assets, cfg Config) bool {
	for _, qualifier := range assets.Qualificatifs {
		if qualifier.Terrain != string(territory.Terrain) &&
			!(qualifier.Terrain == "any" && isBorder(territory.Points, cfg)) {
			continue
		}
		prefix := qualifier.Name + " de "
		if !strings.HasPrefix(territory.Name, prefix) {
			continue
		}
		communeName := strings.TrimPrefix(territory.Name, prefix)
		for _, commune := range assets.Communes {
			if commune.Name == communeName {
				return true
			}
		}
	}
	return false
}
