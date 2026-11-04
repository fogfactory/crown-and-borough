package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

type fakeIssues struct {
	calls  int
	title  string
	body   string
	labels []string
	err    error
}

func (f *fakeIssues) CreateIssue(_ context.Context, title, body string, labels []string) (string, error) {
	f.calls++
	f.title, f.body, f.labels = title, body, labels
	if f.err != nil {
		return "", f.err
	}
	return "https://github.com/example/repo/issues/7", nil
}

func bugsServer(issues IssueCreator, now func() time.Time) http.Handler {
	resolve := BearerActorResolver(func(token string) (store.Actor, error) {
		if token == "" || strings.HasPrefix(token, "bad") {
			return store.Actor{}, ErrUnauthorized
		}
		return store.Actor{ID: token + "-uid"}, nil
	})
	return NewBugsHandler(BugHandlerOptions{Actor: resolve, Issues: issues, Now: now})
}

func postBug(t *testing.T, handler http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/bugs", strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

const validBug = `{"title":"Map frozen","description":"It froze","context":{"version":"1.2.3"}}`

func TestBugsRequireAuthentication(t *testing.T) {
	issues := &fakeIssues{}
	response := postBug(t, bugsServer(issues, nil), "", validBug)
	if response.Code != http.StatusUnauthorized || issues.calls != 0 {
		t.Fatalf("anonymous = %d calls=%d", response.Code, issues.calls)
	}
}

func TestBugsDisabledWithoutIssueClient(t *testing.T) {
	handler := bugsServer((*GitHubIssueClient)(nil), nil)
	response := postBug(t, handler, "alice", validBug)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "bug_reports_disabled") {
		t.Fatalf("disabled = %d %s", response.Code, response.Body.String())
	}
	request := httptest.NewRequest(http.MethodGet, "/api/bugs", nil)
	request.Header.Set("Authorization", "Bearer alice")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), `"enabled":false`) {
		t.Fatalf("availability = %s", recorder.Body.String())
	}
}

func TestBugsCreateIssueAndRedactSecrets(t *testing.T) {
	issues := &fakeIssues{}
	body := `{"title":"Map frozen","description":"token Bearer abc.def.ghi leaked","context":{"note":"ghp_abcdefghijklmnopqrstuvwxyz0123456789"}}`
	response := postBug(t, bugsServer(issues, nil), "alice", body)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), "issues/7") {
		t.Fatalf("create = %d %s", response.Code, response.Body.String())
	}
	if issues.title != "Map frozen" || len(issues.labels) != 1 || issues.labels[0] != "bug" {
		t.Fatalf("issue = %+v", issues)
	}
	if strings.Contains(issues.body, "abc.def.ghi") || strings.Contains(issues.body, "ghp_") {
		t.Fatalf("secret reached the issue: %s", issues.body)
	}
	if !strings.Contains(issues.body, "Captured context") {
		t.Fatalf("missing context: %s", issues.body)
	}
}

func TestBugsMaskSensitiveContextKeys(t *testing.T) {
	issues := &fakeIssues{}
	body := `{"title":"x","description":"y","context":{"game":{"inviteCode":"ABC123","list":[{"password":"hunter2"}],"ok":"visible"}}}`
	if response := postBug(t, bugsServer(issues, nil), "alice", body); response.Code != http.StatusCreated {
		t.Fatalf("create = %d", response.Code)
	}
	if strings.Contains(issues.body, "ABC123") || strings.Contains(issues.body, "hunter2") || !strings.Contains(issues.body, "visible") {
		t.Fatalf("body = %s", issues.body)
	}
}

func TestBugsValidationAndSizeLimits(t *testing.T) {
	handler := bugsServer(&fakeIssues{}, nil)
	for name, body := range map[string]string{
		"empty title": `{"title":" ","description":"x"}`,
		"no desc":     `{"title":"x","description":""}`,
		"unknown":     `{"title":"x","description":"y","extra":1}`,
		"long title":  `{"title":"` + strings.Repeat("a", 121) + `","description":"y"}`,
		"long desc":   `{"title":"x","description":"` + strings.Repeat("a", 4001) + `"}`,
		"not json":    `nope`,
	} {
		if response := postBug(t, handler, "alice", body); response.Code != http.StatusBadRequest {
			t.Errorf("%s = %d", name, response.Code)
		}
	}
	huge := `{"title":"x","description":"y","context":"` + strings.Repeat("a", bugRequestLimit) + `"}`
	if response := postBug(t, handler, "alice", huge); response.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("huge = %d", response.Code)
	}
}

