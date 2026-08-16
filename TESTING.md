# Testing Online Friends

The browser hotseat mode remains available when the Firebase Web variables are
not configured. It is useful for testing the game board and engine locally. The
online flow is enabled when all four required `VITE_FIREBASE_*` values are
present.

## Configuration

1. Copy `web/.env.example` to `web/.env.local`.
2. Fill in the public Firebase Web values for the same project used by the Go
   API. Do not add Admin credentials or service-account JSON to `web/`.
3. Enable Firebase Authentication's email-link provider and add the local and
   deployed origins to the authorized domains.
4. Start the frontend with `make web-dev` and open `http://localhost:5173`.
5. Start the authenticated Go API with the Firebase project and ADC credentials
   configured, or use the deployed Cloud Run service.

The Firestore emulator rules can be verified with `make test-firestore`. The
Firestore emulator is used by CI for `web/src/firestore.rules.test.ts`; it is
not a replacement for Firebase Authentication when manually exercising the
hosted browser flow.

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
