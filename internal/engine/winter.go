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
	return ResolveWinterWithDeckOrders(game, balance, orders, nil)
}

func ResolveWinterWithDeckOrders(
	game *models.GameState,
	balance assetgen.Balance,
	orders map[models.PlayerID][]models.WinterOrder,
	deckOrders map[models.PlayerID][]models.DeckOrder,
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
	if err := validateDeckOrders(game, balance, deckOrders); err != nil {
		return Resolution{}, err
	}

	state := cloneGameState(game)
	ctx := newResolutionContext(state, balance)
	stockBefore := winterStocks(ctx)
	firstNameRNG := newWinterRNG(state.Seed, state.Turn)
	for _, playerID := range sortedPlayerIDs(state.Players) {
		for _, order := range orders[playerID] {
			executeWinterOrder(ctx, playerID, order, firstNameRNG)
		}
	}
	resolveWinterDeckOrders(ctx, deckOrders)
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
		if sources[i].territoryID != sources[j].territoryID {
			return sources[i].territoryID < sources[j].territoryID
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
