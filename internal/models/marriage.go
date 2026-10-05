package models

import "fmt"

// Marriage records a concluded marriage between two nobles of two distinct
// players (specs/succession.md § Conclusion d'un mariage). NobleA and NobleB
// are kept in the order the marriage was concluded. A marriage outlives its
// spouses: the record stays when a noble dies, so a surviving spouse cannot
// remarry and the lineage keeps the alliance. Turn is the absolute
// GameState.Turn of the winter that concluded it.
type Marriage struct {
	NobleA NobleID `json:"nobleA"`
	NobleB NobleID `json:"nobleB"`
	Turn   int     `json:"turn"`
}

// MarriageOf returns the marriage the noble belongs to, living or not.
func (g *GameState) MarriageOf(nobleID NobleID) (Marriage, bool) {
	for _, marriage := range g.Marriages {
		if marriage.NobleA == nobleID || marriage.NobleB == nobleID {
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
		for _, id := range []NobleID{marriage.NobleA, marriage.NobleB} {
			if !nobles[id] && !removedNobles[id] {
				return fmt.Errorf("models: marriage: unknown noble %q", id)
			}
			if married[id] {
				return fmt.Errorf("models: marriage: noble %q is married more than once", id)
			}
			married[id] = true
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
