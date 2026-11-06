package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIError is an error answer of the game server.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details []OrderIssue
}

// OrderIssue is one rejected order line.
type OrderIssue struct {
	Noble   string `json:"noble"`
	Line    int    `json:"line"`
	Source  string `json:"source"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.Status, e.Code, e.Message)
}

// gameClient is the REST client of one account on one server.
type gameClient struct {
	base   string
	http   *http.Client
	tokens *tokenSource
}

func newGameClient(base string, tokens *tokenSource) *gameClient {
	return &gameClient{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 60 * time.Second}, tokens: tokens}
}

func (c *gameClient) do(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	refreshed := false
	for attempt := 0; ; attempt++ {
		token, err := c.tokens.Token(ctx)
		if err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+token)
		if payload != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := c.http.Do(request)
		if err == nil {
			raw, _ := io.ReadAll(response.Body)
			response.Body.Close()
			switch {
			case response.StatusCode == http.StatusUnauthorized && !refreshed:
				refreshed = true
				if _, refreshErr := c.tokens.Invalidate(ctx); refreshErr != nil {
					return refreshErr
				}
				continue
			case response.StatusCode >= 500 && attempt < 3:
				// retry below
			case response.StatusCode >= 300:
				return decodeAPIError(response.StatusCode, raw)
			default:
				if out == nil || len(bytes.TrimSpace(raw)) == 0 {
					return nil
				}
				if text, ok := out.(*string); ok {
					*text = string(raw)
					return nil
				}
				return json.Unmarshal(raw, out)
			}
		} else if attempt >= 3 || ctx.Err() != nil {
			return err
		}
		if err := sleepCtx(ctx, time.Duration(attempt+1)*2*time.Second); err != nil {
			return err
		}
	}
}

func decodeAPIError(status int, raw []byte) error {
	var body struct {
		Error   string       `json:"error"`
		Message string       `json:"message"`
		Errors  []OrderIssue `json:"errors"`
	}
	_ = json.Unmarshal(raw, &body)
	if body.Message == "" {
		body.Message = strings.TrimSpace(string(raw))
	}
	return &APIError{Status: status, Code: body.Error, Message: body.Message, Details: body.Errors}
}

func isAPIError(err error, code string) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == code
}

// SetDisplayName completes the profile the server requires before a game.
func (c *gameClient) SetDisplayName(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPut, "/api/auth/me", map[string]string{"displayName": name}, nil)
}

type GameDetail struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Status     string       `json:"status"`
	Turn       int          `json:"turn"`
	Season     string       `json:"season"`
	Winner     string       `json:"winner"`
	Players    []PlayerSlot `json:"players"`
	InviteCode string       `json:"inviteCode"`
}

type PlayerSlot struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Submitted bool   `json:"submitted"`
}

func (c *gameClient) CreateGame(ctx context.Context, name, seed string, years, players int) (GameDetail, error) {
	var game GameDetail
	body := map[string]any{"name": name, "seed": seed, "years": years, "spectate": true, "players": players}
	err := c.do(ctx, http.MethodPost, "/api/games", body, &game)
	return game, err
}

func (c *gameClient) Game(ctx context.Context, id string) (GameDetail, error) {
	var game GameDetail
	err := c.do(ctx, http.MethodGet, "/api/games/"+id, nil, &game)
	return game, err
}

type JoinResult struct {
	Joined bool       `json:"joined"`
	Player PlayerSlot `json:"player"`
}

func (c *gameClient) Join(ctx context.Context, id, inviteCode string) (JoinResult, error) {
	var result JoinResult
	err := c.do(ctx, http.MethodPost, "/api/games/"+id+"/join", map[string]string{"inviteCode": inviteCode}, &result)
	return result, err
}

func (c *gameClient) Rules(ctx context.Context, id string) (string, error) {
	var text string
	err := c.do(ctx, http.MethodGet, "/api/games/"+id+"/rules", nil, &text)
	return text, err
}

func (c *gameClient) Map(ctx context.Context, id string) (MapData, error) {
	var data MapData
	err := c.do(ctx, http.MethodGet, "/api/games/"+id+"/map", nil, &data)
	return data, err
}

func (c *gameClient) State(ctx context.Context, id string) (map[string]any, error) {
	var state map[string]any
	err := c.do(ctx, http.MethodGet, "/api/games/"+id+"/state", nil, &state)
	return state, err
}

func (c *gameClient) Report(ctx context.Context, id string, index int) (map[string]any, error) {
	var report map[string]any
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/games/%s/reports/%d", id, index), nil, &report)
	return report, err
}

func (c *gameClient) ReportCount(ctx context.Context, id string) (int, error) {
	var list []json.RawMessage
	err := c.do(ctx, http.MethodGet, "/api/games/"+id+"/reports", nil, &list)
	return len(list), err
}

// Orders is the body of an order submission.
type Orders struct {
	Chains  []ChainOrder  `json:"chains"`
	Winter  []WinterOrder `json:"winter"`
	Special []DeckOrder   `json:"special,omitempty"`
}

type ChainOrder struct {
	Noble string `json:"noble"`
	Text  string `json:"text"`
}

type WinterOrder struct {
	Lines string `json:"lines"`
}

type DeckOrder struct {
	Text string `json:"text"`
}

func (c *gameClient) Submit(ctx context.Context, id string, orders Orders) error {
	if orders.Chains == nil {
		orders.Chains = []ChainOrder{}
	}
	if orders.Winter == nil {
		orders.Winter = []WinterOrder{}
	}
	return c.do(ctx, http.MethodPost, "/api/games/"+id+"/orders", orders, nil)
}

func (c *gameClient) ForceResolve(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/api/games/"+id+"/resolve", map[string]any{"force": true}, nil)
}

// ChatMessage mirrors the server message.
type ChatMessage struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
	Author  string `json:"authorHouseId"`
	Body    string `json:"body"`
	Turn    struct {
		Number int    `json:"number"`
		Season string `json:"season"`
		Year   int    `json:"year"`
	} `json:"turn"`
}

// Messages returns the newest messages of a channel ("public" or the other
// house for a private conversation), oldest first.
func (c *gameClient) Messages(ctx context.Context, id, channel string, limit int) ([]ChatMessage, error) {
	var reply struct {
		Messages []ChatMessage `json:"messages"`
	}
	query := url.Values{"channel": {channel}, "limit": {fmt.Sprint(limit)}}
	if err := c.do(ctx, http.MethodGet, "/api/games/"+id+"/messages?"+query.Encode(), nil, &reply); err != nil {
		return nil, err
	}
	// The server answers newest first.
	for left, right := 0, len(reply.Messages)-1; left < right; left, right = left+1, right-1 {
		reply.Messages[left], reply.Messages[right] = reply.Messages[right], reply.Messages[left]
	}
	return reply.Messages, nil
}

func (c *gameClient) PostMessage(ctx context.Context, id, channel, body string) error {
	return c.do(ctx, http.MethodPost, "/api/games/"+id+"/messages", map[string]string{"channel": channel, "body": body}, nil)
}

// MapData is the static map: adjacencies and terrain of every territory.
type MapData struct {
	Territories []MapTerritory `json:"territories"`
	Regions     []MapRegion    `json:"regions"`
}

type MapTerritory struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Terrain     string   `json:"terrain"`
	Village     bool     `json:"village"`
	Adjacencies []string `json:"adjacencies"`
	Impassable  []string `json:"impassable"`
}

type MapRegion struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Territories []string `json:"territories"`
}
