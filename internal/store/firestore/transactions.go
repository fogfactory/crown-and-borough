package firestorestore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
	"github.com/fogfactory/crown-and-borough/internal/turn"
)

var errAlreadyJoined = errors.New("firestorestore: actor already joined")

type submissionWriteResult struct {
	Revision  store.Revision
	PlayerID  models.PlayerID
	Submitted []models.PlayerID
	Remaining []models.PlayerID
}

type resolutionClaimResult struct {
	Claim    resolutionClaim
	Game     gameDocument
	PlayerID models.PlayerID
}

// Join takes a slot with an invitation code. In a lobby the joiner's entry
// (house name, seed ranking) is validated and recorded in the same
// transaction, and the join that takes the last slot starts the game.
func (s *FirestoreStore) Join(ctx context.Context, actor store.Actor, id store.GameID, code string, options store.JoinOptions) (store.JoinResult, error) {
	if err := s.requireClient(); err != nil {
		return store.JoinResult{}, err
	}
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return store.JoinResult{}, store.ErrNotMember
	}
	invitationCode := strings.ToUpper(strings.TrimSpace(code))
	if !validInvitationCode(invitationCode) {
		return store.JoinResult{}, store.ErrInvalidInvitation
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	invitationHash := store.InvitationCodeHash(invitationCode)
	gameDocumentRef := gameRef(s.client, id)
	canonicalDocumentRef := canonicalRef(s.client, id)
	invitationDocumentRef := invitationRef(s.client, invitationHash)
	var joinedPlayer models.PlayerID
	var lobbyJoin *lobbyJoinResult
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		lobbyJoin = nil
		gameSnapshot, err := transaction.Get(gameDocumentRef)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrUnknownGame
		}
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		if strings.TrimSpace(game.SpectatorUID) == actorID {
			return store.ErrSpectator
		}
		if playerID, ok := playerIDForActor(game, actor); ok {
			joinedPlayer = playerID
			return errAlreadyJoined
		}
		if game.Status == store.StatusFinished {
			return store.ErrGameFinished
		}

		invitationSnapshot, err := transaction.Get(invitationDocumentRef)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrInvalidInvitation
		}
		if err != nil {
			return err
		}
		invitation, err := decodeInvitationDocument(invitationSnapshot)
		if err != nil {
			return err
		}
		if invitation.GameID != id {
			return store.ErrInvalidInvitation
		}
		if !invitation.Active {
			return store.ErrInvitationInactive
		}
		if game.Phase == store.PhaseLobby {
			result, err := s.joinLobby(transaction, actor, game, options)
			lobbyJoin = &result
			return err
		}

		// A started game with a free slot is a development game (see
		// Options.StrictMembership): the slot is claimed, and the house takes the
		// name of the account.
		freeIndex := slices.IndexFunc(game.Players, func(player playerDocument) bool { return !store.IsAssignedActor(player.ActorID) })
		if freeIndex < 0 {
			return store.ErrGameFull
		}
		canonicalSnapshot, err := transaction.Get(canonicalDocumentRef)
		s.recordReads(1)
		if err != nil {
			return mapReadError(err, ErrInconsistentGame)
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		if canonical.Revision != game.Revision || canonical.Turn != game.Turn {
			return store.ErrRevisionConflict
		}
		if canonical.Resolution != nil && s.now().Before(canonical.Resolution.LeaseUntil) {
			return store.ErrRevisionConflict
		}
		state := &models.GameState{}
		if err := decodeJSONMap(canonical.State, state); err != nil {
			return err
		}
		profileName := ""
		profileSnapshot, profileErr := transaction.Get(s.client.Collection("players").Doc(actorID))
		s.recordReads(1)
		if profileErr == nil {
			profileDocument, decodeErr := decodeProfileDocument(profileSnapshot)
			if decodeErr != nil {
				return decodeErr
			}
			profileName = profileDocument.DisplayName
		} else if status.Code(profileErr) != codes.NotFound {
			return profileErr
		}
		player := &game.Players[freeIndex]
		player.ActorID = actorID
		if profileName != "" {
			player.Name = profileName
			player.AccountName = profileName
			for index := range state.Players {
				if state.Players[index].ID == player.ID {
					state.Players[index].Name = profileName
					break
				}
			}
		}
		game.MemberUIDs = append(game.MemberUIDs, actorID)
		joinedPlayer = player.ID
		game.Revision++
		game.UpdatedAt = s.now().UTC()
		canonical.Revision = game.Revision
		canonical.UpdatedAt = game.UpdatedAt
		if canonical.State, err = jsonMap(state); err != nil {
			return err
		}
		if err := state.Validate(); err != nil {
			return err
		}
		// A join changes player metadata in the canonical state: the views are
		// refreshed too, so connected players do not keep stale slot names.
		return s.writeStartedProjections(transaction, game, canonical, state)
	})
	if errors.Is(err, errAlreadyJoined) {
		snapshot, snapshotErr := s.Get(operationContext, actor, id)
		if snapshotErr != nil {
			return store.JoinResult{}, snapshotErr
		}
		return store.JoinResult{Snapshot: snapshot, Player: playerSlot(snapshot.Players, joinedPlayer), Joined: false}, nil
	}
	if err != nil {
		return store.JoinResult{}, wrapTransactionResult(err)
	}
	if lobbyJoin != nil {
		return store.JoinResult{Snapshot: lobbyJoin.snapshot, Player: playerSlot(lobbyJoin.snapshot.Players, lobbyJoin.player), Joined: true, Started: lobbyJoin.started}, nil
	}
	snapshot, err := s.Get(operationContext, actor, id)
	if err != nil {
		return store.JoinResult{}, err
	}
	return store.JoinResult{Snapshot: snapshot, Player: playerSlot(snapshot.Players, joinedPlayer), Joined: true}, nil
}

// lobbyJoinResult is a join into a lobby as the actor sees it, built from what
// the transaction wrote.
type lobbyJoinResult struct {
	snapshot store.GameSnapshot
	player   models.PlayerID
	started  bool
}

