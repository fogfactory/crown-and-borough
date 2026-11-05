package mapgen

import (
	"math/rand/v2"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// allPlainTerrain returns a terrain slice of n plain sites, the eligibility
// filter's no-op case: every non-isolated site remains a valid start
// candidate.
func allPlainTerrain(n int) []models.Terrain {
	terrain := make([]models.Terrain, n)
	for index := range terrain {
		terrain[index] = models.TerrainPlain
	}
	return terrain
}

// TestSelectStartsSmallGraphSuccess tests that selectStarts finds a valid
// assignment on a small hand-built graph where placement is clearly possible.
// This pins down expected behavior on a minimal graph, not just fuzzing.
func TestSelectStartsSmallGraphSuccess(t *testing.T) {
	// Graph: 0-1-2-3-4-5-6-7-8 (line topology, 9 territories)
	// For 2 starts:
	//   - Start at 0, home at 2 (distance exactly 2)
	//   - Start at 4, home at 6 (distance exactly 2)
	//   - Distance between starts: 4 (meets >= 4 requirement)
	//   - Distance from home 2 to start 4: 2 (must be >= 3) → FAIL
	// Retry:
	//   - Start at 0, home at 2
	//   - Start at 5, home at 7
	//   - Distance between starts: 5 (OK)
	//   - Distance from home 2 to start 5: 3 (OK)
	//   - Distance from home 7 to start 0: 7 (OK)
	//   - Distance between homes 2 and 7: 5 (OK, >= 2)
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 7}, {7, 8},
	}
	distances := siteDistances(edges, 9)

	rng := rand.New(rand.NewPCG(42, 0))
	starts, homes, err := selectStarts(rng, distances, edges, allPlainTerrain(9), 2)

	if err != nil {
		t.Fatalf("selectStarts returned unexpected error: %v", err)
	}
	if len(starts) != 2 {
		t.Fatalf("returned %d starts, want 2", len(starts))
	}
	if len(homes) != 2 {
		t.Fatalf("returned %d homes, want 2", len(homes))
	}

	// Verify distance constraints are satisfied.
	if distances[starts[0]][starts[1]] < MinimumStartingDistance {
		t.Errorf("starts %d and %d are %d hops apart, want >= %d",
			starts[0], starts[1], distances[starts[0]][starts[1]], MinimumStartingDistance)
	}

	for i := 0; i < 2; i++ {
		if distances[starts[i]][homes[i]] != homeVillageDistance {
			t.Errorf("start %d to its home %d is %d hops, want exactly %d",
				starts[i], homes[i], distances[starts[i]][homes[i]], homeVillageDistance)
		}
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if i != j {
				if distances[homes[i]][starts[j]] < homeVillageMinimumOtherStartDistance {
					t.Errorf("home %d (of start %d) is %d hops from other start %d, want >= %d",
						homes[i], starts[i], distances[homes[i]][starts[j]], starts[j], homeVillageMinimumOtherStartDistance)
				}
			}
		}
	}

	if distances[homes[0]][homes[1]] < homeVillageMinimumSpacing {
		t.Errorf("homes %d and %d are %d hops apart, want >= %d",
			homes[0], homes[1], distances[homes[0]][homes[1]], homeVillageMinimumSpacing)
	}
}

// TestSelectStartsSmallGraphFailure tests that selectStarts returns a clear
// error when the graph is too small/dense to satisfy distance constraints.
// It should return an error rather than panicking, looping forever, or
// silently returning an invalid/incomplete assignment.
func TestSelectStartsSmallGraphFailure(t *testing.T) {
	// Graph: 0-1-2-3 (line topology, 4 territories)
	// Maximum distance is 3 (from 0 to 3).
	// Request 2 starts: they must be >= 4 hops apart.
	// Impossible: max distance is 3.
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3},
	}
	distances := siteDistances(edges, 4)

	rng := rand.New(rand.NewPCG(42, 0))
	starts, homes, err := selectStarts(rng, distances, edges, allPlainTerrain(4), 2)

	if err == nil {
		t.Fatal("selectStarts should return an error when constraints are unsatisfiable")
	}
	if len(starts) != 0 || len(homes) != 0 {
		t.Errorf("on error, returns should be empty; got %d starts and %d homes",
			len(starts), len(homes))
	}
	// Verify the error message is clear (not just a generic panic or nil).
	if err.Error() == "" {
		t.Error("error message is empty")
	}
	if !stringContains(err.Error(), "cannot place") {
		t.Errorf("error message %q does not indicate placement failure", err.Error())
	}
}

