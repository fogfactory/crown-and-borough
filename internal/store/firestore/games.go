package firestorestore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func (s *FirestoreStore) Create(ctx context.Context, actor store.Actor, request store.CreateRequest) (store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return store.GameSnapshot{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()

	// Reuse the established game initialization semantics. This temporary
	// memory store never becomes the source of truth; it only constructs the
	// deterministic initial state and slot assignments before the batch write.
	initializer := store.NewMemoryStoreWithOptions(s.balance, s.assets, store.MemoryStoreOptions{
		PrivacyTracker:   s.privacyTracker,
		StrictMembership: s.strictMembership,
	})
	snapshot, err := initializer.Create(operationContext, actor, request)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	createdAt := s.now().UTC()
	if err := s.writeInitialGame(operationContext, snapshot, createdAt); err != nil {
		return store.GameSnapshot{}, wrapOperation("create game", err)
	}
	return snapshot, nil
}

// CreateWithInvitation is used by the HTTP handler when the adapter supports
// invitations. The game is removed if invitation creation fails, avoiding a
// game that its creator cannot share.
func (s *FirestoreStore) CreateWithInvitation(ctx context.Context, actor store.Actor, request store.CreateRequest) (store.GameCreation, error) {
	snapshot, err := s.Create(ctx, actor, request)
	if err != nil {
		return store.GameCreation{}, err
	}
	invitation, err := s.CreateInvitation(ctx, actor, snapshot.ID)
	if err != nil {
		if cleanupErr := s.deleteGame(ctx, snapshot.ID); cleanupErr != nil {
			return store.GameCreation{}, errors.Join(err, fmt.Errorf("cleanup game after invitation failure: %w", cleanupErr))
		}
		return store.GameCreation{}, err
	}
	return store.GameCreation{Snapshot: snapshot, Invitation: invitation}, nil
}

func (s *FirestoreStore) writeInitialGame(ctx context.Context, snapshot store.GameSnapshot, createdAt time.Time) error {
	state, err := jsonMap(snapshot.State)
	if err != nil {
		return err
	}
	mapData, err := jsonString(snapshot.Map)
	if err != nil {
		return err
	}
	game := gameDocumentFromSnapshot(snapshot, createdAt, createdAt)
	canonical := canonicalDocument{
		SchemaVersion: schemaVersion,
		ID:            snapshot.ID,
		Seed:          snapshot.Seed,
		Turn:          snapshot.State.Turn,
		Revision:      int64(snapshot.Revision),
		State:         state,
		MapJSON:       mapData,
		SubmittedUIDs: []string{},
		UpdatedAt:     createdAt,
	}
	batch := s.client.Batch()
	batch.Set(gameRef(s.client, snapshot.ID), game)
	batch.Set(canonicalRef(s.client, snapshot.ID), canonical)
	for _, player := range snapshot.Players {
		if !isAssignedActor(player.ActorID) {
			continue
		}
		view, err := s.viewDocument(snapshot.ID, player.ActorID, player.ID, snapshot.Revision, snapshot.State, createdAt)
		if err != nil {
			return err
		}
		batch.Set(viewRef(s.client, snapshot.ID, player.ActorID), view)
	}
	_, err = batch.Commit(ctx)
	if err != nil {
		return err
	}
	assigned := assignedPlayerCount(snapshot.Players)
	s.recordWrites(2 + assigned)
	s.recordProjectionWrites(assigned)
	return nil
}

func (s *FirestoreStore) Get(ctx context.Context, actor store.Actor, id store.GameID) (store.GameSnapshot, error) {
	return s.loadSnapshot(ctx, actor, id, true)
}

func (s *FirestoreStore) State(ctx context.Context, actor store.Actor, id store.GameID) (store.GameSnapshot, error) {
	return s.Get(ctx, actor, id)
}

func (s *FirestoreStore) Map(ctx context.Context, actor store.Actor, id store.GameID) (mapgen.MapData, error) {
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return mapgen.MapData{}, err
	}
	return snapshot.Map, nil
}

func (s *FirestoreStore) Supply(ctx context.Context, actor store.Actor, id store.GameID, territoryID models.TerritoryID) (engine.SupplyLine, error) {
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return engine.SupplyLine{}, err
	}
	return engine.FindSupply(snapshot.State, s.balance, territoryID)
}

