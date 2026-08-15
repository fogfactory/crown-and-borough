package store

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine"
)

func TestMemoryProfileStoreEnsuresAndUpdatesProfiles(t *testing.T) {
	profiles := NewMemoryProfileStore()
	ctx := context.Background()

	profile, err := profiles.EnsureProfile(ctx, Actor{ID: "alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("ensure profile: %v", err)
	}
	if profile.UID != "alice" || profile.Email != "alice@example.com" || profile.DisplayName != "" {
		t.Fatalf("initial profile = %#v", profile)
	}
	if _, err := profiles.UpdateProfile(ctx, "alice", "  Alice   de   York "); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	updated, err := profiles.GetProfile(ctx, "alice")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if updated.DisplayName != "Alice de York" {
		t.Fatalf("normalized display name = %q", updated.DisplayName)
	}
	if _, err := profiles.UpdateProfile(ctx, "alice", ""); !errors.Is(err, ErrInvalidDisplayName) {
		t.Fatalf("empty display name error = %v", err)
	}
	if _, err := profiles.UpdateProfile(ctx, "alice", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"); !errors.Is(err, ErrInvalidDisplayName) {
		t.Fatalf("long display name error = %v", err)
	}
}

func TestMemoryProfileStoreConcurrentEnsureIsIdempotent(t *testing.T) {
	profiles := NewMemoryProfileStore()
	const callers = 32
	results := make(chan PlayerProfile, callers)
	errorsCh := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			profile, err := profiles.EnsureProfile(context.Background(), Actor{ID: "same-user", Email: "same@example.com"})
			if err != nil {
				errorsCh <- err
				return
			}
			results <- profile
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatalf("concurrent ensure: %v", err)
	}
	var first PlayerProfile
	for profile := range results {
		if first.UID == "" {
			first = profile
			continue
		}
		if profile.CreatedAt != first.CreatedAt || profile.UID != first.UID {
			t.Fatalf("concurrent profiles differ: %#v and %#v", first, profile)
		}
	}
}

func TestMemoryInvitationStoreStoresOnlyCodeHash(t *testing.T) {
	code := "ABC234"
	invitations := NewMemoryInvitationStoreWithGenerator(func() (string, error) { return code, nil })
	secret, err := invitations.CreateInvitation(context.Background(), "game-1", "alice")
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}
	if secret.Code != code || len(secret.Code) != InvitationCodeLength {
		t.Fatalf("secret = %#v", secret)
	}
	if secret.Code == InvitationCodeHash(secret.Code) {
		t.Fatal("invitation code was returned as its hash")
	}
	invitation, err := invitations.LookupInvitation(context.Background(), code)
	if err != nil {
		t.Fatalf("lookup invitation: %v", err)
	}
	if invitation.CodeHash != InvitationCodeHash(code) || invitation.CodeHash == code {
		t.Fatalf("stored invitation = %#v", invitation)
	}
	if _, err := invitations.LookupInvitation(context.Background(), "wrong1"); !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("invalid invitation error = %v", err)
	}
}

func TestMemoryInvitationStoreRetriesCodeCollisions(t *testing.T) {
	codes := []string{"ABC234", "DEF234"}
	index := 0
	invitations := NewMemoryInvitationStoreWithGenerator(func() (string, error) {
		code := codes[index]
		if index < len(codes)-1 {
			index++
		}
		return code, nil
	})
	first, err := invitations.CreateInvitation(context.Background(), "game-1", "alice")
	if err != nil {
		t.Fatalf("first invitation: %v", err)
	}
	second, err := invitations.CreateInvitation(context.Background(), "game-2", "bob")
	if err != nil {
		t.Fatalf("second invitation: %v", err)
	}
	if first.Code == second.Code {
		t.Fatalf("collision was not retried: %q", first.Code)
	}
}

func TestMemoryStoreConcurrentJoinsClaimTheLastSlotOnce(t *testing.T) {
	gameStore := newTestStore(t)
	created, err := gameStore.Create(context.Background(), Actor{ID: "alice"}, CreateRequest{
		Players:          []engine.PlayerInit{{Name: "Alice"}, {Name: "Bob"}},
		StrictMembership: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	secret, err := gameStore.CreateInvitation(context.Background(), Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	var wait sync.WaitGroup
	errorsCh := make(chan error, 2)
	for _, uid := range []string{"bob", "carol"} {
		uid := uid
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, joinErr := gameStore.Join(context.Background(), Actor{ID: uid}, created.ID, secret.Code)
			errorsCh <- joinErr
		}()
	}
	wait.Wait()
	close(errorsCh)
	successes := 0
	full := 0
	for err := range errorsCh {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrGameFull):
			full++
		default:
			t.Fatalf("concurrent join error = %v", err)
		}
	}
	if successes != 1 || full != 1 {
		t.Fatalf("join outcomes = successes %d/full %d, want 1/1", successes, full)
	}
	memberships, err := gameStore.ListMemberships(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	if len(memberships) != 2 {
		t.Fatalf("memberships = %#v, want two", memberships)
	}
}

func TestMemoryStorePropagatesProfileRenamesToGameViews(t *testing.T) {
	gameStore := newTestStore(t)
	ctx := context.Background()
	if _, err := gameStore.EnsureProfile(ctx, Actor{ID: "alice", Email: "alice@example.com"}); err != nil {
		t.Fatalf("ensure profile: %v", err)
	}
	if _, err := gameStore.UpdateProfile(ctx, "alice", "Alice"); err != nil {
		t.Fatalf("set profile name: %v", err)
	}
	created, err := gameStore.Create(ctx, Actor{ID: "alice"}, CreateRequest{
		Players:          []engine.PlayerInit{{Name: "Ignored"}, {Name: "Other"}},
		StrictMembership: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := gameStore.UpdateProfile(ctx, "alice", "Alicia"); err != nil {
		t.Fatalf("rename profile: %v", err)
	}
	snapshot, err := gameStore.Get(ctx, Actor{ID: "alice"}, created.ID)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	if snapshot.Players[0].Name != "Alicia" || snapshot.State.Players[0].Name != "Alicia" {
		t.Fatalf("renamed player = slot %q/state %q", snapshot.Players[0].Name, snapshot.State.Players[0].Name)
	}
}
