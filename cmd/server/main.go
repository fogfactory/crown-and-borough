package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/auth"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/engine/mapgen"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
	firestorestore "github.com/fogfactory/crown-and-borough/internal/store/firestore"
	webassets "github.com/fogfactory/crown-and-borough/web"
)

const defaultSeed = "crown-and-borough-dev"

var version = "dev"

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
	mountVersion(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /api/map", api.MapHandler(resolveMap))
	mux.Handle("GET /api/state", api.StateHandler(resolveState))
	mountFrontend(mux)
	return mux
}

func newHotseatServer(session *api.Session, rules assetgen.Rules) *http.ServeMux {
	mux := http.NewServeMux()
	mountVersion(mux)
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
	mountFrontend(mux)
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
	rules, err := assetgen.LoadRules(assetsDir, balance)
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
	onlineDevMode := os.Getenv("ONLINE_DEV_MODE") == "true"
	publicAppURL := strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
	if !onlineDevMode && publicAppURL == "" {
		log.Fatal("PUBLIC_APP_URL is required outside ONLINE_DEV_MODE")
	}
	gameStore, closeGameStore, storeErr := newGameStore(context.Background(), balance, assets, onlineDevMode)
	if storeErr != nil {
		log.Fatalf("failed to initialize game store: %v", storeErr)
	}
	defer closeGameStore()
	if persistent, ok := gameStore.(*firestorestore.FirestoreStore); ok {
		if _, restoreErr := persistent.Restore(context.Background()); restoreErr != nil {
			log.Fatalf("failed to restore Firestore games: %v", restoreErr)
		}
	}
	readinessChecks := make([]api.ReadinessCheck, 0, 2)
	if persistent, ok := gameStore.(*firestorestore.FirestoreStore); ok {
		readinessChecks = append(readinessChecks, persistent.Ready)
	}

	var resolveActor api.ActorResolver
	if onlineDevMode {
		resolveActor = api.DevActorResolver("P1")
	} else {
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}
		verifier, verifierErr := auth.NewFirebaseVerifierWithOptions(context.Background(), projectID, auth.FirebaseVerifierOptions{CheckRevoked: true})
		if verifierErr != nil {
			log.Fatalf("failed to initialize Firebase Auth: %v", verifierErr)
		}
		resolveActor = api.FirebaseActorResolver(verifier)
		readinessChecks = append(readinessChecks, verifier.Ready)
	}
	server := newApplicationServerWithCreatorGate(
		session,
		rules,
		gameStore,
		onlineDevMode,
		resolveActor,
		creatorGateForEnvironment(onlineDevMode),
		readinessChecks,
		assets,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, api.WithCORSMode(server, onlineDevMode)); err != nil {
		log.Fatal(err)
	}
}

func newGameStore(ctx context.Context, balance assetgen.Balance, assets assetgen.Assets, onlineDevMode bool) (store.GameStore, func(), error) {
	if onlineDevMode && strings.TrimSpace(os.Getenv("FIRESTORE_EMULATOR_HOST")) == "" {
		return store.NewMemoryStoreWithOptions(balance, assets, store.MemoryStoreOptions{
			PrivacyTracker:   api.TrackTurnPrivacy,
			StrictMembership: false,
		}), func() {}, nil
	}
	persistent, err := firestorestore.NewFromEnv(ctx, balance, assets, firestorestore.Options{
		PrivacyTracker:   api.TrackTurnPrivacy,
		StrictMembership: !onlineDevMode,
	})
	if err != nil {
		return nil, nil, err
	}
	return persistent, func() { _ = persistent.Close() }, nil
}

func newApplicationServer(session *api.Session, rules assetgen.Rules, gameStore store.GameStore, onlineDevMode bool) *http.ServeMux {
	var resolveActor api.ActorResolver
	if onlineDevMode {
		resolveActor = api.DevActorResolver("P1")
	} else {
		resolveActor = api.BearerActorResolver(func(string) (store.Actor, error) {
			return store.Actor{}, api.ErrUnauthorized
		})
	}
	return newApplicationServerWithResolver(session, rules, gameStore, onlineDevMode, resolveActor)
}

func newApplicationServerWithResolver(session *api.Session, rules assetgen.Rules, gameStore store.GameStore, onlineDevMode bool, resolveActor api.ActorResolver, seedAssets ...assetgen.Assets) *http.ServeMux {
	return newApplicationServerWithResolverAndReadiness(session, rules, gameStore, onlineDevMode, resolveActor, nil, seedAssets...)
}

func newApplicationServerWithResolverAndReadiness(session *api.Session, rules assetgen.Rules, gameStore store.GameStore, onlineDevMode bool, resolveActor api.ActorResolver, readinessChecks []api.ReadinessCheck, seedAssets ...assetgen.Assets) *http.ServeMux {
	return newApplicationServerWithCreatorGate(session, rules, gameStore, onlineDevMode, resolveActor, nil, readinessChecks, seedAssets...)
}

func newApplicationServerWithCreatorGate(session *api.Session, rules assetgen.Rules, gameStore store.GameStore, onlineDevMode bool, resolveActor api.ActorResolver, creatorGate api.CreatorGate, readinessChecks []api.ReadinessCheck, seedAssets ...assetgen.Assets) *http.ServeMux {
	var mux *http.ServeMux
	if onlineDevMode {
		mux = newHotseatServer(session, rules)
	} else {
		mux = http.NewServeMux()
		mountVersion(mux)
		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.Handle("GET /api/rules", api.RulesHandler(rules))
	}
	profiles, _ := gameStore.(store.ProfileStore)
	games := api.NewGamesHandlerWithOptions(gameStore, rules, api.GamesHandlerOptions{
		Actor:            resolveActor,
		Profiles:         profiles,
		RequireProfile:   !onlineDevMode,
		StrictMembership: !onlineDevMode,
		InviteBaseURL:    os.Getenv("PUBLIC_APP_URL"),
		CreatorGate:      creatorGate,
	})
	mux.Handle("/api/games", games)
	mux.Handle("/api/games/", games)
	mux.Handle("/api/auth/", api.NewAuthHandler(profiles, resolveActor))
	if len(seedAssets) > 0 {
		mux.Handle("GET /api/seed", api.SeedHandler(seedAssets[0]))
	}
	if len(readinessChecks) > 0 {
		mux.HandleFunc("GET /healthz/ready", api.ReadinessHandler(readinessChecks...))
	}
	if !onlineDevMode {
		mountFrontend(mux)
	}
	return mux
}

func mountFrontend(mux *http.ServeMux) {
	// The same binary serves the SPA in hotseat, emulator, and hosted modes.
	mux.Handle("/", webassets.NewEmbeddedHandler())
}

func mountVersion(mux *http.ServeMux) {
	mux.Handle("GET /api/version", api.VersionHandler(applicationVersion()))
}

func applicationVersion() string {
	if value := strings.TrimSpace(version); value != "" && value != "dev" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("APP_VERSION")); value != "" {
		return value
	}
	return "dev"
}

func enginePlayerID(index int) models.PlayerID {
	return models.PlayerID(fmt.Sprintf("P%d", index))
}

func enginePlayerName(index int) string {
	return fmt.Sprintf("P%d", index)
}
