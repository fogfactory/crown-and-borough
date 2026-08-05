package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
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
	mapData, err := generate(r.seed, r.assets, mapgen.Config{
		Width:        1000,
		Height:       700,
		SiteCount:    mapgen.TerritoriesPerPlayer * players,
		VillageCount: players + 1,
	})
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

func main() {
	assetsDir := os.Getenv("ASSETS_DIR")
	if assetsDir == "" {
		assetsDir = "assets"
	}
	assets, err := assetgen.Load(assetsDir)
	if err != nil {
		log.Fatalf("failed to load assets: %v", err)
	}
	log.Printf("assets loaded from %s: %d communes, %d prenoms", assetsDir, len(assets.Communes), len(assets.Prenoms))

	seed := os.Getenv("SEED")
	if seed == "" {
		seed = defaultSeed
	}
	resolver := &mapResolver{seed: seed, assets: assets}
	if _, err := resolver.resolve(api.DefaultPlayers); err != nil {
		log.Fatalf("failed to generate map: %v", err)
	}
	resolveState := api.StateResolver(resolver.resolveData, seed, assets)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.WithCORS(newServer(resolver.resolve, resolveState))); err != nil {
		log.Fatal(err)
	}
}