// joinLobby takes a free slot of a game whose lobby is open, under the rules of
// store.PlanJoin. The last join closes the lobby: it resolves the seed from the
// votes, seats the members and writes the map, the canonical state, the views
// and the observer view, all in the join's transaction.
func (s *FirestoreStore) joinLobby(transaction *cloudfirestore.Transaction, actor store.Actor, game gameDocument, options store.JoinOptions) (lobbyJoinResult, error) {
	actorID := strings.TrimSpace(actor.ID)
	id := game.ID
	step, err := store.PlanJoin(playerSlots(game.Players), actorID, options, store.OtherHouseNames(game.HouseNames, ""), s.balance.SeedCandidateCount(), game.AwaitingCreator)
	if err != nil {
		return lobbyJoinResult{}, err
	}

	// Every read comes before the first write. The names are on the game
	// document; only the start needs the (private) rankings and the display names.
	var inputs lobbyStartInputs
	if step.Final {
		if inputs, err = s.readLobbyStartInputs(transaction, game, step.Slots); err != nil {
			return lobbyJoinResult{}, err
		}
	}

	game.Players[step.Seat].ActorID = actorID
	game.MemberUIDs = append(game.MemberUIDs, actorID)
	game.Revision++
	game.UpdatedAt = s.now().UTC()
	var ranking []int
	if step.Entry != nil {
		ranking = step.Entry.SeedRanking
		if step.Entry.HouseName != "" {
			game.HouseNames = withHouseName(game.HouseNames, actorID, step.Entry.HouseName)
		}
		if len(ranking) > 0 {
			if err := s.writeSeedRanking(transaction, id, actorID, ranking); err != nil {
				return lobbyJoinResult{}, err
			}
			// The ranking was just written: the start reads it from here.
			inputs.entries = withEntry(inputs.entries, actorID, store.LobbyEntry{SeedRanking: ranking})
		}
	}
	if !step.Final {
		if err := transaction.Set(gameRef(s.client, id), game); err != nil {
			return lobbyJoinResult{}, err
		}
		s.recordWrites(1)
		return lobbyJoinResult{snapshot: s.buildLobbySnapshot(actor, game, ranking), player: step.Slots[step.Seat].ID}, nil
	}
	return s.startLobby(transaction, actor, game, step.Slots, inputs, step.Slots[step.Seat].ID)
}

// lobbyStartInputs is what closing a lobby reads besides the game document:
// the members' lobby entries and account display names.
type lobbyStartInputs struct {
	entries      map[string]store.LobbyEntry
	displayNames map[string]string
}

// readLobbyStartInputs reads what the start of a game needs; every read of a
// transaction comes before its first write.
func (s *FirestoreStore) readLobbyStartInputs(transaction *cloudfirestore.Transaction, game gameDocument, slots []store.PlayerSlot) (lobbyStartInputs, error) {
	entries, err := s.readLobbyEntries(transaction, game.ID)
	if err != nil {
		return lobbyStartInputs{}, err
	}
	displayNames, err := s.loadDisplayNames(transaction, game.Incognito, slots)
	if err != nil {
		return lobbyStartInputs{}, err
	}
	return lobbyStartInputs{entries: entries, displayNames: displayNames}, nil
}

// withEntry sets uid's lobby entry in entries, creating the map if needed.
func withEntry(entries map[string]store.LobbyEntry, uid string, entry store.LobbyEntry) map[string]store.LobbyEntry {
	if entries == nil {
		entries = make(map[string]store.LobbyEntry)
	}
	entries[uid] = entry
	return entries
}

// startLobby closes the lobby: it resolves the seed from the votes, seats the
// members and writes the map, the canonical state, the views and the observer
// view, all in the caller's transaction. game already carries the new revision.
// player is the house the caller gets when the draw does not move them.
func (s *FirestoreStore) startLobby(transaction *cloudfirestore.Transaction, actor store.Actor, game gameDocument, slots []store.PlayerSlot, inputs lobbyStartInputs, player models.PlayerID) (lobbyJoinResult, error) {
	id := game.ID
	actorID := strings.TrimSpace(actor.ID)
	entries := inputs.entries
	for uid, name := range game.HouseNames {
		entry := entries[uid]
		entry.HouseName = name
		entries = withEntry(entries, uid, entry)
	}
	plan, err := store.PlanStart(store.StartInput{
		GameID:       id,
		FixedSeed:    game.FixedSeed,
		Incognito:    game.Incognito,
		YearCount:    game.YearCount,
		Slots:        slots,
		Entries:      entries,
		DisplayNames: inputs.displayNames,
		Balance:      s.balance,
		Assets:       s.assets,
	})
	if err != nil {
		return lobbyJoinResult{}, err
	}
	started := gameDocumentFromSnapshot(store.GameSnapshot{
		ID:           id,
		Name:         game.Name,
		Seed:         plan.Seed,
		YearCount:    plan.State.YearCount,
		Status:       store.StatusPlaying,
		Phase:        store.PhaseStarted,
		Scores:       engine.ComputeScores(plan.State, s.balance),
		Players:      plan.Slots,
		State:        plan.State,
		Revision:     store.Revision(game.Revision),
		CreatedBy:    game.OwnerUID,
		SpectatorUID: game.SpectatorUID,
		Incognito:    game.Incognito,
	}, game.CreatedAt, game.UpdatedAt)
	stateMap, err := jsonMap(plan.State)
	if err != nil {
		return lobbyJoinResult{}, err
	}
	mapJSON, err := jsonString(plan.Map)
	if err != nil {
		return lobbyJoinResult{}, err
	}
	canonical := canonicalDocument{
		SchemaVersion: schemaVersion,
		ID:            id,
		Seed:          plan.Seed,
		Turn:          plan.State.Turn,
		Revision:      game.Revision,
		State:         stateMap,
		MapJSON:       mapJSON,
		SubmittedUIDs: []string{},
		UpdatedAt:     game.UpdatedAt,
	}
	if err := s.writeStartedProjections(transaction, started, canonical, plan.State); err != nil {
		return lobbyJoinResult{}, err
	}
	for _, slot := range plan.Slots {
		if slot.ActorID == actorID {
			player = slot.ID
		}
	}
	snapshot := gameSnapshot(s.balance, started, plan.State, plan.Map, map[models.PlayerID]engine.OrdersInput{}, []store.ReportRecord{})
	return lobbyJoinResult{snapshot: snapshot, player: player, started: true}, nil
}

