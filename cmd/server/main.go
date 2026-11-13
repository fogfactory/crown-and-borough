package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/auth"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/notify"
	"github.com/fogfactory/crown-and-borough/internal/store"
	firestorestore "github.com/fogfactory/crown-and-borough/internal/store/firestore"
	webassets "github.com/fogfactory/crown-and-borough/web"
)

const (
	defaultSeed    = "crown-and-borough-dev"
	defaultPlayers = 4
)

var version = "dev"

// hotseatHost is the development actor that owns the local hotseat game. It
// is also the actor used when a development request names no player.
const hotseatHost = "P1"

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
	if memory, ok := gameStore.(*store.MemoryStore); ok {
		if err := createHotseatGame(context.Background(), memory); err != nil {
			log.Fatalf("failed to create default game: %v", err)
		}
		applyDemoScenario(memory)
	}
	if persistent, ok := gameStore.(*firestorestore.FirestoreStore); ok {
		if _, restoreErr := persistent.Restore(context.Background()); restoreErr != nil {
			log.Fatalf("failed to restore Firestore games: %v", restoreErr)
		}
	}
	readinessChecks := make([]api.ReadinessCheck, 0, 2)
	if persistent, ok := gameStore.(*firestorestore.FirestoreStore); ok {
		readinessChecks = append(readinessChecks, persistent.Ready)
	}

	options := serverOptions{
		Rules:       rules,
		Balance:     balance,
		Store:       gameStore,
		Development: onlineDevMode,
		CreatorGate: creatorGateForEnvironment(onlineDevMode),
		Readiness:   readinessChecks,
		SeedAssets:  &assets,
		BugIssues:   bugIssueClient(),
	}
	if !onlineDevMode {
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		if projectID == "" {
			projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}
		verifier, verifierErr := auth.NewFirebaseVerifierWithOptions(context.Background(), projectID, auth.FirebaseVerifierOptions{CheckRevoked: true})
		if verifierErr != nil {
			log.Fatalf("failed to initialize Firebase Auth: %v", verifierErr)
		}
		options.Actor = api.FirebaseActorResolver(verifier)
		options.Readiness = append(options.Readiness, verifier.Ready)
	}
	server := newApplicationServer(options)

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

// bugIssueClient authenticates as the GitHub App described by
// GITHUB_BUG_APP_ID, GITHUB_BUG_APP_INSTALLATION_ID and
// GITHUB_BUG_APP_PRIVATE_KEY, or else with the GITHUB_BUG_TOKEN bot token
// (local development). Without either the route is disabled.
func bugIssueClient() api.IssueCreator {
	const repository = "fogfactory/crown-and-borough"
	app, err := api.NewGitHubAppIssueClient(
		os.Getenv("GITHUB_BUG_APP_ID"),
		os.Getenv("GITHUB_BUG_APP_INSTALLATION_ID"),
		os.Getenv("GITHUB_BUG_APP_PRIVATE_KEY"),
		repository,
	)
	if err != nil {
		log.Printf("bug reports: GitHub App disabled: %v", err)
	}
	if app != nil {
		return app
	}
	if token := api.NewGitHubIssueClient(os.Getenv("GITHUB_BUG_TOKEN"), repository); token != nil {
		return token
	}
	return nil
}

