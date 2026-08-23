# Strategy: Workflow Artifact Identity

Mode: hold-rigor

## Intent test

One outcome: make root-owned workflow-only executables independently identifiable, atomic, provable, resumable, and product-identity neutral through one existing module.

## Scope decision

Hold settled scope. Do not expand into a generic materializer, engine command, actual feature workflow executable, historical migration, or release automation. This changes when a real cross-feature executable implementation—not just semantic contract—demonstrates shared runtime behavior that cannot remain feature-specific.

## Minimum usable subset

One indivisible release unit: canonical module; ten thin adapters; durable crash-resumable journal contract; ten provider-neutral routes; deterministic transaction/interruption and product-separation proof; normally generated Claude/Codex mirrors. Removing any part leaves duplicate authority, unsafe resume, or unproved host parity. No smaller subset is usable.

## Forward pass

1. Canonical contract defines admission, staleness, durable states, recovery, evidence, and limits.
2. Deterministic fixtures prove identity and process interruption at every durable/replacement boundary.
3. Adapters delete stale migration and retain only phase-local routing.
4. Existing routing/host assertions move from migration phrases to current identity and crash-resume behavior.
5. Normal generation restores host parity; full proof closes product-identity neutrality.

These are implementation stages inside one atomic slice, not independently shippable slices. Every pre-mortem mitigation binds to that slice.

## Pre-mortem

| Failure after shipping | Likelihood | Mitigation | Build binding |
| --- | --- | --- | --- |
| stale identity executes changed bytes | medium | freeze ordered path/numeric-mode/hash plus Vet binding after green preflight; revalidate under owner lock | SLICE-001 stale/golden fixtures |
| concurrent roots overwrite journal/targets | high | exact umask-077 namespace/bootstrap and only Python `fcntl.flock`; generation/preimage compare; first-create/interruption/mutant fixtures; busy zero writes | SLICE-001 contention fixtures |
| process dies during source promotion, journal, complete writes, install/proof/rollback/cleanup/retry/GC | high | canonical operation table; durable intent before every operation; every-boundary kill/resume | SLICE-001 operation trace matrix |
| partial/invalid write leaves trusted stage or backup | high | strict complete-write loop; exact partial metadata validation; unlink/recreate only exact file | SLICE-001 progress/error/death mutants |
| source path swaps between validation and stage creation | medium | no-follow `fstat`; one bounded descriptor read supplies hash/stage | SLICE-001 source-trust fixtures |
| retry loops forever or loses attempt history | medium | immutable epoch rows; same-fingerprint count 3; death-safe handoff; exhaustion GC/no attempt 4 | SLICE-001 retry matrix |
| proof hangs/leaks descendants or output | medium | fresh process group; per-command/aggregate timeout; terminate/grace/kill/reap; output cap | SLICE-001 proof-timeout matrix |
| workflow journal erases lifecycle evidence | high | exact marker-owned atomic section; outside-byte and single-candidate preservation; malformed-marker fail-close | SLICE-001 evidence fixtures |
| proof file changes product candidate/readiness | medium | actual self-built engine before/after separation fixture | SLICE-001 identity test |
| cold resume repeats obsolete writer migration | high | delete migration; current identity/state only | SLICE-001 scenarios/phrase deletion |
| binding rollover leaves permanent stale bundle | medium | absent-journal precondition; validated `.stale-cleanup`/atomic stale-cleaning rename; idempotent lock-held GC | SLICE-001 rollover interruption fixtures |
| generator deletes siblings or a second filesystem writer crosses the seam | high | hash-bound driver alone owns 16/22 delivery journal/restart and private stage; bounded wright authors and normally launches it; DEC-042 launcher cannot write/recover; root verifies only | SLICE-001 generation rollback and actor-split fixture |
| deterministic consumer self-attests | medium | exact five-field oracle; separate OS/engine observer process; every table column/oracle field mutated | SLICE-001 test review |
| caller cannot act/measure | medium | exact routes/output plus `--prove-walkthrough`; TTHW only command-start to first `CLEANED` return | SLICE-001 routing/TTHW proof |
| syntactically valid limits cannot fit mandatory work | medium | checked relational minima for files, diagnostics, history, attempts, proof times, and evidence headroom | SLICE-001 minimum/overflow fixtures |
| instruction budget exceeds limit | medium | remove duplicate stale prose before contract additions | SLICE-001 baseline ratchet |
| 13 shared Reslice destinations lose accepted packet/route/stop behavior | medium | additive edits from five authored preimages; eight normal generated mirrors; dedicated prior-feature regression before writer commit and root proof; all 30 non-overlap/workspace hashes stay exact | SLICE-001 T-017 composition gate |

## YAGNI ledger

| Candidate surface | Decision | Why |
| --- | --- | --- |
| engine workflow-artifact command | exclude | duplicates semantic policy; no shared deterministic state needs Go |
| reusable generic materializer | exclude | one implementation and feature-specific admitted behavior do not justify seam |
| workflow identity JSON file | exclude | canonical `evidence.md` already owns durable proof/resume facts |
| completed-workspace backfill | exclude | no active operation needs it |
| provider CI calls | exclude | offline CI cannot make stable host-compliance claim |
| separate module | exclude | existing standard passes deletion test and owns callers |

## Dimension floor

| Dimension | Evidence | Band |
| --- | --- | --- |
| Outcome ambition | closes identity, transaction, resume, separation | strong |
| Scope discipline | explicit indivisible MUS; no runtime/dependency | strong |
| User value | cold resume and proof no longer depend on actor history | adequate |
| Testability | stable ten-route table; exact resolver/descriptor/state predicates; three source-loss phases; all-boundary kill/resume; actual engine separation | strong |
| Reversibility | explicit pre-proof rollback and post-proof preserve branches inside one source/generated/test release unit | strong |
| Security | no-follow, exact paths, bounded output, no real action | strong |
| Operations | offline local only | strong |
| Compatibility | no backfill; current commands/reasons preserved | strong |
| Delivery | one atomic candidate after three sealed prerequisites | adequate |

Floor: adequate. No scope delta, human gate, or irreversible-risk gate.
