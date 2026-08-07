package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/demo"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

// StateView is the state.json representation served to the frontend. It keeps
// dynamic entities nested under their territory instead of serializing the
// storage-oriented GameState directly.
type StateView struct {
	Turn        int                        `json:"turn"`
	Season      models.Season              `json:"season"`
	AsOf        map[models.TerritoryID]int `json:"asOf"`
	Players     []PlayerView               `json:"players"`
	Territories []TerritoryView            `json:"territories"`
	Nobles      []NobleView                `json:"nobles"`
}

// PlayerView contains the public player metadata needed by the hotseat
// selector. Player-specific visibility is deferred to the information layer.
type PlayerView struct {
	ID    models.PlayerID `json:"id"`
	Name  string          `json:"name"`
	Color string          `json:"color"`
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
// The development endpoint deliberately exposes every chain until a player
// identity and private projection arrive in P2.2/P3.2.
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
}

// OrderView is one public order. Territory and noble references use their
// trigrams instead of internal IDs so the frontend can address map entities.
type OrderView struct {
	Type             models.OrderType                            `json:"type"`
	Position         models.TerritoryCode                        `json:"position"`
	Targets          []models.TerritoryCode                      `json:"targets,omitempty"`
	NobleTargets     []models.NobleCode                          `json:"nobleTargets,omitempty"`
	NobleAssignments map[models.TerritoryCode][]models.NobleCode `json:"nobleAssignments,omitempty"`
	Liaison          models.LiaisonMode                          `json:"liaison"`
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

func projectState(state *models.GameState, freshness map[models.TerritoryID]int) StateView {
	view := StateView{
		AsOf:        make(map[models.TerritoryID]int),
		Players:     []PlayerView{},
		Territories: []TerritoryView{},
		Nobles:      []NobleView{},
	}
	if state == nil {
		return view
	}

	view.Turn = state.Turn
	view.Season = state.Season
	view.Players = make([]PlayerView, 0, len(state.Players))
	for _, player := range state.Players {
		view.Players = append(view.Players, PlayerView{ID: player.ID, Name: player.Name, Color: player.Color})
	}
	view.AsOf = make(map[models.TerritoryID]int, len(state.Territories))
	view.Territories = make([]TerritoryView, 0, len(state.Territories))
	view.Nobles = make([]NobleView, 0, len(state.Nobles))

	armiesByID := make(map[models.ArmyID]models.Army, len(state.Armies))
	for _, army := range state.Armies {
		armiesByID[army.ID] = army
	}
	territoryCodesByID := make(map[models.TerritoryID]models.TerritoryCode, len(state.Territories))
	for _, territory := range state.Territories {
		territoryCodesByID[territory.ID] = models.TerritoryCode(territory.Code)
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

	for _, territory := range state.Territories {
		territoryState := state.TerritoryStates[territory.ID]
		observedTurn, ok := freshness[territory.ID]
		if !ok {
			observedTurn = state.Turn
		}
		view.AsOf[territory.ID] = observedTurn

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
					armyView.Chain = projectChain(chain, territoryCodesByID, nobleCodesByID)
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
	return view
}

func projectChain(
	chain models.Chain,
	territoryCodesByID map[models.TerritoryID]models.TerritoryCode,
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
			Position: territoryCodesByID[order.PositionID],
			Liaison:  order.Liaison,
		}
		if len(order.TargetIDs) != 0 {
			orderView.Targets = make([]models.TerritoryCode, 0, len(order.TargetIDs))
			for _, targetID := range order.TargetIDs {
				orderView.Targets = append(orderView.Targets, territoryCodesByID[targetID])
			}
		}
		if len(order.NobleTargetIDs) != 0 {
			orderView.NobleTargets = make([]models.NobleCode, 0, len(order.NobleTargetIDs))
			for _, targetID := range order.NobleTargetIDs {
				orderView.NobleTargets = append(orderView.NobleTargets, nobleCodesByID[targetID])
			}
		}
		if len(order.NobleAssignments) != 0 {
			orderView.NobleAssignments = make(map[models.TerritoryCode][]models.NobleCode, len(order.NobleAssignments))
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
		stateJSON, err := json.Marshal(projectState(state, demo.DemoFreshness(state)))
		if err != nil {
			return nil, err
		}
		cache[players] = stateJSON
		return stateJSON, nil
	}
}
