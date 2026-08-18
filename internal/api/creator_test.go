package api

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestEmailCreatorGateNormalizesAddresses(t *testing.T) {
	gate := NewEmailCreatorGate(" admin@mail.com, second@example.com ")

	if !gate.Allowed(store.Actor{Email: "ADMIN@MAIL.COM"}) {
		t.Fatal("normalized allowlisted email was rejected")
	}
	if gate.Allowed(store.Actor{Email: "other@example.com"}) {
		t.Fatal("email outside the allowlist was accepted")
	}
	if gate.Allowed(store.Actor{}) {
		t.Fatal("actor without an email was accepted")
	}
}

func TestFirebaseCreatorGateRequiresVerifiedClaim(t *testing.T) {
	gate := FirebaseCreatorGate{}
	if !gate.Allowed(store.Actor{GameCreator: true}) {
		t.Fatal("verified creator claim was rejected")
	}
	if gate.Allowed(store.Actor{}) {
		t.Fatal("missing creator claim was accepted")
	}
}

func TestAnyCreatorGateIsFailClosed(t *testing.T) {
	gate := NewAnyCreatorGate(FirebaseCreatorGate{}, NewEmailCreatorGate("admin@mail.com"))

	if !gate.Allowed(store.Actor{Email: "admin@mail.com"}) {
		t.Fatal("allowlisted actor was rejected")
	}
	if !gate.Allowed(store.Actor{GameCreator: true}) {
		t.Fatal("claimed actor was rejected")
	}
	if gate.Allowed(store.Actor{Email: "other@example.com"}) {
		t.Fatal("unconfigured actor was accepted")
	}
	if NewAnyCreatorGate().Allowed(store.Actor{GameCreator: true}) {
		t.Fatal("empty gate accepted an actor")
	}
}
