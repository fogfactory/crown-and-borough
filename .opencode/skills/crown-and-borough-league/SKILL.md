---
name: crown-and-borough-league
description: Use when the user wants to watch or run a Crown & Borough game played only by AI agents (language models chosen per house, from OpenRouter, OpenCode Zen, Ollama or any OpenAI-compatible endpoint), locally on the Firebase emulators or on a deployed server. Covers configuring a league (seed, name, personas, models), running and resuming it with `botleague`, bot memory, diplomacy through the in-game chat, hosted-server enrollment with email links, and reading the results.
---

# Crown & Borough bot league

A league is a Crown & Borough online game where **every house is played by a
language model** and the human only observes. The `botleague` program
(`cmd/botleague`) does the deterministic work; the models only decide.

```
league.yaml ──► botleague run ──► Firebase Auth (emulator or real, REST, no browser)
                    │                 sign in, display name, join the invitation
                    ├─► game API ──►  create the game (observer host) / join it
                    └─► LLM APIs ──►  one model per house: negotiate, then write orders
```

Each turn: `negotiation_rounds` message-only rounds (every bot reads the chat,
answers, proposes, via `POST /messages`), then one order round (the bot sees
the replies, then submits orders; the server's errors are fed back so it can
repair them, up to `max_repairs`; if it still fails it passes the turn). The
server resolves the turn once every house has submitted.

You (the human) are the **observer host**: you see the full state and every
private conversation (read-only) in the web app. You never play.

## Quick start (local demo)

1. Start the online stack (Docker emulators + server; chat does not exist in
   hotseat or `ONLINE_DEV_MODE`):

   ```bash
   scripts/league-server.sh start        # PORT=8080 by default; `stop` to stop
   ```

2. Copy `league.example.yaml` from this skill, pick the models (see below),
   export the API keys it needs, and validate:

   ```bash
   export OPENROUTER_API_KEY=...         # and/or OPENCODE_API_KEY=...
   go run ./cmd/botleague check my-league.yaml
   ```

   `check` prints every house with its resolved persona, model and API-key
   status. A provider without key is an error before any game exists.

3. Play (resumable: run the same command again after Ctrl-C or a crash):

   ```bash
   go run ./cmd/botleague run my-league.yaml            # to the end of the game
   go run ./cmd/botleague run my-league.yaml -turns 4   # only 4 more turns
   ```

4. Watch: open `http://localhost:8080`, request a sign-in link for the host
   email (`admin@mail.com` by default), open it from the list at
   `http://127.0.0.1:9099/emulator/v1/projects/demo-crown-and-borough/oobCodes`,
   and open the game. The Messages tab shows the public table and all private
   conversations.

Try the plumbing first with `provider: mock` for every house (no key, no cost):
bots hold every noble and greet the table.

## Choosing models

- OpenRouter (`provider: openrouter`, `OPENROUTER_API_KEY`): free models have
  ids ending in `:free`. List them:

  ```bash
  curl -s https://openrouter.ai/api/v1/models | jq -r '.data[] | select(.pricing.prompt=="0") | "\(.id)\t\(.context_length)"'
  ```

- OpenCode Zen (`provider: zen`, `OPENCODE_API_KEY`, base
  `https://opencode.ai/zen/v1`): use the model ids listed in the Zen docs.
- Local (`provider: ollama`, no key, `http://localhost:11434/v1`) or any other
  OpenAI-compatible server: declare it under `providers:` with `base_url`
  (and `api_key_env` if it needs a key).
- The full player rules go in every system prompt (about 20k tokens): pick
  models with a 64k+ context, or set `rules: compact`. Free tiers rate-limit:
  keep `concurrency` low (2-3); the client retries 429/5xx with backoff.
- The models must answer one JSON object; the parser tolerates fences, think
  blocks and prose around it, and asks again once if it cannot read the answer.
  Weak models may still write invalid orders: that is expected, the transcript
  shows how often (see Reading the results).
- Mix models on purpose: one model per house is the point of a league.