// ConfirmLobby is the creator's confirmation that their house name and vote are
// final. In the transaction that records it, the game starts if every slot is
// taken.
func (s *FirestoreStore) ConfirmLobby(ctx context.Context, actor store.Actor, id store.GameID) (store.JoinResult, error) {
	if err := s.requireClient(); err != nil {
		return store.JoinResult{}, err
	}
	actorID := strings.TrimSpace(actor.ID)
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	var result lobbyJoinResult
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		result = lobbyJoinResult{}
		gameSnapshot, err := transaction.Get(gameRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return mapReadError(err, store.ErrUnknownGame)
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		playerID, member := playerIDForActor(game, actor)
		if !member {
			return store.ErrNotMember
		}
		if actorID != strings.TrimSpace(game.OwnerUID) {
			return store.ErrNotCreator
		}
		if game.Phase != store.PhaseLobby {
			return store.ErrGameStarted
		}
		var ranking []int
		if game.FixedSeed == "" {
			document, err := transaction.Get(lobbyRef(s.client, id, actorID))
			s.recordReads(1)
			if ranking, err = decodeSeedRanking(document, err); err != nil {
				return err
			}
		}
		if !game.AwaitingCreator {
			result = lobbyJoinResult{snapshot: s.buildLobbySnapshot(actor, game, ranking), player: playerID}
			return nil
		}
		slots := playerSlots(game.Players)
		var inputs lobbyStartInputs
		final := store.SlotsFilled(slots)
		if final {
			if inputs, err = s.readLobbyStartInputs(transaction, game, slots); err != nil {
				return err
			}
		}
		game.AwaitingCreator = false
		game.Revision++
		game.UpdatedAt = s.now().UTC()
		if final {
			result, err = s.startLobby(transaction, actor, game, slots, inputs, playerID)
			return err
		}
		if err := transaction.Set(gameRef(s.client, id), game); err != nil {
			return err
		}
		s.recordWrites(1)
		result = lobbyJoinResult{snapshot: s.buildLobbySnapshot(actor, game, ranking), player: playerID}
		return nil
	})
	if err != nil {
		return store.JoinResult{}, wrapTransactionResult(err)
	}
	return store.JoinResult{Snapshot: result.snapshot, Player: playerSlot(result.snapshot.Players, result.player), Started: result.started}, nil
}

// withHouseName returns names with uid's house name set, on a copy.
func withHouseName(names map[string]string, uid, name string) map[string]string {
	result := make(map[string]string, len(names)+1)
	for key, value := range names {
		result[key] = value
	}
	result[uid] = name
	return result
}

// readLobbyEntries reads the seed rankings of every member's lobby document,
// by uid.
func (s *FirestoreStore) readLobbyEntries(transaction *cloudfirestore.Transaction, id store.GameID) (map[string]store.LobbyEntry, error) {
	entries := make(map[string]store.LobbyEntry)
	documents := transaction.Documents(lobbyCollection(s.client, id))
	for {
		document, err := documents.Next()
		if errors.Is(err, iterator.Done) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		s.recordReads(1)
		var entry lobbyDocument
		if err := document.DataTo(&entry); err != nil {
			return nil, fmt.Errorf("decode lobby entry %s: %w", document.Ref.Path, err)
		}
		entries[entry.UID] = store.LobbyEntry{SeedRanking: entry.SeedRanking}
	}
}

// loadDisplayNames reads the account display names of the slots' members in one
// batch. An incognito game never records them, so it reads nothing.
func (s *FirestoreStore) loadDisplayNames(transaction *cloudfirestore.Transaction, incognito bool, slots []store.PlayerSlot) (map[string]string, error) {
	if incognito {
		return nil, nil
	}
	refs := make([]*cloudfirestore.DocumentRef, len(slots))
	for index, slot := range slots {
		refs[index] = s.client.Collection("players").Doc(slot.ActorID)
	}
	snapshots, err := transaction.GetAll(refs)
	s.recordReads(len(refs))
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(slots))
	for index, snapshot := range snapshots {
		if !snapshot.Exists() {
			continue
		}
		profile, err := decodeProfileDocument(snapshot)
		if err != nil {
			return nil, err
		}
		names[slots[index].ActorID] = profile.DisplayName
	}
	return names, nil
}

// writeSeedRanking records a member's seed ranking in their lobby document; an
// empty ranking withdraws the vote.
func (s *FirestoreStore) writeSeedRanking(transaction *cloudfirestore.Transaction, id store.GameID, uid string, ranking []int) error {
	if ranking == nil {
		ranking = []int{}
	}
	s.recordWrites(1)
	return transaction.Set(lobbyRef(s.client, id, uid), map[string]any{
		"schemaVersion": schemaVersion,
		"uid":           uid,
		"seedRanking":   ranking,
	}, cloudfirestore.MergeAll)
}

// writeStartedProjections writes, in the transaction, everything that follows
// a change of a started game: its summary, its canonical state, every member's
// view and the observer's. The caller has already updated the documents.
func (s *FirestoreStore) writeStartedProjections(transaction *cloudfirestore.Transaction, game gameDocument, canonical canonicalDocument, state *models.GameState) error {
	id := game.ID
	revision := store.Revision(game.Revision)
	views, err := s.viewProjections(id, game.Players, revision, state, game.UpdatedAt)
	if err != nil {
		return err
	}
	var observer *observerDocument
	if game.SpectatorUID != "" {
		document, err := s.observerDocument(id, game.SpectatorUID, revision, state, game.UpdatedAt, max(0, game.Turn-1))
		if err != nil {
			return err
		}
		observer = &document
	}
	if err := transaction.Set(gameRef(s.client, id), game); err != nil {
		return err
	}
	if err := transaction.Set(canonicalRef(s.client, id), canonical); err != nil {
		return err
	}
	for _, projection := range views {
		if err := transaction.Set(viewRef(s.client, id, projection.actorID), projection.document); err != nil {
			return err
		}
	}
	observerWrites := 0
	if observer != nil {
		if err := transaction.Set(observerRef(s.client, id, game.SpectatorUID), *observer); err != nil {
			return err
		}
		observerWrites = 1
	}
	s.recordWrites(2 + len(views) + observerWrites)
	s.recordProjectionWrites(len(views) + observerWrites)
	return nil
}

