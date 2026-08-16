package auth

import (
	"context"
	"errors"
	"testing"
)

func TestFirebaseProjectClaimsAreBoundToConfiguredProject(t *testing.T) {
	if !validFirebaseProject(
		"crown-project",
		"https://securetoken.google.com/crown-project",
		"crown-project",
	) {
		t.Fatal("valid Firebase claims were rejected")
	}
	if validFirebaseProject(
		"other-project",
		"https://securetoken.google.com/crown-project",
		"crown-project",
	) {
		t.Fatal("token from another project was accepted")
	}
}

func TestFirebaseVerifierRequiresProjectAndToken(t *testing.T) {
	if _, err := NewFirebaseVerifier(context.Background(), ""); !errors.Is(err, ErrMissingProjectID) {
		t.Fatalf("missing project error = %v", err)
	}
	var verifier *FirebaseVerifier
	if _, err := verifier.VerifyIDToken(context.Background(), ""); !errors.Is(err, ErrInvalidIdentity) {
		t.Fatalf("empty token error = %v", err)
	}
}
