package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/engine/orders"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// ritualTestState has two regions (ROS, BOI), a Witch SOR owned by P1 and
// standing in BOI, and a draw pile that schedules the three calamities of next
// year before the bonus cards that fill the hands.
func ritualTestState(t *testing.T) *models.GameState {
	t.Helper()
	state := winterDeckState()
	state.Regions = append(state.Regions, models.Region{ID: "BOI", Seed: "BOI", Territories: []models.TerritoryID{"BOI"}})
	addNoble(state, "N1", "SOR", "P1", "BOI")
	state.Nobles[0].Sex = models.SexFemale
	state.Nobles[0].Dignities = []models.Dignity{models.DignityWitch}
	state.SpecialDeck.Cards = []models.SpecialCard{
		{ID: "C1", Kind: models.CardKindPlague},
		{ID: "B1", Kind: models.CardKindFairWeather},
		{ID: "C2", Kind: models.CardKindBadWeather},
		{ID: "B2", Kind: models.CardKindFairWeather},
		{ID: "C3", Kind: models.CardKindFamine},
		{ID: "B3", Kind: models.CardKindFairWeather},
		{ID: "B4", Kind: models.CardKindFairWeather},
	}
	state.SpecialDeck.DrawPile = []models.SpecialCardID{"C1", "B1", "C2", "B2", "C3", "B3", "B4"}
	return state
}

func ritual(id models.OrderID, calamity models.CardKind, season models.Season) models.WinterOrder {
	return models.WinterOrder{ID: id, Type: models.WinterOrderTypeRitual, NobleCode: "SOR", Calamity: calamity, Season: season}
}

func resolveRitual(t *testing.T, state *models.GameState, sheet ...models.WinterOrder) Resolution {
	t.Helper()
	resolution, err := ResolveWinterWithDeckOrders(state, winterDeckBalance(), map[models.PlayerID][]models.WinterOrder{"P1": sheet}, nil)
	if err != nil {
		t.Fatalf("ResolveWinterWithDeckOrders = %v", err)
	}
	return resolution
}

func ritualOutcome(t *testing.T, resolution Resolution) Event {
	t.Helper()
	for _, event := range eventsOfType(resolution.Events, EventTypeRitual) {
		if event.Reason != "declared" {
			return event
		}
	}
	t.Fatalf("no ritual outcome among %+v", eventsOfType(resolution.Events, EventTypeRitual))
	return Event{}
}

func TestRitualCallsACalamityOntoTheWitchRegion(t *testing.T) {
	resolution := resolveRitual(t, ritualTestState(t), ritual("O1", models.CardKindBadWeather, ""))
	var bent []models.Calamity
	for _, calamity := range resolution.State.Auguries[2].Calamities {
		if calamity.Ritual {
			bent = append(bent, calamity)
		}
	}
	if len(bent) != 1 || bent[0].Kind != models.CardKindBadWeather || bent[0].RegionSeed != "BOI" {
		t.Fatalf("bent calamities = %+v, want the bad weather on BOI", bent)
	}
	if outcome := ritualOutcome(t, resolution); outcome.Reason != "succeeded" || outcome.OwnerID != "P1" {
		t.Errorf("outcome = %+v, want a success for P1", outcome)
	}
	if declared := eventsOfType(resolution.Events, EventTypeRitual); len(declared) != 2 || declared[0].OrderID != "O1" {
		t.Errorf("ritual events = %+v, want the declaration carrying its order id, then the outcome", declared)
	}
}

func TestRitualFailsWhenTheCalledCalamityIsNotDrawn(t *testing.T) {
	state := ritualTestState(t)
	state.SpecialDeck.Cards[0].Kind = models.CardKindBadWeather // no plague left in the pile
	resolution := resolveRitual(t, state, ritual("O1", models.CardKindPlague, ""))
	for _, calamity := range resolution.State.Auguries[2].Calamities {
		if calamity.Ritual {
			t.Errorf("calamity %+v bent although the called kind was not drawn", calamity)
		}
	}
	if outcome := ritualOutcome(t, resolution); outcome.Reason != "failed" {
		t.Errorf("outcome = %+v, want a failure", outcome)
	}
}

