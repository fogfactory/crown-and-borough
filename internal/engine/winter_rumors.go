package engine

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func currentHandRumorEvents(state *models.GameState, handLimit int) []Event {
	if state == nil || state.SpecialDeck == nil {
		return nil
	}
	playersWithCards := 0
	rumorCounts := make(map[models.CardKind]int)
	for _, player := range state.Players {
		hand := state.SpecialDeck.Hands[player.ID]
		if len(hand) > 0 {
			playersWithCards++
		}
		for _, cardID := range hand {
			kind := cardKind(state.SpecialDeck, cardID)
			if kind.IsBonus() {
				rumorCounts[kind]++
			}
		}
	}
	if playersWithCards < 2 {
		return nil
	}
	return rumorEvents(rumorCounts, len(state.Players), handLimit)
}

func rumorEvents(counts map[models.CardKind]int, playerCount, handLimit int) []Event {
	kinds := make([]models.CardKind, 0, len(counts))
	for kind, count := range counts {
		if count > 0 {
			kinds = append(kinds, kind)
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	events := make([]Event, 0, len(kinds))
	for _, kind := range kinds {
		level := rumorLevel(counts[kind], playerCount, handLimit)
		events = append(events, Event{
			Type:       EventTypeRumor,
			Phase:      winterPhase,
			CardKind:   kind,
			RumorKey:   rumorKey(kind, level),
			RumorLevel: level,
		})
	}
	return events
}

func rumorLevel(count, playerCount, handLimit int) int {
	if count <= 0 {
		return 0
	}
	capacity := playerCount * handLimit
	if capacity < 1 {
		return 1
	}
	level := (count*3 + capacity - 1) / capacity
	if level > 3 {
		return 3
	}
	return level
}

func rumorKey(kind models.CardKind, level int) string {
	return fmt.Sprintf("rumor.%s.level%d", kind, level)
}