// TestEligibleStartSitesFiltersMountainsAndIsolation checks the standalone
// eligibility filter: a mountain site is never eligible, and a non-mountain
// site with fewer than minStartNonMountainNeighbours non-mountain neighbours
// is skipped, regardless of its own terrain (#203).
func TestEligibleStartSitesFiltersMountainsAndIsolation(t *testing.T) {
	// Star graph: hub 0 connects to leaves 1..4.
	//   0: plain, 2 non-mountain neighbours among {1,2} -> eligible if 1,2 are non-mountain
	//   1: forest, mountain terrain hub? no, checks its own neighbours (just 0)
	edges := [][2]int{
		{0, 1}, {0, 2}, {0, 3}, {0, 4},
	}
	terrain := []models.Terrain{
		models.TerrainPlain,    // 0: hub, 4 neighbours, all non-mountain but 3 -> eligible
		models.TerrainForest,   // 1: leaf, 1 neighbour (0, non-mountain) -> ineligible (only 1 neighbour)
		models.TerrainMountain, // 2: mountain itself -> ineligible
		models.TerrainHill,     // 3: leaf, 1 neighbour -> ineligible
		models.TerrainSwamp,    // 4: leaf, 1 neighbour -> ineligible
	}
	order := []int{0, 1, 2, 3, 4}

	eligible := eligibleStartSites(order, edges, terrain)
	if len(eligible) != 1 || eligible[0] != 0 {
		t.Fatalf("eligibleStartSites = %v, want only the hub [0]", eligible)
	}
}

// TestEligibleStartSitesRequiresTwoNonMountainNeighbours checks that a
// non-mountain site surrounded by only one non-mountain neighbour (the rest
// mountain) is ineligible, and becomes eligible once a second non-mountain
// neighbour is available.
func TestEligibleStartSitesRequiresTwoNonMountainNeighbours(t *testing.T) {
	// Line: 0-1-2, with 1 as the candidate.
	edges := [][2]int{{0, 1}, {1, 2}}
	order := []int{0, 1, 2}

	oneNonMountain := []models.Terrain{models.TerrainPlain, models.TerrainPlain, models.TerrainMountain}
	if eligible := eligibleStartSites(order, edges, oneNonMountain); len(eligible) != 0 {
		t.Fatalf("eligibleStartSites = %v, want none: site 1 has only one non-mountain neighbour", eligible)
	}

	twoNonMountain := []models.Terrain{models.TerrainPlain, models.TerrainPlain, models.TerrainForest}
	eligible := eligibleStartSites(order, edges, twoNonMountain)
	if len(eligible) != 1 || eligible[0] != 1 {
		t.Fatalf("eligibleStartSites = %v, want only site 1 once it has two non-mountain neighbours", eligible)
	}
}

// TestSelectStartsSkipsIneligibleSites checks that selectStarts never
// chooses a mountain site, or a non-mountain site without at least two
// non-mountain neighbours, as a starting territory (#203).
func TestSelectStartsSkipsIneligibleSites(t *testing.T) {
	// Line topology, 9 territories, with a mountain at each end. Even
	// though the endpoints are far from everything else (good for the
	// minimum starting distance), they must never be chosen: they are
	// mountains, and besides have only one neighbour each.
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 7}, {7, 8},
	}
	terrain := []models.Terrain{
		models.TerrainMountain, models.TerrainPlain, models.TerrainPlain,
		models.TerrainPlain, models.TerrainPlain, models.TerrainPlain,
		models.TerrainPlain, models.TerrainPlain, models.TerrainMountain,
	}
	distances := siteDistances(edges, 9)

	rng := rand.New(rand.NewPCG(42, 0))
	starts, _, err := selectStarts(rng, distances, edges, terrain, 2)
	if err != nil {
		t.Fatalf("selectStarts returned unexpected error: %v", err)
	}
	for _, start := range starts {
		if terrain[start] == models.TerrainMountain {
			t.Errorf("selectStarts chose mountain site %d as a start", start)
		}
		if start == 1 || start == 7 {
			t.Errorf("selectStarts chose site %d, which has only one non-mountain neighbour", start)
		}
	}
}

