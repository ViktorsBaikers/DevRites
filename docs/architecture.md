# DevRites architecture

DevRites is a **distributed but coordinated** set of project-local Claude Code
and Codex skill surfaces backed by one Go control plane. It makes an AI coding
agent behave like a disciplined senior engineer: frame → spec → optional temper
→ define → mandatory vet → build one verified slice → prove with evidence →
polish → review → seal → ship. `converge` is the recovery state when live code
and recorded intent need to meet again.

For the `.devrites/` load order, file budgets, artifact schema, aliases,
traceability rules, and phase-relative completeness model, see
[`engine/workspace-schema.md`](engine/workspace-schema.md).

## Layers

1. **Public lifecycle and workspace skills**: `.claude/skills/rite-*`,
   `user-invocable: true`. Each owns one bounded phase or workspace transition.
   Sequence: `rite-spec`, optional `rite-temper`, `rite-define`, mandatory
   `rite-vet`, `rite-build`, recovery `rite-converge`, `rite-prove`,
   `rite-polish`, `rite-review`, `rite-seal`, `rite-ship`; `rite-plan` repairs
   or reslices an active plan,
   plus the resume verb `rite-resolve` (answer a HITL gate and clear
   `Awaiting human`). The thin `/rite` menu and read-only `rite-status` live in
   the public utility layer below.
   `/rite-seal` **decides** GO/NO-GO and writes the verdict to `seal.md`;
   `/rite-ship` is the eighth core lifecycle rite that **executes** the
   irreversible git ladder and **closes** the task (archives the workspace,
   clears `ACTIVE`). Keeping the decision and the irreversible action as two
   separately-auditable steps is the point.
2. **Public utility and on-ramp skills**: `rite-adopt`, `rite-quick`,
   `rite-frame`, `rite-status`, `rite-doctor`, `rite-learn`, `rite-explain`,
   `rite-customize`, `rite-zoom-out`, `rite-prototype`, `rite-handoff`,
   `rite-pressure-test`, `rite-pov`, `rite-dogfood`, `rite-pr-feedback`, and
   `rite-autocomplete`. These are public commands. `rite-autocomplete` is the
   unattended orchestrator: it drives the whole lifecycle (spec → … → seal →
   ship) end-to-end, choosing the best option at each soft gate, pausing only
   on hard irreversible-risk / blocking / escalating gates or a NO-GO. The
   `devrites-` prefix is **namespace** (collision avoidance against bundled
   Claude Code skill names like `prototype`, `handoff`, `triage`, `diagnose`),
   not a visibility marker; `rite-pressure-test` carries no prefix because it
   doesn't collide.
3. **Internal specialist skills**: `.claude/skills/devrites-*` with
   `user-invocable: false`: `devrites-interview`, `-source-driven`,
   `-doubt`, `-ux-shape` (plans UX/UI into `design-brief.md` at `/rite-spec`),
   `-frontend-craft`, `-browser-proof`, `-debug-recovery`,
   `-api-interface`, `-audit` (dispatches the security / perf / simplify
   reviewer subagent on an axis argument), `-prose-craft`, and
   `-refresh-indexes`. These 11 specialists are model-invoked by public skills
   or host auto-selection; not menu noise. Whether a skill is public or
   internal is governed by the `user-invocable:` flag, not by the name
   prefix.

   Engineering rules live at `.claude/skills/devrites-lib/reference/standards/`.
   Workspace-operating lifecycle rites load `core.md` first and disclose phase-specific
   files on demand; compact utilities keep a narrower local contract. Parallel
   reviewer fan-out at `/rite-seal` is the shared reference file
   `devrites-lib/reference/parallel-dispatch.md`, not a skill.
4. **Supporting references**: `reference/*.md` inside each skill. Long checklists,
   templates, and anti-rationalization tables loaded on demand (progressive
   disclosure) so `SKILL.md` bodies stay small.
5. **Agents**: `.claude/agents/devrites-*` fresh-context subagents: **13 read-only + 1
   write-capable**. The read-only set is twelve reviewers: the post-build fan-out used by
   `/rite-seal` and the doubt loop (`devrites-spec-reviewer`, `-code-reviewer`, `-test-analyst`,
   `-frontend-reviewer`, `-security-auditor`, `-performance-reviewer`, `-devex-reviewer`,
   `-doubt-reviewer`, `-simplifier-reviewer`), the **pre-plan** `devrites-strategy-reviewer`
   (`/rite-temper`), the **pre-build** `devrites-plan-reviewer` (`/rite-vet`), and the
   **build-time** `devrites-forge-judge` (scores competing candidate builds on a `Forge: yes`
   slice), plus the cross-feature `devrites-retrospector` (mines the shipped archive at
   `/rite-ship` close). The one **write-capable** executor, `devrites-slice-wright`, is
   dispatched by `/rite-build` to write one slice in a clean context (the write-side mirror of
   the reviewers).
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

