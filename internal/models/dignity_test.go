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

func TestWitchRitualsAreNotPlagueProof(t *testing.T) {
	witch := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityWitch}}
	if !witch.CanPerformRitual() || witch.ProtectsFromPlague() {
		t.Errorf("witch: ritual %v, plague proof %v; want a ritual and no plague protection", witch.CanPerformRitual(), witch.ProtectsFromPlague())
	}
	witch.Status = models.NobleStatusDungeon
	if witch.CanPerformRitual() {
		t.Error("a witch in a dungeon performs a ritual")
	}
	poisoner := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityPoisoner}}
	if poisoner.CanPerformRitual() || poisoner.Dignities[0].Effect().RivalConsumption != 1 {
		t.Error("the poisoner must burden rivals and perform no ritual")
	}
}
