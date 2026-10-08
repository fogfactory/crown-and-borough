package api

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

func eonState() *models.GameState {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Nobles = []models.Noble{{
		ID: "N1", Code: "GUI", Name: "Guillaume", Sex: models.SexMale, SecretCode: "ELA", SecretName: "Eleonore", SecretSex: models.SexFemale,
		OwnerID: "P1", Status: models.NobleStatusFree,
		Dignities: []models.Dignity{models.DignityChevalierDEon, models.DignityBastard},
	}}
	return state
}

func TestProjectStateKeepsTheSecretIdentityAndHiddenDignityForTheOwner(t *testing.T) {
	state := eonState()
	owner := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).Nobles[0]
	if owner.Secret == nil || owner.Secret.Name != "Eleonore" || owner.Secret.Sex != models.SexFemale || len(owner.Dignities) != 2 {
		t.Errorf("owner view = %+v, want the secret identity and both dignities", owner)
	}
	other := ProjectStateForPlayer(state, "P2", assetgen.Balance{}).Nobles[0]
	if other.Secret != nil || other.Sex != models.SexMale || other.Name != "Sieur Guillaume" ||
		len(other.Dignities) != 1 || other.Dignities[0] != models.DignityBastard {
		t.Errorf("other view = %+v, want only the public male noble", other)
	}
}

func TestProjectReportDropsOthersHiddenDignityNomination(t *testing.T) {
	report := engine.TurnReport{Winter: &engine.WinterReport{Investments: []engine.WinterInvestmentReport{
		{Kind: engine.EventTypeDignity, Player: "P1", Dignity: models.DignityChevalierDEon},
		{Kind: engine.EventTypeDignity, Player: "P1", Dignity: models.DignityDArc},
		{Kind: engine.EventTypeRejected, Player: "P1", Order: &models.WinterOrder{Type: models.WinterOrderTypeDignity, CardCode: "SOR"}},
		{Kind: engine.EventTypeRejected, Player: "P1", Order: &models.WinterOrder{Type: models.WinterOrderTypeDignity, CardCode: "ARC"}},
	}}}
	if got := len(ProjectReport(report, "P1", nil).Winter.Investments); got != 4 {
		t.Errorf("owner investments = %d, want 4", got)
	}
	other := ProjectReport(report, "P2", nil).Winter.Investments
	if len(other) != 2 || other[0].Dignity != models.DignityDArc || other[1].Order.CardCode != "ARC" {
		t.Errorf("other investments = %+v, want only the public dignity", other)
	}
	if len(report.Winter.Investments) != 4 {
		t.Error("projection mutated the source report")
	}
}

func TestProjectStateForecastsCalamitiesForAnAstrologerInWinter(t *testing.T) {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Season = models.SeasonWinter
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{
			{ID: "C1", Kind: models.CardKindPlague}, {ID: "C2", Kind: models.CardKindFairWeather},
			{ID: "C3", Kind: models.CardKindFamine}, {ID: "C4", Kind: models.CardKindBadWeather},
		},
		DrawPile: []models.SpecialCardID{"C1", "C2", "C3", "C4"}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {}, "P2": {}},
	}
	state.Nobles = []models.Noble{{
		ID: "N1", Code: "ELE", Name: "Eleonore", Sex: models.SexFemale, OwnerID: "P1",
		Status: models.NobleStatusFree, Dignities: []models.Dignity{models.DignityAstrologer},
	}}
	got := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).CalamityForecast
	if len(got) != 3 || got[0] != models.CardKindPlague || got[1] != models.CardKindFamine || got[2] != models.CardKindBadWeather {
		t.Errorf("forecast = %v, want the three calamities in draw order", got)
	}
	if other := ProjectStateForPlayer(state, "P2", assetgen.Balance{}).CalamityForecast; len(other) != 0 {
		t.Errorf("other player forecast = %v, want none", other)
	}
	state.Nobles[0].Status = models.NobleStatusDungeon
	if prisoner := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).CalamityForecast; len(prisoner) != 0 {
		t.Errorf("prisoner forecast = %v, want none", prisoner)
	}
	state.Nobles[0].Status = models.NobleStatusFree
	state.Season = models.SeasonSpring
	if spring := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).CalamityForecast; len(spring) != 0 {
		t.Errorf("spring forecast = %v, want none outside winter", spring)
	}
}

func informationState() *models.GameState {
	state := models.NewGameState()
	state.Players = []models.Player{{ID: "P1", Name: "One"}, {ID: "P2", Name: "Two"}}
	state.Territories = []models.Territory{{ID: "AAA"}, {ID: "BBB"}}
	state.Regions = []models.Region{{ID: "R", Seed: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}}}
	state.Armies = []models.Army{{ID: "A1", OwnerID: "P2", TerritoryID: "AAA", Size: 1}}
	state.TerritoryStates = map[models.TerritoryID]models.TerritoryState{"AAA": {Army: ptr(models.ArmyID("A1"))}, "BBB": {}}
	return state
}

func ptr[T any](value T) *T { return &value }

