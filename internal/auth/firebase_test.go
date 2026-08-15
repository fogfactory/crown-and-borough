package auth

import (
	"context"
	"errors"
	"testing"
)

func TestFirebaseProjectClaimsAreBoundToConfiguredProject(t *testing.T) {
	claims := map[string]interface{}{
		"aud": "crown-project",
		"iss": "https://securetoken.google.com/crown-project",
	}
	if !validFirebaseProject(claims, "crown-project") {
		t.Fatal("valid Firebase claims were rejected")
	}
	claims["aud"] = "other-project"
	if validFirebaseProject(claims, "crown-project") {
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
