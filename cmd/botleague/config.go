package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes one league: the server to play on, the game to create or
// join, and the language model behind each house.
type Config struct {
	Name      string `yaml:"name"`
	LeagueDir string `yaml:"league_dir"`

	Server    ServerConfig              `yaml:"server"`
	Game      GameConfig                `yaml:"game"`
	Host      HostConfig                `yaml:"host"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Players   []PlayerConfig            `yaml:"players"`

	// Rules is "full" (the whole player rules) or "compact" (summary,
	// seasons, orders, order cheat sheets).
	Rules string `yaml:"rules"`
	// NegotiationRounds is the number of message-only rounds that precede the
	// order round of every turn. Zero sends messages and orders together.
	NegotiationRounds int `yaml:"negotiation_rounds"`
	// MaxRepairs is how many times a bot may fix orders the server rejected.
	MaxRepairs int `yaml:"max_repairs"`
	// MaxTurns stops the league after that many resolved turns (0: play the
	// game to its end).
	MaxTurns int `yaml:"max_turns"`
	// Concurrency bounds the language-model calls in flight at once.
	Concurrency int `yaml:"concurrency"`
	// TurnTimeoutMinutes is how long the league waits for a turn to resolve
	// before the observer forces it (when it can).
	TurnTimeoutMinutes int `yaml:"turn_timeout_minutes"`
	// MemoryChars caps the private memory a bot keeps between turns.
	MemoryChars int `yaml:"memory_chars"`
}

type ServerConfig struct {
	// URL is the origin of the Crown & Borough server.
	URL string `yaml:"url"`
	// Mode is "emulator" (Firebase Auth emulator, links read from it) or
	// "hosted" (real Firebase, links read from the mails of the bots).
	Mode string `yaml:"mode"`
	// AuthEmulator is the emulator origin, in emulator mode.
	AuthEmulator string `yaml:"auth_emulator"`
	// FirebaseProject is the project id, used by the emulator admin routes.
	FirebaseProject string `yaml:"firebase_project"`
	// APIKey is the Firebase web API key (VITE_FIREBASE_API_KEY), hosted mode.
	APIKey string `yaml:"api_key"`
	// EmailDomain builds the default address of a bot without an email.
	EmailDomain string `yaml:"email_domain"`
}

type GameConfig struct {
	// Create makes the league create the game as the observer host (emulator
	// mode). When false, InviteURL or ID designates an existing game.
	Create     bool   `yaml:"create"`
	Name       string `yaml:"name"`
	Seed       string `yaml:"seed"`
	Years      int    `yaml:"years"`
	InviteURL  string `yaml:"invite_url"`
	ID         string `yaml:"id"`
	InviteCode string `yaml:"invite_code"`
}

type HostConfig struct {
	// Email is the observer account that creates the game (emulator mode).
	// Sign in with it in the web app to watch the game and read every chat.
	Email string `yaml:"email"`
}

type ProviderConfig struct {
	// Kind is "openai" (any chat-completions endpoint) or "mock".
	Kind      string            `yaml:"kind"`
	BaseURL   string            `yaml:"base_url"`
	APIKeyEnv string            `yaml:"api_key_env"`
	Headers   map[string]string `yaml:"headers"`
}

type PlayerConfig struct {
	// Name is the display name of the account; it is shown in the lobby.
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
	// Persona is a catalogue id (conqueror, diplomat, ...); Traits override
	// its play_style, trust and tone.
	Persona  string            `yaml:"persona"`
	Traits   map[string]string `yaml:"traits"`
	Provider string            `yaml:"provider"`
	Model    string            `yaml:"model"`
	// Temperature and MaxTokens tune the model calls.
	Temperature *float64 `yaml:"temperature"`
	MaxTokens   int      `yaml:"max_tokens"`
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slug(value string) string {
	return strings.Trim(slugPattern.ReplaceAllString(strings.ToLower(value), "-"), "-")
}

// builtinProviders are always available; the config may override them.
func builtinProviders() map[string]ProviderConfig {
	return map[string]ProviderConfig{
		"openrouter": {Kind: "openai", BaseURL: "https://openrouter.ai/api/v1", APIKeyEnv: "OPENROUTER_API_KEY"},
		"zen":        {Kind: "openai", BaseURL: "https://opencode.ai/zen/v1", APIKeyEnv: "OPENCODE_API_KEY"},
		"ollama":     {Kind: "openai", BaseURL: "http://localhost:11434/v1"},
		"mock":       {Kind: "mock"},
	}
}

func loadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cfg.normalize(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) normalize(baseDir string) error {
	if c.Name == "" {
		c.Name = "league"
	}
	c.Name = slug(c.Name)
	if c.LeagueDir == "" {
		c.LeagueDir = filepath.Join("league", c.Name)
	}
	if !filepath.IsAbs(c.LeagueDir) {
		c.LeagueDir = filepath.Join(baseDir, c.LeagueDir)
	}
	s := &c.Server
	if s.Mode == "" {
		s.Mode = "emulator"
	}
	if s.Mode != "emulator" && s.Mode != "hosted" {
		return fmt.Errorf("server.mode must be emulator or hosted, got %q", s.Mode)
	}
	if s.URL == "" {
		s.URL = "http://localhost:8080"
	}
	s.URL = strings.TrimRight(s.URL, "/")
	if s.AuthEmulator == "" {
		s.AuthEmulator = "http://127.0.0.1:9099"
	}
	if s.FirebaseProject == "" {
		s.FirebaseProject = "demo-crown-and-borough"
	}
	if s.EmailDomain == "" {
		s.EmailDomain = "bots.local"
	}
	if s.Mode == "emulator" && s.APIKey == "" {
		s.APIKey = "fake-api-key"
	}
	if s.Mode == "hosted" && s.APIKey == "" {
		return fmt.Errorf("server.api_key (the Firebase web API key) is required in hosted mode")
	}
	if c.Game.Name == "" {
		c.Game.Name = c.Name
	}
	if c.Game.Seed == "" {
		c.Game.Seed = c.Name
	}
	if c.Host.Email == "" {
		c.Host.Email = "admin@mail.com"
	}
	if !c.Game.Create && c.Game.InviteURL == "" && c.Game.ID == "" {
		return fmt.Errorf("game: set create: true, or an invite_url / id to join an existing game")
	}
	if c.Game.Create && s.Mode == "hosted" {
		return fmt.Errorf("game.create needs the emulator mode: on a hosted server the human host creates the game (tick \"Observe without playing\") and passes game.invite_url")
	}
	if c.Rules == "" {
		c.Rules = "full"
	}
	if c.Rules != "full" && c.Rules != "compact" {
		return fmt.Errorf("rules must be full or compact")
	}
	if c.MaxRepairs == 0 {
		c.MaxRepairs = 2
	}
	if c.Concurrency == 0 {
		c.Concurrency = 4
	}
	if c.TurnTimeoutMinutes == 0 {
		c.TurnTimeoutMinutes = 20
	}
	if c.MemoryChars == 0 {
		c.MemoryChars = 4000
	}
	if c.NegotiationRounds < 0 || c.NegotiationRounds > 3 {
		return fmt.Errorf("negotiation_rounds must be between 0 and 3")
	}
	providers := builtinProviders()
	for name, provider := range c.Providers {
		providers[name] = provider
	}
	c.Providers = providers
	if len(c.Players) < 2 || len(c.Players) > 8 {
		return fmt.Errorf("players: between 2 and 8 houses are required, got %d", len(c.Players))
	}
	seen := map[string]bool{}
	for index := range c.Players {
		player := &c.Players[index]
		if player.Persona == "" {
			player.Persona = "diplomat"
		}
		if player.Name == "" {
			player.Name = fmt.Sprintf("Bot %d", index+1)
		}
		key := slug(player.Name)
		if key == "" || seen[key] {
			return fmt.Errorf("players[%d]: name %q must be unique", index, player.Name)
		}
		seen[key] = true
		if player.Email == "" {
			player.Email = fmt.Sprintf("bl-%s-%s@%s", c.Name, key, s.EmailDomain)
		}
		if player.Provider == "" {
			return fmt.Errorf("players[%d] (%s): provider is required", index, player.Name)
		}
		provider, ok := c.Providers[player.Provider]
		if !ok {
			return fmt.Errorf("players[%d] (%s): unknown provider %q", index, player.Name, player.Provider)
		}
		if provider.Kind != "mock" && player.Model == "" {
			return fmt.Errorf("players[%d] (%s): model is required", index, player.Name)
		}
		if _, err := loadPersona(player.Persona, player.Traits); err != nil {
			return fmt.Errorf("players[%d] (%s): %w", index, player.Name, err)
		}
	}
	return nil
}
