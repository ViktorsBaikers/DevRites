# Workspace artifact schema

Feature truth: `.devrites/work/<slug>/`, discovered by `state.md`; remnants/archives
are inactive. Spec writes optional `README.md`. Use canonical filenames; both released
cursor forms remain valid. `$rite-upgrade` audits without path/alias/format migration.

## Slug identity

New slugs derive from the objective: lowercase ASCII kebab-case matching
`[a-z0-9]+(?:-[a-z0-9]+)*`, at most 64 characters. Shorten at word boundaries.
After the final shortening or suffix step, trim boundary hyphens; reject empty results.

Before creation, compare a matching workspace's `brief.md` objective: reuse only the
same feature. Otherwise append the smallest unused suffix (`-2`, `-3`, ...), keeping
the trimmed base within 64 characters. Never overwrite a workspace by slug alone;
accept safe legacy basenames. Ordinary phases never rename them.

## Required by phase

| Phase | Required artifacts |
| --- | --- |
| frame | `state.md` |
| spec | `README.md`, `brief.md`, `spec.md`, `state.md`, `decisions.md`, `assumptions.md`, `questions.md` |
| clarify | spec artifacts plus `decision-coverage.md` |
| temper | clarified spec artifacts plus `strategy.md` when temper runs |
| define/plan | clarified spec artifacts plus `architecture.md`, `plan.md`, `tasks.md`, `traceability.md` |
| vet/build/converge | plan artifacts plus `eng-review.md`, `test-plan.md`; Build creates and maintains `touched-files.md` after its first green slice |
| prove/polish/review | vetted plan artifacts plus `evidence.md`, `touched-files.md` |
| seal/ship/done | proof artifacts plus `review.md`, `seal.md`; Ship writes `ship.md` |
| conditional | `flows.md` when diagrams clarify; `visual/` HTML+`.outline.md` companions when a richer reviewable visual earns it (optional; never readiness-required); `design-brief.md` and `browser-evidence.md` for UI; `drift.md` for drift; `polish-report.md` for polish; `handoff.md` only when requested; `references.md` + `references/` when references exist; `investigation-map.md` for pressure-test; `dogfood.md` for dogfood; `packets/` for by-reference dispatch packets and admitted accounts (≤ 64 KiB each; older closed rounds may be deleted; never transcripts); `history/` for verbatim relocated checkpoint narrative (append-only, never a read-next) |

## What each file owns

| File | Job | Budget |
| --- | --- | --- |
| `README.md` | map: phase/status/next action, artifacts/read-next, blocking gates, last update | 120 lines |
| `brief.md` | user request, objective, non-goals, success definition | 80 lines |
| `spec.md` | product WHAT/WHY, requirements, acceptance criteria, edge cases, measurable success, scope boundaries | 260 lines |
| `decision-coverage.md` | topology-first coverage matrix, assumption audit, residual uncertainty, and typed clarity verdict | 200 lines |
| `strategy.md` | temper verdict, scope mode/deltas, pre-mortem risks, deferred ambition | 180 lines |
| `architecture.md` | owning layer, integration points, data/API/events, dependencies, risks, affected boundaries | 180 lines |
| `flows.md` | useful Mermaid sequence/state/data/lifecycle diagrams with why-it-matters text and related IDs | 160 lines |
| `visual/<name>.html` | optional portable human-viewable visualization; pair with sibling `.outline.md`; self-contained preferred | 400 lines |
| `visual/<name>.outline.md` | required machine dual-read companion for the sibling HTML; outline wins on conflict; never a candidate path | 200 lines |
| `visual/README.md` | optional index of visuals in the workspace | 80 lines |
| `decisions.md` | ADR-style `DEC-###` log: status, context, options, decision, consequences, related IDs | 200 lines |
| `assumptions.md` | assumptions with confidence, owner, validation status | 160 lines |
| `questions.md` | current `q-YYYY-MM-DD-NNN` (or released `Q-###`) open/resolved questions, gate, answer, impact | 180 lines |
| `plan.md` | technical approach, slice strategy, canonical shared-contract proof, validation strategy, rollback | 220 lines |
| `tasks.md` | `SLICE-###` vertical slices with AC IDs, likely files, tests/proof, mode/gate, dependencies, done condition | 280 lines |
| `traceability.md` | matrix: AC/REQ ID, slice IDs, test/proof, evidence ID, touched files, status | 220 lines |
| `eng-review.md` | vetted scope/architecture/quality/performance, failure modes, preflight, stable-input binding, `## Deferred findings` (late rows: severity · site · role · kind · round) for Review | 240 lines |
| `test-plan.md` | executable proof commands, preflight/provenance contract, acceptance and interaction coverage | 260 lines |
| `state.md` | compact cursor table/key-values; no narrative log | 120 lines |
| `evidence.md` | `EVID-###` command/action, result, timestamp if available, related AC/slice IDs, limitation | 280 lines |
| `browser-evidence.md` | UI route/viewports/screenshots/console/network/interactions and Visual Verdict | 220 lines |
| `drift.md` | `DRIFT-###` spec/plan drift and resolution | 160 lines |
| `touched-files.md` | sole candidate manifest plus a concern-ordered `## Review trail` of `path:line` stops for human review and Build's deferred late findings (severity · site · role · round) that `$rite-review` must close | 160 lines |
| `design-brief.md` | UI design direction, states, interaction model | 160 lines |
| `handoff.md` | resume objective/last slice/next action/blockers/read-next; sections hold content or `Nothing yet`; unevidenced recollections: `Not tried yet`, never results | 120 lines |
| `polish-report.md` | polish axis deltas, re-verification lines, residual subjective preferences | 200 lines |
| `ship.md` | ship preflight, disclosed Git plan, commit/push/tag/PR record | 120 lines |
| `review.md` | reconciled Review verdict: `## Spec` and `## Code review` accounts, deferred-finding labels/actions, drift | 240 lines |
| `seal.md` | seal preflight, exact reviewer roster accounts, GO/NO-GO verdict and disclosures | 200 lines |
| `ai-spec.md` | model/RAG/agent/eval/LLM-output scope for AI surfaces | 160 lines |

