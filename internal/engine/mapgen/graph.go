package mapgen

import (
	"fmt"
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// filterFrontiers removes selected geometric borders according to terrain.
// Its input and output edge lists are sorted by their site indexes.
func filterFrontiers(rng *rand.Rand, edges [][2]int, terrains []models.Terrain) [][2]int {
	passable := make([][2]int, 0, len(edges))
	for _, edge := range edges {
		removalChance := frontierRemovalChance(terrains[edge[0]], terrains[edge[1]])
		if rng.Float64() >= removalChance {
			passable = append(passable, edge)
		}
	}
	return passable
}

func frontierRemovalChance(first, second models.Terrain) float64 {
	if first == models.TerrainPlain && second == models.TerrainPlain {
		return 0
	}
	if (first == models.TerrainMountain && second == models.TerrainMountain) ||
		(first == models.TerrainMountain && second == models.TerrainSwamp) ||
		(first == models.TerrainSwamp && second == models.TerrainMountain) {
		return 0.75
	}
	return 0.15
}

// repairGraph adds deterministic routes until the movement graph is connected
// and every territory has degree at least two.
func repairGraph(edges [][2]int, centroids [][2]float64, n int) [][2]int {
	return repairGraphWithGeometry(edges, nil, centroids, n)
}

// repairGraphWithGeometry keeps geometric borders separate so routes prefer
// genuinely non-geometric pairs.
func repairGraphWithGeometry(edges, geometricEdges [][2]int, centroids [][2]float64, n int) [][2]int {
	matrix := edgeMatrix(n, edges)
	geometric := edgeMatrix(n, geometricEdges)

	for {
		sets := componentsFromMatrix(matrix, n)
		if len(sets) <= 1 {
			break
		}
		first, second, found := closestAcrossComponents(sets, matrix, geometric, centroids, n, true)
		if !found {
			// A complete geometric graph leaves no non-geometric route candidate.
			// Preserving the connectivity invariant is more important in that case.
			first, second, found = closestAcrossComponents(sets, matrix, geometric, centroids, n, false)
		}
		if !found {
			break
		}
		matrixSetEdge(matrix, n, first, second)
	}

	for {
		degree := degreesFromMatrix(matrix, n)
		territory := -1
		for i, value := range degree {
			if value < 2 {
				territory = i
				break
			}
		}
		if territory < 0 {
			break
		}

		target, found := closestRouteTarget(territory, matrix, geometric, centroids, n, true)
		if !found {
			target, found = closestRouteTarget(territory, matrix, geometric, centroids, n, false)
		}
		if !found {
			break
		}
		matrixSetEdge(matrix, n, territory, target)
	}

	return edgesFromMatrix(matrix, n)
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

func closestAcrossComponents(
	sets [][]int,
	matrix, geometric []bool,
	centroids [][2]float64,
	n int,
	nonGeometricOnly bool,
) (int, int, bool) {
	firstBest, secondBest := -1, -1
	var bestDistance float64
	found := false
	for firstSet := 0; firstSet < len(sets); firstSet++ {
		for secondSet := firstSet + 1; secondSet < len(sets); secondSet++ {
			for _, first := range sets[firstSet] {
				for _, second := range sets[secondSet] {
					if matrixHasEdge(matrix, n, first, second) ||
						(nonGeometricOnly && matrixHasEdge(geometric, n, first, second)) {
						continue
					}
					first, second = orderedPair(first, second)
					distance := centroidDistanceSquared(centroids, first, second)
					if !found || distance < bestDistance ||
						(distance == bestDistance && (first < firstBest || (first == firstBest && second < secondBest))) {
						firstBest, secondBest = first, second
						bestDistance = distance
						found = true
					}
				}
			}
		}
	}
	return firstBest, secondBest, found
}

func closestRouteTarget(
	from int,
	matrix, geometric []bool,
	centroids [][2]float64,
	n int,
	nonGeometricOnly bool,
) (int, bool) {
	bestTarget := -1
	var bestDistance float64
	for target := 0; target < n; target++ {
		if target == from || matrixHasEdge(matrix, n, from, target) ||
			(nonGeometricOnly && matrixHasEdge(geometric, n, from, target)) {
			continue
		}
		distance := centroidDistanceSquared(centroids, from, target)
		if bestTarget < 0 || distance < bestDistance || (distance == bestDistance && target < bestTarget) {
			bestTarget = target
			bestDistance = distance
		}
	}
	return bestTarget, bestTarget >= 0
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

func validateGraph(edges [][2]int, n int) error {
	if len(components(edges, n)) != 1 {
		return fmt.Errorf("mapgen: final graph is disconnected")
	}
	for index, degree := range degrees(edges, n) {
		if degree < 2 {
			return fmt.Errorf("mapgen: territory %s has degree %d, want at least 2", territoryID(index), degree)
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
