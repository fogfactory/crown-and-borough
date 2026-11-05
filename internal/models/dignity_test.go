package models_test

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestLadyDignitiesStackOnlyAcrossVisibility(t *testing.T) {
	visible := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityAstrologer}}
	hidden := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityWitch}}
	both := models.Noble{Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityAstrologer, models.DignityWitch}}
	for _, dignity := range models.LadyDignities {
		for name, test := range map[string]struct {
			lady models.Noble
			want string
		}{
			"visible holder": {visible, pick(dignity.Effect().Hidden, "", "dignity_exclusive")},
			"hidden holder":  {hidden, pick(dignity.Effect().Hidden, "dignity_exclusive", "")},
			"holds both":     {both, "dignity_exclusive"},
		} {
			if dignity == models.DignityAstrologer || dignity == models.DignityWitch {
				continue // already held, or rejected as a duplicate
			}
			if reason := dignity.CanReceive(test.lady, false); reason != test.want {
				t.Errorf("%s on a %s: reason = %q, want %q", dignity, name, reason, test.want)
			}
		}
	}
	// The bastard is not a dignity of the ladies: it stacks with them.
	if reason := models.DignityBastard.CanReceive(both, false); reason != "" {
		t.Errorf("bastard on a lady with two dignities: reason = %q, want none", reason)
	}
}

// pick returns whenHidden for a hidden dignity and otherwise otherwise.
func pick(hidden bool, whenHidden, otherwise string) string {
	if hidden {
		return whenHidden
	}
	return otherwise
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
