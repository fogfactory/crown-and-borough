package main

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// League runs one game between language-model houses.
type League struct {
	cfg  *Config
	auth *firebaseAuth

	bots     []*Bot
	host     *gameClient // the observer who created the game (emulator mode)
	watcher  *gameClient // any account that can read the game detail
	limiter  chan struct{}
	logMu    sync.Mutex
	logFile  *os.File
	sessionP string

	gameID     string
	inviteCode string
	slots      []PlayerSlot
	mapData    MapData
	rules      string
	session    session
	mu         sync.Mutex
	outage     error // set when a provider stays unreachable: the run stops, no turn is passed
}

type session struct {
	GameID     string            `json:"gameId"`
	InviteCode string            `json:"inviteCode"`
	Tokens     map[string]Tokens `json:"tokens"`
}

func newLeague(cfg *Config) (*League, error) {
	if err := os.MkdirAll(cfg.LeagueDir, 0o755); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(filepath.Join(cfg.LeagueDir, "league.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	l := &League{
		cfg:      cfg,
		auth:     newFirebaseAuth(cfg.Server),
		limiter:  make(chan struct{}, cfg.Concurrency),
		logFile:  logFile,
		sessionP: filepath.Join(cfg.LeagueDir, "session.json"),
		session:  session{Tokens: map[string]Tokens{}},
	}
	if raw, readErr := os.ReadFile(l.sessionP); readErr == nil {
		_ = json.Unmarshal(raw, &l.session)
		l.gameID, l.inviteCode = l.session.GameID, l.session.InviteCode
		if l.session.Tokens == nil {
			l.session.Tokens = map[string]Tokens{}
		}
	}
	return l, nil
}

func (l *League) logf(format string, args ...any) {
	line := fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	l.logMu.Lock()
	defer l.logMu.Unlock()
	fmt.Println(line)
	fmt.Fprintln(l.logFile, line)
}

func (l *League) saveSession() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.session.GameID = l.gameID
	l.session.InviteCode = l.inviteCode
	raw, _ := json.MarshalIndent(l.session, "", " ")
	_ = os.WriteFile(l.sessionP, raw, 0o600)
}

func (l *League) tokenSourceFor(email string) *tokenSource {
	return &tokenSource{
		auth:   l.auth,
		tokens: l.session.Tokens[email],
		onSave: func(tokens Tokens) { l.setTokens(email, tokens) },
	}
}

func (l *League) setTokens(email string, tokens Tokens) {
	l.mu.Lock()
	l.session.Tokens[email] = tokens
	l.mu.Unlock()
	l.saveSession()
}

func (l *League) houseNames() map[string]string {
	names := map[string]string{}
	for _, slot := range l.slots {
		names[slot.ID] = slot.Name
	}
	return names
}

// signIn returns a working session for an address, reusing a saved one.
func (l *League) signIn(ctx context.Context, email string) (*tokenSource, error) {
	if saved, ok := l.session.Tokens[email]; ok && saved.RefreshToken != "" {
		source := l.tokenSourceFor(email)
		if _, err := source.Token(ctx); err == nil {
			return source, nil
		}
	}
	if l.cfg.Server.Mode != "emulator" {
		return nil, fmt.Errorf("%s: %w (delete its entry in session.json and run again)", email, errNeedsLink)
	}
	tokens, err := l.auth.EmulatorSignIn(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("sign in %s: %w", email, err)
	}
	l.setTokens(email, tokens)
	return l.tokenSourceFor(email), nil
}

var errNeedsLink = errors.New("a sign-in link is needed")

