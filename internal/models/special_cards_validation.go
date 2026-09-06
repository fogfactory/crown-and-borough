package models

import "fmt"

func validateSpecialDeck(deck *SpecialDeck, auguries map[int]YearAugury, players map[PlayerID]bool) error {
	if deck == nil {
		return nil
	}
	cards := make(map[SpecialCardID]CardKind, len(deck.Cards))
	for _, card := range deck.Cards {
		if card.ID == "" || !card.Kind.IsValid() {
			return fmt.Errorf("models: special card %q: invalid id or kind", card.ID)
		}
		if _, exists := cards[card.ID]; exists {
			return fmt.Errorf("models: special card %q: duplicate id", card.ID)
		}
		cards[card.ID] = card.Kind
	}
	locations := make(map[SpecialCardID]string, len(cards))
	checkCard := func(cardID SpecialCardID, location string, want func(CardKind) bool) error {
		kind, exists := cards[cardID]
		if !exists {
			return fmt.Errorf("models: special card %q: %s references unknown card", cardID, location)
		}
		if want != nil && !want(kind) {
			return fmt.Errorf("models: special card %q: invalid kind %q in %s", cardID, kind, location)
		}
		if previous, exists := locations[cardID]; exists {
			return fmt.Errorf("models: special card %q: located in %s and %s", cardID, previous, location)
		}
		locations[cardID] = location
		return nil
	}
	for index, cardID := range deck.DrawPile {
		if err := checkCard(cardID, fmt.Sprintf("draw pile index %d", index), nil); err != nil {
			return err
		}
	}
	for index, cardID := range deck.Discard {
		if err := checkCard(cardID, fmt.Sprintf("discard index %d", index), nil); err != nil {
			return err
		}
	}
	for playerID, hand := range deck.Hands {
		if !players[playerID] {
			return fmt.Errorf("models: special deck: hand has unknown player %q", playerID)
		}
		for index, cardID := range hand {
			if err := checkCard(cardID, fmt.Sprintf("hand of player %q index %d", playerID, index), CardKind.IsBonus); err != nil {
				return err
			}
		}
	}
	for year, augury := range auguries {
		if augury.Year != 0 && augury.Year != year {
			return fmt.Errorf("models: augury %d: year field does not match key", year)
		}
		for _, season := range []Season{SeasonSpring, SeasonSummer, SeasonWinter} {
			capacity, exists := augury.Capacities[season]
			if !exists || capacity < 0 {
				return fmt.Errorf("models: augury %d: invalid capacity for %q", year, season)
			}
		}
		for season := range augury.Capacities {
			if season != SeasonSpring && season != SeasonSummer && season != SeasonWinter {
				return fmt.Errorf("models: augury %d: invalid season capacity %q", year, season)
			}
		}
		counts := map[Season]int{}
		for index, calamity := range augury.Calamities {
			if !calamity.Kind.IsCalamity() || calamity.Year != year {
				return fmt.Errorf("models: augury %d: invalid calamity at index %d", year, index)
			}
			counts[calamity.Season]++
			if counts[calamity.Season] > augury.Capacities[calamity.Season] {
				return fmt.Errorf("models: augury %d: season %q exceeds capacity", year, calamity.Season)
			}
			cardKind, exists := cards[calamity.CardID]
			if !exists || cardKind != calamity.Kind {
				return fmt.Errorf("models: augury %d: calamity card %q kind does not match", year, calamity.CardID)
			}
			if err := checkCard(calamity.CardID, fmt.Sprintf("augury %d calamity index %d", year, index), CardKind.IsCalamity); err != nil {
				return err
			}
		}
	}
	if len(locations) != len(cards) {
		return fmt.Errorf("models: special deck: %d of %d cards have no location", len(cards)-len(locations), len(cards))
	}
	return nil
}
