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
      plan.md                     # how to build it (from /rite-define)
      tasks.md                    # ordered vertical slices (from /rite-define)
      state.md                    # phase, active slice, risk, next step
      questions.md                # asked questions + answers
      decisions.md                # decisions + rationale
      assumptions.md              # explicit assumptions still standing
      drift.md                    # spec/plan drift observations + resolutions
      touched-files.md            # files changed or intentionally inspected
      evidence.md                 # commands run + results
      browser-evidence.md         # screenshots, routes, console, viewport checks
      design-brief.md             # if UI is involved
      polish-report.md            # normalize+polish output
      review.md                   # review findings + decisions
      seal.md                     # final GO / NO-GO
```

## Rules
- **Slug**: kebab-case, derived from the objective (`add-csv-export`, not `feature1`).
- All artifacts are **human-readable Markdown** — they must survive context compaction
  and a fresh session. (`references/` also holds binary assets the human supplied.)
- **`/rite-spec` creates the workspace** and writes `spec.md` (+ `references/`,
  `references.md`, `brief.md`, `questions.md`, `decisions.md`, `assumptions.md`) and sets
  `ACTIVE`. **`/rite-define` reads `spec.md`** and adds `plan.md` + `tasks.md` and updates
  `state.md`. Other skills read the active workspace; none create a new one.
- Each phase **updates `state.md`** and the relevant evidence files.
- Don't create `evidence.md` / `browser-evidence.md` / `design-brief.md` /
  `polish-report.md` / `review.md` / `seal.md` until the producing phase runs — absence
  is meaningful (it means "not done yet").

## `state.md` template
```markdown
# State: <slug>

- Phase: spec | plan | build | prove | polish | review | seal | done
- Run mode: afk | hitl                              # mirrors .devrites/AFK presence (see below)
- Status: running | awaiting_human | blocked | done
- Active slice: <N — name> | none
- Slice mode: AFK | HITL | none
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
- [ ] Slice 1: <name> — <pending|built|proven|done>
- [ ] Slice 2: ...

## Log
- <date> <phase>: <one line of what happened>
```

## `.devrites/AFK` sentinel (AFK mode toggle)

Presence of the file `.devrites/AFK` puts DevRites into **AFK mode**: skills that would
normally pause for the user instead log to `questions.md` (when the gate severity allows
it — see [`pack/.claude/rules/afk-hitl.md`](../../../rules/afk-hitl.md)) and continue.
The file content is optional YAML configuring loop discipline:

```yaml
# .devrites/AFK — presence = AFK mode active. All keys optional.
max_slices: 10                       # /rite-build decrements per built slice; 0 → forced HITL stop
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition; gets qid/gate/slice as env
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle; everything else pauses
```

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
