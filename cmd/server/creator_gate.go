package main

import (
	"os"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/api"
)

func creatorGateForEnvironment(onlineDevMode bool) api.CreatorGate {
	if onlineDevMode {
		return api.AllowAllCreatorGate{}
	}

	gates := []api.CreatorGate{api.FirebaseCreatorGate{}}
	if emulatorConfigured() {
		gates = append(gates, api.NewEmailCreatorGate(os.Getenv("ALLOWED_CREATOR_EMAILS")))
	}
	return api.NewAnyCreatorGate(gates...)
}

func emulatorConfigured() bool {
	return strings.TrimSpace(os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")) != "" ||
		strings.TrimSpace(os.Getenv("FIRESTORE_EMULATOR_HOST")) != ""
}
