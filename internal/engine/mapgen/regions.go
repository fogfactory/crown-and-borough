package mapgen

import (
	"fmt"
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func ValidateRegions(territories []Territory, regions []models.Region) error {
	if len(regions) == 0 {
		return nil
	}
	byID := make(map[models.TerritoryID]Territory, len(territories))
	for _, territory := range territories {
		byID[models.TerritoryID(territory.ID)] = territory
	}
	seen := make(map[models.TerritoryID]models.RegionID, len(territories))
	regionIDs := make(map[models.RegionID]bool, len(regions))
	for _, region := range regions {
		if region.ID == "" || region.Seed == "" {
			return fmt.Errorf("mapgen: region: id and seed are required")
		}
		if regionIDs[region.ID] {
			return fmt.Errorf("mapgen: region %q: duplicate id", region.ID)
		}
		regionIDs[region.ID] = true
		if len(region.Territories) == 0 || !slices.IsSorted(region.Territories) || !slices.Contains(region.Territories, region.Seed) {
			return fmt.Errorf("mapgen: region %q: invalid territory list", region.ID)
		}
		members := make(map[models.TerritoryID]bool, len(region.Territories))
		for _, territoryID := range region.Territories {
			_, exists := byID[territoryID]
			if !exists {
				return fmt.Errorf("mapgen: region %q: unknown territory %q", region.ID, territoryID)
			}
			if previous, exists := seen[territoryID]; exists {
				return fmt.Errorf("mapgen: territory %q: belongs to regions %q and %q", territoryID, previous, region.ID)
			}
			seen[territoryID] = region.ID
			members[territoryID] = true
		}
		visited := map[models.TerritoryID]bool{region.Seed: true}
		queue := []models.TerritoryID{region.Seed}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, adjacent := range byID[current].Adjacencies {
				adjacentID := models.TerritoryID(adjacent)
				if members[adjacentID] && !visited[adjacentID] {
					visited[adjacentID] = true
					queue = append(queue, adjacentID)
				}
			}
		}
		if len(visited) != len(members) {
			return fmt.Errorf("mapgen: region %q: territories are not connected", region.ID)
		}
	}
	if len(seen) != len(byID) {
		return fmt.Errorf("mapgen: regions cover %d of %d territories", len(seen), len(byID))
	}
	return nil
}
