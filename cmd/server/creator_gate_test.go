package main

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestCreatorGateForEnvironmentKeepsHotseatCompatible(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("ALLOWED_CREATOR_EMAILS", "")

	if !creatorGateForEnvironment(true).Allowed(store.Actor{}) {
		t.Fatal("hotseat creator gate rejected the development actor")
	}
}

func TestCreatorGateForEnvironmentUsesTheEmulatorAllowlist(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "127.0.0.1:9099")
	t.Setenv("FIRESTORE_EMULATOR_HOST", "127.0.0.1:8081")
	t.Setenv("ALLOWED_CREATOR_EMAILS", "admin@mail.com")

	gate := creatorGateForEnvironment(false)
	if !gate.Allowed(store.Actor{Email: "admin@mail.com"}) {
		t.Fatal("emulator allowlisted email was rejected")
	}
	if gate.Allowed(store.Actor{Email: "other@example.com"}) {
		t.Fatal("emulator email outside the allowlist was accepted")
	}
}

func TestCreatorGateForEnvironmentUsesClaimsOutsideTheEmulator(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	t.Setenv("ALLOWED_CREATOR_EMAILS", "admin@mail.com")

	gate := creatorGateForEnvironment(false)
	if !gate.Allowed(store.Actor{GameCreator: true}) {
		t.Fatal("production creator claim was rejected")
	}
	if gate.Allowed(store.Actor{}) {
		t.Fatal("production gate accepted an actor without a creator claim")
	}
	if gate.Allowed(store.Actor{Email: "admin@mail.com"}) {
		t.Fatal("production environment unexpectedly used the email allowlist")
	}
}
