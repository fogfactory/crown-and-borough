package models

// ID types. UUIDs are not needed yet: short, readable identifiers such as
// "T01", "A1", "N1" identify the entities (architecture §4). Distinct named
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
// mapgen concern and does not belong to the model. IsLieuDit gates resource
// production and storage (GDD §3): only lieu-dits generate and hold R.
type Territory struct {
	ID          TerritoryID   `json:"id"`
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Terrain     Terrain       `json:"terrain"`
	IsLieuDit   bool          `json:"lieuDit"`
	Adjacencies []TerritoryID `json:"adjacencies"`
}

// Army is a unit of force. Matricule is the army number used for tie-breaks:
// famine resolution starves the largest piles and then the highest matricule
// (GDD §8), and the chain-reception rule picks the army with the lowest
// matricule (GDD §4).
type Army struct {
	ID          ArmyID      `json:"id"`
	Matricule   int         `json:"matricule"` // army number; famine tie-break (GDD §8)
	OwnerID     PlayerID    `json:"owner"`
	TerritoryID TerritoryID `json:"territory"`
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
// (GDD §7, §8). Owners keep their construction even on neutral or hostile
// territory: territorial control and construction ownership are independent.
type Infrastructure struct {
	ID          InfraID     `json:"id"`
	Type        InfraType   `json:"type"`
	Level       int         `json:"level"` // >= 1; mill yields +1 R per level (GDD §7)
	OwnerID     PlayerID    `json:"owner"`
	TerritoryID TerritoryID `json:"territory"`
}

// TerritoryState is the dynamic layer attached to a single territory: who
// controls it, its stock of R, the armies stationed on it and the
// infrastructures built on it. OwnerID is nil when the territory is neutral;
// a castle construction does not imply control. Armies list the stationed
// entities. Infrastructures follow the "Règle de la Structure Unique": at
// most one per territory (GDD §3), which the pillage order then destroys
// outright (GDD §6, §8) — no ordering or choice is ever needed.
type TerritoryState struct {
	OwnerID         *PlayerID `json:"owner"` // nil = neutral
	Resources       int       `json:"resources"`
	Armies          []ArmyID  `json:"armies"`
	Infrastructures []InfraID `json:"infrastructures"` // at most one per territory (GDD §3)
}