func TestRitualFixesTheSeasonOfTheFirstCalamity(t *testing.T) {
	for _, season := range []models.Season{models.SeasonSpring, models.SeasonSummer, models.SeasonAutumn} {
		resolution := resolveRitual(t, ritualTestState(t), ritual("O1", "", season))
		calamities := resolution.State.Auguries[2].Calamities
		if len(calamities) == 0 || !calamities[0].Ritual || calamities[0].Season != season || calamities[0].RegionSeed != "BOI" {
			t.Fatalf("%s ritual: first calamity = %+v, want it bent onto that season and BOI", season, calamities)
		}
		bent := 0
		for _, calamity := range calamities {
			if calamity.Ritual {
				bent++
			}
		}
		if bent != 1 {
			t.Errorf("%s ritual bent %d calamities, want 1", season, bent)
		}
	}
}

func TestRitualRejections(t *testing.T) {
	dungeon := ritualTestState(t)
	setNobleStatus(dungeon, "N1", models.NobleStatusDungeon)
	notWitch := ritualTestState(t)
	notWitch.Nobles[0].Dignities = []models.Dignity{models.DignityAstrologer}
	for name, test := range map[string]struct {
		state  *models.GameState
		sheet  []models.WinterOrder
		reason string
	}{
		"no witch":      {notWitch, []models.WinterOrder{ritual("O1", models.CardKindFamine, "")}, "noble_not_witch"},
		"dungeon":       {dungeon, []models.WinterOrder{ritual("O1", models.CardKindFamine, "")}, "noble_not_witch"},
		"second ritual": {ritualTestState(t), []models.WinterOrder{ritual("O1", models.CardKindFamine, ""), ritual("O2", "", models.SeasonSummer)}, "ritual_already_used"},
		"bonus card":    {ritualTestState(t), []models.WinterOrder{ritual("O1", models.CardKindFairWeather, "")}, "ritual_invalid_target"},
		"both choices":  {ritualTestState(t), []models.WinterOrder{ritual("O1", models.CardKindFamine, models.SeasonSummer)}, "ritual_invalid_target"},
		"winter season": {ritualTestState(t), []models.WinterOrder{ritual("O1", "", models.SeasonWinter)}, "ritual_invalid_target"},
	} {
		t.Run(name, func(t *testing.T) {
			resolution := resolveRitual(t, test.state, test.sheet...)
			reasons := electionRejections(resolution.Events)
			last := test.sheet[len(test.sheet)-1].ID
			if reasons[last] != test.reason {
				t.Errorf("rejections = %v, want %s for %s", reasons, test.reason, last)
			}
		})
	}
}

func TestParseRitualOrders(t *testing.T) {
	state := ritualTestState(t)
	parsed, parseErrors := orders.ParseWinterOrders("s r sor mt\ns r sor 3", state)
	if len(parseErrors) != 0 || len(parsed) != 2 {
		t.Fatalf("parsed = %+v, errors = %+v", parsed, parseErrors)
	}
	if parsed[0].Type != models.WinterOrderTypeRitual || parsed[0].Calamity != models.CardKindBadWeather || parsed[0].Season != "" {
		t.Errorf("calamity ritual = %+v", parsed[0])
	}
	if parsed[1].Season != models.SeasonAutumn || parsed[1].Calamity != "" {
		t.Errorf("season ritual = %+v", parsed[1])
	}
	for _, line := range []string{"S R SOR", "S R SOR BT", "S R SOR 4", "S R SOR 0", "S R SOR MT 2", "S X SOR MT", "S R ZZZ MT"} {
		if _, errs := orders.ParseWinterOrders(line, state); len(errs) == 0 {
			t.Errorf("ParseWinterOrders(%q) accepted a malformed ritual", line)
		}
	}
}

func TestWinterAidsListTheWitchRitual(t *testing.T) {
	state := ritualTestState(t)
	aids := ForecastWinterAids(state, winterDeckBalance(), "P1")
	if len(aids.Rituals) != 1 || aids.Rituals[0].Noble != "SOR" || aids.Rituals[0].Region != "BOI" {
		t.Errorf("rituals = %+v, want the witch in BOI", aids.Rituals)
	}
	if other := ForecastWinterAids(state, winterDeckBalance(), "P2"); len(other.Rituals) != 0 {
		t.Errorf("another player sees rituals: %+v", other.Rituals)
	}
}
