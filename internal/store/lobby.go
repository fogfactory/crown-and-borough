package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
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
// not depend on the order of rankings. Rankings were validated when recorded;
// one of the wrong length (a withdrawn vote included) is skipped.
func ResolveSeed(id GameID, candidates []string, rankings [][]int) string {
	if len(candidates) == 0 {
		return ""
	}
	points := make([]int, len(candidates))
	voted := false
	for _, ranking := range rankings {
		if len(ranking) != len(candidates) {
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
	seedCandidates := SeedCandidates(in.GameID, in.Balance.SeedCandidateCount())
	seed := strings.TrimSpace(in.FixedSeed)
	if seed == "" {
		rankings := make([][]int, 0, len(slots))
		for _, slot := range slots {
			rankings = append(rankings, in.Entries[slot.ActorID].SeedRanking)
		}
		seed = ResolveSeed(in.GameID, seedCandidates, rankings)
	}
	if in.Incognito {
		SeatIncognitoPlayers(seed, slots)
	}

	// Each house has its candidate names in order of preference. Every house's
	// entered name is claimed before any fallback, so a fallback never takes a
	// name another member asked for; a name already claimed is skipped.
	candidates := make([][]string, len(slots))
	for index, slot := range slots {
		candidates[index] = []string{in.Entries[slot.ActorID].HouseName}
		if !in.Incognito {
			candidates[index] = append(candidates[index], in.DisplayNames[slot.ActorID], slot.Name)
		}
	}
	taken := make(map[string]bool, len(slots))
	names := make([]string, len(slots))
	for pass := range 2 {
		for index, list := range candidates {
			entered, fallbacks := list[:1], list[1:]
			for _, name := range [][]string{entered, fallbacks}[pass] {
				key := strings.ToLower(name)
				if names[index] == "" && name != "" && !taken[key] {
					names[index] = name
					taken[key] = true
				}
			}
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

	mapData, err := engine.GenerateMap(seed, len(inits), in.Assets)
	if err != nil {
		return StartPlan{}, fmt.Errorf("store: generate map: %w", err)
	}
	state, err := engine.CreateGameFromMap(seed, mapData, inits, in.YearCount, in.Balance, in.Assets)
	if err != nil {
		return StartPlan{}, err
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
	if nameTaken(normalized, others) {
		return "", ErrHouseNameTaken
	}
	return normalized, nil
}

// nameTaken reports whether name matches one of others, ignoring case.
func nameTaken(name string, others []string) bool {
	return slices.ContainsFunc(others, func(other string) bool { return strings.EqualFold(other, name) })
}

// OtherHouseNames lists the pending house names (by member uid) entered by the
// members other than exceptUID (pass "" to keep every member).
func OtherHouseNames(names map[string]string, exceptUID string) []string {
	var others []string
	for uid, name := range names {
		if uid != exceptUID && name != "" {
			others = append(others, name)
		}
	}
	return others
}

// PendingHouseNames extracts the pending house names of lobby entries by uid.
func PendingHouseNames(entries map[string]LobbyEntry) map[string]string {
	names := make(map[string]string, len(entries))
	for uid, entry := range entries {
		if entry.HouseName != "" {
			names[uid] = entry.HouseName
		}
	}
	return names
}

// JoinStep is a lobby join once validated: what the join changes, before any
// store writes it.
type JoinStep struct {
	// Slots are the lobby slots with the joiner seated.
	Slots []PlayerSlot
	// Seat is the index of the slot the joiner took.
	Seat int
	// Entry is the joiner's validated lobby entry; nil when they prepared
	// nothing.
	Entry *LobbyEntry
	// Final is true when the join takes the last free slot, so the game starts:
	// the store then gathers the lobby entries and display names, and calls
	// PlanStart.
	Final bool
}

// PlanJoin validates a lobby join, the same for every store: the first free
// slot goes to actorID, and what they prepared (house name, seed ranking) is
// checked against the pending names of the other members (pendingNames, by
// member uid) and the number of seed candidates.
func PlanJoin(slots []PlayerSlot, actorID string, options JoinOptions, pendingNames map[string]string, candidateCount int) (JoinStep, error) {
	seat := slices.IndexFunc(slots, func(player PlayerSlot) bool { return !player.Assigned() })
	if seat < 0 {
		return JoinStep{}, ErrGameFull
	}
	var entry LobbyEntry
	if strings.TrimSpace(options.HouseName) != "" {
		name, err := ValidatePendingHouseName(options.HouseName, OtherHouseNames(pendingNames, ""))
		if err != nil {
			return JoinStep{}, err
		}
		entry.HouseName = name
	}
	if err := ValidateSeedRanking(options.SeedRanking, candidateCount); err != nil {
		return JoinStep{}, err
	}
	entry.SeedRanking = append([]int(nil), options.SeedRanking...)
	step := JoinStep{Slots: append([]PlayerSlot(nil), slots...), Seat: seat}
	step.Slots[seat].ActorID = actorID
	step.Final = SlotsFilled(step.Slots)
	if entry.HouseName != "" || len(entry.SeedRanking) > 0 {
		step.Entry = &entry
	}
	return step, nil
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
