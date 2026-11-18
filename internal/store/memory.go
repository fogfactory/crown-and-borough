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
	"time"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/turn"
)

type IDGenerator func() (GameID, error)

type MemoryStoreOptions struct {
	PrivacyTracker   PrivacyTracker
	IDGenerator      IDGenerator
	ProfileStore     ProfileStore
	InvitationStore  InvitationStore
	StrictMembership bool
	// MaximumPlayers overrides the online limit of MaximumPlayers per game.
	// The local hotseat uses it to allow every player count the engine
	// supports.
	MaximumPlayers int
}

type MemoryStore struct {
	indexMu sync.RWMutex
	idMu    sync.Mutex
	games   map[GameID]*memoryGame

	balance     assetgen.Balance
	assets      assetgen.Assets
	options     MemoryStoreOptions
	profiles    ProfileStore
	invitations InvitationStore
}

type memoryGame struct {
	mu sync.RWMutex

	id               GameID
	name             string
	seed             string
	status           Status
	winner           *models.PlayerID
	players          []PlayerSlot
	mapData          mapgen.MapData
	state            *models.GameState
	submissions      map[models.PlayerID]engine.OrdersInput
	reports          []ReportRecord
	revision         Revision
	createdBy        string
	spectatorUID     string
	strictMembership bool
	incognito        bool
	// yearCount and fixedSeed are kept for the lobby, which has no state yet;
	// fixedSeed is the seed the creator imposed (empty: the vote decides).
	yearCount int
	fixedSeed string
	// lobby holds what each member prepared before the game started, by uid.
	lobby    map[string]LobbyEntry
	joinedAt map[models.PlayerID]time.Time
}

var _ InvitationGameStore = (*MemoryStore)(nil)

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
	if options.ProfileStore == nil {
		options.ProfileStore = NewMemoryProfileStore()
	}
	if options.InvitationStore == nil {
		options.InvitationStore = NewMemoryInvitationStore()
	}
	return &MemoryStore{
		games:       make(map[GameID]*memoryGame),
		balance:     balance,
		assets:      assets,
		options:     options,
		profiles:    options.ProfileStore,
		invitations: options.InvitationStore,
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
	if request.Incognito {
		// Houses start under the engine placeholders; the players name them.
		blanked := make([]engine.PlayerInit, len(request.Players))
		copy(blanked, request.Players)
		for index := range blanked {
			blanked[index].Name = ""
		}
		request.Players = blanked
	}
	players, err := normalizePlayers(request.Players, s.options.MaximumPlayers)
	if err != nil {
		return GameSnapshot{}, err
	}
	yearCount, err := normalizeYearCount(request.YearCount)
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
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Crown & Borough"
	}
	strictMembership := s.options.StrictMembership || request.StrictMembership

	// An online (strict membership) game starts in a lobby: its board is built
	// when the last slot is taken. The local hotseat builds it at once.
	var (
		state   *models.GameState
		mapData mapgen.MapData
		roster  []models.Player
	)
	if strictMembership {
		roster = lobbyRoster(players)
	} else {
		if seed == "" {
			seed = string(id)
		}
		state, err = engine.CreateGameWithYears(seed, players, yearCount, s.balance, s.assets)
		if err != nil {
			return GameSnapshot{}, err
		}
		mapData, err = engine.GenerateMap(seed, len(players), s.assets)
		if err != nil {
			return GameSnapshot{}, err
		}
		roster = state.Players
	}

	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		actorID = string(roster[0].ID)
	}
	slots := make([]PlayerSlot, len(roster))
	joinedAt := make(map[models.PlayerID]time.Time, len(roster))
	now := time.Now().UTC()
	for index, player := range roster {
		slotActorID := ""
		if !strictMembership {
			slotActorID = string(player.ID)
			if index == 0 && !request.Spectate {
				slotActorID = actorID
			} else if slotActorID == actorID {
				// The creator owns the first slot. Avoid assigning the same actor to
				// a later development slot when the creator ID happens to be P2, P3,
				// and so on.
				slotActorID = "slot:" + string(player.ID)
			}
		} else if index == 0 && !request.Spectate {
			slotActorID = actorID
		}
		if slotActorID != "" && !strings.HasPrefix(slotActorID, "slot:") {
			joinedAt[player.ID] = now
		}
		slots[index] = PlayerSlot{
			ID:      player.ID,
			Name:    player.Name,
			Color:   player.Color,
			ActorID: slotActorID,
		}
	}

	game := &memoryGame{
		id:               id,
		name:             name,
		seed:             seed,
		status:           StatusPlaying,
		players:          slots,
		mapData:          mapData,
		state:            state,
		submissions:      make(map[models.PlayerID]engine.OrdersInput),
		revision:         1,
		createdBy:        actorID,
		spectatorUID:     spectatorUID(actorID, request.Spectate),
		joinedAt:         joinedAt,
		strictMembership: strictMembership,
		incognito:        request.Incognito,
		yearCount:        yearCount,
	}
	if strictMembership {
		game.fixedSeed = seed
		game.seed = ""
		game.lobby = make(map[string]LobbyEntry)
	}

	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	if _, exists := s.games[id]; exists {
		return GameSnapshot{}, fmt.Errorf("store: generated duplicate game id %q", id)
	}
	s.games[id] = game
	return s.snapshotForLocked(game, actor)
}