func TestBugsBodyIsTruncatedForGitHub(t *testing.T) {
	issues := &fakeIssues{}
	context := `"` + strings.Repeat("é", 100000) + `"`
	body := `{"title":"x","description":"y","context":` + context + `}`
	handler := bugsServer(issues, nil)
	// 100k runes is ~200KB: under the request limit, over the GitHub limit.
	if response := postBug(t, handler, "alice", body); response.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", response.Code, response.Body.String())
	}
	if len(issues.body) > 65536 || !strings.Contains(issues.body, "truncated") {
		t.Fatalf("body length %d", len(issues.body))
	}
}

func TestBugsRateLimitPerActor(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	issues := &fakeIssues{}
	handler := bugsServer(issues, func() time.Time { return now })
	for i := 0; i < bugActorLimit; i++ {
		if response := postBug(t, handler, "alice", validBug); response.Code != http.StatusCreated {
			t.Fatalf("report %d = %d", i, response.Code)
		}
	}
	if response := postBug(t, handler, "alice", validBug); response.Code != http.StatusTooManyRequests {
		t.Fatalf("over limit = %d", response.Code)
	}
	if response := postBug(t, handler, "bob", validBug); response.Code != http.StatusCreated {
		t.Fatalf("other actor = %d", response.Code)
	}
	now = now.Add(bugActorWindow + time.Second)
	if response := postBug(t, handler, "alice", validBug); response.Code != http.StatusCreated {
		t.Fatalf("after window = %d", response.Code)
	}
}

func TestBugsUpstreamFailureHidesDetails(t *testing.T) {
	issues := &fakeIssues{err: errors.New("secret upstream detail")}
	response := postBug(t, bugsServer(issues, nil), "alice", validBug)
	if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "secret upstream") {
		t.Fatalf("failure = %d %s", response.Code, response.Body.String())
	}
}

func TestGitHubIssueClientSendsTokenAndParsesURL(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"html_url":"https://github.com/o/r/issues/9"}`))
	}))
	defer upstream.Close()
	client := NewGitHubIssueClient(" test-token ", "o/r")
	client.BaseURL = upstream.URL
	url, err := client.CreateIssue(context.Background(), "t", "b", []string{"bug"})
	if err != nil || url != "https://github.com/o/r/issues/9" {
		t.Fatalf("url=%q err=%v", url, err)
	}
	if gotAuth != "Bearer test-token" || gotPath != "/repos/o/r/issues" || gotBody["title"] != "t" {
		t.Fatalf("request auth=%q path=%q body=%v", gotAuth, gotPath, gotBody)
	}
	if NewGitHubIssueClient("  ", "o/r") != nil {
		t.Fatal("empty token must disable the client")
	}
}

func TestGitHubIssueClientRejectsErrors(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer upstream.Close()
	client := NewGitHubIssueClient("t", "o/r")
	client.BaseURL = upstream.URL
	if _, err := client.CreateIssue(context.Background(), "t", "b", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestBugsAttachServerSideGameProjection(t *testing.T) {
	issues := &fakeIssues{}
	handler := NewBugsHandler(BugHandlerOptions{
		Actor:  BearerActorResolver(func(token string) (store.Actor, error) { return store.Actor{ID: token}, nil }),
		Issues: issues,
		State: func(_ context.Context, actor store.Actor, id store.GameID) (any, error) {
			if id != "g1" {
				return nil, errors.New("not found")
			}
			return map[string]any{"viewer": actor.ID, "inviteCode": "SECRET1"}, nil
		},
	})
	body := `{"title":"x","description":"y","gameId":"g1","context":{"version":"1"},"game":{"forged":true}}`
	if response := postBug(t, handler, "alice", strings.Replace(body, `,"game":{"forged":true}`, "", 1)); response.Code != http.StatusCreated {
		t.Fatalf("create = %d", response.Code)
	}
	if !strings.Contains(issues.body, `"viewer": "alice"`) || strings.Contains(issues.body, "SECRET1") {
		t.Fatalf("body = %s", issues.body)
	}
	// A client-supplied game blob is rejected, not published.
	if response := postBug(t, handler, "alice", body); response.Code != http.StatusBadRequest {
		t.Fatalf("forged game = %d", response.Code)
	}
	postBug(t, handler, "bob", `{"title":"x","description":"y","gameId":"other"}`)
	if !strings.Contains(issues.body, "unavailable") {
		t.Fatalf("body = %s", issues.body)
	}
}
