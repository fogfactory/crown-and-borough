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
derives the player from the verified UID. Profiles, memberships, and hashed
invitation codes are currently in memory; Firestore persistence is the next
online milestone.

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
revision per game. The frontend Firebase sign-in flow and Firestore listeners
are intentionally left to the next online milestone.
