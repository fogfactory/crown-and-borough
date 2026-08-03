// Package models defines the pure business model of the game engine: the
// static entities (players, territories, nobles, armies, infrastructures), the
// dynamic layer (territory state) and the invariants validated on the whole
// GameState. It is stdlib-only (no web dependency, no ORM): JSON tags drive
// serialization against the map.json and state.json contracts.
package models

// Terrain describes the dominant relief of a territory. It gates messenger
// speed: plains 2 cases/turn, forest and hill 1 case/turn, mountain and swamp
// 0.5 case/turn (GDD §3). Values align with assets/qualificatifs.csv.
type Terrain string

const (
	TerrainPlain    Terrain = "plain"
	TerrainForest   Terrain = "forest"
	TerrainHill     Terrain = "hill"
	TerrainMountain Terrain = "mountain"
	TerrainSwamp    Terrain = "swamp"
)

// IsValid reports whether the terrain is a known value. Note that "any",
// used in the qualificatifs.csv terrain column, is NOT a game Terrain.
func (t Terrain) IsValid() bool {
	switch t {
	case TerrainPlain, TerrainForest, TerrainHill, TerrainMountain, TerrainSwamp:
		return true
	}
	return false
}

// Season is one of the four turns of the yearly loop: three action turns
// (spring, summer, autumn) and one management turn (winter, GDD §2).
type Season string

const (
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
	SeasonWinter Season = "winter"
)

// IsValid reports whether the season is a known value.
func (s Season) IsValid() bool {
	switch s {
	case SeasonSpring, SeasonSummer, SeasonAutumn, SeasonWinter:
		return true
	}
	return false
}

// SeasonForTurn maps an absolute turn tick to its season: turn 1 is spring of
// year 1, turn 4 is winter of year 1, turn 5 is spring of year 2. Purely an
// arithmetic date conversion, not engine logic.
func SeasonForTurn(turn int) Season {
	switch (turn - 1) % 4 {
	case 0:
		return SeasonSpring
	case 1:
		return SeasonSummer
	case 2:
		return SeasonAutumn
	default:
		return SeasonWinter
	}
}

// InfraType identifies an infrastructure. Each type has a distinct effect on
// the economy or information layer (GDD §7): the mill yields +1 R per level,
// the post relay doubles messenger speed, the watchtower grants permanent T0
// vision, the supply depot extends supply-line reach and the castle turns its
// territory into a lieu-dit.
type InfraType string

const (
	InfraTypeMill        InfraType = "mill"
	InfraTypePostRelay   InfraType = "post_relay"
	InfraTypeWatchtower  InfraType = "watchtower"
	InfraTypeSupplyDepot InfraType = "supply_depot"
	InfraTypeCastle      InfraType = "castle"
)

// IsValid reports whether the infra type is a known value.
func (i InfraType) IsValid() bool {
	switch i {
	case InfraTypeMill, InfraTypePostRelay, InfraTypeWatchtower, InfraTypeSupplyDepot, InfraTypeCastle:
		return true
	}
	return false
}
