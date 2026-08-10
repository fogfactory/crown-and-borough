// Package demo builds the deterministic development state shown by /api/state.
package demo

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	demoTurn       = 5
	minimumPlayers = 2
	maximumPlayers = 16
)

// DemoState creates the static, deterministic mid-game state used by the
// development endpoint. It uses the same seed as map generation while keeping
// each state-building phase independent.
func DemoState(seed string, assets assetgen.Assets, mapData mapgen.MapData, players int) (*models.GameState, error) {
	if players < minimumPlayers || players > maximumPlayers {
		return nil, fmt.Errorf("demo: players must be between %d and %d, got %d", minimumPlayers, maximumPlayers, players)
	}
	if len(mapData.Territories) == 0 {
		return nil, fmt.Errorf("demo: map data has no territories")
	}
	if len(mapData.Territories) < players {
		return nil, fmt.Errorf("demo: map has %d territories for %d players", len(mapData.Territories), players)
	}
	if len(assets.Prenoms) < players {
		return nil, fmt.Errorf("demo: need at least %d first names, got %d", players, len(assets.Prenoms))
	}

	state := &models.GameState{
		ID:              "demo",
		Seed:            seed,
		Turn:            demoTurn,
		Season:          models.SeasonForTurn(demoTurn),
		Players:         make([]models.Player, 0, players),
		Territories:     make([]models.Territory, 0, len(mapData.Territories)),
		Nobles:          make([]models.Noble, 0, players),
		Armies:          make([]models.Army, 0, players*2),
		Chains:          []models.Chain{},
		NextChainID:     1,
		NextArmyID:      1,
		Infrastructures: make([]models.Infrastructure, 0, players+len(mapData.Territories)+3),
		TerritoryStates: make(map[models.TerritoryID]models.TerritoryState, len(mapData.Territories)),
	}

	territoriesByID := make(map[models.TerritoryID]mapgen.Territory, len(mapData.Territories))
	villages := make(map[models.TerritoryID]bool, len(mapData.Territories))
	allTerritoryIDs := make([]models.TerritoryID, 0, len(mapData.Territories))
	villageIDs := make([]models.TerritoryID, 0)
	for _, territory := range mapData.Territories {
		id := models.TerritoryID(territory.ID)
		if id == "" {
			return nil, fmt.Errorf("demo: map territory has an empty id")
		}
		if _, exists := territoriesByID[id]; exists {
			return nil, fmt.Errorf("demo: map has duplicate territory id %q", id)
		}

		adjacencies := make([]models.TerritoryID, len(territory.Adjacencies))
		for index, adjacent := range territory.Adjacencies {
			adjacencies[index] = models.TerritoryID(adjacent)
		}
		state.Territories = append(state.Territories, models.Territory{
			ID:          id,
			Code:        territory.Code,
			Name:        territory.Name,
			Terrain:     territory.Terrain,
			Adjacencies: adjacencies,
		})
		state.TerritoryStates[id] = models.TerritoryState{
			Infrastructures: []models.InfraID{},
		}
		territoriesByID[id] = territory
		allTerritoryIDs = append(allTerritoryIDs, id)
		if territory.Village {
			villages[id] = true
			villageIDs = append(villageIDs, id)
		}
	}

	playerNames := append([]assetgen.Asset(nil), assets.Prenoms...)
	shuffle(newRNG(seed, "players"), playerNames)
	playerColors := [...]string{
		"#a84632", "#2d5f9e", "#7052a1", "#34775c", "#ad7a25",
		"#b3546e", "#1f7a8c", "#7a6b2d", "#c05621", "#4262c0",
		"#8f3b8f", "#5c8a3a", "#96663d", "#3d8fae", "#a64d79", "#6e7f9e",
	}
	for index := 0; index < players; index++ {
		state.Players = append(state.Players, models.Player{
			ID:    models.PlayerID(fmt.Sprintf("P%d", index+1)),
			Name:  playerNames[index].Name,
			Color: playerColors[index],
		})
	}

	starts, err := selectStarts(allTerritoryIDs, villageIDs, territoriesByID, players, newRNG(seed, "starts"))
	if err != nil {
		return nil, err
	}

	controlled := make([][]models.TerritoryID, players)
	occupied := make(map[models.TerritoryID]bool, players*3)
	startSet := make(map[models.TerritoryID]bool, len(starts))
	for index, start := range starts {
		owner := state.Players[index].ID
		if err := setTerritoryOwner(state, start, owner, 3); err != nil {
			return nil, err
		}
		controlled[index] = []models.TerritoryID{start}
		occupied[start] = true
		startSet[start] = true
	}

	controlRNG := newRNG(seed, "control")
	for index, start := range starts {
		target := 2 + controlRNG.IntN(2)
		expansion, err := selectControlTerritories(start, target-1, territoriesByID, villages, occupied, controlRNG)
		if err != nil {
			return nil, fmt.Errorf("demo: player %s: %w", state.Players[index].ID, err)
		}
		for _, territoryID := range expansion {
			if err := setTerritoryOwner(state, territoryID, state.Players[index].ID, 0); err != nil {
				return nil, err
			}
			controlled[index] = append(controlled[index], territoryID)
			occupied[territoryID] = true
		}
	}

	nobleNames, err := selectNobleNames(assets.Prenoms, players, newRNG(seed, "nobles"))
	if err != nil {
		return nil, err
	}

	nextInfrastructureID := 1
	for _, start := range starts {
		if err := addInfrastructure(state, &nextInfrastructureID, models.InfraTypeCastle, start); err != nil {
			return nil, err
		}
	}
	for _, territory := range mapData.Territories {
		territoryID := models.TerritoryID(territory.ID)
		if territory.Village {
			if err := addInfrastructure(state, &nextInfrastructureID, models.InfraTypeVillage, territoryID); err != nil {
				return nil, err
			}
		}
	}

	extraLocations, err := selectInfrastructureLocations(state, mapData.Territories, territoriesByID, startSet, newRNG(seed, "infras"))
	if err != nil {
		return nil, err
	}
	for index, infrastructureType := range []models.InfraType{
		models.InfraTypeMill,
		models.InfraTypeSupplyDepot,
	} {
		if err := addInfrastructure(state, &nextInfrastructureID, infrastructureType, extraLocations[index]); err != nil {
			return nil, err
		}
	}

	secondArmyLocations := make([]models.TerritoryID, players)
	for index, territories := range controlled {
		secondArmyLocations[index] = controlledArmyLocation(
			starts[index],
			territories,
			territoriesByID,
			newRNG(seed, fmt.Sprintf("army-location-%d", index)),
		)
	}

	for index := range controlled {
		owner := state.Players[index].ID
		start := starts[index]
		if err := addArmy(state, owner, start, 1); err != nil {
			return nil, err
		}

		if armyLocation := secondArmyLocations[index]; armyLocation != "" {
			if err := addArmy(state, owner, armyLocation, 2); err != nil {
				return nil, err
			}
		}
	}

	for index, noble := range nobleNames {
		start := starts[index]
		state.Nobles = append(state.Nobles, models.Noble{
			ID:         models.NobleID(fmt.Sprintf("N%d", index+1)),
			Code:       noble.code,
			Name:       fmt.Sprintf("%s de %s", noble.name, territoriesByID[start].Name),
			OwnerID:    state.Players[index].ID,
			LocationID: start,
			Status:     models.NobleStatusFree,
		})
	}

	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("demo: invalid generated state: %w", err)
	}
	return state, nil
}

