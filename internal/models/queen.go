package models

// RefreshQueen updates Queen after a change of the king, of a marriage or of a
// death (specs/politique.md § Reine et régente). The queen is the spouse of
// the king in place. A widow keeps the title, as a dowager queen, while no
// married king is in place; the title ends when she dies, remarries, or when
// the pope dissolves her marriage in her lifetime.
func (g *GameState) RefreshQueen() {
	if g.King != nil {
		for _, marriage := range g.Marriages {
			if (marriage.NobleA == *g.King || marriage.NobleB == *g.King) && marriage.Active(g) {
				spouse := marriage.SpouseOf(*g.King)
				g.Queen = &spouse
				return
			}
		}
		// A king who just died: his widow is the queen even when no refresh ran
		// since the marriage.
		if g.Queen == nil && !g.hasNoble(*g.King) {
			for _, marriage := range g.Marriages {
				if marriage.Dissolved || (marriage.NobleA != *g.King && marriage.NobleB != *g.King) {
					continue
				}
				if widow := marriage.SpouseOf(*g.King); g.hasNoble(widow) {
					g.Queen = &widow
					return
				}
			}
		}
	}
	if g.Queen != nil && !g.isDowager(*g.Queen) {
		g.Queen = nil
	}
}

// isDowager reports whether the noble is a living widow: no active marriage,
// and the latest marriage on record ended with her spouse's death rather than
// by a dissolution.
func (g *GameState) isDowager(id NobleID) bool {
	if !g.hasNoble(id) {
		return false
	}
	if _, married := g.MarriageOf(id); married {
		return false
	}
	var latest *Marriage
	for index := range g.Marriages {
		marriage := &g.Marriages[index]
		if (marriage.NobleA == id || marriage.NobleB == id) && (latest == nil || marriage.Turn >= latest.Turn) {
			latest = marriage
		}
	}
	return latest != nil && !latest.Dissolved
}

// IsQueen reports whether the noble holds the queen's title, dowager or not.
func (g *GameState) IsQueen(id NobleID) bool { return g.Queen != nil && *g.Queen == id }

// IsDowagerQueen reports whether the noble is a queen who is not the spouse of
// the king in place.
func (g *GameState) IsDowagerQueen(id NobleID) bool {
	if !g.IsQueen(id) {
		return false
	}
	if g.King == nil {
		return true
	}
	marriage, married := g.MarriageOf(id)
	return !married || marriage.SpouseOf(id) != *g.King
}

func (g *GameState) hasNoble(id NobleID) bool {
	for _, noble := range g.Nobles {
		if noble.ID == id {
			return true
		}
	}
	return false
}
