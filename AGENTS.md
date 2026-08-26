# Instructions for AI agents

## Branch workflow

- `main` is the protected release branch. Only reviewed bugfixes, UX changes,
  promotion pull requests, and release-please pull requests target it.
- `develop` is the protected integration branch. Feature and maintenance pull
  requests target it; it is never deployed directly.
- Use `fix/*` or `bugfix/*` for bugfixes targeting `main`, `ux/*` for UX changes
  targeting `main`, and `feat/*` or `feature/*` for features targeting `develop`.
- Never push directly to `main` or `develop`. Use a pull request and preserve
  the repository's required review and CI checks.
- Pull request titles must use `type(scope): description`. Use `fix` for
  release-worthy patches, `feat` for minor releases, and `!` or a `BREAKING
  CHANGE` footer for major releases. Maintenance-only commits do not create a
  release unless their type is included in the release changelog.
- Normal pull requests must be squash-merged with the pull request title kept
  as the commit subject; the synchronization exception uses a merge commit on
  `develop`.
- Apply exactly one `release:patch`, `release:minor`, or `release:major` label
  to release-bearing work. The pull request policy checks that the label agrees
  with the Conventional Commit title; release-please performs the actual bump.
- Do not create production tags manually. release-please creates the SemVer
  tag and GitHub release after its release pull request is merged.
- `main` is synchronized back into `develop` by automation. Do not create a
  second synchronization pull request; resolve the generated one if it has a
  conflict.

## Commits and pushes

When the user asks to commit and push:

- Commit and push the current work.
- Add to the commit log a short description of what was done.