For visual pairs load only matching [playbooks](visual-playbooks/index.md); use
the [required outline headings](visual-playbooks/outline-template.md).

## Candidate manifest and bindings

`touched-files.md` contains exactly one `## Touched files` heading and exactly
one authoritative `## Candidate manifest` heading. The first describes scope
without repeating candidate paths; the manifest is the sole authority for candidate scope.

The manifest body is exactly either `No project files.` or this table with one
or more rows:

```markdown
| State | File | Slice | Reason |
| --- | --- | --- | --- |
| present | `path/to/file` | SLICE-001 | Observable reason |
```

Rows are sorted by File: they must already be strictly sorted by normalized
`File`, and readers reject rather than reorder them. `State` is exactly
`present` or `deleted`; `File` is a unique
project-relative UTF-8 path wrapped in one backtick pair; `Slice` and `Reason`
are nonempty. Paths must also be unique under case folding. Every component
rejects Windows-reserved characters and device names (including names with an
extension) plus a trailing dot or space. `## Review trail` may cite
concern-ordered `path:line` review stops, but it cannot define or expand
candidate scope; only manifest rows do.

The public candidate limits are a 1 MiB manifest, 4,096 rows, a 4,096-byte
path, 64 MiB per present file, and 256 MiB across all present files.

A release milestone's union manifest is produced by `devrites-engine state
merge-manifest <release-slug>` — it walks the recorded `sequence_parent` chain,
folds every predecessor manifest, and resolves a path collision in favor of the
later sequence position; a hand-assembled union is not authoritative. When the
workspace cursor declares `sequence_role: release`, `check candidate` and
`check seal` require the manifest to cover the whole recorded chain.

Workspace and audit artifacts are not candidate paths, including all
`.devrites/work/<slug>/visual/`. Durable owners include `.devrites/specs/**`,
`DESIGN.md`, and `docs/adr/**`, plus `.devrites/principles.md`. Under `.devrites`,
only those spec/principles owners qualify; all other siblings (`ACTIVE`, `AFK`,
`CHECKPOINT`, `archive/**`, `work/**`) fail closed. Engine owns malformed path, type,
and size rejection; phases never reinterpret a rejected manifest.

`evidence.md`, `review.md`, and `seal.md` each contain exactly one unindented standalone
binding line; `browser-evidence.md` does too when that file exists:

```text
Candidate SHA-256: <64 lowercase hex>
```

Old unfinished workspaces refresh the manifest and rerun real proof. They never
synthesize a historical pass or use a legacy fallback.

The worktree digest is deterministic but is not an atomic filesystem snapshot
against a malicious concurrent same-size rewrite. Ship's exact Git-index scope,
byte, binding, and secret checks own the final freeze immediately before commit.

Only a structural `Budget override: <reason>` line (not a fenced example)
permits excess. Each line budget also bounds bytes at 400 bytes per budgeted line
(`state.md` 120 lines → 48 KiB); long lines do not evade it.
`devrites-engine orient <slug>` reports `artifact_budgets` (lines/bytes/over/override)
and `bulk_files` (non-canonical files over 64 KiB) so a phase can decide what to read
before loading. Phase writers check these limits before returning; there is no
repository-local script assumption in the installed workflow.

`check readiness` and `check seal` enforce the material band deterministically: a
required artifact at or above 1.5× its line or byte budget with no structural override
line blocks with the exact measurement and the three legal remedies (relocate history
via `$rite-plan revise`, split via `$rite-plan course-correct`, or record the override).
Overshoot below that band stays advisory and is reported by `orient`. Proof ledgers
(`evidence.md`, `browser-evidence.md`, `touched-files.md`) are always advisory: they
scale with proof volume, and `touched-files.md` keeps its own manifest limits.
Because they are never gate-blocked they are also the only artifacts that can grow
unboundedly — so phases read them by index, not whole-file: the standalone binding
line, a `EVID-###`/section-header index (`grep -n`), then bodies only for entries
naming this candidate's AC/slice IDs. An entry naming no in-scope ID is anomaly
evidence to inspect, not skip. A whole-file read is allowed only when `orient`
reports the artifact within budget; completeness for seal is engine-checked
(literal `AC-###` presence), the reader judges quality.

Two reasons are not durable fixes: slice or acceptance-criteria count (an over-budget
`tasks.md`/`plan.md`/`traceability.md` from decomposition size is a
[feature-sequence split](../../rite-plan/reference/slicing.md#feature-ceiling-split-an-epic-never-override-the-budget)),
and history in `state.md` (checkpoint narrative moves verbatim to
`history/<file>-<YYYYMMDD>.md`, append-only, never a phase read-next, with a one-line
pointer left behind). A legacy workspace may carry a temporary `Budget override:` line
as a bridge; `$rite-upgrade` routes it to relocation or the split, and new writers never
create one for those two reasons.
Apply [[`context-hygiene.md`](standards/context-hygiene.md) dispatch packets](standards/context-hygiene.md#dispatch-packets)
for byte-size checks and lossless retrieval; budget overrides neither rewrite protected
history nor require loading it all.

## Native readiness ownership

`decision-coverage.md` owns topology; `test-plan.md`, executable proof; `eng-review.md`,
reviewer verdicts, `Implementation readiness`, and `Readiness inputs SHA-256`. Root
re-reads their IDs/meaning. The digest binds bytes, never semantic correctness.
Readiness checks structure/binding; Seal rechecks both.

Proof commands must be repository-portable: no host wrappers, user-specific absolute paths,
or temporary proof trees. Evidence records the executed command. Optional `visual/`
artifacts never inflate readiness: they are not readiness inputs and do not substitute
for `decision-coverage.md`, `eng-review.md`, or `test-plan.md`.

## Canonical slice grammar

Every producer of `tasks.md` uses this field set. `Dependencies` is slice
ordering; package and service requirements belong in `External dependencies`.

<!-- canonical-slice:start -->
```markdown
## SLICE-001 <observable capability>
Goal: <single observable capability>
Satisfies: AC-001[, AC-002]
Acceptance criteria: <binary criteria this slice closes>
Complexity: <1..5> — <reason>
Mode: <AFK | HITL>
Gate: <advisory | validating | blocking | escalating>
SLA: <15m | 4h | 24h | none>
Checkpoint: <question + why it needs unavailable pre-code evidence or action-time approval | none>
Dependencies: <SLICE-### list | none>
depends_on: [<SLICE-### IDs>]
Consumes / Produces: <interfaces read and exposed>
Known-Gotchas: <ordering hazards and framework footguns | none>
Prior-slice learnings: <constraints learned earlier | none>
Files likely touched: <real paths>
Tests/proof: <exact command + cwd + expected signal + prerequisites/provenance inputs>
Browser proof required: <yes | no>
Frontend craft required: <yes | no>
Design brief states: <UI states/interaction | none>
Visual acceptance: <state × viewport × input target | none>
Fullstack (FE+BE): <yes | no>
External dependencies: <libraries/services | none>
Existing to reuse / extend: <components/utilities/patterns | none>
Rollback notes: <reversal path>
Evidence required: <evidence $rite-prove must capture>
Edge/Prohibition coverage: <EDGE/PROH IDs | none>
Done condition: <checkable, exhaustive completion criterion>
```
<!-- canonical-slice:end -->

`depends_on` is the machine-readable mirror of `Dependencies`; keep the sets
identical and cycle-free. `Gate`, `SLA`, and `Checkpoint` are required for HITL
slices; use `none` when they do not apply. Complexity above 3 triggers reslicing
unless the stated reason makes the boundary irreducible.

## Read next by phase

| Phase | Read |
| --- | --- |
| spec | `README.md`, `brief.md`, `spec.md`, `references.md`, `questions.md` |
| clarify | `README.md`, `state.md`, `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, `questions.md` |
| temper | `spec.md`, `decision-coverage.md`, `decisions.md`, `assumptions.md`, `design-brief.md` |
| define | `README.md`, `state.md`, `spec.md`, `decision-coverage.md`, `architecture.md`, `decisions.md`, `assumptions.md` |
| vet | `state.md`, `spec.md`, `decision-coverage.md`, `README.md`, `traceability.md`, `plan.md`, `tasks.md`, `architecture.md`, `decisions.md` |
| build | `devrites-engine orient <slug>` first; then `state.md`, `eng-review.md` (Build readback + `## Deferred findings`), the selected slice and its `depends_on` via `devrites-engine observe slice <slug> <SLICE-ID>`, the `test-plan.md` and `traceability.md` rows that slice's AC IDs name, open `questions.md` gates, and `touched-files.md` when present. Load whole `tasks.md`/`plan.md`/`architecture.md`/`decision-coverage.md` only when `orient` shows them within budget or the slice cites a section by ID |
| prove | `traceability.md`, `touched-files.md`; each built slice's `Tests/proof` and `Evidence required` via `observe slice`; `evidence.md`/`browser-evidence.md` via the bounded advisory read above |
| review/seal | `README.md`, `traceability.md`, `spec.md`, `decisions.md`, `drift.md`, `touched-files.md`; `evidence.md`/`browser-evidence.md` via the bounded advisory read above |
| handoff | `README.md`, `state.md`, `handoff.md`, then the linked source artifacts |

## Do not duplicate

- Do not make `spec.md` carry deep architecture; link to `architecture.md`/`flows.md`.
- Do not copy acceptance criteria into `plan.md`; reference `AC-###`.
- Do not copy full proof into `handoff.md`; link to `evidence.md`.
- Do not make `state.md` an append-only log; keep only the current cursor. A
  checkpoint entry is at most three lines: what changed, the canonical record path or
  DEC/EVID ID, and the next action. Digests, packet hashes, and finding narratives
  belong to the record they identify, not the cursor.
- Give each normative fact one owner and stable ID; reference rather than copy it.
  Required schema mirrors remain required. Derive counts/mirrors or mechanically
  check equality using available tools, never invented commands. Repair stale
  derived values and assess consumer impact; they are not new policy decisions.
- Do not create optional files before their phase; absence is meaningful. Do not
  treat `visual/` as required for readiness; emit HTML+outline only when a writer
  earns a richer visual, and keep Mermaid in `flows.md` when that is enough.

## ID contract

Use stable IDs: `REQ-001`, `AC-001`, `EDGE-001`, `PROH-001`, `SLICE-001`,
`DEC-001`, `ASM-001`, `DRIFT-001`, `EVID-001`. Current queued questions use
`q-YYYY-MM-DD-NNN`; released table registers may retain `Q-001`. IDs are
append-only identities, not display positions:

- Allocate the next unused numeric suffix for that prefix after scanning its
  owning artifact. For a dated question ID, scan the current date's suffixes.
  Never fill a gap; every deleted or retired ID remains consumed.
- Never renumber an ID, reuse it, or transfer it to a different meaning. Reordering
  sections or rows does not change IDs.
- A wording edit that preserves the obligation keeps its ID. A materially different
  meaning gets a new ID; record the relationship in `decisions.md` or `drift.md`
  before updating downstream references.
- Re-read the owning artifact immediately before an append when another writer may
  have changed it; recompute rather than reserving an ID from stale state.

Old `AC1` and `Slice 1` forms are legacy and should not be generated for new
workspaces. Preserve released legacy forms unless an explicit upgrade owns the
migration; never renumber them incidentally.

