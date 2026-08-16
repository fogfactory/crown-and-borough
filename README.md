# Crown & Borough

## Usage

1. Run `make run-dev` to start the local development backend on port 8080, or
   configure Firebase and run `make run` for the authenticated API.
2. Run `make web-dev` to start the frontend on port 5173.
3. Open http://localhost:5173 in a browser.

## Multiplayer Mode v1

The development server exposes the legacy hotseat session and, when started
with `ONLINE_DEV_MODE=true`, a multi-game in-memory API. The game API is rooted
at `/api/games`: create independent games with `POST /api/games`, inspect them
with `GET /api/games/{id}`, and submit one player's orders through
`POST /api/games/{id}/orders`. A turn resolves when all living players have
submitted, or through the explicit `POST /api/games/{id}/resolve` endpoint. Use
`?player=P1` only with this explicit local development mode.

The authenticated API validates Firebase ID tokens with the Firebase Admin SDK.
Set `FIREBASE_PROJECT_ID`, `PUBLIC_APP_URL` (and provide ADC credentials) before
`make run`.
`GET /api/auth/me` creates the UID profile when needed and `PUT /api/auth/me`
updates its display name. Authenticated game creation requires a completed
profile and returns a six-character invitation code and URL. A member joins
through `POST /api/games/{id}/join` with `{ "inviteCode": "..." }`; the server
derives the player from the verified UID. In hosted mode, profiles,
memberships, canonical state, pending submissions, reports, privacy metadata,
and filtered projections are persisted in Firestore Native mode. No Firebase
token or clear invitation code is stored by the backend.

## Firestore Persistence

Production mode requires Application Default Credentials and
`FIREBASE_PROJECT_ID` (or `GOOGLE_CLOUD_PROJECT`). `FIRESTORE_DATABASE_ID` is
optional and defaults to `(default)`. Local development keeps the in-memory
store unless `FIRESTORE_EMULATOR_HOST` is set; with the emulator configured,
`ONLINE_DEV_MODE=true` uses Firestore as well.

The versioned schema is defined in `internal/store/firestore` and uses
`players/{uid}`, `games/{gameId}`, `canonical/current`, turn submissions,
unfiltered reports, per-player views, filtered reports, and hashed invitations.
`firestore.rules` permits a member to read only the game summary and their own
profile, state view, and filtered reports. Game writes, canonical state, raw
orders, raw reports, privacy metadata, and invitations remain backend-only.

Every submission checks the expected revision in a Firestore transaction.
Resolution first claims the current revision with an `operationId`,
`baseRevision`, and a 30-second recoverable lease. The deterministic engine runs
outside the transaction; the conditional commit then writes the next canonical
state, bounded report history, all filtered projections, and removes the
current submissions. A crashed claim can be reclaimed after its lease expires;
this is not a player deadline.

`FirestoreStore.Metrics()` exposes logical reads, writes, transactions, and
projection writes for emulator and quota benchmarks. Firestore transaction
retries can make billed SDK operations higher than these logical counters.

Use `FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 make test-firestore` when the
Firestore emulator is running. The integration suite models a restart with a
second store instance and the rules suite checks member, non-member, private
projection, and backend-only access.

## Local Docker Compose Stack

The repository includes a fully local online stack with Auth and Firestore
emulators. It starts the emulators and the Go server; the optional `frontend`
profile adds a statically built Nginx frontend with `/api` proxied to the
server.

```bash
cp web/.env.example web/.env.local
make compose-up
make compose-up-frontend
make compose-logs
make compose-down
```

The services use these host ports:

| Service | URL |
| --- | --- |
| Go server | `http://localhost:8080` |
| Auth emulator | `http://localhost:9099` |
| Emulator UI | `http://localhost:4000` |
| Firestore emulator | `127.0.0.1:8081` |
| Frontend profile | `http://localhost:5173` |

If one of the default ports is already in use, override it when starting the
stack, for example `SERVER_PORT=18080 FIRESTORE_PORT=18081 make compose-up`.
The browser emulator endpoints in `web/.env.local` must use the corresponding
host ports.

Compose runs with `ONLINE_DEV_MODE=false` and validates Auth emulator tokens,
so two browsers receive distinct Firebase UIDs and exercise the real online
membership flow. `make run-dev` remains the legacy development path with the
explicit local player resolver. Emulator data is intentionally ephemeral: stop
the stack and start it again to get a clean local database. The Firestore
rules remain mounted from `firestore.rules` for emulator validation.

The frontend profile reads `web/.env.local` during the image build. The Auth
emulator prints email links in `docker compose logs -f auth`; the same links
are available in the Emulator UI under Authentication. Open each link in the
same browser that requested it.

The legacy hotseat game is created at startup with `SEED` and `PLAYERS` (an
integer from 2 to 16, default 4). `POST /api/game` replaces it, while
`POST /api/reset` restores the startup game.

In the browser, choose a player, enter one complete chain per available noble
(header plus order lines), or winter investment lines during winter, then click
**Submit**. The button becomes **Edit** after submission; a later click replaces
that player's pending orders. The noble header is added automatically before the
request is sent. The game advances spring → summer → autumn → winter and displays
the typed turn report. A syntactically valid chain that cannot be received by an
army is reported without blocking the other players' turn.

The interface defaults to English and includes an **EN / FR** language switcher.
The selected language is kept in the browser and is used for the interface,
player-facing validation errors, reports, and the rules document. Order symbols
and memorable command terms remain identical in both languages.

The development hotseat server accepts `GET /api/state?player=P1` to return the
server-filtered private view for the selected player; omitting `player` keeps the
legacy global projection useful for diagnostics. The multi-game store keeps
chain knowledge, combat audiences, pending submissions, reports, and a monotone
revision per game. When the public Firebase Web variables are configured, the
frontend switches to the authenticated friends flow: email-link sign-in,
profile setup, invitations, multi-game navigation, and Firestore listeners on
the public summary and the current player's private view. See
[`TESTING.md`](TESTING.md) for the manual flow and listener contract.
