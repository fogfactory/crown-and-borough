package mapgen

import (
	"math"
	"math/rand/v2"
)

type point struct {
	x float64
	y float64
}

// generateSites uses a jittered grid so every viewport area is represented
// while territory shapes and sizes remain irregular.
func generateSites(rng *rand.Rand, cfg Config) []point {
	cols := int(math.Ceil(math.Sqrt(float64(cfg.SiteCount) * float64(cfg.Width) / float64(cfg.Height))))
	if cols < 1 {
		cols = 1
	}
	rows := int(math.Ceil(float64(cfg.SiteCount) / float64(cols)))

	cells := make([]int, cols*rows)
	for i := range cells {
		cells[i] = i
	}
	shuffle(rng, cells)

	cellWidth := float64(cfg.Width) / float64(cols)
	cellHeight := float64(cfg.Height) / float64(rows)
	sites := make([]point, cfg.SiteCount)
	for i, cell := range cells[:cfg.SiteCount] {
		col := cell % cols
		row := cell / cols
		x := (float64(col)+0.5)*cellWidth + (rng.Float64()*0.7-0.35)*cellWidth
		y := (float64(row)+0.5)*cellHeight + (rng.Float64()*0.7-0.35)*cellHeight
		sites[i] = point{
			x: clampFloat(x, 0, float64(cfg.Width)),
			y: clampFloat(y, 0, float64(cfg.Height)),
		}
	}
	return sites
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
