package mapgen

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	// MinimumStartingDistance is the minimum graph distance between two
	// starting territories. Adjacent territories have distance 1.
	MinimumStartingDistance = 4
	// homeVillageDistance is the exact graph distance between a starting
	// territory and its dedicated home village.
	homeVillageDistance = 2
	// homeVillageMinimumOtherStartDistance keeps a home village away from
	// every starting territory other than its own, so it never sits next to
	// a rival's capital.
	homeVillageMinimumOtherStartDistance = 3
	// homeVillageMinimumSpacing keeps home villages from crowding each
	// other.
	homeVillageMinimumSpacing = 2
	// seatMinimumStartDistance keeps a chef-lieu away from every starting
	// territory.
	seatMinimumStartDistance = 3
	// seatMinimumHomeDistance keeps a chef-lieu away from every home
	// village.
	seatMinimumHomeDistance = 2
	// minStartNonMountainNeighbours is the minimum number of non-mountain
	// passable neighbours a starting territory must have, so a capital
	// always has room to place its outpost armies (#203).
	minStartNonMountainNeighbours = 2
)

// siteDistances returns, for every site, the graph distance (in passable
// hops) to every other site. Unreachable sites are impossible once the
// graph-connectivity invariant holds, so distance is always finite here.
func siteDistances(edges [][2]int, n int) [][]int {
	adjacency := make([][]int, n)
	for _, edge := range edges {
		adjacency[edge[0]] = append(adjacency[edge[0]], edge[1])
		adjacency[edge[1]] = append(adjacency[edge[1]], edge[0])
	}
	distances := make([][]int, n)
	for start := 0; start < n; start++ {
		distances[start] = bfsDistances(adjacency, n, start)
	}
	return distances
}

func bfsDistances(adjacency [][]int, n, start int) []int {
	distance := make([]int, n)
	for index := range distance {
		distance[index] = -1
	}
	distance[start] = 0
	queue := []int{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if distance[next] != -1 {
				continue
			}
			distance[next] = distance[current] + 1
			queue = append(queue, next)
		}
	}
	return distance
}

// selectStarts jointly chooses count starting territories and, for each, a
// dedicated home village at graph distance exactly homeVillageDistance. The
// two are chosen together by backtracking over both starts and homes at
// once: picking every start first and only then hunting for a compatible
// home village routinely fails to find a valid assignment at high player
// counts even though one exists, because early start choices can strand
// later ones without any legal home site.
//
// Only sites eligible per eligibleStartSites can become a starting
// territory: a start's own terrain and neighbourhood determine turn-one
// viability (#203), so ineligible sites are filtered out of the start
// candidate pool before the search begins. Home villages are not
// constrained this way and are still drawn from every site.
func selectStarts(rng *rand.Rand, distances [][]int, edges [][2]int, terrain []models.Terrain, count int) (starts, homes []int, err error) {
	n := len(distances)
	if count < 1 {
		return nil, nil, fmt.Errorf("mapgen: starting position count must be positive, got %d", count)
	}
	if n < count {
		return nil, nil, fmt.Errorf("mapgen: cannot select %d starting positions from %d sites", count, n)
	}

	order := make([]int, n)
	for index := range order {
		order[index] = index
	}
	shuffle(rng, order)

	startOrder := eligibleStartSites(order, edges, terrain)
	if len(startOrder) < count {
		return nil, nil, fmt.Errorf("mapgen: only %d sites are eligible starting positions, need %d", len(startOrder), count)
	}

	homeCandidates := make([][]int, n)
	for site := 0; site < n; site++ {
		for _, candidate := range order {
			if distances[site][candidate] == homeVillageDistance {
				homeCandidates[site] = append(homeCandidates[site], candidate)
			}
		}
	}

	usedAsStart := make([]bool, n)
	usedAsHome := make([]bool, n)
	starts = make([]int, 0, count)
	homes = make([]int, 0, count)
	if !backtrackStarts(startOrder, distances, homeCandidates, count, 0, &starts, &homes, usedAsStart, usedAsHome) {
		return nil, nil, fmt.Errorf("mapgen: cannot place %d starting positions with home villages", count)
	}
	return starts, homes, nil
}

