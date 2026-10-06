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
// "N<number>" form sort last, in slice order. A noble whose dignity places it
// last in the line (the bastard, specs/succession.md § Bâtard) comes after
// every other noble, whatever its purchase order.
func (g *GameState) SuccessionLine(playerID PlayerID) []Noble {
	line := []Noble{}
	for _, noble := range g.Nobles {
		if noble.OwnerID == playerID {
			line = append(line, noble)
		}
	}
	sort.SliceStable(line, func(i, j int) bool {
		if lastI, lastJ := line[i].LastInSuccession(), line[j].LastInSuccession(); lastI != lastJ {
			return lastJ
		}
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

// TitleRank is the succession rank of a fief title: barony 1 up to duchy 4.
func (t FiefTitle) Rank() int { return fiefTitleRanks[t] }

// CanReceiveTitle reports whether nobleID may be granted a fief of the given
// title (specs/succession.md § Lignée): every noble placed above it in its
// owner's line of succession must already hold a fief of an equal or higher
// title. Hostages and prisoners stay in the line and are not skipped. A
// noble whose dignity places it last in the line (the bastard,
// specs/succession.md § Bâtard) may only receive a title when it is the last
// of its lineage: no other noble of its owner is free of that dignity.
func (g *GameState) CanReceiveTitle(nobleID NobleID, title FiefTitle) bool {
	var owner PlayerID
	found := false
	for _, noble := range g.Nobles {
		if noble.ID == nobleID {
			owner, found = noble.OwnerID, true
			break
		}
	}
	if !found {
		return false
	}
	line := g.SuccessionLine(owner)
	for _, candidate := range line {
		if candidate.ID == nobleID && candidate.LastInSuccession() {
			for _, other := range line {
				if !other.LastInSuccession() {
					return false
				}
			}
		}
	}
	for _, above := range line {
		if above.ID == nobleID {
			return true
		}
		held := g.highestHeldFief(above.ID)
		if held == nil || held.Title.Rank() < title.Rank() {
			return false
		}
	}
	return false
}
