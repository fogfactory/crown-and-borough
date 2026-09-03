package engine

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type resolutionContext struct {
	state   *models.GameState
	balance assetgen.Balance

	armiesByID          map[models.ArmyID]*models.Army
	armyAtTerritory     map[models.TerritoryID]models.ArmyID
	chainsByID          map[models.ChainID]*models.Chain
	territoriesByID     map[models.TerritoryID]*models.Territory
	noblesByID          map[models.NobleID]*models.Noble
	noblesByCode        map[models.NobleCode]models.NobleID
	infrastructuresByID map[models.InfraID]*models.Infrastructure

	startArmiesByID      map[models.ArmyID]models.Army
	startArmyAtTerritory map[models.TerritoryID]models.ArmyID
	famished             map[models.ArmyID]bool

	records             map[models.ArmyID]*orderRecord
	attacks             map[models.ArmyID]*attackIntent
	joins               map[models.ArmyID]*joinIntent
	disperses           map[models.ArmyID]*disperseIntent
	disperseResults     map[models.ArmyID]*disperseResolution
	supports            map[models.ArmyID]*supportIntent
	joinResults         map[models.ArmyID]*joinResolution
	attackedTerritories map[models.TerritoryID]bool
	contest             contestState
	dislodged           map[models.ArmyID]*dislodgedArmy
	events              []Event
	deckIntents         []deckOrderIntent
}

func newResolutionContext(state *models.GameState, balance assetgen.Balance) *resolutionContext {
	ctx := &resolutionContext{
		state:                state,
		balance:              balance,
		startArmiesByID:      make(map[models.ArmyID]models.Army, len(state.Armies)),
		startArmyAtTerritory: make(map[models.TerritoryID]models.ArmyID, len(state.Armies)),
		famished:             make(map[models.ArmyID]bool),
		records:              make(map[models.ArmyID]*orderRecord),
		attacks:              make(map[models.ArmyID]*attackIntent),
		joins:                make(map[models.ArmyID]*joinIntent),
		disperses:            make(map[models.ArmyID]*disperseIntent),
		disperseResults:      make(map[models.ArmyID]*disperseResolution),
		supports:             make(map[models.ArmyID]*supportIntent),
		joinResults:          make(map[models.ArmyID]*joinResolution),
		attackedTerritories:  make(map[models.TerritoryID]bool),
		dislodged:            make(map[models.ArmyID]*dislodgedArmy),
	}
	for _, army := range state.Armies {
		copyArmy := army
		if army.ChainID != nil {
			chainID := *army.ChainID
			copyArmy.ChainID = &chainID
		}
		ctx.startArmiesByID[army.ID] = copyArmy
		ctx.startArmyAtTerritory[army.TerritoryID] = army.ID
	}
	ctx.rebuildIndexes()
	return ctx
}

func (ctx *resolutionContext) rebuildIndexes() {
	ctx.armiesByID = make(map[models.ArmyID]*models.Army, len(ctx.state.Armies))
	ctx.armyAtTerritory = make(map[models.TerritoryID]models.ArmyID, len(ctx.state.Armies))
	for i := range ctx.state.Armies {
		army := &ctx.state.Armies[i]
		ctx.armiesByID[army.ID] = army
		ctx.armyAtTerritory[army.TerritoryID] = army.ID
	}
	ctx.chainsByID = make(map[models.ChainID]*models.Chain, len(ctx.state.Chains))
	for i := range ctx.state.Chains {
		chain := &ctx.state.Chains[i]
		ctx.chainsByID[chain.ID] = chain
	}
	ctx.territoriesByID = make(map[models.TerritoryID]*models.Territory, len(ctx.state.Territories))
	for i := range ctx.state.Territories {
		territory := &ctx.state.Territories[i]
		ctx.territoriesByID[territory.ID] = territory
	}
	ctx.noblesByID = make(map[models.NobleID]*models.Noble, len(ctx.state.Nobles))
	ctx.noblesByCode = make(map[models.NobleCode]models.NobleID, len(ctx.state.Nobles))
	for i := range ctx.state.Nobles {
		noble := &ctx.state.Nobles[i]
		ctx.noblesByID[noble.ID] = noble
		ctx.noblesByCode[models.NobleCode(noble.Code)] = noble.ID
	}
	ctx.infrastructuresByID = make(map[models.InfraID]*models.Infrastructure, len(ctx.state.Infrastructures))
	for i := range ctx.state.Infrastructures {
		infrastructure := &ctx.state.Infrastructures[i]
		ctx.infrastructuresByID[infrastructure.ID] = infrastructure
	}
}

