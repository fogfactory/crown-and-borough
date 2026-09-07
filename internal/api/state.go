package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/demo"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// StateView is the state.json representation served to the frontend. It keeps
// dynamic entities nested under their territory instead of serializing the
// storage-oriented GameState directly, and includes the public calendar and
// score snapshot.
type StateView struct {
	Turn        int                                       `json:"turn"`
	Year        int                                       `json:"year"`
	YearCount   int                                       `json:"yearCount"`
	Season      models.Season                             `json:"season"`
	Scores      map[models.PlayerID]engine.ScoreBreakdown `json:"scores"`
	Finished    bool                                      `json:"finished"`
	Winner      *models.PlayerID                          `json:"winner,omitempty"`
	Players     []PlayerView                              `json:"players"`
	Territories []TerritoryView                           `json:"territories"`
	Nobles      []NobleView                               `json:"nobles"`
	SpecialHand []models.CardKind                         `json:"specialHand"`
}

// PlayerView contains the public player metadata needed by the hotseat
// selector. Player-specific filtering is a future server concern.
type PlayerView struct {
	ID               models.PlayerID     `json:"id"`
	Name             string              `json:"name"`
	Color            string              `json:"color"`
	CapitalTerritory *models.TerritoryID `json:"capitalTerritory,omitempty"`
}

// TerritoryView is the live state displayed on one map territory.
type TerritoryView struct {
	ID              models.TerritoryID `json:"id"`
	Owner           *models.PlayerID   `json:"owner"`
	Resources       int                `json:"resources"`
	Army            *ArmyView          `json:"army"`
	Infrastructures []InfraView        `json:"infrastructures"`
}

// ArmyView contains the visible owner, size, and current chain of an army. Its
// ID is an internal storage detail: the frontend addresses an army by territory.
// The current v1 endpoint exposes every chain; server-side player filtering is
// tracked as a later online feature.
type ArmyView struct {
	Owner models.PlayerID `json:"owner"`
	Size  int             `json:"size"`
	Chain *ChainView      `json:"chain"`
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
}

// InfraView contains the visible kind and level of an infrastructure.
type InfraView struct {
	Type  models.InfraType `json:"type"`
	Level int              `json:"level"`
}

// NobleView contains the visible identity, code, status, owner, and location
// of a noble.
type NobleView struct {
	ID       models.NobleID     `json:"id"`
	Code     models.NobleCode   `json:"code"`
	Name     string             `json:"name"`
	Owner    models.PlayerID    `json:"owner"`
	Location models.TerritoryID `json:"location"`
	Status   models.NobleStatus `json:"status"`
}

func projectState(state *models.GameState) StateView {
	return projectStateForViewer(state, nil)
}

// ProjectState returns the public state projection used by the development
// session. Hosted callers should use ProjectStateForPlayer so chain knowledge
// is applied to the viewer.
func ProjectState(state *models.GameState) StateView {
	return projectState(state)
}

func projectStateForPlayer(state *models.GameState, playerID models.PlayerID) StateView {
	return projectStateForViewer(state, &playerID)
}

// ProjectStateForPlayer returns the server-filtered state projection for one
// player. It is exported so persistence adapters can materialize the same
// projection that the REST API returns without importing Firestore into the
// engine or models packages.
func ProjectStateForPlayer(state *models.GameState, playerID models.PlayerID) StateView {
	return projectStateForPlayer(state, playerID)
}

