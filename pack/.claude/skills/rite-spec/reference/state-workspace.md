# `.devrites/` state workspace — canonical format

The shared state contract for **all** DevRites skills. Every phase reads the active
workspace first; if none exists, it stops and tells the user to run `/rite-spec <feature>`
(which creates it).

## Layout
```
.devrites/
  README.md                       # what this dir is (seeded at install)
  ACTIVE                          # single line: active feature slug, or empty
  work/
    <feature-slug>/
      brief.md                    # short user-facing objective
      spec.md                     # what to build + why (from /rite-spec)
      references/                 # saved design/reference assets (screenshots, exports, video)
      references.md               # index of design references (saved files + links)
      strategy.md                 # strategic review: scope mode + pre-mortem + dimension scores (from /rite-temper; optional — always invoked, significance-gated, in /rite-autocomplete)
      plan.md                     # how to build it (from /rite-define)
      tasks.md                    # ordered vertical slices (from /rite-define)
      eng-review.md               # engineering plan review: scope challenge + axis findings + failure modes + parallelization (from /rite-vet — run on every plan; depth scales light/full, never skipped; always in /rite-autocomplete)
      test-plan.md                # build-readable coverage target: coverage diagram + per-gap test requirements + acceptance→test map (from /rite-vet; read by /rite-build + /rite-prove)
      state.md                    # phase, active slice, risk, next step
      questions.md                # asked questions + answers
      decisions.md                # decisions + rationale
      assumptions.md              # explicit assumptions still standing
      drift.md                    # spec/plan drift observations + resolutions
      touched-files.md            # files changed or intentionally inspected
      evidence.md                 # commands run + results
      browser-evidence.md         # screenshots, routes, console, viewport checks
      design-brief.md             # UX/UI contract — if UI involved (from /rite-spec via devrites-ux-shape; refined per slice in /rite-build)
      polish-report.md            # normalize+polish output
      review.md                   # review findings + decisions
      seal.md                     # final GO / NO-GO decision (from /rite-seal)
      ship.md                     # ship record: commit/tag, acceptance, follow-ups (from /rite-ship)
      handoff.md                  # human/next-agent-facing handoff summary (overwritten each handoff)
  archive/
    <feature-slug>/               # closed features (moved here by /rite-ship close-out; all .md preserved)
```

## Rules
- **Slug**: kebab-case, derived from the objective (`add-csv-export`, not `feature1`).
- All artifacts are **human-readable Markdown** — they must survive context compaction
  and a fresh session. (`references/` also holds binary assets the human supplied.)
- **`/rite-spec` creates the workspace** and writes `spec.md` (+ `references/`,
  `references.md`, `brief.md`, `questions.md`, `decisions.md`, `assumptions.md`; **plus
  `design-brief.md` when the feature touches UI**, via `devrites-ux-shape`) and sets
  `ACTIVE`. **`/rite-define` reads `spec.md`** and adds `plan.md` + `tasks.md` and updates
  `state.md`. Other skills read the active workspace; none create a new one.
- Each phase **updates `state.md`** and the relevant evidence files.
- Don't create `strategy.md` / `eng-review.md` / `test-plan.md` / `evidence.md` /
  `browser-evidence.md` / `design-brief.md` / `polish-report.md` / `review.md` / `seal.md` /
  `ship.md` / `handoff.md` until the producing phase runs — absence is meaningful (it means
  "not done yet"). `strategy.md` is written by `/rite-temper` (which may also edit `spec.md` /
  `decisions.md` / `assumptions.md` via the Spec Drift Guard); its absence means the spec was
  planned without a strategic review. `eng-review.md` + `test-plan.md` are written by `/rite-vet`
  (which also hardens `plan.md` / `tasks.md`, and routes acceptance-changing deltas via the Spec
  Drift Guard); their absence means the plan was built without an engineering review. `handoff.md`
  is written by `/rite-handoff` and overwritten each handoff (latest snapshot, not a log).
