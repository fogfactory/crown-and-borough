package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func emitWinterRumors(ctx *resolutionContext) {
	players := make([]models.PlayerID, 0)
	for playerID, kinds := range ctx.deckDraws {
		if len(kinds) > 0 {
			players = append(players, playerID)
		}
	}
	if len(players) < 2 {
		return
	}
	sort.Slice(players, func(i, j int) bool { return players[i] < players[j] })
	rumorCounts := make(map[models.CardKind]int)
	index := 0
	for _, playerID := range players {
		for _, kind := range ctx.deckDraws[playerID] {
			if newRumorRNG(ctx.state.Seed, ctx.state.Turn, index).IntN(2) != 0 {
				index++
				continue
			}
			rumorCounts[kind]++
			index++
		}
	}
	ctx.events = append(ctx.events, rumorEvents(rumorCounts, len(ctx.state.Players), ctx.balance.SpecialOrders.DrawOrdersLimit)...)
}

func rumorEvents(counts map[models.CardKind]int, playerCount, drawLimit int) []Event {
	kinds := make([]models.CardKind, 0, len(counts))
	for kind, count := range counts {
		if count > 0 {
			kinds = append(kinds, kind)
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	events := make([]Event, 0, len(kinds))
	for _, kind := range kinds {
		level := rumorLevel(counts[kind], playerCount, drawLimit)
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

func rumorLevel(count, playerCount, drawLimit int) int {
	if count <= 0 {
		return 0
	}
	capacity := playerCount * drawLimit
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

func newRumorRNG(seed string, turn, index int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|winter-rumor|%d|%d", seed, turn, index)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}
