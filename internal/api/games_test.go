package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestGamesHandlerCreatesListsAndResolvesIndependentGames(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, DevActorResolver("P1"))

	first := createGameHTTP(t, handler, "alice", `{"name":"Alice game","seed":"shared","players":["Alice","P2"]}`)
	second := createGameHTTP(t, handler, "bob", `{"name":"Bob game","seed":"shared","players":["Bob","P3"]}`)
	if first.ID == second.ID {
		t.Fatal("two game creations returned the same id")
	}

	aliceList := requestGames(t, handler, http.MethodGet, "/api/games?player=alice", "")
	if aliceList.Code != http.StatusOK {
		t.Fatalf("Alice list = %d: %s", aliceList.Code, aliceList.Body.String())
	}
	var aliceGames []gameListView
	if err := json.Unmarshal(aliceList.Body.Bytes(), &aliceGames); err != nil {
		t.Fatalf("decode Alice list: %v", err)
	}
	if len(aliceGames) != 1 || aliceGames[0].ID != first.ID {
		t.Fatalf("Alice games = %#v, want only %s", aliceGames, first.ID)
	}

	bobList := requestGames(t, handler, http.MethodGet, "/api/games?player=bob", "")
	if bobList.Code != http.StatusOK {
		t.Fatalf("Bob list = %d: %s", bobList.Code, bobList.Body.String())
	}
	var bobGames []gameListView
	if err := json.Unmarshal(bobList.Body.Bytes(), &bobGames); err != nil {
		t.Fatalf("decode Bob list: %v", err)
	}
	if len(bobGames) != 1 || bobGames[0].ID != second.ID {
		t.Fatalf("Bob games = %#v, want only %s", bobGames, second.ID)
	}

	pending := requestGames(t, handler, http.MethodPost, "/api/games/"+string(first.ID)+"/orders?player=alice", `{"chains":[],"winter":[]}`)
	if pending.Code != http.StatusOK || !strings.Contains(pending.Body.String(), `"status":"pending"`) {
		t.Fatalf("Alice pending submission = %d: %s", pending.Code, pending.Body.String())
	}
	resolved := requestGames(t, handler, http.MethodPost, "/api/games/"+string(first.ID)+"/orders?player=P2", `{"chains":[],"winter":[]}`)
	if resolved.Code != http.StatusOK || !strings.Contains(resolved.Body.String(), `"status":"resolved"`) {
		t.Fatalf("P2 resolution = %d: %s", resolved.Code, resolved.Body.String())
	}
	if strings.Contains(resolved.Body.String(), string(second.ID)) {
		t.Fatal("first game's response exposed the other game id")
	}

	secondState := requestGames(t, handler, http.MethodGet, "/api/games/"+string(second.ID)+"/state?player=bob", "")
	if secondState.Code != http.StatusOK {
		t.Fatalf("second state = %d: %s", secondState.Code, secondState.Body.String())
	}
	var state StateView
	if err := json.Unmarshal(secondState.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode second state: %v", err)
	}
	if state.Turn != 1 {
		t.Fatalf("second game turn = %d, want 1", state.Turn)
	}
	if !strings.Contains(secondState.Body.String(), `"revision":1`) {
		t.Fatalf("second state omitted revision: %s", secondState.Body.String())
	}

	reports := requestGames(t, handler, http.MethodGet, "/api/games/"+string(first.ID)+"/reports?player=alice", "")
	if reports.Code != http.StatusOK || !strings.Contains(reports.Body.String(), `"index":0`) {
		t.Fatalf("reports = %d: %s", reports.Code, reports.Body.String())
	}
	report := requestGames(t, handler, http.MethodGet, "/api/games/"+string(first.ID)+"/reports/0?player=alice", "")
	if report.Code != http.StatusOK || !strings.Contains(report.Body.String(), `"turn":1`) {
		t.Fatalf("report = %d: %s", report.Code, report.Body.String())
	}
}