func selectStarts(
	allTerritoryIDs, villageIDs []models.TerritoryID,
	territoriesByID map[models.TerritoryID]mapgen.Territory,
	players int,
	rng *rand.Rand,
) ([]models.TerritoryID, error) {
	villages := make(map[models.TerritoryID]bool, len(villageIDs))
	for _, territoryID := range villageIDs {
		villages[territoryID] = true
	}

	starts := make([]models.TerritoryID, 0, len(allTerritoryIDs))
	for _, territoryID := range allTerritoryIDs {
		if !villages[territoryID] {
			starts = append(starts, territoryID)
		}
	}
	if len(starts) < players {
		return nil, fmt.Errorf("demo: cannot select %d non-village starting territories from %d", players, len(starts))
	}
	shuffle(rng, starts)

	adjacencies := make(map[models.TerritoryID][]models.TerritoryID, len(territoriesByID))
	for territoryID, territory := range territoriesByID {
		adjacentIDs := make([]models.TerritoryID, len(territory.Adjacencies))
		for index, adjacentID := range territory.Adjacencies {
			adjacentIDs[index] = models.TerritoryID(adjacentID)
		}
		adjacencies[territoryID] = adjacentIDs
	}
	selected, err := engine.SelectStartingTerritories(starts, adjacencies, players)
	if err != nil {
		return nil, fmt.Errorf("demo: select starting territories: %w", err)
	}
	return selected, nil
}