// TestSelectStartsErrorsWithNoEligibleSite checks that selectStarts fails
// clearly, rather than panicking or silently succeeding, when no site
// satisfies the eligibility filter.
func TestSelectStartsErrorsWithNoEligibleSite(t *testing.T) {
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3},
	}
	terrain := make([]models.Terrain, 4)
	for index := range terrain {
		terrain[index] = models.TerrainMountain
	}
	distances := siteDistances(edges, 4)

	rng := rand.New(rand.NewPCG(42, 0))
	starts, homes, err := selectStarts(rng, distances, edges, terrain, 1)
	if err == nil {
		t.Fatal("selectStarts should return an error when no site is eligible")
	}
	if len(starts) != 0 || len(homes) != 0 {
		t.Errorf("on error, returns should be empty; got %d starts and %d homes", len(starts), len(homes))
	}
	if !stringContains(err.Error(), "eligible") {
		t.Errorf("error message %q does not indicate an eligibility failure", err.Error())
	}
}

// TestPlaceSeatsSmallGraphSuccess tests that placeSeats places seats when
// enough eligible sites exist.
func TestPlaceSeatsSmallGraphSuccess(t *testing.T) {
	// Graph: 0-1-2-3-4-5-6-7-8-9-10 (line topology, 11 territories)
	// Starts: {0, 5}
	// Homes: {2, 7}
	// Seats must be >= 3 hops from every start and >= 2 hops from every home.
	// Eligible sites:
	//   - 9: dist(0,9)=9 (OK), dist(5,9)=4 (OK), dist(2,9)=7 (OK), dist(7,9)=2 (OK)
	//   - 10: dist(0,10)=10 (OK), dist(5,10)=5 (OK), dist(2,10)=8 (OK), dist(7,10)=3 (OK)
	// Request 2 seats: should succeed.
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 7}, {7, 8}, {8, 9}, {9, 10},
	}
	distances := siteDistances(edges, 11)

	centroids := make([][2]float64, 11)
	for i := 0; i < 11; i++ {
		centroids[i] = [2]float64{float64(i), 0}
	}

	starts := []int{0, 5}
	homes := []int{2, 7}

	rng := rand.New(rand.NewPCG(42, 0))
	seats, err := placeSeats(rng, centroids, distances, starts, homes, 2)

	if err != nil {
		t.Fatalf("placeSeats returned unexpected error: %v", err)
	}
	if len(seats) != 2 {
		t.Fatalf("returned %d seats, want 2", len(seats))
	}

	// Verify distance constraints.
	for _, seat := range seats {
		for _, start := range starts {
			if distances[start][seat] < seatMinimumStartDistance {
				t.Errorf("seat %d is %d hops from start %d, want >= %d",
					seat, distances[start][seat], start, seatMinimumStartDistance)
			}
		}
		for _, home := range homes {
			if distances[home][seat] < seatMinimumHomeDistance {
				t.Errorf("seat %d is %d hops from home %d, want >= %d",
					seat, distances[home][seat], home, seatMinimumHomeDistance)
			}
		}
	}
}

// TestPlaceSeatsInsufficientEligible tests that placeSeats returns a clear
// error when there are fewer eligible sites than the requested seat count.
func TestPlaceSeatsInsufficientEligible(t *testing.T) {
	// Graph: 0-1-2-3-4-5 (line topology, 6 territories)
	// Starts: {0}
	// Homes: {2}
	// Eligible sites must be:
	//   - Not starts/homes: exclude {0, 2}
	//   - >= 3 hops from start 0: exclude {1, 2}
	//   - >= 2 hops from home 2: exclude {0, 1, 2, 3}
	// Remaining candidates: {4, 5} only.
	// Request 3 seats: impossible.
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5},
	}
	distances := siteDistances(edges, 6)

	centroids := make([][2]float64, 6)
	for i := 0; i < 6; i++ {
		centroids[i] = [2]float64{float64(i), 0}
	}

	starts := []int{0}
	homes := []int{2}

	rng := rand.New(rand.NewPCG(42, 0))
	seats, err := placeSeats(rng, centroids, distances, starts, homes, 3)

	if err == nil {
		t.Fatalf("placeSeats should return an error when requesting %d seats but fewer than %d sites are eligible",
			3, 3)
	}
	if len(seats) != 0 {
		t.Errorf("on error, seats should be empty; got %v", seats)
	}
	// Verify the error message is clear.
	if err.Error() == "" {
		t.Error("error message is empty")
	}
	if !stringContains(err.Error(), "need at least") {
		t.Errorf("error message %q does not indicate insufficient eligible sites", err.Error())
	}
}

// stringContains reports whether s contains substr.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
