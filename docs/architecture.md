# DevRites Architecture

DevRites is a **distributed but coordinated** set of project-local Claude Code skills
that make an AI coding agent behave like a disciplined senior engineer: clarify →
spec+plan → build one verified slice → prove with evidence → polish → review → seal →
ship.

For the `.devrites/` load order, file budgets, artifact schema, aliases,
traceability rules, and phase-relative completeness model, see
[`engine/workspace-schema.md`](engine/workspace-schema.md).

## Layers

1. **Public workflow skills** — `.claude/skills/rite-*`, `user-invocable: true`. Each
   owns exactly one phase and reads/writes the `.devrites/` workspace.
   Sequence: `rite-spec`, `rite-define`, `rite-plan`, `rite-build`,
   `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`, `rite-ship`,
   plus the read-only `rite-status`, the resume verb `rite-resolve` (answer
   a HITL gate and clear `Awaiting human`), and the thin `/rite` menu.
   `/rite-seal` **decides** GO/NO-GO and writes the verdict to `seal.md`;
   `/rite-ship` is the eighth lifecycle phase that **executes** the
   irreversible git ladder and **closes** the task (archives the workspace,
   clears `ACTIVE`). Keeping the decision and the irreversible action as two
   separately-auditable steps is the point.
2. **Public utility skills** — `.claude/skills/rite-zoom-out`,
   `rite-prototype`, `rite-handoff`, `rite-pressure-test`, and
   `rite-autocomplete` — public commands. `rite-autocomplete` is the
   unattended orchestrator: it drives the whole lifecycle (spec → … → seal →
   ship) end-to-end, choosing the best option at each soft gate, pausing only
   on hard irreversible-risk / blocking / escalating gates or a NO-GO. The
   `devrites-` prefix is **namespace** (collision avoidance against bundled
   Claude Code skill names like `prototype`, `handoff`, `triage`, `diagnose`),
   not a visibility marker; `rite-pressure-test` carries no prefix because it
   doesn't collide.
3. **Internal specialist skills** — `.claude/skills/devrites-*` with
   `user-invocable: false`: `devrites-interview`, `-source-driven`,
   `-doubt`, `-ux-shape` (plans UX/UI into `design-brief.md` at `/rite-spec`),
   `-frontend-craft`, `-browser-proof`, `-debug-recovery`,
   `-api-interface`, `-audit` (dispatches the security / perf / simplify
   reviewer subagent on an axis argument). Model-invoked by public skills
   or Claude's auto-selection; not menu noise. Whether a skill is public or
   internal is governed by the `user-invocable:` flag, not by the name
   prefix.

   Engineering rules live at `.claude/skills/devrites-lib/reference/standards/` (each `rite-*` skill Reads
   `.claude/skills/devrites-lib/reference/standards/core.md` as its first step; the other 15 files load on
   demand — no session-start autoload). Parallel reviewer fan-out at
   `/rite-seal` is a reference file
   (`rite-seal/reference/parallel-dispatch.md`), not a skill.
4. **Supporting references** — `reference/*.md` inside each skill. Long checklists,
   templates, and anti-rationalization tables loaded on demand (progressive
   disclosure) so `SKILL.md` bodies stay small.
5. **Agents** — `.claude/agents/devrites-*` fresh-context subagents: **13 read-only + 1
   write-capable**. The read-only set is twelve reviewers — the post-build fan-out used by
   `/rite-seal` and the doubt loop (`devrites-spec-reviewer`, `-code-reviewer`, `-test-analyst`,
   `-frontend-reviewer`, `-security-auditor`, `-performance-reviewer`, `-devex-reviewer`,
   `-doubt-reviewer`, `-simplifier-reviewer`), the **pre-plan** `devrites-strategy-reviewer`
   (`/rite-temper`), the **pre-build** `devrites-plan-reviewer` (`/rite-vet`), and the
   **build-time** `devrites-forge-judge` (scores competing candidate builds on a `Forge: yes`
   slice) — plus the cross-feature `devrites-retrospector` (mines the shipped archive at
   `/rite-ship` close). The one **write-capable** executor, `devrites-slice-wright`, is
   dispatched by `/rite-build` to write one slice in a clean context (the write-side mirror of
   the reviewers).
6. **Engineering rules** — DevRites' own stack-agnostic rules installed to
   `.claude/skills/devrites-lib/reference/standards/`. Each `rite-*` skill Reads `core.md` as its first step
   (step 0); 25 on-demand files load by the phase that needs them:
   - **Craft:** `coding-style.md` · `prose-style.md` · `patterns.md` · `error-handling.md` ·
     `testing.md` · `spec-grammar.md` · `documentation.md` · `skill-authoring.md`.
   - **Quality / safety:** `code-review.md` · `principles.md` · `security.md` ·
     `performance.md` · `observability.md` · `developer-experience.md` · `deprecation.md` ·
     `anti-patterns.md`.
   - **Workflow / ops:** `development-workflow.md` · `git-workflow.md` ·
     `ci-cd.md` · `hooks.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md` ·
     `tooling.md` · `elicitation.md`.
   - **Index:** `README.md` (phase mapping, loading model).

