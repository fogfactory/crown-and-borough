package mapgen

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

const (
	// Frontier simplification happens in raster-cell space so it remains stable
	// when the generation viewport changes size.
	douglasPeuckerTolerance = 3.5
	minimumChainSegment     = 2.0
	jitterRatio             = 0.0075
)

type frontierGeometry struct {
	polygons  [][][2]int
	centroids [][2]float64
	chains    []frontierChain
	padding   int
	width     int
	height    int
}

type frontierChain struct {
	owners    [2]int
	raw       []gridPoint
	grid      []gridPoint
	points    [][2]int
	junctions []bool
}

type frontierKey struct {
	firstOwner  int
	secondOwner int
	firstPoint  gridPoint
	secondPoint gridPoint
}

type frontierReference struct {
	chain   int
	reverse bool
}

type rawFrontierChain struct {
	owner    int
	neighbor int
	points   []gridPoint
}

// extractInteriorFrontiers builds each shared raster boundary once, simplifies
// and jitters that canonical chain once, then reuses it for both neighboring
// output polygons. Frame sites take part in geometry but are never emitted.
func extractInteriorFrontiers(
	grid []int,
	siteCount, interiorCount int,
	cfg Config,
	seed string,
) (frontierGeometry, error) {
	// Try the intended light noise first. Simplified chains can get very close
	// near a raster junction, so retry deterministically with smaller noise
	// rather than emitting a self-crossing polygon.
	for _, ratio := range [...]float64{jitterRatio, jitterRatio / 2, jitterRatio / 4, jitterRatio / 8, 0} {
		geometry, err := extractInteriorFrontiersWithJitter(grid, siteCount, interiorCount, cfg, seed, ratio)
		if err != nil {
			return frontierGeometry{}, err
		}
		if simplePolygons(geometry.polygons) {
			return geometry, nil
		}
	}
	return frontierGeometry{}, fmt.Errorf("mapgen: jittered frontier geometry is self-intersecting")
}

func extractInteriorFrontiersWithJitter(
	grid []int,
	siteCount, interiorCount int,
	cfg Config,
	seed string,
	ratio float64,
) (frontierGeometry, error) {
	loopsByOwner := extractBoundaryLoops(grid, siteCount)
	if loopsByOwner == nil || len(loopsByOwner) != siteCount {
		return frontierGeometry{}, fmt.Errorf("mapgen: could not extract raster boundaries")
	}

	junctions := rasterJunctions(grid)
	chains := make([]frontierChain, 0)
	chainsByKey := make(map[frontierKey][]int)
	references := make([][]frontierReference, interiorCount)

	for owner := 0; owner < interiorCount; owner++ {
		loop := largestBoundaryLoop(loopsByOwner[owner])
		if len(loop.vertices) < 3 || len(loop.vertices) != len(loop.neighbors) {
			return frontierGeometry{}, fmt.Errorf("mapgen: interior site %d has no closed boundary", owner)
		}

		for _, raw := range splitBoundaryLoop(owner, loop, junctions) {
			if raw.neighbor < 0 {
				return frontierGeometry{}, fmt.Errorf("mapgen: interior site %d reaches the viewport border", owner)
			}
			key := canonicalFrontierKey(raw.owner, raw.neighbor, raw.points[0], raw.points[len(raw.points)-1])
			chainIndex, reverse, found := matchingFrontierChain(chainsByKey[key], chains, raw.points)
			if !found {
				chain := buildFrontierChain(seed, cfg, raw.owner, raw.neighbor, raw.points, junctions, ratio)
				chainIndex = len(chains)
				chains = append(chains, chain)
				chainsByKey[key] = append(chainsByKey[key], chainIndex)
			}
			references[owner] = append(references[owner], frontierReference{
				chain:   chainIndex,
				reverse: reverse,
			})
		}
	}

	polygons, err := assembleFrontierPolygons(references, chains)
	if err != nil {
		return frontierGeometry{}, err
	}
	padding, width, height := reanchorPolygons(polygons, cfg)
	centroids := make([][2]float64, len(polygons))
	for i, polygon := range polygons {
		centroids[i] = polygonCentroid(polygon)
	}
	return frontierGeometry{
		polygons:  polygons,
		centroids: centroids,
		chains:    chains,
		padding:   padding,
		width:     width,
		height:    height,
	}, nil
}

