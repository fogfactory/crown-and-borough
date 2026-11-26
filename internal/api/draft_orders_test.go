package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

type draftTestGame struct {
	handler   http.Handler
	id        string
	chainBody string
	chainText string
}

func newDraftTestGame(t *testing.T) draftTestGame {
	t.Helper()
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, DevActorResolver("P1"))
	game := createGameHTTP(t, handler, "alice", `{"name":"Draft game","seed":"draft","players":["Alice","P2"]}`)
	response := requestGames(t, handler, http.MethodGet, "/api/games/"+string(game.ID)+"/state?player=alice", "")
	var state StateView
	if err := json.Unmarshal(response.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	noble := state.Nobles[0]
	text := string(noble.Code) + "\nH " + string(noble.Location)
	encoded, err := json.Marshal(map[string]any{"chains": []map[string]any{{"noble": noble.Code, "text": text}}, "winter": []any{}})
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	return draftTestGame{handler: handler, id: string(game.ID), chainBody: string(encoded), chainText: text}
}

func (g draftTestGame) post(t *testing.T, player, body string) *httptest.ResponseRecorder {
	t.Helper()
	return requestGames(t, g.handler, http.MethodPost, "/api/games/"+g.id+"/orders?player="+player, body)
}

func (g draftTestGame) withStatus(status string) string {
	return strings.Replace(g.chainBody, "{", `{"status":"`+status+`",`, 1)
}

func (g draftTestGame) mine(t *testing.T, player string) mySubmissionView {
	t.Helper()
	response := requestGames(t, g.handler, http.MethodGet, "/api/games/"+g.id+"/my-submission?player="+player, "")
	if response.Code != http.StatusOK {
		t.Fatalf("my-submission %s = %d: %s", player, response.Code, response.Body.String())
	}
	var view mySubmissionView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode my-submission: %v", err)
	}
	return view
}

// detail returns the public game projection as player sees it, with its raw body.
func (g draftTestGame) detail(t *testing.T, player string) (gameDetailView, string) {
	t.Helper()
	response := requestGames(t, g.handler, http.MethodGet, "/api/games/"+g.id+"?player="+player, "")
	if response.Code != http.StatusOK {
		t.Fatalf("game %s = %d: %s", player, response.Code, response.Body.String())
	}
	var view gameDetailView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode game: %v", err)
	}
	return view, response.Body.String()
}