State lives in `.devrites/` as human-readable Markdown so it survives context
compaction and new sessions. The optional `.devrites/AFK` sentinel toggles
the session-level run mode (see "Run modes" below). See `usage.md` for the
workspace file list, and [`command-map.md`](command-map.md) for the full
per-skill catalog with triggers + I/O.

## Design rationale

### Why the engine owns shared orientation (`devrites-lib`)
Every workspace-operating skill's first move is the same: orient on the active feature —
slug, phase, artifacts present, run mode, open-question tally — before acting. Re-deriving
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
different project-local conventions. Plugin packaging can layer on later, but it is a
distribution option, not the current source of install semantics.

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
`/rite` is **only** an entrypoint: it shows a compact phase-grouped menu, prints live
`.devrites/ACTIVE` status via `` !`cmd` `` injection, and suggests the next command.
It deliberately does **not** execute workflows or duplicate skill logic — that would
recreate the mega-command problem. DevRites keeps selection thin and lets each phase
own its context, rather than centralizing every workflow behind one command.

### Why internal skills exist
Specialist processes (doubt, source-driven, frontend craft, browser proof, audits)
are **disciplines**, not user commands. As `user-invocable: false` skills they:
- stay out of the command menu (less cognitive load);
- are invoked automatically by Claude or explicitly by a public skill when their
  trigger conditions hit;
- keep each public `SKILL.md` small by housing the heavy process elsewhere.

### Why spec, architecture, plan, tasks, and traceability are separate artifacts
Doing investigation, spec, plan, and slicing in one batch lets things slip — a gap goes
unasked, placement is guessed, a slice misses a requirement. DevRites splits them so each
is focused and gated. `/rite-spec` **investigates deeply and writes `spec.md`** (product
what/why, requirements, acceptance, non-goals, measurable success) and must pass its
readiness gate. `/rite-define` then turns that **approved spec** into `architecture.md`
(technical map), `plan.md` (approach), `tasks.md` (vertical `SLICE-###` work), and
`traceability.md` (AC/REQ → slice → proof → evidence → files). The spec is fully covered
before any building begins, but it does not become a long technical omnibus. `/rite-plan`
is the separate repair/reslice/re-order tool for an *active* plan when it goes stale or
drifts.

### Why `/rite-polish` is one skill with two progressive-disclosure halves
Polish has two natural halves — **code** (simplify, dead code, naming, plus
backend if BE was touched) and **UI** (normalize to the design system, then
ship-quality detail). The earlier design split them into `rite-polish-code`
and `rite-polish-ui` sub-skills, but a skill-on-skill dispatch is fragile
(the caller has to re-discover the callee by description) and the two halves
share the same operating rules. They now live as `reference/code.md` and
`reference/ui.md` inside the same skill, loaded only when their phase
trigger fires — the canonical progressive-disclosure pattern from
<https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices>.
`/rite-polish` is the orchestrator: always reads `reference/code.md`
(Phase 1 + 2), detects UI scope from the diff and reads `reference/ui.md`
(Phase 3 + 4) when needed. The orchestrator accepts mode tokens (`bolder`,
`quieter`, `distill`, `harden`, `normalize-only`) that pass through to
Phase 4.

Normalization remains the entry gate of UI polish — polishing UI that hasn't
been aligned to the project's design system is "decoration on drift," and
the UI reference refuses to run Phase 4 before Phase 3.

