# Using DevRites

These examples cover the common DevRites workflows. Start a new feature with
`/rite-spec` and bring an existing codebase in with `/rite-adopt`.
`/rite-quick` and `/rite-frame` handle bounded work outside the full feature
lifecycle. Workspace phases first read the active workspace from
`.devrites/ACTIVE` and `.devrites/work/<slug>/`. If none exists, they report the
command that can create or select one. `/rite-upgrade [slug]` is the
compatibility audit for an older active workspace that cannot resume.

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
| `README.md` | `/rite-spec` | canonical compact workspace map: phase, status, next action, artifact map, read-next table, gates |
| `brief.md` | `/rite-spec` | one-line objective + definition of done |
| `spec.md` | `/rite-spec` | product WHAT/WHY, requirements, acceptance, boundaries, measurable success, and one capability-impact declaration for new/materially revised specs |
| `decision-coverage.md` | `/rite-clarify` | topology-first coverage and a native semantic `CLEAR` verdict reconciled against the spec, decisions, and assumptions |
| `architecture.md` | `/rite-define` | owning module/layer, integration points, data/API/events, dependencies, risks |
| `flows.md` | `/rite-spec` or `/rite-define` | optional Mermaid diagrams when sequence/state/data flow clarifies behavior |
| `references/` + `references.md` | `/rite-spec` | saved design refs: screenshots, Figma, video, links |
| `strategy.md` | `/rite-temper` | strategic spec review (optional): scope mode, pre-mortem, dimension scores |
| `plan.md` | `/rite-define` | approach, dependency graph, checkpoints, rollback, and conditional shared-contract provider/consumer proof |
| `tasks.md` | `/rite-define` | ordered `SLICE-###` vertical slices, each mapped to `AC-###` and tagged `Mode: AFK \| HITL` + gate fields |
| `traceability.md` | `/rite-define` | AC/REQ → slices → tests/proofs → evidence → touched files matrix |
| `eng-review.md` | `/rite-vet` | mandatory native engineering review, reconciled `READY` verdict, and stable Build-input binding |
| `test-plan.md` | `/rite-vet` | build-readable coverage target, per-gap test requirements, and acceptance-to-test map reviewed by native agents |
| `state.md` | every phase | working cursor: phase, status, next action, slice, AFK budget, durable clarification return fields, and `Awaiting human` only when paused |
| `questions.md` | every phase | append-only Q&A: qid, slice, gate, status (`open` / `answered` / `dropped`), proposed answer, raised/answered timestamps |
| `decisions.md` / `assumptions.md` | every phase | running logs |
| `drift.md` | Spec Drift Guard | drift events + resolutions |
| `touched-files.md` | `/rite-build`, `/rite-polish` | strict manifest of the exact project candidate; Review trail is navigation only |
| `evidence.md` | `/rite-build`, `/rite-prove` | canonical `EVID-###` command/action proof plus the exact candidate binding |
| `browser-evidence.md` | `/rite-prove`, `/rite-polish` (UI) | screenshots, console, network, viewport runs, Visual Verdict, and exact candidate binding |
| `design-brief.md` | `devrites-ux-shape` during `/rite-spec` | vetted shape, states, interaction model, and design-reference direction; a material later change returns through `/rite-vet` |
| `polish-report.md` | `/rite-polish` | Phase 1-4 findings + fixes |
| `review.md` | `/rite-review` | candidate-bound Spec + Standards axes and severity-labelled findings |
| `seal.md` | `/rite-seal` | candidate-bound GO/NO-GO verdict + acceptance walk + blockers |
| `ship.md` | `/rite-ship` | what shipped: commit SHA(s), branch, tag/PR, acceptance summary, follow-ups |

When `/rite-ship` closes the task, it archives the whole workspace from
`.devrites/work/<slug>/` to `.devrites/archive/<slug>/` and clears
`.devrites/ACTIVE`. Every Markdown file remains in
`.devrites/archive/<slug>/` as an audit trail.

Compatibility is limited to official released cursors in
`.devrites/work/<slug>/state.md`: v1/v2 bullet fields (`Phase`, `Next step`,
`qid`) and v3 table fields (`phase`, `next_action`, `question_id`). Reads do not
rewrite the workspace or emit telemetry. `/rite-upgrade` is the native,
preservation-first audit: age/cursor encoding alone is not a defect, and only a
cited current-contract failure routes repair through its phase owner. Completed
work and evidence stay intact. Candidate defects may route current Prove,
Polish, Review, and Seal in that order; old passes are never synthesized. The
engine owns deterministic v5 schema normalization.

Project-root sentinel (outside the workspace):

