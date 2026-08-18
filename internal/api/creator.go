package api

import (
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// CreatorGate controls the ability to create a new multi-game session. Join
// and gameplay authorization remain membership checks in the store.
type CreatorGate interface {
	Allowed(store.Actor) bool
}

type AllowAllCreatorGate struct{}

func (AllowAllCreatorGate) Allowed(store.Actor) bool {
	return true
}

type FirebaseCreatorGate struct{}

func (FirebaseCreatorGate) Allowed(actor store.Actor) bool {
	return actor.GameCreator
}

type EmailCreatorGate struct {
	emails map[string]struct{}
}

func NewEmailCreatorGate(value string) EmailCreatorGate {
	emails := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		email := strings.ToLower(strings.TrimSpace(item))
		if email != "" {
			emails[email] = struct{}{}
		}
	}
	return EmailCreatorGate{emails: emails}
}

func (g EmailCreatorGate) Allowed(actor store.Actor) bool {
	email := strings.ToLower(strings.TrimSpace(actor.Email))
	if email == "" {
		return false
	}
	_, ok := g.emails[email]
	return ok
}

type AnyCreatorGate struct {
	gates []CreatorGate
}

func NewAnyCreatorGate(gates ...CreatorGate) AnyCreatorGate {
	return AnyCreatorGate{gates: append([]CreatorGate(nil), gates...)}
}

func (g AnyCreatorGate) Allowed(actor store.Actor) bool {
	for _, gate := range g.gates {
		if gate != nil && gate.Allowed(actor) {
			return true
		}
	}
	return false
}