State lives in `.devrites/` as human-readable Markdown so it survives context
compaction and new sessions. The optional `.devrites/AFK` sentinel toggles
the session-level run mode (see "Run modes" below). See `usage.md` for the
workspace file list, [`command-map.md`](command-map.md) for the full
per-skill catalog with triggers + I/O, and
[`capability-surface-selection.md`](capability-surface-selection.md) for where future
capabilities belong.

## Design rationale

### Why the engine owns shared orientation (`devrites-lib`)
Every workspace-operating skill starts by reading the active feature's slug,
phase, present artifacts, run mode, and open-question tally. Re-deriving
that from raw Markdown in each skill was duplicated (step-0 prose across ~20 skills),
token-heavy (counting open gates meant re-reading the append-only `questions.md`, which
only grows), and error-prone (a missed AFK sentinel or a miscounted gate changes behavior).
So orientation is computed once by the `devrites-engine` binary, which prints a compact
digest each skill reads at step 0. The same binary owns the read-only gates
(`build-readiness`, `evidence-fresh`, `check-acceptance`) and state mutators
(`tick-afk`, `resolve`, `close-out`), so Claude Code, Codex, CI, and humans
all exercise the same control plane. `devrites-lib` remains an internal library skill
(`user-invocable: false`, not a command) for shared references. The orientation
path is read-only; mutation stays in dedicated engine subcommands.

### Why the engine owns install/update/uninstall semantics
Install/update/uninstall behavior lives in `engine/internal/install`: manifest writing
and pruning, shared-file marker merge/removal, Codex hook merge/removal, dry-run output,
binary lifecycle, and update flag replay all execute through `devrites-engine`. The
shell entrypoints (`install.sh`, `uninstall.sh`, `update.sh`) and npm entrypoint
(`bin/devrites.mjs`) remain bootstrap shims: they acquire a release bundle or engine
binary, then pass arguments through.

Some duplication is intentionally preserved at host boundaries. Raw `curl | bash`
install/uninstall must be self-contained enough to fetch the bundle before any sibling
files exist. Claude assets are authored under `pack/.claude/**`; Codex assets are
generated into `pack/generated/**` and installed to `.agents/skills`, `.codex/agents`,
`.codex/hooks.json`, and `AGENTS.md` because Codex and Claude use
different project-local conventions. Those generated host artifacts are delivered by the
npm installer; Claude/Codex plugin packaging is intentionally not a distribution path.

### Why `/engine` was rejected
A single `/engine` (or `/devrites`) mega-command would load every phase's instructions
into one context, creating constant context pressure and hiding the intent of each
step. It also makes "do only this phase" hard to enforce and bloats the recurring
token cost (skill bodies stay in context once loaded). DevRites splits the lifecycle
into small skills that load only what the current phase needs.

### Why `rite-*` names
Short, memorable, brand-aligned ("rites" = disciplined steps), and **collision-free**.
Built-in / bundled Claude Code commands include `/plan`, `/review`, `/run`, `/verify`,
`/code-review`, `/simplify`, `/security-review`, `/init`, `/compact`, `/debug`. The
`rite-` prefix avoids all of them. (Collision audit: `research/claude-code-skills-notes.md`.)

### Why a thin menu skill, not a mega-router
`/rite` is **only** an entrypoint: it shows a compact phase-grouped menu, prints
recommended-start guidance from `devrites-engine first-task`, and dispatches a
named verb to its owning skill. Menu mode does not read raw workspace state;
`/rite-status` owns detailed status. `/rite` deliberately does **not** duplicate
workflow logic. Doing so would recreate the mega-command problem. DevRites keeps
selection thin and lets each phase own its context.

### Why internal skills exist
Specialist processes (doubt, source-driven, frontend craft, browser proof, audits)
are **disciplines**, not user commands. As `user-invocable: false` specialist
skills they:
- stay out of the command menu (less cognitive load);
- are invoked automatically by the host model or explicitly by a public skill when their
  trigger conditions hit;
- keep each public `SKILL.md` small by housing the heavy process elsewhere.

