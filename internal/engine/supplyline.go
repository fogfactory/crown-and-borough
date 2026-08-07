package engine

import (
	"errors"
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

var (
	ErrSupplyLineWinter           = errors.New("engine: supply line unavailable in winter")
	ErrSupplyLineUnknownTerritory = errors.New("engine: supply line unknown territory")
	ErrSupplyLineNoArmy           = errors.New("engine: supply line no army")
)

// SupplyLine describes the source and shortest route used to assign supply to
// one army during the next action-turn supply phase.
type SupplyLine struct {
	Territory    models.TerritoryID   `json:"territory"`
	ArmyOwner    models.PlayerID      `json:"armyOwner"`
	ArmySize     int                  `json:"armySize"`
	Rations      int                  `json:"rations"`
	Demand       int                  `json:"demand"`
	Source       *models.TerritoryID  `json:"source"`
	Distance     int                  `json:"distance"`
	Path         []models.TerritoryID `json:"path"`
	Reachable    []models.TerritoryID `json:"reachable"`
	SelfSupplied bool                 `json:"selfSupplied"`
}

// FindSupplyLine projects the supply assignment for an army without mutating
// the game. It intentionally uses the same network and source selection logic
// as resolveSupply.
func FindSupplyLine(game *models.GameState, balance assetgen.Balance, territoryID models.TerritoryID) (SupplyLine, error) {
	if game == nil {
		return SupplyLine{}, fmt.Errorf("engine: supply line: nil game state")
	}
	if game.Season == models.SeasonWinter {
		return SupplyLine{}, ErrSupplyLineWinter
	}
	if _, exists := game.TerritoryStates[territoryID]; !exists {
		return SupplyLine{}, fmt.Errorf("%w %q", ErrSupplyLineUnknownTerritory, territoryID)
	}
	if err := game.Validate(); err != nil {
		return SupplyLine{}, fmt.Errorf("engine: supply line: invalid game state: %w", err)
	}

	ctx := newResolutionContext(cloneGameState(game), balance)
	army := ctx.startArmyAt(territoryID)
	if army == nil {
		return SupplyLine{}, fmt.Errorf("%w at %q", ErrSupplyLineNoArmy, territoryID)
	}

	receivedRations := resolveRations(ctx)
	demand := armyCost(army.Size, balance.CostBase) - receivedRations[army.ID]
	line := SupplyLine{
		Territory: territoryID,
		ArmyOwner: army.OwnerID,
		ArmySize:  army.Size,
		Rations:   receivedRations[army.ID],
		Demand:    demand,
		Path:      []models.TerritoryID{},
		Reachable: []models.TerritoryID{},
	}
	if demand <= 0 {
		line.SelfSupplied = true
		return line, nil
	}

	sources := controlledSupplySources(ctx, army.OwnerID)
	source, distance := closestSupplySource(ctx, territoryID, sources)
	if source == nil {
		return line, nil
	}

	sourceID := source.territoryID
	line.Source = &sourceID
	line.Distance = distance
	line.Reachable = sortedSupplyTerritories(source.reachable)
	line.Path = supplyPath(ctx, source.reachable, sourceID, territoryID)
	return line, nil
}

func sortedSupplyTerritories(reachable map[models.TerritoryID]int) []models.TerritoryID {
	territories := make([]models.TerritoryID, 0, len(reachable))
	for territoryID := range reachable {
		territories = append(territories, territoryID)
	}
	sortTerritoryIDs(territories)
	return territories
}

func supplyPath(
	ctx *resolutionContext,
	reachable map[models.TerritoryID]int,
	sourceID models.TerritoryID,
	targetID models.TerritoryID,
) []models.TerritoryID {
	targetDistance, reachableTarget := reachable[targetID]
	if !reachableTarget {
		return nil
	}
	path := []models.TerritoryID{targetID}
	currentID := targetID
	for currentID != sourceID {
		currentDistance := targetDistance
		var previousID models.TerritoryID
		for _, neighborID := range ctx.sortedNeighbors(currentID) {
			if distance, ok := reachable[neighborID]; ok && distance == currentDistance-1 {
				previousID = neighborID
				break
			}
		}
		if previousID == "" {
			return nil
		}
		path = append(path, previousID)
		currentID = previousID
		targetDistance = currentDistance - 1
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}
