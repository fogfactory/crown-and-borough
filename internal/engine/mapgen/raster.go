package mapgen

import "math"

const minSharedEdges = 3

type gridPoint struct {
	x int
	y int
}

type segment struct {
	from gridPoint
	to   gridPoint
}

type boundarySegment struct {
	segment
	neighbor int
}

type boundaryLoop struct {
	vertices  []gridPoint
	neighbors []int
}

// assignRaster assigns every fixed-grid cell to its nearest site. Equal
// distances resolve to the lower site index because sites are scanned in order.
func assignRaster(sites []point, cfg Config) []int {
	grid := make([]int, gridW*gridH)
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			cellX := (float64(x) + 0.5) * float64(cfg.Width) / float64(gridW)
			cellY := (float64(y) + 0.5) * float64(cfg.Height) / float64(gridH)
			bestSite := 0
			bestDistance := math.Inf(1)
			for siteIndex, site := range sites {
				distance := squaredDistance(cellX, cellY, site.x, site.y)
				if distance < bestDistance {
					bestDistance = distance
					bestSite = siteIndex
				}
			}
			grid[y*gridW+x] = bestSite
		}
	}
	return grid
}

func rasterHasEveryRegion(grid []int, interiorCount int) bool {
	if len(grid) != gridW*gridH || interiorCount == 0 {
		return false
	}
	seen := make([]bool, interiorCount)
	for _, owner := range grid {
		if owner < 0 {
			return false
		}
		if owner < interiorCount {
			seen[owner] = true
		}
	}
	for _, present := range seen {
		if !present {
			return false
		}
	}
	return true
}

// extractBoundaryLoops preserves the neighboring owner for every directed
// grid edge. The direction keeps the owner region on the right side, matching
// chainBoundaryLoops' right-hand traversal rule.
func extractBoundaryLoops(grid []int, siteCount int) [][]boundaryLoop {
	if len(grid) != gridW*gridH || siteCount == 0 {
		return nil
	}
	segmentsByOwner := make([][]boundarySegment, siteCount)
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			owner := grid[y*gridW+x]
			if owner < 0 || owner >= siteCount {
				return nil
			}
			if y == 0 || grid[(y-1)*gridW+x] != owner {
				neighbor := -1
				if y > 0 {
					neighbor = grid[(y-1)*gridW+x]
				}
				segmentsByOwner[owner] = append(segmentsByOwner[owner], boundarySegment{
					segment:  segment{from: gridPoint{x: x, y: y}, to: gridPoint{x: x + 1, y: y}},
					neighbor: neighbor,
				})
			}
			if x == gridW-1 || grid[y*gridW+x+1] != owner {
				neighbor := -1
				if x+1 < gridW {
					neighbor = grid[y*gridW+x+1]
				}
				segmentsByOwner[owner] = append(segmentsByOwner[owner], boundarySegment{
					segment:  segment{from: gridPoint{x: x + 1, y: y}, to: gridPoint{x: x + 1, y: y + 1}},
					neighbor: neighbor,
				})
			}
			if y == gridH-1 || grid[(y+1)*gridW+x] != owner {
				neighbor := -1
				if y+1 < gridH {
					neighbor = grid[(y+1)*gridW+x]
				}
				segmentsByOwner[owner] = append(segmentsByOwner[owner], boundarySegment{
					segment:  segment{from: gridPoint{x: x + 1, y: y + 1}, to: gridPoint{x: x, y: y + 1}},
					neighbor: neighbor,
				})
			}
			if x == 0 || grid[y*gridW+x-1] != owner {
				neighbor := -1
				if x > 0 {
					neighbor = grid[y*gridW+x-1]
				}
				segmentsByOwner[owner] = append(segmentsByOwner[owner], boundarySegment{
					segment:  segment{from: gridPoint{x: x, y: y + 1}, to: gridPoint{x: x, y: y}},
					neighbor: neighbor,
				})
			}
		}
	}

	loopsByOwner := make([][]boundaryLoop, siteCount)
	for owner, segments := range segmentsByOwner {
		loopsByOwner[owner] = chainBoundaryLoops(segments)
	}
	return loopsByOwner
}

func chainBoundaryLoops(segments []boundarySegment) []boundaryLoop {
	outgoing := make([][]int, (gridW+1)*(gridH+1))
	for index, segment := range segments {
		outgoing[gridVertexIndex(segment.from)] = append(outgoing[gridVertexIndex(segment.from)], index)
	}

	used := make([]bool, len(segments))
	loops := make([]boundaryLoop, 0)
	for start := range segments {
		if used[start] {
			continue
		}
		loop, closed := traceBoundaryLoop(start, segments, outgoing, used)
		if closed && len(loop.vertices) >= 3 {
			loops = append(loops, loop)
		}
	}
	return loops
}

