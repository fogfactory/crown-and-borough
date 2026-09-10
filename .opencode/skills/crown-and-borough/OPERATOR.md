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
`{session}`, `{persona}`, `{game}`, `{invite}`, `{email}`, `{display_name}`,
`{private_root}`, and `{prompt}`.

The launcher creates a deterministic email and a private cache/browser profile
for every bot. Only the chat log remains shared. This is important: do not
manually set every bot to the same `CB_PRIVATE_ROOT`, and do not run parallel
bots with a shared browser profile.

Prepare enrollment first, so the human can register every address before the
bot sessions start requesting links:

```bash
./scripts/start-bots.sh \
  --personas conqueror,diplomat,turtle \
  --enroll-only
```

Each bot then uses `extract-auth-link.sh --email EMAIL`; it never consumes the
first unrelated link from the shared Auth emulator log.

Run the offline boundary check after changing the launcher or cache layout:

```bash
./scripts/self-test.sh
```

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
`CB_GAME_ID`, `CB_INVITE_URL`, `CB_EMAIL`, `CB_DISPLAY_NAME`,
`CB_HUMAN_OBSERVER`, `CB_SHARED_ROOT`, `CB_PRIVATE_ROOT`,
`CB_PRIVATE_ROOT_ISOLATED=1`, and `CB_BROWSER_PROFILE_DIR`. Adapt only the
command template when the local Oh My Pi binary uses different flags.

## Shared Files

```text
~/.crown-borough/run/<game-id>/chat.log                 # shared
~/.crown-borough/run/private/<instance-id>/...          # private bot root
~/.crown-borough/run/private/<instance-id>/<game-id>/
  persona.json, auth.json, state-cache.json, memory.md
```

Only `chat-send.sh` writes the shared chat log. The old `moves.txt` files are
deprecated and are not part of the new negotiation protocol.
