package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// The title and description limits mirror TITLE_MAX and DESCRIPTION_MAX in
// web/src/components/BugReportDialog.tsx.
const (
	bugRequestLimit     = 256 << 10
	bugTitleMaxRunes    = 120
	bugDescriptionRunes = 4000
	// GitHub rejects issue bodies above 65536 characters.
	bugBodyMaxBytes = 60000
	bugActorLimit   = 3
	bugActorWindow  = 10 * time.Minute
	bugGlobalLimit  = 30
	bugGlobalWindow = time.Hour
)

// ErrBugReportsDisabled is returned when no GitHub token is configured.
var ErrBugReportsDisabled = errors.New("api: bug reports are disabled")

// IssueCreator files a GitHub issue and returns its public URL.
type IssueCreator interface {
	CreateIssue(ctx context.Context, title, body string, labels []string) (string, error)
}

// GitHubIssueClient creates issues through the GitHub REST API with a bot token.
type GitHubIssueClient struct {
	Token      string
	Repository string // owner/name
	BaseURL    string // defaults to https://api.github.com
	HTTPClient *http.Client
}

// NewGitHubIssueClient returns a client for the repository, or nil when the
// token is empty so the route reports itself as disabled.
func NewGitHubIssueClient(token, repository string) *GitHubIssueClient {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return &GitHubIssueClient{Token: token, Repository: repository}
}

func (c *GitHubIssueClient) CreateIssue(ctx context.Context, title, body string, labels []string) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	payload, err := json.Marshal(map[string]any{"title": title, "body": body, "labels": labels})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/repos/"+c.Repository+"/issues", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		// The error text never carries the token (it is only in a header).
		return "", fmt.Errorf("github request failed: %w", err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github responded %d", response.StatusCode)
	}
	var created struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(data, &created); err != nil || created.HTMLURL == "" {
		return "", errors.New("github response has no issue url")
	}
	return created.HTMLURL, nil
}

// PlayerStateFunc returns the projection of a game as the actor sees it.
type PlayerStateFunc func(ctx context.Context, actor store.Actor, id store.GameID) (any, error)

// BugHandlerOptions configures BugsHandler.
type BugHandlerOptions struct {
	Actor  ActorResolver
	Issues IssueCreator    // nil disables the route
	State  PlayerStateFunc // nil attaches no game projection
	Now    func() time.Time
}

// BugsHandler serves GET /api/bugs (availability) and POST /api/bugs. Its rate
// limits live in process memory: they are global only while the service runs a
// single instance (--max-instances 1 in the Cloud Run deployment).
type BugsHandler struct {
	actor  ActorResolver
	issues IssueCreator
	state  PlayerStateFunc
	now    func() time.Time

	mu     sync.Mutex
	byUser map[string][]time.Time
	global []time.Time
}

func NewBugsHandler(options BugHandlerOptions) *BugsHandler {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	issues := options.Issues
	// A typed nil pointer must also count as "not configured".
	if client, ok := issues.(*GitHubIssueClient); ok && client == nil {
		issues = nil
	}
	return &BugsHandler{actor: options.Actor, issues: issues, state: options.State, now: now, byUser: map[string][]time.Time{}}
}

type bugRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	// GameID names the game whose projection the server attaches: the
	// client never sends game state, so it cannot forge or leak it.
	GameID string `json:"gameId"`
	// Context is the client side capture (version, browser, API calls).
	Context json.RawMessage `json:"context"`
}

func (h *BugsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actor, err := actorFromRequest(r, h.actor)
	if err != nil {
		writeActorError(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, struct {
			Enabled bool `json:"enabled"`
		}{Enabled: h.issues != nil})
	case http.MethodPost:
		h.create(w, r, actor)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BugsHandler) create(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	if h.issues == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "bug_reports_disabled", "bug reports are not enabled on this server")
		return
	}
	var body bugRequest
	if err := decodeJSONBody(w, r, bugRequestLimit, &body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "bug_report_too_large", "the bug report is too large")
			return
		}
		writeAPIError(w, http.StatusBadRequest, "invalid_bug_report", err.Error())
		return
	}
	title := strings.TrimSpace(body.Title)
	description := strings.TrimSpace(body.Description)
	if title == "" || description == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_bug_report", "a title and a description are required")
		return
	}
	if utf8.RuneCountInString(title) > bugTitleMaxRunes || utf8.RuneCountInString(description) > bugDescriptionRunes {
		writeAPIError(w, http.StatusBadRequest, "invalid_bug_report", "the title or the description is too long")
		return
	}
	if !h.allow(actor.ID) {
		w.Header().Set("Retry-After", "600")
		writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "too many bug reports, try again later")
		return
	}
	issueURL, err := h.issues.CreateIssue(r.Context(), redactSecrets(title), buildBugBody(description, h.captureContext(r.Context(), actor, body)), []string{"bug"})
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "bug_report_failed", "the issue could not be created")
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		URL string `json:"url"`
	}{URL: issueURL})
}

