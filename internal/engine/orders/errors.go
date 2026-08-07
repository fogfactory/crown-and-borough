package orders

import (
	"errors"
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

const (
	// ParseCodeUnknownSymbol identifies an unsupported order symbol.
	ParseCodeUnknownSymbol = "unknown_symbol"
	// ParseCodeInvalidCode identifies a malformed or unknown territory or noble code.
	ParseCodeInvalidCode = "invalid_code"
	// ParseCodeMissingTarget identifies an order missing a required target.
	ParseCodeMissingTarget = "missing_target"
	// ParseCodeTooManyTargets identifies an order with extra targets.
	ParseCodeTooManyTargets = "too_many_targets"
	// ParseCodeBadHeader identifies a malformed first non-comment line.
	ParseCodeBadHeader = "bad_header"
	// ParseCodeNoHeader identifies input without a non-comment header line.
	ParseCodeNoHeader = "no_header"
	// ParseCodeNobleNotFound identifies a syntactically valid unknown header noble.
	ParseCodeNobleNotFound = "noble_not_found"
	// ParseCodeUnclosedParenthesis identifies malformed order-line parentheses.
	ParseCodeUnclosedParenthesis = "unclosed_parenthesis"
)

var (
	// ErrNoArmyOnPosition reports that no army occupies the receiving position.
	ErrNoArmyOnPosition = errors.New("no_army_on_position")
	// ErrArmyNotOwned reports that the army at the receiving position has another owner.
	ErrArmyNotOwned = errors.New("army_not_owned")
	// ErrNoblePrisoner reports that a hostage or dungeon noble attempted to emit.
	ErrNoblePrisoner = errors.New("noble_prisoner")
	// ErrEmissionCapacity reports a second emission by one noble in the same turn.
	ErrEmissionCapacity = errors.New("emission_capacity")
	// ErrDisperseSize reports a D order whose destinations do not match army size.
	ErrDisperseSize = errors.New("disperse_size")
	// ErrNoblesNotCovered reports invalid dispersion assignments for co-located nobles.
	ErrNoblesNotCovered = errors.New("nobles_not_covered")
	// ErrNobleNotPrisoner reports an immediate O or K target not held by the army.
	ErrNobleNotPrisoner = errors.New("noble_not_prisoner")
	// ErrInvalidChain reports a malformed or statically invalid submitted chain.
	ErrInvalidChain = errors.New("invalid_chain")
)

// ParseError describes one source-text error. Line is the original one-based
// line number, or zero when no input line could be identified.
type ParseError struct {
	Line    int
	Code    string
	Message string
}

// Error returns a compact, human-readable representation of a parser error.
func (e ParseError) Error() string {
	if e.Line == 0 {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("line %d: %s: %s", e.Line, e.Code, e.Message)
}

// ValidationCodeNotAdjacent is the static adjacency diagnostic. It is
// deferrable: reception keeps the chain and P1.4 revalidates the order
// against the real army position, breaking the chain at that order.
const ValidationCodeNotAdjacent = "not_adjacent"

// ValidationError describes an intrinsic chain validation error. OrderID is
// empty only when the error applies to the chain header rather than one order.
type ValidationError struct {
	OrderID models.OrderID
	Code    string
	Message string
}

// Error returns a compact, human-readable representation of a validation error.
func (e ValidationError) Error() string {
	if e.OrderID == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("order %s: %s: %s", e.OrderID, e.Code, e.Message)
}

// Deferrable reports whether the validation error is deferred to execution
// time instead of rejecting the chain at reception. Only static non-adjacency
// is deferrable: the order is revalidated during P1.4 resolution, which
// invalidates it and breaks the chain there.
func (e ValidationError) Deferrable() bool {
	return e.Code == ValidationCodeNotAdjacent
}

// AssignmentError describes a reception failure. It unwraps to a stable
// sentinel so callers can classify it with errors.Is.
type AssignmentError struct {
	Code    string
	Message string
	cause   error
}

// Error returns the stable code and its English explanatory message.
func (e *AssignmentError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the category sentinel used by errors.Is.
func (e *AssignmentError) Unwrap() error {
	return e.cause
}

func assignmentError(cause error, message string) error {
	return &AssignmentError{Code: cause.Error(), Message: message, cause: cause}
}
