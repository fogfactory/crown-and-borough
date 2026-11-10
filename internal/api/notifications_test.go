package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

type recordingNotifier struct {
	mu       sync.Mutex
	turns    []store.GameSnapshot
	messages []store.ChatPostResult
	// onCall runs inside the notifier, to observe the state of the response.
	onCall func()
}

func (n *recordingNotifier) TurnStarted(_ context.Context, snapshot store.GameSnapshot) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.turns = append(n.turns, snapshot)
	if n.onCall != nil {
		n.onCall()
	}
}

func (n *recordingNotifier) MessagePosted(_ context.Context, result store.ChatPostResult) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messages = append(n.messages, result)
	if n.onCall != nil {
		n.onCall()
	}
}

func newNotifyTestHandler(t *testing.T, notifier Notifier) (http.Handler, store.GameID) {
	t.Helper()
	env := newNotifiedChatTestEnv(t, true, notifier)
	return env.handler, env.gameID
}

func TestPostMessageNotifiesAfterTheCompleteResponse(t *testing.T) {
	notifier := &recordingNotifier{}
	handler, id := newNotifyTestHandler(t, notifier)
	recorder := httptest.NewRecorder()
	var flushedAtNotify bool
	notifier.onCall = func() {
		flushedAtNotify = recorder.Flushed && recorder.Body.Len() > 0
	}
	request := httptest.NewRequest(http.MethodPost, "/api/games/"+string(id)+"/messages?player=alice", strings.NewReader(`{"channel":"P2","body":"hello"}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Connection"); got != "close" {
		t.Fatalf("Connection = %q, want close so the next request does not wait behind the notifications", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != strconv.Itoa(recorder.Body.Len()) {
		t.Fatalf("Content-Length = %q, body = %d bytes", got, recorder.Body.Len())
	}
	if len(notifier.messages) != 1 || !flushedAtNotify {
		t.Fatalf("notified %d times, response flushed before notification = %v", len(notifier.messages), flushedAtNotify)
	}
	result := notifier.messages[0]
	if result.GameName != "Chat game" || len(result.RecipientUIDs) != 1 || result.RecipientUIDs[0] != "bob" {
		t.Fatalf("notification = %+v", result)
	}
}

func TestRejectedMessageDoesNotNotify(t *testing.T) {
	notifier := &recordingNotifier{}
	handler, id := newNotifyTestHandler(t, notifier)
	recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/messages?player=alice", `{"channel":"P2","body":"  "}`)
	if recorder.Code != http.StatusBadRequest || len(notifier.messages) != 0 {
		t.Fatalf("status = %d, notified %d times", recorder.Code, len(notifier.messages))
	}
}

func TestForcedResolveNotifiesTheNewTurn(t *testing.T) {
	notifier := &recordingNotifier{}
	handler, id := newNotifyTestHandler(t, notifier)
	recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/resolve?player=alice", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("resolve = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Length"); got != strconv.Itoa(recorder.Body.Len()) {
		t.Fatalf("Content-Length = %q, body = %d bytes", got, recorder.Body.Len())
	}
	if len(notifier.turns) != 1 || notifier.turns[0].ID != id {
		t.Fatalf("turn notifications = %d", len(notifier.turns))
	}

	// A refused resolve (not the creator) notifies nobody.
	recorder = requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/resolve?player=bob", "")
	if recorder.Code == http.StatusOK || len(notifier.turns) != 1 {
		t.Fatalf("non-creator resolve = %d, notifications = %d", recorder.Code, len(notifier.turns))
	}
}

func TestSubmitThatDoesNotResolveDoesNotNotify(t *testing.T) {
	notifier := &recordingNotifier{}
	handler, id := newNotifyTestHandler(t, notifier)
	recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/orders?player=alice", `{"chains":[],"winter":[]}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("submit = %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(notifier.turns) != 0 {
		t.Fatalf("turn notification sent although two players have not submitted: %d", len(notifier.turns))
	}
}

func TestHandlersWorkWithoutNotifier(t *testing.T) {
	handler, id := newNotifyTestHandler(t, nil)
	if recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/messages?player=alice", `{"channel":"public","body":"hi"}`); recorder.Code != http.StatusCreated {
		t.Fatalf("post = %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder := requestGames(t, handler, http.MethodPost, "/api/games/"+string(id)+"/resolve?player=alice", ""); recorder.Code != http.StatusOK {
		t.Fatalf("resolve = %d: %s", recorder.Code, recorder.Body.String())
	}
}

type recordingActivity struct{ uids []string }

func (r *recordingActivity) Touch(_ context.Context, uid string) { r.uids = append(r.uids, uid) }

func TestWithActivityRecordsAuthenticatedRequestsOnly(t *testing.T) {
	recorder := &recordingActivity{}
	resolve := WithActivity(BearerActorResolver(func(token string) (store.Actor, error) {
		if token != "alice-token" {
			return store.Actor{}, ErrUnauthorized
		}
		return store.Actor{ID: "alice-uid"}, nil
	}), recorder)

	for _, token := range []string{"alice-token", "wrong", "alice-token"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		_, _ = resolve(request)
	}
	if len(recorder.uids) != 2 || recorder.uids[0] != "alice-uid" {
		t.Fatalf("touched = %v", recorder.uids)
	}
	if WithActivity(nil, recorder) != nil {
		t.Fatal("a nil resolver must stay nil")
	}
}