// enrollHosted signs the bots in from the links pasted in links.txt, after
// asking Firebase to mail one to every address that has none yet.
func (l *League) enrollHosted(ctx context.Context) error {
	var missing []string
	for _, player := range l.cfg.Players {
		if _, ok := l.session.Tokens[player.Email]; !ok {
			missing = append(missing, player.Email)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	linksPath := filepath.Join(l.cfg.LeagueDir, "links.txt")
	pending := l.readLinks(linksPath)
	signed := 0
	for _, link := range pending {
		code, err := oobCodeFromLink(link.url)
		if err != nil {
			l.logf("links.txt: %v", err)
			continue
		}
		candidates := missing
		if link.email != "" {
			candidates = []string{link.email}
		}
		for _, email := range candidates {
			if _, done := l.session.Tokens[email]; done {
				continue
			}
			tokens, err := l.auth.SignInWithCode(ctx, email, code)
			if err != nil {
				continue
			}
			l.session.Tokens[email] = tokens
			signed++
			l.logf("signed in %s", email)
			break
		}
	}
	l.saveSession()
	var still []string
	for _, email := range missing {
		if _, ok := l.session.Tokens[email]; !ok {
			still = append(still, email)
		}
	}
	if len(still) == 0 {
		return nil
	}
	for _, email := range still {
		if err := l.auth.RequestLink(ctx, email); err != nil {
			l.logf("could not request a link for %s: %v", email, err)
		}
	}
	return fmt.Errorf("sign-in links requested for %d address(es): %s\nPaste each link from the mails into %s (one per line, optionally \"email link\"), then run the same command again",
		len(still), strings.Join(still, ", "), linksPath)
}

type pastedLink struct{ email, url string }

func (l *League) readLinks(path string) []pastedLink {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var links []pastedLink
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		switch {
		case len(fields) == 0 || strings.HasPrefix(fields[0], "#"):
		case len(fields) == 1:
			links = append(links, pastedLink{url: fields[0]})
		default:
			links = append(links, pastedLink{email: fields[0], url: fields[len(fields)-1]})
		}
	}
	return links
}

// Setup signs everyone in, creates or finds the game and joins the bots.
func (l *League) Setup(ctx context.Context) error {
	cfg := l.cfg
	if cfg.Server.Mode == "hosted" {
		if err := l.enrollHosted(ctx); err != nil {
			return err
		}
	}
	if cfg.Game.Create {
		source, err := l.signIn(ctx, cfg.Host.Email)
		if err != nil {
			return err
		}
		l.host = newGameClient(cfg.Server.URL, source)
		if err := l.host.SetDisplayName(ctx, "Observer"); err != nil {
			return fmt.Errorf("host profile: %w", err)
		}
	}

	for _, player := range cfg.Players {
		source, err := l.signIn(ctx, player.Email)
		if err != nil {
			return err
		}
		persona, err := loadPersona(player.Persona, player.Traits)
		if err != nil {
			return err
		}
		llm, err := newLLM(cfg.Providers[player.Provider], player, l.limiter)
		if err != nil {
			return fmt.Errorf("%s: %w", player.Name, err)
		}
		bot := &Bot{
			cfg:     player,
			persona: persona,
			client:  newGameClient(cfg.Server.URL, source),
			llm:     llm,
			dir:     filepath.Join(cfg.LeagueDir, "bots", slug(player.Name)),
			league:  l,
		}
		if err := bot.client.SetDisplayName(ctx, player.Name); err != nil {
			return fmt.Errorf("%s profile: %w", player.Name, err)
		}
		_ = os.MkdirAll(bot.dir, 0o755)
		l.bots = append(l.bots, bot)
	}
	l.watcher = l.bots[0].client
	if l.host != nil {
		l.watcher = l.host
	}

	if err := l.resolveGame(ctx); err != nil {
		return err
	}
	for _, bot := range l.bots {
		result, err := bot.client.Join(ctx, l.gameID, l.inviteCode)
		if err != nil {
			return fmt.Errorf("%s cannot join the game: %w", bot.cfg.Name, err)
		}
		bot.house = result.Player.ID
	}
	l.saveSession()

	detail, err := l.watcher.Game(ctx, l.gameID)
	if err != nil {
		return err
	}
	l.slots = detail.Players
	names := l.houseNames()
	for _, bot := range l.bots {
		bot.cfg.Name = cmp.Or(names[bot.house], bot.cfg.Name)
	}

	rules, err := l.bots[0].client.Rules(ctx, l.gameID)
	if err != nil {
		return fmt.Errorf("fetch rules: %w", err)
	}
	if cfg.Rules == "compact" {
		rules = compactRules(rules)
	}
	l.rules = rules
	if l.mapData, err = l.bots[0].client.Map(ctx, l.gameID); err != nil {
		return fmt.Errorf("fetch map: %w", err)
	}
	for _, bot := range l.bots {
		bot.system = systemPrompt(bot.house, bot.cfg.Name, bot.persona, l.slots, l.mapData, l.rules)
		bot.load()
	}
	return nil
}

// resolveGame finds the game id and invitation, creating the game if asked.
func (l *League) resolveGame(ctx context.Context) error {
	cfg := l.cfg
	switch {
	case l.gameID != "":
		l.logf("resuming game %s", l.gameID)
	case cfg.Game.InviteURL != "":
		parsed, err := url.Parse(cfg.Game.InviteURL)
		if err != nil {
			return fmt.Errorf("game.invite_url: %w", err)
		}
		l.gameID = parsed.Query().Get("gameId")
		l.inviteCode = parsed.Query().Get("inviteCode")
		if l.gameID == "" || l.inviteCode == "" {
			return errors.New("game.invite_url must carry gameId and inviteCode")
		}
	case cfg.Game.ID != "":
		l.gameID, l.inviteCode = cfg.Game.ID, cfg.Game.InviteCode
	case cfg.Game.Create:
		game, err := l.host.CreateGame(ctx, cfg.Game.Name, cfg.Game.Seed, cfg.Game.Years, len(cfg.Players))
		if err != nil {
			return fmt.Errorf("create game: %w", err)
		}
		l.gameID, l.inviteCode = game.ID, game.InviteCode
		l.logf("created game %q (%s), seed %q", game.Name, game.ID, cfg.Game.Seed)
	}
	if l.inviteCode == "" && l.host != nil {
		var invitation struct {
			InviteCode string `json:"inviteCode"`
		}
		if err := l.host.do(ctx, "GET", "/api/games/"+l.gameID+"/invite", nil, &invitation); err != nil {
			return fmt.Errorf("fetch invitation: %w", err)
		}
		l.inviteCode = invitation.InviteCode
	}
	l.saveSession()
	return nil
}

// Run plays turns until the game ends or max_turns is reached.
func (l *League) Run(ctx context.Context) error {
	cfg := l.cfg
	l.announce()
	played := 0
	reports := map[string]string{}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		detail, err := l.watcher.Game(ctx, l.gameID)
		if err != nil {
			return fmt.Errorf("read game: %w", err)
		}
		if detail.Status == "finished" {
			return l.finish(ctx, detail)
		}
		if cfg.MaxTurns > 0 && played >= cfg.MaxTurns {
			l.logf("max_turns reached (%d): stopping at turn %d (%s)", cfg.MaxTurns, detail.Turn, detail.Season)
			l.printUsage()
			return nil
		}
		l.logf("=== turn %d · %s ===", detail.Turn, detail.Season)
		l.refreshReports(ctx, reports)
		if err := l.playTurn(ctx, detail, reports); err != nil {
			return err
		}
		if err := l.awaitResolution(ctx, detail); err != nil {
			return err
		}
		played++
	}
}