// RenameHouse names the actor's house. In the lobby it records the name the
// house takes when the game starts; once started it renames the house, until
// the house submits its first orders.
func (s *FirestoreStore) RenameHouse(ctx context.Context, actor store.Actor, id store.GameID, name string) (store.RenameResult, error) {
	if err := s.requireClient(); err != nil {
		return store.RenameResult{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	gameDocumentRef := gameRef(s.client, id)
	canonicalDocumentRef := canonicalRef(s.client, id)
	uid := strings.TrimSpace(actor.ID)
	var playerID models.PlayerID
	var previous string
	var lobbySnapshot *store.GameSnapshot
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		lobbySnapshot = nil
		gameSnapshot, err := transaction.Get(gameDocumentRef)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrUnknownGame
		}
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		member, ok := playerIDForActor(game, actor)
		if !ok {
			return store.ErrNotMember
		}
		playerID = member
		if game.Phase == store.PhaseLobby {
			// In the lobby the name is only recorded, for the start.
			pending, err := store.ValidatePendingHouseName(name, store.OtherHouseNames(game.HouseNames, uid))
			if err != nil {
				return err
			}
			var ranking []int
			if game.FixedSeed == "" {
				document, err := transaction.Get(lobbyRef(s.client, id, uid))
				s.recordReads(1)
				if ranking, err = decodeSeedRanking(document, err); err != nil {
					return err
				}
			}
			previous = pending
			game.HouseNames = withHouseName(game.HouseNames, uid, pending)
			game.UpdatedAt = s.now().UTC()
			if err := transaction.Set(gameDocumentRef, game); err != nil {
				return err
			}
			s.recordWrites(1)
			snapshot := s.buildLobbySnapshot(actor, game, ranking)
			lobbySnapshot = &snapshot
			return nil
		}
		normalized, err := store.ValidateHouseRename(playerSlots(game.Players), member, name)
		if err != nil {
			return err
		}
		for index := range game.Players {
			if game.Players[index].ID == member {
				previous = game.Players[index].Name
			}
		}
		if normalized == previous {
			return nil
		}
		canonicalSnapshot, err := transaction.Get(canonicalDocumentRef)
		s.recordReads(1)
		if err != nil {
			return mapReadError(err, ErrInconsistentGame)
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		if canonical.Revision != game.Revision || canonical.Turn != game.Turn {
			return store.ErrRevisionConflict
		}
		if canonical.Resolution != nil && s.now().Before(canonical.Resolution.LeaseUntil) {
			return store.ErrRevisionConflict
		}
		state := &models.GameState{}
		if err := decodeJSONMap(canonical.State, state); err != nil {
			return err
		}
		for index := range game.Players {
			if game.Players[index].ID == member {
				game.Players[index].Name = normalized
			}
		}
		for index := range state.Players {
			if state.Players[index].ID == member {
				state.Players[index].Name = normalized
			}
		}
		if err := state.Validate(); err != nil {
			return err
		}
		game.Revision++
		game.UpdatedAt = s.now().UTC()
		canonical.Revision = game.Revision
		canonical.UpdatedAt = game.UpdatedAt
		if canonical.State, err = jsonMap(state); err != nil {
			return err
		}
		// The views are refreshed too, so connected players see the new name.
		return s.writeStartedProjections(transaction, game, canonical, state)
	})
	if err != nil {
		return store.RenameResult{}, wrapTransactionResult(err)
	}
	if lobbySnapshot != nil {
		return store.RenameResult{Snapshot: *lobbySnapshot, Player: playerSlot(lobbySnapshot.Players, playerID), PreviousName: previous}, nil
	}
	snapshot, err := s.Get(operationContext, actor, id)
	if err != nil {
		return store.RenameResult{}, err
	}
	return store.RenameResult{Snapshot: snapshot, Player: playerSlot(snapshot.Players, playerID), PreviousName: previous}, nil
}

func (s *FirestoreStore) Submit(ctx context.Context, actor store.Actor, id store.GameID, request store.SubmitRequest) (store.SubmitResult, error) {
	if err := s.requireClient(); err != nil {
		return store.SubmitResult{}, err
	}
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return store.SubmitResult{}, err
	}
	playerID, ok := snapshot.PlayerFor(actor)
	if !ok {
		return store.SubmitResult{}, store.ErrNotMember
	}
	if !snapshot.Started() {
		return store.SubmitResult{}, store.ErrGameNotStarted
	}
	if snapshot.Status == store.StatusFinished {
		return store.SubmitResult{}, store.ErrGameFinished
	}
	if !engine.PlayerAlive(snapshot.State, playerID) {
		return store.SubmitResult{}, store.ErrEliminated
	}
	if request.ExpectedRevision != 0 && request.ExpectedRevision != snapshot.Revision {
		return store.SubmitResult{}, store.ErrRevisionConflict
	}
	input, err := turn.NormalizeSubmission(playerID, engine.OrdersInput{
		Chains:  request.Chains,
		Winter:  request.Winter,
		Special: request.Special,
	})
	if err != nil {
		return store.SubmitResult{}, err
	}
	draft := request.Status == store.OrdersDraft
	// Validate before writing; a draft is stored as it is. The transaction
	// below still checks the revision and turn, so this read cannot make an
	// obsolete mutation succeed.
	if !draft {
		if err := turn.ValidateSubmission(snapshot.State, s.balance, input); err != nil {
			return store.SubmitResult{}, err
		}
	}
	write, err := s.writeSubmissionForGame(ctx, actor, id, input, request.ExpectedRevision, draft)
	if err != nil {
		return store.SubmitResult{}, err
	}
	if draft {
		current, err := s.Get(ctx, actor, id)
		if err != nil {
			return store.SubmitResult{}, err
		}
		return store.SubmitResult{
			Status:    string(store.OrdersDraft),
			Player:    playerID,
			Submitted: write.Submitted,
			Remaining: write.Remaining,
			Snapshot:  current,
		}, nil
	}
	if request.Force || len(write.Remaining) == 0 {
		result, resolveErr := s.resolveInternal(ctx, actor, id, write.Revision, request.Force)
		if resolveErr == nil {
			return result, nil
		}
		// A simultaneous final submission may have won the claim. It is safe to
		// return the already committed state rather than report a false failure.
		if errors.Is(resolveErr, store.ErrRevisionConflict) && !request.Force {
			current, currentErr := s.Get(ctx, actor, id)
			if currentErr == nil && current.State.Turn > snapshot.State.Turn {
				return resolvedResultFromSnapshot(current, playerID), nil
			}
		}
		return store.SubmitResult{}, resolveErr
	}
	current, err := s.Get(ctx, actor, id)
	if err != nil {
		return store.SubmitResult{}, err
	}
	return store.SubmitResult{
		Status:    "pending",
		Player:    playerID,
		Submitted: write.Submitted,
		Remaining: write.Remaining,
		Snapshot:  current,
	}, nil
}

