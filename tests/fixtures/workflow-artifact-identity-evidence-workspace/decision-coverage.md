# Decision coverage

DevRites contract: devrites.readiness-artifacts.v2

Decision coverage: CLEAR

## Topology

| Surface | Kind | Related IDs | Evidence |
| --- | --- | --- | --- |
| Workflow Artifact | canonical semantic module and finite table interface | REQ-001–REQ-009 | DEC-001–DEC-015 |
| Vet admission | exact serializable producer contract and relational limits | REQ-002 | DEC-002, DEC-014; grammar/golden vector in spec |
| Owner/generation | umask-077 descriptor bootstrap plus canonical `fcntl.flock` concurrency seam | REQ-005 | DEC-009; first-create/contention/mutant fixtures |
| Retained preflight source | descriptor-stable input plus pre-journal binding-rollover GC | REQ-003, REQ-004, REQ-007 | DEC-003, DEC-004, DEC-006, DEC-013; DRIFT-002, DRIFT-003 |
| Frozen identity | separate workflow identity | REQ-003, REQ-004, REQ-008 | DEC-003, DEC-005 |
| Artifact-set transaction | table-driven complete-write/branched/retry behavior | REQ-005–REQ-007 | DEC-004, DEC-006, DEC-009, DEC-010; AC-003, AC-004 |
| Canonical evidence | marker-owned generation-checked journal | REQ-005, REQ-006, REQ-008 | DEC-005, DEC-011 |
| Build/Prove/Autocomplete/Recovery/AFK/Vet/One-shot | ten thin entry/action/return adapters | REQ-001, REQ-006 | exact per-adapter table in plan |
| Claude/Codex | generator-derived host adapters installed only by the hash-bound driver; bounded wright authors and normally launches it | REQ-009 | DEC-007, DEC-008, DEC-042; private delivery journal |
| Product candidate/readiness | excluded identities | REQ-008, REQ-009 | actual self-built engine before/after fixture |
| Prior Reslice candidate | 13 shared preimages: five additive authored paths plus eight generator-derived mirrors | REQ-001, REQ-009 | DEC-015; T-017 |
| Completed workspaces | no-backfill and source-free idempotent class | route matrix; PROH-002 | DEC-006 |

## Coverage matrix

