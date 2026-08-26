# `.devrites/` state workspace

This is the writer-facing companion to the canonical schema in
[`devrites-lib/reference/workspace-artifact-schema.md`](../../devrites-lib/reference/workspace-artifact-schema.md).
Load that schema before creating or updating workspace artifacts.

## Layout

```text
.devrites/
  ACTIVE
  work/
    <feature-slug>/
      README.md                  # compact workspace map
      brief.md
      spec.md
      architecture.md            # from /rite-define
      flows.md                   # optional Mermaid-first; only when useful
      visual/                    # optional HTML+outline companions (never readiness)
      decisions.md
      assumptions.md
      questions.md
      decision-coverage.md       # from /rite-clarify
      plan.md                    # from /rite-define
      tasks.md                   # from /rite-define
      traceability.md            # from /rite-define
      eng-review.md              # from /rite-vet
      test-plan.md               # from /rite-vet
      state.md
      evidence.md                # from /rite-build or /rite-prove
      browser-evidence.md        # UI only
      drift.md                   # drift only
      touched-files.md           # from /rite-build
      design-brief.md            # UI only
      handoff.md                 # from /rite-handoff
      references.md
      references/
  archive/
    <feature-slug>/
```

## `flows.md` (optional Mermaid-first)

Write `flows.md` only when sequence/state/data/lifecycle diagrams clarify the feature.
Keep **Mermaid in `flows.md`** when that is enough. When a richer reviewable presentation
is needed, **also** emit `.devrites/work/<slug>/visual/<flow>.html` +
`visual/<flow>.outline.md` and link the pair from `flows.md`.

Before any HTML:

1. Open matching playbooks via
   [`../../devrites-lib/reference/visual-playbooks/index.md`](../../devrites-lib/reference/visual-playbooks/index.md)
   (progressive; never preload all seven).
2. Copy required outline headings from
   [`outline-template.md`](../../devrites-lib/reference/visual-playbooks/outline-template.md).
3. Dual-read: agents treat the outline as SSOT; **outline wins** on conflict. No Lavish
   runtime; this is not a new lifecycle phase and never inflates readiness.

`/rite-spec` may seed a thin `flows.md` when investigation already needs a diagram;
`/rite-define` owns richer architecture/flow companions beside `architecture.md`.

## Creation rules

