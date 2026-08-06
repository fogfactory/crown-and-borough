package models

// ID types. UUIDs are not needed yet: short, readable identifiers such as
// "T01", "A1", "N1" identify the entities (architecture §4). Army IDs use a
// global A-prefix counter; T is reserved for territories. Distinct named
// string types keep the domain explicit and prevent cross-type mixups.
type PlayerID string
type TerritoryID string
type ArmyID string
type NobleID string
type InfraID string

// Player is a human participant. Its capital — the castle it designates as its
// stronghold (CapitalCastleID) — arrives in P1.6 and is not modelled here.
type Player struct {
	ID    PlayerID `json:"id"`
	Name  string   `json:"name"`
	Color string   `json:"color"`
}

// Territory is the static geography of the map: the walkable adjacency graph
// (not every frontier is passable, GDD §3). Geometry (points) is a P1.2
// mapgen concern and does not belong to the model.
type Territory struct {
	ID          TerritoryID   `json:"id"`
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Terrain     Terrain       `json:"terrain"`
	Adjacencies []TerritoryID `json:"adjacencies"`
}

// Army is the single force entity stationed on a territory. Size is the
// number of units in the army. When equal-size armies must be ordered, the
// territory code is the lexicographic tie-break (GDD §4, §8).
type Army struct {
	ID          ArmyID      `json:"id"`
	OwnerID     PlayerID    `json:"owner"`
	TerritoryID TerritoryID `json:"territory"`
	Size        int         `json:"size"`
}

// Noble is an immortal, non-combatant entity that emits at most one order
// chain per turn (GDD §6). Its Code is the first-name trigram used to address
// it in orders and reports (GDD §6); codes are unique within a game.
type Noble struct {
	ID         NobleID     `json:"id"`
	Code       string      `json:"code"` // first-name trigram (GDD §6), unique within a game
	Name       string      `json:"name"`
	OwnerID    PlayerID    `json:"owner"`
	LocationID TerritoryID `json:"location"`
}

// Infrastructure is a buildable structure. Level is >= 1: a mill yields +1 R
// per level and a castle honours its defensive bonus regardless of its level
// (GDD §7, §8). An infrastructure belongs to its tile, not to a player: there
// is no owner, whoever controls the territory benefits from it. A neutral
// village is simply a village infrastructure on an uncontrolled territory
// (its owner field no longer exists).
type Infrastructure struct {
	ID          InfraID     `json:"id"`
	Type        InfraType   `json:"type"`
	Level       int         `json:"level"` // >= 1; mill yields +1 R per level (GDD §7)
	TerritoryID TerritoryID `json:"territory"`
}

// TerritoryState is the dynamic layer attached to a single territory: who
// controls it, its stock of R, the army stationed on it and the
// infrastructures built on it. OwnerID is nil when the territory is neutral;
// a castle construction does not imply control. Army is nil when the territory
// is empty. Infrastructures follow the "Règle de la Structure Unique": at most
// one per territory (GDD §3), which the pillage order then destroys outright
// (GDD §6, §8) — no ordering or choice is ever needed.
type TerritoryState struct {
	OwnerID         *PlayerID `json:"owner"` // nil = neutral
	Resources       int       `json:"resources"`
	Army            *ArmyID   `json:"army"`            // nil = no army
	Infrastructures []InfraID `json:"infrastructures"` // at most one per territory (GDD §3)
}