func TestGamesHandlerCreatesGameWithConfiguredYearsAndDefaults(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, DevActorResolver("P1"))

	configured := createGameHTTP(t, handler, "P1", `{"name":"Long game","players":2,"years":12}`)
	if configured.YearCount != 12 || len(configured.Scores) != 2 {
		t.Fatalf("configured game = years %d scores %#v, want 12 and two scores", configured.YearCount, configured.Scores)
	}
	defaulted := createGameHTTP(t, handler, "P1", `{"name":"Default game","players":2}`)
	if defaulted.YearCount != models.DefaultGameYears {
		t.Fatalf("default game years = %d, want %d", defaulted.YearCount, models.DefaultGameYears)
	}
}

func TestGamesHandlerRejectsInvalidYearCount(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, DevActorResolver("P1"))
	for _, years := range []int{-1, 51} {
		response := requestGames(t, handler, http.MethodPost, "/api/games?player=P1", `{"name":"Invalid","players":2,"years":`+strconv.Itoa(years)+`}`)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"invalid_years"`) {
			t.Fatalf("years %d response = %d: %s", years, response.Code, response.Body.String())
		}
	}
}

func TestGamesHandlerUsesServerActorAndPrivateStateProjection(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, DevActorResolver("P1"))
	created := createGameHTTP(t, handler, "P1", `{"name":"Private game","seed":"private","players":["One","Two"]}`)

	stateResponse := requestGames(t, handler, http.MethodGet, "/api/games/"+string(created.ID)+"/state?player=P1", "")
	if stateResponse.Code != http.StatusOK {
		t.Fatalf("initial state = %d: %s", stateResponse.Code, stateResponse.Body.String())
	}
	var initial StateView
	if err := json.Unmarshal(stateResponse.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode initial state: %v", err)
	}
	var noble NobleView
	for _, candidate := range initial.Nobles {
		if candidate.Owner == "P1" && candidate.Status == models.NobleStatusFree {
			noble = candidate
			break
		}
	}
	if noble.Code == "" {
		t.Fatal("P1 has no free noble")
	}
	chainText := string(noble.Code) + "\nH " + string(noble.Location) + "\nH " + string(noble.Location)
	encodedChain := strings.ReplaceAll(chainText, "\n", `\n`)
	first := requestGames(t, handler, http.MethodPost, "/api/games/"+string(created.ID)+"/orders?player=P1", `{"chains":[{"noble":"`+string(noble.Code)+`","text":"`+encodedChain+`"}],"winter":[]}`)
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"status":"pending"`) {
		t.Fatalf("P1 chain submission = %d: %s", first.Code, first.Body.String())
	}
	second := requestGames(t, handler, http.MethodPost, "/api/games/"+string(created.ID)+"/orders?player=P2", `{"chains":[],"winter":[]}`)
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"status":"resolved"`) {
		t.Fatalf("P2 submission = %d: %s", second.Code, second.Body.String())
	}

	p2State := requestGames(t, handler, http.MethodGet, "/api/games/"+string(created.ID)+"/state?player=P2", "")
	if p2State.Code != http.StatusOK {
		t.Fatalf("P2 private state = %d: %s", p2State.Code, p2State.Body.String())
	}
	if !strings.Contains(p2State.Body.String(), `"visibility":"hidden"`) {
		t.Fatalf("P2 state did not hide P1's chain: %s", p2State.Body.String())
	}
	p1State := requestGames(t, handler, http.MethodGet, "/api/games/"+string(created.ID)+"/state?player=P1", "")
	if p1State.Code != http.StatusOK || !strings.Contains(p1State.Body.String(), `"visibility":"known"`) {
		t.Fatalf("P1 state did not expose its known chain: %d: %s", p1State.Code, p1State.Body.String())
	}
}

func TestGamesHandlerRejectsUntrustedIdentityInBearerMode(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandler(gameStore, rules, BearerActorResolver(func(string) (store.Actor, error) {
		return store.Actor{ID: "verified-user"}, nil
	}))

	withoutToken := requestGames(t, handler, http.MethodPost, "/api/games?player=P1", `{"name":"Nope","players":["One","Two"]}`)
	if withoutToken.Code != http.StatusUnauthorized {
		t.Fatalf("without token = %d: %s", withoutToken.Code, withoutToken.Body.String())
	}
	withToken := requestGamesWithHeaders(t, handler, http.MethodPost, "/api/games?player=P1", `{"name":"Verified","players":["One","Two"]}`, map[string]string{"Authorization": "Bearer test-token"})
	if withToken.Code != http.StatusCreated {
		t.Fatalf("with token = %d: %s", withToken.Code, withToken.Body.String())
	}
}

func TestGamesHandlerCreatorGateAllowsOnlyAuthorizedAccounts(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	actors := map[string]store.Actor{
		"allowed-token": {ID: "allowed-uid", Email: "admin@mail.com"},
		"blocked-token": {ID: "blocked-uid", Email: "blocked@example.com"},
	}
	resolve := BearerActorResolver(func(token string) (store.Actor, error) {
		actor, ok := actors[token]
		if !ok {
			return store.Actor{}, ErrUnauthorized
		}
		return actor, nil
	})
	handler := NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{
		Actor:       resolve,
		CreatorGate: NewAnyCreatorGate(FirebaseCreatorGate{}, NewEmailCreatorGate("admin@mail.com")),
	})

	allowed := requestGamesWithHeaders(t, handler, http.MethodPost, "/api/games", `{"name":"Allowed","players":2}`, map[string]string{
		"Authorization": "Bearer allowed-token",
	})
	if allowed.Code != http.StatusCreated {
		t.Fatalf("allowlisted create = %d: %s", allowed.Code, allowed.Body.String())
	}

	blocked := requestGamesWithHeaders(t, handler, http.MethodPost, "/api/games", `{"name":"Blocked","players":2}`, map[string]string{
		"Authorization": "Bearer blocked-token",
	})
	if blocked.Code != http.StatusForbidden || !strings.Contains(blocked.Body.String(), `"creator_not_allowed"`) {
		t.Fatalf("blocked create = %d: %s", blocked.Code, blocked.Body.String())
	}
}

func TestDevelopmentResolverKeepsLegacySlotAccessWhenStrictSlotsAreEnabled(t *testing.T) {
	gameStore, rules := newGamesTestStore(t)
	handler := NewGamesHandlerWithOptions(gameStore, rules, GamesHandlerOptions{
		Actor:            DevActorResolver("P1"),
		StrictMembership: true,
	})
	created := createGameHTTP(t, handler, "P1", `{"name":"dev game","players":2}`)
	response := requestGames(t, handler, http.MethodGet, "/api/games/"+string(created.ID)+"/state?player=P2", "")
	if response.Code != http.StatusOK {
		t.Fatalf("development P2 state = %d: %s", response.Code, response.Body.String())
	}
}

func newGamesTestStore(t *testing.T) (store.GameStore, assetgen.Rules) {
	t.Helper()
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("load balance: %v", err)
	}
	rules, err := assetgen.LoadRules("../../assets")
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	return store.NewMemoryStoreWithOptions(balance, assets, store.MemoryStoreOptions{PrivacyTracker: TrackTurnPrivacy}), rules
}

func createGameHTTP(t *testing.T, handler http.Handler, actor, body string) gameDetailView {
	t.Helper()
	recorder := requestGames(t, handler, http.MethodPost, "/api/games?player="+actor, body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create game for %s = %d: %s", actor, recorder.Code, recorder.Body.String())
	}
	var game gameDetailView
	if err := json.Unmarshal(recorder.Body.Bytes(), &game); err != nil {
		t.Fatalf("decode created game: %v", err)
	}
	if game.ID == "" {
		t.Fatal("created game has no id")
	}
	return game
}

func requestGames(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return requestGamesWithHeaders(t, handler, method, path, body, nil)
}

func requestGamesWithHeaders(t *testing.T, handler http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}
