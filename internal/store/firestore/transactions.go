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
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
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

func (s *FirestoreStore) Join(ctx context.Context, actor store.Actor, id store.GameID, code string) (store.JoinResult, error) {
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
	var revision store.Revision
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
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
		freeIndex := -1
		for index, player := range game.Players {
			if !isAssignedActor(player.ActorID) {
				freeIndex = index
				break
			}
		}
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
			for index := range state.Players {
				if state.Players[index].ID == player.ID {
					state.Players[index].Name = profileName
					break
				}
			}
		}
		game.MemberUIDs = append(game.MemberUIDs, actorID)
		game.Revision++
		game.UpdatedAt = s.now().UTC()
		canonical.Revision = game.Revision
		canonical.UpdatedAt = game.UpdatedAt
		canonical.State, err = jsonMap(state)
		if err != nil {
			return err
		}
		if err := state.Validate(); err != nil {
			return err
		}
		joinedPlayer = player.ID
		revision = store.Revision(game.Revision)
		if err := transaction.Set(gameDocumentRef, game); err != nil {
			return err
		}
		if err := transaction.Set(canonicalDocumentRef, canonical); err != nil {
			return err
		}
		// A join changes player metadata in the canonical state. Refresh existing
		// views too, so connected players do not keep stale slot names.
		views, err := s.viewProjections(id, game.Players, revision, state, game.UpdatedAt)
		if err != nil {
			return err
		}
		for _, projection := range views {
			if err := transaction.Set(viewRef(s.client, id, projection.actorID), projection.document); err != nil {
				return err
			}
		}
		s.recordWrites(2 + len(views))
		s.recordProjectionWrites(len(views))
		return nil
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
	snapshot, err := s.Get(operationContext, actor, id)
	if err != nil {
		return store.JoinResult{}, err
	}
	return store.JoinResult{Snapshot: snapshot, Player: playerSlot(snapshot.Players, joinedPlayer), Joined: true}, nil
}