func TestCorrespondentSeesHostOrdersAndSpySeesHostHand(t *testing.T) {
	state := informationState()
	state.Nobles = []models.Noble{
		{ID: "N1", Code: "ELE", OwnerID: "P1", LocationID: "AAA", Status: models.NobleStatusHostage, Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityCorrespondent, models.DignitySpy}},
		{ID: "N2", Code: "HUG", OwnerID: "P2", LocationID: "BBB", Status: models.NobleStatusFree, Sex: models.SexMale},
	}
	state.SpecialDeck = &models.SpecialDeck{
		Cards: []models.SpecialCard{{ID: "C1", Kind: models.CardKindFairWeather}}, DrawPile: []models.SpecialCardID{}, Discard: []models.SpecialCardID{},
		Hands: map[models.PlayerID][]models.SpecialCardID{"P1": {}, "P2": {"C1"}},
	}
	privacy := ensurePrivacy(state)
	chain := models.Chain{ID: "H1", NobleID: "N2", ArmyID: "A1", Orders: []models.Order{{PositionID: "AAA"}}}
	recordDignityKnowledge(state, privacy, chain, state.Nobles[1], makeChainSnapshot(chain, 1))
	if _, known := privacy.ChainKnowledge["P1"]["H1"]; !known {
		t.Error("the correspondent's owner does not know the host's chain")
	}
	hands := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).SpiedHands
	if len(hands) != 1 || hands[0].Player != "P2" || len(hands[0].SpecialHand) != 1 {
		t.Errorf("spied hands = %+v, want the host's hand", hands)
	}
	state.Nobles[0].Status = models.NobleStatusDungeon
	if hands := ProjectStateForPlayer(state, "P1", assetgen.Balance{}).SpiedHands; len(hands) != 0 {
		t.Errorf("prisoner spy hands = %+v, want none", hands)
	}
}

func TestCastellanSeesChainsOnHerFiefOnlyInACastle(t *testing.T) {
	state := informationState()
	state.Fiefs = []models.Fief{{ID: "F1", CapitalTerritoryID: "AAA", Territories: []models.TerritoryID{"AAA", "BBB"}, OwnerID: "P2"}}
	state.Nobles = []models.Noble{
		{ID: "N1", Code: "ELE", OwnerID: "P1", LocationID: "BBB", Status: models.NobleStatusFree, Sex: models.SexFemale, Dignities: []models.Dignity{models.DignityCastellan}},
		{ID: "N2", Code: "HUG", OwnerID: "P2", LocationID: "AAA", Status: models.NobleStatusFree, Sex: models.SexMale},
	}
	chain := models.Chain{ID: "H1", NobleID: "N2", ArmyID: "A1", Orders: []models.Order{{PositionID: "AAA", TargetIDs: []models.TerritoryID{"BBB"}}}}
	snapshot := makeChainSnapshot(chain, 1)
	privacy := ensurePrivacy(state)
	recordDignityKnowledge(state, privacy, chain, state.Nobles[1], snapshot)
	if _, known := privacy.ChainKnowledge["P1"]["H1"]; known {
		t.Error("the castellan outside a castle saw a chain")
	}
	state.Infrastructures = []models.Infrastructure{{ID: "I1", Type: models.InfraTypeCastle, Level: 1, TerritoryID: "BBB"}}
	recordDignityKnowledge(state, privacy, chain, state.Nobles[1], snapshot)
	if _, known := privacy.ChainKnowledge["P1"]["H1"]; !known {
		t.Error("the castellan in a castle did not see the chain on her fief")
	}
}

func TestProjectStateExposesBishopricsAndReligiousStandings(t *testing.T) {
	state := eonState()
	state.Nobles = append(state.Nobles, models.Noble{ID: "N2", Code: "ADE", Name: "Adhemar", Sex: models.SexMale, OwnerID: "P2", Status: models.NobleStatusDungeon})
	state.Regions = []models.Region{
		{ID: "AAA", Name: "Aaa", Seed: "AAA", Territories: []models.TerritoryID{"AAA"}},
		{ID: "BBB", Name: "Bbb", Seed: "BBB", Territories: []models.TerritoryID{"BBB"}},
	}
	pope := models.NobleID("N1")
	state.Bishops = []models.Bishop{{Region: "AAA", Noble: "N1"}}
	state.Cardinals = []models.NobleID{"N1"}
	state.Pope = &pope
	state.Excommunications = []models.Excommunication{{Noble: "N2", Reason: models.ExcommunicationPapal, By: "N1", Turn: 1}}

	view := ProjectStateForPlayer(state, "P2", assetgen.Balance{})
	if len(view.Bishoprics) != 2 || view.Bishoprics[0].Bishop == nil || *view.Bishoprics[0].Bishop != "GUI" || view.Bishoprics[1].Bishop != nil || view.Bishoprics[0].Name != "Aaa" {
		t.Errorf("bishoprics = %+v", view.Bishoprics)
	}
	if view.Pope == nil || *view.Pope != "GUI" || len(view.Cardinals) != 1 || len(view.Excommunicated) != 1 || view.Excommunicated[0].Noble != "ADE" {
		t.Errorf("pope %v cardinals %v excommunicated %v", view.Pope, view.Cardinals, view.Excommunicated)
	}
	if n := view.Nobles[0]; n.ReligiousTitle != models.ReligiousTitlePope {
		t.Errorf("pope noble view = %+v", n)
	}
	if n := view.Nobles[1]; n.ReligiousTitle != "" {
		t.Errorf("untitled noble view = %+v", n)
	}
	empty := ProjectState(nil, assetgen.Balance{})
	if empty.Bishoprics == nil || empty.Cardinals == nil || empty.Excommunicated == nil {
		t.Error("empty projection must serialize [] not null")
	}
}
