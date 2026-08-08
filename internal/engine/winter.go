package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const winterPhase = 6

// ResolveWinter applies direct winter management orders without resolving
// chains, movement, combat, supply, or the calendar. It clones the input so
// callers can safely retain the state they submitted.
func ResolveWinter(
	game *models.GameState,
	balance assetgen.Balance,
	orders map[models.PlayerID][]models.WinterOrder,
) (Resolution, error) {
	if game == nil {
		return Resolution{}, fmt.Errorf("engine: resolve winter: nil game state")
	}
	if err := game.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve winter: invalid game state: %w", err)
	}
	if game.Season != models.SeasonWinter {
		return Resolution{}, fmt.Errorf("engine: resolve winter: %s state must use Resolve", game.Season)
	}
	if balance.WinterStockDivisor < 1 {
		return Resolution{}, fmt.Errorf("engine: resolve winter: winter stock divisor must be > 0")
	}
	if err := validateWinterPlayers(game, orders); err != nil {
		return Resolution{}, err
	}

	state := cloneGameState(game)
	ctx := newResolutionContext(state, balance)
	stockBefore := winterStocks(ctx)
	firstNameRNG := newWinterRNG(state.Seed, state.Turn)
	for _, playerID := range sortedPlayerIDs(state.Players) {
		for _, order := range orders[playerID] {
			ctx.resolveWinterOrder(playerID, order, firstNameRNG)
		}
	}
	ctx.conserveWinterStocks()
	ctx.repatriateWinterStocks()
	ctx.emitWinterStockEvents(stockBefore)

	if err := state.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("engine: resolve winter: invalid result: %w", err)
	}
	return Resolution{
		State:  state,
		Events: append([]Event(nil), ctx.events...),
	}, nil
}

func validateWinterPlayers(game *models.GameState, orders map[models.PlayerID][]models.WinterOrder) error {
	players := make(map[models.PlayerID]bool, len(game.Players))
	for _, player := range game.Players {
		players[player.ID] = true
	}
	playerIDs := make([]models.PlayerID, 0, len(orders))
	for playerID := range orders {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Slice(playerIDs, func(i, j int) bool { return playerIDs[i] < playerIDs[j] })
	for _, playerID := range playerIDs {
		if !players[playerID] {
			return fmt.Errorf("engine: resolve winter: unknown player %q", playerID)
		}
	}
	return nil
}

func (ctx *resolutionContext) resolveWinterOrder(playerID models.PlayerID, order models.WinterOrder, firstNameRNG *rand.Rand) {
	switch order.Type {
	case models.WinterOrderTypeRecruitNoble:
		ctx.resolveRecruitNoble(playerID, order, firstNameRNG)
	case models.WinterOrderTypeRecruitTroop:
		ctx.resolveRecruitTroop(playerID, order)
	case models.WinterOrderTypeBuild:
		ctx.resolveBuild(playerID, order)
	case models.WinterOrderTypeElectCapital:
		ctx.resolveElectCapital(playerID, order)
	case models.WinterOrderTypeLiberateNoble:
		ctx.resolveLiberateNoble(playerID, order)
	default:
		ctx.rejectWinterOrder(playerID, order, "invalid_winter_order")
	}
}

func (ctx *resolutionContext) resolveRecruitNoble(playerID models.PlayerID, order models.WinterOrder, firstNameRNG *rand.Rand) {
	if !ctx.territoryExists(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "unknown_territory")
		return
	}
	if !ctx.controlsTerritory(playerID, order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "territory_not_controlled")
		return
	}
	if !ctx.hasSettlement(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "noble_requires_settlement")
		return
	}
	army := ctx.currentArmyAt(order.TerritoryID)
	if army == nil || army.OwnerID != playerID {
		ctx.rejectWinterOrder(playerID, order, "noble_requires_owned_army")
		return
	}
	if !ctx.hasAvailableFirstName(ctx.balance.FirstNames) {
		ctx.rejectWinterOrder(playerID, order, "no_available_first_name")
		return
	}
	spent, paid := ctx.payWinterCost(playerID, order.TerritoryID, ctx.balance.Costs.Noble)
	if !paid {
		ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
		return
	}
	firstName := ctx.drawFirstName(firstNameRNG, ctx.balance.FirstNames)
	territory := ctx.territoriesByID[order.TerritoryID]
	noble := models.Noble{
		ID:               nextNobleID(ctx.state.Nobles),
		Code:             firstName.Code,
		Name:             fmt.Sprintf("%s de %s", firstName.Name, territory.Name),
		OwnerID:          playerID,
		LocationID:       order.TerritoryID,
		Status:           models.NobleStatusFree,
		LastEmissionTurn: 0,
	}
	ctx.state.Nobles = append(ctx.state.Nobles, noble)
	ctx.rebuildIndexes()
	ctx.events = append(ctx.events, Event{
		Type:          EventTypeRecruit,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       order.ID,
		TerritoryID:   order.TerritoryID,
		NobleID:       noble.ID,
		NobleCode:     models.NobleCode(noble.Code),
		NobleName:     noble.Name,
		ResourceSpent: spent,
	})
}