func (s *FirestoreStore) Resolve(ctx context.Context, actor store.Actor, id store.GameID) (store.SubmitResult, error) {
	return s.resolveInternal(ctx, actor, id, 0, true)
}

func (s *FirestoreStore) ResolveAt(ctx context.Context, actor store.Actor, id store.GameID, expected store.Revision) (store.SubmitResult, error) {
	return s.resolveInternal(ctx, actor, id, expected, true)
}

func (s *FirestoreStore) resolveInternal(ctx context.Context, actor store.Actor, id store.GameID, expected store.Revision, forced bool) (store.SubmitResult, error) {
	if err := s.requireClient(); err != nil {
		return store.SubmitResult{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	claim, err := s.claimResolution(operationContext, actor, id, expected, forced)
	if err != nil {
		return store.SubmitResult{}, err
	}
	snapshot, err := s.loadSnapshot(operationContext, actor, id, true)
	if err != nil {
		s.releaseFailedResolution(ctx, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	if snapshot.Revision != store.Revision(claim.Claim.BaseRevision) {
		s.releaseFailedResolution(ctx, id, claim.Claim)
		return store.SubmitResult{}, store.ErrRevisionConflict
	}
	// The resolution seals the drafts that pass validation.
	sealed := snapshot.Submissions
	if len(snapshot.Drafted) > 0 {
		_, drafts, readErr := s.readSubmissions(operationContext, id, snapshot.State.Turn)
		if readErr != nil {
			s.releaseFailedResolution(ctx, id, claim.Claim)
			return store.SubmitResult{}, readErr
		}
		sealed = turn.SealDrafts(snapshot.State, s.balance, snapshot.Submissions, drafts)
	}
	submitted, remaining := turn.Progress(snapshot.State, func(playerID models.PlayerID) bool {
		_, ok := sealed[playerID]
		return ok
	})
	resolution, err := turn.Resolve(snapshot.State, s.balance, sealed, s.privacyTracker)
	if err != nil {
		s.releaseFailedResolution(ctx, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	report := resolution.Report
	if err := s.commitResolution(operationContext, claim, snapshot, report, sealed); err != nil {
		// Without a release the game would stay locked, every order and
		// resolution answering a revision conflict, until the lease expires.
		s.releaseFailedResolution(ctx, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	updated, err := s.Get(operationContext, actor, id)
	if err != nil {
		return store.SubmitResult{}, err
	}
	record := store.ReportRecord{Report: report, Privacy: report.State.Privacy}
	return store.SubmitResult{
		Status:    "resolved",
		Player:    claim.PlayerID,
		Submitted: submitted,
		Remaining: remaining,
		Resolved:  true,
		Forced:    forced,
		Report:    &record,
		Snapshot:  updated,
	}, nil
}

func (s *FirestoreStore) claimResolution(ctx context.Context, actor store.Actor, id store.GameID, expected store.Revision, forced bool) (resolutionClaimResult, error) {
	operationID, err := newOperationID()
	if err != nil {
		return resolutionClaimResult{}, err
	}
	var result resolutionClaimResult
	s.recordTransaction()
	err = s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		gameSnapshot, err := transaction.Get(gameRef(s.client, id))
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrUnknownGame
		}
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		if forced && game.OwnerUID != strings.TrimSpace(actor.ID) {
			return store.ErrNotCreator
		}
		playerID, member := playerIDForActor(game, actor)
		if !member && !membershipForActor(game, actor) {
			return store.ErrNotMember
		}
		if game.Phase == store.PhaseLobby {
			return store.ErrGameNotStarted
		}
		if game.Status == store.StatusFinished {
			return store.ErrGameFinished
		}
		canonicalSnapshot, err := transaction.Get(canonicalRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return mapReadError(err, ErrInconsistentGame)
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		if canonical.Revision != game.Revision || canonical.Turn != game.Turn {
			return store.ErrRevisionConflict
		}
		if expected != 0 && int64(expected) != canonical.Revision {
			return store.ErrRevisionConflict
		}
		if canonical.Resolution != nil && s.now().Before(canonical.Resolution.LeaseUntil) {
			return store.ErrRevisionConflict
		}
		now := s.now().UTC()
		claim := resolutionClaim{
			OperationID:  operationID,
			ClaimedAt:    now,
			LeaseUntil:   now.Add(s.leaseTimeout),
			BaseRevision: canonical.Revision,
			Turn:         canonical.Turn,
		}
		canonical.Resolution = &claim
		canonical.UpdatedAt = now
		if err := transaction.Set(canonicalRef(s.client, id), canonical); err != nil {
			return err
		}
		s.recordWrites(1)
		result = resolutionClaimResult{Claim: claim, Game: game, PlayerID: playerID}
		return nil
	})
	if err != nil {
		return resolutionClaimResult{}, wrapTransactionResult(err)
	}
	return result, nil
}

func (s *FirestoreStore) commitResolution(ctx context.Context, claim resolutionClaimResult, snapshot store.GameSnapshot, report engine.TurnReport, sealed map[models.PlayerID]engine.OrdersInput) error {
	gameRefValue := gameRef(s.client, snapshot.ID)
	canonicalRefValue := canonicalRef(s.client, snapshot.ID)
	s.recordTransaction()
	return wrapTransactionResult(s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		if s.commitFault != nil {
			if err := s.commitFault(); err != nil {
				return err
			}
		}
		gameSnapshot, err := transaction.Get(gameRefValue)
		s.recordReads(1)
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		canonicalSnapshot, err := transaction.Get(canonicalRefValue)
		s.recordReads(1)
		if err != nil {
			return err
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		if canonical.Resolution == nil || canonical.Resolution.OperationID != claim.Claim.OperationID || canonical.Resolution.BaseRevision != claim.Claim.BaseRevision || canonical.Revision != claim.Claim.BaseRevision || canonical.Turn != claim.Claim.Turn {
			return store.ErrRevisionConflict
		}
		// Read all current submissions before issuing any writes. The claim
		// prevents another submission transaction from changing this set.
		submissionDocuments := transaction.Documents(submissionCollection(s.client, snapshot.ID, canonical.Turn))
		refs := make([]*cloudfirestore.DocumentRef, 0)
		for {
			document, err := submissionDocuments.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			refs = append(refs, document.Ref)
		}
		s.recordReads(len(refs))
		stateMap, err := jsonMap(report.State)
		if err != nil {
			return err
		}
		reportMap, err := jsonMap(report)
		if err != nil {
			return err
		}
		privacyMap, err := jsonMap(report.State.Privacy)
		if err != nil {
			return err
		}
		updatedAt := s.now().UTC()
		canonical.State = stateMap
		canonical.Turn = report.State.Turn
		canonical.Revision++
		canonical.SubmittedUIDs = []string{}
		canonical.Resolution = nil
		canonical.UpdatedAt = updatedAt
		game.Turn = report.State.Turn
		game.Season = report.State.Season
		game.YearCount = report.State.YearCount
		game.Scores = scoreDocuments(engine.ComputeScores(report.State, s.balance))
		game.Revision = canonical.Revision
		game.SubmittedUIDs = []string{}
		game.DraftUIDs = nil
		// Sealing, by the house or by the host's resolution, locks the house name.
		for index := range game.Players {
			if _, ok := sealed[models.PlayerID(game.Players[index].ID)]; ok {
				game.Players[index].NameLocked = true
			}
		}
		game.RequiredUIDs = sortedRequiredUIDs(report.State, playerSlots(game.Players))
		game.UpdatedAt = updatedAt
		game.Status, game.WinnerUID = statusForState(report.State, s.balance)
		rawReport := reportDocument{
			SchemaVersion: schemaVersion,
			GameID:        snapshot.ID,
			Turn:          report.Header.Turn,
			Report:        reportMap,
			Privacy:       privacyMap,
			CreatedAt:     updatedAt,
		}
		views, err := s.viewProjections(
			snapshot.ID,
			game.Players,
			store.Revision(canonical.Revision),
			report.State,
			updatedAt,
		)
		if err != nil {
			return err
		}
		if err := transaction.Set(canonicalRefValue, canonical); err != nil {
			return err
		}
		if err := transaction.Set(gameRefValue, game); err != nil {
			return err
		}
		if err := transaction.Set(reportCollection(s.client, snapshot.ID).Doc(strconv.Itoa(rawReport.Turn)), rawReport); err != nil {
			return err
		}
		observerWrites := 0
		if game.SpectatorUID != "" {
			observer, observerErr := s.observerDocument(
				snapshot.ID,
				game.SpectatorUID,
				store.Revision(canonical.Revision),
				report.State,
				updatedAt,
				rawReport.Turn,
			)
			if observerErr != nil {
				return observerErr
			}
			if err := transaction.Set(observerRef(s.client, snapshot.ID, game.SpectatorUID), observer); err != nil {
				return err
			}
			observerWrites = 1
		}
		oldReportDeletes := 0
		for _, projection := range views {
			if err := transaction.Set(viewRef(s.client, snapshot.ID, projection.actorID), projection.document); err != nil {
				return err
			}
			filtered := api.ProjectReport(report, projection.playerID, report.State.Privacy)
			filteredMap, err := jsonMap(filtered)
			if err != nil {
				return err
			}
			filteredDocument := filteredReportDocument{
				SchemaVersion: schemaVersion,
				GameID:        snapshot.ID,
				UID:           projection.actorID,
				Revision:      canonical.Revision,
				Turn:          report.Header.Turn,
				Season:        report.Header.Season,
				Report:        filteredMap,
				UpdatedAt:     updatedAt,
			}
			if err := transaction.Set(filteredReportRef(s.client, snapshot.ID, projection.actorID, rawReport.Turn), filteredDocument); err != nil {
				return err
			}
		}
		for _, ref := range refs {
			if err := transaction.Delete(ref); err != nil {
				return err
			}
		}
		if oldTurn := rawReport.Turn - s.maximumReports; oldTurn > 0 {
			oldReportDeletes = 1
			if err := transaction.Delete(reportCollection(s.client, snapshot.ID).Doc(strconv.Itoa(oldTurn))); err != nil {
				return err
			}
			for _, player := range game.Players {
				if store.IsAssignedActor(player.ActorID) {
					oldReportDeletes++
					if err := transaction.Delete(filteredReportRef(s.client, snapshot.ID, player.ActorID, oldTurn)); err != nil {
						return err
					}
				}
			}
		}
		s.recordWrites(3 + len(views)*2 + observerWrites + len(refs) + oldReportDeletes)
		s.recordProjectionWrites(len(views)*2 + observerWrites)
		return nil
	}))
}

// releaseFailedResolution clears the claim of a resolution that did not
// commit. It runs on a fresh deadline because the failure may be the
// operation's own deadline. It only clears this operation's claim, so it is
// a no-op when an ambiguous commit did land or another operation took over;
// if it fails too, the lease still expires.
func (s *FirestoreStore) releaseFailedResolution(ctx context.Context, id store.GameID, claim resolutionClaim) {
	releaseContext, cancel := s.operationContext(context.WithoutCancel(ctx))
	defer cancel()
	_ = s.releaseResolution(releaseContext, id, claim)
}

func (s *FirestoreStore) releaseResolution(ctx context.Context, id store.GameID, claim resolutionClaim) error {
	s.recordTransaction()
	return wrapTransactionResult(s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(canonicalRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return err
		}
		canonical, err := decodeCanonicalDocument(snapshot)
		if err != nil {
			return err
		}
		if canonical.Resolution == nil || canonical.Resolution.OperationID != claim.OperationID {
			return nil
		}
		canonical.Resolution = nil
		canonical.UpdatedAt = s.now().UTC()
		s.recordWrites(1)
		return transaction.Set(canonicalRef(s.client, id), canonical)
	}))
}

func (s *FirestoreStore) Restore(ctx context.Context) ([]store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	documents := s.client.Collection("games").Documents(operationContext)
	result := make([]store.GameSnapshot, 0)
	for {
		document, err := documents.Next()
		if err == iterator.Done {
			return result, nil
		}
		if err != nil {
			return nil, wrapOperation("restore games", err)
		}
		s.recordReads(1)
		game, err := decodeGameDocument(document)
		if err != nil {
			return nil, err
		}
		if game.Phase == store.PhaseLobby {
			snapshot, err := s.lobbySnapshot(operationContext, store.Actor{}, game)
			if err != nil {
				return nil, fmt.Errorf("restore %s: %w", game.ID, err)
			}
			result = append(result, snapshot)
			continue
		}
		canonicalSnapshot, err := canonicalRef(s.client, game.ID).Get(operationContext)
		s.recordReads(1)
		if err != nil {
			return nil, fmt.Errorf("restore %s canonical: %w", game.ID, mapReadError(err, ErrInconsistentGame))
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return nil, fmt.Errorf("restore %s canonical: %w", game.ID, err)
		}
		if canonical.Resolution != nil && !s.now().Before(canonical.Resolution.LeaseUntil) {
			// The previous instance may have crashed after claiming the turn.
			// Re-enter through the same claim/commit path so the engine is never
			// applied twice and the lease remains a recoverable internal detail.
			claim := *canonical.Resolution
			if _, recoveryErr := s.resolveInternal(operationContext, store.Actor{ID: game.OwnerUID}, game.ID, store.Revision(canonical.Revision), false); recoveryErr != nil {
				// A missing owner or a concurrent recovery must not prevent other
				// games from starting. Release only our known claim; a concurrent
				// operation has a different operationId and is left untouched.
				if releaseErr := s.releaseResolution(operationContext, game.ID, claim); releaseErr != nil {
					return nil, fmt.Errorf("restore %s resolution claim: %w", game.ID, releaseErr)
				}
			}
			gameDocumentSnapshot, refreshErr := gameRef(s.client, game.ID).Get(operationContext)
			s.recordReads(1)
			if refreshErr != nil {
				return nil, fmt.Errorf("restore %s after resolution: %w", game.ID, refreshErr)
			}
			game, err = decodeGameDocument(gameDocumentSnapshot)
			if err != nil {
				return nil, err
			}
		}
		snapshot, err := s.loadSnapshotWithDocument(operationContext, store.Actor{}, game, false)
		if err != nil {
			return nil, fmt.Errorf("restore %s: %w", game.ID, err)
		}
		result = append(result, snapshot)
	}
}

// passthroughErrors are the store errors a transaction returns as they are.
var passthroughErrors = []error{
	store.ErrRevisionConflict, store.ErrUnknownGame, store.ErrNotMember, store.ErrGameFull,
	store.ErrInvalidInvitation, store.ErrInvitationInactive, store.ErrSpectator, store.ErrGameFinished,
	store.ErrProfileNotFound, store.ErrGameNotStarted, store.ErrGameStarted, store.ErrInvalidSeedRanking,
	store.ErrInvalidCandidate, store.ErrNameLocked, store.ErrHouseNameTaken, store.ErrInvalidDisplayName,
}

func wrapTransactionResult(err error) error {
	if err == nil {
		return nil
	}
	for _, sentinel := range passthroughErrors {
		if errors.Is(err, sentinel) {
			return err
		}
	}
	if status.Code(err) == codes.Aborted || status.Code(err) == codes.FailedPrecondition {
		return store.ErrRevisionConflict
	}
	return err
}

func statusForState(state *models.GameState, balance assetgen.Balance) (store.Status, string) {
	finished, winner := turn.Outcome(state, balance)
	switch {
	case !finished:
		return store.StatusPlaying, ""
	case winner == nil:
		return store.StatusFinished, ""
	default:
		return store.StatusFinished, string(*winner)
	}
}

func playerSlot(players []store.PlayerSlot, id models.PlayerID) store.PlayerSlot {
	for _, player := range players {
		if player.ID == id {
			return player
		}
	}
	return store.PlayerSlot{ID: id}
}

func resolvedResultFromSnapshot(snapshot store.GameSnapshot, playerID models.PlayerID) store.SubmitResult {
	var report *store.ReportRecord
	if len(snapshot.Reports) > 0 {
		value := snapshot.Reports[len(snapshot.Reports)-1]
		report = &value
	}
	return store.SubmitResult{
		Status:   "resolved",
		Player:   playerID,
		Resolved: true,
		Report:   report,
		Snapshot: snapshot,
	}
}

func (s *FirestoreStore) writeSubmissionForGame(ctx context.Context, actor store.Actor, id store.GameID, input engine.OrdersInput, expected store.Revision, draft bool) (submissionWriteResult, error) {
	var result submissionWriteResult
	s.recordTransaction()
	err := s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		gameSnapshot, err := transaction.Get(gameRef(s.client, id))
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrUnknownGame
		}
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		playerID, member := playerIDForActor(game, actor)
		if !member {
			return store.ErrNotMember
		}
		if game.Phase == store.PhaseLobby {
			return store.ErrGameNotStarted
		}
		if game.Status == store.StatusFinished {
			return store.ErrGameFinished
		}
		canonicalSnapshot, err := transaction.Get(canonicalRef(s.client, id))
		s.recordReads(1)
		if err != nil {
			return mapReadError(err, ErrInconsistentGame)
		}
		canonical, err := decodeCanonicalDocument(canonicalSnapshot)
		if err != nil {
			return err
		}
		if canonical.Revision != game.Revision || canonical.Turn != game.Turn {
			return store.ErrRevisionConflict
		}
		if expected != 0 && int64(expected) != canonical.Revision {
			return store.ErrRevisionConflict
		}
		if canonical.Resolution != nil && s.now().Before(canonical.Resolution.LeaseUntil) {
			return store.ErrRevisionConflict
		}
		state := &models.GameState{}
		if err := decodeJSONMap(canonical.State, state); err != nil {
			return err
		}
		if !engine.PlayerAlive(state, playerID) {
			return store.ErrEliminated
		}
		orders, err := ordersJSON(input)
		if err != nil {
			return err
		}
		wasSealed := slices.Contains(canonical.SubmittedUIDs, actor.ID)
		now := s.now().UTC()
		submission := submissionDocument{
			SchemaVersion: schemaVersion,
			UID:           actor.ID,
			PlayerID:      string(playerID),
			Turn:          game.Turn,
			OrdersJSON:    orders,
			SubmittedAt:   now,
		}
		if draft {
			submission.Status = store.OrdersDraft
		}
		if err := transaction.Set(submissionCollection(s.client, id, game.Turn).Doc(actor.ID), submission); err != nil {
			return err
		}
		if draft {
			// A draft is not a sealing: saving one over sealed orders breaks
			// the seal.
			canonical.SubmittedUIDs = slices.DeleteFunc(canonical.SubmittedUIDs, func(uid string) bool { return uid == actor.ID })
			game.SubmittedUIDs = slices.DeleteFunc(game.SubmittedUIDs, func(uid string) bool { return uid == actor.ID })
			game.DraftUIDs = appendUnique(game.DraftUIDs, actor.ID)
			sort.Strings(game.DraftUIDs)
			if wasSealed && turn.NameUnlockedOnSealBreak(game.Turn) {
				// Breaking the first seal of the game gives the house name back.
				for index := range game.Players {
					if game.Players[index].ID == playerID {
						game.Players[index].NameLocked = false
					}
				}
			}
		} else {
			canonical.SubmittedUIDs = appendUnique(canonical.SubmittedUIDs, actor.ID)
			sort.Strings(canonical.SubmittedUIDs)
			game.SubmittedUIDs = appendUnique(game.SubmittedUIDs, actor.ID)
			sort.Strings(game.SubmittedUIDs)
			game.DraftUIDs = slices.DeleteFunc(game.DraftUIDs, func(uid string) bool { return uid == actor.ID })
			for index := range game.Players {
				if game.Players[index].ID == playerID {
					game.Players[index].NameLocked = true
				}
			}
		}
		canonical.Revision++
		canonical.UpdatedAt = now
		game.Revision = canonical.Revision
		game.UpdatedAt = now
		if err := transaction.Set(canonicalRef(s.client, id), canonical); err != nil {
			return err
		}
		if err := transaction.Set(gameRef(s.client, id), game); err != nil {
			return err
		}
		for _, player := range game.Players {
			if !store.IsAssignedActor(player.ActorID) {
				continue
			}
			if err := transaction.Update(viewRef(s.client, id, player.ActorID), []cloudfirestore.Update{
				{Path: "revision", Value: canonical.Revision},
				{Path: "updatedAt", Value: now},
			}); err != nil {
				return err
			}
		}
		s.recordWrites(3 + assignedDocumentPlayerCount(game.Players))
		s.recordProjectionWrites(assignedDocumentPlayerCount(game.Players))
		submitted, remaining := progressFromUIDs(game, state, canonical.SubmittedUIDs)
		result = submissionWriteResult{
			Revision:  store.Revision(canonical.Revision),
			PlayerID:  playerID,
			Submitted: submitted,
			Remaining: remaining,
		}
		return nil
	})
	return result, wrapTransactionResult(err)
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// progressFromUIDs reports submission progress from the submitted member UIDs
// stored on the canonical document.
func progressFromUIDs(game gameDocument, state *models.GameState, uids []string) (submitted, remaining []models.PlayerID) {
	actors := make(map[models.PlayerID]string, len(game.Players))
	for _, player := range game.Players {
		actors[player.ID] = player.ActorID
	}
	return turn.Progress(state, func(playerID models.PlayerID) bool {
		return slices.Contains(uids, actors[playerID])
	})
}

// SetSeedVote records the actor's ranking of the candidate seeds while the
// lobby is open; an empty ranking withdraws the vote. The vote lives in the
// backend-only lobby document, never on the member-readable game document.
func (s *FirestoreStore) SetSeedVote(ctx context.Context, actor store.Actor, id store.GameID, ranking []int) (store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return store.GameSnapshot{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	uid := strings.TrimSpace(actor.ID)
	var snapshot store.GameSnapshot
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		gameSnapshot, err := transaction.Get(gameRef(s.client, id))
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrUnknownGame
		}
		if err != nil {
			return err
		}
		game, err := decodeGameDocument(gameSnapshot)
		if err != nil {
			return err
		}
		if _, member := playerIDForActor(game, actor); !member {
			return store.ErrNotMember
		}
		if game.Phase != store.PhaseLobby {
			return store.ErrGameStarted
		}
		if err := store.ValidateSeedRanking(ranking, s.balance.SeedCandidateCount()); err != nil {
			return err
		}
		snapshot = s.buildLobbySnapshot(actor, game, ranking)
		return s.writeSeedRanking(transaction, id, uid, ranking)
	})
	if err != nil {
		return store.GameSnapshot{}, wrapTransactionResult(err)
	}
	return snapshot, nil
}

