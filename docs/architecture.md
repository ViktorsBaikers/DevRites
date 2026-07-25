# DevRites architecture

DevRites combines project-local Claude Code and Codex skills with one Go control
plane. It gives AI coding agents a defined engineering process: frame → spec →
mandatory adaptive clarify → optional temper → define → mandatory vet → build
one verified slice → prove with evidence → polish → review → seal → ship.
`converge` is the recovery state for bringing live code and recorded intent
back into agreement.

For the `.devrites/` load order, file budgets, artifact schema, aliases,
traceability rules, and phase-relative completeness model, see
[`engine/workspace-schema.md`](engine/workspace-schema.md).

## Layers

1. **Public lifecycle and workspace skills**: `.claude/skills/rite-*`,
   `user-invocable: true`. Each owns one bounded phase or workspace transition.
   Sequence: `rite-spec`, mandatory adaptive `rite-clarify`, optional `rite-temper`, `rite-define`, mandatory
   `rite-vet`, `rite-build`, recovery `rite-converge`, `rite-prove`,
   `rite-polish`, `rite-review`, `rite-seal`, and `rite-ship`. `rite-plan`
   repairs or reslices an active plan. The resume verb `rite-resolve` answers a
   HITL gate and clears `Awaiting human`. `rite-upgrade` is a conditional
   maintenance route for semantically stale active workspaces, not a lifecycle
   phase. The thin `/rite` menu and read-only `rite-status` live in the public
   utility layer below.
   `/rite-seal` **decides** GO/NO-GO and writes the verdict to `seal.md`;
   `/rite-ship` is the final core lifecycle rite that **executes** the
   irreversible git ladder and **closes** the task by archiving the workspace
   and clearing `ACTIVE`. Separate steps let users audit the release decision
   before any irreversible action.
2. **Public utility and on-ramp skills**: `rite-adopt`, `rite-quick`,
   `rite-frame`, `rite-status`, `rite-doctor`, `rite-upgrade`, `rite-learn`, `rite-explain`,
   `rite-customize`, `rite-zoom-out`, `rite-prototype`, `rite-handoff`,
   `rite-pressure-test`, `rite-pov`, `rite-dogfood`, `rite-pr-feedback`, and
   `rite-autocomplete`. These are public commands. `rite-autocomplete` is the
   unattended orchestrator. It drives the whole lifecycle (spec → … → seal →
   ship), chooses the recommended option at each soft gate, and pauses only for
   hard irreversible-risk, blocking, or escalating gates, or a NO-GO. The
   `devrites-` prefix prevents collisions with bundled Claude Code skill names
   such as `prototype`, `handoff`, `triage`, and `diagnose`; it does not mark
   visibility. `rite-pressure-test` needs no prefix because it does not collide.
3. **Internal specialist skills**: `.claude/skills/devrites-*` with
   `user-invocable: false`: `devrites-interview`, `-source-driven`,
   `-doubt`, `-ux-shape` (plans UX/UI into `design-brief.md` at `/rite-spec`),
   `-frontend-craft`, `-browser-proof`, `-debug-recovery`,
   `-api-interface`, `-audit` (dispatches the security / perf / simplify
   fresh-context reviewer on an axis argument), `-prose-craft`, and
   `-refresh-indexes`. Public skills or host auto-selection invoke these 11
   specialists. They do not appear in the menu. The `user-invocable:` flag,
   rather than the name prefix, determines whether a skill is public.

   Engineering rules live at `.claude/skills/devrites-lib/reference/standards/`.
   Workspace-operating lifecycle rites load `core.md` first and read
   phase-specific files on demand. Compact utilities keep a narrower local
   contract. Parallel
   reviewer fan-out at `/rite-seal` is the shared reference file
   `devrites-lib/reference/parallel-dispatch.md`, not a skill.
4. **Supporting references**: `reference/*.md` inside each skill. Long checklists,
   templates, and anti-rationalization tables loaded on demand (progressive
   disclosure) so `SKILL.md` bodies stay small.