func (s *FirestoreStore) List(ctx context.Context, actor store.Actor) ([]store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		return []store.GameSnapshot{}, nil
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	query := s.client.Collection("games").Where("memberUids", "array-contains", actorID).OrderBy("updatedAt", cloudfirestore.Desc)
	documents := query.Documents(operationContext)
	result := make([]store.GameSnapshot, 0)
	for {
		document, err := documents.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, wrapOperation("list games", err)
		}
		s.recordReads(1)
		game, err := decodeGameDocument(document)
		if err != nil {
			return nil, err
		}
		snapshot, err := s.loadSnapshotWithDocument(operationContext, actor, game, true)
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, nil
}

func (s *FirestoreStore) ListMemberships(ctx context.Context, id store.GameID) ([]store.Membership, error) {
	// This method is a backend-only membership inspection hook. Public handlers
	// use ListActorMemberships, which is constrained to one verified UID.
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	document, err := gameRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return nil, mapReadError(err, store.ErrUnknownGame)
	}
	game, err := decodeGameDocument(document)
	if err != nil {
		return nil, err
	}
	result := make([]store.Membership, 0, len(game.Players))
	for _, player := range game.Players {
		if !isAssignedActor(player.ActorID) {
			continue
		}
		result = append(result, store.Membership{GameID: id, UID: player.ActorID, PlayerID: player.ID})
	}
	return result, nil
}

func (s *FirestoreStore) ListActorMemberships(ctx context.Context, uid string) ([]store.Membership, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return []store.Membership{}, nil
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	documents := s.client.Collection("games").Where("memberUids", "array-contains", uid).Documents(operationContext)
	result := make([]store.Membership, 0)
	for {
		document, err := documents.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, wrapOperation("list memberships", err)
		}
		s.recordReads(1)
		game, err := decodeGameDocument(document)
		if err != nil {
			return nil, err
		}
		for _, player := range game.Players {
			if player.ActorID == uid {
				result = append(result, store.Membership{GameID: game.ID, UID: uid, PlayerID: player.ID})
			}
		}
	}
	slices.SortFunc(result, func(left, right store.Membership) int {
		return strings.Compare(string(left.GameID), string(right.GameID))
	})
	return result, nil
}

func (s *FirestoreStore) Reports(ctx context.Context, actor store.Actor, id store.GameID) ([]store.ReportRecord, error) {
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	return snapshot.Reports, nil
}

func (s *FirestoreStore) Report(ctx context.Context, actor store.Actor, id store.GameID, index int) (store.ReportRecord, error) {
	reports, err := s.Reports(ctx, actor, id)
	if err != nil {
		return store.ReportRecord{}, err
	}
	if index < 0 || index >= len(reports) {
		return store.ReportRecord{}, store.ErrInvalidReport
	}
	return reports[index], nil
}

func (s *FirestoreStore) loadSnapshot(ctx context.Context, actor store.Actor, id store.GameID, requireMember bool) (store.GameSnapshot, error) {
	if err := s.requireClient(); err != nil {
		return store.GameSnapshot{}, err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	document, err := gameRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.GameSnapshot{}, mapReadError(err, store.ErrUnknownGame)
	}
	game, err := decodeGameDocument(document)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	return s.loadSnapshotWithDocument(operationContext, actor, game, requireMember)
}

func (s *FirestoreStore) loadSnapshotWithDocument(ctx context.Context, actor store.Actor, game gameDocument, requireMember bool) (store.GameSnapshot, error) {
	_, member := playerIDForActor(game, actor)
	if requireMember && !member {
		return store.GameSnapshot{}, store.ErrNotMember
	}
	canonicalSnapshot, err := canonicalRef(s.client, game.ID).Get(ctx)
	s.recordReads(1)
	if err != nil {
		return store.GameSnapshot{}, mapReadError(err, ErrInconsistentGame)
	}
	canonical, err := decodeCanonicalDocument(canonicalSnapshot)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	if canonical.ID != game.ID || canonical.Revision != game.Revision || canonical.Turn != game.Turn {
		return store.GameSnapshot{}, fmt.Errorf("%w: summary and canonical revisions differ for %s", ErrInconsistentGame, game.ID)
	}
	state := &models.GameState{}
	if err := decodeJSONMap(canonical.State, state); err != nil {
		return store.GameSnapshot{}, fmt.Errorf("decode state for %s: %w", game.ID, err)
	}
	if game.YearCount == 0 {
		game.YearCount = models.DefaultGameYears
	}
	if state.YearCount == 0 {
		state.YearCount = game.YearCount
	}
	if err := state.Validate(); err != nil {
		return store.GameSnapshot{}, fmt.Errorf("%w: state for %s: %v", ErrInconsistentGame, game.ID, err)
	}
	var mapData mapgen.MapData
	if err := decodeJSONString(canonical.MapJSON, &mapData); err != nil {
		return store.GameSnapshot{}, fmt.Errorf("decode map for %s: %w", game.ID, err)
	}
	submissions, err := s.readSubmissions(ctx, game.ID, game.Turn)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	reports, err := s.readReports(ctx, game.ID)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	return gameSnapshot(game, state, mapData, submissions, reports), nil
}