func projectStateForViewer(state *models.GameState, viewer *models.PlayerID) StateView {
	view := StateView{
		Players:     []PlayerView{},
		Territories: []TerritoryView{},
		Nobles:      []NobleView{},
		SpecialHand: []models.CardKind{},
		Scores:      map[models.PlayerID]engine.ScoreBreakdown{},
	}
	if state == nil {
		return view
	}

	view.Turn = state.Turn
	view.Year = state.Year()
	view.YearCount = state.YearCount
	view.Season = state.Season
	view.Scores = engine.ComputeScores(state)
	view.Finished = engine.GameFinished(state)
	view.Winner = engine.WinnerForFinishedGame(state)
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
	for _, player := range state.Players {
		playerView := PlayerView{ID: player.ID, Name: player.Name, Color: player.Color}
		if player.CapitalCastleID != nil {
			if infrastructure, ok := infrastructuresByID[*player.CapitalCastleID]; ok && infrastructure.Type == models.InfraTypeCastle {
				capitalTerritory := infrastructure.TerritoryID
				playerView.CapitalTerritory = &capitalTerritory
			}
		}
		view.Players = append(view.Players, playerView)
	}

	for _, territory := range state.Territories {
		territoryState := state.TerritoryStates[territory.ID]
		territoryView := TerritoryView{
			ID:              territory.ID,
			Owner:           territoryState.OwnerID,
			Resources:       territoryState.Resources,
			Infrastructures: make([]InfraView, 0, len(territoryState.Infrastructures)),
		}
		if territoryState.Army != nil {
			if army, ok := armiesByID[*territoryState.Army]; ok {
				armyView := &ArmyView{
					Owner: army.OwnerID,
					Size:  army.Size,
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
		for _, infrastructureID := range territoryState.Infrastructures {
			if infrastructure, ok := infrastructuresByID[infrastructureID]; ok {
				territoryView.Infrastructures = append(territoryView.Infrastructures, InfraView{
					Type:  infrastructure.Type,
					Level: infrastructure.Level,
				})
			}
		}
		view.Territories = append(view.Territories, territoryView)
	}
	for _, noble := range state.Nobles {
		view.Nobles = append(view.Nobles, NobleView{
			ID:       noble.ID,
			Code:     models.NobleCode(noble.Code),
			Name:     noble.Name,
			Owner:    noble.OwnerID,
			Location: noble.LocationID,
			Status:   noble.Status,
		})
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
	return view
}

func viewerKnowsChain(state *models.GameState, viewer models.PlayerID, chainID models.ChainID) bool {
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
		orderView := OrderView{
			Type:     order.Type,
			Position: order.PositionID,
			Liaison:  order.Liaison,
		}
		if len(order.TargetIDs) != 0 {
			orderView.Targets = make([]models.TerritoryID, 0, len(order.TargetIDs))
			for _, targetID := range order.TargetIDs {
				orderView.Targets = append(orderView.Targets, targetID)
			}
		}
		if len(order.NobleAssignments) != 0 {
			orderView.NobleAssignments = make(map[models.TerritoryID][]models.NobleCode, len(order.NobleAssignments))
			for destination, nobleCodes := range order.NobleAssignments {
				orderView.NobleAssignments[destination] = append([]models.NobleCode(nil), nobleCodes...)
			}
		}
		view.Orders = append(view.Orders, orderView)
	}
	return view
}

// StateHandler resolves and serves the development state for the requested
// player count.
func StateHandler(resolve func(players int) ([]byte, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		players, ok := requestedPlayers(r)
		if !ok {
			http.Error(w, "invalid players", http.StatusBadRequest)
			return
		}

		stateJSON, err := resolve(players)
		if err != nil {
			http.Error(w, "failed to resolve state", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(stateJSON)
	}
}

// StateResolver remains available for fixture-oriented tests and tools. The
// production hotseat server uses Session.StateHTTP instead, so this resolver is
// never the source of the live game state.
func StateResolver(
	mapData func(players int) (mapgen.MapData, error),
	seed string,
	assets assetgen.Assets,
) func(players int) ([]byte, error) {
	var mu sync.Mutex
	cache := make(map[int][]byte)

	return func(players int) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()

		if stateJSON, ok := cache[players]; ok {
			return stateJSON, nil
		}

		data, err := mapData(players)
		if err != nil {
			return nil, err
		}
		state, err := demo.DemoState(seed, assets, data, players)
		if err != nil {
			return nil, err
		}
		stateJSON, err := json.Marshal(projectState(state))
		if err != nil {
			return nil, err
		}
		cache[players] = stateJSON
		return stateJSON, nil
	}
}
