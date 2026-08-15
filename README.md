# Crown & Borough

## Usage

1. Run `make run` to start the backend on port 8080.
2. Run `make web-dev` to start the frontend on port 5173.
3. Open http://localhost:5173 in a browser.

## Multiplayer Mode v1

The development server exposes the legacy hotseat session and, when started
with `ONLINE_DEV_MODE=true`, a multi-game in-memory API. The game API is rooted
at `/api/games`: create independent games with `POST /api/games`, inspect them
with `GET /api/games/{id}`, and submit one player's orders through
`POST /api/games/{id}/orders`. A turn resolves when all living players have
submitted, or through the explicit `POST /api/games/{id}/resolve` endpoint. Use
`?player=P1` only with this explicit local development mode; hosted
authentication will replace it with the verified actor in a later online
milestone.

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
revision per game. Authentication and persistence are not implemented yet.