func (ctx *resolutionContext) resolveRecruitTroop(playerID models.PlayerID, order models.WinterOrder) {
	if !ctx.territoryExists(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "unknown_territory")
		return
	}
	if !ctx.controlsTerritory(playerID, order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "territory_not_controlled")
		return
	}
	army := ctx.currentArmyAt(order.TerritoryID)
	if army != nil && army.OwnerID != playerID {
		ctx.rejectWinterOrder(playerID, order, "territory_occupied_by_other_player")
		return
	}
	spent, paid := ctx.payWinterCost(playerID, order.TerritoryID, ctx.balance.Costs.Troop)
	if !paid {
		ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
		return
	}
	if army != nil {
		army.Size++
		ctx.events = append(ctx.events, Event{
			Type:          EventTypeRecruit,
			Phase:         winterPhase,
			OwnerID:       playerID,
			OrderID:       order.ID,
			TerritoryID:   order.TerritoryID,
			ArmyID:        army.ID,
			Troops:        1,
			ResourceSpent: spent,
		})
		return
	}
	newArmy := models.Army{
		ID:          ctx.allocateArmyID(),
		OwnerID:     playerID,
		TerritoryID: order.TerritoryID,
		Size:        1,
	}
	ctx.state.Armies = append(ctx.state.Armies, newArmy)
	armyID := newArmy.ID
	state := ctx.state.TerritoryStates[order.TerritoryID]
	state.Army = &armyID
	ctx.state.TerritoryStates[order.TerritoryID] = state
	ctx.rebuildIndexes()
	ctx.events = append(ctx.events, Event{
		Type:          EventTypeRecruit,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       order.ID,
		TerritoryID:   order.TerritoryID,
		ArmyID:        newArmy.ID,
		Troops:        1,
		ResourceSpent: spent,
	})
}

