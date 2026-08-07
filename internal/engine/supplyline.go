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
	ErrSupplyLineNoSource         = errors.New("engine: supply line no source")
)

type SupplyLineKind string

const (
	SupplyLineKindArmy   SupplyLineKind = "army"
	SupplyLineKindSource SupplyLineKind = "source"
)

// SupplyLine describes the source and shortest route used to assign supply to
// one army, or the reachable zone of a selected supply source.
type SupplyLine struct {
	Kind         SupplyLineKind       `json:"kind"`
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
// the game. Army assignment uses the same network and source selection logic as
// resolveSupply; a selected source also exposes its own reachable zone.
func FindSupplyLine(game *models.GameState, balance assetgen.Balance, territoryID models.TerritoryID) (SupplyLine, error) {
	ctx, err := supplyQueryContext(game, balance, territoryID)
	if err != nil {
		return SupplyLine{}, err
	}

	army := ctx.startArmyAt(territoryID)
	if army == nil {
		return SupplyLine{}, fmt.Errorf("%w at %q", ErrSupplyLineNoArmy, territoryID)
	}
	return projectSupplyLine(ctx, balance, territoryID, army), nil
}

// FindSupply resolves the display requested for a selected territory: an army
// line takes precedence, otherwise a controlled source exposes its reachable
// zone.
func FindSupply(game *models.GameState, balance assetgen.Balance, territoryID models.TerritoryID) (SupplyLine, error) {
	ctx, err := supplyQueryContext(game, balance, territoryID)
	if err != nil {
		return SupplyLine{}, err
	}
	if army := ctx.startArmyAt(territoryID); army != nil {
		return projectSupplyLine(ctx, balance, territoryID, army), nil
	}
	return projectSupplyZone(ctx, territoryID)
}

func projectSupplyLine(
	ctx *resolutionContext,
	balance assetgen.Balance,
	territoryID models.TerritoryID,
	army *models.Army,
) SupplyLine {

	receivedRations := resolveRations(ctx)
	demand := armyCost(army.Size, balance.CostBase) - receivedRations[army.ID]
	line := SupplyLine{
		Kind:      SupplyLineKindArmy,
		Territory: territoryID,
		ArmyOwner: army.OwnerID,
		ArmySize:  army.Size,
		Rations:   receivedRations[army.ID],
		Demand:    demand,
		Path:      []models.TerritoryID{},
		Reachable: []models.TerritoryID{},
	}
	if ownerID, isSource := controlledSupplyOwner(ctx, territoryID); isSource &&
		ownerID == army.OwnerID {
		line.Reachable = sortedSupplyTerritories(supplyNetwork(ctx, territoryID, ownerID))
	}
	if demand <= 0 {
		line.SelfSupplied = true
		return line
	}

	sources := controlledSupplySources(ctx, army.OwnerID)
	source, distance := closestSupplySource(ctx, territoryID, sources)
	if source == nil {
		return line
	}

	sourceID := source.territoryID
	line.Source = &sourceID
	line.Distance = distance
	line.Reachable = sortedSupplyTerritories(source.reachable)
	line.Path = supplyPath(ctx, source.reachable, sourceID, territoryID)
	return line
}

// FindSupplyZone projects the network reachable from a controlled castle or
// village selected without an army on its territory.
func FindSupplyZone(game *models.GameState, balance assetgen.Balance, territoryID models.TerritoryID) (SupplyLine, error) {
	ctx, err := supplyQueryContext(game, balance, territoryID)
	if err != nil {
		return SupplyLine{}, err
	}

	return projectSupplyZone(ctx, territoryID)
}

func projectSupplyZone(ctx *resolutionContext, territoryID models.TerritoryID) (SupplyLine, error) {
	ownerID, isSource := controlledSupplyOwner(ctx, territoryID)
	if !isSource {
		return SupplyLine{}, fmt.Errorf("%w at %q", ErrSupplyLineNoSource, territoryID)
	}
	sourceID := territoryID
	return SupplyLine{
		Kind:         SupplyLineKindSource,
		Territory:    territoryID,
		ArmyOwner:    ownerID,
		Source:       &sourceID,
		Path:         []models.TerritoryID{},
		Reachable:    sortedSupplyTerritories(supplyNetwork(ctx, territoryID, ownerID)),
		SelfSupplied: false,
	}, nil
}

func supplyQueryContext(game *models.GameState, balance assetgen.Balance, territoryID models.TerritoryID) (*resolutionContext, error) {
	if game == nil {
		return nil, fmt.Errorf("engine: supply line: nil game state")
	}
	if game.Season == models.SeasonWinter {
		return nil, ErrSupplyLineWinter
	}
	if _, exists := game.TerritoryStates[territoryID]; !exists {
		return nil, fmt.Errorf("%w %q", ErrSupplyLineUnknownTerritory, territoryID)
	}
	if err := game.Validate(); err != nil {
		return nil, fmt.Errorf("engine: supply line: invalid game state: %w", err)
	}
	return newResolutionContext(cloneGameState(game), balance), nil
}

func controlledSupplyOwner(ctx *resolutionContext, territoryID models.TerritoryID) (models.PlayerID, bool) {
	state := ctx.state.TerritoryStates[territoryID]
	if state.OwnerID == nil {
		return "", false
	}
	infrastructure := ctx.infrastructureAt(territoryID)
	if infrastructure == nil {
		return "", false
	}
	switch infrastructure.Type {
	case models.InfraTypeCastle, models.InfraTypeVillage:
		return *state.OwnerID, true
	default:
		return "", false
	}
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