// JoinPreview describes a game to a member or to the holder of its invitation
// code: the authorization of the lobby reads.
func (s *FirestoreStore) JoinPreview(ctx context.Context, actor store.Actor, id store.GameID, code string) (store.JoinPreview, error) {
	if err := s.requireClient(); err != nil {
		return store.JoinPreview{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	document, err := gameRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.JoinPreview{}, mapReadError(err, store.ErrUnknownGame)
	}
	game, err := decodeGameDocument(document)
	if err != nil {
		return store.JoinPreview{}, err
	}
	member := membershipForActor(game, actor)
	if !member {
		invitation, err := s.LookupInvitation(operationContext, code)
		if err != nil {
			return store.JoinPreview{}, err
		}
		if invitation.GameID != id {
			return store.JoinPreview{}, store.ErrInvalidInvitation
		}
	}
	preview := store.JoinPreview{
		ID:        id,
		Name:      game.Name,
		Players:   len(game.Players),
		Seated:    assignedDocumentPlayerCount(game.Players),
		Incognito: game.Incognito,
		Started:   game.Phase != store.PhaseLobby,
		Member:    member,
	}
	if game.Phase == store.PhaseLobby && game.FixedSeed == "" {
		preview.SeedCandidates = store.SeedCandidates(id, s.balance.SeedCandidateCount(), s.assets)
	}
	return preview, nil
}

// SeedCandidateMap generates the map of candidate seed index for a game of the
// given player count.
func (s *FirestoreStore) SeedCandidateMap(id store.GameID, players, index int) (store.CandidateMap, error) {
	return store.GenerateCandidateMap(id, index, players, s.balance, s.assets)
}
