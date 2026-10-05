package models

import (
	"sort"
	"strconv"
)

// SuccessionLine returns the living nobles of playerID in line-of-succession
// order (specs/succession.md § Lignée): purchase order, whatever their sex.
// Noble IDs are a global, strictly increasing recruitment sequence, so the
// order is recovered from them. Hostages and prisoners stay in the line; dead
// nobles (GameState.RemovedNobles) have left it. IDs that do not follow the
// "N<number>" form sort last, in slice order.
func (g *GameState) SuccessionLine(playerID PlayerID) []Noble {
	line := []Noble{}
	for _, noble := range g.Nobles {
		if noble.OwnerID == playerID {
			line = append(line, noble)
		}
	}
	sort.SliceStable(line, func(i, j int) bool {
		a, aOK := nobleSequence(line[i].ID)
		b, bOK := nobleSequence(line[j].ID)
		if aOK != bOK {
			return aOK
		}
		return aOK && a < b
	})
	return line
}

// SuccessionLines returns the line of succession of every player, keyed by
// player ID.
func (g *GameState) SuccessionLines() map[PlayerID][]Noble {
	lines := make(map[PlayerID][]Noble, len(g.Players))
	for _, player := range g.Players {
		lines[player.ID] = g.SuccessionLine(player.ID)
	}
	return lines
}

func nobleSequence(id NobleID) (int, bool) {
	value := string(id)
	if len(value) < 2 || value[0] != 'N' {
		return 0, false
	}
	sequence, err := strconv.Atoi(value[1:])
	if err != nil || sequence < 1 {
		return 0, false
	}
	return sequence, true
}
