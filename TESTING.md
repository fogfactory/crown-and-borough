# Testing Online Friends

The browser hotseat mode remains available when the Firebase Web variables are
not configured. It is useful for testing the game board and engine locally. The
online flow is enabled when all four required `VITE_FIREBASE_*` values are
present.

Use `make run-hotseat` or the `make run-dev` alias for a guaranteed legacy
hotseat build. This target clears the Firebase Web variables only for the
frontend build, so it is safe to keep `web/.env.local` for the emulator flow.

## Fully Local Configuration

The recommended manual setup uses the Firebase Auth and Firestore emulators;
it does not require a Firebase project, email provider, or service-account
credentials.

1. Copy `web/.env.example` to `web/.env.local`.
2. Start the complete stack with `make compose-up`.
3. Open `http://localhost:8080` in two separate browsers or private windows.
4. Submit an email address in each browser. The Auth emulator prints each
   sign-in link in `make compose-logs`; links are also available at
   `http://localhost:4000` under Authentication.
5. Open each link in the same browser that requested it. The link redirects to
   the local `/finish` route and completes sign-in against the emulator.

The copied `.env.local` uses `fake-api-key`, `demo-crown-and-borough`, and the
host emulator ports `9099` and `8081`. These values are intentionally local
and must not be used as hosted Firebase configuration.

Stop the stack with `make compose-down`. Emulator data is ephemeral.

For faster Go iteration without rebuilding the container, use
`make run-online`. It starts Auth and Firestore in Docker and runs the local Go
binary against them. The browser still uses the embedded frontend at
`http://localhost:8080`.

For a frontend-only iteration loop, run `make web-dev` in one terminal and a Go
server in another. Vite serves port 5173 and proxies `/api` to port 8080; this
is the only flow that requires a separate frontend process.

## Hosted Configuration

For the hosted Firebase flow, replace the four public Web values in
`web/.env.local`, clear both emulator host variables, enable Firebase
Authentication's email-link provider, and add the local and deployed origins
to the authorized domains. Build the same-origin image with `make image`, or
deploy it through the CI workflow, and provide `FIREBASE_PROJECT_ID` plus ADC
credentials only to the Go runtime. The Firebase Web values are public build
arguments; Firebase Admin credentials never go into `web/`, Docker build
arguments, the frontend bundle, or the final image.

For a local hosted-like process without Docker, run `make run` with the hosted
Firebase project, `PUBLIC_APP_URL`, and ADC configured. Build the frontend first
with the matching public Web values. The server must never receive a player
identity from a query parameter in this mode.

The Firestore emulator rules can be verified with
`FIRESTORE_EMULATOR_HOST=127.0.0.1:8081 make test-firestore` while the local
Firestore emulator is running. The emulator is also used by CI for
`web/src/firestore.rules.test.ts`.

## Single Image Validation

The automated container smoke test covers the basic HTTP contract. The manual
same-origin and restart check is:

1. Copy `web/.env.example` to `web/.env.local` and run `make compose-up`.
2. Open `http://localhost:8080` without starting Vite.
3. Create a game, join it from a second browser, submit a partial turn, and
   verify `/healthz` and `/healthz/ready` both respond successfully.
4. Stop only the Go service with `docker compose stop server`.
5. Start it again with `docker compose start server` and verify the game,
   pending submission, reports, and listeners are restored from Firestore.
6. Run `make image-smoke` to verify a fresh image starts without a persistent
   volume and serves both `/` and a client-side route.
7. Run `make compose-down` after the test. The emulator data is intentionally
   ephemeral; do not use it as a substitute for a hosted Firestore restart
   test.

## Manual Flow

1. Open the application in two separate browsers or private windows.
2. Sign in as Alice and Bob using email links. Open each link in the same
   browser that requested it.
3. Complete a distinct display name in each browser.
4. From Alice's home page, create a game with at least two slots.
5. Copy the invitation URL and open it in Bob's browser.
6. Confirm that Bob must authenticate before the invitation can be consumed.
7. Confirm that the lobby shows the correct names, open slots, and submitted
   status without exposing invitation data to Bob.
8. Create a second game and open it in another tab. Submit orders in one game
   and verify that the other game's map, drafts, listeners, and report do not
   change.
9. Submit Alice's and Bob's orders. Verify the other browser sees the pending
   slot immediately through the public `games/{gameId}` snapshot.
10. Resolve a turn and verify the private state changes immediately through
    `games/{gameId}/views/{uid}`. Verify that each browser sees only its own
    chain and combat projection.
11. Open the report panel, switch between report turns, and verify that reports
    are loaded through the filtered REST endpoint.
12. Test an explicit forced resolution, winter orders, victory, F5, sign-out,
    reconnect, a revoked membership, a `401`, a `403`, and a missing game.
13. Test a narrow mobile viewport and keyboard navigation through tabs, forms,
    lobby slots, and the map.

## Listener Contract

The frontend listens only to these client-readable documents:

- `games/{gameId}` for the public game summary and lobby submission status.
- `games/{gameId}/views/{uid}` for the authenticated player's filtered state.
- `games` with `memberUids array-contains uid` for the authenticated home list.

It never listens to `canonical`, raw submissions, unfiltered reports, privacy
metadata, or invitations. Every listener returns its `onSnapshot` unsubscribe
function from React cleanup. Cleanup also runs when the account, game, or
membership changes, and permission errors stop the affected listeners.

Orders, forced resolution, invitations, profile changes, map loading, rules,
and filtered report history remain authenticated REST calls. ID tokens come
from `getIdToken()` immediately before each call; a `401` signs the browser out
and returns it to the sign-in page without a forced refresh retry.

Offline email or push notifications are intentionally outside O7. A future
server-side trigger, such as a Cloud Function watching a public game update,
can notify players who do not have a browser listener open.
