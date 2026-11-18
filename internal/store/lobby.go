package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// An online game starts in a lobby: its slots are taken one by one, each
// member may name their house and rank candidate seeds, and nothing of the
// board exists (no map, no state, no start placement) until the last slot is
// taken. PlanStart then builds everything at once.

// LobbyEntry is what a member prepared in the lobby. It is private to the
// member: the seed ranking is never shown to anyone else.
type LobbyEntry struct {
	// HouseName is the pending house name; empty means the default.
	HouseName string `json:"houseName,omitempty"`
	// SeedRanking lists candidate indexes from most to least preferred; empty
	// when the member did not vote.
	SeedRanking []int `json:"seedRanking,omitempty"`
}

// LobbyView is the caller's own part of a game still in its lobby.
type LobbyView struct {
	// SeedCandidates are the seeds put to the vote; empty when the creator
	// fixed the seed.
	SeedCandidates []string `json:"seedCandidates"`
	SeedRanking    []int    `json:"seedRanking"`
	HouseName      string   `json:"houseName"`
}

// SeedCandidates derives the seeds put to the vote from the game id, so every
// instance and every member sees the same ones.
func SeedCandidates(id GameID, count int) []string {
	seeds := make([]string, count)
	for index := range seeds {
		digest := sha256.Sum256([]byte(string(id) + "|candidate|" + strconv.Itoa(index)))
		seeds[index] = hex.EncodeToString(digest[:5])
	}
	return seeds
}

// ValidateSeedRanking accepts an empty ranking (no vote) or a permutation of
// the candidate indexes.
func ValidateSeedRanking(ranking []int, count int) error {
	if len(ranking) == 0 {
		return nil
	}
	if len(ranking) != count {
		return ErrInvalidSeedRanking
	}
	seen := make([]bool, count)
	for _, index := range ranking {
		if index < 0 || index >= count || seen[index] {
			return ErrInvalidSeedRanking
		}
		seen[index] = true
	}
	return nil
}

// ResolveSeed counts the votes with a Borda count: a candidate ranked r-th of
// n earns n-1-r points. A tie is broken by a hash of the game id, the same
// whoever voted; with no vote at all the first candidate wins. The result does
// not depend on the order of rankings.
func ResolveSeed(id GameID, candidates []string, rankings [][]int) string {
	if len(candidates) == 0 {
		return ""
	}
	points := make([]int, len(candidates))
	voted := false
	for _, ranking := range rankings {
		if ValidateSeedRanking(ranking, len(candidates)) != nil || len(ranking) == 0 {
			continue
		}
		voted = true
		for rank, index := range ranking {
			points[index] += len(candidates) - 1 - rank
		}
	}
	if !voted {
		return candidates[0]
	}
	best := 0
	var bestHash [32]byte
	for index, score := range points {
		hash := sha256.Sum256([]byte(string(id) + "|tiebreak|" + strconv.Itoa(index)))
		if index == 0 || score > points[best] || (score == points[best] && bytes.Compare(hash[:], bestHash[:]) < 0) {
			best, bestHash = index, hash
		}
	}
	return candidates[best]
}

// StartInput is everything PlanStart needs once the last slot is taken.
type StartInput struct {
	GameID GameID
	// FixedSeed is the seed the creator imposed; empty means the vote decides.
	FixedSeed string
	Incognito bool
	YearCount int
	// Slots are the lobby slots, every one assigned to a member.
	Slots []PlayerSlot
	// Entries holds the lobby entries by member uid.
	Entries map[string]LobbyEntry
	// DisplayNames holds the account display names by member uid.
	DisplayNames map[string]string
	Balance      assetgen.Balance
	Assets       assetgen.Assets
}

// StartPlan is a game's board and houses, drawn when its lobby closes.
type StartPlan struct {
	Seed  string
	Slots []PlayerSlot
	State *models.GameState
	Map   mapgen.MapData
}

// PlanStart resolves the seed, seats the members on the houses and builds the
// map and the state. In an incognito game the members are drawn over the
// houses from the seed; otherwise each keeps the slot they joined.
//
// Names: a house takes the name its member entered in the lobby, else (when
// the game is not incognito) the member's account display name, else the name
// the creator gave the slot, else its placeholder (P1, P2...). A name another
// house already took is skipped. The account display name is recorded in
// PlayerSlot.AccountName outside incognito games only.
func PlanStart(in StartInput) (StartPlan, error) {
	slots := append([]PlayerSlot(nil), in.Slots...)
	count := in.Balance.SeedCandidateCount()
	candidates := SeedCandidates(in.GameID, count)
	seed := strings.TrimSpace(in.FixedSeed)
	if seed == "" {
		rankings := make([][]int, 0, len(slots))
		for _, slot := range slots {
			rankings = append(rankings, in.Entries[slot.ActorID].SeedRanking)
		}
		seed = ResolveSeed(in.GameID, candidates, rankings)
	}
	if in.Incognito {
		SeatIncognitoPlayers(seed, slots)
	}

	taken := make(map[string]bool, len(slots))
	names := make([]string, len(slots))
	claim := func(index int, name string) {
		key := strings.ToLower(name)
		if names[index] == "" && name != "" && !taken[key] {
			names[index] = name
			taken[key] = true
		}
	}
	for index, slot := range slots {
		claim(index, in.Entries[slot.ActorID].HouseName)
	}
	for index := range slots {
		if !in.Incognito {
			claim(index, in.DisplayNames[slots[index].ActorID])
			claim(index, slots[index].Name)
		}
	}
	inits := make([]engine.PlayerInit, len(slots))
	for index := range slots {
		if names[index] == "" {
			names[index] = string(slots[index].ID)
		}
		slots[index].Name = names[index]
		slots[index].NameLocked = false
		slots[index].AccountName = ""
		if !in.Incognito {
			slots[index].AccountName = in.DisplayNames[slots[index].ActorID]
		}
		inits[index] = engine.PlayerInit{ID: slots[index].ID, Name: names[index], Color: slots[index].Color}
	}

	state, err := engine.CreateGameWithYears(seed, inits, in.YearCount, in.Balance, in.Assets)
	if err != nil {
		return StartPlan{}, err
	}
	mapData, err := engine.GenerateMap(seed, len(inits), in.Assets)
	if err != nil {
		return StartPlan{}, fmt.Errorf("store: generate map: %w", err)
	}
	return StartPlan{Seed: seed, Slots: slots, State: state, Map: mapData}, nil
}

// ValidatePendingHouseName normalizes a name entered in the lobby and checks
// it against the names the other members entered.
func ValidatePendingHouseName(name string, others []string) (string, error) {
	normalized, err := NormalizeDisplayName(name)
	if err != nil {
		return "", err
	}
	for _, other := range others {
		if strings.EqualFold(other, normalized) {
			return "", ErrHouseNameTaken
		}
	}
	return normalized, nil
}

// GenerateCandidateMap builds the map candidate seed index of a game would
// have for count houses. Only the map is returned: the start placement is
// not part of the map contract.
func GenerateCandidateMap(id GameID, index, count int, balance assetgen.Balance, assets assetgen.Assets) (mapgen.MapData, error) {
	candidates := SeedCandidates(id, balance.SeedCandidateCount())
	if index < 0 || index >= len(candidates) {
		return mapgen.MapData{}, ErrInvalidCandidate
	}
	return engine.GenerateMap(candidates[index], count, assets)
}
