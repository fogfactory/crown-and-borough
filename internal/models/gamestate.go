package models

import (
	"fmt"
	"slices"
)

// GameState is the whole game: the immutable world layout plus the evolving
// dynamic state. ID identifies the game, Seed drives map generation (P1.2).
// Turn is the absolute tick (+1 every season, winter included) and Year is
// derived from it: (Turn-1)/4 + 1 (GDD §2). TerritoryStates holds exactly one
// entry per territory (enforced by Validate), so the engine never looks up a
// missing state.
type GameState struct {
	ID              string                         `json:"id"`
	Seed            string                         `json:"seed"`
	Turn            int                            `json:"turn"`
	Season          Season                         `json:"season"`
	Players         []Player                       `json:"players"`
	Territories     []Territory                    `json:"territories"`
	Nobles          []Noble                        `json:"nobles"`
	Troops          []Troop                        `json:"troops"`
	Infrastructures []Infrastructure               `json:"infrastructures"`
	TerritoryStates map[TerritoryID]TerritoryState `json:"territoryStates"`
}

// NewGameState returns a fresh empty state at turn 1, spring of year 1, with
// non-nil collections so that serialization yields [] and {} rather than null.
// Game creation (NewGame) is P1.2 and is deliberately out of scope here.
func NewGameState() *GameState {
	return &GameState{
		Turn:            1,
		Season:          SeasonForTurn(1),
		Players:         []Player{},
		Territories:     []Territory{},
		Nobles:          []Noble{},
		Troops:          []Troop{},
		Infrastructures: []Infrastructure{},
		TerritoryStates: map[TerritoryID]TerritoryState{},
	}
}

// Year returns the game year for the current turn: turn 1..4 are year 1,
// turn 5..8 are year 2, and so on (GDD §2).
func (g *GameState) Year() int {
	return (g.Turn-1)/4 + 1
}

