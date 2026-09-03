package assetgen

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

var calamityKinds = []models.CardKind{models.CardKindPlague, models.CardKindBadWeather, models.CardKindRevolt, models.CardKindFamine}
var bonusKinds = []models.CardKind{models.CardKindFairWeather, models.CardKindAbundantHarvest}

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
	deckSize, err := requiredPositiveInt(path, "special_orders.deck_size", raw.SpecialOrders.DeckSize)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	percentage, err := requiredNonNegativeInt(path, "special_orders.calamity_percentage", raw.SpecialOrders.CalamityPercentage)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	if percentage > 100 || deckSize*percentage%100 != 0 {
		return SpecialOrdersBalance{}, fmt.Errorf("assetgen: %s: special_orders.calamity_percentage does not produce an integer card count", path)
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
	revoltCount, err := requiredNonNegativeInt(path, "special_orders.effects.revolt_army_count", effects.RevoltArmyCount)
	if err != nil {
		return SpecialOrdersBalance{}, err
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
	millBonus, err := requiredNonNegativeInt(path, "special_orders.effects.bonus_mill_production", effects.BonusMillProduction)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	rationBonus, err := requiredNonNegativeInt(path, "special_orders.effects.bonus_army_ration", effects.BonusArmyRation)
	if err != nil {
		return SpecialOrdersBalance{}, err
	}
	return SpecialOrdersBalance{HandLimit: handLimit, DrawOrdersLimit: drawLimit, DeckSize: deckSize, CalamityPercentage: percentage, CalamitySlots: slots, CalamityWeights: calamityWeights, BonusWeights: bonusWeights, Effects: SpecialOrderEffects{PlagueArmyDivisor: plagueDivisor, PlagueNobleMortalityPercentage: mortality, RevoltArmyCount: revoltCount, RevoltArmyMinSize: revoltMin, RevoltArmyMaxSize: revoltMax, BonusMillProduction: millBonus, BonusArmyRation: rationBonus}}, nil
}

func specialSeasonValues(path string, values map[string]*int, name string) (map[models.Season]int, error) {
	if values == nil {
		return nil, missingBalanceValue(path, name)
	}
	result := make(map[models.Season]int, 3)
	for _, season := range []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonWinter} {
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
		if season != string(models.SeasonSpring) && season != string(models.SeasonSummer) && season != string(models.SeasonWinter) {
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
