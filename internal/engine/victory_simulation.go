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

// MarriageHypothesis is a marriage between two nobles that does not exist yet.
type MarriageHypothesis struct {
	Noble  models.NobleCode
	Spouse models.NobleCode
}

// ClaimHypothesis is a claim honoured: the heir takes the fief titles held by
// the target (specs/succession.md § Prétentions).
type ClaimHypothesis struct {
	Heir   models.NobleCode
	Target models.NobleCode
}

// VictoryScenario is the hypothetical change applied to a copy of the state.
type VictoryScenario struct {
	Marriages []MarriageHypothesis
	Claims    []ClaimHypothesis
}

// VictoryProjection is the score and victory status of one player in the
// hypothetical state, next to the same figures for the current state.
type VictoryProjection struct {
	Current   VictoryReading
	Projected VictoryReading
}

// VictoryReading is the title score, the threshold the player is evaluated
// against and the resulting status in one state. Missing is the score still to
// gain to reach Victory.Required, zero once reached.
type VictoryReading struct {
	Score   ScoreBreakdown
	Victory PlayerVictory
	Status  string
	Missing int
}

// SimulateVictory projects playerID's score and victory status after
// scenario, without touching state. A scenario that cannot happen (unknown
// noble, already married noble, claim on an own noble) is an error.
func SimulateVictory(state *models.GameState, balance assetgen.Balance, playerID models.PlayerID, scenario VictoryScenario) (VictoryProjection, error) {
	if state == nil || !slices.ContainsFunc(state.Players, func(p models.Player) bool { return p.ID == playerID }) {
		return VictoryProjection{}, fmt.Errorf("engine: unknown player %q", playerID)
	}
	hypothetical := cloneGameState(state)
	if err := applyVictoryScenario(hypothetical, playerID, scenario); err != nil {
		return VictoryProjection{}, err
	}
	return VictoryProjection{
		Current:   readVictory(state, balance, playerID),
		Projected: readVictory(hypothetical, balance, playerID),
	}, nil
}

func applyVictoryScenario(state *models.GameState, playerID models.PlayerID, scenario VictoryScenario) error {
	byCode := func(code models.NobleCode) (models.Noble, error) {
		for _, noble := range state.Nobles {
			if models.NobleCode(noble.Code) == code {
				return noble, nil
			}
		}
		return models.Noble{}, fmt.Errorf("unknown noble %q", code)
	}
	for _, hypothesis := range scenario.Marriages {
		noble, err := byCode(hypothesis.Noble)
		if err != nil {
			return err
		}
		spouse, err := byCode(hypothesis.Spouse)
		if err != nil {
			return err
		}
		if noble.ID == spouse.ID {
			return fmt.Errorf("noble %q cannot marry itself", hypothesis.Noble)
		}
		for _, id := range []models.NobleID{noble.ID, spouse.ID} {
			if _, married := state.MarriageOf(id); married {
				return fmt.Errorf("noble %q is already married", id)
			}
		}
		state.Marriages = append(state.Marriages, models.Marriage{NobleA: noble.ID, NobleB: spouse.ID, Turn: state.Turn})
	}
	for _, hypothesis := range scenario.Claims {
		heir, err := byCode(hypothesis.Heir)
		if err != nil {
			return err
		}
		target, err := byCode(hypothesis.Target)
		if err != nil {
			return err
		}
		if heir.OwnerID != playerID {
			return fmt.Errorf("heir %q is not a noble of the player", hypothesis.Heir)
		}
		if target.OwnerID == playerID {
			return fmt.Errorf("target %q is a noble of the player", hypothesis.Target)
		}
		for index := range state.Fiefs {
			fief := &state.Fiefs[index]
			if fief.HolderNobleID != nil && *fief.HolderNobleID == target.ID {
				heirID := heir.ID
				fief.OwnerID = heir.OwnerID
				fief.HolderNobleID = &heirID
			}
		}
	}
	return nil
}

func readVictory(state *models.GameState, balance assetgen.Balance, playerID models.PlayerID) VictoryReading {
	reading := VictoryReading{
		Score:   ComputeScores(state, balance)[playerID],
		Victory: ComputeVictoryStatus(state, balance).Players[playerID],
		Status:  VictoryProjectionOngoing,
	}
	reading.Missing = max(reading.Victory.Required-reading.Score.Total, 0)
	if !thresholdReached(state, balance) {
		return reading
	}
	outcome := ResolveVictory(state, balance)
	switch {
	case slices.Contains(outcome.Major, playerID):
		reading.Status = VictoryProjectionMajor
	case outcome.Minor != nil && *outcome.Minor == playerID:
		reading.Status = VictoryProjectionMinor
	default:
		reading.Status = VictoryProjectionFailure
	}
	return reading
}
