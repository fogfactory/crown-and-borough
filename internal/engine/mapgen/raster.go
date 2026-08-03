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

func rasterHasEveryRegion(grid []int, n int) bool {
	if len(grid) != gridW*gridH || n == 0 {
		return false
	}
	seen := make([]bool, n)
	for _, owner := range grid {
		if owner < 0 || owner >= n {
			return false
		}
		seen[owner] = true
	}
	for _, present := range seen {
		if !present {
			return false
		}
	}
	return true
}

// extractPolygons chains grid boundary segments clockwise in screen
// coordinates. At ambiguous vertices it takes the rightmost available turn,
// which keeps the owner region on the right side of the traversal.
func extractPolygons(grid []int, sites []point, cfg Config) ([][][2]int, [][2]float64) {
	polygons := make([][][2]int, len(sites))
	centroids := make([][2]float64, len(sites))
	if len(grid) != gridW*gridH {
		return polygons, centroids
	}

	segmentsByOwner := make([][]segment, len(sites))
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			owner := grid[y*gridW+x]
			if owner < 0 || owner >= len(sites) {
				return polygons, centroids
			}
			if y == 0 || grid[(y-1)*gridW+x] != owner {
				segmentsByOwner[owner] = append(segmentsByOwner[owner], segment{
					from: gridPoint{x: x, y: y},
					to:   gridPoint{x: x + 1, y: y},
				})
			}
			if x == gridW-1 || grid[y*gridW+x+1] != owner {
				segmentsByOwner[owner] = append(segmentsByOwner[owner], segment{
					from: gridPoint{x: x + 1, y: y},
					to:   gridPoint{x: x + 1, y: y + 1},
				})
			}
			if y == gridH-1 || grid[(y+1)*gridW+x] != owner {
				segmentsByOwner[owner] = append(segmentsByOwner[owner], segment{
					from: gridPoint{x: x + 1, y: y + 1},
					to:   gridPoint{x: x, y: y + 1},
				})
			}
			if x == 0 || grid[y*gridW+x-1] != owner {
				segmentsByOwner[owner] = append(segmentsByOwner[owner], segment{
					from: gridPoint{x: x, y: y + 1},
					to:   gridPoint{x: x, y: y},
				})
			}
		}
	}

	for owner, segments := range segmentsByOwner {
		loop := largestLoop(chainLoops(segments))
		polygons[owner] = scaleAndSimplify(loop, cfg)
		centroids[owner] = polygonCentroid(polygons[owner])
	}
	return polygons, centroids
}

func chainLoops(segments []segment) [][]gridPoint {
	outgoing := make([][]int, (gridW+1)*(gridH+1))
	for index, segment := range segments {
		outgoing[gridVertexIndex(segment.from)] = append(outgoing[gridVertexIndex(segment.from)], index)
	}

	used := make([]bool, len(segments))
	loops := make([][]gridPoint, 0)
	for start := range segments {
		if used[start] {
			continue
		}
		loop, closed := traceLoop(start, segments, outgoing, used)
		if closed && len(loop) >= 3 {
			loops = append(loops, loop)
		}
	}
	return loops
}

func traceLoop(start int, segments []segment, outgoing [][]int, used []bool) ([]gridPoint, bool) {
	loop := make([]gridPoint, 0)
	startVertex := segments[start].from
	current := start
	for steps := 0; steps <= len(segments); steps++ {
		if used[current] {
			return loop, false
		}
		currentSegment := segments[current]
		used[current] = true
		loop = append(loop, currentSegment.from)
		if currentSegment.to == startVertex {
			return loop, true
		}

		next := rightHandNext(
			currentSegment,
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

func rightHandNext(current segment, candidates []int, segments []segment, used []bool) int {
	direction := segmentDirection(current)
	priorities := [4]int{
		(direction + 1) % 4,
		direction,
		(direction + 3) % 4,
		(direction + 2) % 4,
	}
	for _, wanted := range priorities {
		for _, candidate := range candidates {
			if !used[candidate] && segmentDirection(segments[candidate]) == wanted {
				return candidate
			}
		}
	}
	return -1
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

func largestLoop(loops [][]gridPoint) []gridPoint {
	var largest []gridPoint
	var largestArea int64
	for _, loop := range loops {
		area := gridPolygonAreaTwice(loop)
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

func scaleAndSimplify(loop []gridPoint, cfg Config) [][2]int {
	points := make([][2]int, 0, len(loop))
	for _, point := range loop {
		points = append(points, [2]int{
			int(math.Round(float64(point.x) * float64(cfg.Width) / float64(gridW))),
			int(math.Round(float64(point.y) * float64(cfg.Height) / float64(gridH))),
		})
	}
	return simplifyPolygon(points)
}

func simplifyPolygon(points [][2]int) [][2]int {
	points = removeConsecutiveDuplicates(points)
	for len(points) >= 3 {
		simplified := make([][2]int, 0, len(points))
		removed := false
		for i, point := range points {
			previous := points[(i+len(points)-1)%len(points)]
			next := points[(i+1)%len(points)]
			if collinear(previous, point, next) {
				removed = true
				continue
			}
			simplified = append(simplified, point)
		}
		points = simplified
		if !removed {
			break
		}
	}
	if polygonAreaTwice(points) < 0 {
		for left, right := 0, len(points)-1; left < right; left, right = left+1, right-1 {
			points[left], points[right] = points[right], points[left]
		}
	}
	return points
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

func collinear(first, middle, last [2]int) bool {
	return int64(middle[0]-first[0])*int64(last[1]-middle[1]) ==
		int64(middle[1]-first[1])*int64(last[0]-middle[0])
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

// extractAdjacency reports sorted site pairs that share at least three grid
// edges. The threshold removes point-only contacts at raster junctions.
func extractAdjacency(grid []int) [][2]int {
	if len(grid) != gridW*gridH {
		return nil
	}
	n := 0
	for _, owner := range grid {
		if owner+1 > n {
			n = owner + 1
		}
	}
	shared := make(map[[2]int]int)
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			owner := grid[y*gridW+x]
			if x+1 < gridW {
				addSharedEdge(shared, owner, grid[y*gridW+x+1])
			}
			if y+1 < gridH {
				addSharedEdge(shared, owner, grid[(y+1)*gridW+x])
			}
		}
	}

	edges := make([][2]int, 0)
	for first := 0; first < n; first++ {
		for second := first + 1; second < n; second++ {
			if shared[[2]int{first, second}] >= minSharedEdges {
				edges = append(edges, [2]int{first, second})
			}
		}
	}
	return edges
}

func addSharedEdge(shared map[[2]int]int, first, second int) {
	if first == second {
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
