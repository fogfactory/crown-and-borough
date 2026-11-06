package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// chatStore is a game store that can also persist chat messages, like the
// Firestore store does.
type chatStore struct {
	*store.MemoryStore
	*store.MemoryChatBackend
}

func newChatTestServer(t *testing.T, development bool) *http.ServeMux {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatal(err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := assetgen.LoadRules("../../assets", balance)
	if err != nil {
		t.Fatal(err)
	}
	memory := store.NewMemoryStoreWithOptions(balance, assets, store.MemoryStoreOptions{PrivacyTracker: api.TrackTurnPrivacy})
	return newApplicationServer(serverOptions{
		Rules:       rules,
		Balance:     balance,
		Store:       chatStore{memory, store.NewMemoryChatBackend()},
		Development: development,
	})
}

func TestChatRoutesAreDisabledInHotseatAndMountedOnline(t *testing.T) {
	const path = "/api/games/unknown/messages"
	hotseat := serve(t, newChatTestServer(t, true), http.MethodPost, path, `{"channel":"public","body":"hi"}`)
	if hotseat.Code != http.StatusNotFound || !strings.Contains(hotseat.Body.String(), "chat_disabled") {
		t.Fatalf("hotseat = %d: %s", hotseat.Code, hotseat.Body.String())
	}
	// Online, the route is mounted: it reaches authentication instead.
	online := serve(t, newChatTestServer(t, false), http.MethodPost, path, `{"channel":"public","body":"hi"}`)
	if online.Code != http.StatusUnauthorized {
		t.Fatalf("online = %d: %s", online.Code, online.Body.String())
	}
}
