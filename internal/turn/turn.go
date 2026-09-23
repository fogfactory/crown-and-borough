// Package turn owns the turn lifecycle shared by every game backend: the
// hotseat session, the memory store and the Firestore store.
//
// It decides how a player's submission is normalized and validated, which
// players the turn still waits for, how submissions are combined and resolved,
// and when a game is over. It performs no I/O and knows nothing about
// persistence, locking or HTTP; backends keep those concerns and call these
// functions so the rules cannot drift between them.
package turn

import (
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// PrivacyTracker is called immediately after the engine has produced the next
// state and report, while the game is exclusively held by the caller. It is an
// injection point for server-side privacy metadata; the engine itself never
// needs to know about viewers.
type PrivacyTracker func(before, after *models.GameState, input engine.OrdersInput, report engine.TurnReport)

// NormalizeSubmission assigns every order of input to playerID. An order that
// already names another player is rejected, so a client can never submit on
// someone else's behalf. The returned error is an *engine.InputErrors.
func NormalizeSubmission(playerID models.PlayerID, input engine.OrdersInput) (engine.OrdersInput, error) {
	normalized := CloneOrders(input)
	inputErrors := &engine.InputErrors{Errors: []engine.InputError{}}
	foreign := func(noble models.NobleCode, key string, index int, owner models.PlayerID) {
		args := []any{index + 1, owner}
		inputErrors.Errors = append(inputErrors.Errors, engine.InputError{
			Player:      playerID,
			Noble:       noble,
			Code:        "foreign_player_order",
			Message:     i18n.EnglishText(i18n.Message{Key: key, Args: args}),
			MessageKey:  key,
			MessageArgs: args,
		})
	}
	for index := range normalized.Chains {
		chain := &normalized.Chains[index]
		if chain.Player != "" && chain.Player != playerID {
			foreign(chain.Noble, i18n.ErrorForeignChain, index, chain.Player)
			continue
		}
		chain.Player = playerID
	}
	for index := range normalized.Winter {
		winter := &normalized.Winter[index]
		if winter.Player != "" && winter.Player != playerID {
			foreign("", i18n.ErrorForeignWinter, index, winter.Player)
			continue
		}
		winter.Player = playerID
	}
	for index := range normalized.Special {
		special := &normalized.Special[index]
		if special.Player != "" && special.Player != playerID {
			foreign("", i18n.ErrorForeignWinter, index, special.Player)
			continue
		}
		special.Player = playerID
	}
	if len(inputErrors.Errors) != 0 {
		return engine.OrdersInput{}, inputErrors
	}
	return normalized, nil
}

// ValidateSubmission checks one player's normalized submission against the
// current state without keeping any result, so an invalid submission is
// rejected before it is stored.
func ValidateSubmission(state *models.GameState, balance assetgen.Balance, input engine.OrdersInput) error {
	_, err := engine.ResolveTurn(state, balance, input)
	return err
}

// Progress splits the players of state into those who submitted and those the
// turn still waits for. Players who neither submitted nor must submit (see
// engine.PlayerMustSubmit) appear in neither list.
func Progress(state *models.GameState, hasSubmitted func(models.PlayerID) bool) (submitted, remaining []models.PlayerID) {
	submitted = []models.PlayerID{}
	remaining = []models.PlayerID{}
	if state == nil {
		return submitted, remaining
	}
	for _, player := range state.Players {
		switch {
		case hasSubmitted(player.ID):
			submitted = append(submitted, player.ID)
		case engine.PlayerMustSubmit(state, player.ID):
			remaining = append(remaining, player.ID)
		}
	}
	return submitted, remaining
}

// CombineSubmissions merges every stored submission, in player order, into the
// single input resolved by the engine. Players without a submission simply
// contribute no orders.
func CombineSubmissions(state *models.GameState, submissions map[models.PlayerID]engine.OrdersInput) engine.OrdersInput {
	combined := engine.OrdersInput{
		Chains:  []engine.ChainSubmission{},
		Winter:  []engine.WinterSubmission{},
		Special: []engine.DeckSubmission{},
	}
	if state == nil {
		return combined
	}
	for _, player := range state.Players {
		input, ok := submissions[player.ID]
		if !ok {
			continue
		}
		combined.Chains = append(combined.Chains, input.Chains...)
		combined.Winter = append(combined.Winter, input.Winter...)
		combined.Special = append(combined.Special, input.Special...)
	}
	return combined
}

// Resolution is the outcome of resolving one turn from stored submissions.
type Resolution struct {
	Input  engine.OrdersInput
	Report engine.TurnReport
}

// Resolve resolves the current turn from every stored submission. The
// privacy tracker, when set, annotates the next state before it is validated.
// The input state is never mutated.
func Resolve(state *models.GameState, balance assetgen.Balance, submissions map[models.PlayerID]engine.OrdersInput, tracker PrivacyTracker) (Resolution, error) {
	combined := CombineSubmissions(state, submissions)
	report, err := engine.ResolveTurn(state, balance, combined)
	if err != nil {
		return Resolution{}, err
	}
	if tracker != nil {
		tracker(state, report.State, combined, report)
	}
	if err := report.State.Validate(); err != nil {
		return Resolution{}, fmt.Errorf("turn: resolved state is invalid: %w", err)
	}
	return Resolution{Input: combined, Report: report}, nil
}

// Outcome reports whether the game is over and, if so, its winner. A finished
// game without a winner is an exact score tie.
func Outcome(state *models.GameState) (finished bool, winner *models.PlayerID) {
	if !engine.GameFinished(state) {
		return false, nil
	}
	return true, engine.WinnerForFinishedGame(state)
}

// CloneOrders copies every order list of input, including special orders.
func CloneOrders(input engine.OrdersInput) engine.OrdersInput {
	return engine.OrdersInput{
		Chains:  append([]engine.ChainSubmission(nil), input.Chains...),
		Winter:  append([]engine.WinterSubmission(nil), input.Winter...),
		Special: append([]engine.DeckSubmission(nil), input.Special...),
	}
}