func traceBoundaryLoop(start int, segments []boundarySegment, outgoing [][]int, used []bool) (boundaryLoop, bool) {
	loop := boundaryLoop{
		vertices:  make([]gridPoint, 0),
		neighbors: make([]int, 0),
	}
	startVertex := segments[start].from
	current := start
	for steps := 0; steps <= len(segments); steps++ {
		if used[current] {
			return loop, false
		}
		currentSegment := segments[current]
		used[current] = true
		loop.vertices = append(loop.vertices, currentSegment.from)
		loop.neighbors = append(loop.neighbors, currentSegment.neighbor)
		if currentSegment.to == startVertex {
			return loop, true
		}

		next := rightHandBoundaryNext(
			currentSegment.segment,
			outgoing[gridVertexIndex(currentSegment.to)],
			segments,
			used,
		)
		if next < 0 {
			return loop, false
		}
		current = next
	}
	return loop, false
}

func rightHandBoundaryNext(current segment, candidates []int, segments []boundarySegment, used []bool) int {
	direction := segmentDirection(current)
	priorities := [4]int{
		(direction + 1) % 4,
		direction,
		(direction + 3) % 4,
		(direction + 2) % 4,
	}
	for _, wanted := range priorities {
		for _, candidate := range candidates {
			if !used[candidate] && segmentDirection(segments[candidate].segment) == wanted {
				return candidate
			}
		}
	}
	return -1
}

func largestBoundaryLoop(loops []boundaryLoop) boundaryLoop {
	var largest boundaryLoop
	var largestArea int64
	for _, loop := range loops {
		area := gridPolygonAreaTwice(loop.vertices)
		if area < 0 {
			area = -area
		}
		if area > largestArea {
			largest = loop
			largestArea = area
		}
	}
	return largest
}

// segmentDirection uses the clockwise screen-coordinate order east, south,
// west, north.
func segmentDirection(segment segment) int {
	dx := segment.to.x - segment.from.x
	dy := segment.to.y - segment.from.y
	switch {
	case dx == 1:
		return 0
	case dy == 1:
		return 1
	case dx == -1:
		return 2
	default:
		return 3
	}
}

func gridVertexIndex(point gridPoint) int {
	return point.y*(gridW+1) + point.x
}

func gridPolygonAreaTwice(points []gridPoint) int64 {
	if len(points) < 3 {
		return 0
	}
	var area int64
	for i, point := range points {
		next := points[(i+1)%len(points)]
		area += int64(point.x)*int64(next.y) - int64(next.x)*int64(point.y)
	}
	return area
}

func removeConsecutiveDuplicates(points [][2]int) [][2]int {
	if len(points) == 0 {
		return points
	}
	unique := make([][2]int, 0, len(points))
	for _, point := range points {
		if len(unique) == 0 || unique[len(unique)-1] != point {
			unique = append(unique, point)
		}
	}
	if len(unique) > 1 && unique[0] == unique[len(unique)-1] {
		unique = unique[:len(unique)-1]
	}
	return unique
}

func polygonAreaTwice(points [][2]int) int64 {
	if len(points) < 3 {
		return 0
	}
	var area int64
	for i, point := range points {
		next := points[(i+1)%len(points)]
		area += int64(point[0])*int64(next[1]) - int64(next[0])*int64(point[1])
	}
	return area
}

func polygonCentroid(points [][2]int) [2]float64 {
	if len(points) == 0 {
		return [2]float64{}
	}
	var x, y float64
	for _, point := range points {
		x += float64(point[0])
		y += float64(point[1])
	}
	return [2]float64{x / float64(len(points)), y / float64(len(points))}
}

// extractAdjacency reports sorted interior-site pairs that share at least
// minSharedEdges grid edges. The threshold removes point-only contacts at
// raster junctions; frame-site contacts deliberately do not become map arcs.
func extractAdjacency(grid []int, interiorCount int) [][2]int {
	if len(grid) != gridW*gridH || interiorCount == 0 {
		return nil
	}
	shared := make(map[[2]int]int)
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			owner := grid[y*gridW+x]
			if x+1 < gridW {
				addInteriorSharedEdge(shared, owner, grid[y*gridW+x+1], interiorCount)
			}
			if y+1 < gridH {
				addInteriorSharedEdge(shared, owner, grid[(y+1)*gridW+x], interiorCount)
			}
		}
	}

	edges := make([][2]int, 0)
	for first := 0; first < interiorCount; first++ {
		for second := first + 1; second < interiorCount; second++ {
			if shared[[2]int{first, second}] >= minSharedEdges {
				edges = append(edges, [2]int{first, second})
			}
		}
	}
	return edges
}

func addInteriorSharedEdge(shared map[[2]int]int, first, second, interiorCount int) {
	if first == second || first < 0 || second < 0 || first >= interiorCount || second >= interiorCount {
		return
	}
	if first > second {
		first, second = second, first
	}
	shared[[2]int{first, second}]++
}

func squaredDistance(firstX, firstY, secondX, secondY float64) float64 {
	dx := firstX - secondX
	dy := firstY - secondY
	return dx*dx + dy*dy
}