func (ctx *resolutionContext) startArmyAt(territoryID models.TerritoryID) *models.Army {
	armyID, exists := ctx.startArmyAtTerritory[territoryID]
	if !exists {
		return nil
	}
	army := ctx.startArmiesByID[armyID]
	return &army
}

func (ctx *resolutionContext) currentArmyAt(territoryID models.TerritoryID) *models.Army {
	armyID, exists := ctx.armyAtTerritory[territoryID]
	if !exists {
		return nil
	}
	return ctx.armiesByID[armyID]
}

func (ctx *resolutionContext) isAdjacent(sourceID, targetID models.TerritoryID) bool {
	territory := ctx.territoriesByID[sourceID]
	if territory == nil {
		return false
	}
	for _, adjacent := range territory.Adjacencies {
		if adjacent == targetID {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) sortedNeighbors(territoryID models.TerritoryID) []models.TerritoryID {
	territory := ctx.territoriesByID[territoryID]
	if territory == nil {
		return nil
	}
	neighbors := append([]models.TerritoryID(nil), territory.Adjacencies...)
	sortTerritoryIDs(neighbors)
	return neighbors
}

func (ctx *resolutionContext) hasInfrastructure(territoryID models.TerritoryID, kind models.InfraType) bool {
	state := ctx.state.TerritoryStates[territoryID]
	for _, infrastructureID := range state.Infrastructures {
		infrastructure := ctx.infrastructuresByID[infrastructureID]
		if infrastructure != nil && infrastructure.Type == kind {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) hasCastle(territoryID models.TerritoryID) bool {
	return ctx.hasInfrastructure(territoryID, models.InfraTypeCastle)
}

func (ctx *resolutionContext) rebuildOccupancy() error {
	for territoryID, state := range ctx.state.TerritoryStates {
		state.Army = nil
		ctx.state.TerritoryStates[territoryID] = state
	}
	sort.Slice(ctx.state.Armies, func(i, j int) bool {
		return lessArmyID(ctx.state.Armies[i].ID, ctx.state.Armies[j].ID)
	})
	seen := make(map[models.TerritoryID]bool, len(ctx.state.Armies))
	for _, army := range ctx.state.Armies {
		if seen[army.TerritoryID] {
			return fmt.Errorf("engine: multiple armies occupy territory %q", army.TerritoryID)
		}
		state, exists := ctx.state.TerritoryStates[army.TerritoryID]
		if !exists {
			return fmt.Errorf("engine: army %q occupies unknown territory %q", army.ID, army.TerritoryID)
		}
		armyID := army.ID
		state.Army = &armyID
		ctx.state.TerritoryStates[army.TerritoryID] = state
		seen[army.TerritoryID] = true
	}
	ctx.rebuildIndexes()
	return nil
}

func (ctx *resolutionContext) removeInfrastructure(infrastructureID models.InfraID) {
	ctx.removeInfrastructureWithStock(infrastructureID, false)
}

func (ctx *resolutionContext) removeInfrastructurePreservingStock(infrastructureID models.InfraID) {
	ctx.removeInfrastructureWithStock(infrastructureID, true)
}

func (ctx *resolutionContext) removeInfrastructureWithStock(infrastructureID models.InfraID, preserveStock bool) {
	infrastructure := ctx.infrastructuresByID[infrastructureID]
	if infrastructure == nil {
		return
	}
	if infrastructure.Type == models.InfraTypeCastle {
		for index := range ctx.state.Players {
			player := &ctx.state.Players[index]
			if player.CapitalCastleID != nil && *player.CapitalCastleID == infrastructureID {
				player.CapitalCastleID = nil
			}
		}
	}
	filtered := make([]models.Infrastructure, 0, len(ctx.state.Infrastructures)-1)
	for _, candidate := range ctx.state.Infrastructures {
		if candidate.ID != infrastructureID {
			filtered = append(filtered, candidate)
		}
	}
	ctx.state.Infrastructures = filtered
	state := ctx.state.TerritoryStates[infrastructure.TerritoryID]
	state.Infrastructures = removeInfraID(state.Infrastructures, infrastructureID)
	if !preserveStock && (infrastructure.Type == models.InfraTypeCastle || infrastructure.Type == models.InfraTypeVillage) {
		state.Resources = 0
	}
	ctx.state.TerritoryStates[infrastructure.TerritoryID] = state
	ctx.rebuildIndexes()
}

func removeInfraID(ids []models.InfraID, remove models.InfraID) []models.InfraID {
	filtered := make([]models.InfraID, 0, len(ids))
	for _, id := range ids {
		if id != remove {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (ctx *resolutionContext) allocateArmyID() models.ArmyID {
	id := models.ArmyID(fmt.Sprintf("A%d", ctx.state.NextArmyID))
	ctx.state.NextArmyID++
	return id
}

func (ctx *resolutionContext) noblesAt(territoryID models.TerritoryID) []models.NobleID {
	nobles := make([]models.NobleID, 0)
	for _, noble := range ctx.state.Nobles {
		if noble.LocationID == territoryID {
			nobles = append(nobles, noble.ID)
		}
	}
	sortNobleIDs(nobles)
	return nobles
}

func (ctx *resolutionContext) moveNobles(nobleIDs []models.NobleID, destinationID models.TerritoryID, armyID models.ArmyID) {
	for _, nobleID := range nobleIDs {
		noble := ctx.noblesByID[nobleID]
		if noble == nil || noble.LocationID == destinationID {
			continue
		}
		sourceID := noble.LocationID
		noble.LocationID = destinationID
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeNobleMovement,
			Phase:         4,
			ArmyID:        armyID,
			NobleID:       nobleID,
			SourceID:      sourceID,
			DestinationID: destinationID,
		})
	}
}

func sortArmyIDs(ids []models.ArmyID) {
	sort.Slice(ids, func(i, j int) bool { return lessArmyID(ids[i], ids[j]) })
}

func lessArmyID(left, right models.ArmyID) bool {
	leftSequence, leftNumeric := armySequence(left)
	rightSequence, rightNumeric := armySequence(right)
	if leftNumeric && rightNumeric {
		if leftSequence != rightSequence {
			return leftSequence < rightSequence
		}
		return left < right
	}
	if leftNumeric != rightNumeric {
		return leftNumeric
	}
	return left < right
}

func armySequence(id models.ArmyID) (int, bool) {
	value := string(id)
	if len(value) < 2 || value[0] != 'A' {
		return 0, false
	}
	sequence, err := strconv.Atoi(value[1:])
	if err != nil || sequence < 1 {
		return 0, false
	}
	return sequence, true
}

func sortTerritoryIDs(ids []models.TerritoryID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

func sortNobleIDs(ids []models.NobleID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}

func sortedArmyMap[V any](values map[models.ArmyID]V) []models.ArmyID {
	ids := make([]models.ArmyID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sortArmyIDs(ids)
	return ids
}

func sortedTerritoryMap[V any](values map[models.TerritoryID]V) []models.TerritoryID {
	ids := make([]models.TerritoryID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sortTerritoryIDs(ids)
	return ids
}
