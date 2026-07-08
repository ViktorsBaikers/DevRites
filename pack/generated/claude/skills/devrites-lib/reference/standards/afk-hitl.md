# AFK & HITL — the pause/resume contract

The rule layer for DevRites's two run modes. Every `rite-*` and `devrites-*` skill that
might pause for a human reads from this contract; `/rite-build`, `/rite-status`,
`/rite-resolve`, and `devrites-doubt` are the primary callers.

The contract is intentionally small: one sentinel, one queue, one verb.

## Run modes

- **HITL (default)** — human is present. At a gap/checkpoint the skill **asks inline** via
  the harness `AskUserQuestion` tool — a ranked **option set** (recommended first, each with
  dimension-tagged rationale; see [Option set](#option-set--how-every-gap-is-presented)). The
  human picks; the skill records the pick to `questions.md` (`answered`) + `decisions.md` and
  **continues in place — no `/rite-resolve` round-trip**. `/rite-resolve` is only for answering
  **async** (a pause that already stopped the session) or in **batch**.
- **AFK** — `.devrites/AFK` is present. For any gate AFK may auto-handle (severity in
  `allow_gates`), the skill **auto-picks the recommended option** (option 1 of the set), records
  it (`gate: advisory` + a `decisions.md` ADR), and continues unattended. Gates above the
  ceiling — and every irreversible-risk item — pause and queue a `questions.md` entry for
  `/rite-resolve`, **unless `allow_irreversible: true`** is set (see [Maximal
  autonomy](#irreversible-risk-list-always-pause)).

`.devrites/AFK` presence is authoritative for run mode; gate-deciding skills re-read the
sentinel at decision time (the shared preamble derives the mode from it). There is no
`state.md` run-mode field to drift out of sync.

## The sentinel — `.devrites/AFK`

Presence = AFK active. The file body is optional YAML:

```yaml
max_slices: 10                       # read-only INITIAL budget; seeds state.md `AFK slices remaining`
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition
allow_gates: [advisory, validating]  # gate severities AFK auto-handles (auto-picks the recommended option)
allow_irreversible: false            # DANGER, opt-in. true → auto-pick the recommended option even on
                                     # irreversible gates (drop-table, auth, public-API break, data-loss).
                                     # Lifts the safety floor; destructive changes ship unattended. Default false.
```

The file is **read-only config** — never rewritten in place. `max_slices` is the initial
budget; the mutable remaining count lives in `state.md` as `AFK slices remaining: <n>`,
seeded from `max_slices` on the first AFK build and decremented by `devrites-engine tick-afk` (which
exits non-zero at 0, forcing a HITL stop). The cap is enforced by the script, not prose.

Missing keys fall back to defaults:

| Key | Default | Meaning |
|---|---|---|
| `max_slices` | unlimited | a missing cap is unsafe; recommend setting one explicitly |
| `notify` | none | no notification fires |
| `allow_gates` | `[advisory]` | AFK auto-handles advisory only by default (auto-picks the recommended option) |
| `allow_irreversible` | `false` | when `true`, AFK auto-picks even irreversible-risk gates — the safety floor is lifted (see [Maximal autonomy](#irreversible-risk-list-always-pause)) |

To leave AFK, delete the file. The next skill invocation reverts to HITL.

## The four gates

Every `Mode: HITL` slice declares a `Gate:` and an `SLA:`. See
[`.claude/skills/rite-define/reference/gates.md`](../../../rite-define/reference/gates.md)
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

## Option set — how every gap is presented

Wherever a gap, checkpoint, or non-trivial decision surfaces (`/rite-spec`, `/rite-define`,
`/rite-build`, `/rite-temper`, `/rite-vet`, `devrites-doubt`, `devrites-interview`), present a
**ranked option set**, never a single bare guess:

- **2–4 concrete options**, the **recommended one first**, labelled `(Recommended)`.
- Each option carries a **one-line rationale tagged by the dimensions that matter** —
  `logic · infra · business · architecture` (add `security` / `UX` / `risk` when in scope).
  Name the trade-off, not just the choice.
- Always include an escape hatch (`Something else — I'll describe it`).
- The recommendation reflects what's best for *this* project (its conventions, stack, scale,
  domain) — not a generic default.

**HITL** renders the set via `AskUserQuestion` (recommended option first; rationale in each
option's description); the human's pick resolves the gate **in place**. **AFK** auto-picks
option 1 (the recommendation) for gates it may auto-handle. Either way the chosen option is
recorded verbatim and the **rejected options stay in `questions.md`** as the considered-alternatives
trail — the audit shows what was weighed, not just what was decided.

## Irreversible-risk list (always pause)

The following always invoke the checkpoint protocol, regardless of `Mode`, `Gate`, or
`allow_gates`:

- Destructive data migration (drop column, drop table, irreversible backfill).
- Auth / authz boundary change.
- Public API break (response shape, removed endpoint, changed status code semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.
- Red tests / types / lint on slice completion (fail-on-red).

When a pause clears and you proceed with a destructive migration, a removal, or a
public-API break, take the **safe path** the gate stopped you for: expand→contract,
prove the old path unused before removing it, and a rollback for every destructive step
([`deprecation.md`](deprecation.md)). The gate exists to make you do it right, not to
abandon the work.

By default, AFK widens what's *automatic*; it never widens what's *irreversible*.

**Maximal autonomy (`allow_irreversible: true` — opt-in, dangerous).** Setting this key in
`.devrites/AFK` lifts the floor: AFK then **auto-picks the recommended option on irreversible
gates too, with no pause**. This ships destructive migrations / auth changes / public-API
breaks / data-loss paths **unattended, with zero human review** — recommended *only* on a
throwaway or sandboxed target you can roll back wholesale. Default is `false`; a missing key
keeps the floor. The floor is the deliberate safety default — `allow_irreversible` is the user
pulling the trigger themselves, not something a stray sentinel can do silently.

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

`/rite-resolve` is the canonical writer for **async** resume — a gate that already paused and
stopped the session (an AFK blocking/escalating/irreversible queue, or a HITL pause the human
walked away from), plus `--batch`. In an **interactive HITL** session the skill resolves the
`AskUserQuestion` pick **in place** (the same `questions.md` `answered` write + `state.md`
clear), so you don't type `/rite-resolve` for gaps you answer live. Both paths flip
`status: open → answered` and clear `Awaiting human` through the **same `devrites-engine resolve` writer** —
one source of truth, two entry points (live pick vs typed verb). Manual edits work but the
script is the contract — use it.

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

The loop limits of the calling skill still apply — after the limit, the unresolved
doubt becomes a blocking question regardless of AFK config.

## Retry cap, stuck loops, and self-resolve

- **Cap retries.** At most **3 attempts** on the same failing check (test, lint, type, build).
  On the third failure, stop guessing and convert it to a `gate: blocking` question — a fourth
  identical attempt is thrash, not progress.
- **Stuck loops pause even in AFK.** A detected loop — the same action repeating, or an
  action↔error ping-pong — pauses regardless of `allow_gates` (`devrites-engine stuck`), the same standing as
  the irreversible-risk list. AFK widens what's automatic, never what's looping.
- **Bias to self-resolve.** Before raising a question, try to answer it from the code, the docs,
  or `decisions.md`. Communicate only for a blocked environment, a deliverable to hand over,
  critical info you genuinely can't access, or a credential / permission you lack. This narrows
  needless pauses and never weakens the blocking / escalating / irreversible gates.
- **Human time is for human-only work.** A `human_intervention` pause is for what the agent
  literally cannot do (create a cloud account, click a console button) — never for writing code,
  writing tests, or reviewing. Punting the agent's own job to the human is not a valid gate.

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
