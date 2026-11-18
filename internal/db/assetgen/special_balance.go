package assetgen

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

var calamityKinds = []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindFamine}
var bonusKinds = []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt, models.CardKindSeigneurialTax, models.CardKindTunnel, models.CardKindAssassination, models.CardKindJustice, models.CardKindAmbush}

// CalamityCardKinds returns the calamity kinds of the special deck in
// canonical order. It is the single list shared by the balance weights, the
// rules rendering and the deck builder.
func CalamityCardKinds() []models.CardKind { return append([]models.CardKind(nil), calamityKinds...) }

// BonusCardKinds returns the bonus kinds of the special deck in canonical
// order. It is the single list shared by the balance weights, the rules
// rendering and the deck builder.
func BonusCardKinds() []models.CardKind { return append([]models.CardKind(nil), bonusKinds...) }

func (raw rawBalance) specialOrders(path string) (SpecialOrdersBalance, error) {
	if raw.SpecialOrders == nil {
		return SpecialOrdersBalance{}, missingBalanceValue(path, "special_orders")
	}
	if raw.SpecialOrders.Effects == nil {
		return SpecialOrdersBalance{}, missingBalanceValue(path, "special_orders.effects")
	}
	handLimit, err := requiredNonNegativeInt(path, "special_orders.hand_limit", raw.SpecialOrders.HandLimit)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	drawLimit, err := requiredNonNegativeInt(path, "special_orders.draw_orders_limit", raw.SpecialOrders.DrawOrdersLimit)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	bonusBase, err := requiredNonNegativeInt(path, "special_orders.bonus_cards_base", raw.SpecialOrders.BonusCardsBase)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	bonusPerPlayer, err := requiredNonNegativeInt(path, "special_orders.bonus_cards_per_player", raw.SpecialOrders.BonusCardsPerPlayer)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	if bonusBase+bonusPerPlayer == 0 {
		return SpecialOrdersBalance{}, fmt.Errorf("assetgen: %s: special_orders.bonus_cards_base and bonus_cards_per_player must not both be zero", path)
	}
	calamityBase, err := requiredNonNegativeInt(path, "special_orders.calamity_cards_base", raw.SpecialOrders.CalamityCardsBase)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	calamityPerPlayer, err := requiredNonNegativeInt(path, "special_orders.calamity_cards_per_player", raw.SpecialOrders.CalamityCardsPerPlayer)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	slotPlayersPerExtra, err := requiredPositiveInt(path, "special_orders.calamity_slot_players_per_extra", raw.SpecialOrders.CalamitySlotPlayersPerExtra)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	slots, err := specialSeasonValues(path, raw.SpecialOrders.CalamitySlots, "special_orders.calamity_slots")
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	calamityWeights, err := specialWeights(path, raw.SpecialOrders.CalamityWeights, "special_orders.calamity_weights", calamityKinds)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	bonusWeights, err := specialWeights(path, raw.SpecialOrders.BonusWeights, "special_orders.bonus_weights", bonusKinds)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	effects := raw.SpecialOrders.Effects
	plagueDivisor, err := requiredPositiveInt(path, "special_orders.effects.plague_army_divisor", effects.PlagueArmyDivisor)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	mortality, err := requiredNonNegativeInt(path, "special_orders.effects.plague_noble_mortality_percentage", effects.PlagueNobleMortalityPercentage)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	if mortality > 100 {
		return SpecialOrdersBalance{}, fmt.Errorf("assetgen: %s: plague mortality must be <= 100", path)
	}
	revoltMin, err := requiredPositiveInt(path, "special_orders.effects.revolt_army_min_size", effects.RevoltArmyMinSize)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	revoltMax, err := requiredPositiveInt(path, "special_orders.effects.revolt_army_max_size", effects.RevoltArmyMaxSize)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	if revoltMin > revoltMax {
		return SpecialOrdersBalance{}, fmt.Errorf("assetgen: %s: revolt army minimum exceeds maximum", path)
	}
	return SpecialOrdersBalance{
		HandLimit: handLimit, DrawOrdersLimit: drawLimit,
		BonusCardsBase: bonusBase, BonusCardsPerPlayer: bonusPerPlayer,
		CalamityCardsBase: calamityBase, CalamityCardsPerPlayer: calamityPerPlayer,
		CalamitySlots: slots, CalamitySlotPlayersPerExtra: slotPlayersPerExtra,
		CalamityWeights: calamityWeights, BonusWeights: bonusWeights,
		Effects: SpecialOrderEffects{PlagueArmyDivisor: plagueDivisor, PlagueNobleMortalityPercentage: mortality, RevoltArmyMinSize: revoltMin, RevoltArmyMaxSize: revoltMax},
	}, nil
}

