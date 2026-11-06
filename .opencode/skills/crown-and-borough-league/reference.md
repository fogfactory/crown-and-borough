# botleague reference

## Commands

```
botleague run <config.yaml> [-turns N]   sign in, create/join, play (resumable)
botleague check <config.yaml>            validate config, personas, API keys
botleague personas                       list the catalogue
```

## Config keys

```yaml
name: league-name            # slug; part of default bot emails
league_dir: ./league/x       # relative to the config file
server:
  url: http://localhost:8080
  mode: emulator | hosted
  auth_emulator: http://127.0.0.1:9099
  firebase_project: demo-crown-and-borough
  api_key: ...               # hosted: Firebase web API key
  email_domain: bots.local   # default address: bl-<league>-<house>@<domain>
game:
  create: true               # emulator only: the host creates the game
  name, seed, years
  invite_url: ...            # join an existing game (hosted)
  id, invite_code: ...       # alternative to invite_url
host:
  email: admin@mail.com      # observer account (emulator creator gate)
rules: full | compact
negotiation_rounds: 0..3
max_repairs: 2
max_turns: 0                 # 0 = to the end
concurrency: 4               # model calls in flight
turn_timeout_minutes: 20     # then the host (emulator) forces the turn
memory_chars: 4000
providers:                   # builtin: openrouter, zen, ollama, mock
  name: { kind: openai|mock, base_url, api_key_env, headers: {} }
players:
  - name, email, persona, traits: {play_style, trust, tone},
    provider, model, temperature, max_tokens
```

Traits: play_style = aggressive, defensive, opportunistic, mercantile, honest,
treacherous, random; trust = high, medium, low, none; tone = curt, formal,
friendly, theatrical, threatening.

## How a bot signs in (no browser)

Both modes use the game's email-link sign-in through the Identity Toolkit REST
API: `accounts:sendOobCode` (EMAIL_SIGNIN) then `accounts:signInWithEmailLink`.
The emulator lists the pending code at
`/emulator/v1/projects/<project>/oobCodes`; on a real server the code is the
`oobCode` parameter of the mailed link. The refresh token is saved in
`session.json` (mode 600) and the ID token is renewed through `securetoken`.

## API used (observer vs player)

Bots: `PUT /api/auth/me`, `POST /api/games/{id}/join`, `GET .../state`,
`.../map`, `.../rules`, `.../reports`, `POST .../orders`, `GET|POST
.../messages`. The observer host: `POST /api/games` (`spectate: true`),
`POST .../resolve` with `force`. See `specs/architecture.md` and `specs/chat.md`.

## Decision format

The model answers one JSON object:
`{"thoughts", "messages":[{"to":"public|P2","body"}], "orders":{"chains":[{"noble","text"}],"winter":[{"lines"}],"special":[{"text"}]}, "memory"}`.
Chains may omit the noble line and winter may be a list of strings; the parser
normalizes both. Messages are cut to 500 characters, at most 4 per call and 1
public, sent one per 1.1 s (the server limit is one per second).
