package assetgen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/models"
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
// translations available alongside it.
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
		rendered, err := renderRules(document, *balance)
		if err != nil {
			return nil, fmt.Errorf("assetgen: %s: %w", path, err)
		}
		document = rendered
	}
	return document, nil
}

func renderRules(document []byte, balance Balance) ([]byte, error) {
	calamityCounts := WeightedCardCounts(balance.SpecialOrders.DeckSize*balance.SpecialOrders.CalamityPercentage/100, balance.SpecialOrders.CalamityWeights, calamityKinds)
	bonusCounts := WeightedCardCounts(balance.SpecialOrders.DeckSize-(balance.SpecialOrders.DeckSize*balance.SpecialOrders.CalamityPercentage/100), balance.SpecialOrders.BonusWeights, bonusKinds)
	values := map[string]string{
		"special_orders.deck_size":                   stringValue(balance.SpecialOrders.DeckSize),
		"special_orders.calamity_percentage":         stringValue(balance.SpecialOrders.CalamityPercentage),
		"special_orders.hand_limit":                  stringValue(balance.SpecialOrders.HandLimit),
		"special_orders.draw_orders_limit":           stringValue(balance.SpecialOrders.DrawOrdersLimit),
		"special_orders.calamity_slots.spring":       stringValue(balance.SpecialOrders.CalamitySlots[models.SeasonSpring]),
		"special_orders.calamity_slots.summer":       stringValue(balance.SpecialOrders.CalamitySlots[models.SeasonSummer]),
		"special_orders.calamity_slots.winter":       stringValue(balance.SpecialOrders.CalamitySlots[models.SeasonWinter]),
		"special_orders.card.plague":                 stringValue(calamityCounts[models.CardKindPlague]),
		"special_orders.card.bad_weather":            stringValue(calamityCounts[models.CardKindBadWeather]),
		"special_orders.card.famine":                 stringValue(calamityCounts[models.CardKindFamine]),
		"special_orders.card.fair_weather":           stringValue(bonusCounts[models.CardKindFairWeather]),
		"special_orders.card.abundant_harvest":       stringValue(bonusCounts[models.CardKindAbundantHarvest]),
		"special_orders.card.revolt":                 stringValue(bonusCounts[models.CardKindRevolt]),
		"special_orders.effects.plague_army_divisor": stringValue(balance.SpecialOrders.Effects.PlagueArmyDivisor),
		"special_orders.effects.revolt_army_count":   stringValue(balance.SpecialOrders.Effects.RevoltArmyCount),
	}
	keys := make([]string, 0, len(values)*2)
	for key, value := range values {
		keys = append(keys, "{{"+key+"}}", value)
	}
	return []byte(strings.NewReplacer(keys...).Replace(string(document))), nil
}

func stringValue(value int) string {
	return fmt.Sprintf("%d", value)
}
