# Instructions for AI agents

## Branch workflow

- `main` is the protected release branch. Only reviewed bugfixes, UX changes,
  promotion pull requests, and release-please pull requests target it.
- `develop` is the protected integration branch. Feature and maintenance pull
  requests target it; it is never deployed directly.
- Use `fix/*` or `bugfix/*` for bugfixes targeting `main`, `ux/*` for UX changes
  targeting `main`, and `feat/*` or `feature/*` for features targeting `develop`.
- Branches created by Claude Code (`claude/*`) are accepted by the pull request
  policy: the Conventional Commit title decides the target (`feat` targets
  `develop`, `fix` targets `main`, anything else targets `develop`).
- Never push directly to `main` or `develop`. Use a pull request and preserve
  the repository's required review and CI checks.
- Pull request titles must use `type(scope): description`. Use `fix` for
  release-worthy patches, `feat` for minor releases, and `!` or a `BREAKING
  CHANGE` footer for major releases. Maintenance-only commits do not create a
  release unless their type is included in the release changelog.
- Normal pull requests must be squash-merged with the pull request title kept
  as the commit subject; promotions from `develop` to `main` and the
  synchronization exception use merge commits so release-please can see the
  individual Conventional Commits.
- Apply exactly one `release:patch`, `release:minor`, or `release:major` label
  to release-bearing work. The pull request policy checks that the label agrees
  with the Conventional Commit title; release-please performs the actual bump.
- Do not create production tags manually. release-please creates the SemVer
  tag and GitHub release after its release pull request is merged.
- `main` is synchronized back into `develop` by automation. Do not create a
  second synchronization pull request; resolve the generated one (branch `chore/sync-main-into-develop`) if it has a
  conflict.

## Commits and pushes

When the user asks to commit and push:

- Commit and push the current work.
- Add to the commit log a short description of what was done.

## Model routing (Claude Code)

The goal is cost efficiency: run each task on the cheapest model that can do it
well. Claude Code cannot downgrade the top-level model mid-session, so routing
happens by **delegating to the model-pinned subagents** in `.claude/agents/`.
The top-level model is set by `opusplan` (Opus in plan mode, Sonnet otherwise);
it reads the request and dispatches to the right agent below.

When in doubt, **route down**: pick the cheaper tier and escalate only if the
subagent reports it is blocked or returns an insufficient result.

Pick by task, cheapest first:

| Task | Model | Subagent |
|------|-------|----------|
| Run tests, builds, linters, make targets, scripts (no source edits) | Haiku | `runner` |
| Simple, localized fix — one file or one layer, change and location already clear | Haiku | `fixer` |
| Real implementation crossing several layers (engine/api/db/front) or non-trivial logic | Sonnet | `implementer` |
| Focused analysis or plan on a single axis — a precise technical point **OR** a game-design point | Sonnet | `analyst` |
| Broad or cross-cutting plan spanning technical **AND** game-design concerns | Opus | `planner` |

Two shortcuts override the table:

- A truly trivial, obvious edit the orchestrator can make in one shot: do it
  directly, no subagent — spawning would cost more than the edit.
- If a `fixer` or `analyst` task turns out bigger than it looked, stop and
  re-dispatch to `implementer` or `planner` rather than pushing the cheaper
  model past its depth.
