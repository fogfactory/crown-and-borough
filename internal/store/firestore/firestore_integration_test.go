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
	if _, err := first.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code); err != nil {
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
		if player.Name != "Robert" {
			t.Fatalf("creator view player P2 name after profile update = %q, want Robert", player.Name)
		}
	}
	if !foundRenamedPlayer {
		t.Fatal("creator view after profile update does not contain player P2")
	}
	updated, err := first.Get(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID)
	if err != nil {
		t.Fatalf("read renamed game: %v", err)
	}
	if updated.Players[1].Name != "Robert" || updated.State.Players[1].Name != "Robert" {
		t.Fatalf("renamed Bob = slot %q/state %q, want Robert", updated.Players[1].Name, updated.State.Players[1].Name)
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
	created, err := gameStore.Create(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
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
	created, err := gameStore.Create(ctx, store.Actor{ID: "alice"}, store.CreateRequest{
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
	joined, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code)
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
	if _, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code); err != nil {
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
	if _, err := gameStore.Join(ctx, store.Actor{ID: "bob"}, created.Snapshot.ID, created.Invitation.Code); !errors.Is(err, store.ErrRevisionConflict) {
		t.Fatalf("join during claim error = %v, want revision conflict", err)
	}
	canonical.Resolution = nil
	if _, err := canonicalRef(client, created.Snapshot.ID).Set(ctx, canonical); err != nil {
		t.Fatalf("clear active claim: %v", err)
	}
}