// lobbyRoster gives the slots of a lobby the identity the engine will give
// the houses: the same ids and colors, in the same order.
func lobbyRoster(players []engine.PlayerInit) []models.Player {
	roster := make([]models.Player, len(players))
	for index, init := range players {
		id := init.ID
		if id == "" {
			id = models.PlayerID(fmt.Sprintf("P%d", index+1))
		}
		_, color := engine.SeatArms(index)
		if init.Color != "" {
			color = init.Color
		}
		roster[index] = models.Player{ID: id, Name: init.Name, Color: color}
	}
	return roster
}

func (s *MemoryStore) CreateWithInvitation(ctx context.Context, actor Actor, request CreateRequest) (GameCreation, error) {
	snapshot, err := s.Create(ctx, actor, request)
	if err != nil {
		return GameCreation{}, err
	}
	invitation, err := s.CreateInvitation(ctx, actor, snapshot.ID)
	if err != nil {
		s.indexMu.Lock()
		delete(s.games, snapshot.ID)
		s.indexMu.Unlock()
		return GameCreation{}, err
	}
	return GameCreation{Snapshot: snapshot, Invitation: invitation}, nil
}

func (s *MemoryStore) GetProfile(ctx context.Context, uid string) (PlayerProfile, error) {
	return s.profiles.GetProfile(ctx, uid)
}

func (s *MemoryStore) EnsureProfile(ctx context.Context, actor Actor) (PlayerProfile, error) {
	return s.profiles.EnsureProfile(ctx, actor)
}

func (s *MemoryStore) UpdateNotifications(ctx context.Context, uid string, update NotificationsUpdate) (PlayerProfile, error) {
	return s.profiles.UpdateNotifications(ctx, uid, update)
}

func (s *MemoryStore) UpdateReleaseNotes(ctx context.Context, uid string, update ReleaseNotesUpdate) (PlayerProfile, error) {
	return s.profiles.UpdateReleaseNotes(ctx, uid, update)
}

