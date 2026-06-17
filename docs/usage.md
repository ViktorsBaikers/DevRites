# DevRites — usage

Worked workflows. Every feature starts with `/rite-spec` (it investigates,
writes the spec, and creates the workspace). Every later phase reads the
active workspace (`.devrites/ACTIVE` → `.devrites/work/<slug>/`) first; if
none exists it tells you to run `/rite-spec <feature>`.

- **Full command reference** → [`command-map.md`](command-map.md)
- **Flow diagrams** → [`flow.md`](flow.md)
- **Architecture rationale** → [`architecture.md`](architecture.md)

## The workspace

`/rite-spec` creates `.devrites/work/<slug>/` and writes the spec;
`/rite-define` adds the plan and tasks. All human-readable Markdown that
survives compaction and new sessions:

| File | Created by | Holds |
|---|---|---|
| `brief.md` | `/rite-spec` | one-line objective + definition of done |
| `spec.md` | `/rite-spec` | what to build + why, placement, acceptance, gaps/decisions |
| `references/` + `references.md` | `/rite-spec` | saved design refs — screenshots, Figma, video, links |
| `plan.md` | `/rite-define` | approach, dependency graph, checkpoints, rollback |
| `tasks.md` | `/rite-define` | ordered vertical slices, each tagged `Mode: AFK \| HITL` + `Gate` / `SLA` / `Checkpoint` when HITL |
| `state.md` | every phase | phase, status, active slice + slice mode, risk, next step (the cursor); plus `Awaiting human` block when paused (run mode is derived from `.devrites/AFK`, not stored here) |
| `questions.md` | every phase | append-only Q&A — qid, slice, gate, status (`open` / `answered` / `dropped`), proposed answer, raised/answered timestamps |
| `decisions.md` / `assumptions.md` | every phase | running logs |
| `drift.md` | Spec Drift Guard | drift events + resolutions |
| `touched-files.md` | `/rite-build` | what files this feature touched |
| `evidence.md` | `/rite-build`, `/rite-prove` | recorded commands + output |
| `browser-evidence.md` | `/rite-prove`, `/rite-polish` (UI) | screenshots, console, network, viewport runs |
| `design-brief.md` | `devrites-frontend-craft` | shape, states, design references match |
| `polish-report.md` | `/rite-polish` | Phase 1-4 findings + fixes |
| `review.md` | `/rite-review` | Spec + Standards axes, severity-labelled findings |
| `seal.md` | `/rite-seal` | GO/NO-GO verdict + acceptance walk + blockers |
| `ship.md` | `/rite-ship` | what shipped — commit SHA(s), branch, tag/PR, acceptance summary, follow-ups |

When `/rite-ship` closes the task it **archives** the whole workspace —
`.devrites/work/<slug>/` → `.devrites/archive/<slug>/` (every `.md` above is
preserved, never deleted) — and clears `.devrites/ACTIVE`. The audit trail lives
on under `.devrites/archive/<slug>/`.

Project-root sentinel (outside the workspace):

| File | Created by | Holds |
|---|---|---|
| `.devrites/AFK` | you (presence = AFK mode active) | optional YAML: `max_slices`, `notify`, `allow_gates`. Empty file = AFK with defaults. See [`pack/.claude/rules/afk-hitl.md`](../pack/.claude/rules/afk-hitl.md). |