func (l *League) announce() {
	l.logf("League %q: game %s on %s", l.cfg.Name, l.gameID, l.cfg.Server.URL)
	for _, bot := range l.bots {
		l.logf("  %s = %s · %s (%s/%s/%s) · %s via %s", bot.house, bot.cfg.Name, bot.persona.DisplayName,
			bot.persona.Traits.PlayStyle, bot.persona.Traits.Trust, bot.persona.Traits.Tone, bot.cfg.Model, bot.cfg.Provider)
	}
	if l.host != nil {
		l.logf("Watch it: open %s and sign in as %s (emulator sign-in links: %s). Chats are read-only for the observer.",
			l.cfg.Server.URL, l.cfg.Host.Email, strings.TrimRight(l.cfg.Server.AuthEmulator, "/")+"/emulator/v1/projects/"+l.cfg.Server.FirebaseProject+"/oobCodes")
	}
}

// refreshReports loads, for every bot, the report of the last resolved turn.
func (l *League) refreshReports(ctx context.Context, reports map[string]string) {
	for _, bot := range l.bots {
		count, err := bot.client.ReportCount(ctx, l.gameID)
		if err != nil || count == 0 {
			continue
		}
		report, err := bot.client.Report(ctx, l.gameID, count-1)
		if err != nil {
			continue
		}
		reports[bot.house] = renderReport(report)
	}
}

// forEachBot runs fn for every bot concurrently and logs failures.
func (l *League) forEachBot(ctx context.Context, bots []*Bot, fn func(*Bot) error) {
	var wg sync.WaitGroup
	for _, bot := range bots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(bot); err != nil && ctx.Err() == nil {
				l.logf("[%s] %v", bot.cfg.Name, err)
			}
		}()
	}
	wg.Wait()
}

