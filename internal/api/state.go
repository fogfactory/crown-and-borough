package api

import (
	"encoding/json"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// StateView is the state.json representation served to the frontend. It keeps
// dynamic entities nested under their territory instead of serializing the
// storage-oriented GameState directly, and includes the public calendar and
// score snapshot.
type StateView struct {
	Turn      int                                       `json:"turn"`
	Year      int                                       `json:"year"`
	YearCount int                                       `json:"yearCount"`
	Season    models.Season                             `json:"season"`
	Scores    map[models.PlayerID]engine.ScoreBreakdown `json:"scores"`
	Victory   engine.VictoryStatus                      `json:"victory"`
	Finished  bool                                      `json:"finished"`
	Winner    *models.PlayerID                          `json:"winner,omitempty"`
	// Winners lists every major winner (two spouses for a joint victory);
	// MinorWinner is the player granted the minor victory, if any.
	Winners     []models.PlayerID `json:"winners,omitempty"`
	MinorWinner *models.PlayerID  `json:"minorWinner,omitempty"`
	Players     []PlayerView      `json:"players"`
	Territories []TerritoryView   `json:"territories"`
	Nobles      []NobleView       `json:"nobles"`
	Fiefs       []FiefView        `json:"fiefs"`
	Marriages   []MarriageView    `json:"marriages"`
	// Deceased lists the nobles removed from the game, so the lineage can
	// still show them and the marriages they were part of.
	Deceased []DeceasedView `json:"deceased"`
	Claims   []ClaimView    `json:"claims"`
	// Bishoprics lists every bishopric (the regions of the map) with its
	// bishop, and Cardinals, Pope and Excommunicated give the other religious
	// standings; all are public (specs/religieux.md).
	Bishoprics     []BishopricView       `json:"bishoprics"`
	Cardinals      []models.NobleCode    `json:"cardinals"`
	Pope           *models.NobleCode     `json:"pope,omitempty"`
	Excommunicated []ExcommunicationView `json:"excommunicated"`
	// HandLimit is special_orders.hand_limit: the cap on the cards a player
	// holds, special-orders hand and noble hand together.
	HandLimit        int               `json:"handLimit"`
	SpecialHand      []models.CardKind `json:"specialHand"`
	NobleHand        []NobleCardView   `json:"nobleHand"`
	NobleDeckSize    int               `json:"nobleDeckSize"`
	NobleDiscardSize int               `json:"nobleDiscardSize"`
	// CalamityForecast lists, in winter, the next calamity cards of the draw
	// pile that a free astrologue of the viewer reveals (specs/dames.md
	// § Astrologue). It is private to that player.
	CalamityForecast []models.CardKind `json:"calamityForecast,omitempty"`
	// OpenElections lists, in winter, the elections the registry frozen at
	// the start of the winter holds, with the viewer's own voices and eligible
	// nobles (see engine.ForecastWinterElections). Ballots are never part of
	// it.
	OpenElections []engine.OpenElection `json:"openElections,omitempty"`
	// WinterAids lists, in winter, the possible winter orders of the viewer
	// (papal sanctions, cardinal purchase, fief sites) with their prices.
	WinterAids *engine.WinterAids `json:"winterAids,omitempty"`
	// RevoltTargets lists, outside winter, the territories a Révolte card can
	// be played on (a public rule: famine, recent tax or trial in the region).
	RevoltTargets []models.TerritoryID `json:"revoltTargets,omitempty"`
	// AppeasementCostBase is the base of the paid revolt appeasement price:
	// base^size R for a rebel army of that size.
	AppeasementCostBase int `json:"appeasementCostBase,omitempty"`
	// SpiedHands are the hands a spy of the viewer reveals: the whole hand of
	// the player holding her hostage.
	SpiedHands          []SpiedHandView             `json:"spiedHands,omitempty"`
	ActiveRegionEffects []models.ActiveRegionEffect `json:"activeRegionEffects"`
	Announcements       []engine.AnnouncementReport `json:"announcements"`
}

// PlayerView contains the public player metadata needed by the hotseat
// selector. Player-specific filtering is a future server concern.
// ProjectedIncome is the territory income the player would receive on the
// next action turn if nothing changes, ignoring any calamity or bonus card
// already drawn this turn (see engine.ForecastIncome). ProjectedMillIncome is
// the separate mill production their own settlements and self-supplied mills
// would receive over the same turn (see engine.ForecastMillIncome): it is not
// included in ProjectedIncome since mills do not route through the capital
// and, since #195, each mill credits exactly one destination.
// ProjectedConsumption and ArmiesAtRisk are the equivalent projection for
// ravitaillement: the net rations every army the player controls will draw
// from stock or the supply network beyond what its own territory already
// produces for it, and the ones that would starve if nothing changes before
// resolution (see engine.ForecastFamineRisk). Both are zero-valued in
// winter, since ravitaillement never happens then.
type PlayerView struct {
	ID                   models.PlayerID     `json:"id"`
	Name                 string              `json:"name"`
	Color                string              `json:"color"`
	CapitalTerritory     *models.TerritoryID `json:"capitalTerritory,omitempty"`
	ProjectedIncome      int                 `json:"projectedIncome"`
	ProjectedMillIncome  int                 `json:"projectedMillIncome"`
	ProjectedConsumption int                 `json:"projectedConsumption"`
	ArmiesAtRisk         []ArmyRiskView      `json:"armiesAtRisk,omitempty"`
	// Succession lists the codes of the player's living nobles in line of
	// succession order (specs/succession.md § Lignée).
	Succession []models.NobleCode `json:"succession"`
}

// ArmyRiskView is the public shape of engine.ArmyFamineRisk: one army the
// famine risk forecast flags as starving, addressed by its territory the
// way the rest of the frontend addresses armies.
type ArmyRiskView struct {
	TerritoryID models.TerritoryID `json:"territoryId"`
	Size        int                `json:"size"`
	Deficit     int                `json:"deficit"`
}

// TerritoryView is the live state displayed on one map territory.
// ProjectedIncome and IncomeDestination back the territory detail panel's
// "rapporte X R à YYY" line; IncomeDestination is empty when the income
// would be lost (see engine.ForecastTerritoryIncome). MillProduction and
// MillDestination are the equivalent projection for a mill on this
// territory (see engine.ForecastMillProduction): MillDestination is this
// same territory when the mill has no eligible adjacent castle or village
// and stocks itself instead (see #195).
type TerritoryView struct {
	ID                models.TerritoryID  `json:"id"`
	Owner             *models.PlayerID    `json:"owner"`
	Resources         int                 `json:"resources"`
	Army              *ArmyView           `json:"army"`
	Infrastructures   []InfraView         `json:"infrastructures"`
	ProjectedIncome   int                 `json:"projectedIncome,omitempty"`
	IncomeDestination *models.TerritoryID `json:"incomeDestination,omitempty"`
	MillProduction    int                 `json:"millProduction,omitempty"`
	MillDestination   *models.TerritoryID `json:"millDestination,omitempty"`
}

// ArmyView contains the visible owner, size, and current chain of an army. Its
// ID is an internal storage detail: the frontend addresses an army by territory.
// The current v1 endpoint exposes every chain; server-side player filtering is
// tracked as a later online feature. Starving mirrors models.Army.Starving:
// set by last turn's ravitaillement when this army's demand went unmet, it
// fights at strength 0 this turn until ravitaillement re-evaluates it at the
// turn's own end (#208).
type ArmyView struct {
	Owner    models.PlayerID `json:"owner"`
	Size     int             `json:"size"`
	Chain    *ChainView      `json:"chain"`
	Starving bool            `json:"starving,omitempty"`
}

// ChainView is the public, code-addressed representation of an active chain.
// It intentionally omits storage IDs for the chain, its orders, and its army.
type ChainView struct {
	Noble        models.NobleCode `json:"noble"`
	CurrentIndex int              `json:"currentIndex"`
	Orders       []OrderView      `json:"orders"`
	Visibility   string           `json:"visibility,omitempty"`
}

// MarshalJSON keeps the public chain shape compact while making an existing
// but undisclosed chain distinguishable from an absent chain. The zero-value
// visibility is retained for the legacy global hotseat projection.
func (view ChainView) MarshalJSON() ([]byte, error) {
	if view.Visibility == "hidden" {
		return json.Marshal(struct {
			Visibility string `json:"visibility"`
		}{Visibility: view.Visibility})
	}
	return json.Marshal(struct {
		Noble        models.NobleCode `json:"noble"`
		CurrentIndex int              `json:"currentIndex"`
		Orders       []OrderView      `json:"orders"`
		Visibility   string           `json:"visibility,omitempty"`
	}{
		Noble:        view.Noble,
		CurrentIndex: view.CurrentIndex,
		Orders:       view.Orders,
		Visibility:   view.Visibility,
	})
}

// OrderView is one public order. Territory and noble references use their
// trigrams instead of internal IDs so the frontend can address map entities.
type OrderView struct {
	Type             models.OrderType                          `json:"type"`
	Position         models.TerritoryID                        `json:"position"`
	Targets          []models.TerritoryID                      `json:"targets,omitempty"`
	NobleAssignments map[models.TerritoryID][]models.NobleCode `json:"nobleAssignments,omitempty"`
	Liaison          models.LiaisonMode                        `json:"liaison"`
	Amount           int                                       `json:"amount,omitempty"`
}

// InfraView contains the visible kind and level of an infrastructure.
// Fortified is only ever true on a village (#193): a village fortified
// through C C keeps its type but gains a castle's defensive bonus.
type InfraView struct {
	Type      models.InfraType `json:"type"`
	Level     int              `json:"level"`
	Fortified bool             `json:"fortified,omitempty"`
}

// NobleView contains the visible identity, code, status, owner, and location
// of a noble.
type NobleView struct {
	ID   models.NobleID   `json:"id"`
	Code models.NobleCode `json:"code"`
	Name string           `json:"name"`
	// FirstName is the noble's first name alone, for short labels.
	FirstName string             `json:"firstName"`
	Owner     models.PlayerID    `json:"owner"`
	Location  models.TerritoryID `json:"location"`
	Status    models.NobleStatus `json:"status"`
	Sex       models.Sex         `json:"sex"`
	// Spouse is the code of the noble this one is married to, set only while
	// the marriage is active (both spouses alive).
	Spouse *models.NobleCode `json:"spouse,omitempty"`
	// Dignities are the permanent distinctions the noble carries (the
	// bastard); they are public.
	Dignities []models.Dignity `json:"dignities,omitempty"`
	// ReligiousTitle is the highest religious title the noble holds, if any.
	ReligiousTitle models.ReligiousTitle `json:"religiousTitle,omitempty"`
	// Secret is the private identity a chevalier d'Éon replaced; owner only.
	Secret *SecretIdentityView `json:"secret,omitempty"`
}

// NobleCardView is one card of the viewer's own noble hand. The hands of the
// other players and the order of the draw pile are never projected.
type NobleCardView struct {
	ID      models.NobleCardID   `json:"id"`
	Kind    models.NobleCardKind `json:"kind"`
	Code    string               `json:"code"`
	Name    string               `json:"name,omitempty"`
	Sex     models.Sex           `json:"sex,omitempty"`
	Dignity models.Dignity       `json:"dignity,omitempty"`
}

// BishopricView is a bishopric: a region of the map, named after its seed
// commune, with its bishop (nil when vacant).
type BishopricView struct {
	Region      models.RegionID      `json:"region"`
	Name        string               `json:"name"`
	Territories []models.TerritoryID `json:"territories"`
	Bishop      *models.NobleCode    `json:"bishop,omitempty"`
}

// ExcommunicationView is an excommunicated noble with the reason (ex officio
// or papal).
type ExcommunicationView struct {
	Noble  models.NobleCode             `json:"noble"`
	Reason models.ExcommunicationReason `json:"reason"`
	Turn   int                          `json:"turn"`
}

// FiefView is a fief addressed by its capital's trigram: no internal fief id
// is exposed to the client (titres.md, #194). Holder is nil when the fief is
// vacant. ProjectedIncome sums engine.ForecastTerritoryIncome's amount over
// every member territory: the fief's own income projection, distinct from
// the capital territory's own projected income since it also receives every
// other member's income (titres.md, #196).
type FiefView struct {
	Capital         models.TerritoryID   `json:"capital"`
	Title           models.FiefTitle     `json:"title"`
	Territories     []models.TerritoryID `json:"territories"`
	Owner           models.PlayerID      `json:"owner"`
	Holder          *models.NobleCode    `json:"holder,omitempty"`
	ProjectedIncome int                  `json:"projectedIncome"`
}

// MarriageView is one concluded marriage, public to every viewer
// (specs/succession.md § Conclusion d'un mariage). The marriage outlives its
// spouses, so a dead spouse is still named by its code.
//
// Active is false once a spouse has died. Category and Weight are set only
// for an active alliance (not for a bastard's marriage): Category is the
// marriage's own category (head or secondary) and Weight its alliance
// weight. ActiveHeadFor lists the houses for which this marriage is the
// active head (specs/succession.md § Tête active).
type MarriageView struct {
	NobleA        models.NobleCode        `json:"nobleA"`
	NobleB        models.NobleCode        `json:"nobleB"`
	Turn          int                     `json:"turn"`
	Active        bool                    `json:"active"`
	Category      engine.AllianceCategory `json:"category,omitempty"`
	Weight        int                     `json:"weight,omitempty"`
	ActiveHeadFor []models.PlayerID       `json:"activeHeadFor,omitempty"`
	// HeadSuccessors tells, for each house this marriage is the active head
	// of, which marriage would become that house's active head if this one
	// ended (no marriage when none would).
	HeadSuccessors []HeadSuccessorView `json:"headSuccessors,omitempty"`
}

// HeadSuccessorView is the marriage that would take over as a house's active
// head. Marriage is nil when the house would be left without a head.
type HeadSuccessorView struct {
	Player   models.PlayerID `json:"player"`
	Marriage *MarriageRef    `json:"marriage,omitempty"`
}

// MarriageRef names a marriage by its two spouses' codes.
type MarriageRef struct {
	NobleA models.NobleCode `json:"nobleA"`
	NobleB models.NobleCode `json:"nobleB"`
}

// ClaimView is a pretension staked by Heir on the titles of Target
// (specs/succession.md § Prétentions), public to every viewer. Rank is the
// heir's position among the claims on Target, 1 being the first to inherit.
type ClaimView struct {
	Heir     models.NobleCode `json:"heir"`
	Target   models.NobleCode `json:"target"`
	Spouse   models.NobleCode `json:"spouse"`
	Turn     int              `json:"turn"`
	WifeSide bool             `json:"wifeSide,omitempty"`
	Rank     int              `json:"rank"`
}

// DeceasedView is a noble that has left the game, public to every viewer
// like the marriages it appears in.
type DeceasedView struct {
	Code  models.NobleCode  `json:"code"`
	Name  string            `json:"name"`
	Owner models.PlayerID   `json:"owner"`
	Sex   models.Sex        `json:"sex"`
	Cause models.DeathCause `json:"cause"`
	Turn  int               `json:"turn"`
}

func projectState(state *models.GameState, balance assetgen.Balance) StateView {
	return projectStateForViewer(state, nil, balance)
}

// ProjectState returns the public state projection used by the development
// session. Hosted callers should use ProjectStateForPlayer so chain knowledge
// is applied to the viewer.
func ProjectState(state *models.GameState, balance assetgen.Balance) StateView {
	return projectState(state, balance)
}

func projectStateForPlayer(state *models.GameState, playerID models.PlayerID, balance assetgen.Balance) StateView {
	return projectStateForViewer(state, &playerID, balance)
}

// ProjectStateForPlayer returns the server-filtered state projection for one
// player. It is exported so persistence adapters can materialize the same
// projection that the REST API returns without importing Firestore into the
// engine or models packages.
func ProjectStateForPlayer(state *models.GameState, playerID models.PlayerID, balance assetgen.Balance) StateView {
	return projectStateForPlayer(state, playerID, balance)
}

func projectStateForViewer(state *models.GameState, viewer *models.PlayerID, balance assetgen.Balance) StateView {
	view := StateView{
		Players:             []PlayerView{},
		Territories:         []TerritoryView{},
		Nobles:              []NobleView{},
		Fiefs:               []FiefView{},
		Bishoprics:          []BishopricView{},
		Cardinals:           []models.NobleCode{},
		Excommunicated:      []ExcommunicationView{},
		HandLimit:           balance.SpecialOrders.HandLimit,
		SpecialHand:         []models.CardKind{},
		NobleHand:           []NobleCardView{},
		ActiveRegionEffects: []models.ActiveRegionEffect{},
		Announcements:       []engine.AnnouncementReport{},
		Scores:              map[models.PlayerID]engine.ScoreBreakdown{},
		Victory:             engine.VictoryStatus{Players: map[models.PlayerID]engine.PlayerVictory{}},
	}
	if state == nil {
		return view
	}

	view.Turn = state.Turn
	view.Year = state.Year()
	view.YearCount = state.YearCount
	view.Season = state.Season
	view.Scores = engine.ComputeScores(state, balance)
	view.Victory = engine.ComputeVictoryStatus(state, balance)
	view.Finished = engine.GameFinished(state, balance)
	if view.Finished {
		outcome := engine.ResolveVictory(state, balance)
		view.Winners, view.MinorWinner = outcome.Major, outcome.Minor
		if len(outcome.Major) == 1 {
			view.Winner = &outcome.Major[0]
		}
	}
	view.Players = make([]PlayerView, 0, len(state.Players))
	view.Territories = make([]TerritoryView, 0, len(state.Territories))
	view.Nobles = make([]NobleView, 0, len(state.Nobles))

	armiesByID := make(map[models.ArmyID]models.Army, len(state.Armies))
	for _, army := range state.Armies {
		armiesByID[army.ID] = army
	}
	nobleCodesByID := make(map[models.NobleID]models.NobleCode, len(state.Nobles))
	for _, noble := range state.Nobles {
		nobleCodesByID[noble.ID] = models.NobleCode(noble.Code)
	}
	chainsByArmyID := make(map[models.ArmyID]models.Chain, len(state.Chains))
	for _, chain := range state.Chains {
		chainsByArmyID[chain.ArmyID] = chain
	}
	infrastructuresByID := make(map[models.InfraID]models.Infrastructure, len(state.Infrastructures))
	for _, infrastructure := range state.Infrastructures {
		infrastructuresByID[infrastructure.ID] = infrastructure
	}
	territoryIncome := engine.ForecastTerritoryIncome(state, balance)
	projectedIncomeByPlayer := make(map[models.PlayerID]int, len(state.Players))
	millIncomeByPlayer := engine.ForecastMillIncome(state, balance)
	millProduction := engine.ForecastMillProduction(state, balance)
	famineRiskByPlayer := engine.ForecastFamineRisk(state, balance)

	// Control is not stored: the projected owner is derived here from the
	// fiefs, the capitals and the stationed armies.
	controllers := state.TerritoryControllers()
	for _, territory := range state.Territories {
		territoryState := state.TerritoryStates[territory.ID]
		var owner *models.PlayerID
		if controller, controlled := controllers[territory.ID]; controlled {
			owner = &controller
		}
		territoryView := TerritoryView{
			ID:              territory.ID,
			Owner:           owner,
			Resources:       territoryState.Resources,
			Infrastructures: make([]InfraView, 0, 1),
		}
		if forecast, ok := territoryIncome[territory.ID]; ok {
			territoryView.ProjectedIncome = forecast.Amount
			if forecast.Destination != "" {
				destination := forecast.Destination
				territoryView.IncomeDestination = &destination
			}
			if owner != nil {
				projectedIncomeByPlayer[*owner] += forecast.Amount
			}
		}
		if forecast, ok := millProduction[territory.ID]; ok {
			territoryView.MillProduction = forecast.Production
			destination := forecast.Destination
			territoryView.MillDestination = &destination
		}
		if territoryState.Army != nil {
			if army, ok := armiesByID[*territoryState.Army]; ok {
				armyView := &ArmyView{
					Owner:    army.OwnerID,
					Size:     army.Size,
					Starving: army.Starving,
				}
				if chain, exists := chainsByArmyID[army.ID]; exists {
					armyView.Chain = projectChain(chain, nobleCodesByID)
					if viewer != nil && !viewerKnowsChain(state, *viewer, chain.ID) {
						armyView.Chain = &ChainView{Visibility: "hidden"}
					} else if viewer != nil {
						armyView.Chain.Visibility = "known"
					}
				}
				territoryView.Army = armyView
			}
		}
		if territoryState.Infrastructures != nil {
			if infrastructure, ok := infrastructuresByID[*territoryState.Infrastructures]; ok {
				territoryView.Infrastructures = append(territoryView.Infrastructures, InfraView{
					Type:      infrastructure.Type,
					Level:     infrastructure.Level,
					Fortified: infrastructure.Fortified,
				})
			}
		}
		view.Territories = append(view.Territories, territoryView)
	}
	for _, player := range state.Players {
		playerView := PlayerView{ID: player.ID, Name: player.Name, Color: player.Color, ProjectedIncome: projectedIncomeByPlayer[player.ID], ProjectedMillIncome: millIncomeByPlayer[player.ID]}
		if famineRisk, ok := famineRiskByPlayer[player.ID]; ok {
			playerView.ProjectedConsumption = famineRisk.NetConsumption
			for _, risk := range famineRisk.ArmiesAtRisk {
				playerView.ArmiesAtRisk = append(playerView.ArmiesAtRisk, ArmyRiskView{
					TerritoryID: risk.TerritoryID,
					Size:        risk.Size,
					Deficit:     risk.Deficit,
				})
			}
		}
		playerView.Succession = []models.NobleCode{}
		for _, noble := range state.SuccessionLine(player.ID) {
			playerView.Succession = append(playerView.Succession, models.NobleCode(noble.Code))
		}
		if player.CapitalCastleID != nil {
			if infrastructure, ok := infrastructuresByID[*player.CapitalCastleID]; ok && infrastructure.Type == models.InfraTypeCastle {
				capitalTerritory := infrastructure.TerritoryID
				playerView.CapitalTerritory = &capitalTerritory
			}
		}
		view.Players = append(view.Players, playerView)
	}
	for _, noble := range state.Nobles {
		nobleView := NobleView{
			ID:        noble.ID,
			Code:      models.NobleCode(noble.Code),
			Name:      state.NobleDisplayName(noble),
			FirstName: noble.FirstName(),
			Owner:     noble.OwnerID,
			Location:  noble.LocationID,
			Status:    noble.Status,
			Sex:       noble.Sex,
		}
		revealHidden := viewer != nil && (*viewer == noble.OwnerID || *viewer == models.SpectatorViewer)
		for _, dignity := range noble.Dignities {
			hidden := dignity.Effect().Hidden && !(dignity == models.DignityChevalierDEon && noble.EonUnmasked)
			if hidden && !revealHidden {
				continue
			}
			nobleView.Dignities = append(nobleView.Dignities, dignity)
		}
		if noble.SecretCode != "" && (revealHidden || noble.EonUnmasked) {
			nobleView.Secret = &SecretIdentityView{Code: models.NobleCode(noble.SecretCode), Name: noble.SecretName, Sex: noble.SecretSex}
		}
		if marriage, married := state.MarriageOf(noble.ID); married {
			spouseID := marriage.NobleA
			if spouseID == noble.ID {
				spouseID = marriage.NobleB
			}
			spouseCode := nobleCodesByID[spouseID]
			nobleView.Spouse = &spouseCode
		}
		view.Nobles = append(view.Nobles, nobleView)
	}
	for _, fief := range state.Fiefs {
		fiefView := FiefView{
			Capital:     fief.CapitalTerritoryID,
			Title:       fief.Title,
			Territories: append([]models.TerritoryID(nil), fief.Territories...),
			Owner:       fief.OwnerID,
		}
		if fief.HolderNobleID != nil {
			if code, exists := nobleCodesByID[*fief.HolderNobleID]; exists {
				fiefView.Holder = &code
			}
		}
		for _, territoryID := range fief.Territories {
			if forecast, ok := territoryIncome[territoryID]; ok {
				fiefView.ProjectedIncome += forecast.Amount
			}
		}
		view.Fiefs = append(view.Fiefs, fiefView)
	}
	view.Deceased = []DeceasedView{}
	for _, removed := range state.RemovedNobles {
		view.Deceased = append(view.Deceased, DeceasedView{Code: models.NobleCode(removed.Code), Name: removed.Name, Owner: removed.OwnerID, Sex: removed.Sex, Cause: removed.Cause, Turn: removed.Turn})
	}
	view.Bishoprics, view.Cardinals, view.Pope, view.Excommunicated = projectReligion(state, nobleCodesByID)
	for i, noble := range view.Nobles {
		view.Nobles[i].ReligiousTitle = state.ReligiousTitleOf(noble.ID)
	}
	view.Claims = []ClaimView{}
	for _, claim := range state.Claims {
		rank := 0
		for i, other := range state.ClaimsOn(claim.Target) {
			if other.Heir == claim.Heir {
				rank = i + 1
			}
		}
		view.Claims = append(view.Claims, ClaimView{Heir: nobleCodesByID[claim.Heir], Target: nobleCodesByID[claim.Target], Spouse: nobleCodesByID[claim.Spouse], Turn: claim.Turn, WifeSide: claim.WifeSide, Rank: rank})
	}
	view.Marriages = []MarriageView{}
	if len(state.Marriages) != 0 {
		codes := make(map[models.NobleID]models.NobleCode, len(state.Nobles)+len(state.RemovedNobles))
		for _, noble := range state.Nobles {
			codes[noble.ID] = models.NobleCode(noble.Code)
		}
		for _, removed := range state.RemovedNobles {
			codes[removed.ID] = models.NobleCode(removed.Code)
		}
		for _, marriage := range state.Marriages {
			marriageView := MarriageView{NobleA: codes[marriage.NobleA], NobleB: codes[marriage.NobleB], Turn: marriage.Turn, Active: marriage.Active(state)}
			if category, ok := engine.MarriageCategory(state, balance, marriage); ok {
				marriageView.Category = category
				marriageView.Weight, _ = engine.AllianceWeight(state, balance, marriage)
				for _, player := range state.Players {
					if head, found := engine.ActiveHeadMarriage(state, balance, player.ID); found && head.NobleA == marriage.NobleA && head.NobleB == marriage.NobleB {
						marriageView.ActiveHeadFor = append(marriageView.ActiveHeadFor, player.ID)
						marriageView.HeadSuccessors = append(marriageView.HeadSuccessors, headSuccessor(state, balance, player.ID, marriage, codes))
					}
				}
			}
			view.Marriages = append(view.Marriages, marriageView)
		}
	}
	if viewer != nil && state.SpecialDeck != nil {
		cardKinds := make(map[models.SpecialCardID]models.CardKind, len(state.SpecialDeck.Cards))
		for _, card := range state.SpecialDeck.Cards {
			cardKinds[card.ID] = card.Kind
		}
		for _, cardID := range state.SpecialDeck.Hands[*viewer] {
			if kind, exists := cardKinds[cardID]; exists && kind.IsBonus() {
				view.SpecialHand = append(view.SpecialHand, kind)
			}
		}
	}
	if state.NobleDeck != nil {
		view.NobleDeckSize = len(state.NobleDeck.DrawPile)
		view.NobleDiscardSize = len(state.NobleDeck.Discard)
		if viewer != nil {
			for _, cardID := range state.NobleDeck.Hands[*viewer] {
				if card, exists := state.NobleDeck.Card(cardID); exists {
					view.NobleHand = append(view.NobleHand, NobleCardView{
						ID: card.ID, Kind: card.Kind, Code: card.Code, Name: card.Name, Sex: card.Sex, Dignity: card.Dignity,
					})
				}
			}
		}
	}
	if viewer != nil {
		view.CalamityForecast = calamityForecastFor(state, *viewer)
		view.OpenElections = engine.ForecastWinterElections(state, balance, *viewer)
		view.WinterAids = engine.ForecastWinterAids(state, balance, *viewer)
		view.SpiedHands = spiedHandsFor(state, *viewer)
	}
	view.RevoltTargets = engine.ForecastRevoltTargets(state, balance)
	view.AppeasementCostBase = balance.Religion.AppeasementCostBase
	view.ActiveRegionEffects = append([]models.ActiveRegionEffect(nil), state.ActiveRegionEffects...)
	view.Announcements = engine.PendingAnnouncements(state, state.Year(), state.Season, true)
	return view
}

func viewerKnowsChain(state *models.GameState, viewer models.PlayerID, chainID models.ChainID) bool {
	if viewer == models.SpectatorViewer {
		return true
	}
	if state == nil || state.Privacy == nil {
		return false
	}
	snapshots := state.Privacy.ChainKnowledge[viewer]
	_, known := snapshots[chainID]
	return known
}

func projectChain(
	chain models.Chain,
	nobleCodesByID map[models.NobleID]models.NobleCode,
) *ChainView {
	view := &ChainView{
		Noble:        nobleCodesByID[chain.NobleID],
		CurrentIndex: chain.CurrentIndex,
		Orders:       make([]OrderView, 0, len(chain.Orders)),
	}
	for _, order := range chain.Orders {
		view.Orders = append(view.Orders, projectOrder(order))
	}
	return view
}

// projectOrder converts one parsed or installed order to its public shape.
func projectOrder(order models.Order) OrderView {
	orderView := OrderView{
		Type:     order.Type,
		Position: order.PositionID,
		Liaison:  order.Liaison,
		Amount:   order.Amount,
	}
	if len(order.TargetIDs) != 0 {
		orderView.Targets = append([]models.TerritoryID(nil), order.TargetIDs...)
	}
	if len(order.NobleAssignments) != 0 {
		orderView.NobleAssignments = make(map[models.TerritoryID][]models.NobleCode, len(order.NobleAssignments))
		for destination, nobleCodes := range order.NobleAssignments {
			orderView.NobleAssignments[destination] = append([]models.NobleCode(nil), nobleCodes...)
		}
	}
	return orderView
}

// SpiedHandView is the hand of a player seen through a spy.
type SpiedHandView struct {
	Player      models.PlayerID   `json:"player"`
	SpecialHand []models.CardKind `json:"specialHand"`
	NobleHand   []NobleCardView   `json:"nobleHand"`
}

// calamityForecastFor is the next calamities of the special deck draw pile
// the astrologues serving the viewer reveal in winter: her own, or one she
// holds hostage.
func calamityForecastFor(state *models.GameState, viewer models.PlayerID) []models.CardKind {
	if state.Season != models.SeasonWinter {
		return nil
	}
	forecast := 0
	for _, noble := range state.Nobles {
		if noble.OwnerID == viewer || hostHolder(state, noble) == viewer {
			forecast = max(forecast, noble.CalamityForecast())
		}
	}
	var kinds []models.CardKind
	for _, calamity := range engine.CalamityForecast(state, forecast) {
		kinds = append(kinds, calamity.Kind)
	}
	return kinds
}

// spiedHandsFor lists the hands of the players that hold a spy of the viewer
// hostage.
func spiedHandsFor(state *models.GameState, viewer models.PlayerID) []SpiedHandView {
	var hands []SpiedHandView
	seen := map[models.PlayerID]bool{}
	for _, noble := range state.Nobles {
		host := hostHolder(state, noble)
		if noble.OwnerID != viewer || host == "" || seen[host] || !noble.SeesHostHand() {
			continue
		}
		seen[host] = true
		hand := SpiedHandView{Player: host, SpecialHand: []models.CardKind{}, NobleHand: []NobleCardView{}}
		if state.SpecialDeck != nil {
			kinds := make(map[models.SpecialCardID]models.CardKind, len(state.SpecialDeck.Cards))
			for _, card := range state.SpecialDeck.Cards {
				kinds[card.ID] = card.Kind
			}
			for _, cardID := range state.SpecialDeck.Hands[host] {
				hand.SpecialHand = append(hand.SpecialHand, kinds[cardID])
			}
		}
		if state.NobleDeck != nil {
			for _, cardID := range state.NobleDeck.Hands[host] {
				if card, exists := state.NobleDeck.Card(cardID); exists {
					hand.NobleHand = append(hand.NobleHand, NobleCardView{ID: card.ID, Kind: card.Kind, Code: card.Code, Name: card.Name, Sex: card.Sex, Dignity: card.Dignity})
				}
			}
		}
		hands = append(hands, hand)
	}
	return hands
}

// SecretIdentityView is the private identity of a noble replaced by a
// chevalier d'Éon.
type SecretIdentityView struct {
	Code models.NobleCode `json:"code"`
	Name string           `json:"name"`
	Sex  models.Sex       `json:"sex"`
}

// headSuccessor simulates the end of marriage and returns the marriage that
// would then be player's active head. The simulation drops the marriage from a
// copy of the state, so alliance weights (density bonus included) are
// recomputed without it.
func headSuccessor(state *models.GameState, balance assetgen.Balance, player models.PlayerID, marriage models.Marriage, codes map[models.NobleID]models.NobleCode) HeadSuccessorView {
	without := *state
	without.Marriages = make([]models.Marriage, 0, len(state.Marriages))
	for _, other := range state.Marriages {
		if other.NobleA != marriage.NobleA || other.NobleB != marriage.NobleB {
			without.Marriages = append(without.Marriages, other)
		}
	}
	view := HeadSuccessorView{Player: player}
	if next, found := engine.ActiveHeadMarriage(&without, balance, player); found {
		view.Marriage = &MarriageRef{NobleA: codes[next.NobleA], NobleB: codes[next.NobleB]}
	}
	return view
}

// projectReligion exposes the bishoprics and the religious standings, all
// public, naming nobles by their public code.
func projectReligion(state *models.GameState, codes map[models.NobleID]models.NobleCode) ([]BishopricView, []models.NobleCode, *models.NobleCode, []ExcommunicationView) {
	bishoprics := make([]BishopricView, 0, len(state.Regions))
	for _, region := range state.Regions {
		bishopric := BishopricView{Region: region.ID, Name: region.Name, Territories: append([]models.TerritoryID(nil), region.Territories...)}
		if bishop, exists := state.BishopOf(region.ID); exists {
			code := codes[bishop]
			bishopric.Bishop = &code
		}
		bishoprics = append(bishoprics, bishopric)
	}
	cardinals := make([]models.NobleCode, 0, len(state.Cardinals))
	for _, cardinal := range state.Cardinals {
		cardinals = append(cardinals, codes[cardinal])
	}
	var pope *models.NobleCode
	if state.Pope != nil {
		code := codes[*state.Pope]
		pope = &code
	}
	excommunicated := make([]ExcommunicationView, 0, len(state.Excommunications))
	for _, excommunication := range state.Excommunications {
		excommunicated = append(excommunicated, ExcommunicationView{Noble: codes[excommunication.Noble], Reason: excommunication.Reason, Turn: excommunication.Turn})
	}
	return bishoprics, cardinals, pope, excommunicated
}