The shape of this directory is also documented in
[`flow.md` § Workspace state model](flow.md#7-workspace-state-model).

## 1) Start a feature — spec then plan

```text
You: I want some kind of reporting thing for admins.

/rite-spec admin-usage-report
  → investigates deeply (codegraph/graphify): current behavior, placement,
    what it resolves
  → asks you one question at a time, each with options + a best guess,
    until gaps are closed
  → gathers any design references you give
  → writes spec.md (creates the workspace)

/rite-define
  → reads the approved spec
  → writes plan.md + vertical task slices + state
  → stops for confirmation
```

## 2) Normal feature — the build loop

```text
/rite-spec add-csv-export    # investigate → spec.md
/rite-define                 # spec → plan + vertical slices + state
/rite-build                  # slice 1 ("export endpoint returns CSV"); stops with evidence
/rite-build                  # slice 2 ("download button + states"); repeat for each slice
/rite-prove                  # ONCE all slices built: full tests + browser proof
/rite-polish                 # code polish (always) + UI normalize+polish (if UI)
/rite-review                 # feature-scoped multi-axis review (Spec + Standards in parallel)
/rite-seal                   # GO / NO-GO decision (no git) → on GO, points at /rite-ship
/rite-ship                   # type-GO + irreversible git ladder + close the task (archive + clear ACTIVE)
```

`/rite-build` never auto-advances — you decide when the next slice runs.
`/rite-seal` **decides**; `/rite-ship` **executes + closes**. To run the whole
sequence unattended, see `/rite-autocomplete` (§11).

## 3) Spec drift mid-build

```text
/rite-build
  → discovers the planned `User.export_token` column doesn't exist and adding
    it changes the data model
  → STOPS (Spec Drift Guard), records the drift in drift.md, classifies it as
    "codebase reality mismatch"
  → because it changes product behavior, asks you:

    I hit spec drift:
    - Spec/plan assumed: a per-user export token column
    - Code/evidence shows: tokens are issued per-session, no user column
    - Why it matters: changes the data model and the auth story
    Which direction should DevRites take?
    1. Keep the requirement, add the column + migration
    2. Adjust to per-session tokens (matches existing behavior)
    3. Split token work into a follow-up feature
    4. Custom

You: 2

/rite-plan repair    # updates spec/plan/tasks to per-session, marks drift resolved
/rite-build          # resumes on the corrected plan
```

## 4) UI feature with browser-harness

```text
/rite-spec settings-theme-toggle
  → you give a Figma link + a screenshot of the target toggle; rite-spec
    views them, saves the screenshot to references/, indexes references.md
  → writes spec.md

/rite-define
  → spec → slices; UI slice marked frontend-craft + browser-proof

/rite-build
  → UI slice → devrites-frontend-craft: register = product surface, **match
    the saved references**, shape the states (default/loading/empty/error/
    disabled), use existing tokens + components, meet CWV + WCAG 2.2, avoid
    anti-AI-slop

/rite-prove
  → devrites-browser-proof: browser-harness detected → new_tab(route),
    screenshot at 375 + 1280, exercise the toggle, check console clean,
    **compare to references/**, record to browser-evidence.md

/rite-polish
  → /rite-polish detects UI scope → reads reference/ui.md for normalize +
    quality bar; re-screenshots; appends to polish-report.md

/rite-review
/rite-seal
/rite-ship
```

If no browser tooling is available, proof is recorded as **pending (manual)**
with exact steps — the seal then weighs the UI risk.

## 5) Backend-only feature

```text
/rite-spec rate-limit-api    # investigate → spec.md (no UI)
/rite-define                 # spec → plan + slices
/rite-build                  # no UI → no frontend craft / browser proof
/rite-prove                  # targeted tests + build/typecheck; runtime check of the limiter
/rite-polish                 # reference/code.md only (no UI scope detected)
/rite-review                 # devrites-audit security fires (auth/abuse surface), measure-first perf
/rite-seal                   # checks rollback for any config/migration change → GO/NO-GO
/rite-ship                   # type-GO + commit/push/tag + close the task
```

## 6) UI-direction prompt — refinement modes

```text
/rite-polish bolder
  → reads reference/ui.md and runs Phase 4 with emphasis on hierarchy +
    weight contrast + decisive accent. Normalize (Phase 3) still runs first.

/rite-polish normalize-only
  → reads reference/ui.md and runs Phase 3 only (drift removal). Stops
    without the quality-bar pass.
```

Other modes: `quieter` · `distill` · `harden`.

## 7) Stuck in unfamiliar code

```text
/rite-zoom-out
  → returns a map: the area in 1 sentence (project's vocabulary), modules in
    scope, callers in / calls out, ADR / decisions.md entries touching the
    area, smallest sensible change-scope.
  → uses codegraph_context + codegraph_explore if available; falls back to
    Grep + Read otherwise.
```

## 8) Mid-flight handoff

```text
/rite-handoff
  → reads the chat, identifies anything chat-only that the workspace doesn't
    already capture (open questions, decisions discussed, assumptions made,
    drift status, ambiguous next action)
  → appends to questions.md / decisions.md / assumptions.md / drift.md /
    state.md as appropriate
  → writes handoff.md referencing each artifact by path
  → the next session (or a fresh agent) reads the workspace alone
```

Run before `/clear` if leaving for > a few hours.

## 9) HITL gate — pre-code pause and resume

```text
/rite-build
  → reads tasks.md slice 03; Mode: HITL, Gate: blocking
  → STOPS before writing any code:

    Slice 03 — list endpoint is HITL (blocking, SLA 15m).
    Checkpoint: Composite (user_id, created_at) index, or two single-col indexes?
    Proposed approach: composite — single read path, both columns used together
    in the most common filter; downside is rebuild cost on bulk updates.
    Decision needed before this slice can build.
    Resume: /rite-resolve q-2026-05-28-001 "<answer>"

  → appends q-...-001 to questions.md (status: open, gate: blocking)
  → writes `Awaiting human` block to state.md, sets Status: awaiting_human

You: /rite-resolve q-2026-05-28-001 "composite — single-col is fine for now"

/rite-resolve
  → flips q-...-001 status → answered (with answered_at + answer)
  → clears `Awaiting human`, sets Status: running
  → recommends /rite-build for slice 03

/rite-build
  → resumes with the answer captured in questions.md
```

If the answer changes acceptance criteria or scope, `/rite-resolve` recommends
`/rite-plan repair` first instead of an immediate `/rite-build`.

## 10) AFK overnight run

```text
# Drop the sentinel before bed. Keys are optional — empty file works.
cat > .devrites/AFK <<'EOF'
max_slices: 10
notify: "curl -d \"$DEVRITES_QID: $DEVRITES_QUESTION\" ntfy.sh/my-topic"
allow_gates: [advisory, validating]
EOF

/rite-build
  → reads sentinel, run mode = afk
  → slice 04 (AFK) → builds; devrites-doubt fires once, logs an advisory
    entry to questions.md and continues
  → slice 05 (AFK) → builds; targeted tests green, evidence.md updated
  → slice 06 (HITL, Gate: validating) → in allow_gates, so builds + queues
    a validating question; slice stays at `built (pending review)`
  → slice 07 (HITL, Gate: blocking) → ALWAYS pauses; writes Awaiting human,
    fires notify hook (your phone pings), STOPs
  → state.md `AFK slices remaining`: 10 → 7 (max_slices stays 10, read-only)

# In the morning:
/rite-status
  → Status: awaiting_human, 1 blocking q-..., 2 validating, 5 advisory
  → recommends /rite-resolve q-...

/rite-resolve q-... "<answer>"
/rite-build                    # continue
```

The loop refuses to mark a slice `built` if tests / types / lint go red — it
writes a blocking question and stops regardless of `allow_gates`. AFK never
silently accepts irreversible risk; see
[`pack/.claude/rules/afk-hitl.md`](../pack/.claude/rules/afk-hitl.md) for the
full list.

## 11) Full unattended lifecycle — `/rite-autocomplete`

```text
/rite-autocomplete "add CSV export for admins" --max-slices 8
  → vague idea → runs devrites-interview once, up front (the only interactive
    window), to ~95% confidence
  → arms AFK, then drives every phase in order: /rite-spec → /rite-define →
    /rite-build ×N → /rite-prove → /rite-polish → /rite-review → /rite-seal
    (→ /rite-ship too when --ship is set)
  → at each soft gate, picks the option the relevant specialist/reviewer
    favours and records the rationale in decisions.md (never silently)
  → seal returns GO → autocomplete STOPS (default) and hands off to /rite-ship

/rite-ship                     # human runs it → renders the type-GO prompt
You: GO                        # → commit · push · tag, then archive + clear ACTIVE
```

Add `--ship` (alias `--yolo`) to auto-confirm the final type-`GO` for a
zero-touch push — autocomplete then proceeds straight to `/rite-ship`. It still
pauses on hard irreversible-risk (auth / migration / public-API / red tests),
blocking / escalating gates, an open `gate: validating`, a NO-GO, exhausted
`max_slices`, or low confidence — writing `state.md` and surfacing *why* before
it stops. Args: `[idea] [--ship|--yolo] [--max-slices N]`.

## Checking in

- `/rite` — compact menu + suggested next command. **Does not** read state.
- `/rite-status` — detailed status: phase, **run mode (AFK / HITL)**, status
  (`running` / `awaiting_human` / `blocked` / `done`), next action, evidence,
  open questions broken down by gate, drift, risks, **handoff readiness**.
- `/rite-resolve <qid> "<answer>"` — answer / `--drop` / `--batch`-resolve an
  open question; the canonical writer for `status: open → answered` and the
  only thing that clears `Awaiting human`.

## Tips

- Commit `.devrites/` so the team and future sessions share feature state.
- **`.devrites/AFK` is per-developer, not per-repo** — gitignore it (or commit
  it deliberately if the team agrees on AFK defaults). The sentinel is
  read-only config: it toggles your local session mode and sets the initial
  `max_slices` budget; nothing else. The mutable remaining-slice count lives
  in `state.md` (`AFK slices remaining`), never in the sentinel.
- One feature active at a time (`ACTIVE`). To start or switch, run
  `/rite-spec <other>` (it creates/selects that workspace and writes its
  spec).
- **Recommended AFK progression**: HITL first to refine the prompt and plan,
  then drop the sentinel for the bulk stretch. Always cap iterations
  (`max_slices: 10` is a reasonable default).
- Let the specialists fire on their triggers — you don't invoke `devrites-*`
  directly. The exceptions are the three public utilities: `/rite-zoom-out`,
  `/rite-prototype`, `/rite-handoff`.
- The `devrites-` prefix is a namespace, not a privacy marker — whether a
  skill is public is the `user-invocable:` flag, documented for each in
  [`command-map.md`](command-map.md).
