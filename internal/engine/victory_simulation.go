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
	// SimulationKill: Noble dies. Their marriage ends, their dignities are
	// lost and their fiefs pass to the living heir of a claim on them, or are
	// left vacant with their owner.
	SimulationKill SimulationActionKind = "kill"
	// SimulationMarry: Noble and Other marry.
	SimulationMarry SimulationActionKind = "marry"
	// SimulationDivorce: the marriage of Noble and Other is dissolved.
	SimulationDivorce SimulationActionKind = "divorce"
	// SimulationClaim: Noble, the heir, stakes a claim on Other, a married
	// noble of another house (specs/succession.md § Prétentions).
	SimulationClaim SimulationActionKind = "claim"
)

// SimulationAction is one hypothetical change, nobles given by code.
type SimulationAction struct {
	Kind  SimulationActionKind
	Noble models.NobleCode
	Other models.NobleCode
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
	noble, err := noblePointer(state, action.Noble)
	if err != nil {
		return err
	}
	switch action.Kind {
	case SimulationKill:
		simulateDeath(state, *noble)
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

// simulateDeath removes the noble from play: fiefs they hold go to the living
// heir of the first claim on them, otherwise stay vacant with their owner;
// claims bearing on them are settled.
func simulateDeath(state *models.GameState, dead models.Noble) {
	claims := state.ClaimsOn(dead.ID)
	for index := range state.Fiefs {
		fief := &state.Fiefs[index]
		if fief.HolderNobleID == nil || *fief.HolderNobleID != dead.ID {
			continue
		}
		fief.HolderNobleID = nil
		for _, claim := range claims {
			for _, heir := range state.Nobles {
				if heir.ID == claim.Heir {
					heirID := heir.ID
					fief.OwnerID = heir.OwnerID
					fief.HolderNobleID = &heirID
					break
				}
			}
			if fief.HolderNobleID != nil {
				break
			}
		}
	}
	state.Claims = slices.DeleteFunc(state.Claims, func(claim models.Claim) bool {
		return claim.Heir == dead.ID || claim.Target == dead.ID
	})
	state.Nobles = slices.DeleteFunc(state.Nobles, func(n models.Noble) bool { return n.ID == dead.ID })
	state.RemovedNobles = append(state.RemovedNobles, models.RemovedNoble{
		ID: dead.ID, Code: dead.Code, Name: dead.Name, Sex: dead.Sex, OwnerID: dead.OwnerID,
		Cause: models.DeathCauseNatural, Turn: state.Turn,
	})
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
