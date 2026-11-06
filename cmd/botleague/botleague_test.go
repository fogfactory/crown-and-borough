package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseDecisionTolerantShapes(t *testing.T) {
	answer := "<think>hmm {not json}</think>Here you go:\n```json\n" + `{
  "thoughts": "go",
  "messages": [{"to": "P2", "body": "pact?"}],
  "orders": {
    "chains": [{"noble": "oge", "text": "CPN A CML\nCML A RBL"}, "PAY\nH CML"],
    "winter": ["R T CPN", "C D CML"],
    "special": [{"text": "P BT FAL"}, "P RA FAL"]
  },
  "memory": "I am X.\nnotes"
}` + "\n```"
	decision, err := parseDecision(answer)
	if err != nil {
		t.Fatal(err)
	}
	if got := decision.Orders.Chains[0]; got.Noble != "OGE" || got.Text != "OGE\nCPN A CML\nCML A RBL" {
		t.Errorf("first chain not normalized: %+v", got)
	}
	if got := decision.Orders.Chains[1]; got.Noble != "PAY" || got.Text != "PAY\nH CML" {
		t.Errorf("string chain not read: %+v", got)
	}
	if len(decision.Orders.Winter) != 1 || decision.Orders.Winter[0].Lines != "R T CPN\nC D CML" {
		t.Errorf("winter lines not merged: %+v", decision.Orders.Winter)
	}
	if len(decision.Orders.Special) != 2 {
		t.Errorf("special orders: %+v", decision.Orders.Special)
	}
	if len(decision.Messages) != 1 || decision.Messages[0].To != "P2" {
		t.Errorf("messages: %+v", decision.Messages)
	}
}

func TestParseDecisionRejectsProse(t *testing.T) {
	if _, err := parseDecision("I will attack at dawn."); err == nil {
		t.Fatal("prose without JSON must be rejected")
	}
	if _, err := parseDecision(`{"orders": {"chains": [`); err == nil {
		t.Fatal("truncated JSON must be rejected")
	}
}

func TestLoadPersonaOverrides(t *testing.T) {
	persona, err := loadPersona("conqueror", map[string]string{"trust": "low", "tone": "friendly"})
	if err != nil {
		t.Fatal(err)
	}
	if persona.Traits.PlayStyle != "aggressive" || persona.Traits.Trust != "low" || persona.Traits.Tone != "friendly" {
		t.Errorf("unexpected traits %+v", persona.Traits)
	}
	if _, err := loadPersona("conqueror", map[string]string{"trust": "maybe"}); err == nil {
		t.Error("an invalid trait value must be rejected")
	}
	if _, err := loadPersona("nobody", nil); err == nil {
		t.Error("an unknown persona must be rejected")
	}
	for _, id := range personaIDs() {
		if _, err := loadPersona(id, nil); err != nil {
			t.Errorf("catalogue persona %s: %v", id, err)
		}
	}
}

