package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// MinimumStartingDistance is the minimum graph distance between two starting
// territories. Adjacent territories have distance 1.
const MinimumStartingDistance = 4

// SelectStartingTerritories chooses count candidates deterministically while
// keeping every selected pair at least MinimumStartingDistance apart. The
// candidate order is the deterministic tie-breaker, so callers can provide a
// sorted or seeded order without adding another source of randomness.
func SelectStartingTerritories(
	candidates []models.TerritoryID,
	adjacencies map[models.TerritoryID][]models.TerritoryID,
	count int,
) ([]models.TerritoryID, error) {
	if count < 1 {
		return nil, fmt.Errorf("engine: starting territory count must be positive, got %d", count)
	}
	if len(candidates) < count {
		return nil, fmt.Errorf("engine: cannot select %d starting territories from %d candidates", count, len(candidates))
	}

	seen := make(map[models.TerritoryID]bool, len(candidates))
	distances := make(map[models.TerritoryID]map[models.TerritoryID]int, len(candidates))
	for _, candidate := range candidates {
		if seen[candidate] {
			return nil, fmt.Errorf("engine: duplicate starting territory candidate %q", candidate)
		}
		seen[candidate] = true
		if _, ok := adjacencies[candidate]; !ok {
			return nil, fmt.Errorf("engine: starting territory candidate %q has no adjacency entry", candidate)
		}
		distances[candidate] = graphDistances(candidate, adjacencies)
	}

	type rankedCandidate struct {
		id       models.TerritoryID
		order    int
		farthest int
	}
	ranked := make([]rankedCandidate, 0, len(candidates))
	for order, candidate := range candidates {
		farthest := -1
		for _, other := range candidates {
			if candidate == other {
				continue
			}
			if distance, reachable := distances[candidate][other]; reachable && distance > farthest {
				farthest = distance
			}
		}
		ranked = append(ranked, rankedCandidate{id: candidate, order: order, farthest: farthest})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].farthest != ranked[j].farthest {
			return ranked[i].farthest > ranked[j].farthest
		}
		return ranked[i].order < ranked[j].order
	})
	ordered := make([]models.TerritoryID, len(ranked))
	for index, candidate := range ranked {
		ordered[index] = candidate.id
	}

	selected, ok := findStartingTerritories(ordered, distances, count, 0, nil)
	if !ok {
		return nil, fmt.Errorf("engine: cannot select %d starting territories at distance %d", count, MinimumStartingDistance)
	}
	return selected, nil
}

func findStartingTerritories(
	candidates []models.TerritoryID,
	distances map[models.TerritoryID]map[models.TerritoryID]int,
	count, offset int,
	selected []models.TerritoryID,
) ([]models.TerritoryID, bool) {
	if len(selected) == count {
		return append([]models.TerritoryID(nil), selected...), true
	}
	needed := count - len(selected)
	if len(candidates)-offset < needed {
		return nil, false
	}

	for index := offset; index <= len(candidates)-needed; index++ {
		candidate := candidates[index]
		compatible := true
		for _, selectedID := range selected {
			distance, reachable := distances[candidate][selectedID]
			if !reachable || distance < MinimumStartingDistance {
				compatible = false
				break
			}
		}
		if !compatible {
			continue
		}

		selected = append(selected, candidate)
		if result, ok := findStartingTerritories(candidates, distances, count, index+1, selected); ok {
			return result, true
		}
		selected = selected[:len(selected)-1]
	}
	return nil, false
}

func graphDistances(start models.TerritoryID, adjacencies map[models.TerritoryID][]models.TerritoryID) map[models.TerritoryID]int {
	distances := map[models.TerritoryID]int{start: 0}
	queue := []models.TerritoryID{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, adjacent := range adjacencies[current] {
			if _, visited := distances[adjacent]; visited {
				continue
			}
			distances[adjacent] = distances[current] + 1
			queue = append(queue, adjacent)
		}
	}
	return distances
}
