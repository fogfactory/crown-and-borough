package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// These temporary P1.5 balance values remain compiled into the resolver until
// P1.6 loads them from the balance asset.
const (
	BaseProduction    = 1
	SupplyRange       = 3
	DepotRangeBonus   = 2
	InfraRationsBonus = 2
	CostBase          = 2
	PillageBonus      = 2
)

// RationTerrain is the instant, non-stockable ration production of each
// terrain type at the start of an action turn.
var RationTerrain = map[models.Terrain]int{
	models.TerrainPlain:    1,
	models.TerrainForest:   1,
	models.TerrainHill:     1,
	models.TerrainMountain: 0,
	models.TerrainSwamp:    0,
}
