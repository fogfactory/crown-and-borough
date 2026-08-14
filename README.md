# Crown & Borough

## Usage

1. Run `make run` to start the backend on port 8080.
2. Run `make web-dev` to start the frontend on port 5173.
3. Open http://localhost:5173 in a browser.

## Multiplayer Mode v1

The development server runs one in-memory game. Players can submit their orders
separately through the HTTP API or the browser; the turn is resolved when every
player has submitted, or when a player forces the resolution. The default game
is created at startup with `SEED` and `PLAYERS` (an integer from 2 to 16,
default 4). `POST /api/game` replaces it, while `POST /api/reset` restores the
startup game.

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
legacy global projection useful for diagnostics. Chain knowledge and combat
audiences are stored with the in-memory game. Authentication, persistence, and
the hosted multi-player routes are not implemented yet.
