package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLadyDignitiesDoNotStack(t *testing.T) {
	lady := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityAstrologer}}
	for _, dignity := range models.LadyDignities {
		if dignity == models.DignityAstrologer {
			continue
		}
		if reason := dignity.CanReceive(lady, false); reason != "dignity_exclusive" {
			t.Errorf("%s on an astrologer: reason = %q, want dignity_exclusive", dignity, reason)
		}
	}
	// The bastard is not a dignity of the ladies: it stacks with one.
	if reason := models.DignityBastard.CanReceive(lady, false); reason != "" {
		t.Errorf("bastard on an astrologer: reason = %q, want none", reason)
	}
}
