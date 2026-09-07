package assetgen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultRulesLanguage is used when the API request omits ?lang.
	DefaultRulesLanguage = "fr"
	playerRulesAsset     = "regles-joueurs.md"
	englishRulesAsset    = "regles-joueurs.en.md"
)

// Rules contains the player-facing rules documents loaded from the assets
// directory. French is required; additional translations are optional.
type Rules struct {
	documents map[string][]byte
}

// LoadRules reads the required French player rules document and any optional
// translations available alongside it. When a balance is provided, supported
// placeholders in the documents are rendered from its values.
func LoadRules(dir string, balances ...Balance) (Rules, error) {
	documents := make(map[string][]byte, 2)
	var balance *Balance
	if len(balances) > 0 {
		balance = &balances[0]
	}

	french, err := readRulesDocument(filepath.Join(dir, playerRulesAsset), true, balance)
	if err != nil {
		return Rules{}, err
	}
	documents[DefaultRulesLanguage] = french

	english, err := readRulesDocument(filepath.Join(dir, englishRulesAsset), false, balance)
	if err != nil {
		return Rules{}, err
	}
	if english != nil {
		documents["en"] = english
	}

	return Rules{documents: documents}, nil
}

// Document returns a copy of the requested document so callers cannot mutate
// the asset retained by the server.
func (r Rules) Document(language string) ([]byte, bool) {
	if language == "" {
		language = DefaultRulesLanguage
	}
	document, ok := r.documents[strings.ToLower(strings.TrimSpace(language))]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), document...), true
}

func readRulesDocument(path string, required bool, balance *Balance) ([]byte, error) {
	document, err := os.ReadFile(path)
	if err != nil {
		if !required && errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("assetgen: %s: %w", path, err)
	}
	if len(bytes.TrimSpace(document)) == 0 {
		return nil, fmt.Errorf("assetgen: %s: empty file", path)
	}
	if balance != nil {
		document = renderRules(document, *balance)
	}
	return document, nil
}

func renderRules(document []byte, balance Balance) []byte {
	values := map[string]string{
		"ration_terrain.plain":    stringValue(balance.RationTerrain["plain"]),
		"ration_terrain.forest":   stringValue(balance.RationTerrain["forest"]),
		"ration_terrain.hill":     stringValue(balance.RationTerrain["hill"]),
		"ration_terrain.mountain": stringValue(balance.RationTerrain["mountain"]),
		"ration_terrain.swamp":    stringValue(balance.RationTerrain["swamp"]),
		"infra_rations_bonus":     stringValue(balance.InfraRationsBonus),
		"base_production":         stringValue(balance.BaseProduction),
	}
	keys := make([]string, 0, len(values)*2)
	for key, value := range values {
		keys = append(keys, "{{"+key+"}}", value)
	}
	return []byte(strings.NewReplacer(keys...).Replace(string(document)))
}

func stringValue(value int) string {
	return fmt.Sprintf("%d", value)
}