func simplePolygons(polygons [][][2]int) bool {
	for _, polygon := range polygons {
		if len(polygon) < 3 || !isSimplePolygon(polygon) {
			return false
		}
	}
	return true
}

func rasterJunctions(grid []int) map[gridPoint]bool {
	junctions := make(map[gridPoint]bool)
	if len(grid) != gridW*gridH {
		return junctions
	}
	for y := 0; y <= gridH; y++ {
		for x := 0; x <= gridW; x++ {
			owners := [4]int{}
			ownerCount := 0
			for cellY := y - 1; cellY <= y; cellY++ {
				for cellX := x - 1; cellX <= x; cellX++ {
					if cellX < 0 || cellX >= gridW || cellY < 0 || cellY >= gridH {
						continue
					}
					owner := grid[cellY*gridW+cellX]
					seen := false
					for i := 0; i < ownerCount; i++ {
						if owners[i] == owner {
							seen = true
							break
						}
					}
					if !seen {
						owners[ownerCount] = owner
						ownerCount++
					}
				}
			}
			if ownerCount >= 3 {
				junctions[gridPoint{x: x, y: y}] = true
			}
		}
	}
	return junctions
}

func splitBoundaryLoop(owner int, loop boundaryLoop, junctions map[gridPoint]bool) []rawFrontierChain {
	n := len(loop.vertices)
	if n == 0 {
		return nil
	}
	breakAfter := make([]bool, n)
	hasBreak := false
	for i := range loop.vertices {
		next := (i + 1) % n
		if junctions[loop.vertices[next]] || loop.neighbors[i] != loop.neighbors[next] {
			breakAfter[i] = true
			hasBreak = true
		}
	}
	if !hasBreak {
		return splitClosedBoundaryLoop(owner, loop)
	}

	start := 0
	for i, value := range breakAfter {
		if value {
			start = (i + 1) % n
			break
		}
	}

	chains := make([]rawFrontierChain, 0)
	points := []gridPoint{loop.vertices[start]}
	neighbor := loop.neighbors[start]
	for step := 0; step < n; step++ {
		index := (start + step) % n
		next := (index + 1) % n
		points = append(points, loop.vertices[next])
		if !breakAfter[index] {
			continue
		}
		chains = append(chains, rawFrontierChain{
			owner:    owner,
			neighbor: neighbor,
			points:   points,
		})
		if step+1 < n {
			points = []gridPoint{loop.vertices[next]}
			neighbor = loop.neighbors[next]
		}
	}
	return chains
}

// A closed two-owner boundary is not expected with the minimum supported
// population, but splitting it deterministically keeps the canonical-chain
// representation valid for malformed or future small configurations.
func splitClosedBoundaryLoop(owner int, loop boundaryLoop) []rawFrontierChain {
	n := len(loop.vertices)
	first := 0
	for i := 1; i < n; i++ {
		if gridPointLess(loop.vertices[i], loop.vertices[first]) {
			first = i
		}
	}
	second := (first + n/2) % n
	if second == first {
		second = (first + 1) % n
	}
	return []rawFrontierChain{
		{
			owner:    owner,
			neighbor: loop.neighbors[first],
			points:   loopPath(loop.vertices, first, second),
		},
		{
			owner:    owner,
			neighbor: loop.neighbors[second],
			points:   loopPath(loop.vertices, second, first),
		},
	}
}

func loopPath(vertices []gridPoint, start, end int) []gridPoint {
	points := []gridPoint{vertices[start]}
	for index := start; index != end; {
		index = (index + 1) % len(vertices)
		points = append(points, vertices[index])
	}
	return points
}

func canonicalFrontierKey(firstOwner, secondOwner int, firstPoint, secondPoint gridPoint) frontierKey {
	if firstOwner > secondOwner {
		firstOwner, secondOwner = secondOwner, firstOwner
	}
	if gridPointLess(secondPoint, firstPoint) {
		firstPoint, secondPoint = secondPoint, firstPoint
	}
	return frontierKey{
		firstOwner:  firstOwner,
		secondOwner: secondOwner,
		firstPoint:  firstPoint,
		secondPoint: secondPoint,
	}
}