## Configuration

See `league.example.yaml` (every key commented) and `reference.md`. The ones
people change:

| Key | Meaning |
|---|---|
| `game.seed`, `game.name`, `game.years` | Same seed = same map and same starting nobles. `years: 0` is the server default. |
| `players[].persona` + `traits` | Catalogue id (`go run ./cmd/botleague personas`); `traits` overrides `play_style`, `trust`, `tone`. |
| `players[].provider` / `model` | The model behind the house. |
| `negotiation_rounds` | 0 = messages and orders in one call (cheapest); 1-3 = real back-and-forth. |
| `rules` | `full` or `compact`. |
| `max_turns` / `-turns N` | Stop early. |
| `memory_chars` | Size of each bot's private memory (default 4000). |

Personas are the files in `cmd/botleague/personas/*.yaml` (embedded in the
binary); add one by dropping a YAML file there. A persona is three independent
traits plus decision rules, not a chat costume. Never swap a persona mid-game.

## Memory

Each house has `league_dir/bots/<house>/`:

- `memory.md`: the bot's own private memory, rewritten by the model each call.
  Its first line is an identity anchor generated from the persona and always
  restored. It holds pacts (scope, expiry, break condition), notes per rival,
  suspicions, lessons and the next-turn plan.
- `history.md`: the last 8 outcomes, written by the league (orders submitted,
  turns passed after a failure).
- `seen.json`: which chat messages were already read (drives `[NEW]` flags).
- `calls/NNN-tN-season-phase.json`: every model call (prompt + raw answer).

The prompt of a call is: system (persona, protocol, format, static map, rules)
+ memory + history + last turn's report + compact current state + recent chat.
Nothing else is replayed, so cost per call stays flat as the game goes on.

## Deployed server

The human host creates the game in the web app with **Observe without
playing** ticked and copies the invitation URL. Then use
`hosted.example.yaml`: set `server.mode: hosted`, the Firebase web API key and
`game.invite_url`, and give every bot an inbox (`you+rouge@gmail.com` works).

First `run`: the league asks Firebase to mail a sign-in link to each bot and
stops. **Ask the user to forward the links** (or paste them into
`<league_dir>/links.txt`, one per line, optionally `email link`), then run the
same command again. Tokens are refreshed automatically afterwards. Force
resolution of a stalled turn from the app as the observer host.

## Reading the results

- `league_dir/league.log`: turns, submitted orders, failures, token usage.
- `league_dir/result.json`: winner and scores at the end.
- `bots/<house>/calls/*.json`: why a bot did something (`thoughts` field of the
  answer) and which orders the server rejected (`-fixN` calls).
- Count `passes the turn` lines per house to rate how well each model handles
  the order syntax.

## When something fails

| Symptom | Action |
|---|---|
| `holds no API key` | Export the variable named in `providers.<name>.api_key_env`. |
| `creator_not_allowed` | Server is not started by `scripts/league-server.sh`, or `host.email` is not in `ALLOWED_CREATOR_EMAILS`. |
| `chat_disabled` (404) | The server runs hotseat or `ONLINE_DEV_MODE`: use `scripts/league-server.sh`. |
| `connection refused :9099` | Emulators are down: `scripts/league-server.sh start`. |
| `game is full` / `invite` errors | The game already has other players: change `game.seed`/`name` and `league_dir`, or delete `session.json`. |
| A house passes every turn | Its model cannot follow the order syntax; read its `-fix` calls and swap the model. |
| Turn never resolves | A house crashed: re-run the command (it only plays houses that have not submitted). |

## Rules of the road

- Never hardcode API keys in the YAML: only environment variable names.
- Do not edit a running league's `session.json` or bot memories by hand
  unless you mean to change the game's course.
- Rule changes live in `specs/gdd.md` and `assets/regles-joueurs.md`; the bots
  read the live rules from the server, so nothing here needs updating when the
  rules change. If order syntax changes, update the "Orders syntax" section of
  `systemPrompt` in `cmd/botleague/prompt.go`.
