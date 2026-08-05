package mapgen

import (
	"math"
	"math/rand/v2"
	"sort"
)

const (
	// interiorMarginRatio keeps delivered territories away from the raster
	// viewport. The sacrificial frame ring absorbs every viewport border cell.
	interiorMarginRatio = 0.08
	frameRing           = 0.5
)

type point struct {
	x float64
	y float64
}

// generateSites returns interior sites first, followed by the frameRing. The
// ring is inset by half the interior margin: even at the midpoint between two
// frame sites its distance to the viewport border is below the distance to an
// interior site. Its effective gap is at most 1.5 times the margin, which is
// below the sqrt(3) bound needed for that guarantee.
func generateSites(rng *rand.Rand, cfg Config) []point {
	margin := interiorMargin(cfg)
	// Keeping the jittered grid one additional margin inside its documented
	// bounding box ensures peripheral territories retain interior neighbours
	// instead of becoming frame-adjacent slivers.
	siteInset := 2 * margin
	innerWidth := float64(cfg.Width) - 2*siteInset
	innerHeight := float64(cfg.Height) - 2*siteInset
	cols := int(math.Ceil(math.Sqrt(float64(cfg.SiteCount) * innerWidth / innerHeight)))
	if cols < 1 {
		cols = 1
	}
	rows := int(math.Ceil(float64(cfg.SiteCount) / float64(cols)))

	cells := interiorGridCells(rng, cols, rows, cfg.SiteCount)

	cellWidth := innerWidth / float64(cols)
	cellHeight := innerHeight / float64(rows)
	sites := make([]point, cfg.SiteCount, cfg.SiteCount+frameSiteCount(cfg, margin))
	for i, cell := range cells[:cfg.SiteCount] {
		col := cell % cols
		row := cell / cols
		x := siteInset + (float64(col)+0.5)*cellWidth + (rng.Float64()*0.7-0.35)*cellWidth
		y := siteInset + (float64(row)+0.5)*cellHeight + (rng.Float64()*0.7-0.35)*cellHeight
		sites[i] = point{
			x: clampFloat(x, siteInset, float64(cfg.Width)-siteInset),
			y: clampFloat(y, siteInset, float64(cfg.Height)-siteInset),
		}
	}
	return append(sites, generateFrameSites(rng, cfg, margin)...)
}

func interiorGridCells(rng *rand.Rand, cols, rows, count int) []int {
	boundary := make([]int, 0, cols*2+max(rows-2, 0)*2)
	interior := make([]int, 0)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			cell := row*cols + col
			if row == 0 || row == rows-1 || col == 0 || col == cols-1 {
				boundary = append(boundary, cell)
				continue
			}
			interior = append(interior, cell)
		}
	}

	cells := make([]int, 0, count)
	if len(boundary) >= count {
		// Required configurations always fit the full boundary. This fallback
		// still distributes a hypothetical smaller population around it.
		for i := 0; i < count; i++ {
			index := i * len(boundary) / count
			cells = append(cells, boundary[index])
		}
	} else {
		cells = append(cells, boundary...)
		shuffle(rng, interior)
		cells = append(cells, interior[:count-len(cells)]...)
	}
	shuffle(rng, cells)
	return cells
}

func interiorMargin(cfg Config) float64 {
	margin := math.Round(interiorMarginRatio * math.Min(float64(cfg.Width), float64(cfg.Height)))
	if margin < 1 {
		return 1
	}
	return margin
}

func frameSiteCount(cfg Config, margin float64) int {
	inset := frameInset(margin)
	perimeter := 2 * (float64(cfg.Width) - 2*inset + float64(cfg.Height) - 2*inset)
	count := int(math.Ceil(perimeter / margin))
	if count < 4 {
		return 4
	}
	return count
}

func generateFrameSites(rng *rand.Rand, cfg Config, margin float64) []point {
	inset := frameInset(margin)
	width := float64(cfg.Width) - 2*inset
	height := float64(cfg.Height) - 2*inset
	perimeter := 2 * (width + height)
	count := frameSiteCount(cfg, margin)
	spacing := perimeter / float64(count)
	positions := make([]float64, count)
	for i := range positions {
		// Tangential jitter remains within +/- one quarter of the nominal
		// spacing. Sorting restores the perimeter order without changing gaps.
		position := float64(i)*spacing + (rng.Float64()*0.5-0.25)*spacing
		position = math.Mod(position, perimeter)
		if position < 0 {
			position += perimeter
		}
		positions[i] = position
	}
	sort.Float64s(positions)

	sites := make([]point, 0, count)
	for _, position := range positions {
		sites = append(sites, framePoint(position, width, height, inset))
	}
	return sites
}

func frameInset(margin float64) float64 {
	// The radial coordinate stays in the documented [margin/4, 3*margin/4]
	// edge band while frame jitter is tangential only.
	return clampFloat(margin*frameRing, margin/4, 3*margin/4)
}

func framePoint(position, width, height, inset float64) point {
	switch {
	case position < width:
		return point{x: inset + position, y: inset}
	case position < width+height:
		return point{x: inset + width, y: inset + position - width}
	case position < 2*width+height:
		return point{x: inset + 2*width + height - position, y: inset + height}
	default:
		return point{x: inset, y: inset + 2*(width+height) - position}
	}
}

func clampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
