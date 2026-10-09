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
	if balance.ProsperityLossThreshold < 1 {
		return Resolution{}, fmt.Errorf("engine: resolve winter: prosperity loss threshold must be > 0")
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
	// The winter resolves in the stages of specs/hiver.md. Stage 0 freezes the
	// registry of open elections; stages 1 (papal sanctions), 3 (cardinal
	// actions) and 4 (marriage dissolutions) have no order yet.
	ctx.openWinterElections()
	// Stage 1: papal sanctions, before every other order.
	ctx.resolvePapalSanctions(orders)
	// Stage 2: management orders, players by identifier then sheet order.
	// Candidacies and votes are only recorded here, they resolve in stage 6.
	for _, playerID := range sortedPlayerIDs(state.Players) {
		for _, order := range orders[playerID] {
			executeWinterOrder(ctx, playerID, order, firstNameRNG)
		}
	}
	// Stage 5: marriages.
	ctx.resolveMarriages()
	// Stage 6: elections, counted on one snapshot; stage 7: investiture.
	ctx.resolveWinterElections()
	ctx.investWinterTitles()
	ctx.investPurchasedCardinals()
	// Stage 8: end of winter (hands, vacant fiefs, stocks, prosperity).
	resolveWinterDeckOrders(ctx, deckOrders)
	// No calamity resolves in winter: the winter turn draws and schedules the
	// following year's calamities but applies none.
	ctx.resolveVacantFiefsAtWinterEnd()
	// Prosperity measures conservation loss only, not stock spent or moved by
	// winter/deck orders above: snapshot right before conservation runs, not
	// stockBefore (captured pre-orders, used only by emitWinterStockEvents
	// for the whole-turn stock report).
	stockBeforeConservation := winterStocks(ctx)
	ctx.conserveWinterStocks()
	ctx.resolveProsperity(stockBeforeConservation)
	ctx.repatriateWinterStocks()
	// Reported after repatriation: a capital replaced this same winter by E C
	// still rapatriates its surplus above as the old capital before losing its
	// anchor here (#215).
	ctx.emitAbandonedControl(ctx.startControl)
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
	ctx.rejectWinterOrderAt(playerID, order, reason, "")
}

// rejectWinterOrderAt rejects a winter order like rejectWinterOrder, but
// attributes it to territoryID rather than defaulting to the order's own
// TerritoryID. A found_fief order spans several territories (TerritoryID is
// only the capital), so a rejection caused by one of the other group members
// must point there for the map marker to land on the actual offending
// territory instead of always the capital.
func (ctx *resolutionContext) rejectWinterOrderAt(playerID models.PlayerID, order models.WinterOrder, reason string, territoryID models.TerritoryID) {
	orderCopy := order
	ctx.events = append(ctx.events, Event{
		Type:          EventTypeRejected,
		Phase:         winterPhase,
		OwnerID:       playerID,
		OrderID:       order.ID,
		TerritoryID:   territoryID,
		ResourceSpent: 0,
		Reason:        reason,
		WinterOrder:   &orderCopy,
	})
}

func (ctx *resolutionContext) territoryExists(territoryID models.TerritoryID) bool {
	return ctx.territoriesByID[territoryID] != nil
}

// controlsTerritory reports whether playerID controlled territoryID when the
// winter began. Winter orders change fiefs and capitals but never move an army,
// and what they unanchor only takes effect at the end of the winter (see
// emitAbandonedControl), so every winter rule reads the start-of-winter
// control, whatever an earlier order of the same winter did.
func (ctx *resolutionContext) controlsTerritory(playerID models.PlayerID, territoryID models.TerritoryID) bool {
	return controlledBy(ctx.controllerAtStart, playerID, territoryID)
}

func (ctx *resolutionContext) playerByID(playerID models.PlayerID) *models.Player {
	for index := range ctx.state.Players {
		if ctx.state.Players[index].ID == playerID {
			return &ctx.state.Players[index]
		}
	}
	return nil
}