func (s *FirestoreStore) Submit(ctx context.Context, actor store.Actor, id store.GameID, request store.SubmitRequest) (store.SubmitResult, error) {
	if err := s.requireClient(); err != nil {
		return store.SubmitResult{}, err
	}
	snapshot, err := s.Get(ctx, actor, id)
	if err != nil {
		return store.SubmitResult{}, err
	}
	playerID, ok := snapshotPlayerID(snapshot, actor)
	if !ok {
		return store.SubmitResult{}, store.ErrNotMember
	}
	if snapshot.Status == store.StatusFinished {
		return store.SubmitResult{}, store.ErrGameFinished
	}
	if !isAlive(snapshot.State, playerID) {
		return store.SubmitResult{}, store.ErrEliminated
	}
	if request.ExpectedRevision != 0 && request.ExpectedRevision != snapshot.Revision {
		return store.SubmitResult{}, store.ErrRevisionConflict
	}
	input, err := normalizeSubmission(playerID, request)
	if err != nil {
		return store.SubmitResult{}, err
	}
	// Validate before writing. The transaction below still checks the revision
	// and turn, so this read cannot make an obsolete mutation succeed.
	if _, err := engine.ResolveTurn(snapshot.State, s.balance, input); err != nil {
		return store.SubmitResult{}, err
	}
	write, err := s.writeSubmissionForGame(ctx, actor, id, input, request.ExpectedRevision)
	if err != nil {
		return store.SubmitResult{}, err
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
		_ = s.releaseResolution(operationContext, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	if snapshot.Revision != store.Revision(claim.Claim.BaseRevision) {
		_ = s.releaseResolution(operationContext, id, claim.Claim)
		return store.SubmitResult{}, store.ErrRevisionConflict
	}
	submitted, remaining := submissionStatus(snapshot)
	combined := combineSubmissions(snapshot)
	before := snapshot.State
	report, err := engine.ResolveTurn(before, s.balance, combined)
	if err != nil {
		_ = s.releaseResolution(operationContext, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	if s.privacyTracker != nil {
		s.privacyTracker(before, report.State, combined, report)
	}
	if err := report.State.Validate(); err != nil {
		_ = s.releaseResolution(operationContext, id, claim.Claim)
		return store.SubmitResult{}, err
	}
	if err := s.commitResolution(operationContext, claim, snapshot, report); err != nil {
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
		if !member {
			return store.ErrNotMember
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

func (s *FirestoreStore) commitResolution(ctx context.Context, claim resolutionClaimResult, snapshot store.GameSnapshot, report engine.TurnReport) error {
	gameRefValue := gameRef(s.client, snapshot.ID)
	canonicalRefValue := canonicalRef(s.client, snapshot.ID)
	s.recordTransaction()
	return wrapTransactionResult(s.client.RunTransaction(ctx, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
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
		game.Scores = scoreDocuments(engine.ComputeScores(report.State))
		game.Revision = canonical.Revision
		game.SubmittedUIDs = []string{}
		game.UpdatedAt = updatedAt
		game.Status, game.WinnerUID = statusForState(report.State)
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
				if isAssignedActor(player.ActorID) {
					oldReportDeletes++
					if err := transaction.Delete(filteredReportRef(s.client, snapshot.ID, player.ActorID, oldTurn)); err != nil {
						return err
					}
				}
			}
		}
		s.recordWrites(3 + len(views)*2 + len(refs) + oldReportDeletes)
		s.recordProjectionWrites(len(views) * 2)
		return nil
	}))
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

func wrapTransactionResult(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrRevisionConflict) || errors.Is(err, store.ErrUnknownGame) || errors.Is(err, store.ErrNotMember) || errors.Is(err, store.ErrGameFull) || errors.Is(err, store.ErrInvalidInvitation) || errors.Is(err, store.ErrInvitationInactive) || errors.Is(err, store.ErrGameFinished) || errors.Is(err, store.ErrProfileNotFound) {
		return err
	}
	if status.Code(err) == codes.Aborted || status.Code(err) == codes.FailedPrecondition {
		return store.ErrRevisionConflict
	}
	return err
}

func snapshotPlayerID(snapshot store.GameSnapshot, actor store.Actor) (models.PlayerID, bool) {
	for _, player := range snapshot.Players {
		if player.ActorID == actor.ID && isAssignedActor(player.ActorID) {
			return player.ID, true
		}
		if actor.Development && player.ActorID == "" && string(player.ID) == actor.ID {
			return player.ID, true
		}
	}
	return "", false
}

func normalizeSubmission(playerID models.PlayerID, request store.SubmitRequest) (engine.OrdersInput, error) {
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

func isAlive(state *models.GameState, playerID models.PlayerID) bool {
	return engine.PlayerAlive(state, playerID)
}

func submissionStatus(snapshot store.GameSnapshot) ([]models.PlayerID, []models.PlayerID) {
	submitted := make([]models.PlayerID, 0, len(snapshot.Submissions))
	remaining := make([]models.PlayerID, 0, len(snapshot.Players))
	for _, player := range snapshot.Players {
		if _, ok := snapshot.Submissions[player.ID]; ok {
			submitted = append(submitted, player.ID)
			continue
		}
		if isAlive(snapshot.State, player.ID) {
			remaining = append(remaining, player.ID)
		}
	}
	return submitted, remaining
}

func combineSubmissions(snapshot store.GameSnapshot) engine.OrdersInput {
	combined := engine.OrdersInput{Chains: []engine.ChainSubmission{}, Winter: []engine.WinterSubmission{}}
	for _, player := range snapshot.State.Players {
		input, ok := snapshot.Submissions[player.ID]
		if !ok {
			continue
		}
		combined.Chains = append(combined.Chains, input.Chains...)
		combined.Winter = append(combined.Winter, input.Winter...)
	}
	return combined
}

func statusForState(state *models.GameState) (store.Status, string) {
	if !engine.GameFinished(state) {
		return store.StatusPlaying, ""
	}
	winner := engine.WinnerForFinishedGame(state)
	if winner != nil {
		return store.StatusFinished, string(*winner)
	}
	return store.StatusFinished, ""
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

func (s *FirestoreStore) writeSubmissionForGame(ctx context.Context, actor store.Actor, id store.GameID, input engine.OrdersInput, expected store.Revision) (submissionWriteResult, error) {
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
		if !isAlive(state, playerID) {
			return store.ErrEliminated
		}
		orders, err := ordersJSON(input)
		if err != nil {
			return err
		}
		now := s.now().UTC()
		submission := submissionDocument{
			SchemaVersion: schemaVersion,
			UID:           actor.ID,
			PlayerID:      string(playerID),
			Turn:          game.Turn,
			OrdersJSON:    orders,
			SubmittedAt:   now,
		}
		if err := transaction.Set(submissionCollection(s.client, id, game.Turn).Doc(actor.ID), submission); err != nil {
			return err
		}
		canonical.SubmittedUIDs = appendUnique(canonical.SubmittedUIDs, actor.ID)
		sort.Strings(canonical.SubmittedUIDs)
		canonical.Revision++
		canonical.UpdatedAt = now
		game.SubmittedUIDs = appendUnique(game.SubmittedUIDs, actor.ID)
		sort.Strings(game.SubmittedUIDs)
		game.Revision = canonical.Revision
		game.UpdatedAt = now
		if err := transaction.Set(canonicalRef(s.client, id), canonical); err != nil {
			return err
		}
		if err := transaction.Set(gameRef(s.client, id), game); err != nil {
			return err
		}
		for _, player := range game.Players {
			if !isAssignedActor(player.ActorID) {
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
		result = submissionWriteResult{
			Revision:  store.Revision(canonical.Revision),
			PlayerID:  playerID,
			Submitted: submittedFromUIDs(game, canonical.SubmittedUIDs),
			Remaining: remainingFromUIDs(game, state, canonical.SubmittedUIDs),
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

func submittedFromUIDs(game gameDocument, uids []string) []models.PlayerID {
	result := make([]models.PlayerID, 0, len(uids))
	for _, player := range game.Players {
		if slices.Contains(uids, player.ActorID) {
			result = append(result, player.ID)
		}
	}
	return result
}

func remainingFromUIDs(game gameDocument, state *models.GameState, uids []string) []models.PlayerID {
	result := make([]models.PlayerID, 0)
	for _, player := range game.Players {
		if slices.Contains(uids, player.ActorID) {
			continue
		}
		if isAlive(state, player.ID) {
			result = append(result, player.ID)
		}
	}
	return result
}