// Validate checks every invariant of the state and returns the first problem
// found, fail-fast, with errors prefixed "models: ". It is the single entry
// point for semantic validity: structural construction mistakes (asymmetric
// adjacencies, dangling references, double-indexing drift between an entity
// and its TerritoryState entry) surface here rather than in per-type JSON
// unmarshalling.
func (g *GameState) Validate() error {
	if g == nil {
		return fmt.Errorf("models: gamestate: nil state")
	}

	// 1. Turn and season.
	if g.Turn < 1 {
		return fmt.Errorf("models: turn: must be >= 1, got %d", g.Turn)
	}
	if !g.Season.IsValid() {
		return fmt.Errorf("models: season: invalid season %q", g.Season)
	}
	if want := SeasonForTurn(g.Turn); g.Season != want {
		return fmt.Errorf("models: season: %q does not match turn %d (want %q)", g.Season, g.Turn, want)
	}

	// 2. Players: unique ids.
	players := make(map[PlayerID]bool, len(g.Players))
	for _, p := range g.Players {
		if p.ID == "" {
			return fmt.Errorf("models: player: empty id")
		}
		if players[p.ID] {
			return fmt.Errorf("models: player %q: duplicate id", p.ID)
		}
		players[p.ID] = true
	}

	// 3. Territories: unique ids, unique codes, valid terrain, code shape
	// dictated by IsLieuDit (3 uppercase letters for a lieu-dit, 4 otherwise).
	terrs := make(map[TerritoryID]*Territory, len(g.Territories))
	codes := make(map[string]TerritoryID, len(g.Territories))
	for i := range g.Territories {
		t := &g.Territories[i]
		if t.ID == "" {
			return fmt.Errorf("models: territory: empty id")
		}
		if _, dup := terrs[t.ID]; dup {
			return fmt.Errorf("models: territory %q: duplicate id", t.ID)
		}
		if prev, dup := codes[t.Code]; dup {
			return fmt.Errorf("models: territory %q: duplicate code %q (already used by %q)", t.ID, t.Code, prev)
		}
		if !t.Terrain.IsValid() {
			return fmt.Errorf("models: territory %q: invalid terrain %q", t.ID, t.Terrain)
		}
		if t.IsLieuDit {
			if !isCode(t.Code, 3) {
				return fmt.Errorf("models: territory %q: invalid lieu-dit code %q (want exactly 3 uppercase letters)", t.ID, t.Code)
			}
		} else if !isCode(t.Code, 4) {
			return fmt.Errorf("models: territory %q: invalid code %q (want exactly 4 uppercase letters)", t.ID, t.Code)
		}
		terrs[t.ID] = t
		codes[t.Code] = t.ID
	}

	// 4. Adjacencies: existing targets, no self-adjacency, no duplicate edge,
	// strict symmetry (A -> B implies B -> A).
	for i := range g.Territories {
		t := &g.Territories[i]
		seen := make(map[TerritoryID]bool, len(t.Adjacencies))
		for _, a := range t.Adjacencies {
			if a == t.ID {
				return fmt.Errorf("models: territory %q: self-adjacency", t.ID)
			}
			if seen[a] {
				return fmt.Errorf("models: territory %q: duplicate adjacency %q", t.ID, a)
			}
			seen[a] = true
			at := terrs[a]
			if at == nil {
				return fmt.Errorf("models: territory %q: adjacency %q does not exist", t.ID, a)
			}
			if !slices.Contains(at.Adjacencies, t.ID) {
				return fmt.Errorf("models: territory %q: asymmetric adjacency: %q does not list it back", t.ID, a)
			}
		}
	}

	// 5. Troops: unique ids, matricule unique per owner, existing owner and
	// territory, and presence in the territory state's troop list.
	troops := make(map[TroopID]*Troop, len(g.Troops))
	matricules := make(map[PlayerID]map[int]bool)
	for i := range g.Troops {
		t := &g.Troops[i]
		if t.ID == "" {
			return fmt.Errorf("models: troop: empty id")
		}
		if _, dup := troops[t.ID]; dup {
			return fmt.Errorf("models: troop %q: duplicate id", t.ID)
		}
		if !players[t.OwnerID] {
			return fmt.Errorf("models: troop %q: unknown owner %q", t.ID, t.OwnerID)
		}
		if terrs[t.TerritoryID] == nil {
			return fmt.Errorf("models: troop %q: unknown territory %q", t.ID, t.TerritoryID)
		}
		seen, ok := matricules[t.OwnerID]
		if !ok {
			seen = make(map[int]bool)
			matricules[t.OwnerID] = seen
		}
		if seen[t.Matricule] {
			return fmt.Errorf("models: troop %q: duplicate matricule %d for owner %q", t.ID, t.Matricule, t.OwnerID)
		}
		seen[t.Matricule] = true
		st, ok := g.TerritoryStates[t.TerritoryID]
		if !ok {
			return fmt.Errorf("models: troop %q: missing TerritoryState for territory %q", t.ID, t.TerritoryID)
		}
		if !slices.Contains(st.Troops, t.ID) {
			return fmt.Errorf("models: troop %q: territory %q does not list it in its troops", t.ID, t.TerritoryID)
		}
		troops[t.ID] = t
	}

	// 6. Nobles: unique ids, trigram code unique within the game, existing
	// owner and location.
	nobles := make(map[NobleID]bool, len(g.Nobles))
	nobleCodes := make(map[string]NobleID, len(g.Nobles))
	for i := range g.Nobles {
		n := &g.Nobles[i]
		if n.ID == "" {
			return fmt.Errorf("models: noble: empty id")
		}
		if nobles[n.ID] {
			return fmt.Errorf("models: noble %q: duplicate id", n.ID)
		}
		if !isCode(n.Code, 3) {
			return fmt.Errorf("models: noble %q: invalid code %q (want exactly 3 uppercase letters)", n.ID, n.Code)
		}
		if prev, dup := nobleCodes[n.Code]; dup {
			return fmt.Errorf("models: noble %q: duplicate code %q (already used by %q)", n.ID, n.Code, prev)
		}
		nobleCodes[n.Code] = n.ID
		if !players[n.OwnerID] {
			return fmt.Errorf("models: noble %q: unknown owner %q", n.ID, n.OwnerID)
		}
		if terrs[n.LocationID] == nil {
			return fmt.Errorf("models: noble %q: unknown territory %q", n.ID, n.LocationID)
		}
		nobles[n.ID] = true
	}

	// 7. Infrastructures: unique ids, valid type, level >= 1, existing owner
	// and territory, presence in the territory state's infrastructure list.
	infras := make(map[InfraID]*Infrastructure, len(g.Infrastructures))
	for i := range g.Infrastructures {
		in := &g.Infrastructures[i]
		if in.ID == "" {
			return fmt.Errorf("models: infrastructure: empty id")
		}
		if _, dup := infras[in.ID]; dup {
			return fmt.Errorf("models: infrastructure %q: duplicate id", in.ID)
		}
		if !in.Type.IsValid() {
			return fmt.Errorf("models: infrastructure %q: invalid type %q", in.ID, in.Type)
		}
		if in.Level < 1 {
			return fmt.Errorf("models: infrastructure %q: level must be >= 1, got %d", in.ID, in.Level)
		}
		if !players[in.OwnerID] {
			return fmt.Errorf("models: infrastructure %q: unknown owner %q", in.ID, in.OwnerID)
		}
		if terrs[in.TerritoryID] == nil {
			return fmt.Errorf("models: infrastructure %q: unknown territory %q", in.ID, in.TerritoryID)
		}
		st, ok := g.TerritoryStates[in.TerritoryID]
		if !ok {
			return fmt.Errorf("models: infrastructure %q: missing TerritoryState for territory %q", in.ID, in.TerritoryID)
		}
		if !slices.Contains(st.Infrastructures, in.ID) {
			return fmt.Errorf("models: infrastructure %q: territory %q does not list it in its infrastructures", in.ID, in.TerritoryID)
		}
		infras[in.ID] = in
	}

	// 8. Territory states: exact coverage, valid owners, existing and
	// consistent entity ids in both directions, no duplicate troop id within
	// a list, at most one infrastructure per territory (GDD §3), non-negative
	// resources.
	for id, st := range g.TerritoryStates {
		if terrs[id] == nil {
			return fmt.Errorf("models: territoryState: %q references unknown territory", id)
		}
		if st.Resources < 0 {
			return fmt.Errorf("models: territoryState %q: negative resources %d", id, st.Resources)
		}
		if st.OwnerID != nil && !players[*st.OwnerID] {
			return fmt.Errorf("models: territoryState %q: unknown owner %q", id, *st.OwnerID)
		}
		seenTroops := make(map[TroopID]bool, len(st.Troops))
		for _, tid := range st.Troops {
			if seenTroops[tid] {
				return fmt.Errorf("models: territoryState %q: duplicate troop %q", id, tid)
			}
			seenTroops[tid] = true
			t := troops[tid]
			if t == nil {
				return fmt.Errorf("models: territoryState %q: unknown troop %q", id, tid)
			}
			if t.TerritoryID != id {
				return fmt.Errorf("models: territoryState %q: troop %q is stationed in territory %q", id, tid, t.TerritoryID)
			}
		}
		if len(st.Infrastructures) > 1 {
			return fmt.Errorf("models: territoryState %q: multiple infrastructures (want at most one per territory)", id)
		}
		for _, iid := range st.Infrastructures {
			in := infras[iid]
			if in == nil {
				return fmt.Errorf("models: territoryState %q: unknown infrastructure %q", id, iid)
			}
			if in.TerritoryID != id {
				return fmt.Errorf("models: territoryState %q: infrastructure %q is built in territory %q", id, iid, in.TerritoryID)
			}
		}
	}
	for id := range terrs {
		if _, ok := g.TerritoryStates[id]; !ok {
			return fmt.Errorf("models: territory %q: missing TerritoryState entry", id)
		}
	}
	return nil
}

// isCode reports whether s contains exactly n uppercase ASCII letters.
func isCode(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
