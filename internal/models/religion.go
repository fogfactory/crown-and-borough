package models

import (
	"fmt"
	"slices"
)

// ReligiousTitle is a title of the Church a noble can hold (specs/religieux.md).
// Titles stack: a cardinal is always a bishop and the pope is a bishop or a
// cardinal, so a noble can hold several; only the highest counts for votes.
type ReligiousTitle string

const (
	ReligiousTitleNone     ReligiousTitle = ""
	ReligiousTitleBishop   ReligiousTitle = "bishop"
	ReligiousTitleCardinal ReligiousTitle = "cardinal"
	ReligiousTitlePope     ReligiousTitle = "pope"
)

// IsValid reports whether the title is a known value (the empty title included).
func (t ReligiousTitle) IsValid() bool {
	switch t {
	case ReligiousTitleNone, ReligiousTitleBishop, ReligiousTitleCardinal, ReligiousTitlePope:
		return true
	}
	return false
}

// Rank orders the titles: none 0, bishop 1, cardinal 2, pope 3.
func (t ReligiousTitle) Rank() int {
	switch t {
	case ReligiousTitleBishop:
		return 1
	case ReligiousTitleCardinal:
		return 2
	case ReligiousTitlePope:
		return 3
	}
	return 0
}

// ExcommunicationReason tells how a noble was excommunicated.
type ExcommunicationReason string

const (
	// ExcommunicationExOfficio is automatic (a revealed chevalier d'Éon or
	// Sorcière): it is not counted in the pope's limits and cannot be lifted.
	ExcommunicationExOfficio ExcommunicationReason = "ex_officio"
	// ExcommunicationPapal is pronounced by the pope and ends with the pope's
	// death; the pope can lift it.
	ExcommunicationPapal ExcommunicationReason = "papal"
)

// IsValid reports whether the reason is a known value.
func (r ExcommunicationReason) IsValid() bool {
	return r == ExcommunicationExOfficio || r == ExcommunicationPapal
}

// Bishop records the bishop of a bishopric (a region of the map).
type Bishop struct {
	Region RegionID `json:"region"`
	Noble  NobleID  `json:"noble"`
}

// Excommunication is the flag and reason of an excommunicated noble. By is the
// pope who pronounced a papal excommunication and is empty for an ex officio one.
type Excommunication struct {
	Noble  NobleID               `json:"noble"`
	Reason ExcommunicationReason `json:"reason"`
	By     NobleID               `json:"by,omitempty"`
	Turn   int                   `json:"turn"`
}

// BishopOf returns the bishop of the bishopric, if any.
func (g *GameState) BishopOf(region RegionID) (NobleID, bool) {
	for _, bishop := range g.Bishops {
		if bishop.Region == region {
			return bishop.Noble, true
		}
	}
	return "", false
}

// IsBishop reports whether the noble is the bishop of a bishopric.
func (g *GameState) IsBishop(id NobleID) bool {
	return slices.ContainsFunc(g.Bishops, func(bishop Bishop) bool { return bishop.Noble == id })
}

// IsCardinal reports whether the noble is a cardinal.
func (g *GameState) IsCardinal(id NobleID) bool { return slices.Contains(g.Cardinals, id) }

// IsPope reports whether the noble is the pope.
func (g *GameState) IsPope(id NobleID) bool { return g.Pope != nil && *g.Pope == id }

// BishopricOf returns the bishopric the noble is bishop of, if any.
func (g *GameState) BishopricOf(id NobleID) (RegionID, bool) {
	for _, bishop := range g.Bishops {
		if bishop.Noble == id {
			return bishop.Region, true
		}
	}
	return "", false
}

// RegionOfTerritory returns the bishopric a territory belongs to.
func (g *GameState) RegionOfTerritory(id TerritoryID) (Region, bool) {
	for _, region := range g.Regions {
		if slices.Contains(region.Territories, id) {
			return region, true
		}
	}
	return Region{}, false
}

// ExcommunicationOf returns the excommunication of the noble, if any.
func (g *GameState) ExcommunicationOf(id NobleID) (Excommunication, bool) {
	for _, excommunication := range g.Excommunications {
		if excommunication.Noble == id {
			return excommunication, true
		}
	}
	return Excommunication{}, false
}

// ReligiousTitleOf returns the highest religious title the noble holds, whatever
// its standing: the "titre le plus haut" rule for votes.
func (g *GameState) ReligiousTitleOf(id NobleID) ReligiousTitle {
	switch {
	case g.IsPope(id):
		return ReligiousTitlePope
	case g.IsCardinal(id):
		return ReligiousTitleCardinal
	case g.IsBishop(id):
		return ReligiousTitleBishop
	}
	return ReligiousTitleNone
}

// VotingReligiousTitle is the title that counts for the noble's votes and
// powers: its highest title unless the noble is excommunicated or in a dungeon
// (titles are then suspended, not lost), none otherwise.
func (g *GameState) VotingReligiousTitle(id NobleID) ReligiousTitle {
	if _, excommunicated := g.ExcommunicationOf(id); excommunicated {
		return ReligiousTitleNone
	}
	for _, noble := range g.Nobles {
		if noble.ID == id && noble.Status != NobleStatusDungeon {
			return g.ReligiousTitleOf(id)
		}
	}
	return ReligiousTitleNone
}