func (l *League) playTurn(ctx context.Context, detail GameDetail, reports map[string]string) error {
	rounds := l.cfg.NegotiationRounds
	for round := 1; round <= rounds; round++ {
		l.logf("negotiation round %d/%d", round, rounds)
		l.forEachBot(ctx, l.bots, func(bot *Bot) error {
			state, err := bot.client.State(ctx, l.gameID)
			if err != nil {
				return fmt.Errorf("state: %w", err)
			}
			report := ""
			if round == 1 {
				report = reports[bot.house]
			}
			return bot.Negotiate(ctx, round, rounds, state, report)
		})
	}

	submitted := map[string]bool{}
	for _, slot := range detail.Players {
		submitted[slot.ID] = slot.Submitted
	}
	var todo []*Bot
	for _, bot := range l.bots {
		if !submitted[bot.house] {
			todo = append(todo, bot)
		}
	}
	l.logf("orders round (%d bot(s))", len(todo))
	label := fmt.Sprintf("T%d %s", detail.Turn, detail.Season)
	l.forEachBot(ctx, todo, func(bot *Bot) error {
		state, err := bot.client.State(ctx, l.gameID)
		if err != nil {
			return fmt.Errorf("state: %w", err)
		}
		report := ""
		if rounds == 0 {
			report = reports[bot.house]
		}
		summary, err := bot.PlayOrders(ctx, state, report)
		if errors.Is(err, errProviderUnavailable) {
			l.logf("[%s] %v", bot.cfg.Name, err)
			l.mu.Lock()
			l.outage = err
			l.mu.Unlock()
			return nil
		}
		if err != nil {
			l.logf("[%s] passes the turn: %v", bot.cfg.Name, err)
			bot.Remember(fmt.Sprintf("%s: you could not submit valid orders (%v); the turn passed with no orders.", label, err))
			if passErr := bot.Pass(ctx); passErr != nil {
				return fmt.Errorf("pass: %w", passErr)
			}
			return nil
		}
		l.logf("[%s] %s", bot.cfg.Name, summary)
		bot.Remember(fmt.Sprintf("%s: you submitted %s", label, summary))
		return nil
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.outage != nil {
		return fmt.Errorf("stopped before the turn resolved, run the same command to resume: %w", l.outage)
	}
	return nil
}

// awaitResolution blocks until the server moves past the given turn.
func (l *League) awaitResolution(ctx context.Context, before GameDetail) error {
	deadline := time.Now().Add(time.Duration(l.cfg.TurnTimeoutMinutes) * time.Minute)
	forced := false
	failures := 0
	for {
		detail, err := l.watcher.Game(ctx, l.gameID)
		if err != nil {
			if failures++; failures > 5 || ctx.Err() != nil {
				return err
			}
			if sleepErr := sleepCtx(ctx, 5*time.Second); sleepErr != nil {
				return sleepErr
			}
			continue
		}
		failures = 0
		if detail.Status == "finished" || detail.Turn != before.Turn || detail.Season != before.Season {
			return nil
		}
		if time.Now().After(deadline) && !forced {
			forced = true
			var waiting []string
			for _, slot := range detail.Players {
				if !slot.Submitted {
					waiting = append(waiting, slot.ID)
				}
			}
			if l.host == nil {
				l.logf("turn %d is still waiting for %s after %d min; the observer must force it", detail.Turn, strings.Join(waiting, ", "), l.cfg.TurnTimeoutMinutes)
			} else if err := l.host.ForceResolve(ctx, l.gameID); err != nil {
				l.logf("force resolve failed: %v", err)
			} else {
				l.logf("turn %d forced (waiting for %s)", detail.Turn, strings.Join(waiting, ", "))
			}
		}
		if err := sleepCtx(ctx, 1500*time.Millisecond); err != nil {
			return err
		}
	}
}

func (l *League) finish(ctx context.Context, detail GameDetail) error {
	l.logf("=== game finished ===")
	if detail.Winner != "" {
		for _, bot := range l.bots {
			if bot.house == detail.Winner {
				l.logf("winner: %s (%s) — %s via %s", bot.house, bot.cfg.Name, bot.cfg.Model, bot.cfg.Provider)
			}
		}
	}
	if state, err := l.bots[0].client.State(ctx, l.gameID); err == nil {
		raw, _ := json.MarshalIndent(map[string]any{"winner": state["winner"], "winners": state["winners"], "scores": state["scores"]}, "", " ")
		_ = os.WriteFile(filepath.Join(l.cfg.LeagueDir, "result.json"), raw, 0o644)
		scores, _ := json.Marshal(state["scores"])
		l.logf("scores: %s", scores)
	}
	l.printUsage()
	return nil
}

func (l *League) printUsage() {
	for _, bot := range l.bots {
		l.logf("usage %s (%s): %d calls, %d prompt tokens, %d output tokens", bot.cfg.Name, bot.cfg.Model, bot.usage.Calls, bot.usage.PromptToks, bot.usage.OutputToks)
	}
}
