package engine

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// The adjudication corpus pins the observable result of Resolve on a large
// set of random action turns: every seed builds a small board with attacks,
// supports, holds, joins, dispersions and pillages, and the golden file keeps
// a digest of the full Resolution (state and events) of that turn and of the
// following one, which replays the surviving chains (loops, pending
// dispersions). It was recorded before the adjudicator was rewritten as a
// decision graph, so any divergence from the previous combat semantics is
// caught here. Turns on which the previous adjudicator returned an error are
// recorded as such and not compared.
//
//	go test ./internal/engine -run TestAdjudicationCorpus -corpus.update
//	go test ./internal/engine -run TestAdjudicationCorpus -corpus.dump=1285
var (
	corpusUpdate = flag.Bool("corpus.update", false, "rewrite the adjudication corpus golden file")
	corpusSeeds  = flag.Int("corpus.seeds", adjudicationCorpusSeeds, "number of corpus seeds to resolve")
	corpusOut    = flag.String("corpus.out", "", "write the digests of -corpus.seeds seeds to this file instead of comparing")
	corpusDump   = flag.Int64("corpus.dump", -1, "print the orders and the full resolution of one corpus seed")
)

const (
	adjudicationCorpusSeeds  = 10000
	adjudicationCorpusGolden = "testdata/adjudication_corpus.golden"
	corpusErrorDigest        = "error"
)