func (ctx *resolutionContext) resolveBuild(playerID models.PlayerID, order models.WinterOrder) {
	if !ctx.territoryExists(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "unknown_territory")
		return
	}
	if !ctx.controlsTerritory(playerID, order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "territory_not_controlled")
		return
	}
	if !isBuildableInfrastructure(order.InfraType) {
		ctx.rejectWinterOrder(playerID, order, "invalid_infrastructure")
		return
	}
	if order.InfraType == models.InfraTypeMill && !ctx.millCanBeBuiltAt(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "mill_requires_productive_neighbor")
		return
	}
	existing := ctx.infrastructureAt(order.TerritoryID)
	if existing != nil {
		if existing.Type == models.InfraTypeMill && order.InfraType == models.InfraTypeMill {
			spent, paid := ctx.payWinterCost(playerID, order.TerritoryID, ctx.balance.Costs.Mill)
			if !paid {
				ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
				return
			}
			existing.Level++
			ctx.events = append(ctx.events, Event{
				Type:               EventTypeUpgrade,
				Phase:              winterPhase,
				OwnerID:            playerID,
				OrderID:            order.ID,
				TerritoryID:        order.TerritoryID,
				InfrastructureID:   existing.ID,
				InfrastructureType: existing.Type,
				Level:              existing.Level,
				ResourceSpent:      spent,
			})
			return
		}
		if existing.Type != models.InfraTypeVillage || order.InfraType != models.InfraTypeCastle {
			ctx.rejectWinterOrder(playerID, order, "structure_present")
			return
		}
	}
	cost, exists := infrastructureCost(ctx.balance.Costs, order.InfraType)
	if !exists {
		ctx.rejectWinterOrder(playerID, order, "invalid_infrastructure")
		return
	}
	spent, paid := ctx.payWinterCost(playerID, order.TerritoryID, cost)
	if !paid {
		ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
		return
	}
	if existing != nil {
		ctx.removeInfrastructurePreservingStock(existing.ID)
	}
	infrastructure := ctx.addWinterInfrastructure(order.InfraType, order.TerritoryID)
	capitalAssigned := false
	if order.InfraType == models.InfraTypeCastle {
		if _, _, hasCapital := ctx.capitalTerritory(playerID); !hasCapital {
			ctx.setCapital(playerID, infrastructure.ID)
			capitalAssigned = true
		}
	}
	ctx.events = append(ctx.events, Event{
		Type:               EventTypeBuild,
		Phase:              winterPhase,
		OwnerID:            playerID,
		OrderID:            order.ID,
		TerritoryID:        order.TerritoryID,
		InfrastructureID:   infrastructure.ID,
		InfrastructureType: infrastructure.Type,
		Level:              infrastructure.Level,
		ResourceSpent:      spent,
	})
	if capitalAssigned {
		ctx.events = append(ctx.events, Event{
			Type:               EventTypeCapitalElected,
			Phase:              winterPhase,
			OwnerID:            playerID,
			OrderID:            order.ID,
			TerritoryID:        order.TerritoryID,
			InfrastructureID:   infrastructure.ID,
			InfrastructureType: infrastructure.Type,
			ResourceSpent:      0,
			Automatic:          true,
		})
	}
}

func (ctx *resolutionContext) resolveElectCapital(playerID models.PlayerID, order models.WinterOrder) {
	if !ctx.territoryExists(order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "unknown_territory")
		return
	}
	if !ctx.controlsTerritory(playerID, order.TerritoryID) {
		ctx.rejectWinterOrder(playerID, order, "territory_not_controlled")
		return
	}
	infrastructure := ctx.infrastructureAt(order.TerritoryID)
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle {
		ctx.rejectWinterOrder(playerID, order, "capital_requires_controlled_castle")
		return
	}
	ctx.setCapital(playerID, infrastructure.ID)
	ctx.events = append(ctx.events, Event{
		Type:               EventTypeCapitalElected,
		Phase:              winterPhase,
		OwnerID:            playerID,
		OrderID:            order.ID,
		TerritoryID:        order.TerritoryID,
		InfrastructureID:   infrastructure.ID,
		InfrastructureType: infrastructure.Type,
		ResourceSpent:      0,
	})
}

