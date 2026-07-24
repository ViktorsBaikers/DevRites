# Using DevRites

These examples cover the common DevRites workflows. Start a new feature with
`/rite-spec` and bring an existing codebase in with `/rite-adopt`.
`/rite-quick` and `/rite-frame` handle bounded work outside the full feature
lifecycle. Workspace phases first read the active workspace from
`.devrites/ACTIVE` and `.devrites/work/<slug>/`. If none exists, they report the
command that can create or select one. `/rite-upgrade [slug]` is the
conditional maintenance route for an active workspace planned under older
DevRites rules.

- **Full command reference** → [`command-map.md`](command-map.md)
- **Flow diagrams** → [`flow.md`](flow.md)
- **Architecture rationale** → [`architecture.md`](architecture.md)
- **Workspace schema** → [`engine/workspace-schema.md`](engine/workspace-schema.md)

## The workspace

`/rite-spec` creates `.devrites/work/<slug>/` and writes the spec.
`/rite-clarify` adds decision coverage, `/rite-define` adds the plan, and
`/rite-vet` adds implementation readiness. These human-readable files preserve
the work across compaction and new sessions:

| File | Created by | Holds |
|---|---|---|
| `README.md` / `index.md` / `feature.md` | `/rite-spec` | compact workspace map: phase, status, next action, artifact map, read-next table, gates |
| `brief.md` | `/rite-spec` | one-line objective + definition of done |
| `spec.md` | `/rite-spec` | product WHAT/WHY, requirements, acceptance, boundaries, measurable success |
| `decision-coverage.md` | `/rite-clarify` | topology-first coverage plus semantic `CLEAR` verdict bound to all decision inputs and the current readiness-artifact contract |
| `architecture.md` | `/rite-define` | owning module/layer, integration points, data/API/events, dependencies, risks |
| `flows.md` | `/rite-spec` or `/rite-define` | optional Mermaid diagrams when sequence/state/data flow clarifies behavior |
| `references/` + `references.md` | `/rite-spec` | saved design refs: screenshots, Figma, video, links |
| `strategy.md` | `/rite-temper` | strategic spec review (optional): scope mode, pre-mortem, dimension scores |
| `plan.md` | `/rite-define` | approach, dependency graph, checkpoints, rollback |
| `tasks.md` | `/rite-define` | ordered `SLICE-###` vertical slices, each mapped to `AC-###` and tagged `Mode: AFK \| HITL` + gate fields |
| `traceability.md` | `/rite-define` | AC/REQ → slices → tests/proofs → evidence → touched files matrix |
| `eng-review.md` | `/rite-vet` | mandatory engineering review with semantic, input-digest-bound readiness verdict and readiness-artifact contract |
| `test-plan.md` | `/rite-vet` | build-readable coverage target and readiness-artifact contract: coverage diagram, per-gap test requirements, acceptance→test map (read by `/rite-build` + `/rite-prove`) |
| `state.md` | every phase | working cursor: phase, status, next action, slice, AFK budget, durable clarification return fields, and `Awaiting human` only when paused |
| `recovery-attempts.jsonl` | technical recovery | durable three-failure budget per root-cause fingerprint |
| `.wright-allowlist` | `/rite-build` root | exact normalized source/test paths the sole wright may change |
| `status.md` | every phase | compatibility alias for the canonical `state.md` cursor |
| `questions.md` | every phase | append-only Q&A: qid, slice, gate, status (`open` / `answered` / `dropped`), proposed answer, raised/answered timestamps |
| `decisions.md` / `assumptions.md` | every phase | running logs |
| `drift.md` | Spec Drift Guard | drift events + resolutions |
| `touched-files.md` | `/rite-build` | what files this feature touched |
| `evidence.md` | `/rite-build`, `/rite-prove` | canonical `EVID-###` command/action proof |
| `proof.md` | `/rite-build`, `/rite-prove` | transition alias for `evidence.md` |
| `browser-evidence.md` | `/rite-prove`, `/rite-polish` (UI) | screenshots, console, network, viewport runs |
| `design-brief.md` | `devrites-frontend-craft` | shape, states, design references match |
| `polish-report.md` | `/rite-polish` | Phase 1-4 findings + fixes |
| `review.md` | `/rite-review` | Spec + Standards axes, severity-labelled findings |
| `seal.md` | `/rite-seal` | GO/NO-GO verdict + acceptance walk + blockers |
| `ship.md` | `/rite-ship` | what shipped: commit SHA(s), branch, tag/PR, acceptance summary, follow-ups |

When `/rite-ship` closes the task, it archives the whole workspace from
`.devrites/work/<slug>/` to `.devrites/archive/<slug>/` and clears
`.devrites/ACTIVE`. Every Markdown file remains in
`.devrites/archive/<slug>/` as an audit trail.

Backward compatibility: older `.devrites/features/<slug>/` workspaces remain
readable; migration should add the canonical `.devrites/work` shape without
deleting the old files. `feature.md`/`index.md` can still act as the workspace
map, `status.md` as the cursor alias for `state.md`, and `proof.md` as the proof
alias for `evidence.md`. Workspace maps use schema v2; migration upgrades
compatible declarations but never fabricates clarification, readiness, or proof
evidence. That structural schema is separate from the semantic planning
contract. `/rite-upgrade` reconciles active unfinished planning with
`devrites.readiness-artifacts.v2`; it does not rewrite current, completed, or
archived workspaces.

