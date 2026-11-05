package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// MinimumStartingDistance is the minimum graph distance between two starting
// territories. Adjacent territories have distance 1. Starting-position
// selection itself lives in mapgen, alongside the rest of map generation;
// this re-export lets engine code and tests reason about the constraint
// without importing mapgen directly.
const MinimumStartingDistance = mapgen.MinimumStartingDistance

// startingOutpostTerritories selects count passable neighbours of startID to
// host a starting outpost army (#203): eligible neighbours are ones whose
// terrain ration feeds their own 1-troop garrison (armyCost(1, ...)) on its
// own, so a fresh outpost never starves before its owner can reinforce it.
// Mountain neighbours never qualify, since their ration is 0. Candidates are
// ranked by ration descending, then by territory ID for a deterministic
// tie-break, mirroring mapgen's own deterministic ordering conventions.
func startingOutpostTerritories(state *models.GameState, startID models.TerritoryID, balance assetgen.Balance, count int) ([]models.TerritoryID, error) {
	start := territoryByID(state.Territories, startID)
	cost := armyCost(1, balance.CostBase)

	type candidate struct {
		id     models.TerritoryID
		ration int
	}
	candidates := make([]candidate, 0, len(start.Adjacencies))
	for _, neighborID := range start.Adjacencies {
		neighbor := territoryByID(state.Territories, neighborID)
		ration := balance.RationTerrain[neighbor.Terrain]
		if ration < cost {
			continue
		}
		candidates = append(candidates, candidate{id: neighborID, ration: ration})
	}
	if len(candidates) < count {
		return nil, fmt.Errorf("engine: start %s has %d eligible outpost neighbours, need %d", startID, len(candidates), count)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ration != candidates[j].ration {
			return candidates[i].ration > candidates[j].ration
		}
		return candidates[i].id < candidates[j].id
	})

	territories := make([]models.TerritoryID, count)
	for index := 0; index < count; index++ {
		territories[index] = candidates[index].id
	}
	return territories, nil
}
