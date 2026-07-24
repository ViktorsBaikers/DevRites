# ADR-0006: Clock seam + Go static-analysis CI gates

- **Status:** Accepted
- **Date:** 2026-07-08

## Context

Two gaps surfaced during a control-plane reliability audit:

1. **Unwired static analysis.** `engine/Makefile` defined `staticcheck` and
   `govulncheck` targets, but CI ran only `gofmt` + `go vet`. No `-race`, no
   custom analysis on the trust-root binary.
2. **A wall-clock leak.** `resolve next-qid` derived today's date from a raw
   `time.Now()` with no seam, so its golden snapshot was pinned to the date it
   was recorded and failed **every other day** — a clock-seam check would have
   caught the latent failure at author time.

## Decision

- **Clock seam.** Wall-clock reads in the resolve command flow through one
  overridable point, `clockNow()`, honoring `DEVRITES_NOW` (RFC-3339 or bare
  `YYYY-MM-DD`). Date-derived output is deterministic under test; goldens pin the
  clock instead of tracking the real date.
- **Engine CI gates.** The `engine` CI job installs pinned `staticcheck`
  (2025.1.1) + `govulncheck` and runs both as **blocking** steps, and runs the
  suite with `-race`. `make quality` mirrors this so a green local gate means a
  green pipeline.
- **Coverage** is measured (`make cover`) but **not** yet a hard floor: the
  number is understated because most behaviour is exercised through CLI-level
  integration tests in `tests/` that don't attribute to per-package coverage.
  A ratchet-only floor is a follow-up once attribution is meaningful.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Regenerate the date golden and move on | Fixes it for one day; the test rots again at the next date boundary. The seam fixes the class. |
| Introduce `golangci-lint` | New dependency + config surface; `staticcheck` + `govulncheck` were already defined in the Makefile — wire what exists. |
| Inject a full `Clock` interface everywhere | Heavier than the one failing path needs; the env seam is the minimal correct fix. A shared clock package is a Proposed generalization. |
| Hard coverage floor now | Misleading at ~23% aggregate due to integration-test attribution; a floor here would gate on noise. |

## Consequences

- The engine suite is deterministic across dates and race-checked; static
  analysis and CVE scanning block on the trust-root binary.
- `DEVRITES_NOW` is a supported test seam; other date-deriving commands
  (`learnings`, `conventions`, `footprint`, `stuck`) still read `time.Now()`
  directly and can adopt the same seam when they need deterministic output.
- Guard test `tests/adr_0006_clock_seam_test.go` locks the seam behaviour.