- Create or reuse the slug exactly under the canonical schema's
  [Slug identity](../../devrites-lib/reference/workspace-artifact-schema.md#slug-identity)
  contract.
- `/rite-spec` creates the workspace map, `brief.md`, `spec.md`, `decisions.md`,
  `assumptions.md`, `questions.md`, `state.md`, optional `references.md` /
  `references/`, optional `flows.md` when a diagram already clarifies investigation,
  and optional `design-brief.md` for UI.
- `/rite-clarify` adds `decision-coverage.md`.
- `/rite-define` adds `architecture.md`, `plan.md`, `tasks.md`, and
  `traceability.md`; may add or enrich `flows.md` and optional `visual/` HTML+outline
  companions when Mermaid alone is not enough.
- `/rite-vet` adds `eng-review.md` and `test-plan.md`.
- Later phases add only the artifact they own. Do not create optional files as
  empty placeholders; absence means the phase has not produced that artifact.
- Each artifact starts with a summary and links to deeper source files instead of
  copying their content.
- Use stable IDs: `REQ-001`, `AC-001`, `EDGE-001`, `PROH-001`, `SLICE-001`,
  `DEC-001`, `ASM-001`, `DRIFT-001`, `EVID-001`; apply the canonical schema's
  append-only ID contract. Queued questions use `q-YYYY-MM-DD-NNN` below.

## README.md template

```markdown
# <Feature> Workspace
phase: spec
status: running
next_action: /rite-clarify after spec readiness passes
last_updated: <date>

## Artifact map
| File | Job |
| --- | --- |
| brief.md | Objective and bounds |
| spec.md | Product contract |
| decision-coverage.md | Clarification verdict; absent until /rite-clarify |
| architecture.md | Technical map; absent until /rite-define |
| plan.md | Technical approach; absent until /rite-define |
| tasks.md | Vertical slices; absent until /rite-define |
| traceability.md | Coverage dashboard; absent until /rite-define |
| evidence.md | Proof log; absent until /rite-prove |

## Read next
| Phase / role | Read |
| --- | --- |
| Spec | brief.md, spec.md, questions.md |
| Clarify | spec.md, decisions.md, assumptions.md, questions.md |
| Build | state.md, decision-coverage.md, eng-review.md, test-plan.md, tasks.md, plan.md |
| Review | traceability.md, evidence.md, decisions.md |

## Blocking gates
None.
```

## state.md template

```markdown
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | spec |
| status | running |
| active_slice | none |
| slice_mode | none |
| risk | none |
| next_action | <single command + reason, or terminal: none — technical recovery exhausted for an unchanged causal fingerprint> |
| return_phase | <originating later phase; clarification or agent-owned technical backtracking only> |
| return_next_action | <saved originating command> |

## Awaiting human
Only present when status is awaiting_human.
| Key | Value |
| --- | --- |
| question_id | q-YYYY-MM-DD-NNN |
| gate | blocking |
| blocking_slices | SLICE-001 |
```

`state.md` is a compact cursor, not a history file. Put proof in `evidence.md`,
decisions in `decisions.md`, assumptions in `assumptions.md`, and drift in
`drift.md`. Omit both `return_*` rows outside a later-phase `/rite-clarify`
retrofit or agent-owned technical backtracking chain.

Before an active caller moves backward, it copies its current phase and
non-empty action into the two return rows. Nested Plan, Vet, Build, and proof
work preserves that valid cursor. Once the prerequisite chain is green, the
controlling caller restores the saved phase/action and removes both rows in one
rewrite. A real HITL or exhausted-recovery stop retains the cursor for cold
resume. Never overwrite an existing valid return cursor, and preserve every
unrelated Markdown byte. `/rite-clarify` applies its stricter native cursor
protocol below.

On cold resume, a terminal `next_action` is a claim to verify, not a counter.
Reconcile the exact fingerprint and its attempts in `drift.md` / `evidence.md`.
If a consumptive action retained a distinct fingerprint below the no-progress
cap, rewrite the cursor into caller-owned recovery and preserve or reconstruct
the return rows only from the current phase and approved recorded action. The
spent action authorization blocks another execution, not that rewrite.

`afk_slices_remaining` is mutable runtime state, not `.devrites/AFK`
configuration. Only the controlling root writes it under the shared
[`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md#the-sentinel-devritesafk)
contract and Build's
[`afk-discipline.md`](../../rite-build/reference/afk-discipline.md#iteration-cap).
Ordinary workspace creation preserves the field without initializing,
increasing, renaming, or deleting it. Released bullet workspaces spell it `AFK
slices remaining`; preserve that presentation rather than migrating it.

## questions.md entry format

```markdown
## q-YYYY-MM-DD-NNN
status: open | answered | dropped
slice: spec | plan | SLICE-001
gate: advisory | validating | blocking | escalating
question: <one crisp sentence>
options:
  1. <recommended> (Recommended) - <trade-off>
  2. <alternative> - <trade-off>
proposed: <best current answer>
raised_at: <iso>
answered_at:
answer:
impact: <affected AC/REQ/slice IDs>
```

No open blocking/escalating question may remain before define/build/prove gates.
For a new entry, scan every `## q-YYYY-MM-DD-NNN` header for the current date,
choose one above the highest suffix (or `001` when none exist), advancing until
it is unused, then re-read
`questions.md` immediately before the append and recompute. If the candidate is
no longer unused, retry the scan; never reserve or derive it with an engine
command.

## brief.md template

```markdown
# Brief: <Feature>

## Objective
<One sentence.>

## Non-goals
- <Explicitly out of scope.>

## Success definition
<One sentence that says what "done" means.>
```

## Compactness

High-traffic files should stay within the budgets in
`workspace-artifact-schema.md`. If a file truly must exceed its budget, add
`Budget override: <reason>` near the top.
