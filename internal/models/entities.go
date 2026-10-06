package models

// ID types. UUIDs are not needed yet: short, readable identifiers such as
// "ROS", "A1", "N1" identify the entities (architecture §4). Distinct named
// string types keep the domain explicit and prevent cross-type mixups.
type PlayerID string

// SpectatorViewer is used only by server-side projections. It is never a
// player in GameState and must not be persisted as an owner or privacy key.
const SpectatorViewer PlayerID = "*spectator"
const NeutralPlayerID PlayerID = "NEUTRAL"

type TerritoryID string
type ArmyID string
type NobleID string
type InfraID string

// Player is a human participant. CapitalCastleID identifies the castle it
// designates as its stronghold, or is nil when it has no designated capital.
type Player struct {
	ID              PlayerID `json:"id"`
	Name            string   `json:"name"`
	Color           string   `json:"color"`
	CapitalCastleID *InfraID `json:"capitalCastle,omitempty"`
}

// Territory is the static geography of the map: the walkable adjacency graph
// (not every frontier is passable, GDD §3). Geometry (points) is a P1.2
// mapgen concern and does not belong to the model.
type Territory struct {
	ID          TerritoryID   `json:"id"`
	Name        string        `json:"name"`
	Terrain     Terrain       `json:"terrain"`
	Adjacencies []TerritoryID `json:"adjacencies"`
}

// Army is the single force entity stationed on a territory. Size is the number
// of abstract troops in the army. ChainID is nil when the army is Sans Ordre.
// When equal-size armies must be ordered, the territory ID is the lexicographic
// tie-break (GDD §4, §8). Starving is set by the previous turn's famine
// resolution when this army's demand went unmet without an auto-pillage
// rescue (engine.resolveFamine, #208): it fights at strength 0, cannot
// support, grants no noble bonus, and cannot send a transfer for the turn it
// carries, then is recomputed (including cleared) at that same turn's own
// ravitaillement. A dispersed army copies it to every resulting splinter
// (plain struct-copy semantics, like every other field but ChainID and Size);
// a fusion keeps only the surviving host's own value and discards the
// absorbed army's, again with no special-casing needed.
type Army struct {
	ID          ArmyID      `json:"id"`
	OwnerID     PlayerID    `json:"owner"`
	TerritoryID TerritoryID `json:"territory"`
	Size        int         `json:"size"`
	ChainID     *ChainID    `json:"chain"`
	Starving    bool        `json:"starving,omitempty"`
}

// Noble is an immortal, non-combatant entity that emits at most one order
// chain per turn (GDD §6). Its Code is the first-name trigram used to address
// it in orders and reports (GDD §6); codes are unique within a game. The army
// it rides is derived from LocationID because only one army may occupy a
// territory; a noble can deliberately remain alone after that army is lost.
type Noble struct {
	ID               NobleID     `json:"id"`
	Code             string      `json:"code"` // first-name trigram (GDD §6), unique within a game
	Name             string      `json:"name"`
	Sex              Sex         `json:"sex"` // fixed at recruitment (specs/succession.md § Sexe des nobles)
	OwnerID          PlayerID    `json:"owner"`
	LocationID       TerritoryID `json:"location"`
	Status           NobleStatus `json:"status"`
	LastEmissionTurn int         `json:"lastEmissionTurn"`
	// PlacedTurn is the absolute turn the noble was recruited (0 for a
	// starting noble): a claim needs it to fall within one of the noble's
	// parents' marriage (specs/succession.md § Prétentions).
	PlacedTurn int `json:"placedTurn,omitempty"`
	// Dignities are the permanent distinctions the noble carries, conferred
	// by a dignity card (specs/succession.md § Bâtard).
	Dignities []Dignity `json:"dignities,omitempty"`
}

// RemovedNoble is the lineage record of a noble who has permanently left
// play (specs/succession.md § Lignée): it keeps the identity that noble had
// while alive, plus the cause and turn of death, instead of discarding that
// history once the noble leaves GameState.Nobles. Its position in the
// recruitment order stays recoverable from its ID, a global, strictly
// increasing sequence shared with GameState.Nobles. Both its ID and its Code
// stay reserved forever so a later recruit never collides with them (see
// engine.nextNobleID; a noble card is consumed when played, so its code is never drawn again).
type RemovedNoble struct {
	ID      NobleID    `json:"id"`
	Code    string     `json:"code"`
	Name    string     `json:"name"`
	Sex     Sex        `json:"sex"`
	OwnerID PlayerID   `json:"owner"`
	Cause   DeathCause `json:"cause"`
	Turn    int        `json:"turn"`
}

// Infrastructure is a buildable structure. Level is >= 1: a mill yields +1 R
// per level and a castle honours its defensive bonus regardless of its level
// (GDD §7, §8). An infrastructure belongs to its tile, not to a player: there
// is no owner, whoever controls the territory benefits from it. A neutral
// village is simply a village infrastructure on an uncontrolled territory.
// Fortified only ever applies to a village (GDD §8, #193): it keeps its
// InfraType (not a distinct infrastructure) and every village behavior
// (stock, production, income), and additionally gains a castle's defensive
// bonus, with the same auto-capture exception. It is always false for every
// other InfraType.
type Infrastructure struct {
	ID          InfraID     `json:"id"`
	Type        InfraType   `json:"type"`
	Level       int         `json:"level"` // >= 1; mill yields +1 R per level (GDD §7)
	TerritoryID TerritoryID `json:"territory"`
	Fortified   bool        `json:"fortified,omitempty"`
}

// TerritoryState is the dynamic layer attached to a single territory: its
// stock of R, the army stationed on it and the infrastructure built on it.
// Territorial control is not stored: it is derived from the fiefs, the
// players' capitals and the stationed armies (see GameState.TerritoryController).
// A castle construction does not imply control. Army is nil when the territory
// is empty. Infrastructure follows the "Règle de la Structure Unique": at most
// one per territory (GDD §3), which the pillage order then destroys outright
// (GDD §6, §8) — no ordering or choice is ever needed, so the field is a
// pointer rather than a slice. Action-season transfers may leave a positive
// cache on an ordinary territory; winter decides whether that cache survives.
type TerritoryState struct {
	Resources       int      `json:"resources"`
	Army            *ArmyID  `json:"army"`            // nil = no army
	Infrastructures *InfraID `json:"infrastructures"` // nil = no infrastructure (GDD §3)
}
