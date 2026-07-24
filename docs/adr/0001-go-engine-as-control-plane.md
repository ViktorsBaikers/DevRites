# ADR-0001: Go engine as deterministic control plane

- **Status:** Accepted
- **Date:** 2026-07-08 (backfilled; decision predates the ADR log)

> Network scope was narrowed by [ADR-0008](0008-sanctioned-engine-network-boundary.md):
> deterministic workspace operations remain network-free, while explicit
> updater/source-cache I/O is isolated in `internal/iohooks`.

## Context

DevRites orchestrates an LLM through a spec-driven lifecycle. Two kinds of work
are entangled: **judgment** (is this spec good? is this code right?) which only
a model can do, and **bookkeeping** (which phase are we in? are the required
sections present? what's the next question id?) which must be exact, fast, and
identical every run. Letting the model do the bookkeeping makes the process
non-reproducible and burns context on arithmetic.

## Decision

Ship a single statically-linked Go binary (`CGO_ENABLED=0`, stdlib-only, zero
third-party deps in the hot path) as the **control plane**: it owns all
deterministic state transitions, gates, and derivations over `.devrites/`. It
makes **no** model or network calls. The filesystem is the data plane; the LLM
supplies judgment. Commands are a hand-rolled `switch` dispatch, not a CLI
framework.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Bookkeeping inside the agent prompt | Non-deterministic, context-hungry, unauditable — the exact failure this system exists to remove. |
| Node/TS engine | Drags a package tree + supply-chain surface; the pure-Go single binary has none and cross-compiles to every target from one runner. |
| Cobra / urfave CLI framework | A dependency and a config surface for a switch statement the stdlib already handles. |

## Consequences

- Reproducible process: same state in, same verdict out, no wall-clock or
  network variance (the one remaining wall-clock read is now seamed — ADR-0006).
- Zero-dependency binary: trivial supply chain, `go install`-able, no runtime.
- Cost: the engine can express only what's deterministic. Anything needing
  judgment must round-trip to the model — by design.
- The engine is the trust root, so it carries the strictest CI gates (ADR-0006).
