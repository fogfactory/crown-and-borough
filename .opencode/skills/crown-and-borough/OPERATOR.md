# Crown & Borough bot operator guide

This skill is one player per Oh My Pi session. Run three or four sessions in
parallel, each with a different persona and a different Firebase email. The
human may either join as a player or create an **observer host** game.

## Observer Host Flow

1. Start the stack:

   ```bash
   docker compose up -d --build
   ```

2. Sign in to the web app as the host and create a game.
3. Tick **Observe without playing**. The requested number remains the number
   of player slots; the host is extra and does not consume one.
4. Open the created game once and copy its invitation URL.
5. Give that URL to every bot session or pass it to `start-bots.sh`.
6. Set `CB_HUMAN_OBSERVER=1` for all bot sessions. Bots will know that the
   human is observing, while the server gives the host full state and reports.

The observer can inspect the live `games/{id}/observer/{uid}` projection and
force-resolve a stalled turn. It cannot submit orders and never appears in the
player slot list.

## Player Host Flow

Create the game without the checkbox. The host occupies the first slot. Set
`CB_HUMAN_SLOT` to the player id assigned to that slot when starting the bots.
Bots should treat that player as a real negotiator, not as an observer.

## Persona Assignment

Use a different base persona for each parallel session. Trait overrides are
supported by `persona-init.sh`:

```bash
CB_INSTANCE_ID=cb-red-01 ./scripts/persona-init.sh \
  --id diplomat --trait play_style=opportunistic --trait trust=low
```

Recommended three-player test:

```text
cb-red-01   conqueror
cb-blue-02  diplomat + low trust
cb-green-03 turtle
```

Recommended four-player stress test:

```text
cb-red-01   conqueror
cb-blue-02  intriguer
cb-green-03 merchant
cb-gold-04  honorable
```

## Token Bootstrap

The API tools require a Firebase ID token. The browser flow remains the source
of authentication and invitation acceptance. When the Oh My Pi browser bridge
can expose the token, cache it in that same session:

```bash
./scripts/auth-cache.sh save \
  --token "$CB_AUTH_TOKEN" \
  --email red.wolf@mail.com \
  --uid "$CB_UID" \
  --api-url http://localhost:8080
```

The token is stored under the instance/game directory with mode `600`. It is
never written to the shared chat log. If the bridge cannot expose a token, use
the browser for the individual API action or configure the harness's
authenticated request export before switching to API-first play.

## Launcher

`start-bots.sh` starts one harness process per bot in parallel. Since Oh My Pi
installations expose different command-line flags, the launcher accepts a
command template through `CB_HARNESS_COMMAND`. The template may contain
`{session}`, `{persona}`, `{game}`, `{invite}`, and `{prompt}`.

Without a template it prints the launch commands without running them:

```bash
./scripts/start-bots.sh \
  --game-id GAME_ID \
  --invite-url 'http://localhost:8080/join?gameId=GAME_ID&inviteCode=CODE' \
  --personas conqueror,diplomat,turtle \
  --spectate
```

For a harness whose command accepts a prompt argument:

```bash
CB_HARNESS_COMMAND='omp --session {session} --prompt {prompt}' \
  ./scripts/start-bots.sh \
    --game-id GAME_ID \
    --invite-url 'http://localhost:8080/join?gameId=GAME_ID&inviteCode=CODE' \
    --personas conqueror,diplomat,turtle \
    --spectate
```

Each process receives `OMP_SESSION_ID`, `CB_INSTANCE_ID`, `CB_PERSONA`,
`CB_GAME_ID`, `CB_INVITE_URL`, and `CB_HUMAN_OBSERVER`. Adapt only the command
template when the local Oh My Pi binary uses different flags.

## Shared Files

```text
~/.crown-borough/run/<game-id>/chat.log
~/.crown-borough/run/<game-id>/<instance-id>/persona.json
~/.crown-borough/run/<game-id>/<instance-id>/auth.json
~/.crown-borough/run/<game-id>/<instance-id>/game.json
~/.crown-borough/run/<game-id>/<instance-id>/state-cache.json
~/.crown-borough/run/<game-id>/<instance-id>/memory.md
```

Only `chat-send.sh` writes the shared chat log. The old `moves.txt` files are
deprecated and are not part of the new negotiation protocol.