| Surface | Dimension | Status | Canonical reference | Owner / validation gate | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| Admission | exact grammar/golden vector plus checked relational target/content/file/diagnostic/journal/attempt/proof minima | closed | REQ-002; DEC-002, DEC-014 | parser, min-minus-one/exact/overflow and hostile fixtures | accepted contract cannot execute |
| Concurrency | exact mkdirat/openat/bootstrap and Python `fcntl.flock`; nonblocking exclusion; generation/owned-hash compare | closed | REQ-005; DEC-009 | first-create/interruption/two-root/lockf-mutant/unsupported/stale-generation fixtures | split-brain roots overwrite evidence/targets |
| Identity timing | freeze after green preflight with Vet binding | closed | REQ-003; DEC-003 | ordering and stale mutants | unproved bytes execute |
| Retained source | exact promotion/descriptor bytes through retryable `FAILED`; validated rollover stale-GC before journal; terminal GC; source-free rerun | closed | REQ-003, REQ-004, REQ-007; DEC-003, DEC-006, DEC-013; DRIFT-003 | promotion/rollover-marker/rename/unlink/retry/swap/loss/terminal-GC fixtures | wrong bytes drive stage or stale bundle blocks forever |
| Preparation | canonical operation table; durable intent and strict complete write before create/write/mode/sync | closed | REQ-004, REQ-005; DEC-004, DEC-009, DEC-010 | every positive partial write plus invalid-progress/error/death mutants | partial stage/backup becomes authority |
| Transaction | success, pre-install failure, pre-proof rollback, post-proof preserve, retry, and exhaustion branches | closed | REQ-005–REQ-007; DEC-004, DEC-006, DEC-010 | branch/source-loss/replace/rollback/cleanup/retry/exhaustion/every-boundary traces | proved set rolls back, partial set survives, or loop never terminates |
| Proof process | fresh process group; command/aggregate timeout; terminate/grace/kill/reap; bounded output | closed | REQ-007; DEC-010 | hang/descendant/output-cap/timeout boundaries | proof hangs or leaks descendants/output |
| Evidence | marker-owned atomic generation-checked section preserving outside bytes and one candidate binding | closed | REQ-005, REQ-006, REQ-008; DEC-005, DEC-011 | prior EVID rows, malformed markers, crash/retry/cleanup/idempotent assertions | workflow journal erases lifecycle proof |
| Resume | current authority plus exact route precedence/actions/cursors; no migration; finite retry | closed | REQ-004, REQ-006, REQ-007; DEC-006, DEC-009 | route overlap, all adapter rows, handoff death, source-free rerun | stale history or wrong caller action controls resume |
| Thin adapters | exact ten entry/action/return rows only | closed | REQ-001; AC-001 | structural/phrase deletion and table-row assertions | duplicate policy drifts |
| Diagnostics | finite reason/boundary/route map and exact bounded ASCII line | closed | AC-006; DEC-009 | every row, collision, unrecognized-input, hostile/secret/path/error mutants | output leaks or caller cannot act |
| Test authority | parsed driver plus exact five-field oracle; separate process derives OS/engine assertion; per-column/per-field mutations | closed | DEC-012 | consumer-report rejection, table and oracle mutants | consumer self-attests own correctness |
| Product separation | workflow paths excluded; candidate/readiness/built count unchanged | closed | REQ-008, REQ-009; DEC-005 | actual self-built engine before/after fixture | proof support changes shipped product |
| Host semantics | bounded wright authors candidate; hash-bound driver alone owns 16/22 private delivery journal; private generation; complete-tree validation/restart/rollback; DEC-042 exact launcher fallback | closed | REQ-009; DEC-008; DEC-042 | delivery-state interruption, staged manifest, sibling equality, rollback, host parity, launcher actor split | root writes product, launcher changes controls/recovers, or partial host set survives |
| Prior-candidate composition | 13 shared Reslice destinations: five authored change additively, eight mirrors derive normally; previous digest historical; workspace/30 non-overlap exact | closed | DEC-015; plan overlap section | dedicated Reslice suite plus shared/non-overlap hashes | Workflow Artifact silently breaks sealed prior semantics |
| Blast radius | 16 authored, 22 generated, six inspected direct references, generator/validators unchanged | closed | spec inventory; plan allowlists | exact preimage, full generated tree, and outside equality | hidden caller or script drift survives |
| Historical compatibility | no writer-exhaustion migration or completed-workspace backfill | closed | REQ-006; DEC-006 | stale/historical route siblings | obsolete actor history controls resume |
| Runtime engine | excluded | closed | PROH-001; non-goals | exact outside-allowlist equality | semantic policy moves into control plane |
| UI/data/auth/network | applicability | not-applicable | spec non-goals | Review | unrelated scope grows |

## Assumption audit

| Assumption | Evidence | Confidence | Owner | Validation | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| ASM-001 rooted fixture and already-available module-selected Go 1.26.5 support actual engine comparison | engine root resolution; rooted tests; observed `go -C engine env GOVERSION` = `go1.26.5` | high | Build | recapture toolchain then self-build actual engine before mutation | block before fixture; no mock/network fallback |
| ASM-002 normal generation mirrors changed canonical Markdown to both hosts when `DEVRITES_HOST_ARTIFACT_DIR` points at private stage | existing generator contract and host test | high | Build | full staged-tree validation, exact 22-file install, parity, outside-manifest equality | stop; restore preimages; never target default tree or hand-edit derivatives |
| ASM-003 stale-prose deletion creates fixed instruction-budget headroom | current 854,997/855,000 baseline and duplicate sections | medium | Build | baseline after exact edit | compact duplicate adapters; never raise cap silently |

## Residual uncertainty

| Item | Why nonblocking | Owner | Validation gate |
| --- | --- | --- | --- |
| Exact compact wording | interface, states, routes, and proof are fixed; wording remains reversible | Build/Vet | thin-adapter and semantic assertions |
| Final canonical byte total | duplicate deletion precedes additions and cap is fixed | Build | instruction baseline |
| Optional pinned Claude/Codex compliance rate | outside unconditional Done claim | future eval owner | claim-bounded report if separately run |
| Full-suite test count | observed, not contractual | Prove | exact full-suite output |

## Readiness verdict

All material module, relational admission, exact flock/bootstrap, source rollover/terminal GC, identity, complete-write/retry, bounded proof, evidence, route/diagnostic, observer oracle, adapter, sole-wright host delivery/restart, prior-candidate composition, rollback, and exclusion decisions have owners and validation. No Partial/Missing row, unowned assumption, human gate, or acceptance ambiguity remains. DRIFT-003 includes final narrow Plan/DevEx findings; DRIFT-004 closes the 13-path Reslice overlap—five authored and eight generated—without expanding acceptance or the writer allowlist. No third reviewer loop is permitted. Deterministic final Vet gates remain before source change.