Project-root sentinel (outside the workspace):

| File | Created by | Holds |
|---|---|---|
| `.devrites/AFK` | you (presence = AFK mode active) | optional YAML: `max_slices`, `notify`, `allow_gates`. Empty file = AFK with defaults. See [`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md). |
| `.devrites/CHECKPOINT` | you or `/rite-autocomplete` (presence = checkpoint mode) | empty sentinel. When set, `/rite-build` commits each proven slice local-only as `WIP(<slug>)` so a crash mid-build loses neither source nor reasoning; `/rite-ship` collapses the WIP run into the one feature commit. See [`pack/.claude/skills/rite-build/reference/checkpoint.md`](../pack/.claude/skills/rite-build/reference/checkpoint.md). |

The shape of this directory is also documented in
[`flow.md` § Workspace state model](flow.md#7-workspace-state-model).

## 1) Start a feature: spec then plan

```text
You: I want some kind of reporting thing for admins.

/rite-spec admin-usage-report
  → investigates deeply (codegraph/graphify): current behavior, placement,
    what it resolves
  → asks you one question at a time, each with options + a best guess,
    until gaps are closed
  → gathers any design references you give
  → writes spec.md (creates the workspace)

/rite-clarify
  → scans the full topology; asks zero questions when the contract is complete
  → writes fresh Decision coverage: CLEAR

/rite-define
  → reads the approved spec
  → writes plan.md + vertical task slices + state
  → stops for confirmation

/rite-vet
  → reviews every defined plan; uses a light or full pass based on stakes
  → writes eng-review.md + test-plan.md before build
```

## 2) Normal feature: the build loop

```text
/rite-spec add-csv-export    # investigate → spec.md
/rite-clarify                # topology scan → Decision coverage: CLEAR
/rite-define                 # clarified spec → plan + vertical slices + state
/rite-vet                    # mandatory plan review; light or full based on stakes
/rite-build                  # slice 1 ("export endpoint returns CSV"); stops with evidence
/rite-build                  # slice 2 ("download button + states"); repeat for each slice
/rite-prove                  # ONCE all slices built: full tests + browser proof
/rite-polish                 # code polish (always) + UI normalize+polish (if UI)
/rite-review                 # feature-scoped multi-axis review (Spec + Standards in parallel)
/rite-seal                   # GO / NO-GO decision (no git) → on GO, points at /rite-ship
/rite-ship                   # type-GO + irreversible git ladder + close the task (archive + clear ACTIVE)
```

`/rite-build` never starts the next slice automatically; you decide when it
runs. For each slice, the root writes an exact `.wright-allowlist`. The snapshot,
reconciliation check, test/package integrity, and close steps use the same
original baseline. A retry snapshot refreshes only the dispatch boundary.
`/rite-seal` **decides**; `/rite-ship` **executes + closes**. To run the whole
sequence unattended, see `/rite-autocomplete` (§11).

## 2a) Continue a workspace planned under older rules

Run the upgrade after updating DevRites when build readiness reports code `8`,
or invoke it directly for an older active workspace:

```text
/rite-upgrade ark-panda-redesign
  → a fresh read-only devrites-upgrade-planner compares the active unfinished
    planning surface with devrites.readiness-artifacts.v2
  → completed source, slices, decisions, and evidence stay unchanged
  → stale old-engine proof recipes and machine-local command wrappers leave the
    active canonical plan
  → current coverage and engineering-readiness gates run again
  → an already-current workspace that passes readiness is a no-op; completed
    workspaces and archives are also left alone
```

This is not the same as `devrites-engine update`, which refreshes the installed
engine and pack, or `devrites-engine migrate`, which normalizes workspace layout
and structural state schema.

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
/rite-vet            # rechecks the repaired plan before build
```

## 4) UI feature with Playwright MCP

```text
/rite-spec settings-theme-toggle
  → you give a Figma link + a screenshot of the target toggle; rite-spec
    views them, saves the screenshot to references/, indexes references.md
  → writes spec.md

/rite-define
  → spec → slices; UI slice marked frontend-craft + browser-proof

/rite-vet
  → hardens the UI plan + acceptance-to-test coverage before code

/rite-build
  → UI slice → devrites-frontend-craft: register = product surface, **match
    the saved references**, shape the states (default/loading/empty/error/
    disabled), use existing tokens + components, meet CWV + WCAG 2.2, avoid
    anti-AI-slop

/rite-prove
  → devrites-browser-proof: Playwright MCP detected → browser_navigate(route),
    screenshot at 375 + 1280, exercise the toggle, check console clean,
    **compare to references/**, record to browser-evidence.md

/rite-polish
  → /rite-polish detects UI scope → reads reference/ui.md for normalize +
    quality bar; re-screenshots; appends to polish-report.md

/rite-review
/rite-seal
/rite-ship
```

If browser tooling is unavailable, the proof records exact manual steps and a
status of **pending (manual)**. The seal then accounts for the remaining UI
risk.

## 5) Backend-only feature

```text
/rite-spec rate-limit-api    # investigate → spec.md (no UI)
/rite-clarify                # topology scan; zero questions when already complete
/rite-define                 # clarified spec → plan + slices
/rite-vet                    # mandatory plan review; checks architecture, tests, perf
/rite-build                  # no UI → no frontend craft / browser proof
/rite-prove                  # targeted tests + build/typecheck; runtime check of the limiter
/rite-polish                 # reference/code.md only (no UI scope detected)
/rite-review                 # devrites-audit security fires (auth/abuse surface), measure-first perf
/rite-seal                   # checks rollback for any config/migration change → GO/NO-GO
/rite-ship                   # type-GO + commit/push/tag + close the task
```

## 6) UI-direction prompt: refinement modes

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

Run this before `/clear` or a break of more than a few hours.

## 9) HITL gate: pre-code pause and resume

```text
/rite-build
  → reads tasks.md slice 03; Mode: HITL, Gate: blocking
  → STOPS before writing any code:

    SLICE-003: export policy is HITL (blocking, SLA 15m).
    Checkpoint: Should deactivated users appear in the admin export?
    Proposed approach: exclude them by default and expose an explicit
    include-deactivated filter; this changes visible product behavior.
    Product decision needed before this slice can build.
    Resume: /rite-resolve q-2026-05-28-001 "<answer>"

  → appends q-...-001 to questions.md (status: open, gate: blocking)
  → writes `Awaiting human` block to state.md, sets Status: awaiting_human

You: /rite-resolve q-2026-05-28-001 "exclude by default; add the filter"

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
# Drop the sentinel before bed. Keys are optional: empty file works.
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

The loop does not mark a slice `built` while tests, type checks, or lint are
red. It uses the same wright and durable three-failure recovery budget for each
root-cause fingerprint. If the budget runs out, it records a blocker with the
reproduction and unsuccessful approaches instead of asking for retry approval.
AFK never silently accepts
genuine product, risk, or access decisions; see
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md) for the
full list.

## 11) Full unattended lifecycle: `/rite-autocomplete`

```text
/rite-autocomplete "add CSV export for admins" --max-slices 8
  → vague idea → runs devrites-interview, then /rite-spec + /rite-clarify in
    the only interactive window
  → Decision coverage: CLEAR → arms AFK, then drives: /rite-temper →
    /rite-define → /rite-vet → /rite-build ×N → /rite-prove → /rite-polish →
    /rite-review → /rite-seal (→ /rite-ship too when --ship is set)
  → at each soft gate, picks the option the relevant specialist/reviewer
    favours and records the rationale in decisions.md (never silently)
  → seal returns GO → autocomplete STOPS (default) and hands off to /rite-ship

/rite-ship                     # human runs it → renders the type-GO prompt
You: GO                        # → commit · push · tag, then archive + clear ACTIVE
```

Add `--ship` (alias `--yolo`) to confirm the final type-`GO` automatically.
Autocomplete then proceeds directly to `/rite-ship` without another prompt. It
still pauses for genuine product/scope/policy decisions, irreversible risk,
human-only access/actions, a NO-GO, exhausted `max_slices`, or low confidence.
Objective red checks use bounded recovery instead. Args:
`[idea] [--ship|--yolo] [--max-slices N]`.

## Checking in

- `/rite`: compact menu + suggested next command. **Does not** read state.
- `/rite-status`: detailed status: phase, **run mode (AFK / HITL)**, status
  (`running` / `awaiting_human` / `blocked` / `done`), next action, evidence,
  open questions broken down by gate, drift, risks, **handoff readiness**.
- `/rite-resolve <qid> "<answer>"`: answer / `--drop` / `--batch`-resolve an
  open question; the canonical writer for `status: open → answered` and the
  only thing that clears `Awaiting human`.

## Tips

- Commit `.devrites/` so the team and future sessions share feature state.
- **`.devrites/AFK` is per-developer, not per-repo**: gitignore it (or commit
  it deliberately if the team agrees on AFK defaults). The sentinel is
  read-only config: it toggles your local session mode and sets the initial
  `max_slices` budget; nothing else. The mutable remaining-slice count lives
  in `state.md` (`AFK slices remaining`), never in the sentinel.
- One feature active at a time (`ACTIVE`). Start a new workspace with
  `/rite-spec <feature>`; switch to an existing one with `/rite use <slug>`.
- **Recommended AFK progression**: HITL first to refine the prompt and plan,
  then drop the sentinel for the bulk stretch. Always cap iterations
  (`max_slices: 10` is a reasonable default).
- Invoke public `rite` / `rite-*` skills directly; let the model-invocable
  `devrites-*` specialists fire from their documented triggers. `devrites-lib`
  is an internal reference library, not a workflow.
- Prefixes are namespaces, not invocation policy. `user-invocable` and
  `disable-model-invocation` frontmatter are authoritative; the effective policy
  for every skill is catalogued in [`command-map.md`](command-map.md).
