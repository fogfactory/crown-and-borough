package store

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

type IDGenerator func() (GameID, error)

type MemoryStoreOptions struct {
	PrivacyTracker PrivacyTracker
	IDGenerator    IDGenerator
}

type MemoryStore struct {
	indexMu sync.RWMutex
	idMu    sync.Mutex
	games   map[GameID]*memoryGame

	balance assetgen.Balance
	assets  assetgen.Assets
	options MemoryStoreOptions
}

type memoryGame struct {
	mu sync.RWMutex

	id          GameID
	name        string
	seed        string
	status      Status
	winner      *models.PlayerID
	players     []PlayerSlot
	mapData     mapgen.MapData
	state       *models.GameState
	submissions map[models.PlayerID]engine.OrdersInput
	reports     []ReportRecord
	revision    Revision
	createdBy   string
}

func NewMemoryStore(balance assetgen.Balance, assets assetgen.Assets) *MemoryStore {
	return NewMemoryStoreWithOptions(balance, assets, MemoryStoreOptions{})
}

func NewMemoryStoreWithOptions(balance assetgen.Balance, assets assetgen.Assets, options MemoryStoreOptions) *MemoryStore {
	if options.IDGenerator == nil {
		options.IDGenerator = newUUIDv4
	}
	if options.PrivacyTracker == nil {
		options.PrivacyTracker = trackOwnerChainKnowledge
	}
	return &MemoryStore{
		games:   make(map[GameID]*memoryGame),
		balance: balance,
		assets:  assets,
		options: options,
	}
}

// trackOwnerChainKnowledge is a conservative default for callers that use the
// store without the richer API privacy tracker. The API server injects
// TrackTurnPrivacy, which additionally handles hostages, replacement history,
// and combat audiences.
func trackOwnerChainKnowledge(_ *models.GameState, after *models.GameState, _ engine.OrdersInput, _ engine.TurnReport) {
	if after == nil {
		return
	}
	if after.Privacy == nil {
		after.Privacy = &models.PrivacyMeta{}
	}
	if after.Privacy.ChainKnowledge == nil {
		after.Privacy.ChainKnowledge = make(map[models.PlayerID]map[models.ChainID]models.ChainSnapshot)
	}
	armies := make(map[models.ArmyID]models.Army, len(after.Armies))
	for _, army := range after.Armies {
		armies[army.ID] = army
	}
	nobles := make(map[models.NobleID]models.Noble, len(after.Nobles))
	for _, noble := range after.Nobles {
		nobles[noble.ID] = noble
	}
	for _, chain := range after.Chains {
		army, armyOK := armies[chain.ArmyID]
		noble, nobleOK := nobles[chain.NobleID]
		if !armyOK || !nobleOK {
			continue
		}
		orders, err := cloneJSON(chain.Orders)
		if err != nil {
			continue
		}
		snapshot := models.ChainSnapshot{
			ID:           chain.ID,
			NobleID:      chain.NobleID,
			ArmyID:       chain.ArmyID,
			Orders:       orders,
			CurrentIndex: chain.CurrentIndex,
			CapturedTurn: after.Turn,
		}
		for _, playerID := range []models.PlayerID{army.OwnerID, noble.OwnerID} {
			if playerID == "" {
				continue
			}
			if after.Privacy.ChainKnowledge[playerID] == nil {
				after.Privacy.ChainKnowledge[playerID] = make(map[models.ChainID]models.ChainSnapshot)
			}
			after.Privacy.ChainKnowledge[playerID][chain.ID] = snapshot
		}
	}
}