func (ctx *resolutionContext) resolveLiberateNoble(playerID models.PlayerID, order models.WinterOrder) {
	nobleID, exists := ctx.noblesByCode[order.NobleCode]
	if !exists {
		ctx.rejectWinterOrder(playerID, order, "unknown_noble")
		return
	}
	noble := ctx.noblesByID[nobleID]
	if noble == nil || noble.OwnerID != playerID {
		ctx.rejectWinterOrder(playerID, order, "noble_not_owned")
		return
	}
	if noble.Status == models.NobleStatusFree {
		ctx.rejectWinterOrder(playerID, order, "noble_not_prisoner")
		return
	}
	capitalTerritoryID, _, hasCapital := ctx.capitalTerritory(playerID)
	if !hasCapital {
		ctx.rejectWinterOrder(playerID, order, "no_capital")
		return
	}
	spent, paid := ctx.payWinterCost(playerID, capitalTerritoryID, ctx.balance.Costs.Liberation)
	if !paid {
		ctx.rejectWinterOrder(playerID, order, "insufficient_resources")
		return
	}
	previousStatus := noble.Status
	noble.Status = models.NobleStatusFree
	noble.LocationID = capitalTerritoryID
	ctx.events = append(ctx.events, Event{
		Type:           EventTypeLiberation,
		Phase:          winterPhase,
		OwnerID:        playerID,
		OrderID:        order.ID,
		NobleID:        noble.ID,
		NobleCode:      models.NobleCode(noble.Code),
		NobleName:      noble.Name,
		PreviousStatus: previousStatus,
		Status:         noble.Status,
		TerritoryID:    capitalTerritoryID,
		ResourceSpent:  spent,
	})
}

func (ctx *resolutionContext) rejectWinterOrder(playerID models.PlayerID, order models.WinterOrder, reason string) {
	orderCopy := order
	ctx.events = append(ctx.events, Event{
		Type:          EventTypeRejected,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       order.ID,
		ResourceSpent: 0,
		Reason:        reason,
		WinterOrder:   &orderCopy,
	})
}

func (ctx *resolutionContext) territoryExists(territoryID models.TerritoryID) bool {
	return ctx.territoriesByID[territoryID] != nil
}

func (ctx *resolutionContext) controlsTerritory(playerID models.PlayerID, territoryID models.TerritoryID) bool {
	state, exists := ctx.state.TerritoryStates[territoryID]
	return exists && state.OwnerID != nil && *state.OwnerID == playerID
}

func (ctx *resolutionContext) playerByID(playerID models.PlayerID) *models.Player {
	for index := range ctx.state.Players {
		if ctx.state.Players[index].ID == playerID {
			return &ctx.state.Players[index]
		}
	}
	return nil
}

func (ctx *resolutionContext) capitalTerritory(playerID models.PlayerID) (models.TerritoryID, models.InfraID, bool) {
	player := ctx.playerByID(playerID)
	if player == nil || player.CapitalCastleID == nil {
		return "", "", false
	}
	infrastructure := ctx.infrastructuresByID[*player.CapitalCastleID]
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle || !ctx.controlsTerritory(playerID, infrastructure.TerritoryID) {
		return "", "", false
	}
	return infrastructure.TerritoryID, infrastructure.ID, true
}

func (ctx *resolutionContext) setCapital(playerID models.PlayerID, infrastructureID models.InfraID) {
	player := ctx.playerByID(playerID)
	if player == nil {
		return
	}
	capitalID := infrastructureID
	player.CapitalCastleID = &capitalID
}

func (ctx *resolutionContext) hasAvailableFirstName(firstNames []assetgen.Asset) bool {
	for _, firstName := range firstNames {
		if _, exists := ctx.noblesByCode[models.NobleCode(firstName.Code)]; !exists {
			return true
		}
	}
	return false
}

func (ctx *resolutionContext) drawFirstName(rng *rand.Rand, firstNames []assetgen.Asset) assetgen.Asset {
	start := rng.IntN(len(firstNames))
	for offset := 0; offset < len(firstNames); offset++ {
		candidate := firstNames[(start+offset)%len(firstNames)]
		if _, exists := ctx.noblesByCode[models.NobleCode(candidate.Code)]; !exists {
			return candidate
		}
	}
	return assetgen.Asset{}
}

func newWinterRNG(seed string, turn int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|winter-noble|%d", seed, turn)))
	lo := binary.BigEndian.Uint64(digest[:8])
	hi := binary.BigEndian.Uint64(digest[8:16])
	return rand.New(rand.NewPCG(lo, hi))
}

