//go:build integration

package firestorestore

import (
	"context"
	"os"
	"testing"

	cloudfirestore "cloud.google.com/go/firestore"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func newDraftTestStore(t *testing.T) (*FirestoreStore, *cloudfirestore.Client) {
	t.Helper()
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
	client, err := cloudfirestore.NewClient(context.Background(), projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return NewWithClient(balance, assets, Options{Client: client, StrictMembership: true}), client
}

// startDraftGame starts a two-house game for the accounts of one test only:
// other integration tests count the games of "alice" and "bob".
func startDraftGame(t *testing.T, gameStore *FirestoreStore, name, creatorID, guestID string) store.GameSnapshot {
	t.Helper()
	ctx := context.Background()
	creator := store.Actor{ID: creatorID}
	created, err := gameStore.CreateWithInvitation(ctx, creator, store.CreateRequest{
		Name:    name,
		Seed:    "firestore-" + creatorID,
		Players: []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}
	if _, err := gameStore.ConfirmLobby(ctx, creator, created.Snapshot.ID); err != nil {
		t.Fatalf("confirm creator: %v", err)
	}
	joined, err := gameStore.Join(ctx, store.Actor{ID: guestID}, created.Snapshot.ID, created.Invitation.Code, store.JoinOptions{})
	if err != nil {
		t.Fatalf("join game: %v", err)
	}
	return joined.Snapshot
}

func nameLocked(snapshot store.GameSnapshot, actorID string) bool {
	for _, player := range snapshot.Players {
		if player.ActorID == actorID {
			return player.NameLocked
		}
	}
	return false
}

func TestFirestoreStoreDraftsSealAndBreakTheSeal(t *testing.T) {
	ctx := context.Background()
	gameStore, client := newDraftTestStore(t)
	alice := store.Actor{ID: "draft-alice"}
	id := startDraftGame(t, gameStore, "Drafts", "draft-alice", "draft-bob").ID

	draft, err := gameStore.Submit(ctx, alice, id, store.SubmitRequest{Status: store.OrdersDraft})
	if err != nil {
		t.Fatalf("save draft: %v", err)
	}
	if draft.Status != "draft" || len(draft.Snapshot.Submissions) != 0 || !draft.Snapshot.Drafted[draft.Player] {
		t.Fatalf("draft = status %q submissions %d drafted %v", draft.Status, len(draft.Snapshot.Submissions), draft.Snapshot.Drafted)
	}
	if got := draft.Snapshot.OrdersStatusOf(draft.Player); got != store.OrdersDraft {
		t.Fatalf("status of the drafting house = %q", got)
	}
	if nameLocked(draft.Snapshot, "draft-alice") {
		t.Fatal("a draft must not lock the house name")
	}
	mine, err := gameStore.MySubmission(ctx, alice, id)
	if err != nil || mine.Status != store.OrdersDraft {
		t.Fatalf("my submission = %#v, %v", mine, err)
	}
	// Bob never sees Alice's draft content, only her status.
	if bobs, err := gameStore.MySubmission(ctx, store.Actor{ID: "draft-bob"}, id); err != nil || bobs.Status != store.OrdersPending {
		t.Fatalf("Bob's submission = %#v, %v", bobs, err)
	}

	sealed, err := gameStore.Submit(ctx, alice, id, store.SubmitRequest{})
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed.Status != "pending" || len(sealed.Snapshot.Submissions) != 1 || sealed.Snapshot.Drafted[sealed.Player] || !nameLocked(sealed.Snapshot, "draft-alice") {
		t.Fatalf("sealed = status %q submissions %d drafted %v", sealed.Status, len(sealed.Snapshot.Submissions), sealed.Snapshot.Drafted)
	}
	// A document written before drafts existed has no status: it is sealed.
	document, err := submissionCollection(client, id, sealed.Snapshot.State.Turn).Doc("draft-alice").Get(ctx)
	if err != nil {
		t.Fatalf("read submission document: %v", err)
	}
	if status, _ := document.DataAt("status"); status != nil {
		t.Fatalf("a sealed document carries status %v, want none", status)
	}

	broken, err := gameStore.Submit(ctx, alice, id, store.SubmitRequest{Status: store.OrdersDraft, ExpectedRevision: sealed.Snapshot.Revision})
	if err != nil {
		t.Fatalf("break the seal: %v", err)
	}
	if broken.Status != "draft" || len(broken.Snapshot.Submissions) != 0 || !broken.Snapshot.Drafted[broken.Player] {
		t.Fatalf("broken = status %q submissions %d drafted %v", broken.Status, len(broken.Snapshot.Submissions), broken.Snapshot.Drafted)
	}
	if nameLocked(broken.Snapshot, "draft-alice") {
		t.Fatal("breaking the first seal must give the house name back")
	}
}

func TestFirestoreStoreHostResolveSealsDraftsAndClearsThem(t *testing.T) {
	ctx := context.Background()
	gameStore, _ := newDraftTestStore(t)
	alice := store.Actor{ID: "resolve-alice"}
	id := startDraftGame(t, gameStore, "Host resolve", "resolve-alice", "resolve-bob").ID
	for _, actor := range []store.Actor{alice, {ID: "resolve-bob"}} {
		if _, err := gameStore.Submit(ctx, actor, id, store.SubmitRequest{Status: store.OrdersDraft}); err != nil {
			t.Fatalf("draft %s: %v", actor.ID, err)
		}
	}
	resolved, err := gameStore.Resolve(ctx, alice, id)
	if err != nil {
		t.Fatalf("host resolve: %v", err)
	}
	if resolved.Status != "resolved" || resolved.Snapshot.State.Turn != 2 || len(resolved.Submitted) != 2 {
		t.Fatalf("resolved = status %q turn %d submitted %v", resolved.Status, resolved.Snapshot.State.Turn, resolved.Submitted)
	}
	if len(resolved.Snapshot.Drafted) != 0 {
		t.Fatalf("drafts survive the resolution: %v", resolved.Snapshot.Drafted)
	}
	for _, actor := range []string{"resolve-alice", "resolve-bob"} {
		if !nameLocked(resolved.Snapshot, actor) {
			t.Fatalf("the house of %s was sealed by the host: its name must be locked", actor)
		}
		mine, err := gameStore.MySubmission(ctx, store.Actor{ID: actor}, id)
		if err != nil || mine.Status != store.OrdersPending || mine.Turn != 2 {
			t.Fatalf("%s next turn = %#v, %v", actor, mine, err)
		}
	}
}
