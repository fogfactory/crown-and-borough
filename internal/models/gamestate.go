package models

import (
	"fmt"
	"slices"
	"strconv"
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
	Armies          []Army                         `json:"armies"`
	Chains          []Chain                        `json:"chains"`
	NextChainID     int                            `json:"nextChainId"`
	NextArmyID      int                            `json:"nextArmyId"`
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
		Armies:          []Army{},
		Chains:          []Chain{},
		NextChainID:     1,
		NextArmyID:      1,
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
	if g.NextChainID < 1 {
		return fmt.Errorf("models: next chain id: must be >= 1, got %d", g.NextChainID)
	}
	if g.NextArmyID < 1 {
		return fmt.Errorf("models: next army id: must be >= 1, got %d", g.NextArmyID)
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

	// 3. Territories: unique ids, unique codes, valid terrain and trigram codes.
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
		if !isCode(t.Code, 3) {
			return fmt.Errorf("models: territory %q: invalid code %q (want exactly 3 uppercase letters)", t.ID, t.Code)
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

	// 5. Armies: unique ids, existing owner and territory, positive size, and
	// a TerritoryState entry for the army's territory. The reverse index check
	// is completed after territory states have validated their pointers.
	armies := make(map[ArmyID]*Army, len(g.Armies))
	for i := range g.Armies {
		army := &g.Armies[i]
		if army.ID == "" {
			return fmt.Errorf("models: army: empty id")
		}
		if _, dup := armies[army.ID]; dup {
			return fmt.Errorf("models: army %q: duplicate id", army.ID)
		}
		if !players[army.OwnerID] {
			return fmt.Errorf("models: army %q: unknown owner %q", army.ID, army.OwnerID)
		}
		if terrs[army.TerritoryID] == nil {
			return fmt.Errorf("models: army %q: unknown territory %q", army.ID, army.TerritoryID)
		}
		if army.Size < 1 {
			return fmt.Errorf("models: army %q: size must be >= 1, got %d", army.ID, army.Size)
		}
		if _, ok := g.TerritoryStates[army.TerritoryID]; !ok {
			return fmt.Errorf("models: army %q: missing TerritoryState for territory %q", army.ID, army.TerritoryID)
		}
		armies[army.ID] = army
	}
	for armyID := range armies {
		if sequence, isAllocatedID := armySequence(armyID); isAllocatedID && sequence >= g.NextArmyID {
			return fmt.Errorf("models: next army id %d must be greater than stored army %q", g.NextArmyID, armyID)
		}
	}

	// 6. Nobles: unique ids, trigram code unique within the game, existing
	// owner and location, a persistent valid status, and an emission record
	// that cannot be in the future.
	nobles := make(map[NobleID]bool, len(g.Nobles))
	nobleOwners := make(map[NobleID]PlayerID, len(g.Nobles))
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
		if !n.Status.IsValid() {
			return fmt.Errorf("models: noble %q: invalid status %q", n.ID, n.Status)
		}
		if n.LastEmissionTurn < 0 || n.LastEmissionTurn > g.Turn {
			return fmt.Errorf("models: noble %q: last emission turn %d must be between 0 and %d", n.ID, n.LastEmissionTurn, g.Turn)
		}
		nobles[n.ID] = true
		nobleOwners[n.ID] = n.OwnerID
	}

	// 7. Chains: unique ids, valid references and complete stored orders. The
	// parser may produce an unassigned chain, but every chain in GameState has
	// already passed reception and must be fully linked to its army.
	chains := make(map[ChainID]*Chain, len(g.Chains))
	pendingArmies := make(map[ArmyID]ChainID, len(g.Chains))
	for i := range g.Chains {
		chain := &g.Chains[i]
		if chain.ID == "" {
			return fmt.Errorf("models: chain: empty id")
		}
		if _, duplicate := chains[chain.ID]; duplicate {
			return fmt.Errorf("models: chain %q: duplicate id", chain.ID)
		}
		if !nobles[chain.NobleID] {
			return fmt.Errorf("models: chain %q: unknown noble %q", chain.ID, chain.NobleID)
		}
		if armies[chain.ArmyID] == nil {
			return fmt.Errorf("models: chain %q: unknown army %q", chain.ID, chain.ArmyID)
		}
		if nobleOwners[chain.NobleID] != armies[chain.ArmyID].OwnerID {
			return fmt.Errorf("models: chain %q: noble %q does not own army %q", chain.ID, chain.NobleID, chain.ArmyID)
		}
		if len(chain.Orders) == 0 {
			return fmt.Errorf("models: chain %q: no orders", chain.ID)
		}
		if chain.CurrentIndex < 0 || chain.CurrentIndex >= len(chain.Orders) {
			return fmt.Errorf("models: chain %q: current index %d is outside its %d orders", chain.ID, chain.CurrentIndex, len(chain.Orders))
		}

		orderIDs := make(map[OrderID]bool, len(chain.Orders))
		for _, order := range chain.Orders {
			if order.ID == "" {
				return fmt.Errorf("models: chain %q: order with empty id", chain.ID)
			}
			if orderIDs[order.ID] {
				return fmt.Errorf("models: chain %q: duplicate order id %q", chain.ID, order.ID)
			}
			orderIDs[order.ID] = true
			if !order.Type.IsValid() {
				return fmt.Errorf("models: chain %q: order %q has invalid type %q", chain.ID, order.ID, order.Type)
			}
			if order.ArmyID != chain.ArmyID {
				return fmt.Errorf("models: chain %q: order %q references army %q instead of %q", chain.ID, order.ID, order.ArmyID, chain.ArmyID)
			}
			if !order.Liaison.IsValid() {
				return fmt.Errorf("models: chain %q: order %q has invalid liaison %q", chain.ID, order.ID, order.Liaison)
			}
			if terrs[order.PositionID] == nil {
				return fmt.Errorf("models: chain %q: order %q references unknown position %q", chain.ID, order.ID, order.PositionID)
			}
			for _, targetID := range order.TargetIDs {
				if terrs[targetID] == nil {
					return fmt.Errorf("models: chain %q: order %q references unknown target %q", chain.ID, order.ID, targetID)
				}
			}
			for _, nobleTargetID := range order.NobleTargetIDs {
				if !nobles[nobleTargetID] {
					return fmt.Errorf("models: chain %q: order %q references unknown noble target %q", chain.ID, order.ID, nobleTargetID)
				}
			}
			for destinationCode, assignedCodes := range order.NobleAssignments {
				if _, exists := codes[string(destinationCode)]; !exists {
					return fmt.Errorf("models: chain %q: order %q references unknown assignment destination %q", chain.ID, order.ID, destinationCode)
				}
				for _, nobleCode := range assignedCodes {
					if nobleCode == "*" {
						continue
					}
					if _, exists := nobleCodes[string(nobleCode)]; !exists {
						return fmt.Errorf("models: chain %q: order %q references unknown assigned noble %q", chain.ID, order.ID, nobleCode)
					}
				}
			}
		}
		if chain.PendingDisperse != nil {
			pending := chain.PendingDisperse
			if chain.Orders[chain.CurrentIndex].Type != OrderTypeDisperse || chain.Orders[chain.CurrentIndex].Liaison != LiaisonModeLoop {
				return fmt.Errorf("models: chain %q: pending dispersion does not match a D order", chain.ID)
			}
			pendingArmy := armies[pending.ArmyID]
			if pendingArmy == nil {
				return fmt.Errorf("models: chain %q: pending dispersion references unknown army %q", chain.ID, pending.ArmyID)
			}
			if terrs[pending.SourceID] == nil || pendingArmy.TerritoryID != pending.SourceID {
				return fmt.Errorf("models: chain %q: pending dispersion army %q is not at source %q", chain.ID, pending.ArmyID, pending.SourceID)
			}
			carrier := armies[chain.ArmyID]
			if carrier == nil || carrier.OwnerID != pendingArmy.OwnerID {
				return fmt.Errorf("models: chain %q: pending dispersion army %q does not share its carrier owner", chain.ID, pending.ArmyID)
			}
			if pending.ArmyID != chain.ArmyID && pendingArmy.ChainID != nil {
				return fmt.Errorf("models: chain %q: pending dispersion army %q already carries chain %q", chain.ID, pending.ArmyID, *pendingArmy.ChainID)
			}
			if existingChainID, exists := pendingArmies[pending.ArmyID]; exists {
				return fmt.Errorf("models: chain %q: pending dispersion army %q is already used by chain %q", chain.ID, pending.ArmyID, existingChainID)
			}
			pendingArmies[pending.ArmyID] = chain.ID
			if len(pending.TargetIDs) == 0 {
				return fmt.Errorf("models: chain %q: pending dispersion has no targets", chain.ID)
			}
			if len(pending.TargetIDs) != pendingArmy.Size {
				return fmt.Errorf("models: chain %q: pending dispersion has %d targets for army %q of size %d", chain.ID, len(pending.TargetIDs), pending.ArmyID, pendingArmy.Size)
			}
			pendingTargets := make(map[TerritoryID]bool, len(pending.TargetIDs))
			for _, targetID := range pending.TargetIDs {
				if terrs[targetID] == nil {
					return fmt.Errorf("models: chain %q: pending dispersion references unknown target %q", chain.ID, targetID)
				}
				if pendingTargets[targetID] {
					return fmt.Errorf("models: chain %q: pending dispersion duplicates target %q", chain.ID, targetID)
				}
				pendingTargets[targetID] = true
			}
			for destinationCode, assignedCodes := range pending.NobleAssignments {
				destinationID, exists := codes[string(destinationCode)]
				if !exists || !pendingTargets[destinationID] {
					return fmt.Errorf("models: chain %q: pending dispersion references invalid assignment destination %q", chain.ID, destinationCode)
				}
				for _, nobleCode := range assignedCodes {
					if _, exists := nobleCodes[string(nobleCode)]; !exists {
						return fmt.Errorf("models: chain %q: pending dispersion references unknown assigned noble %q", chain.ID, nobleCode)
					}
				}
			}
		}
		chains[chain.ID] = chain
	}
	for chainID := range chains {
		if sequence, isAllocatedID := chainSequence(chainID); isAllocatedID && sequence >= g.NextChainID {
			return fmt.Errorf("models: next chain id %d must be greater than stored chain %q", g.NextChainID, chainID)
		}
	}

	// 8. Infrastructures: unique ids, valid type, level >= 1, existing
	// territory, presence in the territory state's infrastructure list.
	// Infrastructures have no owner: they belong to their tile (GDD §3).
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

	// 9. Territory states: exact coverage, valid owners, and a valid army
	// pointer whose territory agrees with the map key. At most one
	// infrastructure is allowed per territory (GDD §3), and resources are
	// non-negative.
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
		if st.Army != nil {
			army := armies[*st.Army]
			if army == nil {
				return fmt.Errorf("models: territoryState %q: unknown army %q", id, *st.Army)
			}
			if army.TerritoryID != id {
				return fmt.Errorf("models: territoryState %q: army %q is stationed in territory %q", id, *st.Army, army.TerritoryID)
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
	for id, army := range armies {
		st := g.TerritoryStates[army.TerritoryID]
		if st.Army == nil || *st.Army != id {
			return fmt.Errorf("models: army %q: territory %q does not reference it", id, army.TerritoryID)
		}
	}
	for id, army := range armies {
		if army.ChainID == nil {
			continue
		}
		chain := chains[*army.ChainID]
		if chain == nil {
			return fmt.Errorf("models: army %q: references unknown chain %q", id, *army.ChainID)
		}
		if chain.ArmyID != id {
			return fmt.Errorf("models: army %q: chain %q belongs to army %q", id, *army.ChainID, chain.ArmyID)
		}
	}
	for id, chain := range chains {
		army := armies[chain.ArmyID]
		if army.ChainID == nil || *army.ChainID != id {
			return fmt.Errorf("models: chain %q: army %q does not reference it", id, chain.ArmyID)
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

func chainSequence(id ChainID) (int, bool) {
	value := string(id)
	if len(value) < 2 || value[0] != 'C' {
		return 0, false
	}
	sequence, err := strconv.Atoi(value[1:])
	if err != nil || sequence < 1 {
		return 0, false
	}
	return sequence, true
}

func armySequence(id ArmyID) (int, bool) {
	value := string(id)
	if len(value) < 2 || value[0] != 'A' {
		return 0, false
	}
	sequence, err := strconv.Atoi(value[1:])
	if err != nil || sequence < 1 {
		return 0, false
	}
	return sequence, true
}