### Why spec, architecture, plan, tasks, and traceability are separate artifacts
Combining investigation, specification, planning, and slicing makes it easier
to miss a question, guess at placement, or leave a requirement out of a slice.
DevRites splits them so each
is focused and gated. `/rite-spec` **investigates deeply and writes `spec.md`** (product
what/why, requirements, acceptance, non-goals, measurable success) and must pass its
readiness gate. `/rite-define` then turns that **approved spec** into `architecture.md`
(technical map), `plan.md` (approach), `tasks.md` (vertical `SLICE-###` work), and
`traceability.md` (AC/REQ → slice → proof → evidence → files). The spec is fully covered
before any building begins, but it does not become a long technical omnibus. `/rite-plan`
is the separate repair/reslice/re-order tool for an *active* plan when it goes stale or
drifts.

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
Deciding "is this safe to ship" and *actually shipping* are different acts with
different blast radii. `/rite-seal` is a **pure decision gate**: it walks acceptance
against evidence, fans out the fresh-context reviewers, and writes the GO / NO-GO
verdict to `seal.md` and runs no git. On GO it sets `state.md` `Next step:
/rite-ship` and stops. `/rite-ship` is the eighth core lifecycle rite: it refuses to run
without a GO recorded in `seal.md`, renders the type-`GO` prompt, runs the irreversible
git ladder (commit → push → tag/PR per the project's convention), writes `ship.md`,
then **closes the task** by setting phase `done` and archiving `.devrites/work/<slug>/` →
`.devrites/archive/<slug>/` (every `.md` preserved, never deleted) and clears
`.devrites/ACTIVE`. A GO seal is a verdict, not an authorization to push; keeping the
decision and the irreversible action as two separately-auditable steps is the point.

### Why `/rite-autocomplete` exists (the unattended orchestrator)
Some features are routine enough to run end-to-end without per-phase human iteration.
`/rite-autocomplete` drives the whole lifecycle (spec → temper → define → vet →
build×N → prove → polish → review → seal → ship) by reading each phase's
`SKILL.md` and executing its
workflow, carrying state through the workspace files rather than chat. A vague prompt
triggers an up-front `devrites-interview`, which is the only interactive window.
After that it runs unattended, choosing the best option at each soft gate and recording the rationale
in `decisions.md`. It does **not** weaken the safety gates: hard irreversible-risk
(auth / migration / public-API / red tests), blocking / escalating gates, an open
`gate: validating`, a NO-GO, exhausted `max_slices`, or low confidence all still pause.
By default it stops at the final type-`GO`; the `--ship` flag (alias `--yolo`)
auto-confirms it for a zero-touch push.

### Why persistent `.devrites/` state
Long features outlive a single context window. Durable Markdown for the spec,
plan, tasks, state, evidence, drift, and decisions lets any phase reload the
current position in any session, even after compaction. This is the main thing DevRites adds over typical
session-scoped workflows, which don't persist feature state.

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

Why a four-gate taxonomy (instead of a single "ask the user" pause): a single gate
becomes a queue under load. Mixing `advisory` (audit-only log) with `validating`
(async: build continues, merge blocks) and reserving `blocking` for synchronous halts
keeps the loop alive when the answer can wait, and pauses hard when it cannot.
AFK always pauses for destructive migrations, auth/authz boundaries, public API
breaks, and failed tests, type checks, or lint, regardless of the sentinel. See
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md) for the full
contract.

## Design choices at a glance

- **Surface**: 29 public `rite-*` skills (42 total): the thin `/rite` menu
  (carries the routing) + 8 lifecycle phases (`rite-spec`, `rite-define`,
  `rite-build`, `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`,
  `rite-ship`: seal **decides**, ship **executes + closes**) + the
  `rite-temper` (strategic, optional) and `rite-vet` (engineering, every plan)
  reviews + the `rite-quick` express lane and `rite-frame` pre-flight/self-audit
  lens + `rite-adopt` (onboard an existing codebase) + `rite-learn` (cross-feature
  lessons) + `rite-status` + `rite-doctor` (install health) +
  `rite-customize` (project-local overrides/extensions) +
  `rite-explain` (grounded concept/diff/idea/recap explanation) +
  `rite-pov` (external-option verdicts) + `rite-dogfood` (browser QA) +
  `rite-pr-feedback` (review-thread closure) +
  the `rite-plan` replan verb + `rite-converge` recovery + the `rite-resolve`
  resume verb + 4 ideation /
  handoff utilities (`rite-zoom-out`, `rite-prototype`, `rite-handoff`,
  `rite-pressure-test`) + `rite-autocomplete` (the unattended full-lifecycle
  orchestrator), plus 11 internal model-invoked `devrites-*` specialists and the
  `devrites-lib` library, not one mega-command. The `devrites-` prefix is a
  namespace (collision avoidance), not a public/internal marker.
  `user-invocable:` is. The 11 specialists are model-invoked;
  `devrites-lib` explicitly disables model invocation.
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
- **Review**: **feature-scoped** multi-axis review with severity labels +
  fresh-context subagents at the seal.
- **Scope**: clarify → seal (decide) → ship (commit → push → tag or PR,
  following the project's convention) → close. The CI
  pipeline stays with the project.
- **Install**: project-local, manifest-managed host artifacts; the optional
  shared engine binary is the sole sanctioned global artifact.

## Deviations from the original build brief (and why)

1. **Invocation semantics are explicit in frontmatter.** `user-invocable`
   controls whether a skill is a public command; `disable-model-invocation`
   independently marks explicit-only public utilities and the internal
   `devrites-lib` library. Model-invocable phases can still hand off and route;
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
