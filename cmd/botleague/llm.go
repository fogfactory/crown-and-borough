package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Message is one chat-completions message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage counts the tokens a bot spent.
type Usage struct {
	Calls      int `json:"calls"`
	PromptToks int `json:"promptTokens"`
	OutputToks int `json:"completionTokens"`
}

func (u *Usage) add(other Usage) {
	u.Calls += other.Calls
	u.PromptToks += other.PromptToks
	u.OutputToks += other.OutputToks
}

// LLM completes a conversation. Implementations are safe for concurrent use.
type LLM interface {
	Complete(ctx context.Context, messages []Message) (string, Usage, error)
}

// newLLM builds the client of a player from its provider.
func newLLM(provider ProviderConfig, player PlayerConfig, limiter chan struct{}) (LLM, error) {
	switch provider.Kind {
	case "mock":
		return &mockLLM{}, nil
	case "openai", "":
		key := ""
		if provider.APIKeyEnv != "" {
			key = os.Getenv(provider.APIKeyEnv)
			if key == "" {
				return nil, fmt.Errorf("the environment variable %s holds no API key", provider.APIKeyEnv)
			}
		}
		if provider.BaseURL == "" {
			return nil, fmt.Errorf("provider has no base_url")
		}
		return &openAIClient{
			base:        strings.TrimRight(provider.BaseURL, "/"),
			key:         key,
			headers:     provider.Headers,
			model:       player.Model,
			temperature: player.Temperature,
			maxTokens:   player.MaxTokens,
			http:        &http.Client{Timeout: 6 * time.Minute},
			limiter:     limiter,
		}, nil
	}
	return nil, fmt.Errorf("unknown provider kind %q", provider.Kind)
}

// openAIClient talks to any OpenAI-compatible /chat/completions endpoint
// (OpenRouter, OpenCode Zen, Ollama, LM Studio, ...).
type openAIClient struct {
	base        string
	key         string
	headers     map[string]string
	model       string
	temperature *float64
	maxTokens   int
	http        *http.Client
	limiter     chan struct{}
}

func (c *openAIClient) Complete(ctx context.Context, messages []Message) (string, Usage, error) {
	body := map[string]any{"model": c.model, "messages": messages}
	if c.temperature != nil {
		body["temperature"] = *c.temperature
	}
	if c.maxTokens > 0 {
		body["max_tokens"] = c.maxTokens
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", Usage{}, err
	}
	// A transport error, a 429 or a 5xx is the provider being unreachable, not
	// the model answering badly: keep trying until the outage budget is spent.
	var lastErr error
	started := time.Now()
	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, backoff(attempt, lastErr)); err != nil {
				return "", Usage{}, err
			}
		}
		text, usage, retry, err := c.limitedOnce(ctx, payload)
		if err == nil {
			return text, usage, nil
		}
		lastErr = err
		if !retry {
			return "", Usage{}, err
		}
		if time.Since(started) > outageBudget {
			return "", Usage{}, fmt.Errorf("model %s: %w: %v", c.model, errProviderUnavailable, lastErr)
		}
	}
}

// limitedOnce holds a concurrency slot only while a request is in flight, so
// a bot waiting out a rate limit does not block the others.
func (c *openAIClient) limitedOnce(ctx context.Context, payload []byte) (string, Usage, bool, error) {
	select {
	case c.limiter <- struct{}{}:
		defer func() { <-c.limiter }()
	case <-ctx.Done():
		return "", Usage{}, false, ctx.Err()
	}
	return c.once(ctx, payload)
}

type retryAfterError struct {
	err   error
	after time.Duration
}

func (e *retryAfterError) Error() string { return e.err.Error() }
func (e *retryAfterError) Unwrap() error { return e.err }

// errProviderUnavailable marks a model call that kept failing on the network or
// on 429/5xx for outageBudget: the league stops instead of passing turns.
var errProviderUnavailable = errors.New("provider unavailable")

// outageBudget is how long a call keeps retrying an unreachable provider.
var outageBudget = 20 * time.Minute

func backoff(attempt int, last error) time.Duration {
	if retry, ok := last.(*retryAfterError); ok && retry.after > 0 {
		return retry.after
	}
	base := time.Duration(1<<min(attempt, 6)) * 2 * time.Second
	return base + time.Duration(rand.Int63n(int64(time.Second)))
}

func (c *openAIClient) once(ctx context.Context, payload []byte) (string, Usage, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", Usage{}, false, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		request.Header.Set("Authorization", "Bearer "+c.key)
	}
	for name, value := range c.headers {
		request.Header.Set(name, value)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return "", Usage{}, ctx.Err() == nil, err
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		failure := &retryAfterError{err: fmt.Errorf("model %s: %s: %s", c.model, response.Status, snippet(raw, 300))}
		if seconds, convErr := strconv.Atoi(response.Header.Get("Retry-After")); convErr == nil {
			failure.after = min(time.Duration(seconds)*time.Second, 2*time.Minute)
		}
		return "", Usage{}, true, failure
	}
	if response.StatusCode >= 300 {
		return "", Usage{}, false, fmt.Errorf("model %s: %s: %s", c.model, response.Status, snippet(raw, 500))
	}
	var reply struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return "", Usage{}, true, fmt.Errorf("model %s: unreadable answer: %s", c.model, snippet(raw, 300))
	}
	if reply.Error != nil && len(reply.Choices) == 0 {
		return "", Usage{}, true, fmt.Errorf("model %s: %s", c.model, reply.Error.Message)
	}
	if len(reply.Choices) == 0 || strings.TrimSpace(reply.Choices[0].Message.Content) == "" {
		return "", Usage{}, true, fmt.Errorf("model %s: empty answer", c.model)
	}
	return reply.Choices[0].Message.Content, Usage{Calls: 1, PromptToks: reply.Usage.Prompt, OutputToks: reply.Usage.Completion}, false, nil
}

func snippet(raw []byte, limit int) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > limit {
		text = text[:limit] + "…"
	}
	return text
}

// mockLLM is a deterministic stand-in that holds every noble in place and
// greets the table: it lets the plumbing be exercised without an API key.
type mockLLM struct {
	mu    sync.Mutex
	calls int
}

var mockNoblePattern = regexp.MustCompile(`(?m)^- ([A-Z]{3}) at ([A-Z]{3}) \(free\)`)
var mockPhasePattern = regexp.MustCompile(`PHASE: (\w+)`)

func (m *mockLLM) Complete(_ context.Context, messages []Message) (string, Usage, error) {
	m.mu.Lock()
	m.calls++
	calls := m.calls
	m.mu.Unlock()
	last := messages[len(messages)-1].Content
	phase := "orders"
	if match := mockPhasePattern.FindStringSubmatch(last); match != nil {
		phase = match[1]
	}
	decision := Decision{
		Thoughts: "mock: hold position",
		Memory:   fmt.Sprintf("- mock memory after call %d", calls),
	}
	if calls%3 == 1 {
		decision.Messages = []OutMessage{{To: "public", Body: "The mock lord greets the table."}}
	}
	if phase == "orders" {
		if !strings.Contains(last, "Season: winter") {
			own := last
			if _, after, found := strings.Cut(last, "Yours:\n"); found {
				own = after
			}
			for _, match := range mockNoblePattern.FindAllStringSubmatch(own, -1) {
				decision.Orders.Chains = append(decision.Orders.Chains, ChainOrder{Noble: match[1], Text: match[1] + "\nH " + match[2]})
			}
		}
	}
	raw, _ := json.Marshal(decision)
	return string(raw), Usage{Calls: 1}, nil
}