func (s *FirestoreStore) readSubmissions(ctx context.Context, id store.GameID, turn int) (map[models.PlayerID]engine.OrdersInput, error) {
	result := make(map[models.PlayerID]engine.OrdersInput)
	documents := submissionCollection(s.client, id, turn).Documents(ctx)
	for {
		snapshot, err := documents.Next()
		if err == iterator.Done {
			return result, nil
		}
		if err != nil {
			return nil, wrapOperation("read submissions", err)
		}
		s.recordReads(1)
		document, err := decodeSubmissionDocument(snapshot)
		if err != nil {
			return nil, err
		}
		input, err := decodeOrdersJSON(document.OrdersJSON)
		if err != nil {
			return nil, fmt.Errorf("decode submission %s: %w", snapshot.Ref.Path, err)
		}
		result[models.PlayerID(document.PlayerID)] = input
	}
}

func (s *FirestoreStore) readReports(ctx context.Context, id store.GameID) ([]store.ReportRecord, error) {
	documents := reportCollection(s.client, id).OrderBy("turn", cloudfirestore.Asc).Documents(ctx)
	records := make([]store.ReportRecord, 0)
	for {
		snapshot, err := documents.Next()
		if err == iterator.Done {
			return records, nil
		}
		if err != nil {
			return nil, wrapOperation("read reports", err)
		}
		s.recordReads(1)
		document, err := decodeReportDocument(snapshot)
		if err != nil {
			return nil, err
		}
		var report engine.TurnReport
		if err := decodeJSONMap(document.Report, &report); err != nil {
			return nil, err
		}
		var privacy *models.PrivacyMeta
		if document.Privacy != nil {
			privacy = &models.PrivacyMeta{}
			if err := decodeJSONMap(document.Privacy, privacy); err != nil {
				return nil, err
			}
		}
		records = append(records, store.ReportRecord{Report: report, Privacy: privacy})
	}
}

func (s *FirestoreStore) viewDocument(id store.GameID, uid string, playerID models.PlayerID, revision store.Revision, state *models.GameState, updatedAt time.Time) (viewDocument, error) {
	view := api.ProjectStateForPlayer(state, playerID)
	stateMap, err := jsonMap(view)
	if err != nil {
		return viewDocument{}, err
	}
	return viewDocument{
		SchemaVersion: schemaVersion,
		GameID:        id,
		UID:           uid,
		Revision:      int64(revision),
		Turn:          state.Turn,
		Season:        state.Season,
		State:         stateMap,
		UpdatedAt:     updatedAt,
	}, nil
}

