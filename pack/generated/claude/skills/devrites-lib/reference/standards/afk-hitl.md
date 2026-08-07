# AFK & HITL: the pause/resume contract

Defines two modes for `rite-*`/`devrites-*` skills that may pause; main
callers are `/rite-build`, `/rite-status`, `/rite-resolve`, and
`devrites-doubt`.

The contract uses one sentinel, one queue, and one resume verb.

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
max_slices: 10                       # read-only INITIAL budget; seeds state.md `AFK slices remaining`
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition
allow_gates: [advisory, validating]  # gate severities AFK auto-handles (auto-picks the recommended option)
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

Missing keys fall back to defaults:

| Key | Default | Meaning |
|---|---|---|
| `max_slices` | unlimited | a missing cap is unsafe; recommend setting one explicitly |
| `notify` | none | no notification fires |
| `allow_gates` | `[advisory]` | AFK auto-handles advisory only by default (auto-picks the recommended option) |

To leave AFK, delete the file. The next skill invocation reverts to HITL.

## The four gates

Every `Mode: HITL` slice declares a `Gate:` and an `SLA:`. See
[`.claude/skills/rite-define/reference/gates.md`](../../../rite-define/reference/gates.md)
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
- Always include an escape hatch (`Something else — I'll describe it`).
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

`devrites-doubt` and similar skills that "ask the user" follow this rule when
`.devrites/AFK` exists:

- Finding severity ≤ slice's gate ceiling (slice's `Gate:` plus `.devrites/AFK`
  `allow_gates`) → log to `questions.md` as `gate: advisory`, record the trade-off in
  `decisions.md`, proceed.
- Finding severity > gate ceiling, OR finding touches the irreversible-risk list →
  log to `questions.md` as `gate: blocking`, set `state.md` `Status: awaiting_human`,
  fire `notify:`, STOP.

The loop limits of the calling skill still apply. At the limit, classify the unresolved
finding by the decision-ownership rule above: human-owned uncertainty becomes a blocking
question; an objective technical finding becomes a recorded blocker with its required
changes, not a request for permission to retry.

## Retry cap, no-progress loops, and self-resolve

- **Fingerprint the failed invariant, not the review round.** An exact causal
  fingerprint is the owning gate/invariant plus its minimal reproduction and
  decisive failure signal. A new DEC/DRIFT number, changed line number, reviewer
  wording, or splitting one finding does not create a new fingerprint.
- **Cap no-progress retries:** three no-progress attempts per exact causal fingerprint
  across wright and recovery is a hard cap. Only a repair whose narrow recheck
  leaves that fingerprint open or reproduces the same decisive failure consumes
  an attempt. Closing a prior finding with discriminating evidence is progress:
  mark that fingerprint resolved and do not charge it as no-progress. A genuinely
  new Critical or Important finding with a different failed invariant and exact
  evidence gets its own fingerprint and budget; Suggestion, Nit, FYI, or renamed
  evidence cannot open, reset, or extend recovery.
- **Persist accounting in existing records.** Record each fingerprint,
  reproduction, attempted correction, `progress: resolved|no-progress`, and
  decisive result in `drift.md` and `evidence.md`. Cold resume derives the count
  from those records. There is no recovery counter file or command.
- **Classify exhaustion:** human-owned contract/risk/access gaps open their gate. Otherwise
  preserve reproduction/dead ends, set `Status: blocked` and `Next step: none — technical recovery exhausted for <causal fingerprint>; requires new evidence or changed failure conditions`.
  Do not emit `/rite-plan unblock`, another phase command, a question, or
  `/rite-resolve`. Reinvocation with the unchanged fingerprint remains blocked
  and never resets the retry cap.
- **Reassess no-progress loops:** After repeated attempts without progress, stop and
  reassess the failure and approach rather than continuing blindly.
- **Resolve agent-owned questions first.** Before raising a question, try to answer it from the code, the docs,
  or `decisions.md`. Communicate only for a blocked environment, a deliverable to hand over,
  critical info you genuinely can't access, or a credential / permission you lack. This narrows
  needless pauses and never weakens the blocking / escalating / irreversible gates.
- **Use human intervention only for human-owned work.** A `human_intervention` pause is for what the agent
  literally cannot do (create a cloud account, click a console button): never for writing code,
  writing tests, or reviewing. The agent's own work is not a valid human gate.

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

- Skill: `/rite-resolve` (`.claude/skills/rite-resolve/SKILL.md`).
- Workflow integration: `/rite-build` (`.claude/skills/rite-build/SKILL.md`),
  workflow steps 0 + 2a (readiness / HITL pre-flight) and steps 4-6 (doubt / fail-on-red /
  record) on the wright's return.
- Render contract: `.claude/skills/rite-build/reference/checkpoint-protocol.md`.
- Loop discipline: `.claude/skills/rite-build/reference/afk-discipline.md`.
- Gate taxonomy: `.claude/skills/rite-define/reference/gates.md`.
- Schema: `.claude/skills/rite-spec/reference/state-workspace.md`.
- Doubt's AFK exception: `.claude/skills/devrites-doubt/SKILL.md` (AFK exception section).
