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
		rendered, err := renderRules(document, *balance)
		if err != nil {
			return nil, fmt.Errorf("assetgen: %s: %w", path, err)
		}
		document = rendered
	}
	return document, nil
}

// ruleArmyCostSizes are the army sizes whose supply cost can be rendered with
// {{army_cost.N}} in the rules documents.
const ruleArmyCostSizes = 5

func renderRules(document []byte, balance Balance) ([]byte, error) {
	values := map[string]string{
		"territory_income":                            stringValue(balance.TerritoryIncome),
		"village_income":                              stringValue(balance.VillageIncome),
		"village_max_level":                           stringValue(balance.VillageMaxLevel),
		"supply_range":                                stringValue(balance.SupplyRange),
		"depot_range_bonus":                           stringValue(balance.DepotRangeBonus),
		"cost_base":                                   stringValue(balance.CostBase),
		"pillage_bonus":                               stringValue(balance.PillageBonus),
		"noble_command_bonus":                         stringValue(balance.NobleCommandBonus),
		"castle_defense_bonus":                        stringValue(balance.CastleDefenseBonus),
		"city_defense_bonus":                          stringValue(balance.CityDefenseBonus),
		"ration_terrain.plain":                        stringValue(balance.RationTerrain["plain"]),
		"ration_terrain.forest":                       stringValue(balance.RationTerrain["forest"]),
		"ration_terrain.hill":                         stringValue(balance.RationTerrain["hill"]),
		"ration_terrain.mountain":                     stringValue(balance.RationTerrain["mountain"]),
		"ration_terrain.swamp":                        stringValue(balance.RationTerrain["swamp"]),
		"winter_stock_divisor":                        stringValue(balance.WinterStockDivisor),
		"village_stock_cap":                           stringValue(balance.VillageStockCap),
		"castle_stock_cap":                            stringValue(balance.CastleStockCap),
		"prosperity_loss_threshold":                   stringValue(balance.ProsperityLossThreshold),
		"costs.castle":                                stringValue(balance.Costs.Castle),
		"costs.troop":                                 stringValue(balance.Costs.Troop),
		"costs.supply_depot":                          stringValue(balance.Costs.SupplyDepot),
		"costs.fief_per_territory":                    stringValue(balance.Costs.FiefPerTerritory),
		"costs.annex_per_territory":                   stringValue(balance.Costs.AnnexPerTerritory),
		"costs.rank_per_territory":                    stringValue(balance.Costs.RankPerTerritory),
		"royal.orders_per_winter":                     stringValue(balance.Royal.OrdersPerWinter),
		"noble_limit":                                 stringValue(balance.NobleLimit),
		"noble_limit_max":                             stringValue(balance.NobleLimitMax),
		"starting_nobles":                             stringValue(balance.StartingNobles),
		"starting_troops":                             stringValue(balance.StartingTroops),
		"starting_outposts":                           stringValue(balance.StartingOutposts),
		"starting_resources":                          stringValue(balance.StartingResources),
		"special_orders.bonus_cards_base":             stringValue(balance.SpecialOrders.BonusCardsBase),
		"special_orders.bonus_cards_per_player":       stringValue(balance.SpecialOrders.BonusCardsPerPlayer),
		"special_orders.calamity_cards_base":          stringValue(balance.SpecialOrders.CalamityCardsBase),
		"special_orders.calamity_cards_per_player":    stringValue(balance.SpecialOrders.CalamityCardsPerPlayer),
		"special_orders.slot_players_per_extra":       stringValue(balance.SpecialOrders.CalamitySlotPlayersPerExtra),
		"special_orders.deck_rows":                    specialDeckRows(balance.SpecialOrders),
		"special_orders.hand_limit":                   stringValue(balance.SpecialOrders.HandLimit),
		"special_orders.draw_orders_limit":            stringValue(balance.SpecialOrders.DrawOrdersLimit),
		"special_orders.effects.plague_army_divisor":  stringValue(balance.SpecialOrders.Effects.PlagueArmyDivisor),
		"special_orders.effects.revolt_army_min_size": stringValue(balance.SpecialOrders.Effects.RevoltArmyMinSize),
		"special_orders.effects.revolt_army_max_size": stringValue(balance.SpecialOrders.Effects.RevoltArmyMaxSize),
		"religion.appeasement_cost_base":              stringValue(balance.Religion.AppeasementCostBase),
		"religion.cardinal_cost":                      stringValue(balance.Religion.CardinalCost),
	}
	for index, cost := range balance.Costs.MillLevels {
		values[fmt.Sprintf("costs.mill_levels.%d", index)] = stringValue(cost)
	}
	cost := 1
	for size := 1; size <= ruleArmyCostSizes; size++ {
		values[fmt.Sprintf("army_cost.%d", size)] = stringValue(cost)
		cost *= balance.CostBase
	}
	keys := make([]string, 0, len(values)*2)
	for key, value := range values {
		keys = append(keys, "{{"+key+"}}", value)
	}
	return []byte(strings.NewReplacer(keys...).Replace(string(document))), nil
}

// The table of deck sizes of the player rules covers every player count of a
// game.
const (
	ruleDeckMinPlayers = models.MinimumGamePlayers
	ruleDeckMaxPlayers = models.MaximumGamePlayers
)

// specialDeckRows renders, one row per number of players, the bonus cards, the
// calamities, the deck size and the calamity slots per season of the special
// deck, for the "{{special_orders.deck_rows}}" placeholder of the rules.
func specialDeckRows(config SpecialOrdersBalance) string {
	rows := make([]string, 0, ruleDeckMaxPlayers-ruleDeckMinPlayers+1)
	for players := ruleDeckMinPlayers; players <= ruleDeckMaxPlayers; players++ {
		rows = append(rows, fmt.Sprintf("| %d | %d | %d | %d | %s |", players,
			config.BonusCardCount(players), config.CalamityCardCount(players), config.DeckSize(players),
			calamitySlotsCell(config, players)))
	}
	return strings.Join(rows, "\n")
}

// calamitySlotsCell renders the calamity slots per season: one number when the
// three seasons agree, otherwise spring / summer / autumn.
func calamitySlotsCell(config SpecialOrdersBalance, players int) string {
	spring := config.CalamitySlotsFor(models.SeasonSpring, players)
	summer := config.CalamitySlotsFor(models.SeasonSummer, players)
	autumn := config.CalamitySlotsFor(models.SeasonAutumn, players)
	if spring == summer && summer == autumn {
		return stringValue(spring)
	}
	return fmt.Sprintf("%d / %d / %d", spring, summer, autumn)
}

func stringValue(value int) string {
	return fmt.Sprintf("%d", value)
}
