package mapgen

import (
	"fmt"
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// pruneFrontiers starts with every geometric arc traversable. It consumes one
// seeded draw per sorted arc, but only reclassifies an arc when doing so keeps
// the traversable graph connected and both endpoint degrees at least two.
func pruneFrontiers(
	rng *rand.Rand,
	arcs [][2]int,
	terrain []models.Terrain,
) (passable, impassable [][2]int) {
	n := len(terrain)
	matrix := edgeMatrix(n, arcs)
	for _, arc := range arcs {
		roll := rng.Float64()
		if roll >= frontierRemovalChance(terrain[arc[0]], terrain[arc[1]]) {
			continue
		}
		if canDowngradeArc(matrix, n, arc[0], arc[1]) {
			matrixClearEdge(matrix, n, arc[0], arc[1])
		}
	}
	passable = edgesFromMatrix(matrix, n)
	return passable, differenceEdges(arcs, passable)
}

func frontierRemovalChance(first, second models.Terrain) float64 {
	if first == models.TerrainPlain && second == models.TerrainPlain {
		return 0
	}
	if difficultFrontier(first, second) {
		return 0.75
	}
	return 0.15
}

func difficultFrontier(first, second models.Terrain) bool {
	return (first == models.TerrainMountain && second == models.TerrainMountain) ||
		(first == models.TerrainMountain && second == models.TerrainSwamp) ||
		(first == models.TerrainSwamp && second == models.TerrainMountain)
}

// capDegrees demotes only existing passable geometric arcs. Candidate order is
// difficult terrain pairs, then longest centroid distance, then lower indexes.
func capDegrees(
	passable [][2]int,
	terrain []models.Terrain,
	centroids [][2]float64,
) ([][2]int, [][2]int, error) {
	n := len(terrain)
	if len(centroids) != n {
		return nil, nil, fmt.Errorf("mapgen: centroid count does not match terrain count")
	}
	matrix := edgeMatrix(n, passable)
	for {
		degree := degreesFromMatrix(matrix, n)
		overfull := -1
		for territory, value := range degree {
			if value > maxDegree(terrain[territory]) {
				overfull = territory
				break
			}
		}
		if overfull < 0 {
			final := edgesFromMatrix(matrix, n)
			return final, differenceEdges(passable, final), nil
		}

		first, second, found := bestDegreeCapCandidate(matrix, terrain, centroids, overfull)
		if !found {
			return nil, nil, fmt.Errorf(
				"mapgen: cannot enforce max degree for territory %s (degree %d, max %d)",
				territoryID(overfull), degree[overfull], maxDegree(terrain[overfull]),
			)
		}
		matrixClearEdge(matrix, n, first, second)
	}
}

// enforceDegreeCaps keeps stochastic pruning from turning an otherwise valid
// geometric graph into an irreducible degree-cap configuration. A downgrade is
// restored only when the cap phase proves it needs additional structure; a
// graph that remains impossible after every restoration returns the explicit
// cap error unchanged.
func enforceDegreeCaps(
	passable, impassable [][2]int,
	terrain []models.Terrain,
	centroids [][2]float64,
) ([][2]int, [][2]int, error) {
	capped, demoted, err := capDegrees(passable, terrain, centroids)
	if err == nil {
		return capped, mergeEdges(impassable, demoted), nil
	}

	restored := append([][2]int(nil), passable...)
	for index, edge := range impassable {
		restored = mergeEdges(restored, [][2]int{edge})
		capped, demoted, capErr := capDegrees(restored, terrain, centroids)
		if capErr == nil {
			return capped, mergeEdges(impassable[index+1:], demoted), nil
		}
	}
	return nil, nil, err
}

func maxDegree(terrain models.Terrain) int {
	switch terrain {
	case models.TerrainMountain, models.TerrainSwamp, models.TerrainHill:
		return 3
	case models.TerrainPlain, models.TerrainForest:
		return 5
	default:
		return 0
	}
}

func bestDegreeCapCandidate(
	matrix []bool,
	terrain []models.Terrain,
	centroids [][2]float64,
	overfull int,
) (int, int, bool) {
	n := len(terrain)
	bestFirst, bestSecond := -1, -1
	bestDifficult := false
	bestDistance := 0.0
	for other := 0; other < n; other++ {
		if other == overfull || !matrixHasEdge(matrix, n, overfull, other) ||
			!canDowngradeArc(matrix, n, overfull, other) {
			continue
		}
		first, second := orderedPair(overfull, other)
		difficult := difficultFrontier(terrain[first], terrain[second])
		distance := centroidDistanceSquared(centroids, first, second)
		if bestFirst < 0 ||
			(difficult && !bestDifficult) ||
			(difficult == bestDifficult && (distance > bestDistance ||
				(distance == bestDistance && (first < bestFirst || (first == bestFirst && second < bestSecond))))) {
			bestFirst, bestSecond = first, second
			bestDifficult = difficult
			bestDistance = distance
		}
	}
	return bestFirst, bestSecond, bestFirst >= 0
}

func canDowngradeArc(matrix []bool, n, first, second int) bool {
	if !matrixHasEdge(matrix, n, first, second) {
		return false
	}
	degree := degreesFromMatrix(matrix, n)
	if degree[first] < 3 || degree[second] < 3 {
		return false
	}
	matrixClearEdge(matrix, n, first, second)
	connected := len(componentsFromMatrix(matrix, n)) == 1
	matrixSetEdge(matrix, n, first, second)
	return connected
}

func differenceEdges(all, included [][2]int) [][2]int {
	result := make([][2]int, 0, len(all))
	index := 0
	for _, edge := range all {
		for index < len(included) && edgeLess(included[index], edge) {
			index++
		}
		if index >= len(included) || included[index] != edge {
			result = append(result, edge)
		}
	}
	return result
}

func mergeEdges(first, second [][2]int) [][2]int {
	merged := make([][2]int, 0, len(first)+len(second))
	left, right := 0, 0
	for left < len(first) && right < len(second) {
		switch {
		case first[left] == second[right]:
			merged = append(merged, first[left])
			left++
			right++
		case edgeLess(first[left], second[right]):
			merged = append(merged, first[left])
			left++
		default:
			merged = append(merged, second[right])
			right++
		}
	}
	merged = append(merged, first[left:]...)
	merged = append(merged, second[right:]...)
	return merged
}

func edgeLess(first, second [2]int) bool {
	return first[0] < second[0] || (first[0] == second[0] && first[1] < second[1])
}

func components(edges [][2]int, n int) [][]int {
	return componentsFromMatrix(edgeMatrix(n, edges), n)
}

func componentsFromMatrix(matrix []bool, n int) [][]int {
	seen := make([]bool, n)
	sets := make([][]int, 0)
	for start := 0; start < n; start++ {
		if seen[start] {
			continue
		}
		seen[start] = true
		queue := []int{start}
		set := make([]int, 0)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			set = append(set, current)
			for candidate := 0; candidate < n; candidate++ {
				if !seen[candidate] && matrixHasEdge(matrix, n, current, candidate) {
					seen[candidate] = true
					queue = append(queue, candidate)
				}
			}
		}
		sets = append(sets, set)
	}
	return sets
}

