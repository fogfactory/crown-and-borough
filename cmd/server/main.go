package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
)

const defaultSeed = "crown-and-borough-dev"

// devVillageCount scales the number of neutral villages laid on the map for
// the dev server. Fixed at 5 for now; P1.2d will introduce ?players=N and
// tie VillageCount = players + 1 (player starts land on distinct villages).
const devVillageCount = 5

func newServer(mapJSON []byte) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /api/map", api.MapHandler(mapJSON))
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
	log.Printf("generating map with seed %q", seed)
	mapData, err := mapgen.Generate(seed, assets, mapgen.Config{
		Width:        1000,
		Height:       700,
		SiteCount:    64,
		VillageCount: devVillageCount,
	})
	if err != nil {
		log.Fatalf("failed to generate map: %v", err)
	}
	mapJSON, err := json.Marshal(mapData)
	if err != nil {
		log.Fatalf("failed to marshal map: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.WithCORS(newServer(mapJSON))); err != nil {
		log.Fatal(err)
	}
}
