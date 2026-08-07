# Crown & Borough

## Usage

1. Run `make run` to start the backend on port 8080.
2. Run `make web-dev` to start the frontend on port 5173.
3. Open http://localhost:5173 in a browser.

## Hotseat mode

The development server runs one in-memory hotseat game. All players use the
same machine and see the complete, current map (T0 vision); there is no
authentication or persistence in this temporary mode. The default game is
created at startup with `SEED` and `PLAYERS` (an integer from 2 to 5, default
4). `POST /api/game` replaces it, while `POST /api/reset` restores the startup
game. In the browser, choose a player, enter one complete chain per available
noble (header plus order lines), or winter investment lines during winter, then
click **Résoudre le tour**. The frontend submits all players' orders together,
advances spring → summer → autumn → winter automatically, and displays the
typed turn report. A syntactically valid chain that cannot be received by an
army is lost and reported without blocking the other players' turn. This mode
will be replaced by the online, private-vision flow when the P2/P3 server work
is implemented.
