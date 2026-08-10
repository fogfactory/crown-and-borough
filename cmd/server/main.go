package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
)

const defaultSeed = "crown-and-borough-dev"

type mapGenerator func(string, assetgen.Assets, mapgen.Config) (mapgen.MapData, error)

type mapResolver struct {
	seed     string
	assets   assetgen.Assets
	generate mapGenerator

	mu sync.Mutex
	// A resolver owns one fixed seed, so players is the remaining component of
	// the semantic (seed, players) cache key.
	cache map[int]mapgen.MapData
}

func (r *mapResolver) resolve(players int) ([]byte, error) {
	mapData, err := r.resolveData(players)
	if err != nil {
		return nil, err
	}
	return json.Marshal(mapData)
}

func (r *mapResolver) resolveData(players int) (mapgen.MapData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if mapData, ok := r.cache[players]; ok {
		return mapData, nil
	}

	generate := r.generate
	if generate == nil {
		generate = mapgen.Generate
	}
	mapData, err := generate(r.seed, r.assets, engine.GameMapConfig(players))
	if err != nil {
		return mapgen.MapData{}, err
	}

	if r.cache == nil {
		r.cache = make(map[int]mapgen.MapData)
	}
	r.cache[players] = mapData
	log.Printf("map generated: seed=%q players=%d", r.seed, players)
	return mapData, nil
}

func newServer(
	resolveMap func(players int) ([]byte, error),
	resolveState func(players int) ([]byte, error),
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /api/map", api.MapHandler(resolveMap))
	mux.Handle("GET /api/state", api.StateHandler(resolveState))
	return mux
}

func newHotseatServer(session *api.Session, rules assetgen.Rules) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/map", session.MapHTTP)
	mux.HandleFunc("GET /api/state", session.StateHTTP)
	mux.HandleFunc("GET /api/supply", session.SupplyHTTP)
	mux.Handle("GET /api/rules", api.RulesHandler(rules))
	mux.HandleFunc("POST /api/game", session.GameHTTP)
	mux.HandleFunc("POST /api/orders", session.OrdersHTTP)
	mux.HandleFunc("POST /api/reset", session.ResetHTTP)
	return mux
}

func main() {
	assetsDir := os.Getenv("ASSETS_DIR")
	if assetsDir == "" {
		assetsDir = "assets"
	}
	assets, err := assetgen.Load(assetsDir)
	if err != nil {
		log.Fatalf("failed to load assets: %v", err)
	}
	balance, err := assetgen.LoadBalance(assetsDir)
	if err != nil {
		log.Fatalf("failed to load balance: %v", err)
	}
	rules, err := assetgen.LoadRules(assetsDir)
	if err != nil {
		log.Fatalf("failed to load player rules: %v", err)
	}
	log.Printf("assets loaded from %s: %d communes, %d prenoms", assetsDir, len(assets.Communes), len(assets.Prenoms))

	seed := os.Getenv("SEED")
	if seed == "" {
		seed = defaultSeed
	}
	playerCount, err := api.ParsePlayerCount(os.Getenv("PLAYERS"), api.DefaultPlayers)
	if err != nil {
		log.Fatalf("failed to parse PLAYERS: %v", err)
	}
	players := make([]engine.PlayerInit, playerCount)
	for index := range players {
		players[index] = engine.PlayerInit{ID: enginePlayerID(index + 1), Name: enginePlayerName(index + 1)}
	}
	session, err := api.NewSession(seed, players, balance, assets)
	if err != nil {
		log.Fatalf("failed to create default game: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.WithCORS(newHotseatServer(session, rules))); err != nil {
		log.Fatal(err)
	}
}

func enginePlayerID(index int) models.PlayerID {
	return models.PlayerID(fmt.Sprintf("P%d", index))
}

func enginePlayerName(index int) string {
	return fmt.Sprintf("P%d", index)
}
