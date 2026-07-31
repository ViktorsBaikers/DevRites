# ADR-0027: Content-bound build readiness

- **Status:** Accepted
- **Date:** 2026-08-03

## Context

`check readiness` verifies that the current phase has nonempty required files and
no open human gate. It does not establish that the plan Vet reviewed is the plan
Build will consume. A reproduced Build-phase workspace remained ready after
`plan.md` bytes changed and its original modification time was restored.

Native agents must continue to own semantic judgments such as `CLEAR`, `READY`,
plan quality, and test adequacy. A byte-identity check does not make those
judgments, but it can prevent an old judgment from silently attaching to new
inputs. The identity cannot simply hash every structurally required file:
Build intentionally updates `state.md`, `questions.md`, `decisions.md`, and
`assumptions.md` between slices.

## Decision

- Add one fixed, versioned readiness identity for the stable Build contract. Its
  required records are `spec.md`, `decision-coverage.md`, `architecture.md`,
  `plan.md`, `tasks.md`, `traceability.md`, and `test-plan.md`. Optional records
  bind both presence and absence for `strategy.md`, `design-brief.md`,
  `ai-spec.md`, and `.devrites/principles.md`.
- Exclude mutable cursor, question, decision, assumption, review, drift,
  evidence, handoff, and final-verdict records. A material change discovered in
  one of those records must still fold into a stable owner before Vet may renew
  `READY`.
- Serialize a domain/version, fixed record count and order, logical path,
  presence state, content length, and raw bytes into SHA-256. Reject symlinks,
  non-regular inputs, more than 1 MiB per input, and more than 8 MiB in total.
  The engine adds no dependency, configurable graph, schema file, or second
  state plane.
- `devrites-engine check readiness --emit-binding <slug>` is a read-only helper
  that prints exactly one `Readiness inputs SHA-256: <64 lowercase hex>` line.
  Vet records that exact standalone line in `eng-review.md` only after fold-back,
  proof preflight, semantic review, and recheck are complete.
- Ordinary `check readiness` verifies exactly one current, non-fenced binding
  whenever the target phase requires `eng-review.md`. `check seal` repeats the
  same verification before final candidate-evidence checks. Missing, malformed,
  duplicate, or stale bindings fail closed with a stable reason and route back
  through Vet; the engine never refreshes the record automatically.
- If Build discovers a missing or materially changed `design-brief.md`, shaping
  completes before source work and returns through Vet before dispatch. A
  post-readiness design contract cannot silently drive implementation.

This narrowly supersedes ADR-0022's prohibition on readiness digests. It
preserves that ADR's thin-engine boundary and prohibition on semantic prose
parsing, agent orchestration, compatibility telemetry, and migration machinery.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep structural readiness only | It permits an observed false-ready handoff after reviewed planning bytes change. |
| Hash every phase-required workspace file | Build legitimately mutates several ledgers, causing false invalidation between slices. |
| Use mtimes | Restored timestamps do not identify reviewed bytes. |
| Let Vet run shell-specific hash commands | Host implementations would drift and weaken cross-platform identity. |
| Auto-refresh the binding when stale | Re-labels changed inputs as reviewed without rerunning the semantic owner. |
| Add a configurable artifact graph | Creates another workflow authority for one fixed lifecycle dependency. |

## Consequences

Readiness and Seal now detect stable planning drift independently of model
memory and timestamps while semantic review stays native. Existing unfinished
workspaces without the binding must return through `/rite-vet`; `/rite-upgrade`
already routes readiness defects there. Optional contract artifacts cannot be
added or removed after Vet without visible invalidation.

The identity is a sequential read rather than an atomic filesystem snapshot.
The ordinary gate recomputes it after the binding is written, and Seal repeats
the check; hostile concurrent workspace mutation remains outside the local
agent-workflow threat model.

Guard test: `engine/tests/adr_0027_readiness_binding_test.go`.
