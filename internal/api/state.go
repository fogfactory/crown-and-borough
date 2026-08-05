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
	Territories []TerritoryView            `json:"territories"`
	Nobles      []NobleView                `json:"nobles"`
}

// TerritoryView is the live state displayed on one map territory.
type TerritoryView struct {
	ID              models.TerritoryID `json:"id"`
	Owner           *models.PlayerID   `json:"owner"`
	Resources       int                `json:"resources"`
	Troops          []TroopView        `json:"troops"`
	Infrastructures []InfraView        `json:"infrastructures"`
}

// TroopView contains the visible identity and owner of a troop.
type TroopView struct {
	ID    models.TroopID  `json:"id"`
	Owner models.PlayerID `json:"owner"`
}

// InfraView contains the visible kind and level of an infrastructure.
type InfraView struct {
	Type  models.InfraType `json:"type"`
	Level int              `json:"level"`
}

// NobleView contains the visible identity, owner and location of a noble.
type NobleView struct {
	ID       models.NobleID     `json:"id"`
	Name     string             `json:"name"`
	Owner    models.PlayerID    `json:"owner"`
	Location models.TerritoryID `json:"location"`
}

func projectState(state *models.GameState, freshness map[models.TerritoryID]int) StateView {
	view := StateView{
		AsOf:        make(map[models.TerritoryID]int),
		Territories: []TerritoryView{},
		Nobles:      []NobleView{},
	}
	if state == nil {
		return view
	}

	view.Turn = state.Turn
	view.Season = state.Season
	view.AsOf = make(map[models.TerritoryID]int, len(state.Territories))
	view.Territories = make([]TerritoryView, 0, len(state.Territories))
	view.Nobles = make([]NobleView, 0, len(state.Nobles))

	troopsByID := make(map[models.TroopID]models.Troop, len(state.Troops))
	for _, troop := range state.Troops {
		troopsByID[troop.ID] = troop
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
			Troops:          make([]TroopView, 0, len(territoryState.Troops)),
			Infrastructures: make([]InfraView, 0, len(territoryState.Infrastructures)),
		}
		for _, troopID := range territoryState.Troops {
			if troop, ok := troopsByID[troopID]; ok {
				territoryView.Troops = append(territoryView.Troops, TroopView{
					ID:    troop.ID,
					Owner: troop.OwnerID,
				})
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
			Name:     noble.Name,
			Owner:    noble.OwnerID,
			Location: noble.LocationID,
		})
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

// StateResolver serves a static development state only. P1.7 replaces this
// with the dynamic resolver driven by turn reports and hotseat play.
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