func selectControlTerritories(
	start models.TerritoryID,
	count int,
	territoriesByID map[models.TerritoryID]mapgen.Territory,
	villages, occupied map[models.TerritoryID]bool,
	rng *rand.Rand,
) ([]models.TerritoryID, error) {
	selected := make([]models.TerritoryID, 0, count)
	selectedSet := make(map[models.TerritoryID]bool, count)
	visited := map[models.TerritoryID]bool{start: true}
	frontier := []models.TerritoryID{start}

	for len(frontier) > 0 && len(selected) < count {
		next := make([]models.TerritoryID, 0)
		candidates := make([]models.TerritoryID, 0)
		for _, territoryID := range frontier {
			territory, ok := territoriesByID[territoryID]
			if !ok {
				return nil, fmt.Errorf("unknown starting territory %q", territoryID)
			}
			for _, adjacent := range territory.Adjacencies {
				adjacentID := models.TerritoryID(adjacent)
				if visited[adjacentID] {
					continue
				}
				if _, exists := territoriesByID[adjacentID]; !exists {
					continue
				}
				visited[adjacentID] = true
				next = append(next, adjacentID)
				if !villages[adjacentID] && !occupied[adjacentID] && !selectedSet[adjacentID] {
					candidates = append(candidates, adjacentID)
				}
			}
		}

		shuffle(rng, candidates)
		for _, territoryID := range candidates {
			if len(selected) == count {
				break
			}
			selected = append(selected, territoryID)
			selectedSet[territoryID] = true
		}
		frontier = next
	}
	if len(selected) != count {
		// Small fixture maps may not have enough non-village territory for the
		// requested staging size. Keep the reachable partial expansion so the
		// demo can still exercise its one-army fallback.
		return selected, nil
	}
	return selected, nil
}

type nobleName struct {
	name string
	code string
}

func selectNobleNames(names []assetgen.Asset, players int, rng *rand.Rand) ([]nobleName, error) {
	candidates := append([]assetgen.Asset(nil), names...)
	shuffle(rng, candidates)

	selected := make([]nobleName, 0, players)
	codes := make(map[string]bool, players)
	for _, candidate := range candidates {
		code := nobleCode(candidate.Name)
		if code == "" || codes[code] {
			continue
		}
		selected = append(selected, nobleName{name: candidate.Name, code: code})
		codes[code] = true
		if len(selected) == players {
			return selected, nil
		}
	}
	return nil, fmt.Errorf("demo: cannot select %d first names with distinct noble codes", players)
}

func nobleCode(name string) string {
	code := make([]rune, 0, 3)
	for _, character := range strings.ToUpper(name) {
		character = foldAccent(character)
		if character < 'A' || character > 'Z' {
			continue
		}
		code = append(code, character)
		if len(code) == 3 {
			return string(code)
		}
	}
	return ""
}

func foldAccent(character rune) rune {
	switch character {
	case 'À', 'Á', 'Â', 'Ã', 'Ä', 'Å':
		return 'A'
	case 'Ç':
		return 'C'
	case 'È', 'É', 'Ê', 'Ë':
		return 'E'
	case 'Ì', 'Í', 'Î', 'Ï':
		return 'I'
	case 'Ñ':
		return 'N'
	case 'Ò', 'Ó', 'Ô', 'Õ', 'Ö':
		return 'O'
	case 'Ù', 'Ú', 'Û', 'Ü':
		return 'U'
	case 'Ý', 'Ÿ':
		return 'Y'
	}
	return character
}

func setTerritoryOwner(state *models.GameState, territoryID models.TerritoryID, owner models.PlayerID, resources int) error {
	territoryState, ok := state.TerritoryStates[territoryID]
	if !ok {
		return fmt.Errorf("demo: unknown territory %q", territoryID)
	}
	territoryState.OwnerID = playerIDPointer(owner)
	territoryState.Resources = resources
	state.TerritoryStates[territoryID] = territoryState
	return nil
}

func addInfrastructure(state *models.GameState, nextID *int, infrastructureType models.InfraType, territoryID models.TerritoryID) error {
	territoryState, ok := state.TerritoryStates[territoryID]
	if !ok {
		return fmt.Errorf("demo: unknown territory %q", territoryID)
	}
	if len(territoryState.Infrastructures) != 0 {
		return fmt.Errorf("demo: territory %q already has infrastructure", territoryID)
	}

	id := models.InfraID(fmt.Sprintf("I%d", *nextID))
	state.Infrastructures = append(state.Infrastructures, models.Infrastructure{
		ID:          id,
		Type:        infrastructureType,
		Level:       1,
		TerritoryID: territoryID,
	})
	territoryState.Infrastructures = append(territoryState.Infrastructures, id)
	state.TerritoryStates[territoryID] = territoryState
	*nextID = *nextID + 1
	return nil
}

