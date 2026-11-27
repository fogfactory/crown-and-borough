package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
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
	// Firestore documents are capped at 1 MiB; the rest of a report is small.
	bugGameViewMaxBytes = 1000000
	bugActorLimit       = 3
	bugActorWindow      = 10 * time.Minute
	bugGlobalLimit      = 30
	bugGlobalWindow     = time.Hour
)

// ErrBugReportsDisabled is returned when no GitHub token is configured.
var ErrBugReportsDisabled = errors.New("api: bug reports are disabled")

// IssueCreator files a GitHub issue and returns its public URL.
type IssueCreator interface {
	CreateIssue(ctx context.Context, title, body string, labels []string) (string, error)
}

// GitHubIssueClient creates issues through the GitHub REST API. Its tokens come
// from a TokenSource: a fixed bot token or a GitHub App installation.
type GitHubIssueClient struct {
	Tokens     TokenSource
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
	return &GitHubIssueClient{Tokens: staticToken(token), Repository: repository}
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
	token, err := c.Tokens.Token(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/repos/"+c.Repository+"/issues", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
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

// PlayerGameView is the projection of a game as one player sees it, with the
// labels that name a bug report.
type PlayerGameView struct {
	GameName   string
	Turn       int
	PlayerID   models.PlayerID
	PlayerName string
	View       any
}

// PlayerStateFunc returns the projection of a game as the actor sees it.
type PlayerStateFunc func(ctx context.Context, actor store.Actor, id store.GameID) (PlayerGameView, error)

// BugHandlerOptions configures BugsHandler.
type BugHandlerOptions struct {
	Actor  ActorResolver
	Issues IssueCreator    // nil disables the route
	State  PlayerStateFunc // nil attaches no game projection
	// Reports keeps the game view privately; the public issue only says where
	// it lives. Nil keeps reports in process memory.
	Reports store.BugReportStore
	Now     func() time.Time
}

// BugsHandler serves GET /api/bugs (availability) and POST /api/bugs. Its rate
// limits live in process memory: they are global only while the service runs a
// single instance (--max-instances 1 in the Cloud Run deployment).
type BugsHandler struct {
	actor   ActorResolver
	issues  IssueCreator
	state   PlayerStateFunc
	reports store.BugReportStore
	now     func() time.Time

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
	reports := options.Reports
	if reports == nil {
		reports = store.NewMemoryBugReportStore()
	}
	return &BugsHandler{actor: options.Actor, issues: issues, state: options.State, reports: reports, now: now, byUser: map[string][]time.Time{}}
}

type bugRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	// GameID names the game whose projection the server stores privately: the
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
	at, retryAfter, ok := h.allow(actor.ID)
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "too many bug reports, try again later")
		return
	}
	title = redactSecrets(title)
	report, gameNote := h.prepareGameView(r.Context(), actor, store.GameID(strings.TrimSpace(body.GameID)), title)
	issueURL, err := h.issues.CreateIssue(r.Context(), title, buildBugBody(description, body.Context, gameNote), []string{"bug"})
	if err != nil {
		// Nothing was filed, and nothing was stored yet: just give the slot
		// back, so an outage does not lock players out.
		h.release(actor.ID, at)
		writeAPIError(w, http.StatusBadGateway, "bug_report_failed", "the issue could not be created")
		return
	}
	if report != nil {
		h.storeGameView(context.WithoutCancel(r.Context()), report, issueURL)
	}
	writeJSON(w, http.StatusCreated, struct {
		URL string `json:"url"`
	}{URL: issueURL})
}

// storeGameView completes the report's ID with the issue's number, so a
// maintainer finds it from the issue, and stores it linked back to the issue.
// An issue number is unique, so CreateBugReport can never overwrite another
// report. The issue already points at the document, so a failure is only
// logged. report.ID holds the readable prefix on entry.
func (h *BugsHandler) storeGameView(ctx context.Context, report *store.BugReport, issueURL string) {
	number, ok := issueNumber(issueURL)
	if !ok {
		log.Printf("bug report: no issue number in %q, game view not stored", issueURL)
		return
	}
	report.ID += "-issue-" + number
	report.IssueURL = issueURL
	if err := h.reports.CreateBugReport(ctx, *report); err != nil {
		log.Printf("bug report %s: could not store the game view: %v", report.ID, err)
	}
}

// issueNumber returns the last path segment of an issue URL.
func issueNumber(issueURL string) (string, bool) {
	number := path.Base(issueURL)
	if n, err := strconv.Atoi(number); err != nil || n <= 0 {
		return "", false
	}
	return number, true
}