func (s *MemoryStore) Create(_ context.Context, actor Actor, request CreateRequest) (GameSnapshot, error) {
	players, err := normalizePlayers(request.Players)
	if err != nil {
		return GameSnapshot{}, err
	}
	s.idMu.Lock()
	id, err := s.options.IDGenerator()
	s.idMu.Unlock()
	if err != nil {
		return GameSnapshot{}, fmt.Errorf("store: generate game id: %w", err)
	}
	if id == "" {
		return GameSnapshot{}, fmt.Errorf("store: generated empty game id")
	}
	seed := strings.TrimSpace(request.Seed)
	if seed == "" {
		seed = string(id)
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Crown & Borough"
	}

	state, err := engine.CreateGame(seed, players, s.balance, s.assets)
	if err != nil {
		return GameSnapshot{}, err
	}
	mapData, err := engine.GenerateMap(seed, len(players), s.assets)
	if err != nil {
		return GameSnapshot{}, err
	}

	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		actorID = string(state.Players[0].ID)
	}
	slots := make([]PlayerSlot, len(state.Players))
	for index, player := range state.Players {
		slotActorID := string(player.ID)
		if index == 0 {
			slotActorID = actorID
		} else if slotActorID == actorID {
			// The creator owns the first slot. Avoid assigning the same actor to
			// a later development slot when the creator ID happens to be P2, P3,
			// and so on.
			slotActorID = "slot:" + string(player.ID)
		}
		slots[index] = PlayerSlot{
			ID:      player.ID,
			Name:    player.Name,
			Color:   player.Color,
			ActorID: slotActorID,
		}
	}

	game := &memoryGame{
		id:          id,
		name:        name,
		seed:        seed,
		status:      StatusPlaying,
		players:     slots,
		mapData:     mapData,
		state:       state,
		submissions: make(map[models.PlayerID]engine.OrdersInput),
		revision:    1,
		createdBy:   actorID,
	}

	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	if _, exists := s.games[id]; exists {
		return GameSnapshot{}, fmt.Errorf("store: generated duplicate game id %q", id)
	}
	s.games[id] = game
	return s.snapshotLocked(game)
}

