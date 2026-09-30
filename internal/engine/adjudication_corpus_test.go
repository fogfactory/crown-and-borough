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
	"regexp"
	"sort"
	"strconv"
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
// decision graph; the seeds on which the rewrite intentionally differs are
// listed with their reason and their current digests in the divergences
// file. Turns on which the previous adjudicator returned an error are
// recorded as such and not compared.
//
// After a deliberate rule change, -corpus.update records the current results
// as the new reference and empties the divergences list.
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
	adjudicationDivergences  = "testdata/adjudication_corpus.divergences"
	corpusErrorDigest        = "error"
)

func TestAdjudicationCorpus(t *testing.T) {
	if *corpusDump >= 0 {
		state, description, startControl := corpusState(t, *corpusDump)
		for turn := 1; turn <= 2; turn++ {
			resolution, err := resolveFromControl(state, testBalance(), nil, startControl)
			encoded, _ := json.MarshalIndent(resolution, "", "  ")
			t.Logf("seed %d turn %d\n%s\nerr=%v\n%s", *corpusDump, turn, description, err, encoded)
			if err != nil {
				return
			}
			state, description, startControl = resolution.State, "", nil
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
		if *corpusUpdate {
			clearCorpusDivergences(t)
		}
		return
	}

	golden := readCorpusGolden(t)
	for seed, digest := range readCorpusDivergences(t) {
		if seed < len(golden) {
			golden[seed] = digest
		}
	}
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

// readCorpusDivergences returns the current digests of the seeds listed in
// the divergences file, whose lines read "seed category turn1 turn2".
func readCorpusDivergences(t *testing.T) map[int]string {
	t.Helper()
	raw, err := os.ReadFile(adjudicationDivergences)
	if err != nil {
		t.Fatalf("read corpus divergences: %v", err)
	}
	divergences := make(map[int]string)
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		seed, err := strconv.Atoi(fields[0])
		if err != nil || len(fields) < 3 {
			t.Fatalf("malformed corpus divergence %q", line)
		}
		divergences[seed] = strings.Join(fields[2:], " ")
	}
	return divergences
}

// clearCorpusDivergences keeps only the comment header of the divergences
// file once the golden file records the current results.
func clearCorpusDivergences(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(adjudicationDivergences)
	if err != nil {
		t.Fatalf("read corpus divergences: %v", err)
	}
	var header strings.Builder
	for _, line := range strings.SplitAfter(string(raw), "\n") {
		if strings.HasPrefix(line, "#") {
			header.WriteString(line)
		}
	}
	if err := os.WriteFile(adjudicationDivergences, []byte(header.String()), 0o644); err != nil {
		t.Fatalf("write corpus divergences: %v", err)
	}
}

// corpusDigest resolves two consecutive turns from seed and returns their
// digests separated by a space; a turn that errors is recorded as "error" and
// stops the sequence.
func corpusDigest(t *testing.T, seed int64) string {
	t.Helper()
	state, _, startControl := corpusState(t, seed)
	digests := make([]string, 0, 2)
	for turn := 1; turn <= 2; turn++ {
		resolution, err := resolveFromControl(state, testBalance(), nil, startControl)
		if err != nil {
			digests = append(digests, corpusErrorDigest)
			break
		}
		encoded, err := json.Marshal(resolution)
		if err != nil {
			t.Fatalf("seed %d: marshal resolution: %v", seed, err)
		}
		sum := sha256.Sum256(legacyControlJSON(t, encoded, resolution.State))
		digests = append(digests, hex.EncodeToString(sum[:4]))
		state, startControl = resolution.State, nil
		assertControlAnchored(t, seed, turn, state)
	}
	return strings.Join(digests, " ")
}

// territoryStateStart matches the beginning of a marshaled TerritoryState.
var territoryStateStart = regexp.MustCompile(`"([^"]+)":\{"resources":`)

// legacyControlJSON rewrites a marshaled resolution into the shape the golden
// digests were recorded in, when TerritoryState still stored its controller:
// an "owner" field, first, holding the controller derived from state. The
// controller is derived now, so it is not marshaled any more, but the digests
// still pin it, which keeps the corpus a check that the derived control equals
// the control the previous implementation stored after every turn.
func legacyControlJSON(t *testing.T, encoded []byte, state *models.GameState) []byte {
	t.Helper()
	controllers := state.TerritoryControllers()
	return territoryStateStart.ReplaceAllFunc(encoded, func(match []byte) []byte {
		submatch := territoryStateStart.FindSubmatch(match)
		owner := []byte("null")
		if controller, controlled := controllers[models.TerritoryID(submatch[1])]; controlled {
			var err error
			if owner, err = json.Marshal(controller); err != nil {
				t.Fatalf("marshal controller: %v", err)
			}
		}
		return []byte(fmt.Sprintf(`"%s":{"owner":%s,"resources":`, submatch[1], owner))
	})
}

// assertControlAnchored checks #215's invariant on a resolved state: a
// territory has a controller only through a fief membership, the controller's
// own capital, or one of the controller's armies standing on it. Control is
// derived from those three anchors now, so the check has become an agreement
// test between the derivations production uses: the per-territory state
// method (API projection, validation), its bulk form (scores, reports) and
// the engine's index-based read, on a fresh context where the start-of-turn
// snapshot and the current derivation must coincide. It is a free regression
// check across the corpus's ~20 000 resolved turns, on top of the golden
// digests above, which only pin the observable result.
func assertControlAnchored(t *testing.T, seed int64, turn int, state *models.GameState) {
	t.Helper()
	ctx := newResolutionContext(state, testBalance())
	bulk := state.TerritoryControllers()
	for _, territory := range state.Territories {
		territoryID := territory.ID
		want, wantControlled := bulk[territoryID]
		if got, controlled := state.TerritoryController(territoryID); controlled != wantControlled || got != want {
			t.Errorf("seed %d turn %d: territory %q controller %q (%t), bulk derivation says %q (%t)", seed, turn, territoryID, got, controlled, want, wantControlled)
		}
		if got, controlled := ctx.controllerNow(territoryID); controlled != wantControlled || got != want {
			t.Errorf("seed %d turn %d: territory %q engine controller %q (%t), state derivation says %q (%t)", seed, turn, territoryID, got, controlled, want, wantControlled)
		}
		if got, controlled := ctx.controllerAtStart(territoryID); controlled != wantControlled || got != want {
			t.Errorf("seed %d turn %d: territory %q start controller %q (%t), state derivation says %q (%t)", seed, turn, territoryID, got, controlled, want, wantControlled)
		}
		if !wantControlled {
			continue
		}
		if anchorOwner, anchored := ctx.anchorOwner(territoryID); anchored && anchorOwner == want {
			continue
		}
		if army := ctx.currentArmyAt(territoryID); army != nil && army.OwnerID == want {
			continue
		}
		t.Errorf("seed %d turn %d: territory %q controlled by %q without a fief, capital, or army anchor", seed, turn, territoryID, want)
	}
}

// corpusState builds a valid action-turn state from seed. Every order is
// statically valid (adjacent targets, terminal joins), so Resolve reaches the
// adjudicator; supports favour real attacks to exercise the combat table.
// Besides the state, it returns the control the turn starts from: the
// controllers derived from the armies, plus the empty castles that stay held
// by a player without any army or anchor on them, which the state can no longer
// express (see resolveFromControl).
func corpusState(t *testing.T, seed int64) (*models.GameState, string, map[models.TerritoryID]models.PlayerID) {
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
	heldBy := make(map[models.TerritoryID]models.PlayerID)

	var description strings.Builder
	for index, id := range ids {
		switch roll := random.Intn(12); {
		case roll < 2:
			addInfrastructure(state, models.Infrastructure{ID: models.InfraID(fmt.Sprintf("I%d", index)), Type: models.InfraTypeCastle, Level: 1, TerritoryID: id})
			territoryState := state.TerritoryStates[id]
			territoryState.Resources = random.Intn(6)
			if territoryState.Army == nil && random.Intn(2) == 0 {
				heldBy[id] = models.PlayerID(fmt.Sprintf("P%d", 1+random.Intn(3)))
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
		if controller, controlled := heldBy[id]; controlled {
			owner = string(controller)
		} else if controller, controlled := state.TerritoryController(id); controlled {
			owner = string(controller)
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
	startControl := state.TerritoryControllers()
	for id, controller := range heldBy {
		startControl[id] = controller
	}
	return state, description.String(), startControl
}
