# AFK & HITL — the pause/resume contract

The rule layer for DevRites's two run modes. Every `rite-*` and `devrites-*` skill that
might pause for a human reads from this contract; `/rite-build`, `/rite-status`,
`/rite-resolve`, and `devrites-doubt` are the primary callers.

The contract is intentionally small: one sentinel, one queue, one verb.

## Run modes

- **HITL (default)** — human is present. Slices marked `Mode: HITL` pause at a
  checkpoint **before** writing code; the workflow resumes on `/rite-resolve`.
- **AFK** — `.devrites/AFK` is present. Slices marked `Mode: AFK` run unattended;
  discretionary pauses (e.g. `devrites-doubt` findings) downgrade to advisory entries
  in `questions.md` instead of blocking, subject to the gate ceiling.

`.devrites/AFK` presence is authoritative for run mode; gate-deciding skills re-read the
sentinel at decision time (the shared preamble derives the mode from it). There is no
`state.md` run-mode field to drift out of sync.

## The sentinel — `.devrites/AFK`

Presence = AFK active. The file body is optional YAML:

```yaml
max_slices: 10                       # read-only INITIAL budget; seeds state.md `AFK slices remaining`
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle
```

The file is **read-only config** — never rewritten in place. `max_slices` is the initial
budget; the mutable remaining count lives in `state.md` as `AFK slices remaining: <n>`,
seeded from `max_slices` on the first AFK build and decremented by `tick-afk.sh` (which
exits non-zero at 0, forcing a HITL stop). The cap is enforced by the script, not prose.

Missing keys fall back to defaults:

| Key | Default | Meaning |
|---|---|---|
| `max_slices` | unlimited | a missing cap is unsafe; recommend setting one explicitly |
| `notify` | none | no notification fires |
| `allow_gates` | `[advisory]` | AFK auto-handles advisory only by default |

To leave AFK, delete the file. The next skill invocation reverts to HITL.

## The four gates

Every `Mode: HITL` slice declares a `Gate:` and an `SLA:`. See
[`.claude/skills/rite-define/reference/gates.md`](../skills/rite-define/reference/gates.md)
for the full taxonomy. Summary:

| Gate | Stakes | Pause? | SLA | AFK auto-handle when in `allow_gates`? |
|---|---|---|---|---|
| advisory | low | no | none | yes (log + proceed) |
| validating | medium | async | 4h | yes (build + queue, no merge until resolved) |
| blocking | high | sync | 15m | **no** — always pauses |
| escalating | novel pattern | sync to specialist | 24h | **no** — always pauses |

`blocking` and `escalating` always pause regardless of `allow_gates`. They are the
"AFK never silently accepts" guarantees in protocol form.

An open `gate: validating` entry is **merge-blocking by definition**: at `/rite-seal` any
`questions.md` entry with `gate: validating` and `status: open` is a NO-GO, regardless of
its behavior impact. A slice marked `built (pending review)` is **not done** until that
validating gate resolves.

## Irreversible-risk list (always pause)

The following always invoke the checkpoint protocol, regardless of `Mode`, `Gate`, or
`allow_gates`:

- Destructive data migration (drop column, drop table, irreversible backfill).
- Auth / authz boundary change.
- Public API break (response shape, removed endpoint, changed status code semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.
- Red tests / types / lint on slice completion (fail-on-red).

AFK widens what's *automatic*; it never widens what's *irreversible*.

## `questions.md` schema

Append-only. One entry per qid. Format:

```markdown
## q-YYYY-MM-DD-NNN
status: open | answered | dropped
slice: <slice id, e.g. 03-list-endpoint, or "spec" / "plan">
gate: advisory | validating | blocking | escalating
question: <one crisp sentence>
proposed: <agent's tentative answer, or "none">
raised_at: <iso>
answered_at: <iso, when status flips off "open">
answer: <human's reply or drop reason, verbatim>
```

Rules:
- `NNN` is sequential per date — the next-available 3-digit integer.
- `status: open` is the only state `/rite-resolve` can mutate; `answered` and `dropped`
  are terminal.
- The file is the audit trail. Don't edit answered/dropped entries — open a new qid that
  references the old one (`supersedes: q-...-OLD`) and resolve it.

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

## The resume verb — `/rite-resolve`

Three shapes:

```
/rite-resolve <qid> "<answer>"
/rite-resolve --drop <qid> ["<reason>"]
/rite-resolve --batch <path-to-yaml>
```

`/rite-resolve` is the **only** canonical writer that flips `status: open → answered`
and clears `state.md`'s `Awaiting human`. Manual edits work but the skill is the
contract — use it.

The skill does **not** auto-run the next `/rite-build`. The user types the next command
explicitly so:
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

The loop limits of the calling skill still apply — after the limit, the unresolved
doubt becomes a blocking question regardless of AFK config.

## What the rule does NOT cover

This contract is about **human pauses**. It does not weaken or replace:

- `/rite-prove`, `/rite-review`, `/rite-seal` — feature-scoped gates that always run.
- Spec Drift Guard — answer that changes acceptance criteria routes through
  `/rite-plan repair`, not silently into the slice.
- `evidence.md` writes — every AFK iteration still records evidence; un-recorded passes
  are unproven at `/rite-prove`.
- `/clear` / `/compact` advice — context-hygiene rules are unchanged.

AFK shifts the boundary between automatic and "ask"; nothing else.

## Cross-reference

- Skill: `/rite-resolve` (`.claude/skills/rite-resolve/SKILL.md`).
- Workflow integration: `/rite-build` (`.claude/skills/rite-build/SKILL.md`),
  workflow steps 0 + 2a (readiness / HITL pre-flight) and steps 4–6 (doubt / fail-on-red /
  record) on the wright's return.
- Render contract: `.claude/skills/rite-build/reference/checkpoint-protocol.md`.
- Loop discipline: `.claude/skills/rite-build/reference/afk-discipline.md`.
- Gate taxonomy: `.claude/skills/rite-define/reference/gates.md`.
- Schema: `.claude/skills/rite-spec/reference/state-workspace.md`.
- Doubt's AFK exception: `.claude/skills/devrites-doubt/SKILL.md` (AFK exception section).
