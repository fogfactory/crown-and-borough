package orders

import (
	"errors"
	"fmt"

	"github.com/fogfactory/crown-and-borough/internal/i18n"
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
	// ParseCodeSpecialKind identifies an unknown or non-playable card kind.
	ParseCodeSpecialKind = "special_kind"
	// ParseCodeSpecialRegion identifies an unknown region seed.
	ParseCodeSpecialRegion = "special_region"
)

var (
	// ErrNoArmyOnPosition reports that no army occupies the receiving position.
	ErrNoArmyOnPosition = errors.New("no_army_on_position")
	// ErrArmyNotOwned reports that the army at the receiving position has another owner.
	ErrArmyNotOwned = errors.New("army_not_owned")
	// ErrNoblePrisoner reports that a dungeon noble attempted to emit.
	ErrNoblePrisoner = errors.New("noble_prisoner")
	// ErrEmissionCapacity reports a second emission by one noble in the same turn.
	ErrEmissionCapacity = errors.New("emission_capacity")
	// ErrInvalidChain reports a malformed or statically invalid submitted chain.
	ErrInvalidChain = errors.New("invalid_chain")
)

// ParseError describes one source-text error. Line is the original one-based
// line number, or zero when no input line could be identified.
type ParseError struct {
	Line        int
	Code        string
	Message     string
	MessageKey  string `json:"-"`
	MessageArgs []any  `json:"-"`
}

// Error returns a compact, human-readable representation of a parser error.
func (e ParseError) Error() string {
	if e.Line == 0 {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("line %d: %s: %s", e.Line, e.Code, e.Message)
}

// ValidationCodeNotAdjacent is the static adjacency diagnostic.
const ValidationCodeNotAdjacent = "not_adjacent"

// ValidationError describes an intrinsic chain validation error. OrderID is
// empty only when the error applies to the chain header rather than one order.
type ValidationError struct {
	OrderID     models.OrderID
	Code        string
	Message     string
	MessageKey  string `json:"-"`
	MessageArgs []any  `json:"-"`
}

// Error returns a compact, human-readable representation of a validation error.
func (e ValidationError) Error() string {
	if e.OrderID == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("order %s: %s: %s", e.OrderID, e.Code, e.Message)
}

// AssignmentError describes a reception failure. It unwraps to a stable
// sentinel so callers can classify it with errors.Is.
type AssignmentError struct {
	Code        string
	Message     string
	MessageKey  string
	MessageArgs []any
	cause       error
}

// Error returns the stable code and its English explanatory message.
func (e *AssignmentError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the category sentinel used by errors.Is.
func (e *AssignmentError) Unwrap() error {
	return e.cause
}

func assignmentError(cause error, key string, args ...any) error {
	message := i18n.Message{Key: key, Args: args}
	return &AssignmentError{
		Code:        cause.Error(),
		Message:     i18n.EnglishText(message),
		MessageKey:  key,
		MessageArgs: append([]any(nil), args...),
		cause:       cause,
	}
}

// CatalogMessage extracts the player-facing catalog message carried by an
// assignment failure without changing the stable errors.Is category.
func CatalogMessage(err error) (i18n.Message, bool) {
	var assignment *AssignmentError
	if !errors.As(err, &assignment) || assignment.MessageKey == "" {
		return i18n.Message{}, false
	}
	return i18n.Message{Key: assignment.MessageKey, Args: append([]any(nil), assignment.MessageArgs...)}, true
}

func parseMessage(line int, code, key string, args ...any) ParseError {
	message := i18n.Message{Key: key, Args: args}
	return ParseError{
		Line:        line,
		Code:        code,
		Message:     i18n.EnglishText(message),
		MessageKey:  key,
		MessageArgs: append([]any(nil), args...),
	}
}

func validationMessage(orderID models.OrderID, code, key string, args ...any) ValidationError {
	message := i18n.Message{Key: key, Args: args}
	return ValidationError{
		OrderID:     orderID,
		Code:        code,
		Message:     i18n.EnglishText(message),
		MessageKey:  key,
		MessageArgs: append([]any(nil), args...),
	}
}