func degrees(edges [][2]int, n int) []int {
	return degreesFromMatrix(edgeMatrix(n, edges), n)
}

func degreesFromMatrix(matrix []bool, n int) []int {
	degree := make([]int, n)
	for first := 0; first < n; first++ {
		for second := first + 1; second < n; second++ {
			if matrixHasEdge(matrix, n, first, second) {
				degree[first]++
				degree[second]++
			}
		}
	}
	return degree
}

func orderedPair(first, second int) (int, int) {
	if first > second {
		return second, first
	}
	return first, second
}

func centroidDistanceSquared(centroids [][2]float64, first, second int) float64 {
	return squaredDistance(
		centroids[first][0],
		centroids[first][1],
		centroids[second][0],
		centroids[second][1],
	)
}

func validateGraph(edges [][2]int, terrain []models.Terrain, n int) error {
	if len(terrain) != n {
		return fmt.Errorf("mapgen: terrain count does not match graph size")
	}
	if len(components(edges, n)) != 1 {
		return fmt.Errorf("mapgen: final graph is disconnected")
	}
	for index, degree := range degrees(edges, n) {
		maximum := maxDegree(terrain[index])
		if maximum == 0 {
			return fmt.Errorf("mapgen: territory %s has invalid terrain %q", territoryID(index), terrain[index])
		}
		if degree < 2 {
			return fmt.Errorf("mapgen: territory %s has degree %d, want at least 2", territoryID(index), degree)
		}
		if degree > maximum {
			return fmt.Errorf("mapgen: territory %s has degree %d, want at most %d", territoryID(index), degree, maximum)
		}
	}
	return nil
}

func edgeMatrix(n int, edges [][2]int) []bool {
	matrix := make([]bool, n*n)
	for _, edge := range edges {
		if edge[0] < 0 || edge[0] >= n || edge[1] < 0 || edge[1] >= n || edge[0] == edge[1] {
			continue
		}
		matrixSetEdge(matrix, n, edge[0], edge[1])
	}
	return matrix
}

func matrixHasEdge(matrix []bool, n, first, second int) bool {
	return matrix[first*n+second]
}

func matrixSetEdge(matrix []bool, n, first, second int) {
	matrix[first*n+second] = true
	matrix[second*n+first] = true
}

func matrixClearEdge(matrix []bool, n, first, second int) {
	matrix[first*n+second] = false
	matrix[second*n+first] = false
}

func edgesFromMatrix(matrix []bool, n int) [][2]int {
	edges := make([][2]int, 0)
	for first := 0; first < n; first++ {
		for second := first + 1; second < n; second++ {
			if matrixHasEdge(matrix, n, first, second) {
				edges = append(edges, [2]int{first, second})
			}
		}
	}
	return edges
}
