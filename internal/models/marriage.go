package models

import "fmt"

// Marriage records a concluded marriage between two nobles of two distinct
// players (specs/succession.md § Conclusion d'un mariage). NobleA and NobleB
// are kept in the order the marriage was concluded. The record outlives its
// spouses so the lineage keeps the alliance, but the marriage itself ends
// with the death of either spouse: the survivor is free to remarry (see
// Active). Turn is the absolute GameState.Turn of the winter that concluded
// it.
type Marriage struct {
	NobleA NobleID `json:"nobleA"`
	NobleB NobleID `json:"nobleB"`
	Turn   int     `json:"turn"`
	// Dissolved is set when the pope dissolved the marriage (stage 4 of the
	// winter); DissolvedTurn is the turn it happened. The record is kept so
	// the Claims it justified stay valid.
	Dissolved     bool `json:"dissolved,omitempty"`
	DissolvedTurn int  `json:"dissolvedTurn,omitempty"`
}

// Active reports whether both spouses are still alive and the pope has not
// dissolved the marriage: the death of either ends it.
func (m Marriage) Active(g *GameState) bool {
	if m.Dissolved {
		return false
	}
	living := 0
	for _, noble := range g.Nobles {
		if noble.ID == m.NobleA || noble.ID == m.NobleB {
			living++
		}
	}
	return living == 2
}

// MarriageOf returns the active marriage the noble belongs to, if any. A
// marriage ended by the death of the other spouse no longer counts.
func (g *GameState) MarriageOf(nobleID NobleID) (Marriage, bool) {
	for _, marriage := range g.Marriages {
		if (marriage.NobleA == nobleID || marriage.NobleB == nobleID) && marriage.Active(g) {
			return marriage, true
		}
	}
	return Marriage{}, false
}

// validateMarriages checks every marriage against the live nobles and the
// removed ones (both sets are passed in by Validate).
func (g *GameState) validateMarriages(nobles, removedNobles map[NobleID]bool) error {
	owners := make(map[NobleID]PlayerID, len(g.Nobles)+len(g.RemovedNobles))
	sexes := make(map[NobleID]Sex, len(g.Nobles)+len(g.RemovedNobles))
	for _, noble := range g.Nobles {
		owners[noble.ID] = noble.OwnerID
		sexes[noble.ID] = noble.Sex
	}
	for _, removed := range g.RemovedNobles {
		owners[removed.ID] = removed.OwnerID
		sexes[removed.ID] = removed.Sex
	}
	married := make(map[NobleID]bool, 2*len(g.Marriages))
	for _, marriage := range g.Marriages {
		active := marriage.Active(g)
		for _, id := range []NobleID{marriage.NobleA, marriage.NobleB} {
			if !nobles[id] && !removedNobles[id] {
				return fmt.Errorf("models: marriage: unknown noble %q", id)
			}
			// Only active marriages exclude each other: a widowed noble keeps
			// its ended marriage on record and may marry again.
			if active && married[id] {
				return fmt.Errorf("models: marriage: noble %q is in more than one active marriage", id)
			}
			if active {
				married[id] = true
			}
		}
		if marriage.NobleA == marriage.NobleB {
			return fmt.Errorf("models: marriage: noble %q cannot marry itself", marriage.NobleA)
		}
		if owners[marriage.NobleA] == owners[marriage.NobleB] {
			return fmt.Errorf("models: marriage %q/%q: spouses belong to the same player", marriage.NobleA, marriage.NobleB)
		}
		if sexes[marriage.NobleA] == sexes[marriage.NobleB] {
			return fmt.Errorf("models: marriage %q/%q: spouses must be of different sex", marriage.NobleA, marriage.NobleB)
		}
		if marriage.Turn < 0 || marriage.Turn > g.Turn {
			return fmt.Errorf("models: marriage %q/%q: turn %d must be between 0 and %d", marriage.NobleA, marriage.NobleB, marriage.Turn, g.Turn)
		}
	}
	return nil
}
