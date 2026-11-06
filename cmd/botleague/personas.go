package main

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed personas/*.yaml
var personaFiles embed.FS

// Persona is the temperament of a house. The three traits are independent
// dimensions that shape target choice, deal quality and betrayal thresholds,
// not a chat costume.
type Persona struct {
	ID            string   `yaml:"id"`
	DisplayName   string   `yaml:"display_name"`
	Traits        Traits   `yaml:"traits"`
	DecisionRules []string `yaml:"decision_rules"`
}

type Traits struct {
	PlayStyle string `yaml:"play_style"`
	Trust     string `yaml:"trust"`
	Tone      string `yaml:"tone"`
}

var (
	validPlayStyles = []string{"aggressive", "defensive", "opportunistic", "mercantile", "honest", "treacherous", "random"}
	validTrust      = []string{"high", "medium", "low", "none"}
	validTones      = []string{"curt", "formal", "friendly", "theatrical", "threatening"}
)

// loadPersona reads a catalogue persona and applies trait overrides.
func loadPersona(id string, overrides map[string]string) (Persona, error) {
	raw, err := personaFiles.ReadFile("personas/" + id + ".yaml")
	if err != nil {
		return Persona{}, fmt.Errorf("unknown persona %q (known: %s)", id, strings.Join(personaIDs(), ", "))
	}
	var persona Persona
	if err := yaml.Unmarshal(raw, &persona); err != nil {
		return Persona{}, fmt.Errorf("persona %s: %w", id, err)
	}
	for name, value := range overrides {
		switch name {
		case "play_style":
			persona.Traits.PlayStyle = value
		case "trust":
			persona.Traits.Trust = value
		case "tone":
			persona.Traits.Tone = value
		default:
			return Persona{}, fmt.Errorf("unknown trait %q (play_style, trust or tone)", name)
		}
	}
	if !slices.Contains(validPlayStyles, persona.Traits.PlayStyle) {
		return Persona{}, fmt.Errorf("invalid play_style %q (%s)", persona.Traits.PlayStyle, strings.Join(validPlayStyles, ", "))
	}
	if !slices.Contains(validTrust, persona.Traits.Trust) {
		return Persona{}, fmt.Errorf("invalid trust %q (%s)", persona.Traits.Trust, strings.Join(validTrust, ", "))
	}
	if !slices.Contains(validTones, persona.Traits.Tone) {
		return Persona{}, fmt.Errorf("invalid tone %q (%s)", persona.Traits.Tone, strings.Join(validTones, ", "))
	}
	return persona, nil
}

func personaIDs() []string {
	entries, _ := personaFiles.ReadDir("personas")
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	return ids
}

// Anchor is the permanent identity sentence at the head of a bot's memory.
func (p Persona) Anchor(houseName string) string {
	return fmt.Sprintf("I am %s (%s). Personality: play_style=%s, trust=%s, tone=%s. Stay in this role when evaluating deals and orders.",
		houseName, p.DisplayName, p.Traits.PlayStyle, p.Traits.Trust, p.Traits.Tone)
}

// Brief is the persona section of the system prompt.
func (p Persona) Brief() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Your persona is %s.\n", p.DisplayName)
	fmt.Fprintf(&b, "- play_style: %s\n- trust: %s\n- tone: %s\n", p.Traits.PlayStyle, p.Traits.Trust, p.Traits.Tone)
	b.WriteString("These dimensions are independent and change target selection, investment, deal quality, willingness to reveal information and betrayal thresholds. They are not a costume for your chat messages. A high-trust persona still punishes a broken pact; a low-trust persona may keep a promise that buys time. Never become uniformly cooperative because someone asks nicely.\n")
	if len(p.DecisionRules) > 0 {
		b.WriteString("Decision rules:\n")
		for _, rule := range p.DecisionRules {
			fmt.Fprintf(&b, "- %s\n", rule)
		}
	}
	return b.String()
}
