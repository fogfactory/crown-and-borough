// Command botleague plays a Crown & Borough game between language-model
// houses. The human only observes: the league signs the bots in, creates (or
// joins) the game, then loops over the turns, asking every model to
// negotiate through the in-game chat and to write its orders.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const usage = `botleague plays Crown & Borough between language-model houses.

usage:
  botleague run <config.yaml>      sign the bots in, create/join the game, play it (resumable)
  botleague check <config.yaml>    validate the config, providers and API keys without playing
  botleague personas               list the persona catalogue

flags of run:
  -turns N    stop after N more turns (overrides max_turns)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = runCommand(os.Args[2:])
	case "check":
		err = checkCommand(os.Args[2:])
	case "personas":
		err = personasCommand()
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "botleague:", err)
		os.Exit(1)
	}
}

func runCommand(args []string) error {
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	turns := flags.Int("turns", 0, "stop after this many turns")
	if err := flags.Parse(reorderFlags(args)); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("run needs one config file")
	}
	cfg, err := loadConfig(flags.Arg(0))
	if err != nil {
		return err
	}
	if *turns > 0 {
		cfg.MaxTurns = *turns
	}
	league, err := newLeague(cfg)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := league.Setup(ctx); err != nil {
		return err
	}
	err = league.Run(ctx)
	if errors.Is(err, context.Canceled) {
		league.logf("interrupted: run the same command to resume")
		return nil
	}
	return err
}

// reorderFlags lets flags follow the config path.
func reorderFlags(args []string) []string {
	var flagsPart, rest []string
	for index := 0; index < len(args); index++ {
		if strings.HasPrefix(args[index], "-") {
			flagsPart = append(flagsPart, args[index])
			if !strings.Contains(args[index], "=") && index+1 < len(args) {
				index++
				flagsPart = append(flagsPart, args[index])
			}
		} else {
			rest = append(rest, args[index])
		}
	}
	return append(flagsPart, rest...)
}

func checkCommand(args []string) error {
	if len(args) != 1 {
		return errors.New("check needs one config file")
	}
	cfg, err := loadConfig(args[0])
	if err != nil {
		return err
	}
	limiter := make(chan struct{}, 1)
	fmt.Printf("league %q on %s (%s), league dir %s\n", cfg.Name, cfg.Server.URL, cfg.Server.Mode, cfg.LeagueDir)
	problems := 0
	for _, player := range cfg.Players {
		persona, _ := loadPersona(player.Persona, player.Traits)
		status := "ok"
		if _, err := newLLM(cfg.Providers[player.Provider], player, limiter); err != nil {
			status = err.Error()
			problems++
		}
		fmt.Printf("  %-18s %-26s %s/%s/%s  %s via %s  [%s]\n", player.Name, player.Email,
			persona.Traits.PlayStyle, persona.Traits.Trust, persona.Traits.Tone, player.Model, player.Provider, status)
	}
	if problems > 0 {
		return fmt.Errorf("%d player(s) cannot run", problems)
	}
	return nil
}

func personasCommand() error {
	for _, id := range personaIDs() {
		persona, err := loadPersona(id, nil)
		if err != nil {
			return err
		}
		fmt.Printf("%-11s %s/%s/%s  %s\n", id, persona.Traits.PlayStyle, persona.Traits.Trust, persona.Traits.Tone, persona.DisplayName)
	}
	return nil
}
