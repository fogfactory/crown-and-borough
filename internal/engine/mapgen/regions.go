package mapgen

import (
	"container/heap"
	"fmt"
	"slices"
	"sort"

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

type regionQueueItem struct {
	territory models.TerritoryID
	seed      models.TerritoryID
	distance  int
}

type regionQueue []regionQueueItem

func (q regionQueue) Len() int { return len(q) }
func (q regionQueue) Less(i, j int) bool {
	if q[i].distance != q[j].distance {
		return q[i].distance < q[j].distance
	}
	if q[i].seed != q[j].seed {
		return q[i].seed < q[j].seed
	}
	return q[i].territory < q[j].territory
}
func (q regionQueue) Swap(i, j int)   { q[i], q[j] = q[j], q[i] }
func (q *regionQueue) Push(value any) { *q = append(*q, value.(regionQueueItem)) }
func (q *regionQueue) Pop() any {
	old := *q
	last := len(old) - 1
	value := old[last]
	*q = old[:last]
	return value
}

func generateRegions(territories []Territory) ([]models.Region, error) {
	if len(territories) == 0 {
		return []models.Region{}, nil
	}
	byID := make(map[models.TerritoryID]Territory, len(territories))
	seeds := make([]models.TerritoryID, 0)
	for _, territory := range territories {
		territoryID := models.TerritoryID(territory.ID)
		byID[territoryID] = territory
		if territory.Village {
			seeds = append(seeds, territoryID)
		}
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("mapgen: cannot generate regions without village seeds")
	}
	sort.Slice(seeds, func(i, j int) bool { return seeds[i] < seeds[j] })
	best := make(map[models.TerritoryID]regionQueueItem, len(territories))
	queue := &regionQueue{}
	heap.Init(queue)
	for _, seed := range seeds {
		item := regionQueueItem{territory: seed, seed: seed, distance: 0}
		best[seed] = item
		heap.Push(queue, item)
	}
	for queue.Len() > 0 {
		item := heap.Pop(queue).(regionQueueItem)
		current, exists := best[item.territory]
		if !exists || current.distance != item.distance || current.seed != item.seed {
			continue
		}
		for _, adjacent := range byID[item.territory].Adjacencies {
			neighbor := models.TerritoryID(adjacent)
			candidate := regionQueueItem{territory: neighbor, seed: item.seed, distance: item.distance + 1}
			previous, assigned := best[neighbor]
			if assigned && (previous.distance < candidate.distance || (previous.distance == candidate.distance && previous.seed <= candidate.seed)) {
				continue
			}
			best[neighbor] = candidate
			heap.Push(queue, candidate)
		}
	}
	if len(best) != len(byID) {
		return nil, fmt.Errorf("mapgen: regions do not cover all territories")
	}
	members := make(map[models.TerritoryID][]models.TerritoryID, len(seeds))
	for territoryID, item := range best {
		members[item.seed] = append(members[item.seed], territoryID)
	}
	regions := make([]models.Region, 0, len(seeds))
	for _, seed := range seeds {
		territoriesInRegion := members[seed]
		sort.Slice(territoriesInRegion, func(i, j int) bool { return territoriesInRegion[i] < territoriesInRegion[j] })
		regions = append(regions, models.Region{ID: models.RegionID(seed), Seed: seed, Territories: territoriesInRegion})
	}
	if err := rebalanceRegions(territories, regions); err != nil {
		return nil, err
	}
	if err := ValidateRegions(territories, regions); err != nil {
		return nil, err
	}
	return regions, nil
}

type regionMove struct {
	donor       int
	target      int
	territories []models.TerritoryID
	rangeAfter  int
	diffAfter   int
}

func rebalanceRegions(territories []Territory, regions []models.Region) error {
	if len(regions) <= 1 {
		return nil
	}
	limit := 2 * (len(regions) - 1)
	byID := make(map[models.TerritoryID]Territory, len(territories))
	for _, territory := range territories {
		byID[models.TerritoryID(territory.ID)] = territory
	}
	members := make([]map[models.TerritoryID]bool, len(regions))
	for index, region := range regions {
		members[index] = make(map[models.TerritoryID]bool, len(region.Territories))
		for _, territoryID := range region.Territories {
			members[index][territoryID] = true
		}
	}

	for {
		minimum, maximum := regionSizeRange(regions)
		if maximum-minimum <= limit {
			return nil
		}
		move, found := bestRegionMove(byID, regions, members)
		if !found {
			return fmt.Errorf("mapgen: cannot balance regions within %d territories", limit)
		}
		for _, territoryID := range move.territories {
			delete(members[move.donor], territoryID)
			members[move.target][territoryID] = true
		}
		regions[move.donor].Territories = sortedRegionMembers(members[move.donor])
		regions[move.target].Territories = sortedRegionMembers(members[move.target])
	}
}

func regionSizeRange(regions []models.Region) (int, int) {
	minimum := len(regions[0].Territories)
	maximum := minimum
	for _, region := range regions[1:] {
		size := len(region.Territories)
		minimum = min(minimum, size)
		maximum = max(maximum, size)
	}
	return minimum, maximum
}

func sortedRegionMembers(members map[models.TerritoryID]bool) []models.TerritoryID {
	result := make([]models.TerritoryID, 0, len(members))
	for territoryID := range members {
		result = append(result, territoryID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func bestRegionMove(byID map[models.TerritoryID]Territory, regions []models.Region, members []map[models.TerritoryID]bool) (regionMove, bool) {
	var best regionMove
	found := false
	for donorIndex, donor := range regions {
		for targetIndex, target := range regions {
			if donorIndex == targetIndex || len(donor.Territories)-len(target.Territories) <= 1 {
				continue
			}
			for _, territoryID := range donor.Territories {
				if territoryID == donor.Seed || !hasAdjacentMember(byID[territoryID], members[targetIndex]) {
					continue
				}
				movedTerritories, removable := removableRegionPart(byID, members[donorIndex], donor.Seed, territoryID)
				if !removable {
					continue
				}
				moveSize := len(movedTerritories)
				if len(donor.Territories)-len(target.Territories) < moveSize {
					continue
				}
				sizes := make([]int, len(regions))
				for index, region := range regions {
					sizes[index] = len(region.Territories)
				}
				sizes[donorIndex] -= moveSize
				sizes[targetIndex] += moveSize
				afterMin, afterMax := sizes[0], sizes[0]
				for _, size := range sizes[1:] {
					afterMin = min(afterMin, size)
					afterMax = max(afterMax, size)
				}
				candidate := regionMove{
					donor: donorIndex, target: targetIndex, territories: movedTerritories,
					rangeAfter: afterMax - afterMin,
					diffAfter:  abs(sizes[donorIndex] - sizes[targetIndex]),
				}
				if !found || betterRegionMove(candidate, best, regions) {
					best = candidate
					found = true
				}
			}
		}
	}
	return best, found
}

func betterRegionMove(candidate, current regionMove, regions []models.Region) bool {
	if candidate.rangeAfter != current.rangeAfter {
		return candidate.rangeAfter < current.rangeAfter
	}
	if candidate.diffAfter != current.diffAfter {
		return candidate.diffAfter < current.diffAfter
	}
	if regions[candidate.donor].Seed != regions[current.donor].Seed {
		return regions[candidate.donor].Seed < regions[current.donor].Seed
	}
	if regions[candidate.target].Seed != regions[current.target].Seed {
		return regions[candidate.target].Seed < regions[current.target].Seed
	}
	if candidate.territories[0] != current.territories[0] {
		return candidate.territories[0] < current.territories[0]
	}
	return len(candidate.territories) < len(current.territories)
}

func hasAdjacentMember(territory Territory, target map[models.TerritoryID]bool) bool {
	for _, adjacent := range territory.Adjacencies {
		if target[models.TerritoryID(adjacent)] {
			return true
		}
	}
	return false
}

func removableRegionPart(byID map[models.TerritoryID]Territory, members map[models.TerritoryID]bool, seed, removed models.TerritoryID) ([]models.TerritoryID, bool) {
	if removed == seed || len(members) <= 1 {
		return nil, false
	}
	visited := map[models.TerritoryID]bool{seed: true}
	queue := []models.TerritoryID{seed}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, adjacent := range byID[current].Adjacencies {
			adjacentID := models.TerritoryID(adjacent)
			if adjacentID == removed || !members[adjacentID] || visited[adjacentID] {
				continue
			}
			visited[adjacentID] = true
			queue = append(queue, adjacentID)
		}
	}
	if len(visited) == 0 {
		return nil, false
	}
	moved := make([]models.TerritoryID, 0, len(members)-len(visited))
	for territoryID := range members {
		if !visited[territoryID] {
			moved = append(moved, territoryID)
		}
	}
	sort.Slice(moved, func(i, j int) bool { return moved[i] < moved[j] })
	return moved, true
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