func selectInfrastructureLocations(
	state *models.GameState,
	territories []mapgen.Territory,
	territoriesByID map[models.TerritoryID]mapgen.Territory,
	starts map[models.TerritoryID]bool,
	rng *rand.Rand,
) ([]models.TerritoryID, error) {
	millPreferred := make([]models.TerritoryID, 0)
	millFallback := make([]models.TerritoryID, 0)
	preferred := make([]models.TerritoryID, 0)
	fallback := make([]models.TerritoryID, 0)
	for _, territory := range territories {
		territoryID := models.TerritoryID(territory.ID)
		territoryState := state.TerritoryStates[territoryID]
		if len(territoryState.Infrastructures) != 0 {
			continue
		}
		isPreferred := territoryState.OwnerID != nil && !starts[territoryID]
		if adjacentToCastle(territoryID, territoriesByID, starts) {
			if isPreferred {
				millPreferred = append(millPreferred, territoryID)
			} else {
				millFallback = append(millFallback, territoryID)
			}
		}
		if isPreferred {
			preferred = append(preferred, territoryID)
			continue
		}
		fallback = append(fallback, territoryID)
	}
	shuffle(rng, millPreferred)
	shuffle(rng, millFallback)
	millCandidates := append(millPreferred, millFallback...)
	if len(millCandidates) == 0 {
		return nil, fmt.Errorf("demo: cannot place a mill adjacent to a castle")
	}
	millLocation := millCandidates[0]

	preferred = withoutTerritory(preferred, millLocation)
	fallback = withoutTerritory(fallback, millLocation)
	shuffle(rng, preferred)
	shuffle(rng, fallback)
	remaining := append(preferred, fallback...)
	if len(remaining) < 2 {
		return nil, fmt.Errorf("demo: need two empty territories after the mill, got %d", len(remaining))
	}
	return append([]models.TerritoryID{millLocation}, remaining[:2]...), nil
}

func adjacentToCastle(
	territoryID models.TerritoryID,
	territoriesByID map[models.TerritoryID]mapgen.Territory,
	starts map[models.TerritoryID]bool,
) bool {
	for _, adjacent := range territoriesByID[territoryID].Adjacencies {
		if starts[models.TerritoryID(adjacent)] {
			return true
		}
	}
	return false
}

func withoutTerritory(territories []models.TerritoryID, excluded models.TerritoryID) []models.TerritoryID {
	filtered := make([]models.TerritoryID, 0, len(territories))
	for _, territoryID := range territories {
		if territoryID != excluded {
			filtered = append(filtered, territoryID)
		}
	}
	return filtered
}

func addArmy(state *models.GameState, owner models.PlayerID, territoryID models.TerritoryID, size int) error {
	territoryState, ok := state.TerritoryStates[territoryID]
	if !ok {
		return fmt.Errorf("demo: unknown territory %q", territoryID)
	}
	if territoryState.Army != nil {
		return fmt.Errorf("demo: territory %q already has an army", territoryID)
	}
	if size < 1 {
		return fmt.Errorf("demo: army size must be >= 1, got %d", size)
	}

	id := models.ArmyID(fmt.Sprintf("A%d", state.NextArmyID))
	state.Armies = append(state.Armies, models.Army{
		ID:          id,
		OwnerID:     owner,
		TerritoryID: territoryID,
		Size:        size,
	})
	territoryState.Army = &id
	state.TerritoryStates[territoryID] = territoryState
	state.NextArmyID++
	return nil
}

func controlledArmyLocation(
	start models.TerritoryID,
	controlled []models.TerritoryID,
	territoriesByID map[models.TerritoryID]mapgen.Territory,
	rng *rand.Rand,
) models.TerritoryID {
	if len(controlled) < 2 {
		return ""
	}
	controlledSet := make(map[models.TerritoryID]bool, len(controlled)-1)
	for _, territoryID := range controlled[1:] {
		controlledSet[territoryID] = true
	}
	candidates := make([]models.TerritoryID, 0, len(controlledSet))
	for _, adjacent := range territoriesByID[start].Adjacencies {
		if territoryID := models.TerritoryID(adjacent); controlledSet[territoryID] {
			candidates = append(candidates, territoryID)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	shuffle(rng, candidates)
	return candidates[0]
}

func playerIDPointer(id models.PlayerID) *models.PlayerID {
	copy := id
	return &copy
}

// newRNG deliberately mirrors internal/engine/mapgen/rng.go. The generator is
// duplicated because mapgen's phase helper is intentionally not exported.
func newRNG(seed, phase string) *rand.Rand {
	digest := sha256.Sum256([]byte(seed + "|" + phase))
	lo := binary.BigEndian.Uint64(digest[:8])
	hi := binary.BigEndian.Uint64(digest[8:16])
	return rand.New(rand.NewPCG(lo, hi))
}

func shuffle[T any](rng *rand.Rand, values []T) {
	for index := len(values) - 1; index > 0; index-- {
		swap := rng.IntN(index + 1)
		values[index], values[swap] = values[swap], values[index]
	}
}