func TestAdjudicationCorpus(t *testing.T) {
	if *corpusDump >= 0 {
		state, description := corpusState(t, *corpusDump)
		for turn := 1; turn <= 2; turn++ {
			resolution, err := Resolve(state, testBalance())
			encoded, _ := json.MarshalIndent(resolution, "", "  ")
			t.Logf("seed %d turn %d\n%s\nerr=%v\n%s", *corpusDump, turn, description, err, encoded)
			if err != nil {
				return
			}
			state, description = resolution.State, ""
		}
		return
	}
	digests := make([]string, *corpusSeeds)
	for seed := range digests {
		digests[seed] = corpusDigest(t, int64(seed))
	}
	if *corpusOut != "" || *corpusUpdate {
		path := *corpusOut
		if path == "" {
			path = adjudicationCorpusGolden
		}
		if err := os.WriteFile(path, []byte(strings.Join(digests, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write corpus: %v", err)
		}
		return
	}

	golden := readCorpusGolden(t)
	if len(golden) < len(digests) {
		t.Fatalf("golden corpus has %d seeds, want at least %d", len(golden), len(digests))
	}
	skipped, mismatches := 0, 0
	for seed, digest := range digests {
		want := strings.Fields(golden[seed])
		got := strings.Fields(digest)
		for turn := len(want) - 1; turn >= 0; turn-- {
			if want[turn] == corpusErrorDigest {
				// An error ends the recorded turns: compare only those before it.
				want, got = want[:turn], got[:min(turn, len(got))]
			}
		}
		if len(want) == 0 {
			skipped++
			continue
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			mismatches++
			if mismatches <= 10 {
				t.Errorf("seed %d: resolution digest %s, want %s (inspect with -corpus.dump=%d)", seed, digest, golden[seed], seed)
			}
		}
	}
	if mismatches > 10 {
		t.Errorf("%d seeds differ from the golden corpus", mismatches)
	}
	t.Logf("compared %d seeds, skipped %d recorded errors", len(digests)-skipped, skipped)
}

func readCorpusGolden(t *testing.T) []string {
	t.Helper()
	file, err := os.Open(adjudicationCorpusGolden)
	if err != nil {
		t.Fatalf("open golden corpus: %v", err)
	}
	defer file.Close()
	var digests []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		digests = append(digests, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read golden corpus: %v", err)
	}
	return digests
}

// corpusDigest resolves two consecutive turns from seed and returns their
// digests separated by a space; a turn that errors is recorded as "error" and
// stops the sequence.
func corpusDigest(t *testing.T, seed int64) string {
	t.Helper()
	state, _ := corpusState(t, seed)
	digests := make([]string, 0, 2)
	for turn := 1; turn <= 2; turn++ {
		resolution, err := Resolve(state, testBalance())
		if err != nil {
			digests = append(digests, corpusErrorDigest)
			break
		}
		encoded, err := json.Marshal(resolution)
		if err != nil {
			t.Fatalf("seed %d: marshal resolution: %v", seed, err)
		}
		sum := sha256.Sum256(encoded)
		digests = append(digests, hex.EncodeToString(sum[:4]))
		state = resolution.State
	}
	return strings.Join(digests, " ")
}

// corpusState builds a valid action-turn state from seed. Every order is
// statically valid (adjacent targets, terminal joins), so Resolve reaches the
// adjudicator; supports favour real attacks to exercise the combat table.
func corpusState(t *testing.T, seed int64) (*models.GameState, string) {
	t.Helper()
	random := rand.New(rand.NewSource(seed))
	count := 4 + random.Intn(5)
	ids := make([]models.TerritoryID, count)
	for index := range ids {
		ids[index] = models.TerritoryID(fmt.Sprintf("T%c%c", 'A'+index, 'A'+index))
	}
	adjacent := make(map[models.TerritoryID]map[models.TerritoryID]bool, count)
	link := func(left, right models.TerritoryID) {
		if left == right {
			return
		}
		for _, pair := range [][2]models.TerritoryID{{left, right}, {right, left}} {
			if adjacent[pair[0]] == nil {
				adjacent[pair[0]] = make(map[models.TerritoryID]bool)
			}
			adjacent[pair[0]][pair[1]] = true
		}
	}
	for index := range ids {
		link(ids[index], ids[(index+1)%count])
	}
	for extra := 0; extra < count; extra++ {
		link(ids[random.Intn(count)], ids[random.Intn(count)])
	}
	neighbors := make(map[models.TerritoryID][]models.TerritoryID, count)
	territories := make([]models.Territory, count)
	for index, id := range ids {
		for other := range adjacent[id] {
			neighbors[id] = append(neighbors[id], other)
		}
		sort.Slice(neighbors[id], func(i, j int) bool { return neighbors[id][i] < neighbors[id][j] })
		territories[index] = territory(string(id), string(id), neighbors[id]...)
	}

	var armies []models.Army
	for _, id := range ids {
		if random.Intn(10) < 3 {
			continue
		}
		size := 1 + random.Intn(3)
		if random.Intn(10) == 0 {
			size = 4
		}
		armies = append(armies, models.Army{
			ID:          models.ArmyID(fmt.Sprintf("A%d", len(armies)+1)),
			OwnerID:     models.PlayerID(fmt.Sprintf("P%d", 1+random.Intn(3))),
			TerritoryID: id,
			Size:        size,
		})
	}
	state := testState(t, territories, armies)

	var description strings.Builder
	for index, id := range ids {
		switch roll := random.Intn(12); {
		case roll < 2:
			addInfrastructure(state, models.Infrastructure{ID: models.InfraID(fmt.Sprintf("I%d", index)), Type: models.InfraTypeCastle, Level: 1, TerritoryID: id})
			territoryState := state.TerritoryStates[id]
			territoryState.Resources = random.Intn(6)
			if territoryState.Army == nil && random.Intn(2) == 0 {
				ownerID := models.PlayerID(fmt.Sprintf("P%d", 1+random.Intn(3)))
				territoryState.OwnerID = &ownerID
			}
			state.TerritoryStates[id] = territoryState
		case roll < 5 && state.TerritoryStates[id].Army != nil:
			addInfrastructure(state, models.Infrastructure{ID: models.InfraID(fmt.Sprintf("I%d", index)), Type: models.InfraTypeVillage, Level: 1, TerritoryID: id})
			territoryState := state.TerritoryStates[id]
			territoryState.Resources = 1 + random.Intn(6)
			state.TerritoryStates[id] = territoryState
		}
		territoryState := state.TerritoryStates[id]
		owner := "-"
		if territoryState.OwnerID != nil {
			owner = string(*territoryState.OwnerID)
		}
		infrastructure := "-"
		if territoryState.Infrastructures != nil {
			infrastructure = string(*territoryState.Infrastructures)
		}
		fmt.Fprintf(&description, "  %s adj=%v owner=%s infra=%s stock=%d\n", id, neighbors[id], owner, infrastructure, territoryState.Resources)
	}

	nobleOf := make(map[models.ArmyID]models.NobleID, len(armies))
	for index, army := range armies {
		nobleID := models.NobleID(fmt.Sprintf("N%d", index+1))
		addNoble(state, nobleID, fmt.Sprintf("N%c%c", 'A'+index/26, 'A'+index%26), army.OwnerID, army.TerritoryID)
		if random.Intn(4) == 0 {
			setNobleStatus(state, nobleID, models.NobleStatusHostage)
		}
		nobleOf[army.ID] = nobleID
	}

	armyAt := make(map[models.TerritoryID]models.Army, len(armies))
	for _, army := range armies {
		armyAt[army.TerritoryID] = army
	}
	attacks := make(map[models.TerritoryID]models.TerritoryID)
	orderFor := make(map[models.ArmyID]models.Order, len(armies))
	pick := func(from models.TerritoryID) models.TerritoryID {
		return neighbors[from][random.Intn(len(neighbors[from]))]
	}
	for _, army := range armies {
		roll := random.Intn(100)
		switch {
		case roll < 38:
			target := pick(army.TerritoryID)
			attacks[army.TerritoryID] = target
			orderFor[army.ID] = models.Order{Type: models.OrderTypeAttack, TargetIDs: []models.TerritoryID{target}}
		case roll < 48:
			orderFor[army.ID] = models.Order{Type: models.OrderTypeJoin, TargetIDs: []models.TerritoryID{pick(army.TerritoryID)}}
		case roll < 62:
			targetCount := 1 + random.Intn(army.Size+1)
			targets := make([]models.TerritoryID, 0, targetCount)
			for len(targets) < targetCount {
				if random.Intn(6) == 0 {
					targets = append(targets, army.TerritoryID)
				} else {
					targets = append(targets, pick(army.TerritoryID))
				}
			}
			order := models.Order{Type: models.OrderTypeDisperse, TargetIDs: targets}
			if random.Intn(10) < 7 {
				order.NobleAssignments = map[models.TerritoryID][]models.NobleCode{targets[random.Intn(len(targets))]: {"*"}}
			}
			orderFor[army.ID] = order
		case roll < 66:
			orderFor[army.ID] = models.Order{Type: models.OrderTypePillage}
		case roll < 74:
			orderFor[army.ID] = models.Order{Type: models.OrderTypeHold}
		case roll < 94:
			// Supports are chosen once every attack is known.
		default:
			continue
		}
	}
	for _, army := range armies {
		if _, decided := orderFor[army.ID]; decided || random.Intn(100) >= 83 {
			continue
		}
		var offensive [][2]models.TerritoryID
		for source, target := range attacks {
			if source != army.TerritoryID && target != army.TerritoryID && adjacent[army.TerritoryID][target] {
				offensive = append(offensive, [2]models.TerritoryID{source, target})
			}
		}
		sort.Slice(offensive, func(i, j int) bool {
			return offensive[i][0] < offensive[j][0]
		})
		switch {
		case len(offensive) > 0 && random.Intn(10) < 7:
			pair := offensive[random.Intn(len(offensive))]
			orderFor[army.ID] = models.Order{Type: models.OrderTypeSupport, TargetIDs: []models.TerritoryID{pair[0], pair[1]}}
		case random.Intn(2) == 0:
			orderFor[army.ID] = models.Order{Type: models.OrderTypeSupport, TargetIDs: []models.TerritoryID{pick(army.TerritoryID)}}
		default:
			supported := pick(army.TerritoryID)
			var destinations []models.TerritoryID
			for _, destination := range neighbors[army.TerritoryID] {
				if adjacent[supported][destination] {
					destinations = append(destinations, destination)
				}
			}
			if len(destinations) == 0 {
				orderFor[army.ID] = models.Order{Type: models.OrderTypeHold}
				break
			}
			orderFor[army.ID] = models.Order{Type: models.OrderTypeSupport, TargetIDs: []models.TerritoryID{supported, destinations[random.Intn(len(destinations))]}}
		}
	}
	for _, army := range armies {
		order, exists := orderFor[army.ID]
		if !exists {
			fmt.Fprintf(&description, "  %s(%s,%d)@%s no order\n", army.ID, army.OwnerID, army.Size, army.TerritoryID)
			continue
		}
		order.PositionID = army.TerritoryID
		if random.Intn(4) == 0 {
			order.Liaison = models.LiaisonModeLoop
		}
		orders := []models.Order{order}
		if order.Type != models.OrderTypeJoin && random.Intn(5) == 0 {
			orders = append(orders, models.Order{Type: models.OrderTypeHold, PositionID: army.TerritoryID})
		}
		addChainOrders(t, state, army.ID, nobleOf[army.ID], orders...)
		fmt.Fprintf(&description, "  %s(%s,%d)@%s %s %v %v %s\n", army.ID, army.OwnerID, army.Size, army.TerritoryID, order.Type, order.TargetIDs, order.NobleAssignments, order.Liaison)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("seed %d: invalid corpus state: %v\n%s", seed, err, description.String())
	}
	return state, description.String()
}