func newGameStore(ctx context.Context, balance assetgen.Balance, assets assetgen.Assets, onlineDevMode bool) (store.GameStore, func(), error) {
	if onlineDevMode && strings.TrimSpace(os.Getenv("FIRESTORE_EMULATOR_HOST")) == "" {
		return store.NewMemoryStoreWithOptions(balance, assets, store.MemoryStoreOptions{
			PrivacyTracker:   api.TrackTurnPrivacy,
			StrictMembership: false,
			MaximumPlayers:   engine.MaximumGamePlayers,
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

// createHotseatGame creates the local hotseat game described by SEED and
// PLAYERS, owned by hotseatHost, so the hotseat frontend opens on a game.
func createHotseatGame(ctx context.Context, gameStore store.GameStore) error {
	seed := os.Getenv("SEED")
	if seed == "" {
		seed = defaultSeed
	}
	playerCount, err := parsePlayerCount(os.Getenv("PLAYERS"), defaultPlayers)
	if err != nil {
		return fmt.Errorf("parse PLAYERS: %w", err)
	}
	players := make([]engine.PlayerInit, playerCount)
	for index := range players {
		players[index] = engine.PlayerInit{Name: fmt.Sprintf("P%d", index+1)}
	}
	_, err = gameStore.Create(ctx, store.Actor{ID: hotseatHost, Development: true}, store.CreateRequest{
		Name:    "Hotseat",
		Seed:    seed,
		Players: players,
	})
	return err
}

func parsePlayerCount(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("players must be an integer: %w", err)
	}
	return count, nil
}

// serverOptions configures the application server. Development trusts the
// ?player= query and X-Dev-Player header (ONLINE_DEV_MODE); otherwise Actor
// must authenticate requests.
type serverOptions struct {
	Rules       assetgen.Rules
	Balance     assetgen.Balance
	Store       store.GameStore
	Development bool
	Actor       api.ActorResolver
	CreatorGate api.CreatorGate
	Readiness   []api.ReadinessCheck
	SeedAssets  *assetgen.Assets
	// BugIssues files the issues of POST /api/bugs; nil disables the route.
	BugIssues api.IssueCreator
}

func newApplicationServer(options serverOptions) *http.ServeMux {
	resolveActor := options.Actor
	if resolveActor == nil {
		if options.Development {
			resolveActor = api.DevActorResolver(hotseatHost)
		} else {
			resolveActor = api.BearerActorResolver(func(string) (store.Actor, error) {
				return store.Actor{}, api.ErrUnauthorized
			})
		}
	}

	// Email notifications and the activity they rely on are online only, like
	// the chat: hotseat has no accounts to write to.
	var notifier api.Notifier
	if state, ok := options.Store.(store.NotificationStateStore); ok && !options.Development {
		mailer, mailerErr := notify.NewMailerFromEnv(os.Getenv)
		if mailerErr != nil {
			log.Fatalf("failed to configure email notifications: %v", mailerErr)
		}
		if _, silent := mailer.(notify.LogMailer); silent {
			log.Print("email notifications: SMTP_HOST or SMTP_FROM is empty, emails are logged and skipped")
		}
		if profileReader, ok := options.Store.(notify.ProfileReader); ok {
			notifier = notify.NewNotifier(mailer, profileReader, state, os.Getenv("PUBLIC_APP_URL"))
		} else {
			log.Print("email notifications: disabled, the store has no player profiles")
		}
		resolveActor = api.WithActivity(resolveActor, notify.NewActivityTracker(state))
	}

	mux := http.NewServeMux()
	mountVersion(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /api/rules", api.RulesHandler(options.Rules))
	profiles, _ := options.Store.(store.ProfileStore)
	// The chat is online only: hotseat has no accounts to sign messages for.
	var chat *store.ChatService
	if backend, ok := options.Store.(store.ChatBackend); ok && !options.Development {
		chat = store.NewChatService(options.Store, backend)
	}
	games := api.NewGamesHandlerWithOptions(options.Store, options.Rules, api.GamesHandlerOptions{
		Actor:            resolveActor,
		Balance:          options.Balance,
		Profiles:         profiles,
		RequireProfile:   !options.Development,
		StrictMembership: !options.Development,
		InviteBaseURL:    os.Getenv("PUBLIC_APP_URL"),
		CreatorGate:      options.CreatorGate,
		Chat:             chat,
		Notifier:         notifier,
	})
	// The hotseat client does not declare a build: only the hosted server checks it.
	var gamesRoute http.Handler = games
	if !options.Development {
		gamesRoute = api.RequireClientVersion(applicationVersion(), games)
	}
	mux.Handle("/api/games", gamesRoute)
	mux.Handle("/api/games/", gamesRoute)
	mux.Handle("/api/auth/", api.NewAuthHandler(profiles, resolveActor))
	// Firestore keeps bug reports in their own collection; without it the
	// handler keeps them in memory.
	bugReports, _ := options.Store.(store.BugReportStore)
	mux.Handle("/api/bugs", api.NewBugsHandler(api.BugHandlerOptions{Actor: resolveActor, Issues: options.BugIssues, State: games.PlayerState, Reports: bugReports}))
	if options.SeedAssets != nil {
		mux.Handle("GET /api/seed", api.SeedHandler(*options.SeedAssets))
	}
	if len(options.Readiness) > 0 {
		mux.HandleFunc("GET /healthz/ready", api.ReadinessHandler(options.Readiness...))
	}
	mountFrontend(mux)
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