func (s *MemoryStore) List(_ context.Context, actor Actor) ([]GameSnapshot, error) {
	s.indexMu.RLock()
	games := make([]*memoryGame, 0, len(s.games))
	for _, game := range s.games {
		games = append(games, game)
	}
	s.indexMu.RUnlock()

	result := make([]GameSnapshot, 0, len(games))
	for _, game := range games {
		game.mu.RLock()
		if _, ok := game.playerForActorLocked(actor); !ok {
			game.mu.RUnlock()
			continue
		}
		snapshot, err := s.snapshotLocked(game)
		game.mu.RUnlock()
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	// UUIDs are random, so sort explicitly to keep list responses stable.
	sortSnapshots(result)
	return result, nil
}

func (s *MemoryStore) Get(_ context.Context, actor Actor, id GameID) (GameSnapshot, error) {
	game, err := s.game(id)
	if err != nil {
		return GameSnapshot{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if _, ok := game.playerForActorLocked(actor); !ok {
		return GameSnapshot{}, ErrNotMember
	}
	return s.snapshotLocked(game)
}

func (s *MemoryStore) Map(_ context.Context, actor Actor, id GameID) (mapgen.MapData, error) {
	game, err := s.game(id)
	if err != nil {
		return mapgen.MapData{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if _, ok := game.playerForActorLocked(actor); !ok {
		return mapgen.MapData{}, ErrNotMember
	}
	return cloneMap(game.mapData), nil
}

func (s *MemoryStore) State(ctx context.Context, actor Actor, id GameID) (GameSnapshot, error) {
	return s.Get(ctx, actor, id)
}

func (s *MemoryStore) Supply(_ context.Context, actor Actor, id GameID, territoryID models.TerritoryID) (engine.SupplyLine, error) {
	game, err := s.game(id)
	if err != nil {
		return engine.SupplyLine{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if _, ok := game.playerForActorLocked(actor); !ok {
		return engine.SupplyLine{}, ErrNotMember
	}
	return engine.FindSupply(game.state, s.balance, territoryID)
}

func (s *MemoryStore) Submit(_ context.Context, actor Actor, id GameID, request SubmitRequest) (SubmitResult, error) {
	game, err := s.game(id)
	if err != nil {
		return SubmitResult{}, err
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	playerID, ok := game.playerForActorLocked(actor)
	if !ok {
		return SubmitResult{}, ErrNotMember
	}
	return s.submitLocked(game, playerID, request)
}

func (s *MemoryStore) Resolve(ctx context.Context, actor Actor, id GameID) (SubmitResult, error) {
	return s.ResolveAt(ctx, actor, id, 0)
}

func (s *MemoryStore) ResolveAt(_ context.Context, actor Actor, id GameID, expected Revision) (SubmitResult, error) {
	game, err := s.game(id)
	if err != nil {
		return SubmitResult{}, err
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	playerID, ok := game.playerForActorLocked(actor)
	if !ok {
		return SubmitResult{}, ErrNotMember
	}
	if game.status == StatusFinished {
		return SubmitResult{}, ErrGameFinished
	}
	if expected != 0 && expected != game.revision {
		return SubmitResult{}, ErrRevisionConflict
	}
	return s.resolveLocked(game, playerID, true)
}

func (s *MemoryStore) Reports(_ context.Context, actor Actor, id GameID) ([]ReportRecord, error) {
	game, err := s.game(id)
	if err != nil {
		return nil, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if _, ok := game.playerForActorLocked(actor); !ok {
		return nil, ErrNotMember
	}
	return cloneReports(game.reports), nil
}

func (s *MemoryStore) Report(ctx context.Context, actor Actor, id GameID, index int) (ReportRecord, error) {
	reports, err := s.Reports(ctx, actor, id)
	if err != nil {
		return ReportRecord{}, err
	}
	if index < 0 || index >= len(reports) {
		return ReportRecord{}, ErrInvalidReport
	}
	return reports[index], nil
}

func (s *MemoryStore) game(id GameID) (*memoryGame, error) {
	s.indexMu.RLock()
	game := s.games[id]
	s.indexMu.RUnlock()
	if game == nil {
		return nil, ErrUnknownGame
	}
	return game, nil
}

func (s *MemoryStore) submitLocked(game *memoryGame, playerID models.PlayerID, request SubmitRequest) (SubmitResult, error) {
	if game.status == StatusFinished {
		return SubmitResult{}, ErrGameFinished
	}
	if !game.isAliveLocked(playerID) {
		return SubmitResult{}, ErrEliminated
	}
	if request.ExpectedRevision != 0 && request.ExpectedRevision != game.revision {
		return SubmitResult{}, ErrRevisionConflict
	}

	input, err := normalizeSubmission(playerID, request)
	if err != nil {
		return SubmitResult{}, err
	}
	if game.submissions == nil {
		game.submissions = make(map[models.PlayerID]engine.OrdersInput)
	}
	previous, replaced := game.submissions[playerID]
	game.submissions[playerID] = input

	submitted, remaining := game.submissionStatusLocked()
	if request.Force || len(remaining) == 0 {
		result, resolveErr := s.resolveLocked(game, playerID, request.Force, submitted, remaining)
		if resolveErr != nil {
			if replaced {
				game.submissions[playerID] = previous
			} else {
				delete(game.submissions, playerID)
			}
			return SubmitResult{}, resolveErr
		}
		return result, nil
	}
	if _, err := engine.ResolveTurn(game.state, s.balance, input); err != nil {
		if replaced {
			game.submissions[playerID] = previous
		} else {
			delete(game.submissions, playerID)
		}
		return SubmitResult{}, err
	}
	game.revision++
	snapshot, snapshotErr := s.snapshotLocked(game)
	if snapshotErr != nil {
		game.revision--
		if replaced {
			game.submissions[playerID] = previous
		} else {
			delete(game.submissions, playerID)
		}
		return SubmitResult{}, snapshotErr
	}
	return SubmitResult{
		Status:    "pending",
		Player:    playerID,
		Submitted: submitted,
		Remaining: remaining,
		Snapshot:  snapshot,
	}, nil
}

func (s *MemoryStore) resolveLocked(game *memoryGame, playerID models.PlayerID, forced bool, status ...[]models.PlayerID) (SubmitResult, error) {
	var submitted, remaining []models.PlayerID
	if len(status) == 2 {
		submitted, remaining = status[0], status[1]
	} else {
		submitted, remaining = game.submissionStatusLocked()
	}
	combined := engine.OrdersInput{
		Chains: []engine.ChainSubmission{},
		Winter: []engine.WinterSubmission{},
	}
	for _, player := range game.state.Players {
		input, exists := game.submissions[player.ID]
		if !exists {
			continue
		}
		combined.Chains = append(combined.Chains, input.Chains...)
		combined.Winter = append(combined.Winter, input.Winter...)
	}

	before := game.state
	report, err := engine.ResolveTurn(before, s.balance, combined)
	if err != nil {
		return SubmitResult{}, err
	}
	if s.options.PrivacyTracker != nil {
		s.options.PrivacyTracker(before, report.State, combined, report)
	}
	record := ReportRecord{
		Report:  cloneReport(report),
		Privacy: clonePrivacy(report.State.Privacy),
	}
	nextReports := append(cloneReports(game.reports), record)
	nextRevision := game.revision + 1
	nextGame := &memoryGame{
		id:          game.id,
		name:        game.name,
		seed:        game.seed,
		status:      game.status,
		winner:      clonePlayerID(game.winner),
		players:     append([]PlayerSlot(nil), game.players...),
		mapData:     game.mapData,
		state:       report.State,
		submissions: make(map[models.PlayerID]engine.OrdersInput),
		reports:     nextReports,
		revision:    nextRevision,
		createdBy:   game.createdBy,
	}
	nextGame.updateStatusLocked()
	snapshot, err := s.snapshotLocked(nextGame)
	if err != nil {
		return SubmitResult{}, err
	}
	game.state = nextGame.state
	game.reports = nextReports
	game.submissions = nextGame.submissions
	game.revision = nextGame.revision
	game.status = nextGame.status
	game.winner = nextGame.winner

	result := SubmitResult{
		Status:    "resolved",
		Player:    playerID,
		Submitted: submitted,
		Remaining: remaining,
		Resolved:  true,
		Forced:    forced,
		Report:    &ReportRecord{Report: cloneReport(record.Report), Privacy: clonePrivacy(record.Privacy)},
		Snapshot:  snapshot,
	}
	return result, nil
}

func (game *memoryGame) playerForActorLocked(actor Actor) (models.PlayerID, bool) {
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return "", false
	}
	for _, player := range game.players {
		if player.ActorID == actorID {
			return player.ID, true
		}
	}
	return "", false
}

func (game *memoryGame) isAliveLocked(playerID models.PlayerID) bool {
	for _, territory := range game.state.TerritoryStates {
		if territory.OwnerID != nil && *territory.OwnerID == playerID {
			return true
		}
	}
	for _, army := range game.state.Armies {
		if army.OwnerID == playerID {
			return true
		}
	}
	return false
}

func (game *memoryGame) submissionStatusLocked() ([]models.PlayerID, []models.PlayerID) {
	submitted := make([]models.PlayerID, 0, len(game.submissions))
	remaining := make([]models.PlayerID, 0, len(game.players))
	for _, player := range game.players {
		if _, exists := game.submissions[player.ID]; exists {
			submitted = append(submitted, player.ID)
			continue
		}
		if game.isAliveLocked(player.ID) {
			remaining = append(remaining, player.ID)
		}
	}
	return submitted, remaining
}

func (game *memoryGame) updateStatusLocked() {
	alive := make([]models.PlayerID, 0, len(game.players))
	for _, player := range game.players {
		if game.isAliveLocked(player.ID) {
			alive = append(alive, player.ID)
		}
	}
	if len(alive) > 1 {
		return
	}
	game.status = StatusFinished
	if len(alive) == 1 {
		winner := alive[0]
		game.winner = &winner
	} else {
		game.winner = nil
	}
}

func (s *MemoryStore) snapshotLocked(game *memoryGame) (GameSnapshot, error) {
	state, err := cloneState(game.state)
	if err != nil {
		return GameSnapshot{}, err
	}
	return GameSnapshot{
		ID:          game.id,
		Name:        game.name,
		Seed:        game.seed,
		Status:      game.status,
		Winner:      clonePlayerID(game.winner),
		Players:     append([]PlayerSlot(nil), game.players...),
		Map:         cloneMap(game.mapData),
		State:       state,
		Submissions: cloneSubmissions(game.submissions),
		Reports:     cloneReports(game.reports),
		Revision:    game.revision,
		CreatedBy:   game.createdBy,
	}, nil
}

func normalizePlayers(players []engine.PlayerInit) ([]engine.PlayerInit, error) {
	if len(players) < MinimumPlayers || len(players) > MaximumPlayers {
		return nil, ErrInvalidPlayers
	}
	result := make([]engine.PlayerInit, len(players))
	for index, player := range players {
		result[index] = player
		result[index].Name = strings.TrimSpace(result[index].Name)
		if result[index].Name == "" {
			result[index].Name = fmt.Sprintf("P%d", index+1)
		}
	}
	return result, nil
}

func normalizeSubmission(playerID models.PlayerID, request SubmitRequest) (engine.OrdersInput, error) {
	input := engine.OrdersInput{
		Chains: append([]engine.ChainSubmission(nil), request.Chains...),
		Winter: append([]engine.WinterSubmission(nil), request.Winter...),
	}
	for index := range input.Chains {
		if input.Chains[index].Player != "" && input.Chains[index].Player != playerID {
			return engine.OrdersInput{}, fmt.Errorf("store: chain %d belongs to another player", index+1)
		}
		input.Chains[index].Player = playerID
	}
	for index := range input.Winter {
		if input.Winter[index].Player != "" && input.Winter[index].Player != playerID {
			return engine.OrdersInput{}, fmt.Errorf("store: winter order %d belongs to another player", index+1)
		}
		input.Winter[index].Player = playerID
	}
	return input, nil
}

func cloneState(source *models.GameState) (*models.GameState, error) {
	if source == nil {
		return nil, nil
	}
	return cloneJSON(source)
}

func cloneMap(source mapgen.MapData) mapgen.MapData {
	clone, err := cloneJSON(source)
	if err != nil {
		panic(err)
	}
	return clone
}

func cloneSubmissions(source map[models.PlayerID]engine.OrdersInput) map[models.PlayerID]engine.OrdersInput {
	if source == nil {
		return nil
	}
	clone, err := cloneJSON(source)
	if err != nil {
		panic(err)
	}
	return clone
}

func cloneReports(source []ReportRecord) []ReportRecord {
	if source == nil {
		return nil
	}
	clone := make([]ReportRecord, len(source))
	for index, record := range source {
		clone[index] = record
		clone[index].Report = cloneReport(record.Report)
		clone[index].Privacy = clonePrivacy(record.Privacy)
	}
	return clone
}

func cloneReport(source engine.TurnReport) engine.TurnReport {
	clone, err := cloneJSON(source)
	if err != nil {
		panic(err)
	}
	return clone
}

func clonePrivacy(source *models.PrivacyMeta) *models.PrivacyMeta {
	if source == nil {
		return nil
	}
	clone, err := cloneJSON(source)
	if err != nil {
		panic(err)
	}
	return clone
}

func clonePlayerID(source *models.PlayerID) *models.PlayerID {
	if source == nil {
		return nil
	}
	clone := *source
	return &clone
}

func cloneJSON[T any](source T) (T, error) {
	data, err := jsonMarshal(source)
	if err != nil {
		var zero T
		return zero, err
	}
	var clone T
	if err := jsonUnmarshal(data, &clone); err != nil {
		var zero T
		return zero, err
	}
	return clone, nil
}

// These small variables keep the cloning helpers easy to replace with a
// persistence codec later without coupling the store to an HTTP package.
var jsonMarshal = func(value any) ([]byte, error) {
	return json.Marshal(value)
}

var jsonUnmarshal = func(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func newUUIDv4() (GameID, error) {
	var bytes [16]byte
	if _, err := cryptorand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return GameID(encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]), nil
}

func sortSnapshots(snapshots []GameSnapshot) {
	slices.SortFunc(snapshots, func(left, right GameSnapshot) int {
		return strings.Compare(string(left.ID), string(right.ID))
	})
}