5. **Agents**: `.claude/agents/devrites-*` contains **18 flat depth-one roles**:
   17 read-only leaves and the sole source/test writer,
   `devrites-slice-wright`. The read-only set includes three bounded work
   leaves (`devrites-evidence-scout`, `devrites-plan-drafter`, and
   `devrites-proof-runner`), the fresh `devrites-upgrade-planner`, plus the
   existing reviewers, auditors, judge, and retrospector. Public rites remain
   authoritative: leaves return typed
   evidence, never ask the human, change phase, or write canonical
   `.devrites/**` state. See [`orchestration.md`](orchestration.md) for the
   dispatch, fallback, identity, and reconciliation contract.
6. **Engineering rules**: DevRites' own stack-agnostic rules installed to
   `.claude/skills/devrites-lib/reference/standards/`. Workspace-operating lifecycle
   skills read `core.md` in step 0; compact utilities load only their local or conditional
   rules. On-demand files load by the phase that needs them:
   - **Craft:** `coding-style.md` · `prose-style.md` · `patterns.md` · `error-handling.md` ·
     `testing.md` · `spec-grammar.md` · `documentation.md` · `skill-authoring.md`.
   - **Quality / safety:** `code-review.md` · `principles.md` · `security.md` ·
     `performance.md` · `observability.md` · `developer-experience.md` · `deprecation.md` ·
     `anti-patterns.md`.
   - **Workflow / ops:** `development-workflow.md` · `git-workflow.md` ·
     `ci-cd.md` · `hooks.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md` ·
     `tooling.md` · `elicitation.md`.
   - **Checklists:** `definition-of-done.md` · `review-checklist.md` ·
     `test-proof-checklist.md` · `browser-proof-checklist.md` · `security-checklist.md`.
   - **Index:** `README.md` (phase mapping, loading model).

State lives in `.devrites/` as human-readable Markdown so later sessions can
reload it after context compaction. The optional `.devrites/AFK` sentinel
toggles the session-level run mode (see "Run modes" below). See `usage.md` for
the workspace file list, [`command-map.md`](command-map.md) for the full
per-skill catalog with triggers and I/O, and
[`capability-surface-selection.md`](capability-surface-selection.md) for where future
capabilities belong.

## Design rationale

### Why the engine owns shared orientation (`devrites-lib`)
Every workspace-operating skill needs the active feature's slug, phase, present
artifacts, run mode, and open-question tally. Parsing raw Markdown in each skill
duplicated the same setup across about 20 skills. Counting open gates also meant
rereading the append-only `questions.md`, and a missed AFK sentinel or
miscounted gate could change behavior. The `devrites-engine` binary now computes
orientation once and prints a compact digest that each skill reads at step 0.
The same binary owns the read-only gates
(`build-readiness`, `evidence-fresh`, `check-acceptance`) and state mutators
(`tick-afk`, `resolve`, `close-out`), so Claude Code, Codex, CI, and humans
all exercise the same control plane. `devrites-lib` remains an internal library
skill (`user-invocable: false`, not a command) for shared references. The
orientation command only reads state; dedicated engine subcommands handle
mutations.

### Why the engine owns install/update/uninstall semantics
`engine/internal/install` implements install, update, and uninstall behavior:
manifest writing and pruning, shared-file marker merging and removal, Codex hook
merging and removal, dry-run output, binary lifecycle, and update flag replay.
The shell entrypoints
(`install.sh`, `uninstall.sh`, `update.sh`) and npm entrypoint
(`bin/devrites.mjs`) remain bootstrap shims. They acquire a release bundle or
engine binary, then pass arguments through.

Directly managed manifest entries carry SHA-256 ownership records. Before the
first refresh, prune, or uninstall mutation, the engine classifies every
affected path. It preserves customized or legacy entries without hashes unless
`--force` is explicit, and it checks each path again immediately before a
destructive action. Marker-owned shared files retain their block or hook merge
policy. The engine rejects existing symlinks, junctions, and resolved target
escapes even under force.

Binary replacement is a separate transaction. Before replacement, the binary
at the exact staged path must report the requested release version. A backup in
the same directory remains until the binary at the installed path passes the
same check in a new process. On failure, the engine atomically restores the old
bytes and mode or removes a bad first install. It does not use `PATH` to
establish binary identity.

