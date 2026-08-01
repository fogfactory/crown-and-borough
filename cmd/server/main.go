package main

import (
	"log"
	"net/http"
	"os"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

func newServer() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
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
	log.Printf("assets loaded from %s: %d communes, %d prenoms, %d qualificatifs",
		assetsDir, len(assets.Communes), len(assets.Prenoms), len(assets.Qualificatifs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, newServer()); err != nil {
		log.Fatal(err)
	}
}