// BonusCardCount is the number of bonus (event) cards of the special deck of a
// game with the given number of players.
func (s SpecialOrdersBalance) BonusCardCount(players int) int {
	return s.BonusCardsBase + s.BonusCardsPerPlayer*players
}

// CalamityCardCount is the number of calamity cards of the special deck of a
// game with the given number of players.
func (s SpecialOrdersBalance) CalamityCardCount(players int) int {
	return s.CalamityCardsBase + s.CalamityCardsPerPlayer*players
}

// DeckSize is the size of the special deck of a game with the given number of
// players.
func (s SpecialOrdersBalance) DeckSize(players int) int {
	return s.BonusCardCount(players) + s.CalamityCardCount(players)
}

// CalamitySlotsFor is the number of calamities that can be scheduled in the
// given season of a year in a game with the given number of players.
func (s SpecialOrdersBalance) CalamitySlotsFor(season models.Season, players int) int {
	slots := s.CalamitySlots[season]
	if s.CalamitySlotPlayersPerExtra > 0 {
		slots += players / s.CalamitySlotPlayersPerExtra
	}
	return slots
}

func specialSeasonValues(path string, values map[string]*int, name string) (map[models.Season]int, error) {
	if values == nil {
		return nil, missingBalanceValue(path, name)
	}
	result := make(map[models.Season]int, 3)
	for _, season := range []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn} {
		value, exists := values[string(season)]
		if !exists || value == nil {
			return nil, missingBalanceValue(path, name+"."+string(season))
		}
		if *value < 0 {
			return nil, fmt.Errorf("assetgen: %s: value %q must be >= 0", path, name+"."+string(season))
		}
		result[season] = *value
	}
	for season := range values {
		if season != string(models.SeasonSpring) && season != string(models.SeasonSummer) && season != string(models.SeasonAutumn) {
			return nil, fmt.Errorf("assetgen: %s: invalid season %q in %s", path, season, name)
		}
	}
	return result, nil
}

func specialWeights(path string, values map[string]*int, name string, kinds []models.CardKind) (map[models.CardKind]int, error) {
	if values == nil {
		return nil, missingBalanceValue(path, name)
	}
	result := make(map[models.CardKind]int, len(kinds))
	total := 0
	for _, kind := range kinds {
		value, exists := values[string(kind)]
		if !exists || value == nil {
			return nil, missingBalanceValue(path, name+"."+string(kind))
		}
		if *value < 0 {
			return nil, fmt.Errorf("assetgen: %s: value %q must be >= 0", path, name+"."+string(kind))
		}
		result[kind] = *value
		total += *value
	}
	for kind := range values {
		valid := false
		for _, known := range kinds {
			if kind == string(known) {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("assetgen: %s: invalid kind %q in %s", path, kind, name)
		}
	}
	if total == 0 {
		return nil, fmt.Errorf("assetgen: %s: weights in %s must not all be zero", path, name)
	}
	return result, nil
}

func weightedCounts(total int, weights map[models.CardKind]int, canonical []models.CardKind) map[models.CardKind]int {
	result := make(map[models.CardKind]int, len(canonical))
	weightTotal := 0
	for _, kind := range canonical {
		weightTotal += weights[kind]
	}
	if total == 0 || weightTotal == 0 {
		return result
	}
	type remainder struct {
		kind      models.CardKind
		remainder int
	}
	remainders := make([]remainder, 0, len(canonical))
	allocated := 0
	for _, kind := range canonical {
		product := total * weights[kind]
		result[kind] = product / weightTotal
		allocated += result[kind]
		remainders = append(remainders, remainder{kind: kind, remainder: product % weightTotal})
	}
	sort.SliceStable(remainders, func(i, j int) bool { return remainders[i].remainder > remainders[j].remainder })
	for index := 0; index < total-allocated; index++ {
		result[remainders[index].kind]++
	}
	return result
}