| File | Created by | Holds |
|---|---|---|
| `.devrites/AFK` | you (presence = AFK mode active) | optional YAML: `max_slices`, `notify`, `allow_gates`. Empty file = AFK with defaults. See [`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md). |
| `.devrites/CHECKPOINT` | you or `/rite-autocomplete` (presence = checkpoint mode) | empty sentinel. When set, `/rite-build` commits each proven slice local-only as `WIP(<slug>)` so a crash mid-build loses neither source nor reasoning; after literal `GO`, `/rite-ship` collapses an eligible disclosed WIP run into the feature commit. See [`pack/.claude/skills/rite-build/reference/checkpoint.md`](../pack/.claude/skills/rite-build/reference/checkpoint.md). |

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
  → writes spec.md with one Capability impact declaration (creates the workspace)

/rite-clarify
  → scans the full topology; asks zero questions when the contract is complete
  → writes fresh Decision coverage: CLEAR

/rite-define
  → reads the approved spec
  → writes plan.md + vertical task slices + state; a changed API/event/schema
    boundary names one canonical Shared contract proof artifact and consuming
    provider/consumer tests (otherwise a justified no-impact statement)
  → stops for confirmation

/rite-vet
  → reviews every defined plan; uses a light or full pass based on stakes
  → writes eng-review.md + test-plan.md and binds their stable Build inputs
```

For an existing capability, a MODIFIED requirement contains its full next
version and preserves every prior scenario and normative/source-grounded claim
unless an accepted `DEC-###` explicitly authorizes removal. During proof,
behavior needs a positive, discriminating assertion and decisive observed
signal. Skipped/filtered/pending or zero-test runs, assertion-free or
tautological tests, unexecuted commands, and exit status alone do not prove
behavior; compile, typecheck, lint, and build prove only their static criterion.

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
/rite-ship                   # read-only preflight → type-GO → commit + optional approved remote actions → close
```

In default HITL, `/rite-build` never starts the next slice automatically; you
decide when it runs. The explicit `.devrites/AFK` sentinel is the bounded
low-risk chaining exception described in §10. For each slice, the root states
exact project-relative paths in the writer task, waits, rejects any extra path
in `git diff --name-only`, reviews test integrity, runs repository proof, and
maintains the strict candidate manifest.
Prove binds real evidence to its content digest; Polish completes durable
capability/design/ADR rollups and affected re-proof before Review. Ship never
changes candidate paths. Its pre-GO work is read-only and discloses the exact
attempt; literal `GO` then authorizes eligible checkpoint collapse, exact
staging, staged scope/byte/binding/secret validation, commit and reverification,
optional approved push/tag/PR actions, and archive. See
[candidate integrity](candidate-integrity.md).
`/rite-seal` **decides**; `/rite-ship` **executes + closes**. To run the whole
sequence unattended, see `/rite-autocomplete` (§11).

## 2a) Audit an older workspace that cannot resume

Invoke the native compatibility route for an older active unfinished workspace:

```text
/rite-upgrade ark-panda-redesign
  → exact read-only devrites-upgrade-planner cites each applicable current rule
    and its workspace evidence
  → returns current, repairable, unsupported, or gap; age is never enough
  → only admitted defects route through Clarify, Plan repair, Converge, Vet,
    Prove, Polish, Review, or Seal
  → ambiguous legacy candidate scope is a gap; current owners run fresh proof
    and never synthesize an old pass
  → protected source, completed work, decisions, and evidence are rechecked
  → current/completed work is a no-op; unsupported or unverifiable input stops
```

This is separate from the npm/shell update flow, which acquires a local
candidate and asks the engine to refresh the installed binary and pack. Upgrade
does not update the pack or migrate workspace structure.

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

Settled technical objective failures stay with bounded recovery in the active
slice. Plan repair runs only when the durable plan is wrong; product, policy,
or irreversible-risk choices pause for the human.

## 4) UI feature with Playwright MCP

```text
/rite-spec settings-theme-toggle
  → you give a Figma link + a screenshot of the target toggle; rite-spec
    views them, saves the screenshot to references/, indexes references.md
  → devrites-ux-shape writes design-brief.md; rite-spec writes spec.md

/rite-clarify
  → scans the full topology; asks zero questions when the contract is complete
  → writes fresh Decision coverage: CLEAR

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
status of **pending (manual)**. Pending work does not prove the affected
criterion; Seal blocks when that criterion is required for acceptance.

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
/rite-ship                   # read-only preflight → type-GO → commit + optional approved remote actions → close
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

This is the explicit exception to the one-slice HITL return: the sentinel
authorizes the same `/rite-build` invocation to chain only the slices admitted
by its cap and pause rules.

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
red. The caller and recovery loop count at most three failed attempts for each
causal fingerprint from current context plus recorded Dead ends/evidence. If the budget runs out, it records a blocker with the
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
You: GO                        # → commit, optional approved push/tag/PR, then archive + clear ACTIVE
```

Add `--ship` (alias `--yolo`) to continue through `/rite-ship` preflight.
Autocomplete discloses the exact Git plan, then still stops for a fresh literal
`GO` and native host approval. It also pauses for genuine product/scope/policy
decisions, irreversible risk, human-only access/actions, a NO-GO, exhausted
`max_slices`, or low confidence. Objective red checks use bounded recovery
instead. Args:
`[idea] [--ship|--yolo] [--max-slices N] [--full] [--cross-model]`.
Optional flags are inactive unless their exact token occurs in the invocation;
`--max-slices` requires one positive base-10 integer. `--full` selects the Full
execution profile, and `--cross-model` arms Vet's second opinion.
Autocomplete never rewrites an existing `.devrites/AFK`. After Vet fixes the
pending-slice count, it stores the lower effective budget in `state.md`; an
existing remaining counter can only stay the same or decrease.

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
  in `state.md` (`AFK slices remaining`), never in the sentinel. The root
  charges it once with each green pending → built transition and stops before
  another dispatch at zero; malformed values fail closed.
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