func gridPointLess(first, second gridPoint) bool {
	return first.y < second.y || (first.y == second.y && first.x < second.x)
}

func matchingFrontierChain(candidates []int, chains []frontierChain, points []gridPoint) (int, bool, bool) {
	for _, candidate := range candidates {
		if sameGridPath(chains[candidate].raw, points) {
			return candidate, false, true
		}
		if reversedGridPath(chains[candidate].raw, points) {
			return candidate, true, true
		}
	}
	return 0, false, false
}

func sameGridPath(first, second []gridPoint) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}

func reversedGridPath(first, second []gridPoint) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[len(second)-1-i] {
			return false
		}
	}
	return true
}

func buildFrontierChain(
	seed string,
	cfg Config,
	firstOwner, secondOwner int,
	raw []gridPoint,
	junctions map[gridPoint]bool,
	ratio float64,
) frontierChain {
	grid := simplifyFrontierChain(raw, junctions)
	points := make([][2]int, len(grid))
	flags := make([]bool, len(grid))
	for i, point := range grid {
		points[i] = jitteredViewportPoint(seed, cfg, point, ratio)
		flags[i] = junctions[point]
	}
	firstOwner, secondOwner = orderedPair(firstOwner, secondOwner)
	return frontierChain{
		owners:    [2]int{firstOwner, secondOwner},
		raw:       append([]gridPoint(nil), raw...),
		grid:      grid,
		points:    points,
		junctions: flags,
	}
}

func simplifyFrontierChain(points []gridPoint, junctions map[gridPoint]bool) []gridPoint {
	points = douglasPeucker(points, douglasPeuckerTolerance)
	return removeShortChainSegments(points, junctions)
}

func douglasPeucker(points []gridPoint, tolerance float64) []gridPoint {
	if len(points) <= 2 {
		return append([]gridPoint(nil), points...)
	}
	keep := make([]bool, len(points))
	keep[0] = true
	keep[len(points)-1] = true
	var simplify func(int, int)
	simplify = func(first, last int) {
		if last-first <= 1 {
			return
		}
		maxIndex := -1
		maxDistance := -1.0
		for index := first + 1; index < last; index++ {
			distance := gridPointSegmentDistance(points[index], points[first], points[last])
			// Scanning in ascending order and only replacing strictly larger
			// values gives the lower source index deterministic tie-breaking.
			if distance > maxDistance {
				maxDistance = distance
				maxIndex = index
			}
		}
		if maxDistance > tolerance {
			keep[maxIndex] = true
			simplify(first, maxIndex)
			simplify(maxIndex, last)
		}
	}
	simplify(0, len(points)-1)

	simplified := make([]gridPoint, 0, len(points))
	for index, point := range points {
		if keep[index] {
			simplified = append(simplified, point)
		}
	}
	return simplified
}

func gridPointSegmentDistance(point, first, last gridPoint) float64 {
	dx := float64(last.x - first.x)
	dy := float64(last.y - first.y)
	if dx == 0 && dy == 0 {
		return math.Hypot(float64(point.x-first.x), float64(point.y-first.y))
	}
	factor := (float64(point.x-first.x)*dx + float64(point.y-first.y)*dy) / (dx*dx + dy*dy)
	projectedX := float64(first.x) + factor*dx
	projectedY := float64(first.y) + factor*dy
	return math.Hypot(float64(point.x)-projectedX, float64(point.y)-projectedY)
}

func removeShortChainSegments(points []gridPoint, junctions map[gridPoint]bool) []gridPoint {
	if len(points) <= 2 {
		return points
	}
	filtered := make([]gridPoint, 0, len(points))
	filtered = append(filtered, points[0])
	for index := 1; index < len(points)-1; index++ {
		previous := filtered[len(filtered)-1]
		current := points[index]
		if gridPointDistance(previous, current) < minimumChainSegment &&
			!junctions[previous] && !junctions[current] {
			continue
		}
		filtered = append(filtered, current)
	}
	return append(filtered, points[len(points)-1])
}

func gridPointDistance(first, second gridPoint) float64 {
	return math.Hypot(float64(second.x-first.x), float64(second.y-first.y))
}