func validConfig() *Config {
	return &Config{
		Game: GameConfig{Create: true},
		Players: []PlayerConfig{
			{Name: "Red", Provider: "mock"},
			{Name: "Blue", Provider: "openrouter", Model: "vendor/model:free"},
		},
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := validConfig()
	if err := cfg.normalize("/base"); err != nil {
		t.Fatal(err)
	}
	if cfg.Players[0].Email != "bl-league-red@bots.local" || cfg.Players[0].Persona != "diplomat" {
		t.Errorf("player defaults: %+v", cfg.Players[0])
	}
	if cfg.Server.Mode != "emulator" || cfg.LeagueDir != "/base/league/league" {
		t.Errorf("server/league dir defaults: %+v %s", cfg.Server, cfg.LeagueDir)
	}
	if cfg.Providers["openrouter"].BaseURL == "" || cfg.Providers["zen"].BaseURL == "" {
		t.Error("builtin providers missing")
	}
}

func TestConfigRejectsMistakes(t *testing.T) {
	cases := map[string]func(*Config){
		"duplicate names":       func(c *Config) { c.Players[1].Name = "red" },
		"missing model":         func(c *Config) { c.Players[1].Model = "" },
		"unknown provider":      func(c *Config) { c.Players[1].Provider = "nope" },
		"one player":            func(c *Config) { c.Players = c.Players[:1] },
		"no game":               func(c *Config) { c.Game.Create = false },
		"hosted needs key":      func(c *Config) { c.Server.Mode = "hosted"; c.Game.Create = false; c.Game.ID = "g" },
		"hosted cannot create":  func(c *Config) { c.Server.Mode = "hosted"; c.Server.APIKey = "k" },
		"bad persona":           func(c *Config) { c.Players[0].Persona = "wizard" },
		"too many negotiations": func(c *Config) { c.NegotiationRounds = 9 },
	}
	for name, mutate := range cases {
		cfg := validConfig()
		mutate(cfg)
		if err := cfg.normalize("."); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestOobCodeFromLink(t *testing.T) {
	code, err := oobCodeFromLink("https://x.firebaseapp.com/__/auth/action?apiKey=k&mode=signIn&oobCode=ABC123&continueUrl=http%3A%2F%2Fx")
	if err != nil || code != "ABC123" {
		t.Fatalf("got %q, %v", code, err)
	}
	if _, err := oobCodeFromLink("https://example.com/"); err == nil {
		t.Error("a link without oobCode must be rejected")
	}
}

func TestRenderStateIsCompact(t *testing.T) {
	state := map[string]any{
		"turn": float64(3), "year": float64(1), "season": "summer",
		"scores":  map[string]any{"P1": map[string]any{"total": float64(1)}},
		"players": []any{map[string]any{"id": "P1", "name": "Red", "capitalTerritory": "CPN", "projectedIncome": float64(3)}},
		"territories": []any{
			map[string]any{"id": "CPN", "owner": "P1", "resources": float64(6), "army": map[string]any{"owner": "P1", "size": float64(2)}, "infrastructures": []any{map[string]any{"type": "castle", "level": float64(1)}}},
			map[string]any{"id": "ZZZ", "owner": nil, "resources": float64(0), "army": nil, "infrastructures": []any{}},
		},
		"nobles": []any{
			map[string]any{"code": "OGE", "location": "CPN", "status": "free", "name": "Oger", "owner": "P1"},
			map[string]any{"code": "AME", "location": "PYV", "status": "free", "name": "Ameline", "owner": "P2"},
		},
	}
	text := renderState(state, "P1")
	for _, want := range []string{"Season: summer", "CPN P1 stock6 army2(P1) castle1", "Neutral and empty: ZZZ", "Yours:\n- OGE at CPN (free)"} {
		if !strings.Contains(text, want) {
			t.Errorf("state text misses %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "AME at PYV") > strings.Index(text, "Yours:") {
		t.Error("rival nobles must come before the player's own")
	}
}

func TestCompactRulesKeepsOrderTables(t *testing.T) {
	rules := "# Règles\n## L'essentiel pour gagner\nwin\n## 3. Combattre\nfight\n## Annexe A — Aide-mémoire des ordres\ntable\n"
	got := compactRules(rules)
	if !strings.Contains(got, "win") || !strings.Contains(got, "table") || strings.Contains(got, "fight") {
		t.Errorf("compactRules: %q", got)
	}
}

func TestUpdateMemoryKeepsAnchorAndLimit(t *testing.T) {
	cfg := validConfig()
	if err := cfg.normalize(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	cfg.MemoryChars = 20
	persona, _ := loadPersona("turtle", nil)
	bot := &Bot{cfg: cfg.Players[0], house: "P1", persona: persona, dir: t.TempDir(), league: &League{cfg: cfg}}
	bot.updateMemory("I am someone else.\n" + strings.Repeat("x", 100))
	lines := strings.SplitN(bot.memory, "\n", 2)
	if !strings.Contains(lines[0], "play_style=defensive") {
		t.Errorf("anchor lost: %q", lines[0])
	}
	if strings.Count(bot.memory, "x") > 20 {
		t.Errorf("memory not bounded: %q", bot.memory)
	}
}

func TestOpenAIClientRetriesAndParses(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer secret" || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			Model    string    `json:"model"`
			Messages []Message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "vendor/free:free" || len(body.Messages) != 1 {
			t.Errorf("unexpected body %+v", body)
		}
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"thoughts\":\"ok\"}"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`))
	}))
	defer server.Close()
	t.Setenv("TEST_LLM_KEY", "secret")
	client, err := newLLM(ProviderConfig{Kind: "openai", BaseURL: server.URL + "/v1", APIKeyEnv: "TEST_LLM_KEY"}, PlayerConfig{Model: "vendor/free:free"}, make(chan struct{}, 1))
	if err != nil {
		t.Fatal(err)
	}
	answer, usage, err := client.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || usage.PromptToks != 12 || usage.OutputToks != 3 {
		t.Errorf("calls=%d usage=%+v", calls, usage)
	}
	if decision, err := parseDecision(answer); err != nil || decision.Thoughts != "ok" {
		t.Errorf("answer %q: %v", answer, err)
	}
}

func TestNewLLMNeedsKey(t *testing.T) {
	t.Setenv("MISSING_KEY_FOR_TEST", "")
	if _, err := newLLM(ProviderConfig{Kind: "openai", BaseURL: "http://x", APIKeyEnv: "MISSING_KEY_FOR_TEST"}, PlayerConfig{Model: "m"}, nil); err == nil {
		t.Error("a missing API key must be reported before the game starts")
	}
}

// The mock answers by reading the rendered prompt: this fails if a change of
// prompt.go breaks what it relies on.
func TestMockReadsRenderedState(t *testing.T) {
	state := map[string]any{
		"turn": float64(1), "year": float64(1), "season": "spring",
		"nobles": []any{
			map[string]any{"code": "OGE", "location": "CPN", "status": "free", "name": "Oger", "owner": "P1"},
			map[string]any{"code": "AME", "location": "PYV", "status": "free", "name": "Ameline", "owner": "P2"},
		},
	}
	prompt := "PHASE: orders\n" + renderState(state, "P1")
	answer, _, err := (&mockLLM{}).Complete(context.Background(), []Message{{Role: "user", Content: prompt}})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := parseDecision(answer)
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Orders.Chains) != 1 || decision.Orders.Chains[0].Text != "OGE\nH CPN" {
		t.Errorf("the mock must hold only its own noble: %+v", decision.Orders.Chains)
	}
}

// The order examples of the system prompt must stay valid against the player
// rules, which are the reference for the syntax.
func TestPromptOrderExamplesMatchPlayerRules(t *testing.T) {
	raw, err := os.ReadFile("../../assets/regles-joueurs.md")
	if err != nil {
		t.Fatal(err)
	}
	rules := string(raw)
	prompt := systemPrompt("P1", "Red", Persona{}, nil, MapData{}, "")
	// prompt example -> the syntax the rules give for it
	cases := map[string]string{
		"CPN A CML":    "`XXX A YYY`",
		"R T CPN":      "`R T XXX`",
		"C D CML":      "`C D XXX`",
		"P BT FAL":     "`P BT XXX`",
		"J (junction)": "`XXX J YYY`",
	}
	for example, syntax := range cases {
		if !strings.Contains(prompt, example) {
			t.Errorf("the prompt no longer shows %q", example)
		}
		if !strings.Contains(rules, syntax) {
			t.Errorf("the player rules no longer document %s (shown in the prompt as %q)", syntax, example)
		}
	}
}

func TestNormalizeRecipient(t *testing.T) {
	for input, want := range map[string]string{
		"P2 Maison Bleue": "P2",
		"p4":              "P4",
		" P1 ":            "P1",
		"public":          "public",
	} {
		if got := normalizeRecipient(input); got != want {
			t.Errorf("normalizeRecipient(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeChainRewritesPositionHold(t *testing.T) {
	chain := normalizeChain(ChainOrder{Noble: "AVE", Text: "AVE\nPVL H PVL\n(PGR H PGR)"})
	if want := "AVE\nH PVL\n(H PGR)"; chain.Text != want {
		t.Errorf("chain text = %q, want %q", chain.Text, want)
	}
}

func TestParseDecisionAcceptsRawLineBreaksInStrings(t *testing.T) {
	answer := "{\"orders\":{\"chains\":[{\"noble\":\"ELI\",\"text\":\"ELI\nCHL A HTC\"}]},\"memory\":\"line one\nline two\"}"
	decision, err := parseDecision(answer)
	if err != nil {
		t.Fatal(err)
	}
	if len(decision.Orders.Chains) != 1 || decision.Orders.Chains[0].Text != "ELI\nCHL A HTC" {
		t.Errorf("chains = %+v", decision.Orders.Chains)
	}
	if decision.Memory != "line one\nline two" {
		t.Errorf("memory = %q", decision.Memory)
	}
}

func TestOpenAIClientReportsAnUnreachableProvider(t *testing.T) {
	previous := outageBudget
	outageBudget = 0
	defer func() { outageBudget = previous }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()
	client := &openAIClient{base: server.URL, model: "m", http: server.Client(), limiter: make(chan struct{}, 1)}
	_, _, err := client.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if !errors.Is(err, errProviderUnavailable) {
		t.Fatalf("err = %v, want errProviderUnavailable", err)
	}
}