### Why seal and ship are separate phases (`/rite-seal`, `/rite-ship`)
Deciding "is this safe to ship" and *actually shipping* are different acts with
different blast radii. `/rite-seal` is a **pure decision gate**: it walks acceptance
against evidence, fans out the fresh-context reviewers, and writes the GO / NO-GO
verdict to `seal.md` — it runs no git. On GO it sets `state.md` `Next step:
/rite-ship` and stops. `/rite-ship` is the eighth lifecycle phase: it refuses to run
without a GO recorded in `seal.md`, renders the type-`GO` prompt, runs the irreversible
git ladder (commit → push → tag/PR per the project's convention), writes `ship.md`,
then **closes the task** — sets phase `done` and archives `.devrites/work/<slug>/` →
`.devrites/archive/<slug>/` (every `.md` preserved, never deleted) and clears
`.devrites/ACTIVE`. A GO seal is a verdict, not an authorization to push; keeping the
decision and the irreversible action as two separately-auditable steps is the point.

### Why `/rite-autocomplete` exists (the unattended orchestrator)
Some features are routine enough to run end-to-end without per-phase human iteration.
`/rite-autocomplete` drives the whole lifecycle (spec → define → build×N → prove →
polish → review → seal → ship) by reading each phase's `SKILL.md` and executing its
workflow, carrying state through the workspace files rather than chat. A vague prompt
triggers an up-front `devrites-interview` — the only interactive window — after which it
runs unattended, choosing the best option at each soft gate and recording the rationale
in `decisions.md`. It does **not** weaken the safety gates: hard irreversible-risk
(auth / migration / public-API / red tests), blocking / escalating gates, an open
`gate: validating`, a NO-GO, exhausted `max_slices`, or low confidence all still pause.
By default it stops at the final type-`GO`; the `--ship` flag (alias `--yolo`)
auto-confirms it for a zero-touch push.

### Why persistent `.devrites/` state
Long features outlive a single context window. Durable Markdown (spec/plan/tasks/
state/evidence/drift/decisions) lets any phase — in any session, after any compaction
— reload exactly where work stands. This is the main thing DevRites adds over typical
session-scoped workflows, which don't persist feature state.

### Run modes — HITL & AFK

DevRites runs the same lifecycle two ways, configured at two levels:

- **Per-slice** (planning-time) — each `tasks.md` slice declares `Mode: AFK | HITL`.
  HITL slices add `Gate: advisory | validating | blocking | escalating`, `SLA`, and
  `Checkpoint` so the agent knows how disruptive the pause should be.
- **Per-session** (run-time) — the presence of `.devrites/AFK` flips the session-level
  default. Empty file = AFK with safe defaults; YAML body widens behavior
  (`max_slices`, `notify`, `allow_gates`).

The pause primitive is a **pre-action interrupt**, not a post-action review queue —
inspired by LangGraph's `interrupt()` / `Command(resume=)` model. `/rite-build` writes
an `Awaiting human` block to `state.md` and a question to `questions.md`, then stops.
`/rite-resolve` is the canonical resume verb. Restarting a session reads the workspace
back into a consistent state because the pause is durable Markdown, not chat memory.

Why a four-gate taxonomy (instead of a single "ask the user" pause): a single gate
becomes a queue under load. Mixing `advisory` (audit-only log) with `validating`
(async — build continues, merge blocks) and reserving `blocking` for synchronous halts
keeps the loop alive when the answer can wait, and pauses hard when it can't. The
"AFK never silently accepts irreversible risk" rule — destructive migrations,
auth/authz boundaries, public API breaks, red tests/types/lint — always pauses
regardless of the sentinel. See
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](../pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md) for the full
contract.

## Design choices at a glance

- **Surface**: 25 public `rite-*` skills (38 total) — the thin `/rite` menu
  (carries the routing) + 8 lifecycle phases (`rite-spec`, `rite-define`,
  `rite-build`, `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`,
  `rite-ship` — seal **decides**, ship **executes + closes**) + the
  `rite-temper` (strategic, optional) and `rite-vet` (engineering, every plan)
  reviews + the `rite-quick` express lane and `rite-frame` pre-flight/self-audit
  lens + `rite-adopt` (onboard an existing codebase) + `rite-learn` (cross-feature
  lessons) + `rite-status` + `rite-doctor` (install health) +
  the `rite-plan` replan verb + the `rite-resolve` resume verb + 4 ideation /
  handoff utilities (`rite-zoom-out`, `rite-prototype`, `rite-handoff`,
  `rite-pressure-test`) + `rite-autocomplete` (the unattended full-lifecycle
  orchestrator) — plus 11 internal model-invoked `devrites-*` specialists and the
  `devrites-lib` library, not one mega-command. The `devrites-` prefix is a
  namespace (collision avoidance), not a public/internal marker —
  `user-invocable:` is. All `devrites-*` skills are model-invoked.
- **Selection**: the `/rite` menu skill carries the routing table; every
  workflow skill enforces a "right skill, right time" rule in its body.
- **State**: durable `.devrites/` Markdown that survives compaction and new sessions.
  `.devrites/AFK` presence is the single source of truth for run mode (HITL/AFK);
  there is no `state.md` run-mode field to drift out of sync.
- **Run modes**: same lifecycle runs HITL (default; pause at typed gate) or AFK
  (drop `.devrites/AFK`; loop continues, discretionary pauses downgrade to
  advisory log, irreversible risk always pauses).
- **Slice rule**: build **one vertical slice, then stop** — no auto-continue.
- **Drift**: an explicit **Spec Drift Guard** in build/prove/polish/review/seal.
- **Design**: `devrites-frontend-craft` + a four-phase `/rite-polish` orchestrator (code + backend always; UI normalize + polish when UI is in scope).
- **Review**: **feature-scoped** five-axis review with severity labels + fresh-context
  subagents at the seal.
- **Scope**: clarify → seal (decide) → ship (the irreversible git ladder —
  commit → push → tag/PR — per the project's own convention) → close; the CI
  pipeline stays with the project.
- **Install**: project-local, manifest-managed; ships DevRites' own engineering rules.

## Deviations from the original build brief (and why)

1. **`user-invocable` semantics corrected from the docs.** The brief said set
   `user-invocable: true` for public and `false` for internal. That is exactly right
   for internal skills. For public skills we keep them model-invocable (no
   `disable-model-invocation`) so phases can hand off and the selector can route;
   per-phase side-effect discipline (e.g. "stop after one slice") is enforced in the
   skill **body**, which is the correct mechanism (invocation flags can't express it).
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
Source pack lives under `pack/.claude/` (skills, agents, and rules); the installer copies
it into a target project's `.claude/`, including DevRites' engineering rules in
`.claude/skills/devrites-lib/reference/standards/`.