func gameDocumentFromSnapshot(snapshot store.GameSnapshot, createdAt, updatedAt time.Time) gameDocument {
	players := make([]playerDocument, len(snapshot.Players))
	memberUIDs := make([]string, 0, len(snapshot.Players))
	for index, player := range snapshot.Players {
		players[index] = playerDocument{ID: player.ID, Name: player.Name, Color: player.Color, ActorID: player.ActorID}
		if isAssignedActor(player.ActorID) {
			memberUIDs = append(memberUIDs, player.ActorID)
		}
	}
	winner := ""
	if snapshot.Winner != nil {
		winner = string(*snapshot.Winner)
	}
	return gameDocument{
		SchemaVersion: schemaVersion,
		ID:            snapshot.ID,
		Name:          snapshot.Name,
		Seed:          snapshot.Seed,
		OwnerUID:      snapshot.CreatedBy,
		MemberUIDs:    memberUIDs,
		Players:       players,
		Status:        snapshot.Status,
		Turn:          snapshot.State.Turn,
		Season:        snapshot.State.Season,
		YearCount:     snapshot.YearCount,
		Scores:        scoreDocuments(snapshot.Scores),
		WinnerUID:     winner,
		SubmittedUIDs: sortedSubmittedUIDs(snapshot.Submissions, snapshot.Players),
		Revision:      int64(snapshot.Revision),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func gameSnapshot(document gameDocument, state *models.GameState, mapData mapgen.MapData, submissions map[models.PlayerID]engine.OrdersInput, reports []store.ReportRecord) store.GameSnapshot {
	players := make([]store.PlayerSlot, len(document.Players))
	for index, player := range document.Players {
		players[index] = store.PlayerSlot{ID: player.ID, Name: player.Name, Color: player.Color, ActorID: player.ActorID}
	}
	var winner *models.PlayerID
	if document.WinnerUID != "" {
		value := models.PlayerID(document.WinnerUID)
		winner = &value
	}
	scores := make(map[models.PlayerID]engine.ScoreBreakdown, len(document.Scores))
	for playerID, score := range document.Scores {
		scores[models.PlayerID(playerID)] = score
	}
	if len(scores) == 0 {
		scores = engine.ComputeScores(state)
	}
	yearCount := document.YearCount
	if yearCount == 0 {
		yearCount = state.YearCount
	}
	return store.GameSnapshot{
		ID:          document.ID,
		Name:        document.Name,
		Seed:        document.Seed,
		YearCount:   yearCount,
		Status:      document.Status,
		Winner:      winner,
		Scores:      scores,
		Players:     players,
		Map:         mapData,
		State:       state,
		Submissions: submissions,
		Reports:     reports,
		Revision:    store.Revision(document.Revision),
		CreatedBy:   document.OwnerUID,
	}
}

func scoreDocuments(scores map[models.PlayerID]engine.ScoreBreakdown) map[string]engine.ScoreBreakdown {
	result := make(map[string]engine.ScoreBreakdown, len(scores))
	for playerID, score := range scores {
		result[string(playerID)] = score
	}
	return result
}

func sortedSubmittedUIDs(submissions map[models.PlayerID]engine.OrdersInput, players []store.PlayerSlot) []string {
	result := make([]string, 0, len(submissions))
	for _, player := range players {
		if _, ok := submissions[player.ID]; ok && isAssignedActor(player.ActorID) {
			result = append(result, player.ActorID)
		}
	}
	sort.Strings(result)
	return result
}

func isAssignedActor(actorID string) bool {
	return actorID != "" && !strings.HasPrefix(actorID, "slot:")
}

func assignedPlayerCount(players []store.PlayerSlot) int {
	count := 0
	for _, player := range players {
		if isAssignedActor(player.ActorID) {
			count++
		}
	}
	return count
}

func assignedDocumentPlayerCount(players []playerDocument) int {
	count := 0
	for _, player := range players {
		if isAssignedActor(player.ActorID) {
			count++
		}
	}
	return count
}

func playerIDForActor(game gameDocument, actor store.Actor) (models.PlayerID, bool) {
	actorID := strings.TrimSpace(actor.ID)
	for _, player := range game.Players {
		if player.ActorID == actorID && isAssignedActor(player.ActorID) {
			return player.ID, true
		}
		if actor.Development && player.ActorID == "" && string(player.ID) == actorID {
			return player.ID, true
		}
	}
	return "", false
}

func membershipForActor(game gameDocument, actor store.Actor) bool {
	_, ok := playerIDForActor(game, actor)
	return ok
}

func (s *FirestoreStore) deleteGame(ctx context.Context, id store.GameID) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	root := gameRef(s.client, id)
	snapshot, err := root.Get(operationContext)
	s.recordReads(1)
	if isCode(err, codes.NotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	game, err := decodeGameDocument(snapshot)
	if err != nil {
		return err
	}
	refs := []*cloudfirestore.DocumentRef{
		canonicalRef(s.client, id),
		root,
	}
	collect := func(documents *cloudfirestore.DocumentIterator) error {
		for {
			document, err := documents.Next()
			if err == iterator.Done {
				return nil
			}
			if err != nil {
				return err
			}
			s.recordReads(1)
			refs = append(refs, document.Ref)
		}
	}
	if err := collect(root.Collection("views").Documents(operationContext)); err != nil {
		return err
	}
	if err := collect(reportCollection(s.client, id).Documents(operationContext)); err != nil {
		return err
	}
	if err := collect(submissionCollection(s.client, id, game.Turn).Documents(operationContext)); err != nil {
		return err
	}
	for _, player := range game.Players {
		if !isAssignedActor(player.ActorID) {
			continue
		}
		if err := collect(gameRef(s.client, id).Collection("reports").Doc(player.ActorID).Collection("turns").Documents(operationContext)); err != nil {
			return err
		}
	}
	invitations := s.client.Collection("invitations").Where("gameId", "==", id).Documents(operationContext)
	if err := collect(invitations); err != nil {
		return err
	}
	for start := 0; start < len(refs); start += 450 {
		end := min(start+450, len(refs))
		batch := s.client.Batch()
		for _, ref := range refs[start:end] {
			batch.Delete(ref)
		}
		if _, err := batch.Commit(operationContext); err != nil {
			return err
		}
		s.recordWrites(end - start)
	}
	return nil
}