- **`/rite-ship` closes the feature**: it writes `ship.md`, sets `state.md` phase `done`,
  then archives `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` (every `.md`
  preserved) and clears `.devrites/ACTIVE`. Closing **relocates** the audit trail; it
  never deletes it. Re-open by moving the dir back and re-pointing `ACTIVE`.

## `state.md` template
```markdown
# State: <slug>

- Phase: spec | temper | plan | vet | build | prove | polish | review | seal | ship | done   # `temper` (pre-plan) + `vet` (post-plan, pre-build) only when those optional reviews ran; spec→plan→build directly otherwise
- Status: running | awaiting_human | blocked | done
- Active slice: <N — name> | none
- Slice mode: AFK | HITL | none
- Spec gate: passed <iso> | none                    # optional — set when the spec readiness gate passes
- Plan approved: <iso> | none                       # optional — /rite-define sets it when the human confirms the plan
- AFK slices remaining: <n> | none                  # mutable counter; initialized from .devrites/AFK max_slices on first AFK build
- Risk: <highest current risk> | none
- Next step: <single recommended command + why>

## Awaiting human   (only when Status == awaiting_human)
- qid: <q-YYYY-MM-DD-NNN>
- gate: advisory | validating | blocking | escalating
- question: <crisp text>
- proposed: <agent's tentative answer>
- raised_at: <iso>
- blocking_slices: [<slice ids that cannot advance until answered>]

## Slice progress
- [ ] Slice 1: <name> — <pending|built>
- [ ] Slice 2: ...

## Log
- <date> <phase>: <one line of what happened>
```

A slice is only ever `pending` or `built`. Acceptance proof lives at the **feature**
level in `evidence.md` (recorded by `/rite-prove`), not per slice — there is no per-slice
`proven` / `done` state.

## `.devrites/AFK` sentinel (AFK mode toggle)

Presence of the file `.devrites/AFK` puts DevRites into **AFK mode**: skills that would
normally pause for the user instead log to `questions.md` (when the gate severity allows
it — see [`.claude/rules/afk-hitl.md`](../../../rules/afk-hitl.md)) and continue.
`.devrites/AFK` presence is the **single source of truth** for run mode (the shared preamble
derives it); skills re-read the sentinel at decision time rather than trusting a mirrored
`state.md` field. The file content is optional YAML configuring loop discipline:

```yaml
# .devrites/AFK — presence = AFK mode active. All keys optional.
max_slices: 10                       # read-only INITIAL budget; the mutable remaining count lives in state.md
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition; see afk-discipline.md for the env table
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle; everything else pauses
```

`max_slices` is **read-only config** — never rewritten in place. The mutable remaining
count lives in `state.md` as `AFK slices remaining: <n>`, initialized from `max_slices` on
the first AFK build and decremented per built slice (the cap is enforced by
`tick-afk.sh`, not by prose).

Absent or empty file = AFK active with defaults (`max_slices: unlimited`, no `notify`,
`allow_gates: [advisory]`). **Destructive migrations, auth/authz boundary changes, and
public-API breaks always pause regardless of `allow_gates`** — AFK never silently accepts
irreversible risk.

## `questions.md` entry format

`questions.md` is append-only. Each entry:

```markdown
## q-2026-05-28-001
status: open | answered | dropped
slice: 03-list-endpoint              # which slice raised it (or "spec" / "plan")
gate: advisory | validating | blocking | escalating
question: <one crisp sentence the human can answer>
proposed: <agent's tentative answer, or "none">
raised_at: 2026-05-28T17:30:00Z
answered_at: <iso when status flips to answered>
answer: <human's reply, verbatim>
```

`/rite-resolve <qid> "<answer>"` is the canonical mutator — manual edits work but the
skill keeps `state.md` and `questions.md` consistent.

## `brief.md` template
```markdown
# <Feature>
Objective: <one sentence — what and for whom>
Why now: <one line>
Definition of done: <one line>
```

`decisions.md`, `assumptions.md`, `questions.md`, `drift.md`, `touched-files.md` are
append-only running logs — date-stamped bullet entries.
