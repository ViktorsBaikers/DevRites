# DevRites Architecture

DevRites is a **distributed but coordinated** set of project-local Claude Code skills
that make an AI coding agent behave like a disciplined senior engineer: clarify →
spec+plan → build one verified slice → prove with evidence → polish → review → seal.

## Layers

1. **Public workflow skills** — `.claude/skills/rite-*`, `user-invocable: true`. Each
   owns exactly one phase and reads/writes the `.devrites/` workspace.
   Sequence: `rite-spec`, `rite-define`, `rite-plan`, `rite-build`,
   `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`, plus the
   read-only `rite-status`, the resume verb `rite-resolve` (answer a HITL
   gate and clear `Awaiting human`), and the thin `/rite` menu.
2. **Public utility skills** — `.claude/skills/rite-zoom-out`,
   `rite-prototype`, `rite-handoff`, and `rite-pressure-test` —
   public commands. The `devrites-` prefix is **namespace** (collision
   avoidance against bundled Claude Code skill names like `prototype`,
   `handoff`, `triage`, `diagnose`), not a visibility marker;
   `rite-pressure-test` carries no prefix because it doesn't collide.
3. **Internal specialist skills** — `.claude/skills/devrites-*` with
   `user-invocable: false`: `devrites-interview`, `-source-driven`,
   `-doubt`, `-frontend-craft`, `-browser-proof`, `-debug-recovery`,
   `-api-interface`, `-audit` (dispatches the security / perf / simplify
   reviewer subagent on an axis argument). Model-invoked by public skills
   or Claude's auto-selection; not menu noise. Whether a skill is public or
   internal is governed by the `user-invocable:` flag, not by the name
   prefix.

   Engineering rules live at `.claude/rules/` (each `rite-*` skill Reads
   `.claude/rules/core.md` as its first step; the other 15 files load on
   demand — no session-start autoload). Parallel reviewer fan-out at
   `/rite-seal` is a reference file
   (`rite-seal/reference/parallel-dispatch.md`), not a skill.
4. **Supporting references** — `reference/*.md` inside each skill. Long checklists,
   templates, and anti-rationalization tables loaded on demand (progressive
   disclosure) so `SKILL.md` bodies stay small.
5. **Agents** — `.claude/agents/devrites-*` fresh-context reviewers used by `/rite-seal`
   and the doubt loop: `devrites-spec-reviewer`, `-code-reviewer`,
   `-test-analyst`, `-frontend-reviewer`, `-security-auditor`,
   `-performance-reviewer`, `-doubt-reviewer`, `-simplifier-reviewer`.
6. **Engineering rules** — DevRites' own stack-agnostic rules installed to
   `.claude/rules/`. Each `rite-*` skill Reads `core.md` as its first step
   (step 0); 15 on-demand files load by the phase that needs them:
   - **Craft:** `coding-style.md` · `patterns.md` · `error-handling.md` ·
     `testing.md` · `documentation.md`.
   - **Quality / safety:** `code-review.md` · `security.md` ·
     `performance.md` · `anti-patterns.md`.
   - **Workflow / ops:** `development-workflow.md` · `git-workflow.md` ·
     `hooks.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md`.
   - **Index:** `README.md` (phase mapping, loading model).

State lives in `.devrites/` as human-readable Markdown so it survives context
compaction and new sessions. The optional `.devrites/AFK` sentinel toggles
the session-level run mode (see "Run modes" below). See `usage.md` for the
workspace file list, and [`command-map.md`](command-map.md) for the full
per-skill catalog with triggers + I/O.

## Design rationale

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

### Why spec and plan are separate phases (`/rite-spec`, `/rite-define`, `/rite-plan`)
Doing investigation, spec, plan, and slicing in one batch lets things slip — a gap goes
unasked, placement is guessed, a slice misses a requirement. DevRites splits them so each
is focused and gated. `/rite-spec` **investigates deeply and writes `spec.md`** (what/why,
placement, issues, gaps closed with the user, design references) and must pass its
readiness gate. `/rite-define` then turns that **approved spec** into `plan.md` +
`tasks.md` (the how + vertical slices), checking every acceptance criterion maps to a
slice. The spec is fully covered before any planning begins. `/rite-plan` is the separate
repair/reslice/re-order tool for an *active* plan when it goes stale or drifts.

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
[`pack/.claude/rules/afk-hitl.md`](../pack/.claude/rules/afk-hitl.md) for the full
contract.

## Design choices at a glance

- **Surface**: 15 public `rite-*` skills — the thin `/rite` menu (carries the
  routing) + 8 lifecycle phases (`rite-spec`, `rite-define`, `rite-plan`,
  `rite-build`, `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`) +
  `rite-status` + the `rite-resolve` resume verb + 4 utilities
  (`rite-zoom-out`, `rite-prototype`, `rite-handoff`, `rite-pressure-test`) —
  plus 8 internal model-invoked `devrites-*` specialists, not one
  mega-command. The `devrites-` prefix is a namespace (collision avoidance),
  not a public/internal marker — `user-invocable:` is. All `devrites-*`
  skills are model-invoked.
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
- **Scope**: clarify → seal; git/CI stay with the project and Claude Code's built-ins.
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
`.claude/rules/`.
