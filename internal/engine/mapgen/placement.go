package mapgen

import (
	"errors"
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

// selectStarts draws several valid start placements and keeps the fairest:
// the one that leaves the least-served start the largest share of the map,
// measured as the territories at least as close to it as to any other start.
// A placement that squeezes one player between two others, or packs all the
// players on one side, leaves that player a small share and loses.
func selectStarts(rng *rand.Rand, distances [][]int, edges [][2]int, terrain []models.Terrain, count int) (starts, homes []int, err error) {
	starts, homes, err = searchStarts(rng, distances, edges, terrain, count, startSearchBudget)
	if err != nil {
		return nil, nil, err
	}
	bestShare, bestSpread := startShareRange(distances, starts)
	for trial := 1; trial < startTrials; trial++ {
		candidate, candidateHomes, trialErr := searchStarts(rng, distances, edges, terrain, count, startTrialBudget)
		if trialErr != nil {
			continue
		}
		share, spread := startShareRange(distances, candidate)
		if share > bestShare || (share == bestShare && spread < bestSpread) {
			starts, homes, bestShare, bestSpread = candidate, candidateHomes, share, spread
		}
	}
	return starts, homes, nil
}

// startShareRange returns the smallest start share and the gap between the
// largest and the smallest. A site equally close to several starts is split
// between them.
func startShareRange(distances [][]int, starts []int) (smallest, spread float64) {
	shares := make([]float64, len(starts))
	for site := range distances {
		nearest := distances[starts[0]][site]
		for _, start := range starts[1:] {
			nearest = min(nearest, distances[start][site])
		}
		ties := 0
		for _, start := range starts {
			if distances[start][site] == nearest {
				ties++
			}
		}
		for index, start := range starts {
			if distances[start][site] == nearest {
				shares[index] += 1 / float64(ties)
			}
		}
	}
	smallest, largest := shares[0], shares[0]
	for _, share := range shares[1:] {
		smallest, largest = math.Min(smallest, share), math.Max(largest, share)
	}
	return smallest, largest - smallest
}

// searchStarts jointly chooses count starting territories and, for each, a
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
func searchStarts(rng *rand.Rand, distances [][]int, edges [][2]int, terrain []models.Terrain, count, budget int) (starts, homes []int, err error) {
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
	if !backtrackStarts(startOrder, distances, homeCandidates, count, 0, &starts, &homes, usedAsStart, usedAsHome, &budget) {
		if budget <= 0 {
			return nil, nil, fmt.Errorf("%w placing %d starting positions", errStartSearchBudget, count)
		}
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
	budget *int,
) bool {
	if len(*starts) == count {
		return true
	}
	if *budget <= 0 {
		return false
	}
	*budget--
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

			if backtrackStarts(order, distances, homeCandidates, count, index+1, starts, homes, usedAsStart, usedAsHome, budget) {
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

// startSearchBudget bounds the backtracking over start positions. A site
// order with no valid placement would otherwise be explored exhaustively;
// running out of budget fails the attempt so generation retries another draw.
const startSearchBudget = 200_000

const (
	// startTrials is the number of valid start placements compared for
	// fairness; the value was picked empirically (see TestStartsShareTheMapFairly).
	startTrials = 24
	// startTrialBudget is the search budget of every trial after the first.
	// The first trial gets the full startSearchBudget so a placement is found
	// whenever one is reachable; the others only refine it.
	startTrialBudget = startSearchBudget / 10
)

var errStartSearchBudget = errors.New("mapgen: start search budget exhausted")

const (
	// seatCandidateTrials is the number of random seat sets scored when
	// choosing the chefs-lieux.
	seatCandidateTrials = 400
	// seatMinimumSpacing keeps two chefs-lieux at least this many passable
	// hops apart.
	seatMinimumSpacing = 3
	// seatOutlineFactor scales the minimum frontier length into the minimum
	// outline contact a bishopric needs to count as touching the map edge.
	seatOutlineFactor = 1.5
	// seatSpreadDiagonalRatio is the minimum distance between two chefs-lieux,
	// as a share of the map diagonal.
	seatSpreadDiagonalRatio = 0.075
)

// eligibleSeats lists the sites that may host a chef-lieu: not a start nor a
// home village, at least seatMinimumStartDistance from every start and
// seatMinimumHomeDistance from every home village.
func eligibleSeats(distances [][]int, starts, homes []int) []int {
	n := len(distances)
	excluded := make([]bool, n)
	for _, start := range starts {
		excluded[start] = true
	}
	for _, home := range homes {
		excluded[home] = true
	}
	eligible := make([]int, 0, n)
	for site := 0; site < n; site++ {
		if excluded[site] ||
			!farEnoughFrom(distances, starts, site, seatMinimumStartDistance) ||
			!farEnoughFrom(distances, homes, site, seatMinimumHomeDistance) {
			continue
		}
		eligible = append(eligible, site)
	}
	return eligible
}

// seatScore ranks a candidate seat set; lower is better, compared in order:
// regions missing a wide enough contact with the map outline, starts that sit
// alone in their region, then the negated minimum seat spacing.
type seatScore struct {
	edgeMisses int
	loneStarts int
	spread     float64
}

func (s seatScore) betterThan(other seatScore) bool {
	if s.edgeMisses != other.edgeMisses {
		return s.edgeMisses < other.edgeMisses
	}
	if s.loneStarts != other.loneStarts {
		return s.loneStarts < other.loneStarts
	}
	return s.spread > other.spread
}

// seatSearch gathers what chooseSeats needs: the territories and their
// geometry, the sites eligible for a chef-lieu, the starts, the outline contact
// per site, the minimum outline contact and centroid spacing, and the number of
// chefs-lieux to place.
type seatSearch struct {
	territories                []Territory
	centroids                  [][2]float64
	distances                  [][]int
	eligible, starts           []int
	outline                    []float64
	edgeMinimum, spreadMinimum float64
	count                      int
}

// chooseSeats picks the chefs-lieux so that bishoprics are contested: it
// scores many random seat sets by the regions they produce and keeps the one
// where the fewest starts sit alone in their region, every region reaching
// the map outline. The greedy farthest-point set joins the trials as a
// fallback, so a result exists whenever one valid set does.
func chooseSeats(rng *rand.Rand, search seatSearch) ([]int, error) {
	territories, centroids, distances := search.territories, search.centroids, search.distances
	eligible, starts, outline := search.eligible, search.starts, search.outline
	edgeMinimum, spreadMinimum, count := search.edgeMinimum, search.spreadMinimum, search.count
	if count > len(eligible) {
		return nil, fmt.Errorf("mapgen: need at least %d eligible sites for chefs-lieux, have %d", count, len(eligible))
	}
	startIDs := make(map[models.TerritoryID]bool, len(starts))
	for _, start := range starts {
		startIDs[models.TerritoryID(territories[start].ID)] = true
	}
	index := make(map[models.TerritoryID]int, len(territories))
	for site, territory := range territories {
		index[models.TerritoryID(territory.ID)] = site
	}

	var best []int
	var bestScore seatScore
	consider := func(seats []int, spread float64) {
		ids := make([]models.TerritoryID, len(seats))
		for i, seat := range seats {
			ids[i] = models.TerritoryID(territories[seat].ID)
		}
		regions, err := generateRegions(territories, ids)
		if err != nil {
			return
		}
		score := seatScore{spread: spread}
		for _, region := range regions {
			touches, starting := false, 0
			for _, id := range region.Territories {
				if outline[index[id]] >= edgeMinimum {
					touches = true
				}
				if startIDs[id] {
					starting++
				}
			}
			if !touches {
				score.edgeMisses++
			}
			if starting == 1 {
				score.loneStarts++
			}
		}
		if best == nil || score.betterThan(bestScore) {
			best, bestScore = append([]int(nil), seats...), score
		}
	}

	greedy := greedySeats(rng, centroids, eligible, count)
	consider(greedy, minSeatSpread(centroids, greedy))
	pool := append([]int(nil), eligible...)
	for trial := 0; trial < seatCandidateTrials; trial++ {
		shuffle(rng, pool)
		seats := make([]int, 0, count)
		for _, site := range pool {
			if farEnoughFrom(distances, seats, site, seatMinimumSpacing) {
				seats = append(seats, site)
				if len(seats) == count {
					break
				}
			}
		}
		if len(seats) == count {
			if spread := minSeatSpread(centroids, seats); spread >= spreadMinimum*spreadMinimum {
				consider(seats, spread)
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("mapgen: no valid chef-lieu placement")
	}
	sort.Ints(best)
	return best, nil
}

func minSeatSpread(centroids [][2]float64, seats []int) float64 {
	spread := math.Inf(1)
	for i, first := range seats {
		for _, second := range seats[i+1:] {
			spread = math.Min(spread, centroidDistanceSquared(centroids, first, second))
		}
	}
	return spread
}

// greedySeats is the farthest-point spread used before seats were scored.
func greedySeats(rng *rand.Rand, centroids [][2]float64, eligible []int, count int) []int {
	selected := make(map[int]bool, count)
	first := eligible[rng.IntN(len(eligible))]
	selected[first] = true
	chosen := []int{first}
	for len(chosen) < count {
		bestSite, bestDistance := -1, -1.0
		for _, site := range eligible {
			if selected[site] {
				continue
			}
			nearest := squaredDistanceToChosen(centroids, site, chosen)
			if bestSite == -1 || nearest > bestDistance || (nearest == bestDistance && site < bestSite) {
				bestSite, bestDistance = site, nearest
			}
		}
		selected[bestSite] = true
		chosen = append(chosen, bestSite)
	}
	return chosen
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
