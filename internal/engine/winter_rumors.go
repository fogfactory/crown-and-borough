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
	index := 0
	for _, playerID := range players {
		for _, kind := range ctx.deckDraws[playerID] {
			if newRumorRNG(ctx.state.Seed, ctx.state.Turn, index).IntN(2) != 0 {
				index++
				continue
			}
			ctx.events = append(ctx.events, Event{Type: EventTypeRumor, Phase: winterPhase, CardKind: kind, RumorKey: rumorKey(kind)})
			index++
		}
	}
}

func rumorKey(kind models.CardKind) string {
	return "rumor." + string(kind)
}

func newRumorRNG(seed string, turn, index int) *rand.Rand {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|winter-rumor|%d|%d", seed, turn, index)))
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(digest[:8]), binary.BigEndian.Uint64(digest[8:16])))
}
