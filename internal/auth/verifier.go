// Package auth contains the server-side identity boundary.
package auth

import "context"

type Identity struct {
	UID         string
	Email       string
	GameCreator bool
}

type Verifier interface {
	VerifyIDToken(context.Context, string) (Identity, error)
}
