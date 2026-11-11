//go:build integration

package firestorestore

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestFirestoreChatBackend(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	backend := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{Client: client})
	gameID := store.GameID("chat-" + time.Now().Format("150405.000000"))
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	message := func(channel string, at time.Time, body string) store.ChatMessage {
		return store.ChatMessage{
			Channel:       channel,
			AuthorHouseID: models.PlayerID("P1"),
			Body:          body,
			CreatedAt:     at,
			Turn:          store.ChatTurn{Number: 3, Season: models.SeasonAutumn, Year: 1},
		}
	}
	first, err := backend.AppendChatMessage(ctx, gameID, "alice", message("public", base, "one"), nil, time.Second)
	if err != nil || first.ID == "" {
		t.Fatalf("first message: %v %#v", err, first)
	}
	if _, err := backend.AppendChatMessage(ctx, gameID, "alice", message("public", base.Add(500*time.Millisecond), "too soon"), nil, time.Second); !errors.Is(err, store.ErrChatRateLimited) {
		t.Fatalf("rate limit = %v", err)
	}
	for i := 1; i <= 3; i++ {
		at := base.Add(time.Duration(i) * 2 * time.Second)
		if _, err := backend.AppendChatMessage(ctx, gameID, "alice", message("P1_P2", at, string(rune('a'+i))), []string{"alice", "bob"}, time.Second); err != nil {
			t.Fatalf("private message %d: %v", i, err)
		}
	}

	page, err := backend.ListChatMessages(ctx, gameID, "P1_P2", store.ChatCursor{}, 2)
	if err != nil || len(page) != 2 || page[0].Body != "d" || page[1].Body != "c" {
		t.Fatalf("first page = %#v, %v", page, err)
	}
	if page[0].Turn.Season != models.SeasonAutumn || page[0].AuthorHouseID != "P1" {
		t.Errorf("round trip lost fields: %#v", page[0])
	}
	next, err := backend.ListChatMessages(ctx, gameID, "P1_P2", page[1].Cursor(), 2)
	if err != nil || len(next) != 1 || next[0].Body != "b" {
		t.Fatalf("second page = %#v, %v", next, err)
	}

	if _, err := backend.SetChatReadCursor(ctx, gameID, "alice", store.ChatReadCursor{Channel: "public", LastReadAt: base.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	cursor, err := backend.SetChatReadCursor(ctx, gameID, "alice", store.ChatReadCursor{Channel: "P1_P2", LastReadAt: base})
	if err != nil {
		t.Fatal(err)
	}
	older, err := backend.SetChatReadCursor(ctx, gameID, "alice", store.ChatReadCursor{Channel: "P1_P2", LastReadAt: base.Add(-time.Hour)})
	if err != nil || !older.LastReadAt.Equal(cursor.LastReadAt) {
		t.Fatalf("cursor moved back: %v, %v", older, err)
	}
	snapshot, err := client.Collection("games").Doc(string(gameID)).Collection(chatReadsCollection).Doc("alice").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var reads chatReadsDocument
	if err := snapshot.DataTo(&reads); err != nil || len(reads.Channels) != 2 {
		t.Fatalf("cursors = %#v, %v (a second channel must not erase the first)", reads, err)
	}
}

func TestFirestoreNotificationState(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	backend := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{Client: client})
	suffix := time.Now().Format("150405.000000")
	gameID := store.GameID("notify-" + suffix)
	uid := "alice-" + suffix
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Activity.
	if last, err := backend.LastActivity(ctx, uid); err != nil || !last.IsZero() {
		t.Fatalf("activity before touch = %v, %v", last, err)
	}
	if err := backend.TouchActivity(ctx, uid, base); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if last, err := backend.LastActivity(ctx, uid); err != nil || !last.Equal(base) {
		t.Fatalf("activity = %v, %v", last, err)
	}

	// Message email claim: once, then again after a read cursor moves.
	claim := func(at time.Time) bool {
		t.Helper()
		ok, err := backend.ClaimMessageEmail(ctx, gameID, uid, at)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		return ok
	}
	if !claim(base) || claim(base.Add(time.Minute)) {
		t.Fatal("the first claim must win and the second must lose")
	}
	if _, err := backend.SetChatReadCursor(ctx, gameID, uid, store.ChatReadCursor{Channel: "public", LastReadAt: base.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if !claim(base.Add(3 * time.Minute)) {
		t.Fatal("a claim after a read must win")
	}
}

// A profile written before the email notifications existed has no
// notifications field: it reads as all off and can be updated.
func TestFirestoreLegacyProfileHasNotificationsOff(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not configured")
	}
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		projectID = "crown-and-borough-integration"
	}
	ctx := context.Background()
	client, err := cloudfirestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("create emulator client: %v", err)
	}
	defer client.Close()
	backend := NewWithClient(assetgen.Balance{}, assetgen.Assets{}, Options{Client: client})
	uid := "legacy-" + time.Now().Format("150405.000000")
	now := time.Now().UTC()
	if _, err := client.Collection("players").Doc(uid).Set(ctx, map[string]any{
		"schemaVersion": schemaVersion, "uid": uid, "email": uid + "@example.com", "displayName": "Legacy",
		"createdAt": now, "updatedAt": now, "lastSeenVersion": "1.0.0", "hideReleaseNotes": false,
	}); err != nil {
		t.Fatal(err)
	}
	profile, err := backend.GetProfile(ctx, uid)
	if err != nil || profile.Notifications != (store.NotificationPreferences{}) {
		t.Fatalf("legacy profile = %+v, %v", profile, err)
	}
	on, french := true, "fr"
	profile, err = backend.UpdateNotifications(ctx, uid, store.NotificationsUpdate{NewTurn: &on, Language: &french})
	if err != nil || !profile.Notifications.NewTurn || profile.Notifications.Language != "fr" || profile.DisplayName != "Legacy" {
		t.Fatalf("updated profile = %+v, %v", profile, err)
	}
}
