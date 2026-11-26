//go:build integration

package firestorestore

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestFirestoreStorePersistsAndRestoresATurn(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	options := Options{Client: client, StrictMembership: true}
	first := NewWithClient(balance, assets, options)
	if _, err := first.EnsureProfile(ctx, store.Actor{ID: "alice", Email: "alice@example.com"}); err != nil {
		t.Fatalf("ensure Alice: %v", err)
	}
	if _, err := first.UpdateProfile(ctx, "alice", "Alice"); err != nil {
		t.Fatalf("profile Alice: %v", err)
	}
	if _, err := first.EnsureProfile(ctx, store.Actor{ID: "bob", Email: "bob@example.com"}); err != nil {
		t.Fatalf("ensure Bob: %v", err)
	}
	if _, err := first.UpdateProfile(ctx, "bob", "Bob"); err != nil {
		t.Fatalf("profile Bob: %v", err)
	}
	created, err := first.CreateWithInvitation(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Persistent game",
		Seed:    "firestore-integration",
		Players: []engine.PlayerInit{{Name: "Alice"}, {}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := first.ConfirmLobby(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID); err != nil {
		t.Fatalf("confirm creator: %v", err)
	}
	if _, err := first.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{}); err != nil {
		t.Fatalf("join game: %v", err)
	}
	creatorViewSnapshot, err := viewRef(first.client, created.Snapshot.ID, "alice").Get(ctx)
	if err != nil {
		t.Fatalf("read creator view after join: %v", err)
	}
	var creatorView viewDocument
	if err := creatorViewSnapshot.DataTo(&creatorView); err != nil {
		t.Fatalf("decode creator view after join: %v", err)
	}
	var creatorState api.StateView
	if err := decodeJSONMap(creatorView.State, &creatorState); err != nil {
		t.Fatalf("decode creator state after join: %v", err)
	}
	foundJoinedPlayer := false
	for _, player := range creatorState.Players {
		if player.ID != "P2" {
			continue
		}
		foundJoinedPlayer = true
		if player.Name != "Bob" {
			t.Fatalf("creator view player P2 name = %q, want Bob", player.Name)
		}
	}
	if !foundJoinedPlayer {
		t.Fatal("creator view does not contain player P2")
	}
	if _, err := first.EnsureProfile(ctx, store.Actor{ID: "bob", Email: "bob@example.com"}); err != nil {
		t.Fatalf("ensure Bob profile: %v", err)
	}
	if _, err := first.UpdateProfile(ctx, "bob", "Robert"); err != nil {
		t.Fatalf("update Bob profile: %v", err)
	}
	creatorViewSnapshot, err = viewRef(first.client, created.Snapshot.ID, "alice").Get(ctx)
	if err != nil {
		t.Fatalf("read creator view after profile update: %v", err)
	}
	if err := creatorViewSnapshot.DataTo(&creatorView); err != nil {
		t.Fatalf("decode creator view after profile update: %v", err)
	}
	creatorState = api.StateView{}
	if err := decodeJSONMap(creatorView.State, &creatorState); err != nil {
		t.Fatalf("decode creator state after profile update: %v", err)
	}
	foundRenamedPlayer := false
	for _, player := range creatorState.Players {
		if player.ID != "P2" {
			continue
		}
		foundRenamedPlayer = true
		// The house name is its own: a profile update moves the account name only.
		if player.Name != "Bob" {
			t.Fatalf("creator view player P2 name after profile update = %q, want Bob", player.Name)
		}
	}
	if !foundRenamedPlayer {
		t.Fatal("creator view after profile update does not contain player P2")
	}
	updated, err := first.Get(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("read renamed game: %v", err)
	}
	if updated.Players[1].Name != "Bob" || updated.State.Players[1].Name != "Bob" || updated.Players[1].AccountName != "Robert" {
		t.Fatalf("renamed Bob = slot %q/state %q/account %q, want house Bob, account Robert", updated.Players[1].Name, updated.State.Players[1].Name, updated.Players[1].AccountName)
	}
	aliceGames, err := first.List(ctx, store.Actor{ID: "alice"})
	if err != nil {
		t.Fatalf("list Alice games: %v", err)
	}
	if len(aliceGames) != 1 || aliceGames[0].ID != created.Snapshot.ID {
		t.Fatalf("Alice games = %#v, want the created game", aliceGames)
	}
	bobMemberships, err := first.ListActorMemberships(ctx, "bob")
	if err != nil {
		t.Fatalf("list Bob memberships: %v", err)
	}
	if len(bobMemberships) != 1 || bobMemberships[0].GameID != created.Snapshot.ID {
		t.Fatalf("Bob memberships = %#v, want the created game", bobMemberships)
	}
	pending, err := first.Submit(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID, store.SubmitRequest{})
	if err != nil {
		t.Fatalf("partial submit: %v", err)
	}
	if pending.Status != "pending" || len(pending.Snapshot.Submissions) != 1 {
		t.Fatalf("pending result = %#v", pending)
	}

	// A second adapter instance reads the same documents and therefore models a
	// Cloud Run restart without sharing any in-memory game state.
	restarted := NewWithClient(balance, assets, options)
	restored, err := restarted.Get(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("restore after restart: %v", err)
	}
	if restored.Revision != pending.Snapshot.Revision || restored.State.Turn != 1 {
		t.Fatalf("restored snapshot = revision %d turn %d", restored.Revision, restored.State.Turn)
	}
	aliceSub, err := restarted.MySubmission(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("alice MySubmission after restart: %v", err)
	}
	if aliceSub.Status != store.OrdersSubmitted || aliceSub.Turn != 1 {
		t.Fatalf("alice MySubmission after restart = %#v, want submitted:true turn 1", aliceSub)
	}
	bobSub, err := restarted.MySubmission(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("bob MySubmission before submitting: %v", err)
	}
	if bobSub.Status != store.OrdersPending {
		t.Fatalf("bob MySubmission before submitting = %#v, want submitted:false", bobSub)
	}
	resolved, err := restarted.Submit(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, store.SubmitRequest{ExpectedRevision: restored.Revision})
	if err != nil {
		t.Fatalf("resolve restored turn: %v", err)
	}
	if resolved.Status != "resolved" || resolved.Snapshot.State.Turn != 2 || len(resolved.Snapshot.Reports) != 1 {
		t.Fatalf("resolved result = %#v", resolved)
	}
	metrics := first.Metrics()
	if metrics.Reads == 0 || metrics.Writes == 0 || metrics.Transactions == 0 || metrics.ProjectionWrites == 0 {
		t.Fatalf("Firestore metrics = %+v, want reads, writes, transactions, and projections", metrics)
	}
	if err := first.deleteGame(ctx, created.Snapshot.ID); err != nil {
		t.Fatalf("delete game subtree: %v", err)
	}
	if _, err := gameRef(first.client, created.Snapshot.ID).Get(ctx); !isCode(err, codes.NotFound) {
		t.Fatalf("deleted game read error = %v, want not found", err)
	}
	if _, err := first.LookupInvitation(ctx, created.Invitation.Code); !errors.Is(err, store.ErrInvalidInvitation) {
		t.Fatalf("deleted invitation error = %v, want invalid invitation", err)
	}
}

// The views written when a year enters winter carry the winter aids, whose
// fief sites must stay storable: Firestore rejects nested arrays.
func TestFirestoreStoreResolvesIntoWinter(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	s := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	alice := store.Actor{ID: "winter-alice"}
	created, err := startedGame(ctx, s, alice, store.CreateRequest{
		Name:    "Winter game",
		Seed:    "firestore-winter",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	id := created.ID
	defer func() {
		if err := s.deleteGame(ctx, id); err != nil {
			t.Errorf("delete game subtree: %v", err)
		}
	}()
	// Spring, summer and autumn: the third forced resolution enters winter.
	for range 3 {
		if _, err := s.Resolve(ctx, alice, id); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}
	viewSnapshot, err := viewRef(s.client, id, alice.ID).Get(ctx)
	if err != nil {
		t.Fatalf("read winter view: %v", err)
	}
	var view viewDocument
	if err := viewSnapshot.DataTo(&view); err != nil {
		t.Fatalf("decode winter view: %v", err)
	}
	var state api.StateView
	if err := decodeJSONMap(view.State, &state); err != nil {
		t.Fatalf("decode winter state: %v", err)
	}
	if state.Season != models.SeasonWinter || state.WinterAids == nil || len(state.WinterAids.FiefSites) == 0 {
		t.Fatalf("winter view = season %s aids %+v, want winter with a fief site", state.Season, state.WinterAids)
	}
	if len(state.WinterAids.FiefSites[0].Edges) == 0 {
		t.Fatalf("fief site %+v has no edges", state.WinterAids.FiefSites[0])
	}
}

// A resolution whose commit fails must not keep the game locked until the
// lease expires: the next resolution goes through at once.
func TestFirestoreStoreReleasesTheClaimWhenTheCommitFails(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	// A lease far longer than the test: only an explicit release frees the game.
	s := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true, LeaseTimeout: time.Hour})
	alice := store.Actor{ID: "commit-alice"}
	created, err := startedGame(ctx, s, alice, store.CreateRequest{
		Name:    "Failed commit",
		Seed:    "firestore-failed-commit",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	id := created.ID
	defer func() {
		if err := s.deleteGame(ctx, id); err != nil {
			t.Errorf("delete game subtree: %v", err)
		}
	}()
	commitErr := errors.New("commit refused")
	s.commitFault = func() error { return commitErr }
	if _, err := s.Resolve(ctx, alice, id); !errors.Is(err, commitErr) {
		t.Fatalf("resolve with a failing commit error = %v, want %v", err, commitErr)
	}
	canonicalSnapshot, err := canonicalRef(s.client, id).Get(ctx)
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	canonical, err := decodeCanonicalDocument(canonicalSnapshot)
	if err != nil {
		t.Fatalf("decode canonical: %v", err)
	}
	if canonical.Resolution != nil || canonical.Turn != 1 {
		t.Fatalf("canonical after the failed commit = turn %d claim %+v, want turn 1 and no claim", canonical.Turn, canonical.Resolution)
	}
	s.commitFault = nil
	resolved, err := s.Resolve(ctx, alice, id)
	if err != nil {
		t.Fatalf("resolve after the failed commit: %v", err)
	}
	if resolved.Snapshot.State.Turn != 2 {
		t.Fatalf("turn after resolution = %d, want 2", resolved.Snapshot.State.Turn)
	}
}

func TestFirestoreStoreReadinessAcceptsAReachableMissingDocument(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	client, err := cloudfirestore.NewClient(context.Background(), projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	adapter := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{Client: client, OperationTimeout: time.Second})
	if err := adapter.Ready(context.Background()); err != nil {
		t.Fatalf("readiness check = %v, want a reachable Firestore", err)
	}
}

func TestFirestoreStoreConcurrentResolutionClaimsOneOperation(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	created, err := startedGame(ctx, gameStore, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Concurrent resolution",
		Seed:    "firestore-concurrent-resolution",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}

	results := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, resolveErr := gameStore.ResolveAt(ctx, store.Actor{ID: "alice"}, created.ID, created.Revision)
			results <- resolveErr
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	conflicts := 0
	for resolveErr := range results {
		switch {
		case resolveErr == nil:
			successes++
		case errors.Is(resolveErr, store.ErrRevisionConflict):
			conflicts++
		default:
			t.Fatalf("concurrent resolve error = %v", resolveErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent resolve outcomes = %d successes/%d conflicts, want 1/1", successes, conflicts)
	}
	final, err := gameStore.Get(ctx, store.Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("read final game: %v", err)
	}
	if final.State.Turn != 2 || len(final.Reports) != 1 || final.Revision != created.Revision+1 {
		t.Fatalf("final game = turn %d reports %d revision %d, want 2/1/%d", final.State.Turn, len(final.Reports), final.Revision, created.Revision+1)
	}
}

func TestFirestoreStoreRestoreReclaimsExpiredResolutionLease(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	created, err := startedGame(ctx, gameStore, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Recoverable claim",
		Seed:    "firestore-recoverable-claim",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	canonicalSnapshot, err := canonicalRef(client, created.ID).Get(ctx)
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	canonical, err := decodeCanonicalDocument(canonicalSnapshot)
	if err != nil {
		t.Fatalf("decode canonical: %v", err)
	}
	canonical.Resolution = &resolutionClaim{
		OperationID:  "crashed-operation",
		ClaimedAt:    time.Now().Add(-time.Minute),
		LeaseUntil:   time.Now().Add(-time.Second),
		BaseRevision: canonical.Revision,
		Turn:         canonical.Turn,
	}
	if _, err := canonicalRef(client, created.ID).Set(ctx, canonical); err != nil {
		t.Fatalf("seed expired claim: %v", err)
	}

	restored, err := gameStore.Restore(ctx)
	if err != nil {
		t.Fatalf("restore expired claim: %v", err)
	}
	for _, snapshot := range restored {
		if snapshot.ID != created.ID {
			continue
		}
		if snapshot.State.Turn != 2 || len(snapshot.Reports) != 1 {
			t.Fatalf("restored claim snapshot = turn %d reports %d, want 2/1", snapshot.State.Turn, len(snapshot.Reports))
		}
		return
	}
	t.Fatalf("restored games did not contain %s", created.ID)
}

func TestFirestoreStoreSubmissionsReplaceAndRejectStaleRevisions(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	created, err := gameStore.CreateWithInvitation(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Revisioned submissions",
		Seed:    "firestore-revisioned-submissions",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := gameStore.ConfirmLobby(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID); err != nil {
		t.Fatalf("confirm creator: %v", err)
	}
	joined, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{})
	if err != nil {
		t.Fatalf("join game: %v", err)
	}
	first, err := gameStore.Submit(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID, store.SubmitRequest{ExpectedRevision: joined.Snapshot.Revision})
	if err != nil {
		t.Fatalf("first submission: %v", err)
	}
	if first.Status != "pending" || first.Snapshot.Revision != joined.Snapshot.Revision+1 {
		t.Fatalf("first submission = status %q revision %d", first.Status, first.Snapshot.Revision)
	}
	replacement, err := gameStore.Submit(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID, store.SubmitRequest{ExpectedRevision: first.Snapshot.Revision})
	if err != nil {
		t.Fatalf("replacement submission: %v", err)
	}
	if replacement.Status != "pending" || replacement.Snapshot.Revision != first.Snapshot.Revision+1 || len(replacement.Snapshot.Submissions) != 1 {
		t.Fatalf("replacement = status %q revision %d submissions %d", replacement.Status, replacement.Snapshot.Revision, len(replacement.Snapshot.Submissions))
	}
	if _, err := gameStore.Submit(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, store.SubmitRequest{ExpectedRevision: first.Snapshot.Revision}); !errors.Is(err, store.ErrRevisionConflict) {
		t.Fatalf("stale submission error = %v, want revision conflict", err)
	}
	resolved, err := gameStore.Submit(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, store.SubmitRequest{ExpectedRevision: replacement.Snapshot.Revision})
	if err != nil {
		t.Fatalf("last submission: %v", err)
	}
	if resolved.Status != "resolved" || resolved.Snapshot.Revision != replacement.Snapshot.Revision+2 || resolved.Snapshot.State.Turn != 2 {
		t.Fatalf("resolved = status %q revision %d turn %d", resolved.Status, resolved.Snapshot.Revision, resolved.Snapshot.State.Turn)
	}
	if _, err := gameStore.ResolveAt(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID, replacement.Snapshot.Revision); !errors.Is(err, store.ErrRevisionConflict) {
		t.Fatalf("double resolution error = %v, want revision conflict", err)
	}
}

func TestFirestoreStoreConcurrentSubmissionsDoNotLoseOrders(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	created, err := gameStore.CreateWithInvitation(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Concurrent submissions",
		Seed:    "firestore-concurrent-submissions",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := gameStore.ConfirmLobby(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID); err != nil {
		t.Fatalf("confirm creator: %v", err)
	}
	if _, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{}); err != nil {
		t.Fatalf("join game: %v", err)
	}
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for _, actorID := range []string{"alice", "bob"} {
		actorID := actorID
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, submitErr := gameStore.Submit(ctx, store.Actor{ID: actorID}, created.Snapshot.ID, store.SubmitRequest{})
			results <- submitErr
		}()
	}
	wait.Wait()
	close(results)
	for submitErr := range results {
		if submitErr != nil {
			t.Fatalf("concurrent submission: %v", submitErr)
		}
	}
	final, err := gameStore.Get(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("read concurrent final state: %v", err)
	}
	if final.State.Turn != 2 || len(final.Reports) != 1 || len(final.Submissions) != 0 {
		t.Fatalf("concurrent final state = turn %d reports %d submissions %d", final.State.Turn, len(final.Reports), len(final.Submissions))
	}
}

func TestFirestoreStoreJoinRejectsAnActiveResolutionClaim(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	created, err := gameStore.CreateWithInvitation(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
		Name:    "Claimed join",
		Seed:    "firestore-claimed-join",
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	// A game that predates lobbies is started yet may have a free slot: start
	// this one, then free Bob's slot as such a game would have it.
	if _, err := gameStore.ConfirmLobby(ctx, store.Actor{ID: "alice"}, created.Snapshot.ID); err != nil {
		t.Fatalf("confirm creator: %v", err)
	}
	if _, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{}); err != nil {
		t.Fatalf("join game: %v", err)
	}
	gameSnapshot, err := gameRef(client, created.Snapshot.ID).Get(ctx)
	if err != nil {
		t.Fatalf("read game: %v", err)
	}
	game, err := decodeGameDocument(gameSnapshot)
	if err != nil {
		t.Fatalf("decode game: %v", err)
	}
	for index := range game.Players {
		if game.Players[index].ActorID == "bob" {
			game.Players[index].ActorID = ""
		}
	}
	game.MemberUIDs = []string{"alice"}
	if _, err := gameRef(client, created.Snapshot.ID).Set(ctx, game); err != nil {
		t.Fatalf("free Bob's slot: %v", err)
	}
	canonicalSnapshot, err := canonicalRef(client, created.Snapshot.ID).Get(ctx)
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}
	canonical, err := decodeCanonicalDocument(canonicalSnapshot)
	if err != nil {
		t.Fatalf("decode canonical: %v", err)
	}
	canonical.Resolution = &resolutionClaim{
		OperationID:  "active-resolution",
		ClaimedAt:    time.Now(),
		LeaseUntil:   time.Now().Add(time.Minute),
		BaseRevision: canonical.Revision,
		Turn:         canonical.Turn,
	}
	if _, err := canonicalRef(client, created.Snapshot.ID).Set(ctx, canonical); err != nil {
		t.Fatalf("seed active claim: %v", err)
	}
	if _, err := gameStore.Join(ctx, store.Actor{ID: "carol"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{}); !errors.Is(err, store.ErrRevisionConflict) {
		t.Fatalf("join during claim error = %v, want revision conflict", err)
	}
	canonical.Resolution = nil
	if _, err := canonicalRef(client, created.Snapshot.ID).Set(ctx, canonical); err != nil {
		t.Fatalf("clear active claim: %v", err)
	}
}

// startedGame creates a two-player online game for creator and has Bob join
// it: an online game has no board before its last slot is taken.
func startedGame(ctx context.Context, s *FirestoreStore, creator store.Actor, request store.CreateRequest) (store.GameSnapshot, error) {
	created, err := s.CreateWithInvitation(ctx, creator, request)
	if err != nil {
		return store.GameSnapshot{}, err
	}
	if _, err := s.ConfirmLobby(ctx, creator, created.Snapshot.ID); err != nil {
		return store.GameSnapshot{}, err
	}
	joined, err := s.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{})
	if err != nil {
		return store.GameSnapshot{}, err
	}
	return joined.Snapshot, nil
}

func TestFirestoreLobbyStartsWithTheLastJoinAndCountsItsEntry(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	assets, err := assetgen.Load("../../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	gameStore := NewWithClient(balance, assets, Options{Client: client, StrictMembership: true})
	alice, bob, carol := store.Actor{ID: "lobby-alice"}, store.Actor{ID: "lobby-bob"}, store.Actor{ID: "lobby-carol"}
	created, err := gameStore.CreateWithInvitation(ctx, alice, store.CreateRequest{
		Name:    "Lobby game",
		Players: []engine.PlayerInit{{}, {}, {}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	id := created.Snapshot.ID
	defer func() {
		if err := gameStore.deleteGame(ctx, id); err != nil {
			t.Errorf("delete game subtree: %v", err)
		}
	}()
	// A lobby holds the game document alone: no canonical state, no view.
	if _, err := canonicalRef(client, id).Get(ctx); !isCode(err, codes.NotFound) {
		t.Fatalf("canonical before start = %v, want not found", err)
	}
	if _, err := gameStore.State(ctx, alice, id); !errors.Is(err, store.ErrGameNotStarted) {
		t.Fatalf("state in the lobby = %v, want ErrGameNotStarted", err)
	}
	if _, err := gameStore.SetSeedVote(ctx, alice, id, []int{1, 0, 2}); err != nil {
		t.Fatalf("vote: %v", err)
	}
	if _, err := gameStore.RenameHouse(ctx, alice, id, "House Stark"); err != nil {
		t.Fatalf("lobby name: %v", err)
	}
	// The vote sits in a backend-only document, never on the game document.
	if _, err := gameStore.Join(ctx, bob, id, created.Invitation.Code, store.JoinOptions{SeedRanking: []int{1, 2, 0}, HouseName: "House Tyrell"}); err != nil {
		t.Fatalf("join bob: %v", err)
	}
	// Pending names live on the game document (so uniqueness needs no lobby
	// scan); the lobby documents keep rankings only.
	lobbyGame, err := gameRef(client, id).Get(ctx)
	if err != nil {
		t.Fatalf("read lobby game: %v", err)
	}
	lobbyDoc, err := decodeGameDocument(lobbyGame)
	if err != nil {
		t.Fatalf("decode lobby game: %v", err)
	}
	if lobbyDoc.Phase != store.PhaseLobby || lobbyDoc.HouseNames["lobby-alice"] != "House Stark" || lobbyDoc.HouseNames["lobby-bob"] != "House Tyrell" {
		t.Fatalf("lobby document = phase %q, names %v", lobbyDoc.Phase, lobbyDoc.HouseNames)
	}
	if _, err := gameStore.RenameHouse(ctx, bob, id, "HOUSE STARK"); !errors.Is(err, store.ErrHouseNameTaken) {
		t.Fatalf("a name another member entered = %v, want ErrHouseNameTaken", err)
	}
	rawLobby, err := lobbyRef(client, id, "lobby-bob").Get(ctx)
	if err != nil {
		t.Fatalf("read bob lobby document: %v", err)
	}
	if _, hasName := rawLobby.DataAt("houseName"); hasName == nil {
		t.Fatal("the lobby document must not hold the house name")
	}
	if _, err := gameStore.Join(ctx, bob, id, created.Invitation.Code, store.JoinOptions{HouseName: "house stark"}); err != nil {
		t.Fatalf("an existing member joining again must be idempotent: %v", err)
	}
	// The last slot is taken, but the creator has not confirmed yet.
	filled, err := gameStore.Join(ctx, carol, id, created.Invitation.Code, store.JoinOptions{SeedRanking: []int{2, 1, 0}, HouseName: "House Arryn"})
	if err != nil {
		t.Fatalf("join carol: %v", err)
	}
	if filled.Started || filled.Snapshot.Started() {
		t.Fatal("the game started before its creator confirmed")
	}
	if _, err := gameStore.ConfirmLobby(ctx, bob, id); !errors.Is(err, store.ErrNotCreator) {
		t.Fatalf("confirm by a guest = %v, want ErrNotCreator", err)
	}
	last, err := gameStore.ConfirmLobby(ctx, alice, id)
	if err != nil {
		t.Fatalf("confirm alice: %v", err)
	}
	if !last.Started || !last.Snapshot.Started() {
		t.Fatal("the creator's confirmation did not start the game")
	}
	// Borda: candidate 1 = 2+2+1, candidate 2 = 0+... : candidate 1 wins.
	if want := store.SeedCandidates(id, 3, assets)[1]; last.Snapshot.Seed != want {
		t.Fatalf("seed = %s, want candidate 1 %s", last.Snapshot.Seed, want)
	}
	if last.Player.Name != "House Stark" {
		t.Fatalf("alice's house = %q", last.Player.Name)
	}
	if _, err := canonicalRef(client, id).Get(ctx); err != nil {
		t.Fatalf("canonical after start: %v", err)
	}
	startedGame, err := gameRef(client, id).Get(ctx)
	if err != nil {
		t.Fatalf("read started game: %v", err)
	}
	if doc, decodeErr := decodeGameDocument(startedGame); decodeErr != nil || doc.Phase != store.PhaseStarted || len(doc.HouseNames) != 0 {
		t.Fatalf("started document = phase %q, names %v, %v", doc.Phase, doc.HouseNames, decodeErr)
	}
	for _, actor := range []store.Actor{alice, bob, carol} {
		if _, err := viewRef(client, id, actor.ID).Get(ctx); err != nil {
			t.Fatalf("view of %s after start: %v", actor.ID, err)
		}
	}
	if _, err := gameStore.SetSeedVote(ctx, alice, id, []int{0, 1, 2}); !errors.Is(err, store.ErrGameStarted) {
		t.Fatalf("vote after start = %v, want ErrGameStarted", err)
	}
}