func (ctx *resolutionContext) payWinterCost(playerID models.PlayerID, targetID models.TerritoryID, cost int) (int, bool) {
	if cost == 0 {
		return 0, true
	}
	sources := ctx.winterPaymentSources(playerID, targetID)
	total := 0
	for _, sourceID := range sources {
		total += ctx.state.TerritoryStates[sourceID].Resources
	}
	if total < cost {
		return 0, false
	}
	remaining := cost
	spent := 0
	for _, sourceID := range sources {
		if remaining == 0 {
			break
		}
		state := ctx.state.TerritoryStates[sourceID]
		paid := min(state.Resources, remaining)
		state.Resources -= paid
		ctx.state.TerritoryStates[sourceID] = state
		remaining -= paid
		spent += paid
	}
	return spent, true
}

func (ctx *resolutionContext) winterPaymentSources(playerID models.PlayerID, targetID models.TerritoryID) []models.TerritoryID {
	distances := ctx.winterDistances(targetID)
	type source struct {
		territoryID models.TerritoryID
		distance    int
		reachable   bool
	}
	sources := make([]source, 0)
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		if !ctx.controlsTerritory(playerID, territoryID) || !ctx.hasSettlement(territoryID) {
			continue
		}
		distance, reachable := distances[territoryID]
		sources = append(sources, source{territoryID: territoryID, distance: distance, reachable: reachable})
	}
	sort.Slice(sources, func(i, j int) bool {
		if sources[i].reachable != sources[j].reachable {
			return sources[i].reachable
		}
		if sources[i].reachable && sources[i].distance != sources[j].distance {
			return sources[i].distance < sources[j].distance
		}
		leftCode := ctx.territoryCode(sources[i].territoryID)
		rightCode := ctx.territoryCode(sources[j].territoryID)
		if leftCode != rightCode {
			return leftCode < rightCode
		}
		return sources[i].territoryID < sources[j].territoryID
	})
	ordered := make([]models.TerritoryID, len(sources))
	for index, source := range sources {
		ordered[index] = source.territoryID
	}
	return ordered
}

func (ctx *resolutionContext) winterDistances(startID models.TerritoryID) map[models.TerritoryID]int {
	if !ctx.territoryExists(startID) {
		return map[models.TerritoryID]int{}
	}
	distances := map[models.TerritoryID]int{startID: 0}
	queue := []models.TerritoryID{startID}
	for len(queue) > 0 {
		territoryID := queue[0]
		queue = queue[1:]
		for _, neighborID := range ctx.sortedNeighbors(territoryID) {
			if _, visited := distances[neighborID]; visited {
				continue
			}
			distances[neighborID] = distances[territoryID] + 1
			queue = append(queue, neighborID)
		}
	}
	return distances
}

func (ctx *resolutionContext) millCanBeBuiltAt(territoryID models.TerritoryID) bool {
	if ctx.hasSettlement(territoryID) {
		return true
	}
	for _, neighborID := range ctx.sortedNeighbors(territoryID) {
		if ctx.hasSettlement(neighborID) {
			return true
		}
	}
	return false
}

func isBuildableInfrastructure(infrastructureType models.InfraType) bool {
	switch infrastructureType {
	case models.InfraTypeMill, models.InfraTypeCastle, models.InfraTypeSupplyDepot:
		return true
	}
	return false
}

func infrastructureCost(costs assetgen.Costs, infrastructureType models.InfraType) (int, bool) {
	switch infrastructureType {
	case models.InfraTypeMill:
		return costs.Mill, true
	case models.InfraTypeCastle:
		return costs.Castle, true
	case models.InfraTypeSupplyDepot:
		return costs.SupplyDepot, true
	}
	return 0, false
}