// captureContext merges the client capture with the player's own projection of
// the game, read by the server. The result is untrusted until masked.
func (h *BugsHandler) captureContext(ctx context.Context, actor store.Actor, body bugRequest) json.RawMessage {
	captured := map[string]any{}
	if len(body.Context) > 0 {
		var client any
		if json.Unmarshal(body.Context, &client) == nil {
			captured["client"] = client
		}
	}
	if gameID := strings.TrimSpace(body.GameID); gameID != "" && h.state != nil {
		if game, err := h.state(ctx, actor, store.GameID(gameID)); err == nil {
			captured["game"] = game
		} else {
			captured["game"] = "unavailable"
		}
	}
	if len(captured) == 0 {
		return nil
	}
	raw, err := json.Marshal(captured)
	if err != nil {
		return nil
	}
	return raw
}

func (h *BugsHandler) allow(actorID string) bool {
	now := h.now()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.global = prune(h.global, now.Add(-bugGlobalWindow))
	for id, times := range h.byUser {
		if id != actorID && len(prune(times, now.Add(-bugActorWindow))) == 0 {
			delete(h.byUser, id)
		}
	}
	recent := prune(h.byUser[actorID], now.Add(-bugActorWindow))
	if len(recent) >= bugActorLimit || len(h.global) >= bugGlobalLimit {
		if len(recent) == 0 {
			delete(h.byUser, actorID)
		}
		return false
	}
	h.byUser[actorID] = append(recent, now)
	h.global = append(h.global, now)
	return true
}

func prune(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0]
	for _, at := range times {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	return kept
}

// Keep in sync with SECRET_IN_TEXT and SENSITIVE_KEY in web/src/lib/bug-report.ts.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]*`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`),
}

var sensitiveKey = regexp.MustCompile(`(?i)token|authorization|password|secret|api[-_]?key|cookie|invite|credential`)

// maskContext decodes the client-supplied context, masks sensitive keys and
// token-like strings at any depth, and returns it indented. The server cannot
// trust the client to have masked anything.
func maskContext(raw json.RawMessage) (string, bool) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	masked, err := json.MarshalIndent(maskValue(value), "", "  ")
	if err != nil {
		return "", false
	}
	return string(masked), true
}

func maskValue(value any) any {
	switch typed := value.(type) {
	case string:
		return redactSecrets(typed)
	case []any:
		for index, item := range typed {
			typed[index] = maskValue(item)
		}
	case map[string]any:
		for key, item := range typed {
			if sensitiveKey.MatchString(key) {
				typed[key] = "[redacted]"
			} else {
				typed[key] = maskValue(item)
			}
		}
	}
	return value
}

// redactSecrets masks authentication material that slipped into free text,
// as a second line of defense behind the client-side masking.
func redactSecrets(text string) string {
	for _, pattern := range secretPatterns {
		text = pattern.ReplaceAllString(text, "[redacted]")
	}
	return text
}

func buildBugBody(description string, rawContext json.RawMessage) string {
	var out strings.Builder
	out.WriteString(redactSecrets(description))
	out.WriteString("\n\n_Reported from the in-game bug report dialog._\n")
	if len(rawContext) == 0 || string(rawContext) == "null" {
		return out.String()
	}
	contextText, ok := maskContext(rawContext)
	if !ok {
		return out.String()
	}
	// Never let a stray fence close the code block early.
	contextText = strings.ReplaceAll(contextText, "```", "'''")
	const head = "\n<details><summary>Captured context</summary>\n\n```json\n"
	const tail = "\n```\n\n</details>\n"
	room := bugBodyMaxBytes - out.Len() - len(head) - len(tail) - len("\n… truncated")
	if room <= 0 {
		return out.String()
	}
	if len(contextText) > room {
		contextText = truncateUTF8(contextText, room) + "\n… truncated"
	}
	out.WriteString(head + contextText + tail)
	return out.String()
}

// truncateUTF8 cuts text to at most limit bytes without splitting a rune.
func truncateUTF8(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}