// prepareGameView builds the private report from the player's own projection
// of the game and returns the line the public issue shows in its place. The
// title is already redacted. The report is stored once the issue exists.
func (h *BugsHandler) prepareGameView(ctx context.Context, actor store.Actor, gameID store.GameID, title string) (*store.BugReport, string) {
	if gameID == "" || h.state == nil {
		return nil, ""
	}
	game, err := h.state(ctx, actor, gameID)
	if err != nil {
		if !errors.Is(err, store.ErrNotMember) && !errors.Is(err, store.ErrUnknownGame) {
			log.Printf("bug report: could not read game %s for %s: %v", gameID, actor.ID, err)
		}
		return nil, "_Game view: unavailable._"
	}
	raw, err := json.Marshal(game.View)
	if err != nil {
		log.Printf("bug report: could not encode the view of game %s: %v", gameID, err)
		return nil, "_Game view: unavailable._"
	}
	view, err := maskJSON(raw, false)
	if err != nil {
		log.Printf("bug report: could not encode the view of game %s: %v", gameID, err)
		return nil, "_Game view: unavailable._"
	}
	if len(view) > bugGameViewMaxBytes {
		log.Printf("bug report: view of game %s is %d bytes, over the stored limit", gameID, len(view))
		return nil, "_Game view: too large to store._"
	}
	report := &store.BugReport{
		ID:        bugReportIDPrefix(game),
		GameID:    gameID,
		Turn:      game.Turn,
		PlayerID:  game.PlayerID,
		UID:       actor.ID,
		Title:     title,
		GameView:  view,
		CreatedAt: h.now(),
	}
	// The full ID names the game and the player: the public issue only says
	// which document to look for, by its suffix.
	return report, fmt.Sprintf("Game view (private, maintainers only): Firestore `%s`, document ending in this issue's number (`-issue-<number>`)", store.BugReportCollection)
}

// bugReportIDPrefix reads as <game>-t<turn>-<player>; the issue number
// completes it, so a maintainer can tell reports apart in the Firestore
// console.
func bugReportIDPrefix(game PlayerGameView) string {
	return fmt.Sprintf("%s-t%d-%s",
		reportIDPart(game.GameName, "game"), game.Turn,
		reportIDPart(game.PlayerName, string(game.PlayerID)))
}

func reportIDPart(value, fallback string) string {
	slug := assetgen.Slug(value)
	if slug == "" {
		slug = assetgen.Slug(fallback)
	}
	if slug == "" {
		slug = "unknown"
	}
	if runes := []rune(slug); len(runes) > 32 {
		slug = strings.TrimRight(string(runes[:32]), "-")
	}
	return slug
}

// allow records a report at the returned time, or reports how long the actor
// must wait before the limits let one through.
func (h *BugsHandler) allow(actorID string) (time.Time, time.Duration, bool) {
	now := h.now()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.global = prune(h.global, now.Add(-bugGlobalWindow))
	// prune reuses the backing array, so every pruned slice is stored back.
	for id, times := range h.byUser {
		if kept := prune(times, now.Add(-bugActorWindow)); len(kept) == 0 {
			delete(h.byUser, id)
		} else {
			h.byUser[id] = kept
		}
	}
	recent := h.byUser[actorID]
	var wait time.Duration
	if len(recent) >= bugActorLimit {
		wait = recent[len(recent)-bugActorLimit].Add(bugActorWindow).Sub(now)
	}
	if len(h.global) >= bugGlobalLimit {
		wait = max(wait, h.global[len(h.global)-bugGlobalLimit].Add(bugGlobalWindow).Sub(now))
	}
	if wait > 0 {
		return time.Time{}, wait, false
	}
	h.byUser[actorID] = append(recent, now)
	h.global = append(h.global, now)
	return now, 0, true
}

// release gives back the slot allow recorded at the given time.
func (h *BugsHandler) release(actorID string, at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.global = removeTime(h.global, at)
	if times := removeTime(h.byUser[actorID], at); len(times) == 0 {
		delete(h.byUser, actorID)
	} else {
		h.byUser[actorID] = times
	}
}

func removeTime(times []time.Time, at time.Time) []time.Time {
	for index, entry := range times {
		if entry.Equal(at) {
			return append(times[:index], times[index+1:]...)
		}
	}
	return times
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

// Keep in sync with SECRET_IN_TEXT, SENSITIVE_KEY and SENSITIVE_QUERY in
// web/src/lib/bug-report.ts.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]*`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`),
}

// sensitiveQueryParam matches a sensitive query parameter inside a URL or
// path, keeping its name so the masked URL stays readable.
var sensitiveQueryParam = regexp.MustCompile(`(?i)([?&][^=&\s"]*(?:token|key|auth|secret|password|code|sig)[^=&\s"]*=)[^&\s#"]+`)

var sensitiveKey = regexp.MustCompile(`(?i)token|authorization|password|secret|api[-_]?key|cookie|invite|credential`)

// maskJSON decodes raw JSON, masks sensitive keys and token-like strings at
// any depth, and re-encodes it (indented when indent is set). The server
// cannot trust the client to have masked anything, and the private store is
// not a reason to keep credentials either.
func maskJSON(raw []byte, indent bool) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	var masked []byte
	var err error
	if indent {
		masked, err = json.MarshalIndent(maskValue(value), "", "  ")
	} else {
		masked, err = json.Marshal(maskValue(value))
	}
	return string(masked), err
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
	return sensitiveQueryParam.ReplaceAllString(text, "${1}[redacted]")
}

var mention = regexp.MustCompile(`@([A-Za-z0-9])`)

// defuseMentions breaks @handles with a zero-width space so a player cannot
// make the bot notify arbitrary GitHub users or teams.
func defuseMentions(text string) string {
	return mention.ReplaceAllString(text, "@​$1")
}

// buildBugBody writes the public issue body: the description, the note
// standing for the private game view, and the masked client capture.
func buildBugBody(description string, rawContext json.RawMessage, gameNote string) string {
	var out strings.Builder
	out.WriteString(defuseMentions(redactSecrets(description)))
	out.WriteString("\n\n_Reported from the in-game bug report dialog._\n")
	if gameNote != "" {
		out.WriteString("\n" + gameNote + "\n")
	}
	if len(rawContext) == 0 || string(rawContext) == "null" {
		return out.String()
	}
	contextText, err := maskJSON(rawContext, true)
	if err != nil {
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
