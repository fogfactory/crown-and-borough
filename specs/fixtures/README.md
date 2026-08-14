# Contract fixtures

These fixtures are complete public JSON documents used by the contract tests.

- `map.json` is a complete `map.json` response. Territory `id` values are the
  canonical three-letter territory identifiers; no territorial `code` field is
  present.
- `state-*.json` are complete `state.json` responses. They cover territory
  identity, an army without a chain, and an army whose chain exists but is
  hidden.
- `report-combat-*.json` are complete report responses. They cover the exact
  combat view and the general combat view.

The private projection contract deliberately distinguishes these values:

```json
"chain": null
```

means that no chain is active, while:

```json
"chain": { "visibility": "hidden" }
```

means that a chain exists but its details are not revealed to this player.

An exact combat contains the forces and army references relevant to the
recipient. A general combat contains only the public summary and omits those
details, so the two views cannot be confused by a client.
