package firestorestore

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func isCode(err error, code codes.Code) bool {
	return err != nil && status.Code(err) == code
}

func mapReadError(err error, fallback error) error {
	if err == nil {
		return nil
	}
	if isCode(err, codes.NotFound) {
		return fallback
	}
	return err
}

func mapTransactionError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.Aborted, codes.FailedPrecondition:
		return store.ErrRevisionConflict
	default:
		return err
	}
}

func wrapOperation(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrUnknownGame) || errors.Is(err, store.ErrNotMember) || errors.Is(err, store.ErrRevisionConflict) {
		return err
	}
	return fmt.Errorf("firestorestore: %s: %w", operation, err)
}