func (ctx *resolutionContext) addWinterInfrastructure(infrastructureType models.InfraType, territoryID models.TerritoryID) *models.Infrastructure {
	infrastructure := models.Infrastructure{
		ID:          nextInfrastructureID(ctx.state.Infrastructures),
		Type:        infrastructureType,
		Level:       1,
		TerritoryID: territoryID,
	}
	ctx.state.Infrastructures = append(ctx.state.Infrastructures, infrastructure)
	state := ctx.state.TerritoryStates[territoryID]
	state.Infrastructures = append(state.Infrastructures, infrastructure.ID)
	ctx.state.TerritoryStates[territoryID] = state
	ctx.rebuildIndexes()
	return ctx.infrastructuresByID[infrastructure.ID]
}

func nextNobleID(nobles []models.Noble) models.NobleID {
	return models.NobleID(fmt.Sprintf("N%d", nextIDSequence(nobleIDs(nobles), 'N')))
}

func nextInfrastructureID(infrastructures []models.Infrastructure) models.InfraID {
	return models.InfraID(fmt.Sprintf("I%d", nextIDSequence(infrastructureIDs(infrastructures), 'I')))
}

func nextIDSequence(ids []string, prefix byte) int {
	next := 1
	for _, id := range ids {
		if len(id) < 2 || id[0] != prefix {
			continue
		}
		sequence, err := strconv.Atoi(id[1:])
		if err == nil && sequence >= next {
			next = sequence + 1
		}
	}
	return next
}

func nobleIDs(nobles []models.Noble) []string {
	ids := make([]string, len(nobles))
	for index, noble := range nobles {
		ids[index] = string(noble.ID)
	}
	return ids
}

func infrastructureIDs(infrastructures []models.Infrastructure) []string {
	ids := make([]string, len(infrastructures))
	for index, infrastructure := range infrastructures {
		ids[index] = string(infrastructure.ID)
	}
	return ids
}

func winterStocks(ctx *resolutionContext) map[models.TerritoryID]int {
	stocks := make(map[models.TerritoryID]int, len(ctx.state.TerritoryStates))
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		stocks[territoryID] = ctx.state.TerritoryStates[territoryID].Resources
	}
	return stocks
}

func (ctx *resolutionContext) conserveWinterStocks() {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if !ctx.hasSettlement(territoryID) {
			state.Resources = 0
			ctx.state.TerritoryStates[territoryID] = state
			continue
		}
		state.Resources = (state.Resources + ctx.balance.WinterStockDivisor - 1) / ctx.balance.WinterStockDivisor
		ctx.state.TerritoryStates[territoryID] = state
	}
}

func (ctx *resolutionContext) repatriateWinterStocks() {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if state.OwnerID == nil || !ctx.hasSettlement(territoryID) {
			continue
		}
		capitalTerritoryID, _, hasCapital := ctx.capitalTerritory(*state.OwnerID)
		if !hasCapital || capitalTerritoryID == territoryID {
			continue
		}
		maximum := ctx.balance.VillageStockCap
		if ctx.hasCastle(territoryID) {
			maximum = ctx.balance.CastleStockCap
		}
		if state.Resources <= maximum {
			continue
		}
		surplus := state.Resources - maximum
		state.Resources = maximum
		ctx.state.TerritoryStates[territoryID] = state
		capitalState := ctx.state.TerritoryStates[capitalTerritoryID]
		capitalState.Resources += surplus
		ctx.state.TerritoryStates[capitalTerritoryID] = capitalState
	}
}

func (ctx *resolutionContext) emitWinterStockEvents(stockBefore map[models.TerritoryID]int) {
	for _, territoryID := range sortedStateTerritoryIDs(ctx) {
		state := ctx.state.TerritoryStates[territoryID]
		if !ctx.hasSettlement(territoryID) && stockBefore[territoryID] == 0 && state.Resources == 0 {
			continue
		}
		ownerID := models.PlayerID("")
		if state.OwnerID != nil {
			ownerID = *state.OwnerID
		}
		ctx.events = append(ctx.events, Event{
			Type:        EventTypeWinterStock,
			Phase:       winterPhase,
			OwnerID:     ownerID,
			TerritoryID: territoryID,
			StockBefore: stockBefore[territoryID],
			StockAfter:  state.Resources,
		})
	}
}