// UpdateProfile records the new display name as the account name of the
// member's houses, outside incognito games. It never renames a house: the
// house name is the member's own choice, per game.
func (s *MemoryStore) UpdateProfile(ctx context.Context, uid, displayName string) (PlayerProfile, error) {
	profile, err := s.profiles.UpdateProfile(ctx, uid, displayName)
	if err != nil {
		return PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	s.indexMu.RLock()
	games := make([]*memoryGame, 0, len(s.games))
	for _, game := range s.games {
		games = append(games, game)
	}
	s.indexMu.RUnlock()
	for _, game := range games {
		game.mu.Lock()
		if !game.incognito {
			for index := range game.players {
				if game.players[index].ActorID == uid && game.players[index].AccountName != "" {
					game.players[index].AccountName = profile.DisplayName
				}
			}
		}
		game.mu.Unlock()
	}
	return profile, nil
}

func (s *MemoryStore) ListMemberships(_ context.Context, id GameID) ([]Membership, error) {
	game, err := s.game(id)
	if err != nil {
		return nil, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	memberships := make([]Membership, 0, len(game.players))
	for _, player := range game.players {
		if !isAssignedSlot(player) {
			continue
		}
		joinedAt := game.joinedAt[player.ID]
		memberships = append(memberships, Membership{GameID: id, UID: player.ActorID, PlayerID: player.ID, JoinedAt: joinedAt})
	}
	if game.spectatorUID != "" {
		memberships = append(memberships, Membership{GameID: id, UID: game.spectatorUID, Spectator: true})
	}
	return memberships, nil
}

func (s *MemoryStore) ListActorMemberships(_ context.Context, uid string) ([]Membership, error) {
	uid = strings.TrimSpace(uid)
	s.indexMu.RLock()
	games := make([]*memoryGame, 0, len(s.games))
	for _, game := range s.games {
		games = append(games, game)
	}
	s.indexMu.RUnlock()
	result := make([]Membership, 0)
	for _, game := range games {
		game.mu.RLock()
		if game.spectatorUID == uid {
			result = append(result, Membership{GameID: game.id, UID: uid, Spectator: true})
		}
		for _, player := range game.players {
			if !isAssignedSlot(player) || player.ActorID != uid {
				continue
			}
			result = append(result, Membership{GameID: game.id, UID: uid, PlayerID: player.ID, JoinedAt: game.joinedAt[player.ID]})
		}
		game.mu.RUnlock()
	}
	slices.SortFunc(result, func(left, right Membership) int {
		return strings.Compare(string(left.GameID), string(right.GameID))
	})
	return result, nil
}

func (s *MemoryStore) CreateInvitation(ctx context.Context, actor Actor, id GameID) (InvitationSecret, error) {
	game, err := s.game(id)
	if err != nil {
		return InvitationSecret{}, err
	}
	game.mu.RLock()
	createdBy := game.createdBy
	hasFreeSlot := false
	for _, player := range game.players {
		if !isAssignedSlot(player) {
			hasFreeSlot = true
			break
		}
	}
	game.mu.RUnlock()
	if strings.TrimSpace(actor.ID) == "" || strings.TrimSpace(actor.ID) != createdBy {
		return InvitationSecret{}, ErrNotCreator
	}
	if game.strictMembership && !hasFreeSlot {
		return InvitationSecret{}, ErrGameFull
	}
	return s.invitations.CreateInvitation(ctx, id, createdBy)
}

func (s *MemoryStore) Join(ctx context.Context, actor Actor, id GameID, code string) (JoinResult, error) {
	return s.JoinWith(ctx, actor, id, code, JoinOptions{})
}

func (s *MemoryStore) JoinWith(ctx context.Context, actor Actor, id GameID, code string, options JoinOptions) (JoinResult, error) {
	if err := contextError(ctx); err != nil {
		return JoinResult{}, err
	}
	game, err := s.game(id)
	if err != nil {
		return JoinResult{}, err
	}
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return JoinResult{}, ErrNotMember
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	if game.spectatorForActorLocked(actor) {
		return JoinResult{}, ErrSpectator
	}
	if playerID, ok := game.playerForActorLocked(actor); ok {
		snapshot, snapshotErr := s.snapshotForLocked(game, actor)
		if snapshotErr != nil {
			return JoinResult{}, snapshotErr
		}
		return JoinResult{Snapshot: snapshot, Player: playerSlotForID(game.players, playerID), Joined: false}, nil
	}
	invitation, err := s.invitations.LookupInvitation(ctx, code)
	if err != nil {
		return JoinResult{}, err
	}
	if invitation.GameID != id {
		return JoinResult{}, ErrInvalidInvitation
	}
	if game.status == StatusFinished {
		return JoinResult{}, ErrGameFinished
	}
	freeIndex := -1
	for index, player := range game.players {
		if !isAssignedSlot(player) {
			freeIndex = index
			break
		}
	}
	if freeIndex < 0 {
		return JoinResult{}, ErrGameFull
	}
	var entry *LobbyEntry
	if game.state == nil {
		var err error
		if entry, err = game.joinEntryLocked(actorID, options, s.balance.SeedCandidateCount()); err != nil {
			return JoinResult{}, err
		}
	}
	player := &game.players[freeIndex]
	player.ActorID = actorID
	if entry != nil {
		game.lobby[actorID] = *entry
	}
	if game.joinedAt == nil {
		game.joinedAt = make(map[models.PlayerID]time.Time)
	}
	game.joinedAt[player.ID] = time.Now().UTC()
	started := false
	if game.state == nil && SlotsFilled(game.players) {
		if err := s.startLocked(ctx, game); err != nil {
			player.ActorID = ""
			delete(game.joinedAt, player.ID)
			delete(game.lobby, actorID)
			return JoinResult{}, err
		}
		started = true
	}
	game.revision++
	snapshot, err := s.snapshotForLocked(game, actor)
	if err != nil {
		return JoinResult{}, err
	}
	slot := playerSlotForID(game.players, player.ID)
	if playerID, ok := game.playerForActorLocked(actor); ok {
		slot = playerSlotForID(game.players, playerID)
	}
	return JoinResult{Snapshot: snapshot, Player: slot, Joined: true, Started: started}, nil
}

// joinEntryLocked validates what a joiner prepared for the lobby, against the
// names the members entered so far, and returns their entry (nil when they
// prepared nothing).
func (game *memoryGame) joinEntryLocked(uid string, options JoinOptions, candidates int) (*LobbyEntry, error) {
	var entry LobbyEntry
	if strings.TrimSpace(options.HouseName) != "" {
		var others []string
		for _, other := range game.lobby {
			if other.HouseName != "" {
				others = append(others, other.HouseName)
			}
		}
		name, err := ValidatePendingHouseName(options.HouseName, others)
		if err != nil {
			return nil, err
		}
		entry.HouseName = name
	}
	if err := ValidateSeedRanking(options.SeedRanking, candidates); err != nil {
		return nil, err
	}
	entry.SeedRanking = append([]int(nil), options.SeedRanking...)
	if entry.HouseName == "" && len(entry.SeedRanking) == 0 {
		return nil, nil
	}
	return &entry, nil
}

// JoinPreview describes a game to a member or to the holder of its
// invitation code.
func (s *MemoryStore) JoinPreview(ctx context.Context, actor Actor, id GameID, code string) (JoinPreview, error) {
	game, err := s.game(id)
	if err != nil {
		return JoinPreview{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	member := game.viewerForActorLocked(actor)
	if !member {
		if err := s.checkInvitation(ctx, id, code); err != nil {
			return JoinPreview{}, err
		}
	}
	preview := JoinPreview{
		ID:        id,
		Name:      game.name,
		Players:   len(game.players),
		Incognito: game.incognito,
		Started:   game.state != nil,
		Member:    member,
	}
	for _, slot := range game.players {
		if isAssignedSlot(slot) {
			preview.Seated++
		}
	}
	if game.state == nil && game.fixedSeed == "" {
		preview.SeedCandidates = SeedCandidates(id, s.balance.SeedCandidateCount())
	}
	return preview, nil
}

// checkInvitation accepts a live invitation code of the game.
func (s *MemoryStore) checkInvitation(ctx context.Context, id GameID, code string) error {
	invitation, err := s.invitations.LookupInvitation(ctx, code)
	if err != nil {
		return err
	}
	if invitation.GameID != id {
		return ErrInvalidInvitation
	}
	return nil
}

// startLocked closes the lobby: it resolves the seed, seats the members and
// builds the map and the state. The game is left untouched on failure.
func (s *MemoryStore) startLocked(ctx context.Context, game *memoryGame) error {
	displayNames := make(map[string]string, len(game.players))
	for _, slot := range game.players {
		if profile, err := s.profiles.GetProfile(ctx, slot.ActorID); err == nil {
			displayNames[slot.ActorID] = profile.DisplayName
		}
	}
	plan, err := PlanStart(StartInput{
		GameID:       game.id,
		FixedSeed:    game.fixedSeed,
		Incognito:    game.incognito,
		YearCount:    game.yearCount,
		Slots:        game.players,
		Entries:      game.lobby,
		DisplayNames: displayNames,
		Balance:      s.balance,
		Assets:       s.assets,
	})
	if err != nil {
		return err
	}
	// joinedAt follows the members through the draw.
	joinedByActor := make(map[string]time.Time, len(game.players))
	for _, slot := range game.players {
		joinedByActor[slot.ActorID] = game.joinedAt[slot.ID]
	}
	for _, slot := range plan.Slots {
		game.joinedAt[slot.ID] = joinedByActor[slot.ActorID]
	}
	game.players = plan.Slots
	game.seed = plan.Seed
	game.state = plan.State
	game.mapData = plan.Map
	game.lobby = nil
	return nil
}

// RenameHouse names the actor's house. In the lobby it records the name the
// house takes when the game starts; once started it renames the house, until
// the house submits its first orders.
func (s *MemoryStore) RenameHouse(ctx context.Context, actor Actor, id GameID, name string) (RenameResult, error) {
	if err := contextError(ctx); err != nil {
		return RenameResult{}, err
	}
	game, err := s.game(id)
	if err != nil {
		return RenameResult{}, err
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	playerID, ok := game.playerForActorLocked(actor)
	if !ok {
		return RenameResult{}, ErrNotMember
	}
	if game.state == nil {
		uid := strings.TrimSpace(actor.ID)
		var others []string
		for other, entry := range game.lobby {
			if other != uid && entry.HouseName != "" {
				others = append(others, entry.HouseName)
			}
		}
		normalized, err := ValidatePendingHouseName(name, others)
		if err != nil {
			return RenameResult{}, err
		}
		entry := game.lobby[uid]
		entry.HouseName = normalized
		game.lobby[uid] = entry
		snapshot, err := s.snapshotForLocked(game, actor)
		if err != nil {
			return RenameResult{}, err
		}
		return RenameResult{Snapshot: snapshot, Player: playerSlotForID(game.players, playerID), PreviousName: normalized}, nil
	}
	normalized, err := ValidateHouseRename(game.players, playerID, name)
	if err != nil {
		return RenameResult{}, err
	}
	previous := playerSlotForID(game.players, playerID).Name
	if normalized != previous {
		for index := range game.players {
			if game.players[index].ID == playerID {
				game.players[index].Name = normalized
			}
		}
		for index := range game.state.Players {
			if game.state.Players[index].ID == playerID {
				game.state.Players[index].Name = normalized
			}
		}
		game.revision++
	}
	snapshot, err := s.snapshotForLocked(game, actor)
	if err != nil {
		return RenameResult{}, err
	}
	return RenameResult{Snapshot: snapshot, Player: playerSlotForID(game.players, playerID), PreviousName: previous}, nil
}

// SetSeedVote records the actor's ranking of the candidate seeds while the
// lobby is open; an empty ranking withdraws the vote.
func (s *MemoryStore) SetSeedVote(ctx context.Context, actor Actor, id GameID, ranking []int) (GameSnapshot, error) {
	if err := contextError(ctx); err != nil {
		return GameSnapshot{}, err
	}
	game, err := s.game(id)
	if err != nil {
		return GameSnapshot{}, err
	}
	game.mu.Lock()
	defer game.mu.Unlock()
	if _, ok := game.playerForActorLocked(actor); !ok {
		return GameSnapshot{}, ErrNotMember
	}
	if game.state != nil {
		return GameSnapshot{}, ErrGameStarted
	}
	if err := ValidateSeedRanking(ranking, s.balance.SeedCandidateCount()); err != nil {
		return GameSnapshot{}, err
	}
	uid := strings.TrimSpace(actor.ID)
	entry := game.lobby[uid]
	entry.SeedRanking = append([]int(nil), ranking...)
	game.lobby[uid] = entry
	return s.snapshotForLocked(game, actor)
}

// SeedCandidateMap generates the map of candidate seed index for the game's
// player count.
func (s *MemoryStore) SeedCandidateMap(ctx context.Context, actor Actor, id GameID, code string, index int) (mapgen.MapData, error) {
	game, err := s.game(id)
	if err != nil {
		return mapgen.MapData{}, err
	}
	game.mu.RLock()
	allowed := game.viewerForActorLocked(actor)
	count := len(game.players)
	game.mu.RUnlock()
	if !allowed {
		if err := s.checkInvitation(ctx, id, code); err != nil {
			return mapgen.MapData{}, err
		}
	}
	return GenerateCandidateMap(id, index, count, s.balance, s.assets)
}

func playerSlotForID(players []PlayerSlot, id models.PlayerID) PlayerSlot {
	for _, player := range players {
		if player.ID == id {
			return player
		}
	}
	return PlayerSlot{ID: id}
}

func slotIndex(players []PlayerSlot, id models.PlayerID) int {
	for index, player := range players {
		if player.ID == id {
			return index
		}
	}
	return -1
}

func isAssignedSlot(player PlayerSlot) bool {
	return IsAssignedActor(strings.TrimSpace(player.ActorID))
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
		if !game.viewerForActorLocked(actor) {
			game.mu.RUnlock()
			continue
		}
		snapshot, err := s.snapshotForLocked(game, actor)
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
	if !game.viewerForActorLocked(actor) {
		return GameSnapshot{}, ErrNotMember
	}
	return s.snapshotForLocked(game, actor)
}

func (s *MemoryStore) Map(_ context.Context, actor Actor, id GameID) (mapgen.MapData, error) {
	game, err := s.game(id)
	if err != nil {
		return mapgen.MapData{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if !game.viewerForActorLocked(actor) {
		return mapgen.MapData{}, ErrNotMember
	}
	if game.state == nil {
		return mapgen.MapData{}, ErrGameNotStarted
	}
	return cloneMap(game.mapData), nil
}

// State is Get for the projections of the game: an incognito game keeps its
// placement secret until every slot is taken.
func (s *MemoryStore) State(ctx context.Context, actor Actor, id GameID) (GameSnapshot, error) {
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return GameSnapshot{}, err
	}
	if !snapshot.Started() {
		return GameSnapshot{}, ErrGameNotStarted
	}
	return snapshot, nil
}

func (s *MemoryStore) Supply(_ context.Context, actor Actor, id GameID, territoryID models.TerritoryID, special string) (engine.SupplyLine, error) {
	game, err := s.game(id)
	if err != nil {
		return engine.SupplyLine{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if !game.viewerForActorLocked(actor) {
		return engine.SupplyLine{}, ErrNotMember
	}
	if game.state == nil {
		return engine.SupplyLine{}, ErrGameNotStarted
	}
	playerID, _ := game.playerForActorLocked(actor)
	return engine.FindPlayerSupply(game.state, s.balance, territoryID, playerID, special)
}

func (s *MemoryStore) TransferSupply(_ context.Context, actor Actor, id GameID, sourceID, targetID models.TerritoryID) (engine.TransferLine, error) {
	game, err := s.game(id)
	if err != nil {
		return engine.TransferLine{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	if !game.viewerForActorLocked(actor) {
		return engine.TransferLine{}, ErrNotMember
	}
	if game.state == nil {
		return engine.TransferLine{}, ErrGameNotStarted
	}
	return engine.FindTransfer(game.state, s.balance, sourceID, targetID)
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
	if request.Force && strings.TrimSpace(game.createdBy) != strings.TrimSpace(actor.ID) {
		return SubmitResult{}, ErrNotCreator
	}
	if game.state == nil {
		return SubmitResult{}, ErrGameNotStarted
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
	if !ok && !game.spectatorForActorLocked(actor) {
		return SubmitResult{}, ErrNotMember
	}
	if strings.TrimSpace(game.createdBy) != strings.TrimSpace(actor.ID) {
		return SubmitResult{}, ErrNotCreator
	}
	if game.state == nil {
		return SubmitResult{}, ErrGameNotStarted
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
	if !game.viewerForActorLocked(actor) {
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

func (s *MemoryStore) MySubmission(_ context.Context, actor Actor, id GameID) (PlayerSubmission, error) {
	game, err := s.game(id)
	if err != nil {
		return PlayerSubmission{}, err
	}
	game.mu.RLock()
	defer game.mu.RUnlock()
	playerID, ok := game.playerForActorLocked(actor)
	if !ok {
		return PlayerSubmission{}, ErrNotMember
	}
	if game.state == nil {
		return PlayerSubmission{}, ErrGameNotStarted
	}
	currentTurn := game.state.Turn
	season := game.state.Season
	input, submitted := game.submissions[playerID]
	if !submitted {
		return PlayerSubmission{
			Turn:      currentTurn,
			Season:    season,
			Submitted: false,
		}, nil
	}
	return PlayerSubmission{
		Turn:      currentTurn,
		Season:    season,
		Submitted: true,
		Orders:    turn.CloneOrders(input),
	}, nil
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

	input, err := turn.NormalizeSubmission(playerID, engine.OrdersInput{
		Chains:  request.Chains,
		Winter:  request.Winter,
		Special: request.Special,
	})
	if err != nil {
		return SubmitResult{}, err
	}
	if game.submissions == nil {
		game.submissions = make(map[models.PlayerID]engine.OrdersInput)
	}
	previous, replaced := game.submissions[playerID]
	game.submissions[playerID] = input
	slot := slotIndex(game.players, playerID)
	wasLocked := slot >= 0 && game.players[slot].NameLocked
	if slot >= 0 {
		game.players[slot].NameLocked = true
	}
	rollback := func() {
		if replaced {
			game.submissions[playerID] = previous
		} else {
			delete(game.submissions, playerID)
		}
		if slot >= 0 && !wasLocked {
			game.players[slot].NameLocked = false
		}
	}

	submitted, remaining := game.submissionStatusLocked()
	if request.Force || len(remaining) == 0 {
		result, resolveErr := s.resolveLocked(game, playerID, request.Force, submitted, remaining)
		if resolveErr != nil {
			rollback()
			return SubmitResult{}, resolveErr
		}
		return result, nil
	}
	if err := turn.ValidateSubmission(game.state, s.balance, input); err != nil {
		rollback()
		return SubmitResult{}, err
	}
	game.revision++
	snapshot, snapshotErr := s.snapshotLocked(game)
	if snapshotErr != nil {
		game.revision--
		rollback()
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
	before := game.state
	resolution, err := turn.Resolve(before, s.balance, game.submissions, s.options.PrivacyTracker)
	if err != nil {
		return SubmitResult{}, err
	}
	report := resolution.Report
	record := ReportRecord{
		Report:  cloneReport(report),
		Privacy: clonePrivacy(report.State.Privacy),
	}
	nextReports := append(cloneReports(game.reports), record)
	nextRevision := game.revision + 1
	nextGame := &memoryGame{
		id:               game.id,
		name:             game.name,
		seed:             game.seed,
		status:           game.status,
		winner:           clonePlayerID(game.winner),
		players:          append([]PlayerSlot(nil), game.players...),
		mapData:          game.mapData,
		state:            report.State,
		submissions:      make(map[models.PlayerID]engine.OrdersInput),
		reports:          nextReports,
		revision:         nextRevision,
		createdBy:        game.createdBy,
		spectatorUID:     game.spectatorUID,
		strictMembership: game.strictMembership,
		joinedAt:         cloneJoinedAt(game.joinedAt),
	}
	nextGame.updateStatusLocked(s.balance)
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

func cloneJoinedAt(source map[models.PlayerID]time.Time) map[models.PlayerID]time.Time {
	if source == nil {
		return nil
	}
	clone := make(map[models.PlayerID]time.Time, len(source))
	for playerID, joinedAt := range source {
		clone[playerID] = joinedAt
	}
	return clone
}

func (game *memoryGame) playerForActorLocked(actor Actor) (models.PlayerID, bool) {
	return PlayerForActor(game.players, actor)
}

func (game *memoryGame) spectatorForActorLocked(actor Actor) bool {
	return IsSpectator(game.spectatorUID, actor)
}

func (game *memoryGame) viewerForActorLocked(actor Actor) bool {
	if _, ok := game.playerForActorLocked(actor); ok {
		return true
	}
	return game.spectatorForActorLocked(actor)
}

func (game *memoryGame) isAliveLocked(playerID models.PlayerID) bool {
	return engine.PlayerAlive(game.state, playerID)
}

func (game *memoryGame) submissionStatusLocked() ([]models.PlayerID, []models.PlayerID) {
	return turn.Progress(game.state, func(playerID models.PlayerID) bool {
		_, ok := game.submissions[playerID]
		return ok
	})
}

func (game *memoryGame) updateStatusLocked(balance assetgen.Balance) {
	finished, winner := turn.Outcome(game.state, balance)
	game.winner = winner
	game.status = StatusPlaying
	if finished {
		game.status = StatusFinished
	}
}

// snapshotForLocked is snapshotLocked plus the caller's own lobby data.
func (s *MemoryStore) snapshotForLocked(game *memoryGame, actor Actor) (GameSnapshot, error) {
	snapshot, err := s.snapshotLocked(game)
	if err != nil {
		return GameSnapshot{}, err
	}
	if game.state == nil {
		if _, member := game.playerForActorLocked(actor); member {
			entry := game.lobby[strings.TrimSpace(actor.ID)]
			view := &LobbyView{SeedCandidates: []string{}, SeedRanking: append([]int{}, entry.SeedRanking...), HouseName: entry.HouseName}
			if game.fixedSeed == "" {
				view.SeedCandidates = SeedCandidates(game.id, s.balance.SeedCandidateCount())
			}
			snapshot.Lobby = view
		}
	}
	return snapshot, nil
}

func (s *MemoryStore) snapshotLocked(game *memoryGame) (GameSnapshot, error) {
	state, err := cloneState(game.state)
	if err != nil {
		return GameSnapshot{}, err
	}
	var scores map[models.PlayerID]engine.ScoreBreakdown
	yearCount := game.yearCount
	if game.state != nil {
		scores = engine.ComputeScores(game.state, s.balance)
		yearCount = game.state.YearCount
	}
	return GameSnapshot{
		ID:           game.id,
		Name:         game.name,
		Seed:         game.seed,
		YearCount:    yearCount,
		Status:       game.status,
		Winner:       clonePlayerID(game.winner),
		Scores:       scores,
		Players:      append([]PlayerSlot(nil), game.players...),
		Map:          cloneMap(game.mapData),
		State:        state,
		Submissions:  cloneSubmissions(game.submissions),
		Reports:      cloneReports(game.reports),
		Revision:     game.revision,
		CreatedBy:    game.createdBy,
		SpectatorUID: game.spectatorUID,
		Incognito:    game.incognito,
		FixedSeed:    game.fixedSeed,
	}, nil
}

func normalizePlayers(players []engine.PlayerInit, maximum int) ([]engine.PlayerInit, error) {
	if maximum == 0 {
		maximum = MaximumPlayers
	}
	if len(players) < MinimumPlayers || len(players) > maximum {
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

func spectatorUID(actorID string, spectate bool) string {
	if !spectate {
		return ""
	}
	return strings.TrimSpace(actorID)
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
