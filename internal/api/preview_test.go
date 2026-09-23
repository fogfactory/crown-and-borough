package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func newPreviewTestHandler(t *testing.T) (http.Handler, store.GameStore) {
	t.Helper()
	gameStore, rules := newGamesTestStore(t)
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	return NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{
		Actor:   DevActorResolver("P1"),
		Balance: balance,
	}), gameStore
}

func TestOrdersPreviewReturnsParsedChainsAndLocalizedErrors(t *testing.T) {
	handler, gameStore := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "P1", `{"name":"Preview","seed":"preview-api","players":["One","Two"]}`)
	snapshot, err := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	noble := snapshot.State.Nobles[0]
	text := strings.ReplaceAll(noble.Code+"\nH "+string(noble.LocationID)+"\nNOT AN ORDER", "\n", `\n`)
	path := "/api/games/" + string(game.ID) + "/orders/preview?player=P1&lang=fr"

	response := requestGames(t, handler, http.MethodPost, path, `{"chains":[{"noble":"`+noble.Code+`","text":"`+text+`"}],"winter":[]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("preview = %d: %s", response.Code, response.Body.String())
	}
	var preview OrdersPreviewView
	if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if len(preview.Chains) != 1 || len(preview.Chains[0].Orders) != 1 || preview.Chains[0].Orders[0].Type != "hold" {
		t.Fatalf("chains = %#v, want the hold line", preview.Chains)
	}
	if len(preview.Errors) != 1 || preview.Errors[0].Line != 3 || preview.Errors[0].Message == "" {
		t.Fatalf("errors = %#v, want one localized error on line 3", preview.Errors)
	}
	english := requestGames(t, handler, http.MethodPost, strings.Replace(path, "lang=fr", "lang=en", 1), `{"chains":[{"noble":"`+noble.Code+`","text":"`+text+`"}],"winter":[]}`)
	var englishPreview OrdersPreviewView
	if err := json.Unmarshal(english.Body.Bytes(), &englishPreview); err != nil || len(englishPreview.Errors) != 1 {
		t.Fatalf("english preview = %s (%v)", english.Body.String(), err)
	}
	if englishPreview.Errors[0].Message == preview.Errors[0].Message {
		t.Errorf("preview error %q is not localized", preview.Errors[0].Message)
	}
}

func TestOrdersPreviewNeverChangesTheGame(t *testing.T) {
	handler, gameStore := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "P1", `{"name":"Preview","seed":"preview-pure","players":["One","Two"]}`)
	before, err := gameStore.Get(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("load game: %v", err)
	}
	response := requestGames(t, handler, http.MethodPost, "/api/games/"+string(game.ID)+"/orders/preview?player=P1", `{"chains":[],"winter":[]}`)
	if response.Code != http.StatusOK {
		t.Fatalf("preview = %d: %s", response.Code, response.Body.String())
	}
	after, err := gameStore.Get(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("reload game: %v", err)
	}
	if after.Revision != before.Revision || len(after.Submissions) != 0 {
		t.Fatalf("preview changed the game: revision %d -> %d, submissions %v", before.Revision, after.Revision, after.Submissions)
	}
}

func TestOrdersPreviewRejectsNonPlayersAndOtherMethods(t *testing.T) {
	handler, _ := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "host", `{"name":"Preview","seed":"preview-access","players":2,"spectate":true}`)
	path := "/api/games/" + string(game.ID) + "/orders/preview"

	if response := requestGames(t, handler, http.MethodPost, path+"?player=host", `{"chains":[]}`); response.Code != http.StatusForbidden {
		t.Errorf("spectator preview = %d, want 403: %s", response.Code, response.Body.String())
	}
	if response := requestGames(t, handler, http.MethodGet, path+"?player=P1", ""); response.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET preview = %d, want 405", response.Code)
	}
	if response := requestGames(t, handler, http.MethodPost, "/api/games/"+string(game.ID)+"/orders/other?player=P1", `{}`); response.Code != http.StatusNotFound {
		t.Errorf("unknown orders subresource = %d, want 404", response.Code)
	}
}

func TestSubmittedOrdersIncludeTheServerPreview(t *testing.T) {
	handler, gameStore := newPreviewTestHandler(t)
	game := createGameHTTP(t, handler, "host", `{"name":"Preview","seed":"preview-spectator","players":2,"spectate":true}`)
	snapshot, err := gameStore.State(context.Background(), store.Actor{ID: "P1", Development: true}, game.ID)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	noble := snapshot.State.Nobles[0]
	text := strings.ReplaceAll(noble.Code+"\nH "+string(noble.LocationID), "\n", `\n`)
	if submit := requestGames(t, handler, http.MethodPost, "/api/games/"+string(game.ID)+"/orders?player=P1", `{"chains":[{"noble":"`+noble.Code+`","text":"`+text+`"}],"winter":[]}`); submit.Code != http.StatusOK {
		t.Fatalf("submit = %d: %s", submit.Code, submit.Body.String())
	}

	response := requestGames(t, handler, http.MethodGet, "/api/games/"+string(game.ID)+"/submitted-orders?player=host", "")
	if response.Code != http.StatusOK {
		t.Fatalf("submitted orders = %d: %s", response.Code, response.Body.String())
	}
	var submitted submittedOrdersView
	if err := json.Unmarshal(response.Body.Bytes(), &submitted); err != nil {
		t.Fatalf("decode submitted orders: %v", err)
	}
	if len(submitted.Submissions) != 1 {
		t.Fatalf("submissions = %#v, want P1 only", submitted.Submissions)
	}
	chains := submitted.Submissions[0].Preview.Chains
	if len(chains) != 1 || len(chains[0].Orders) != 1 || chains[0].Orders[0].Type != "hold" {
		t.Fatalf("P1 preview chains = %#v, want the submitted hold", chains)
	}
}