Some duplication is intentionally preserved at host boundaries. Raw `curl | bash`
install/uninstall must be self-contained enough to fetch the bundle before any sibling
files exist. Claude assets are authored under `pack/.claude/**`; Codex assets are
generated into `pack/generated/**` and installed to `.agents/skills`, `.codex/agents`,
`.codex/hooks.json`, and `AGENTS.md` because Codex and Claude use
different project-local conventions. Those generated host artifacts are delivered by the
npm installer; Claude/Codex plugin packaging is intentionally not a distribution path.

### Why semantic upgrade is separate from update and migration

Three operations solve different problems. `devrites-engine update` refreshes
the installed binary and pack. `devrites-engine migrate` normalizes workspace
layout and structural state schema. Neither one can claim that an active plan
still follows the current planning rules.

`/rite-upgrade [slug]` owns that semantic reconciliation. Build readiness uses
the `devrites.readiness-artifacts.v2` contract declared by
`decision-coverage.md`, `eng-review.md`, and `test-plan.md`; it returns code `8`
when those existing artifacts are stale. The rite then asks a fresh read-only
`devrites-upgrade-planner` to classify the gap before the root changes anything.
It preserves completed source, slice bodies, decisions, and evidence, repairs
only active unfinished planning, removes obsolete proof recipes and local
wrappers, and reruns the current readiness gates. Already-current workspaces
that pass readiness, completed workspaces, and archives remain untouched. See
[ADR-0012](adr/0012-semantic-workspace-upgrades.md).

### Why `/engine` was rejected
A single `/engine` (or `/devrites`) command would load every phase's
instructions into one context. That would increase context pressure, obscure
the purpose of each step, and make phase boundaries harder to enforce. Skill
bodies stay in context once loaded, so the recurring token cost would also
grow. Separate skills load only what the current phase needs.

### Why `rite-*` names
The team chose `rite-` because it is short, easy to remember, and matches the
product's use of "rites" for disciplined steps. The prefix also avoids
collisions. Built-in or bundled Claude Code commands include `/plan`, `/review`,
`/run`, `/verify`, `/code-review`, `/simplify`, `/security-review`, `/init`,
`/compact`, and `/debug`. (Collision audit:
`research/claude-code-skills-notes.md`.)

### Why a thin menu skill, not a mega-router
`/rite` is an entrypoint. It shows a compact phase-grouped menu, prints
recommended-start guidance from `devrites-engine first-task`, and dispatches a
named verb to its owning skill. Menu mode does not read raw workspace state;
`/rite-status` owns detailed status. `/rite` does not duplicate workflow logic,
which keeps selection small and leaves each phase in control of its context.

### Why internal skills exist
Specialist processes such as doubt, source-driven research, frontend craft,
browser proof, and audits are not direct user commands. As
`user-invocable: false` skills they:
- stay out of the command menu (less cognitive load);
- are invoked automatically by the host model or explicitly by a public skill when their
  trigger conditions hit;
- keep each public `SKILL.md` small by housing the heavy process elsewhere.

### Why spec, architecture, plan, tasks, and traceability are separate artifacts
Combining investigation, specification, planning, and slicing makes it easier
to miss a question, choose the wrong placement, or omit a requirement from a
slice. Each concern therefore has its own artifact and gate. `/rite-spec`
investigates the request and codebase in depth, then writes `spec.md` with the
product what and why, requirements, acceptance criteria, non-goals, and
measurable success. The spec
must pass its readiness gate. `/rite-clarify` then audits the full actor,
journey, data, interface, and operations topology and writes
`decision-coverage.md`. A complete spec needs no questions.
`/rite-define` turns that **approved, clarified spec** into `architecture.md`
(technical map), `plan.md` (approach), `tasks.md` (vertical `SLICE-###` work), and
`traceability.md` (AC/REQ → slice → proof → evidence → files). `/rite-vet` then records
`Implementation readiness: READY` before build. Both verdict artifacts are
semantically validated and bound to their canonical inputs by SHA-256 digest;
marker text alone or a stale digest cannot pass. This checks the full spec
before Build without turning `spec.md` into a long technical document.
`/rite-plan` separately repairs, reslices, or reorders an active plan when it
goes stale or drifts.

