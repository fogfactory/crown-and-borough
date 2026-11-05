package main

import (
	"log"
	"os"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// demoScenario applies the forged-state scenario named by DEMO_SCENARIO to the
// hotseat game. The scenarios only exist in demo builds (-tags demo, see
// scripts/demo.sh); demoRegistered is nil in every other build.
var demoRegistered func(name string, memory *store.MemoryStore) error

func applyDemoScenario(memory *store.MemoryStore) {
	name := os.Getenv("DEMO_SCENARIO")
	if name == "" {
		return
	}
	if demoRegistered == nil {
		log.Fatal("DEMO_SCENARIO needs a demo build: use scripts/demo.sh")
	}
	if err := demoRegistered(name, memory); err != nil {
		log.Fatalf("demo scenario %q: %v", name, err)
	}
	log.Printf("demo scenario %q applied", name)
}
