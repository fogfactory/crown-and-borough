## Change Type

- [ ] Bugfix: use a `fix/*` branch and target `main`; apply `release:patch`.
- [ ] UX improvement without a contract change: use a `ux/*` branch and target `main`; apply `release:patch` or `release:minor`.
- [ ] Feature or contract change: use a `feat/*` branch and target `develop`; apply `release:minor` or `release:major`.
- [ ] Maintenance or documentation: use `chore/*`, `docs/*`, `refactor/*`, `test/*`, `ci/*`, or `build/*` and target `develop`.

## Public Contract

- [ ] This change does not modify a public API, schema, or persisted contract.
- [ ] This change is breaking and the title contains `!` or the body contains a `BREAKING CHANGE` footer.

## Checklist

- [ ] The pull request target matches the branch policy above.
- [ ] The title follows `type(scope): description` and is written in English.
- [ ] The appropriate `release:patch`, `release:minor`, or `release:major` label is applied when this change belongs in a release.
- [ ] Tests and documentation have been updated where necessary.
