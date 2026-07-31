# Automation hooks

Automate easy-to-forget checks at the earliest affordable stage. Keep local hooks fast
and scoped so developers continue to run them.

## Stage the work by cost (the 10-second rule)
- **pre-commit** (must finish in well under ~10s): format, lint, and secret-scan the
  exact **staged index blobs only** via `devrites-engine secret-scan --staged`.
- **commit-msg**: enforce the commit convention (e.g. Conventional Commits). DevRites
  ships a `commit-msg` hook for its own repo as the reference example.
- **pre-push** (seconds to a couple of minutes): broader/affected tests.
- **CI** (no time pressure): the full test suite, build, integration, and deeper
  security scans. CI is the source of truth for "green", not local hooks.

## Secret scanning
Scan for credentials/keys/tokens before they enter history: catching a secret
pre-commit is cheap; rotation is not. Findings disclose metadata only; `$rite-ship`
owns fail-closed PR-body transport.

## Adoption & escape hatches
- Introduce hooks gradually (formatting → linting → security → tests) so the team
  adopts them instead of disabling them.
- Bypassing a hook (`--no-verify`) is for genuine emergencies, not routine. CI must
  re-check what a bypass skipped, so a skipped local check can't reach the trunk.
- Hooks are checked into the repo and shared, so the whole team gets the same gates.
