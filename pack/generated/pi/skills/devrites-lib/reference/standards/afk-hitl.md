# AFK & HITL: the pause/resume contract

Defines two modes for `rite-*`/`devrites-*` skills that may pause; main
callers are `/rite-build`, `/rite-status`, `/rite-resolve`, and
`devrites-doubt`.

The contract uses one sentinel, one queue, and one resume verb.

## Contents

- [Run modes](#run-modes)
- [The sentinel: `.devrites/AFK`](#the-sentinel-devritesafk)
- [Unattended resource envelope](#unattended-resource-envelope)
- [The four gates](#the-four-gates)
- [Option set: how every gap is presented](#option-set-how-every-gap-is-presented)
- [Decision ownership: search before asking](#decision-ownership-search-before-asking)
- [Irreversible-risk list (always pause)](#irreversible-risk-list-always-pause)
- [`questions.md` schema](#questionsmd-schema)
- [`state.md` `Awaiting human` block](#statemd-awaiting-human-block)
- [The resume verb: `/rite-resolve`](#the-resume-verb-riteresolve)
- [AFK exception for discretionary pauses](#afk-exception-for-discretionary-pauses)
- [Retry cap, no-progress loops, and self-resolve](#retry-cap-no-progress-loops-and-self-resolve)
- [What the rule does NOT cover](#what-the-rule-does-not-cover)
- [Cross-reference](#cross-reference)

## Run modes

- **HITL (default):** human is present. At a gap/checkpoint the skill **asks inline** via
  the harness `AskUserQuestion` tool: a ranked **option set** (recommended first, each with
  dimension-tagged rationale; see [Option set](#option-set-how-every-gap-is-presented)). The
  human picks; the skill records the pick to `questions.md` (`answered`) + `decisions.md` and
  **continues in place: no `/rite-resolve` round-trip**. `/rite-resolve` is only for answering
  **async** (a pause that already stopped the session) or in **batch**.
  **If the current surface has no interactive question tool** (Codex outside Plan mode:
  `request_user_input` is Plan-mode-only.) Render the same option set as a plain numbered
  list in chat and **end the turn**. The human's reply is the selection. Auto-picking an option
  is **AFK's contract, gated by the `.devrites/AFK` sentinel**: a missing tool never
  converts a HITL gap into a self-answered one.
- **AFK:** `.devrites/AFK` is present. For any gate AFK may auto-handle (severity in
  `allow_gates`), the skill **auto-picks the recommended option** (option 1 of the set), records
  it (`gate: advisory` + a `decisions.md` ADR), and continues unattended. Gates above the
  ceiling (and every irreversible-risk item) pause and queue a `questions.md` entry for
  `/rite-resolve`.

`.devrites/AFK` presence is authoritative for run mode; gate-deciding skills re-read
it at decision time. There is no `state.md` run-mode field to drift out of sync.

## The sentinel: `.devrites/AFK`

Presence = AFK active. The file body is optional YAML:

```yaml
max_slices: 10                       # whole-workspace writer budget; seeds state.md remaining count
max_agents: 32                       # native agent dispatches in one host activation
max_minutes: 120                     # wall-clock minutes in one host activation
max_review_queue: 8                  # unresolved review/gate items admitted before fan-out stops
# max_tokens: 200000                 # optional stricter host-observed token cap
# max_cost_usd: 10                   # optional stricter host-observed cost cap
notify: "ntfy.sh/my-topic"           # shell command; examples: rite-build/reference/afk-discipline.md
allow_gates: [advisory, validating]  # gate severities AFK auto-handles (auto-picks the recommended option)
continue_sequence: true              # after Seal GO, open the next recorded continuation
max_workspaces: 5                    # workspaces one armed sequence may open
max_parallel: 10                     # cap on eligible path-disjoint Build batches (1 = serial)
```

The file is **read-only config**: never rewritten in place. `max_slices` is the initial
budget; the mutable remaining count is the `state.md` cursor
`afk_slices_remaining` (`AFK slices remaining: <n>` in the released bullet
form), owned by the controlling root. Recognize either spelling and preserve
the existing table or bullet presentation. Before dispatch, a configured
`max_slices` and any existing remaining value must be decimal nonnegative
integers; malformed or negative values fail closed. The pending → built state transition spends
exactly one slice: on the first green slice write `max_slices - 1`, otherwise
write `remaining - 1`, never below zero. Re-reading an already built slice does
not spend again. At zero, stop before the next dispatch.

An orchestrator that can derive a stricter budget only after planning may pre-seed
`afk_slices_remaining` in `state.md` before the first dispatch instead of rewriting the
sentinel. It uses the minimum of pending work and every configured/explicit cap. An
existing counter may be lowered but never increased or reinitialized. Once present, that state
counter is the effective remaining budget even when the read-only sentinel omits
`max_slices`.

### Sequence continuation (`continue_sequence`, `max_workspaces`)

`continue_sequence: true` lets one armed run open the next **recorded** continuation
after a `Seal GO` instead of stopping. Absent means `1` — the current workspace only;
an existing sentinel never receives the default implicitly. `max_workspaces` is the
sequence budget (nonnegative decimal, fail closed): spend one when a new workspace is
opened, never when resuming the current one. `max_slices` stays **per workspace** —
each new workspace's `state.md` seeds its own `afk_slices_remaining`.

The chain lives in root-owned `state.md` cursor fields, not in chat:
`sequence_parent` (the immediately preceding workspace slug), `sequence_position`
(1-based ordinal), `sequence_workspaces_remaining` (slots the chain may still
open after this workspace), and — on the release milestone only —
`sequence_role: release`. Seeding is exactly-once because the values are
derived, not remembered: a continuation's fields are computed from its recorded
parent and the parent's counter at creation, and a resumed run that finds a
continuation workspace missing the fields re-derives the same values from its
`brief.md` parent/position plus the parent's counter — never a second charge. A
workspace without a recorded parent/position is not a sequence member; the
counter only decreases, malformed values fail closed, and `0` opens nothing.
`sequence_role: release` makes `check candidate` and `check seal` require the
candidate manifest to cover the whole recorded chain — run
`devrites-engine state merge-manifest <slug>` before Prove.

- Only continuations recorded in the parent's `decisions.md` sequence
  ([slicing.md § Continuation workspaces](../../../rite-plan/reference/slicing.md#continuation-workspaces))
  qualify; no recorded next entry, or the cap/review-queue bound reached, stops
  with the winning reason.
- Milestones are **not** shipped: they stay sealed and unarchived in `.devrites/work/`
  with `Next step: /rite-ship` preserved. Per-slice `WIP(<slug>)` checkpoints still
  land locally ([checkpoint.md](../../../rite-build/reference/checkpoint.md)), so the tree
  stays clean and crash-safe.
- Git stays human-gated. The release ship is one disclosed plan, one literal `GO`, and
  one native approval; it collapses the sequence's `WIP` commits into the single
  release commit and may archive the sealed predecessors.
- Every human-owned, safety, access, exhaustion, and `NO-GO` condition still stops the
  run exactly as without this field.

`max_parallel` caps a Build batch for unattended runs; `1` forces the serial cycle.
Unattended runs take the largest eligible set ≤ cap and recompute after every completed
round (serial slice or parallel integrate)
([parallel-batch.md § Dynamic selection and re-batching](../../../rite-build/reference/parallel-batch.md#dynamic-selection-and-re-batching));
an explicit `/rite-build --parallel N` cap wins for that invocation.

## Unattended resource envelope

AFK writer admission needs a bounded input queue, effective slice cap, and valid
`max_agents`, `max_minutes`, and `max_review_queue`. Existing sentinels
that declare writer admission but miss/malform these fail closed; cold resume keeps the
state-owned slice counter. A leftover `expires_at` is ignored and never rewritten.
Read-only watchers use equivalent native caps from [`loop-operations.md`](loop-operations.md).

`max_agents` counts every leaf in the native activation, including failures and
parallel branches; do not add dispatch telemetry to `.devrites/`. `max_review_queue`
counts open validating questions plus unresolved admitted Critical/Important findings.
Above it stop; at it run only reconciliation that reduces the queue. Optional
`max_tokens`/`max_cost_usd` lower enforceable native caps; if declared but unobservable, stop.

Numeric limits are nonnegative decimals. Before costly checks, fan-out, or writing,
run cheap readiness, reject overlap, count queue, and confirm agent/time/token/cost
headroom; re-check after every result. Never start one call that can exceed remaining
headroom. Agent/time/token/cost counters are per native activation and start fresh only
for a genuinely new activation. Slices, recovery attempts, and current
review queue remain durable/recomputed across wakes. Persist each activation stop and
checkpoint before notification.

New sentinels write no `expires_at` and no notification/token/cost cap.
Post-Vet pending count may lower slices. Existing files
never receive missing defaults implicitly.

To leave AFK, delete the file. The next skill invocation reverts to HITL.

## The four gates

Every `Mode: HITL` slice declares a `Gate:` and an `SLA:`. See
[`.pi/skills/rite-define/reference/gates.md`](../../../rite-define/reference/gates.md)
for the full taxonomy. Summary:

| Gate | Stakes | Pause? | SLA | AFK auto-handle when in `allow_gates`? |
|---|---|---|---|---|
| advisory | low | no | none | yes (log + proceed) |
| validating | medium | async | 4h | yes (build + queue, no merge until resolved) |
| blocking | high | sync | 15m | **no** (always pauses) |
| escalating | novel pattern | sync to specialist | 24h | **no** (always pauses) |

`blocking` and `escalating` always pause regardless of `allow_gates`.

An open `gate: validating` entry is **merge-blocking by definition**: at `/rite-seal` any
`questions.md` entry with `gate: validating` and `status: open` is a NO-GO, regardless of
its behavior impact. A slice marked `built (pending review)` is **not done** until that
validating gate resolves.

## Option set: how every gap is presented

Wherever a gap, checkpoint, or non-trivial decision surfaces (`/rite-spec`, `/rite-clarify`, `/rite-define`,
`/rite-build`, `/rite-temper`, `/rite-vet`, `devrites-doubt`, `devrites-interview`), present a
**ranked option set**, never a single bare guess:

- **2-4 concrete options**, the **recommended one first**, labelled `(Recommended)`.
- Each option has a **one-line, dimension-tagged rationale**: `logic · infra · business ·
  architecture` (add `security` / `UX` / `risk` in scope) and trade-off.
- Always include an escape hatch (`Something else — describe it`).
- With more than four materially distinct choices, first ask a discriminating question or use
  sequential packets, then obtain final confirmation. Materially distinct options MUST NOT
  be silently dropped, merged, or preselected to fit the UI.
- Recommend for project conventions, stack, scale, and domain, not a generic default.

**HITL** renders the set via `AskUserQuestion`; the human's pick resolves the gate **in place**.
**AFK** auto-picks option 1 for gates it may auto-handle. Record the chosen option verbatim and
keep the **rejected options in `questions.md`**.

## Decision ownership: search before asking

A gate is human only when its remaining choice is human-owned. First search live code,
project/decision docs, and authoritative dependency sources; make and record reversible
implementation/test choices. Ask only about product, scope, acceptance, architecture policy,
irreversible risk, or human-only access/action.

Objective test/build/tool failure runs bounded `devrites-debug-recovery`; fix it or record a
technical blocker. Never ask permission for another attempt, test, parser repair, or probe.
Close decisions at the earliest informed phase: product in spec, coverage in clarify,
scope/risk in temper, architecture/dependencies in define, and proof/toolchain in vet. Build
keeps only unavailable-pre-code or mandatory action-time checkpoints.

## Irreversible-risk list (always pause)

The following always invoke the checkpoint protocol, regardless of `Mode`, `Gate`, or
`allow_gates`:

- Destructive data migration (drop column, drop table, irreversible backfill).
- Auth / authz boundary change.
- Public API break (response shape, removed endpoint, changed status code semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.

When a pause clears and you proceed with a destructive migration, a removal, or a
public-API break, take the **safe path** the gate stopped you for: expand→contract,
prove the old path unused before removing it, and a rollback for every destructive step
([`deprecation.md`](deprecation.md)). The gate requires the safe path; it does not
cancel the work.

By default, AFK widens what's *automatic*; it never widens what's *irreversible*.

Red checks remain hard non-advance build gates, but are not inherently irreversible or
human-owned; bounded recovery owns them.

## `questions.md` schema

Append-only. One entry per qid. Format:

```markdown
## q-YYYY-MM-DD-NNN
status: open | answered | dropped
slice: <slice id, e.g. 03-list-endpoint, or "spec" / "plan">
gate: advisory | validating | blocking | escalating
question: <one crisp sentence>
options: |                                    # ranked option set; recommended FIRST (see "Option set")
  1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
  2. <alternative> — <dimension-tagged rationale + trade-off>
  3. Something else — describe it
proposed: <the recommended option restated — the HITL default + the AFK auto-pick>
raised_at: <iso>
answered_at: <iso, when status flips off "open">
answer: <chosen option (or human's verbatim reply / drop reason)>
```

Rules:
- `NNN` is sequential per date: the next-available 3-digit integer.
- `status: open` is the only state `/rite-resolve` can mutate; `answered` and `dropped`
  are terminal.
- The file is the audit trail. Don't edit answered/dropped entries: open a new qid that
  references the old one (`supersedes: q-...-OLD`) and resolve it.

AFK never authorizes destructive Git. The native host permission/sandbox
boundary owns any such request and requires explicit user approval.

## `state.md` `Awaiting human` block

When a HITL gate fires, `/rite-build` writes:

```markdown
- Status: awaiting_human
- Next step: /rite-resolve <qid> "<answer>"

## Awaiting human
- qid: <q-...>
- gate: <gate>
- question: <crisp text>
- proposed: <agent's tentative answer>
- raised_at: <iso>
- blocking_slices: [<slice ids that cannot advance>]
```

`/rite-resolve` removes the block on success and flips `Status: running`.

## The resume verb: `/rite-resolve`

Three shapes:

```
/rite-resolve <qid> "<answer>"
/rite-resolve --drop <qid> ["<reason>"]
/rite-resolve --batch <path-to-yaml>
```

`/rite-resolve` is the canonical writer for **async** resume: a gate that already paused and
stopped the session (an AFK blocking/escalating/irreversible queue, or a HITL pause the human
walked away from), plus `--batch`. In an **interactive HITL** session the skill resolves the
`AskUserQuestion` pick **in place** (the same `questions.md` `answered` write + `state.md`
clear), so you don't type `/rite-resolve` for gaps you answer live. Both paths flip
`status: open → answered` and clear `Awaiting human` through the **same `devrites-engine state resolve` writer**:
one source of truth, two entry points (live pick vs typed verb). Use the writer;
manual edits are never destructive-operation authority.

When `/rite-resolve` does resume a stopped session, the skill does **not** auto-run the next
`/rite-build`. The user types the next command explicitly so:
- A `/rite-plan repair` can land first if the answer changes scope.
- The user sees the workspace state before resuming.
- Each verb has one mutation; chaining is a hidden side-effect.

## AFK exception for discretionary pauses

With `.devrites/AFK`, apply decision ownership first. Accepted in-scope technical
corrections return to the caller for repair/verification; severity blocks acceptance,
not authorized repair. Never silently accept a defect or broaden existing approval.

Discretionary ceilings apply only to
human-owned trade-off/risk decisions; accepted technical corrections do not enter
these branches. Compare Suggestion/Nit/FYI→advisory, Important→validating,
Critical→blocking against advisory < validating < blocking < escalating:

- Within the slice's `Gate:` and `.devrites/AFK` `allow_gates`: record advisory in
  `questions.md`, trade-off in `decisions.md`, proceed.
- Above ceiling, missing authority, or unapproved irreversible risk: record blocking
  question, `Status: awaiting_human`, fire `notify:`, STOP.

Recovery uses only the [retry contract](#retry-cap-no-progress-loops-and-self-resolve).

## Retry cap, no-progress loops, and self-resolve

Owns all phase recovery, including Doubt/Vet/serial/parallel Build. Resource,
access, safety and irreversible-action boundaries independently stop work.

- **Fingerprint the failed invariant, not the review round.** Identify owning
  invariant + defect mechanism + minimal reproduction/decisive failure signal.
  DEC/DRIFT IDs, line numbers, wording and splitting one cause create no new budget.
- **Cap no-progress retries:** three no-progress attempts per exact causal fingerprint
  across wright/recovery. Only a correction whose narrow recheck leaves that cause
  open or reproduces its failure consumes an attempt.
  Closing a prior finding with discriminating evidence is progress: resolve it,
  do not charge. A new Critical or Important finding gets its own budget only for
  a distinct evidenced cause. The same invariant with a different evidenced mechanism qualifies;
  renaming/splitting an unchanged cause does not. Suggestion/Nit/FYI cannot extend
  recovery. Initial discovery and expected test-first RED are not corrections;
  total rounds, re-batches and distinct-finding counts never exhaust recovery.
- **Rechecks are scoped; late findings never reopen the round.** *Late* applies only
  from a phase's **second** review pass (Vet 1b, Build's recheck after the first repair,
  Review step 7); an initial pass has no late findings and every supported finding is
  folded normally. A recheck packet
  names the open fingerprints, the correction diff, and the dependents of each
  changed clause/hunk; the owning reviewer's verdict covers exactly that. A finding
  on text/code unchanged since that role's previous pass is *late*: Critical with a
  concrete failure path → new fingerprint under this budget; otherwise recorded
  (Plan/Vet: `eng-review.md` `## Deferred findings`; Build: `touched-files.md`
  `## Review trail`) with severity · site · role · kind · round, and consumed by the
  next full-inventory gate (Vet `mechanism` → one `test-plan.md` proof row; Build →
  the `/rite-review` roster) — never by another repair round in this phase. Review
  closes every deferred row with a labeled verdict; Seal blocks on any without one.
  **Failing case:** round 4 finds three Important items on clauses untouched since
  round 1 and dispatches another repair.
- **Separate consumptive authority from recovery.** Spent one-shot authorization
  blocks another execution, not offline recovery. Retained evidence starts caller-owned
  diagnosis/correction immediately; the next consumptive execution needs fresh authority.
- **Persist existing records.** Put fingerprint, reproduction, correction,
  `progress: resolved|no-progress` and decisive result in `drift.md` and `evidence.md`.
  Cold resume derives counts there; no new counter file/command.
- **Reconcile terminal cursors.** Retained distinct Critical/Important evidence below
  cap resumes despite stale `Next step: none`; age/spent action authority is not exhaustion.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"unattended root reaches current admitted Workflow Artifact work","action":"invoke classifier; execute returned route without wright/slice charge","return":"saved lifecycle phase/action; no intermediate reply"} -->
- **Classify exhaustion:** human-owned contract/risk/access gaps open their gate.
  Otherwise preserve reproduction/dead ends, set `Status: blocked` and
  `Next step: none — technical recovery exhausted; requires new evidence or changed failure conditions`.
  Emit no phase command, retry-permission question or `/rite-resolve`.
  Reinvocation with unchanged cause stays blocked; it never resets the cap.
- **Resolve agent-owned questions first** from code/docs/decisions. Human communication
  is for blocked environment, handover, inaccessible critical information or missing
  credentials/permission—not writing, testing or review. Blocking/escalating/
  irreversible gates remain.

## What the rule does NOT cover

This contract is about **human pauses**. It does not weaken or replace:

- `/rite-prove`, `/rite-review`, `/rite-seal`: feature-scoped gates that always run.
- Spec Drift Guard: answer that changes acceptance criteria routes through
  `/rite-plan repair`, not silently into the slice.
- `evidence.md` writes: every AFK iteration still records evidence; un-recorded passes
  are unproven at `/rite-prove`.
- `/clear` / `/compact` advice: context-hygiene rules are unchanged.

AFK changes which decisions are automatic. It changes nothing else.

## Cross-reference

- Skill: `/rite-resolve` (`.pi/skills/rite-resolve/SKILL.md`).
- Workflow integration: `/rite-build` (`.pi/skills/rite-build/SKILL.md`),
  the readiness / HITL pre-flight stages before dispatch, and the DOUBT → PROVE
  (fail-on-red) → RECORD stages on the wright's return
  ([`one-slice-cycle.md`](../../../rite-build/reference/one-slice-cycle.md)).
- Render contract: `.pi/skills/rite-build/reference/checkpoint-protocol.md`.
- Loop discipline: `.pi/skills/rite-build/reference/afk-discipline.md`.
- Gate taxonomy: `.pi/skills/rite-define/reference/gates.md`.
- Schema: `.pi/skills/rite-spec/reference/state-workspace.md`.
- Doubt's AFK exception: `.pi/skills/devrites-doubt/SKILL.md` (AFK exception section).
