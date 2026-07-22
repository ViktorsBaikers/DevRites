# Automation hooks

Automate the checks humans forget, at the cheapest stage that can catch the problem.
The guiding constraint: **the hook nobody runs is worse than a slower hook that still
catches things, so keep local hooks fast and scoped.**

## Stage the work by cost (the 10-second rule)
- **pre-commit** (must finish in well under ~10s): format, lint, and secret-scan the
  **staged files only**, plus ultra-fast checks. If it's slow, developers bypass it.
- **commit-msg**: enforce the commit convention (e.g. Conventional Commits). DevRites
  ships a `commit-msg` hook for its own repo as the reference example.
- **pre-push** (seconds to a couple of minutes): broader/affected tests.
- **CI** (no time pressure): the full test suite, build, integration, and deeper
  security scans. CI is the source of truth for "green", not local hooks.

## What to run where
| Stage | Run | Don't run |
|---|---|---|
| pre-commit | formatter, fast linter, secret scan, changed-files checks | the full test suite |
| commit-msg | commit-message lint | anything slow |
| pre-push | affected unit/integration tests | end-to-end matrices |
| CI | everything (suite, build, e2e, security) |: |

## Keep hooks fast
- Operate on **staged/changed files**, not the whole tree.
- Lean on tool caches (most modern linters/formatters cache and re-run only what changed).
- Prefer fast tools; a slow check belongs in CI, not the commit path.

## Secret scanning
Scan for credentials/keys/tokens before they enter history: catching a secret
pre-commit is cheap; rotating a leaked one is not.

## Adoption & escape hatches
- Introduce hooks gradually (formatting → linting → security → tests) so the team
  adopts them instead of disabling them.
- Bypassing a hook (`--no-verify`) is for genuine emergencies, not routine. CI must
  re-check what a bypass skipped, so a skipped local check can't reach the trunk.
- Hooks are checked into the repo and shared, so the whole team gets the same gates.