// capitalTerritory returns the castle territory of playerID's capital. A
// capital castle is always controlled by its owner, since a capital is a
// permanent anchor (models.GameState.TerritoryController) and losing the
// control of it clears the designation.
func (ctx *resolutionContext) capitalTerritory(playerID models.PlayerID) (models.TerritoryID, models.InfraID, bool) {
	player := ctx.playerByID(playerID)
	if player == nil || player.CapitalCastleID == nil {
		return "", "", false
	}
	infrastructure := ctx.infrastructuresByID[*player.CapitalCastleID]
	if infrastructure == nil || infrastructure.Type != models.InfraTypeCastle {
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

func newWinterRNG(seed string, turn int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|winter-noble|%d", seed, turn)))
	lo := binary.BigEndian.Uint64(digest[:8])
	hi := binary.BigEndian.Uint64(digest[8:16])
	return rand.New(rand.NewPCG(lo, hi))
}

func (ctx *resolutionContext) payWinterCost(playerID models.PlayerID, targetID models.TerritoryID, cost int) (int, bool) {
	return ctx.payFromSources(ctx.winterPaymentSources(playerID, targetID), cost)
}

// payFromSources spends cost across sources in order, draining each one
// before moving to the next. It fails without spending anything when the
// combined stock of every source falls short.
func (ctx *resolutionContext) payFromSources(sources []models.TerritoryID, cost int) (int, bool) {
	if cost == 0 {
		return 0, true
	}
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

// millUpgradePaymentSources is the payment order for upgrading the mill at
// millID: its own stock first (the one exception to "only castles and
// villages pay"), then the single settlement its production would be routed
// to (its adjacent castle under the same control, else its adjacent village,
// per millRecipient), then the usual winter payment network as a fallback for
// anything farther out.
func (ctx *resolutionContext) millUpgradePaymentSources(playerID models.PlayerID, millID models.TerritoryID) []models.TerritoryID {
	sources := []models.TerritoryID{millID}
	if recipientID := millRecipientIn(ctx, millID, ctx.controllerAtStart); recipientID != millID && !ctx.occupiedAgainstStartController(recipientID, ctx.currentArmyAt(recipientID)) {
		sources = append(sources, recipientID)
	}
	seen := make(map[models.TerritoryID]bool, len(sources))
	for _, sourceID := range sources {
		seen[sourceID] = true
	}
	for _, sourceID := range ctx.winterPaymentSources(playerID, millID) {
		if seen[sourceID] {
			continue
		}
		sources = append(sources, sourceID)
		seen[sourceID] = true
	}
	return sources
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
		if ctx.occupiedAgainstStartController(territoryID, ctx.currentArmyAt(territoryID)) {
			// A settlement occupied against its controller pays for no
			// winter investment, its own or anyone else's (titres.md).
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
		return millCostForLevel(costs, 1)
	case models.InfraTypeCastle:
		return costs.Castle, true
	case models.InfraTypeSupplyDepot:
		return costs.SupplyDepot, true
	}
	return 0, false
}

func millCostForLevel(costs assetgen.Costs, targetLevel int) (int, bool) {
	if targetLevel < 1 || targetLevel > len(costs.MillLevels) {
		return 0, false
	}
	return costs.MillLevels[targetLevel-1], true
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
	infrastructureID := infrastructure.ID
	state.Infrastructures = &infrastructureID
	ctx.state.TerritoryStates[territoryID] = state
	ctx.rebuildIndexes()
	return ctx.infrastructuresByID[infrastructure.ID]
}

// nextNobleID scans both the living nobles and the lineage of removed ones:
// a dead noble's ID must stay reserved forever rather than be handed out
// again to a later recruit (specs/succession.md § Lignée).
func nextNobleID(nobles []models.Noble, removedNobles []models.RemovedNoble) models.NobleID {
	ids := nobleIDs(nobles)
	for _, removed := range removedNobles {
		ids = append(ids, string(removed.ID))
	}
	return models.NobleID(fmt.Sprintf("N%d", nextIDSequence(ids, 'N')))
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
		if ctx.hasInfrastructure(territoryID, models.InfraTypeSupplyDepot) {
			continue
		}
		// A mill conserves its stock exactly like a castle or village (see
		// #195), but repatriateWinterStocks below never moves it: only a
		// castle or village's surplus is repatriated to the capital.
		if !ctx.hasSettlement(territoryID) && !ctx.hasInfrastructure(territoryID, models.InfraTypeMill) {
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
		controllerID, controlled := ctx.controllerAtStart(territoryID)
		if !controlled || !ctx.hasSettlement(territoryID) {
			continue
		}
		if ctx.occupiedAgainstStartController(territoryID, ctx.currentArmyAt(territoryID)) {
			// Occupied against its controller: its stock stays there and
			// follows the normal conservation rule instead of being
			// repatriated (titres.md).
			continue
		}
		capitalTerritoryID, _, hasCapital := ctx.capitalTerritory(controllerID)
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
		// The control the winter ends on, once emitAbandonedControl has
		// reported what its orders unanchored.
		ownerID, _ := ctx.controllerNow(territoryID)
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
