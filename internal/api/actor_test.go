package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/auth"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

type testIdentityVerifier struct {
	identity auth.Identity
	err      error
}

func (v testIdentityVerifier) VerifyIDToken(context.Context, string) (auth.Identity, error) {
	return v.identity, v.err
}

func TestFirebaseActorResolverUsesVerifiedUIDAndIgnoresQueryIdentity(t *testing.T) {
	resolver := FirebaseActorResolver(testIdentityVerifier{identity: auth.Identity{UID: "verified-uid", Email: "user@example.com"}})
	request, err := http.NewRequest(http.MethodGet, "/api/games?player=spoofed", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer token")
	actor, err := resolver(request)
	if err != nil {
		t.Fatalf("resolve actor: %v", err)
	}
	if actor != (store.Actor{ID: "verified-uid", Email: "user@example.com"}) {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestFirebaseActorResolverCarriesCreatorClaim(t *testing.T) {
	resolver := FirebaseActorResolver(testIdentityVerifier{identity: auth.Identity{
		UID:         "creator-uid",
		Email:       "creator@example.com",
		GameCreator: true,
	}})
	request, err := http.NewRequest(http.MethodGet, "/api/games", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer token")
	actor, err := resolver(request)
	if err != nil {
		t.Fatalf("resolve actor: %v", err)
	}
	if !actor.GameCreator {
		t.Fatal("creator claim was not carried to the actor")
	}
}