// eligibleStartSites returns the subset of order (preserving its order) that
// qualifies as a starting territory: non-mountain terrain with at least
// minStartNonMountainNeighbours non-mountain passable neighbours. A start
// that fails this filter risks starving its garrison turn one and leaves no
// room for its outpost armies (#203).
func eligibleStartSites(order []int, edges [][2]int, terrain []models.Terrain) []int {
	adjacency := make([][]int, len(terrain))
	for _, edge := range edges {
		adjacency[edge[0]] = append(adjacency[edge[0]], edge[1])
		adjacency[edge[1]] = append(adjacency[edge[1]], edge[0])
	}

	eligible := make([]int, 0, len(order))
	for _, site := range order {
		if terrain[site] == models.TerrainMountain {
			continue
		}
		nonMountainNeighbours := 0
		for _, neighbour := range adjacency[site] {
			if terrain[neighbour] != models.TerrainMountain {
				nonMountainNeighbours++
			}
		}
		if nonMountainNeighbours < minStartNonMountainNeighbours {
			continue
		}
		eligible = append(eligible, site)
	}
	return eligible
}

func backtrackStarts(
	order []int,
	distances [][]int,
	homeCandidates [][]int,
	count, offset int,
	starts, homes *[]int,
	usedAsStart, usedAsHome []bool,
) bool {
	if len(*starts) == count {
		return true
	}
	needed := count - len(*starts)
	if len(order)-offset < needed {
		return false
	}

	for index := offset; index <= len(order)-needed; index++ {
		candidate := order[index]
		if usedAsStart[candidate] || usedAsHome[candidate] {
			continue
		}
		if !startCompatible(distances, *starts, *homes, candidate) {
			continue
		}

		for _, home := range homeCandidates[candidate] {
			if usedAsStart[home] || usedAsHome[home] || home == candidate {
				continue
			}
			if !homeCompatible(distances, *starts, *homes, home) {
				continue
			}

			*starts = append(*starts, candidate)
			*homes = append(*homes, home)
			usedAsStart[candidate] = true
			usedAsHome[home] = true

			if backtrackStarts(order, distances, homeCandidates, count, index+1, starts, homes, usedAsStart, usedAsHome) {
				return true
			}

			*starts = (*starts)[:len(*starts)-1]
			*homes = (*homes)[:len(*homes)-1]
			usedAsStart[candidate] = false
			usedAsHome[home] = false
		}
	}
	return false
}

// startCompatible reports whether candidate can join starts without
// violating the starting-distance floor against other starts, or crowding
// an already-placed home village that is not its own.
func startCompatible(distances [][]int, starts, homes []int, candidate int) bool {
	for _, start := range starts {
		if distances[start][candidate] < MinimumStartingDistance {
			return false
		}
	}
	for _, home := range homes {
		if distances[home][candidate] < homeVillageMinimumOtherStartDistance {
			return false
		}
	}
	return true
}

// homeCompatible reports whether home can join homes without sitting too
// close to a rival start or to another home village.
func homeCompatible(distances [][]int, starts, homes []int, home int) bool {
	for _, start := range starts {
		if distances[home][start] < homeVillageMinimumOtherStartDistance {
			return false
		}
	}
	for _, other := range homes {
		if distances[home][other] < homeVillageMinimumSpacing {
			return false
		}
	}
	return true
}

// placeSeats chooses count chefs-lieux among the sites left over once starts
// and homes are placed, maximizing the minimum centroid distance between
// them (the same greedy max-min spread previously used for every neutral
// village). Eligible sites stay at least seatMinimumStartDistance from every
// start and seatMinimumHomeDistance from every home village.
func placeSeats(rng *rand.Rand, centroids [][2]float64, distances [][]int, starts, homes []int, count int) ([]int, error) {
	n := len(centroids)
	excluded := make([]bool, n)
	for _, start := range starts {
		excluded[start] = true
	}
	for _, home := range homes {
		excluded[home] = true
	}

	eligible := make([]int, 0, n)
	for site := 0; site < n; site++ {
		if excluded[site] {
			continue
		}
		if !farEnoughFrom(distances, starts, site, seatMinimumStartDistance) {
			continue
		}
		if !farEnoughFrom(distances, homes, site, seatMinimumHomeDistance) {
			continue
		}
		eligible = append(eligible, site)
	}
	if count > len(eligible) {
		return nil, fmt.Errorf("mapgen: need at least %d eligible sites for chefs-lieux, have %d", count, len(eligible))
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
	sort.Ints(chosen)
	return chosen, nil
}

func farEnoughFrom(distances [][]int, from []int, site, minimum int) bool {
	for _, other := range from {
		if distances[other][site] < minimum {
			return false
		}
	}
	return true
}

// squaredDistanceToChosen returns the squared centroid distance from site to
// its nearest already-chosen site.
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
