package engine

import (
	"fmt"
	"slices"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// Projected victory statuses (specs/titres.md § Lisibilité et simulateur).
// Ongoing means no unit crossed its threshold in the projected state.
const (
	VictoryProjectionMajor   = "major"
	VictoryProjectionMinor   = "minor"
	VictoryProjectionFailure = "failure"
	VictoryProjectionOngoing = "ongoing"
)

// SimulationActionKind names a hypothetical change of the board.
type SimulationActionKind string

const (
	// SimulationKill: Noble dies, together with AlsoKill when it is set (the
	// deaths are settled together, as in one resolution). Their marriage ends,
	// their dignities are lost and the fiefs they hold or are exiled from follow
	// GameState.SettleFiefTitlesOfMissingNobles: a fief goes to the living heir
	// of a claim on its titulaire (a free noble of another house garrisoning
	// the capital usurps it instead, the heir being exiled), else is left vacant
	// with its owner, then returned to the crown when that owner has no noble
	// left. A usurped fief returns to its exile when only the usurper dies, and
	// becomes the usurper's own (or passes to the heir of a claim on the exile as
	// new exile) when only the exile dies; it is vacant when both die. Garrison
	// assumes that noble stands with an army on the capital of the claimed
	// fiefs, NoGarrison that nobody does, whatever the real positions.
	SimulationKill SimulationActionKind = "kill"
	// SimulationMarry: Noble and Other marry.
	SimulationMarry SimulationActionKind = "marry"
	// SimulationDivorce: the marriage of Noble and Other is dissolved.
	SimulationDivorce SimulationActionKind = "divorce"
	// SimulationClaim: Noble, the heir, stakes a claim on Other, a married
	// noble of another house (specs/succession.md § Prétentions).
	SimulationClaim SimulationActionKind = "claim"
	// SimulationRank: the fief with capital Fief rises by one rank, as a
	// winter T R order would (specs/titres.md § Lisibilité et simulateur).
	// Neither the cost nor the once-per-winter limit is simulated, so a fief
	// can rise several ranks in a row.
	SimulationRank SimulationActionKind = "rank"
	// SimulationCrown: Noble is crowned, replacing any king in place; the vacant
	// fiefs of players without a noble return to the new crown player.
	SimulationCrown SimulationActionKind = "crown"
	// SimulationDethrone: the throne becomes vacant while the king stays
	// alive. This is a what-if with no counterpart in play: the tenure is for
	// life and a captured king keeps the crown (specs/politique.md § Mandat,
	// capture et mort), so the throne only empties through a kill.
	SimulationDethrone SimulationActionKind = "dethrone"
	// SimulationConquer: Noble, a free noble of another house, conquers the
	// fief with capital Fief. Noble is assumed to stand with an army on the
	// capital (nothing about armies or positions is simulated). The fief changes
	// hands as when an army with a noble captures its capital: Noble usurps a
	// held fief and its titulaire is exiled, a usurped fief loses its usurper
	// and keeps its exile, a vacant fief passes vacant, and a noble of the
	// exile's house restores the exile.
	SimulationConquer SimulationActionKind = "conquer"
	// SimulationRestore: the house of the exile of the fief with capital Fief
	// retakes the capital with an army: the exile is titulaire again and the
	// usurper loses everything.
	SimulationRestore SimulationActionKind = "restore"
	// SimulationRenounceTitle: Noble gives up the exiled or usurped title it
	// carries on the fief with capital Fief, as a winter T D order would.
	SimulationRenounceTitle SimulationActionKind = "renounce_title"
	// SimulationRenounceClaim: Noble, the heir, gives up their claim, as a
	// winter T C order would. The claim card is not simulated.
	SimulationRenounceClaim SimulationActionKind = "renounce_claim"
)

// SimulationAction is one hypothetical change, nobles given by code.
type SimulationAction struct {
	Kind  SimulationActionKind
	Noble models.NobleCode
	Other models.NobleCode
	// Fief is the capital territory of the fief of a rank, conquer, restore or
	// renounce_title action.
	Fief models.TerritoryID
	// AlsoKill is a second noble who dies with Noble in a kill action.
	AlsoKill models.NobleCode
	// Garrison is a noble assumed to garrison, with an army, the capital of the
	// fiefs a kill passes to the heir of a claim; NoGarrison assumes nobody does.
	// At most one of them is set; with neither, the real armies decide.
	Garrison   models.NobleCode
	NoGarrison bool
}

// VictoryReading is a player's title score, the threshold they are evaluated
// against and the resulting status in one state. Missing is the score still
// to gain to reach Victory.Required, zero once reached.
type VictoryReading struct {
	Score   ScoreBreakdown
	Victory PlayerVictory
	Status  string
	Missing int
}

// VictorySimulation compares every player's reading in the current state and
// in the state obtained by applying the actions in order. State is the
// hypothetical state, ready to be projected for a viewer.
type VictorySimulation struct {
	Current   map[models.PlayerID]VictoryReading
	Projected map[models.PlayerID]VictoryReading
	State     *models.GameState
}

// SimulateVictory applies actions, in order, to a copy of state and reads every
// player's score and victory status before and after. state is not modified.
// An action that cannot happen (unknown noble, already married noble, claim on
// an unmarried noble...) is an error.
func SimulateVictory(state *models.GameState, balance assetgen.Balance, actions []SimulationAction) (VictorySimulation, error) {
	if state == nil {
		return VictorySimulation{}, fmt.Errorf("engine: no state to simulate")
	}
	hypothetical := cloneGameState(state)
	for index, action := range actions {
		if err := applySimulationAction(hypothetical, action); err != nil {
			return VictorySimulation{}, fmt.Errorf("action %d: %w", index+1, err)
		}
	}
	return VictorySimulation{
		Current:   readVictories(state, balance),
		Projected: readVictories(hypothetical, balance),
		State:     hypothetical,
	}, nil
}

func noblePointer(state *models.GameState, code models.NobleCode) (*models.Noble, error) {
	for index := range state.Nobles {
		if models.NobleCode(state.Nobles[index].Code) == code {
			return &state.Nobles[index], nil
		}
	}
	return nil, fmt.Errorf("unknown noble %q", code)
}

func applySimulationAction(state *models.GameState, action SimulationAction) error {
	switch action.Kind {
	case SimulationRank:
		return simulateRank(state, action)
	case SimulationRestore:
		return simulateRestore(state, action)
	case SimulationDethrone:
		if state.King == nil {
			return fmt.Errorf("the throne is already vacant")
		}
		state.King = nil
		state.RefreshQueen()
		return nil
	}
	noble, err := noblePointer(state, action.Noble)
	if err != nil {
		return err
	}
	switch action.Kind {
	case SimulationKill:
		if err := simulateKill(state, *noble, action); err != nil {
			return err
		}
		state.RefreshQueen()
		state.ReturnOrphanFiefsToCrown()
		return nil
	case SimulationConquer:
		return simulateConquer(state, *noble, action)
	case SimulationRenounceTitle:
		return simulateRenounceTitle(state, *noble, action)
	case SimulationRenounceClaim:
		if _, claiming := state.RenounceClaim(noble.ID); !claiming {
			return fmt.Errorf("noble %q holds no claim", noble.Code)
		}
		return nil
	case SimulationCrown:
		if err := simulateCrown(state, *noble); err != nil {
			return err
		}
		state.RefreshQueen()
		state.ReturnOrphanFiefsToCrown()
		return nil
	case SimulationMarry, SimulationDivorce, SimulationClaim:
	default:
		return fmt.Errorf("unknown action %q", action.Kind)
	}
	other, err := noblePointer(state, action.Other)
	if err != nil {
		return err
	}
	if noble.ID == other.ID {
		return fmt.Errorf("noble %q cannot be paired with itself", action.Noble)
	}
	switch action.Kind {
	case SimulationMarry:
		for _, id := range []models.NobleID{noble.ID, other.ID} {
			if _, married := state.MarriageOf(id); married {
				return fmt.Errorf("noble %q is already married", id)
			}
		}
		state.Marriages = append(state.Marriages, models.Marriage{NobleA: noble.ID, NobleB: other.ID, Turn: state.Turn})
	case SimulationDivorce:
		before := len(state.Marriages)
		state.Marriages = slices.DeleteFunc(state.Marriages, func(m models.Marriage) bool {
			return m.Active(state) && m.SpouseOf(noble.ID) == other.ID
		})
		if len(state.Marriages) == before {
			return fmt.Errorf("nobles %q and %q are not married", action.Noble, action.Other)
		}
	case SimulationClaim:
		return simulateClaim(state, *noble, *other)
	}
	state.RefreshQueen()
	return nil
}

// simulateRank raises the fief by one rank. The holder must be able to carry
// the higher title in the succession line; the winter cost and the
// once-per-winter limit are not simulated.
func simulateRank(state *models.GameState, action SimulationAction) error {
	for index := range state.Fiefs {
		fief := &state.Fiefs[index]
		if fief.CapitalTerritoryID != action.Fief {
			continue
		}
		title, ok := fief.Title.Next()
		if !ok {
			return fmt.Errorf("fief %q is already of the highest rank", action.Fief)
		}
		if fief.HolderNobleID == nil {
			return fmt.Errorf("fief %q has no holder", action.Fief)
		}
		if !state.CanReceiveTitle(*fief.HolderNobleID, title) {
			return fmt.Errorf("the holder of fief %q cannot receive the title of %s", action.Fief, title)
		}
		fief.Title = title
		return nil
	}
	return fmt.Errorf("unknown fief %q", action.Fief)
}

func fiefByCapitalIn(state *models.GameState, capital models.TerritoryID) (*models.Fief, error) {
	for index := range state.Fiefs {
		if state.Fiefs[index].CapitalTerritoryID == capital {
			return &state.Fiefs[index], nil
		}
	}
	return nil, fmt.Errorf("unknown fief %q", capital)
}

// simulateConquer hands the fief to the noble's house as an army of theirs
// with that noble on the capital would (GameState.ConquerFief).
func simulateConquer(state *models.GameState, noble models.Noble, action SimulationAction) error {
	fief, err := fiefByCapitalIn(state, action.Fief)
	if err != nil {
		return err
	}
	if noble.Status != models.NobleStatusFree {
		return fmt.Errorf("noble %q is not free", noble.Code)
	}
	if _, conquered := state.ConquerFief(fief, noble.OwnerID, &noble); !conquered {
		return fmt.Errorf("fief %q already belongs to the house of %q", action.Fief, noble.Code)
	}
	return nil
}

// simulateRestore gives the fief back to its exile: the exile's house retakes
// the capital with an army.
func simulateRestore(state *models.GameState, action SimulationAction) error {
	fief, err := fiefByCapitalIn(state, action.Fief)
	if err != nil {
		return err
	}
	if fief.ExiledNobleID == nil {
		return fmt.Errorf("fief %q has no exile", action.Fief)
	}
	var exile *models.Noble
	for index := range state.Nobles {
		if state.Nobles[index].ID == *fief.ExiledNobleID {
			exile = &state.Nobles[index]
		}
	}
	if exile == nil {
		return fmt.Errorf("the exile of fief %q is no longer in play", action.Fief)
	}
	if _, restored := state.ConquerFief(fief, exile.OwnerID, nil); !restored {
		return fmt.Errorf("the exile of fief %q and its usurper belong to the same house", action.Fief)
	}
	return nil
}

// simulateRenounceTitle makes the noble give up its exiled or usurped title on
// the fief (GameState.RenounceContestedTitle).
func simulateRenounceTitle(state *models.GameState, noble models.Noble, action SimulationAction) error {
	fief, err := fiefByCapitalIn(state, action.Fief)
	if err != nil {
		return err
	}
	if kind, _ := state.RenounceContestedTitle(fief, noble.ID); kind == "" {
		return fmt.Errorf("noble %q carries no exiled or usurped title on fief %q", noble.Code, action.Fief)
	}
	return nil
}

// simulateKill validates the optional second death and garrison of a kill
// action, then settles the deaths together.
func simulateKill(state *models.GameState, dead models.Noble, action SimulationAction) error {
	deaths := []models.Noble{dead}
	if action.AlsoKill != "" {
		also, err := noblePointer(state, action.AlsoKill)
		if err != nil {
			return err
		}
		if also.ID == dead.ID {
			return fmt.Errorf("noble %q cannot die twice", action.AlsoKill)
		}
		deaths = append(deaths, *also)
	}
	var garrison *models.Noble
	switch {
	case action.Garrison != "" && action.NoGarrison:
		return fmt.Errorf("a garrison cannot be both assumed and excluded")
	case action.Garrison != "":
		noble, err := noblePointer(state, action.Garrison)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(deaths, func(n models.Noble) bool { return n.ID == noble.ID }) {
			return fmt.Errorf("garrison noble %q dies in the same action", action.Garrison)
		}
		if noble.Status != models.NobleStatusFree {
			return fmt.Errorf("garrison noble %q is not free", action.Garrison)
		}
		// A copy: the nobles slice shifts when the dead are removed.
		copied := *noble
		garrison = &copied
	}
	simulateDeaths(state, deaths, garrison, action.NoGarrison)
	return nil
}

// simulateCrown crowns the noble under the conditions of a king's election.
func simulateCrown(state *models.GameState, noble models.Noble) error {
	switch {
	case state.IsKing(noble.ID):
		return fmt.Errorf("noble %q is already king", noble.Code)
	case !noble.Sex.CanHoldReligiousOrRoyalTitle() || !noble.CanBeKing() || state.IsPope(noble.ID):
		return fmt.Errorf("noble %q cannot be king", noble.Code)
	}
	if _, excommunicated := state.ExcommunicationOf(noble.ID); excommunicated {
		return fmt.Errorf("noble %q is excommunicated", noble.Code)
	}
	id := noble.ID
	state.King = &id
	return nil
}

// simulateClaim stakes heir's claim on target, under the rules of a C order
// except for the card and the placement turn.
func simulateClaim(state *models.GameState, heir, target models.Noble) error {
	if heir.OwnerID == target.OwnerID {
		return fmt.Errorf("claim on a noble of the same house")
	}
	if heir.VoidsClaims() {
		return fmt.Errorf("noble %q cannot claim", heir.Code)
	}
	if _, claiming := state.ClaimOf(heir.ID); claiming {
		return fmt.Errorf("noble %q already holds a claim", heir.Code)
	}
	marriage, married := state.MarriageOf(target.ID)
	if !married {
		return fmt.Errorf("noble %q is not married", target.Code)
	}
	spouseID := marriage.SpouseOf(target.ID)
	var spouse models.Noble
	for _, candidate := range state.Nobles {
		if candidate.ID == spouseID {
			spouse = candidate
		}
	}
	if spouse.OwnerID != heir.OwnerID {
		return fmt.Errorf("noble %q is not married into the house of %q", target.Code, heir.Code)
	}
	wife := target
	if spouse.Sex == models.SexFemale {
		wife = spouse
	}
	state.Claims = append(state.Claims, models.Claim{
		Heir: heir.ID, Target: target.ID, Spouse: spouseID, Turn: state.Turn,
		WifeSide: wife.OwnerID == heir.OwnerID,
	})
	return nil
}

// simulateDeaths removes the nobles from play, settled together: the fiefs
// they hold or are exiled from follow SettleFiefTitlesOfMissingNobles (a fief
// goes to the living heir of the first claim on its titulaire, or to a free
// noble of an army on the capital, who usurps it, the heir being exiled;
// otherwise it stays vacant with its owner; a usurped fief follows its usurper
// and exile); the crown of a king goes to the heir of a claim on the royal
// couple; claims bearing on them are settled. garrison, when set, stands for
// the army of the capital of every claimed fief; noGarrison removes the real
// one.
func simulateDeaths(state *models.GameState, deaths []models.Noble, garrison *models.Noble, noGarrison bool) {
	dead := make(map[models.NobleID]bool, len(deaths))
	for _, noble := range deaths {
		dead[noble.ID] = true
	}
	state.Nobles = slices.DeleteFunc(state.Nobles, func(n models.Noble) bool { return dead[n.ID] })
	usurperOf := func(capital models.TerritoryID, heir models.Noble) *models.Noble {
		switch {
		case noGarrison:
			return nil
		case garrison != nil:
			if garrison.OwnerID == heir.OwnerID {
				return nil
			}
			return garrison
		}
		return state.ClaimUsurperAt(capital, heir.OwnerID)
	}
	served := make(map[models.NobleID]bool)
	livingHeirOf := func(_ *models.Fief, missing models.NobleID) *models.Noble {
		for _, claim := range state.ClaimsOn(missing) {
			for i := range state.Nobles {
				if state.Nobles[i].ID == claim.Heir {
					return &state.Nobles[i]
				}
			}
		}
		return nil
	}
	for _, settlement := range state.SettleFiefTitlesOfMissingNobles(livingHeirOf) {
		fief := settlement.Fief
		switch settlement.Outcome {
		case models.FiefTitleClaimed:
			for _, heir := range state.Nobles {
				if heir.ID != settlement.Heir {
					continue
				}
				if usurper := usurperOf(fief.CapitalTerritoryID, heir); usurper != nil {
					// A garrison noble on the capital usurps; the heir is exiled.
					fief.Usurp(*usurper, heir.ID)
				} else {
					fief.GiveTo(heir)
				}
			}
			served[settlement.Heir] = true
		case models.FiefTitleReexiled:
			served[settlement.Heir] = true
		}
	}
	if crowned, passed := state.PassCrownOfMissingKing(); passed {
		served[crowned] = true
	}
	state.ExpireClaims(func(claim models.Claim) (models.ClaimOutcome, bool) {
		switch {
		case dead[claim.Heir]:
			return models.ClaimLapsed, true
		case served[claim.Heir]:
			return models.ClaimHonoured, true
		case dead[claim.Target]:
			return models.ClaimLapsed, true
		}
		return "", false
	})
	for _, noble := range deaths {
		state.RemovedNobles = append(state.RemovedNobles, models.RemovedNoble{
			ID: noble.ID, Code: noble.Code, Name: noble.Name, Sex: noble.Sex, OwnerID: noble.OwnerID,
			Cause: models.DeathCauseNatural, Turn: state.Turn,
		})
	}
	state.DropReligiousTitlesOfMissingNobles()
}

func readVictories(state *models.GameState, balance assetgen.Balance) map[models.PlayerID]VictoryReading {
	scores := ComputeScores(state, balance)
	goals := ComputeVictoryStatus(state, balance).Players
	var outcome VictoryOutcome
	reached := thresholdReached(state, balance)
	if reached {
		outcome = ResolveVictory(state, balance)
	}
	readings := make(map[models.PlayerID]VictoryReading, len(state.Players))
	for _, player := range state.Players {
		reading := VictoryReading{
			Score:   scores[player.ID],
			Victory: goals[player.ID],
			Status:  VictoryProjectionOngoing,
		}
		reading.Missing = max(reading.Victory.Required-reading.Score.Total, 0)
		switch {
		case !reached:
		case slices.Contains(outcome.Major, player.ID):
			reading.Status = VictoryProjectionMajor
		case outcome.Minor != nil && *outcome.Minor == player.ID:
			reading.Status = VictoryProjectionMinor
		default:
			reading.Status = VictoryProjectionFailure
		}
		readings[player.ID] = reading
	}
	return readings
}