func TestDraftIsStoredAsIsAndVisibleOnlyToItsAuthor(t *testing.T) {
	g := newDraftTestGame(t)

	if response := g.post(t, "alice", g.withStatus("draft")); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"draft"`) {
		t.Fatalf("draft = %d: %s", response.Code, response.Body.String())
	}
	mine := g.mine(t, "alice")
	if mine.Status != store.OrdersDraft || len(mine.Chains) != 1 || mine.Chains[0].Text != g.chainText {
		t.Fatalf("alice my-submission = %#v, want her draft", mine)
	}

	other := g.mine(t, "P2")
	if other.Status != store.OrdersPending || len(other.Chains) != 0 {
		t.Fatalf("P2 my-submission = %#v, want pending and empty", other)
	}
	for _, player := range []string{"alice", "P2"} {
		view, raw := g.detail(t, player)
		if view.Players[0].OrdersStatus != store.OrdersDraft || view.Players[0].Submitted || view.Players[1].OrdersStatus != store.OrdersPending {
			t.Fatalf("players seen by %s = %#v, want P1 draft (not submitted) and P2 pending", player, view.Players)
		}
		if strings.Contains(raw, g.chainText) {
			t.Fatalf("game projection for %s leaks the draft content: %s", player, raw)
		}
	}

	intruder := g.post(t, "intruder", g.withStatus("draft"))
	if intruder.Code != http.StatusForbidden {
		t.Fatalf("intruder draft = %d, want 403", intruder.Code)
	}
}

func TestDraftSkipsEngineValidationButSealingDoesNot(t *testing.T) {
	g := newDraftTestGame(t)
	invalid := `{"status":"draft","chains":[{"noble":"ZZZ","text":"ZZZ\nnot an order"}],"winter":[]}`
	if response := g.post(t, "alice", invalid); response.Code != http.StatusOK {
		t.Fatalf("invalid draft = %d: %s", response.Code, response.Body.String())
	}
	if mine := g.mine(t, "alice"); mine.Status != store.OrdersDraft || len(mine.Chains) != 1 {
		t.Fatalf("my-submission = %#v, want the invalid draft kept", mine)
	}
	sealed := g.post(t, "alice", strings.Replace(invalid, `"draft"`, `"submitted"`, 1))
	if sealed.Code == http.StatusOK {
		t.Fatalf("sealing an invalid order = %d, want a rejection", sealed.Code)
	}
	if mine := g.mine(t, "alice"); mine.Status != store.OrdersDraft {
		t.Fatalf("status after refused sealing = %q, want draft", mine.Status)
	}
}

func TestSealAndBreakTheSeal(t *testing.T) {
	g := newDraftTestGame(t)
	if response := g.post(t, "alice", g.chainBody); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("seal = %d: %s", response.Code, response.Body.String())
	}
	if mine := g.mine(t, "alice"); mine.Status != store.OrdersSubmitted {
		t.Fatalf("status = %q, want submitted", mine.Status)
	}
	view, _ := g.detail(t, "P2")
	if view.Players[0].OrdersStatus != store.OrdersSubmitted || !view.Players[0].Submitted {
		t.Fatalf("P2 sees %#v, want P1 sealed", view.Players[0])
	}
	if !view.Players[0].NameLocked {
		t.Fatal("sealing must lock the house name")
	}

	// Breaking the seal: same content as a draft, with the expected revision.
	stale := strings.Replace(g.withStatus("draft"), `{"status"`, `{"revision":999,"status"`, 1)
	if response := g.post(t, "alice", stale); response.Code != http.StatusConflict {
		t.Fatalf("seal break with a stale revision = %d, want 409: %s", response.Code, response.Body.String())
	}
	if mine := g.mine(t, "alice"); mine.Status != store.OrdersSubmitted {
		t.Fatalf("a refused seal break changed the status to %q", mine.Status)
	}
	view, _ = g.detail(t, "alice")
	body := strings.Replace(g.withStatus("draft"), `{"status"`, `{"revision":`+strconv.Itoa(int(view.Revision))+`,"status"`, 1)
	response := g.post(t, "alice", body)
	if response.Code != http.StatusOK {
		t.Fatalf("seal break = %d: %s", response.Code, response.Body.String())
	}
	var result gameOrdersResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "draft" || len(result.Submitted) != 0 || result.Resolved {
		t.Fatalf("seal break result = %#v", result)
	}
	mine := g.mine(t, "alice")
	if mine.Status != store.OrdersDraft || len(mine.Chains) != 1 || mine.Chains[0].Text != g.chainText {
		t.Fatalf("my-submission after seal break = %#v, want the intact draft", mine)
	}
	if view, _ = g.detail(t, "P2"); view.Players[0].OrdersStatus != store.OrdersDraft || view.Players[0].Submitted {
		t.Fatalf("P2 sees %#v, want P1 back to draft", view.Players[0])
	} else if view.Players[0].NameLocked {
		t.Fatal("breaking the first seal must give the house name back")
	}

	// A broken seal no longer counts as played: P2 sealing does not resolve.
	if response := g.post(t, "P2", `{"chains":[],"winter":[]}`); !strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("P2 seal = %d: %s", response.Code, response.Body.String())
	}
}

func TestOrdersStatusIsValidated(t *testing.T) {
	g := newDraftTestGame(t)
	if response := g.post(t, "alice", g.withStatus("pending")); response.Code != http.StatusBadRequest {
		t.Fatalf("status pending = %d, want 400", response.Code)
	}
	forced := strings.Replace(g.withStatus("draft"), `{"status"`, `{"force":true,"status"`, 1)
	if response := g.post(t, "alice", forced); response.Code != http.StatusBadRequest {
		t.Fatalf("forced draft = %d, want 400", response.Code)
	}
}

func TestHostResolveSealsValidDraftsAndIgnoresInvalidOnes(t *testing.T) {
	g := newDraftTestGame(t)
	g.post(t, "alice", g.withStatus("draft"))
	g.post(t, "P2", `{"status":"draft","chains":[{"noble":"ZZZ","text":"ZZZ\nnot an order"}],"winter":[]}`)

	response := requestGames(t, g.handler, http.MethodPost, "/api/games/"+g.id+"/resolve?player=alice", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"resolved"`) {
		t.Fatalf("resolve = %d: %s", response.Code, response.Body.String())
	}
	var result gameOrdersResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Submitted) != 1 || result.Submitted[0] != models.PlayerID("P1") {
		t.Fatalf("submitted = %v, want only P1 (valid draft sealed, invalid one ignored)", result.Submitted)
	}
	for _, player := range []string{"alice", "P2"} {
		if mine := g.mine(t, player); mine.Status != store.OrdersPending || mine.Turn != 2 {
			t.Fatalf("%s after resolution = %#v, want pending on turn 2", player, mine)
		}
	}
	if view, _ := g.detail(t, "P2"); view.Players[0].OrdersStatus != store.OrdersPending {
		t.Fatalf("drafts survived the turn: %#v", view.Players)
	}
}