// CardinalCap is the maximum number of cardinals in play: base plus one more
// per full playersPerExtra players (religion.cardinal_cap_base and
// religion.cardinal_players_per_extra), so 1 below 6 players, 2 below 12, etc.
func CardinalCap(players, base, playersPerExtra int) int {
	if playersPerExtra <= 0 {
		return base
	}
	return base + players/playersPerExtra
}

// DropReligiousTitlesOfMissingNobles clears the religious titles and
// excommunications bound to nobles who left play: a dead noble's titles are
// vacant, an excommunication ends with its subject, and a papal excommunication
// ends with the pope who pronounced it.
func (g *GameState) DropReligiousTitlesOfMissingNobles() {
	living := make(map[NobleID]bool, len(g.Nobles))
	for _, noble := range g.Nobles {
		living[noble.ID] = true
	}
	g.Bishops = slices.DeleteFunc(g.Bishops, func(bishop Bishop) bool { return !living[bishop.Noble] })
	g.Cardinals = slices.DeleteFunc(g.Cardinals, func(id NobleID) bool { return !living[id] })
	if g.Pope != nil && !living[*g.Pope] {
		g.Pope = nil
	}
	g.Excommunications = slices.DeleteFunc(g.Excommunications, func(e Excommunication) bool {
		return !living[e.Noble] || (e.Reason == ExcommunicationPapal && !living[e.By])
	})
}

// validateReligion checks the religious titles and excommunications: one bishop
// per bishopric, a cardinal is a bishop, the pope is a bishop or a cardinal, an
// excommunicated noble holds no title, and every referenced noble is alive.
func (g *GameState) validateReligion(nobles map[NobleID]bool) error {
	regions := make(map[RegionID]bool, len(g.Regions))
	for _, region := range g.Regions {
		regions[region.ID] = true
	}
	bishopRegions := make(map[RegionID]bool, len(g.Bishops))
	bishops := make(map[NobleID]bool, len(g.Bishops))
	for _, bishop := range g.Bishops {
		if !regions[bishop.Region] {
			return fmt.Errorf("models: bishop %q: unknown bishopric %q", bishop.Noble, bishop.Region)
		}
		if !nobles[bishop.Noble] {
			return fmt.Errorf("models: bishopric %q: unknown bishop %q", bishop.Region, bishop.Noble)
		}
		if bishopRegions[bishop.Region] {
			return fmt.Errorf("models: bishopric %q: more than one bishop", bishop.Region)
		}
		if bishops[bishop.Noble] {
			return fmt.Errorf("models: noble %q: bishop of more than one bishopric", bishop.Noble)
		}
		bishopRegions[bishop.Region] = true
		bishops[bishop.Noble] = true
	}
	cardinals := make(map[NobleID]bool, len(g.Cardinals))
	for _, cardinal := range g.Cardinals {
		if !nobles[cardinal] {
			return fmt.Errorf("models: cardinal %q: unknown noble", cardinal)
		}
		if cardinals[cardinal] {
			return fmt.Errorf("models: cardinal %q: duplicate", cardinal)
		}
		if !bishops[cardinal] {
			return fmt.Errorf("models: cardinal %q: must be a bishop", cardinal)
		}
		cardinals[cardinal] = true
	}
	if g.Pope != nil {
		if !nobles[*g.Pope] {
			return fmt.Errorf("models: pope %q: unknown noble", *g.Pope)
		}
		if !bishops[*g.Pope] && !cardinals[*g.Pope] {
			return fmt.Errorf("models: pope %q: must be a bishop or a cardinal", *g.Pope)
		}
	}
	excommunicated := make(map[NobleID]bool, len(g.Excommunications))
	for _, excommunication := range g.Excommunications {
		id := excommunication.Noble
		if !nobles[id] {
			return fmt.Errorf("models: excommunication of %q: unknown noble", id)
		}
		if excommunicated[id] {
			return fmt.Errorf("models: noble %q: excommunicated twice", id)
		}
		excommunicated[id] = true
		if !excommunication.Reason.IsValid() {
			return fmt.Errorf("models: excommunication of %q: invalid reason %q", id, excommunication.Reason)
		}
		if excommunication.Turn < 0 || excommunication.Turn > g.Turn {
			return fmt.Errorf("models: excommunication of %q: turn %d must be between 0 and %d", id, excommunication.Turn, g.Turn)
		}
		switch {
		case excommunication.Reason == ExcommunicationPapal && !nobles[excommunication.By]:
			return fmt.Errorf("models: excommunication of %q: unknown pope %q", id, excommunication.By)
		case excommunication.Reason == ExcommunicationPapal && excommunication.By == id:
			return fmt.Errorf("models: excommunication of %q: a noble cannot excommunicate itself", id)
		case excommunication.Reason == ExcommunicationPapal && (g.Pope == nil || *g.Pope != excommunication.By):
			return fmt.Errorf("models: excommunication of %q: %q is not the pope", id, excommunication.By)
		case excommunication.Reason == ExcommunicationExOfficio && excommunication.By != "":
			return fmt.Errorf("models: excommunication of %q: an ex officio excommunication has no pope", id)
		}
		if bishops[id] || cardinals[id] || (g.Pope != nil && *g.Pope == id) {
			return fmt.Errorf("models: noble %q: excommunicated but still holds a religious title", id)
		}
	}
	return nil
}