func jitteredViewportPoint(seed string, cfg Config, point gridPoint, ratio float64) [2]int {
	baseX := float64(point.x) * float64(cfg.Width) / float64(gridW)
	baseY := float64(point.y) * float64(cfg.Height) / float64(gridH)
	amplitude := frontierJitterAmplitude(cfg, ratio)
	if amplitude == 0 {
		return [2]int{int(math.Round(baseX)), int(math.Round(baseY))}
	}

	digest := sha256.Sum256([]byte(seed + "|jitter|" + strconv.Itoa(point.x) + "|" + strconv.Itoa(point.y)))
	angle := 2 * math.Pi * unitInterval(binary.BigEndian.Uint64(digest[:8]))
	radius := amplitude * math.Sqrt(unitInterval(binary.BigEndian.Uint64(digest[8:16])))
	return [2]int{
		int(math.Round(baseX + radius*math.Cos(angle))),
		int(math.Round(baseY + radius*math.Sin(angle))),
	}
}

func unitInterval(value uint64) float64 {
	return float64(value) / float64(^uint64(0))
}

func frontierJitterAmplitude(cfg Config, ratio float64) float64 {
	// A 100x100 viewport has sub-pixel raster cells after scaling. Keeping its
	// jitter at zero avoids integer-rounding collapses while remaining within
	// the documented maximum; normal viewports begin at 0.75% of the diagonal
	// and use the deterministic fallback ratio when necessary.
	if math.Min(float64(cfg.Width)/gridW, float64(cfg.Height)/gridH) < 1 {
		return 0
	}
	return ratio * math.Hypot(float64(cfg.Width), float64(cfg.Height))
}

func assembleFrontierPolygons(references [][]frontierReference, chains []frontierChain) ([][][2]int, error) {
	polygons := make([][][2]int, len(references))
	for owner, ownerReferences := range references {
		if len(ownerReferences) == 0 {
			return nil, fmt.Errorf("mapgen: interior site %d has no frontier chains", owner)
		}
		points := make([][2]int, 0)
		for _, reference := range ownerReferences {
			chain := chains[reference.chain].points
			for i := 0; i < len(chain); i++ {
				index := i
				if reference.reverse {
					index = len(chain) - 1 - i
				}
				point := chain[index]
				if len(points) == 0 {
					points = append(points, point)
					continue
				}
				if i == 0 {
					if points[len(points)-1] != point {
						return nil, fmt.Errorf("mapgen: frontier chains do not meet for interior site %d", owner)
					}
					continue
				}
				points = append(points, point)
			}
		}
		if len(points) > 1 && points[0] == points[len(points)-1] {
			points = points[:len(points)-1]
		}
		points = removeConsecutiveDuplicates(points)
		if polygonAreaTwice(points) < 0 {
			reversePolygon(points)
		}
		polygons[owner] = points
	}
	return polygons, nil
}

func reversePolygon(points [][2]int) {
	for left, right := 0, len(points)-1; left < right; left, right = left+1, right-1 {
		points[left], points[right] = points[right], points[left]
	}
}

func mapPadding(cfg Config) int {
	return int(math.Round(0.02 * math.Hypot(float64(cfg.Width), float64(cfg.Height))))
}

// reanchorPolygons translates the generated interior bounding box so its
// minimum x/y equals padding. The returned dimensions retain the same padding
// on the right and bottom without adding an explicit size to map.json.
func reanchorPolygons(polygons [][][2]int, cfg Config) (padding, width, height int) {
	padding = mapPadding(cfg)
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := math.MinInt, math.MinInt
	for _, polygon := range polygons {
		for _, point := range polygon {
			minX = min(minX, point[0])
			minY = min(minY, point[1])
			maxX = max(maxX, point[0])
			maxY = max(maxY, point[1])
		}
	}
	if minX == math.MaxInt {
		return padding, 0, 0
	}
	deltaX := padding - minX
	deltaY := padding - minY
	for polygonIndex := range polygons {
		for pointIndex := range polygons[polygonIndex] {
			polygons[polygonIndex][pointIndex][0] += deltaX
			polygons[polygonIndex][pointIndex][1] += deltaY
		}
	}
	return padding, maxX + deltaX + padding, maxY + deltaY + padding
}