### Why `/rite-polish` is one skill with two progressive-disclosure halves
Polish has two natural halves: **code** (simplify, dead code, naming, plus
backend if BE was touched) and **UI** (normalize to the design system, then
ship-quality detail). The earlier design split them into `rite-polish-code`
and `rite-polish-ui` sub-skills, but a skill-on-skill dispatch is fragile
(the caller has to re-discover the callee by description) and the two halves
share the same operating rules. They now live as `reference/code.md` and
`reference/ui.md` inside the same skill, loaded only when their phase
trigger fires. This follows the supporting-files pattern from Claude Code's current
[skills reference](https://code.claude.com/docs/en/slash-commands#add-supporting-files).
`/rite-polish` is the orchestrator: always reads `reference/code.md`
(Phase 1 + 2), detects UI scope from the diff and reads `reference/ui.md`
(Phase 3 + 4) when needed. The orchestrator accepts mode tokens (`bolder`,
`quieter`, `distill`, `harden`, `normalize-only`) that pass through to
Phase 4.

Normalization remains the entry gate for UI polish. The UI reference refuses
to run Phase 4 before Phase 3 because detail work would otherwise reinforce
patterns that do not match the project's design system.

### Why seal and ship are separate phases (`/rite-seal`, `/rite-ship`)
Deciding whether a change is safe to ship has a smaller blast radius than
shipping it. `/rite-seal` checks acceptance against evidence, dispatches the
fresh-context reviewers, writes the GO or NO-GO verdict to `seal.md`, and runs
no git commands. On GO, it sets `state.md` to `Next step: /rite-ship` and stops.
`/rite-ship` refuses to run without a GO in `seal.md`. It renders the type-`GO`
prompt, runs the irreversible git ladder (commit → push → tag/PR under the
project's convention), and writes `ship.md`. It then sets phase `done`, moves
`.devrites/work/<slug>/` to `.devrites/archive/<slug>/` with every `.md` file
preserved, and clears `.devrites/ACTIVE`. The GO verdict does not authorize a
push; the separate Ship step does.

### Why `/rite-autocomplete` exists (the unattended orchestrator)
Some features are routine enough to run without human input at every phase.
`/rite-autocomplete` drives the whole lifecycle (spec → clarify → temper → define → vet →
build×N → prove → polish → review → seal → ship) by reading each phase's
`SKILL.md` and executing its
workflow. It carries state through workspace files rather than chat. A vague
prompt starts `devrites-interview`; `/rite-spec` and `/rite-clarify` then
finish the only interactive window. After decision coverage is CLEAR it runs unattended,
choosing the recommended option at each soft gate and recording the rationale
in `decisions.md`. It does **not** weaken the safety gates: genuine
product/scope/policy decisions, irreversible risk, human-only access/actions,
an open human-owned gate, a NO-GO, exhausted `max_slices`, or low confidence
still pause. Agents use bounded recovery for red tests, runtime failures, and
missing technical coverage. By default the workflow stops at the final
type-`GO`; the `--ship` flag (alias `--yolo`) confirms it and pushes without
another prompt.

### Why persistent `.devrites/` state
Long features outlive a single context window. Durable Markdown for the spec,
plan, tasks, state, evidence, drift, and decisions lets any phase reload the
current position in any session, even after compaction. Later-phase
clarification stores `return_phase` and `return_next_action` in `state.md`;
technical recovery stores a fingerprinted three-attempt budget in
`recovery-attempts.jsonl`. Session-scoped workflows cannot provide the same
resume behavior because they do not persist feature state.

### Run modes: HITL & AFK

DevRites runs the same lifecycle two ways, configured at two levels:

- **Per-slice** (planning-time): each `tasks.md` slice declares `Mode: AFK | HITL`.
  HITL slices add `Gate: advisory | validating | blocking | escalating`, `SLA`, and
  `Checkpoint` so the agent knows how disruptive the pause should be.
- **Per-session** (run-time): the presence of `.devrites/AFK` flips the session-level
  default. Empty file = AFK with safe defaults; YAML body widens behavior
  (`max_slices`, `notify`, `allow_gates`).

The pause primitive is a **pre-action interrupt**, not a post-action review queue.
It follows LangGraph's `interrupt()` / `Command(resume=)` model. `/rite-build` writes
an `Awaiting human` block to `state.md` and a question to `questions.md`, then stops.
`/rite-resolve` is the canonical resume verb. Restarting a session reads the workspace
back into a consistent state because the pause is durable Markdown, not chat memory.

Why a four-gate taxonomy (instead of a single "ask the user" pause): a single
gate becomes a queue under load. `advisory` writes only an audit log.
`validating` lets the build continue but blocks the merge. `blocking` stops
synchronously. This lets work continue when an answer can wait and pauses it
when the answer is required.
AFK always pauses for genuine product/scope/policy choices, irreversible risk,
and human-only access or actions, regardless of the sentinel. Objective test,
type, lint, runtime, and coverage failures stay inside bounded technical
recovery; exhaustion records a blocker without inventing a question. See
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md) for the full
contract.

## Design choices at a glance

- **Surface**: 31 public `rite-*` skills (44 total), plus the thin `/rite`
  menu: 32 public and 12 internal. The lifecycle
  includes mandatory adaptive Clarify, optional Temper, mandatory Vet, and
  Converge recovery; Seal **decides** and Ship **executes + closes**. Eleven
  `devrites-*` specialists are model-invoked and `devrites-lib` is the shared
  non-command library. Visibility comes from `user-invocable:`, not the
  namespace prefix.
- **Selection**: the `/rite` menu skill carries the routing table; every
  workflow skill enforces a "right skill, right time" rule in its body.
- **State**: durable `.devrites/` Markdown that survives compaction and new sessions.
  `.devrites/AFK` presence is the single source of truth for run mode (HITL/AFK);
  there is no `state.md` run-mode field to drift out of sync.
- **Run modes**: same lifecycle runs HITL (default; pause at typed gate) or AFK
  (drop `.devrites/AFK`; loop continues, discretionary pauses downgrade to
  advisory log, irreversible risk always pauses).
- **Slice rule**: build **one vertical slice, then stop**. There is no automatic continuation.
- **Drift**: an explicit **Spec Drift Guard** in build/prove/polish/review/seal.
- **Design**: `devrites-frontend-craft` + a four-phase `/rite-polish` orchestrator (code + backend always; UI normalize + polish when UI is in scope).
- **Agents**: 18 fresh-context roles at flat depth one: 17 read-only leaves and
  one wright. Dispatch order is named, then guarded generic; if neither can
  spawn, the workflow stops for HITL.
- **Review**: **feature-scoped** multi-axis review with severity labels and
  fresh-context agents at the seal.
- **Scope**: clarify → seal (decide) → ship (commit → push → tag or PR,
  following the project's convention) → close. The CI
  pipeline stays with the project.
- **Install**: project-local, manifest-managed host artifacts; the optional
  shared engine binary is the only allowed global artifact.

## Deviations from the original build brief (and why)

1. **Invocation semantics are explicit in frontmatter.** `user-invocable`
   controls whether a skill is a public command; `disable-model-invocation`
   independently marks explicit-only public utilities and the internal
   `devrites-lib` library. `required-agent-roles` is the canonical list of
   unconditional fresh-agent roles for one invocation (`none` is explicit);
   Codex derives its fail-closed dispatch receipt from the installed mirror.
   Model-invocable phases can still hand off and route;
   per-phase side-effect discipline (for example, "stop after one slice") is
   enforced in the skill **body**, because invocation flags cannot express it.
2. **`context: fork` used selectively.** Verified it is a real field (isolated
   subagent, body-as-prompt, no conversation history). It is applied only to the three
   self-contained read-only audit skills (`devrites-audit simplify`,
   `devrites-audit security`, `devrites-audit perf`), which re-read
   `.devrites/` + `git diff` themselves. Interactive skills (doubt, interview, craft)
   are **not** forked because they need live context and user turns.
3. **Fresh-context adversarial review via real `.claude/agents/`** rather than relying
   on `context: fork` everywhere, because an agent can be handed the workspace path to
   read, whereas a forked skill cannot see the conversation.

Everything else follows the brief's structure, command names, state model, and
acceptance criteria.

## Repository layout

See the top-level tree in `README.md` and the installed-target tree in `usage.md`.
The canonical source pack lives under `pack/.claude/` (skills, agents, rules,
and Claude hook wiring). The build renders `pack/generated/` host artifacts;
the installer writes Claude assets under project `.claude/` and Codex assets
under `.agents/`, `.codex/`, and the managed `AGENTS.md` block.
