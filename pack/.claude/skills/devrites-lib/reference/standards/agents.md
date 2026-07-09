# Agent orchestration

DevRites uses **project-local** agents under `.claude/agents/` (never a global location).
It separates three roles: **specialist skills** (model-invoked disciplines that run inline),
**review subagents** (fresh-context, **read-only** reviewers spawned for independent judgment),
and the **executor subagent** (`devrites-slice-wright` — fresh-context but **write-capable**,
the one agent that produces code).

## Model tier policy

Use [`model-tiers.md`](../model-tiers.md) when a harness supports model choice. Wright defaults to
standard/inherited, with the strongest available model for high-risk architecture, migration, auth,
or public API slices; cheap tiers are only for mechanical read-only scouts. Seal/spec/security
reviewers stay strongest/inherited. If the harness cannot select models, record "host default".

## Review subagents — `.claude/agents/`
Fresh-context, read-only reviewers. Each is given the active feature workspace path
(`.devrites/work/<slug>/`) + the diff, and returns findings — they do **not** edit code.

**Which reviewer fires when is defined once** in
[`parallel-dispatch.md`](../../../devrites-lib/reference/parallel-dispatch.md) — the seal/review
fan-out **roster** and its checkable triggers, plus the accounting gate that proves no reviewer is
silently skipped. The table below is the full catalog (it also covers the phase-locked agents the
roster excludes); its **When** column is a summary — on any discrepancy, the roster wins.

| Agent | Purpose | When |
|-------|---------|------|
| `devrites-spec-reviewer` | Does the diff implement the spec? Missing / partial / wrong criteria; scope creep | `/rite-review` (Spec axis); `/rite-seal` |
| `devrites-code-reviewer` | Correctness / readability / architecture / maintainability | `/rite-seal`; after a slice |
| `devrites-test-analyst` | Do the tests actually prove the acceptance criteria? | `/rite-seal` |
| `devrites-frontend-reviewer` | UX flow, a11y, responsive, design-system, anti-AI-slop | `/rite-seal` on UI features |
| `devrites-security-auditor` | Untrusted input, trust boundaries, secrets, deps | `/rite-seal` when input/auth/data in scope |
| `devrites-performance-reviewer` | Measure-first perf (N+1, hot paths, payload size) | `/rite-seal` when perf relevant |
| `devrites-devex-reviewer` | Developer-experience scorecard + the predict-vs-measure boomerang (TTHW, getting-started, error-message quality, ergonomics, docs) | `/rite-vet` (predict) + `/rite-seal` when a developer-facing surface (API / CLI / SDK / webhook / config / errors / getting-started) is in scope |
| `devrites-simplifier-reviewer` | Behavior-preserving simplification (Chesterton's Fence, deletion test) — Suggestion-only by design | `/rite-polish` Phase 1; `devrites-audit simplify` |
| `devrites-doubt-reviewer` | Adversarial check of a single claim/decision | `devrites-doubt` loop; risky decisions |
| `devrites-strategy-reviewer` | Spec-vs-rubric strategic review (ambition / scope / premise / pre-mortem / YAGNI / testability / irreversibility / cross-cutting / convention) — **before** any plan or code | `/rite-temper` loop (pre-plan) |
| `devrites-plan-reviewer` | Plan-vs-rubric engineering review (architecture / scope-reuse / plan code-quality / test-coverage design / performance / reversibility / failure-mode coverage), confidence-banded with a quote-the-source verification gate — **after define, before build** | `/rite-vet` loop (pre-build) |
| `devrites-forge-judge` | Comparative judge of K=2–3 competing candidate implementations of one slice (acceptance / test strength / principle fit / simplicity / reuse / anti-slop) — picks the single winner to land, names grafts | `/rite-build` forge step, on a `Forge: yes` slice |

## The executor subagent — `.claude/agents/devrites-slice-wright.md`

The system's one **write-capable** agent, and the mirror of the reviewers: where a reviewer
reads a finished diff in a fresh context, the wright **writes** one slice in a fresh context.

| Agent | Purpose | When |
|-------|---------|------|
| `devrites-slice-wright` | Turn ONE fully-specified slice contract into the smallest complete, idiomatic, proven implementation — orient → TDD → verify, anti-AI-slop, feature scope only | `/rite-build` — the build-core dispatch step |

`/rite-build` is the orchestrator: it owns the gates and the workspace, dispatches the wright
for the implementation, then doubts, gates, and records the return. Tools:
`Read, Edit, Write, Bash, Glob, Grep` (+ a code-intelligence index when present). It writes
**code and tests only** — never the `.devrites/` bookkeeping files; it returns that data and
the orchestrator persists it, so there is exactly one canonical writer of workspace state and
the HITL/AFK contract stays intact. **Single-threaded: one wright per slice, never a parallel
fan-out of writers *sharing a tree*** (concurrent writers on one tree make conflicting implicit
decisions). The one sanctioned exception is a **forge** slice (`Forge: yes`, flagged by
`/rite-vet`): K=2–3 candidate wrights build the slice on distinct strategies in **isolated**
worktrees, `devrites-forge-judge` scores them, and exactly **one** winner's diff lands — no tree
ever has two authors, so the invariant holds. Contract + return shape + fallback + forge:
`.claude/skills/rite-build/reference/wright-dispatch.md`, `.../reference/forge.md`.

## The cross-feature analyst — `.claude/agents/devrites-retrospector.md`

A read-only agent like the reviewers, but its scope is the **archive, not a diff**: it reads the
shipped `.devrites/archive/<slug>/` workspaces and reports the patterns one feature can't show —
a finding class reviewers keep raising, recurring drift, dead-ends, the GO/NO-GO and rework trend.

| Agent | Purpose | When |
|-------|---------|------|
| `devrites-retrospector` | Cross-feature retrospective synthesis — mine recurring patterns + trends across shipped features, **draft** graduation candidates (rule / principle / convention / dismiss) | `/rite-ship` close, cadence-gated by `devrites-engine learnings nudge` (fires only with enough cross-feature signal) |

It **proposes, never imposes**: it writes nothing itself — the caller (`/rite-ship`) persists the
digest to `.devrites/retro.md` and routes any promotion through `/rite-learn`, where the human
confirms. This makes the cross-feature learning that otherwise waits for someone to run `/rite-learn`
fire automatically at close, while keeping rule/principle promotion a deliberate human amendment
(`principles.md` governance). The capture half is already automatic (`/rite-seal` step 9a); this is
the synthesis half.

## Namespaces — `rite-*` is the user surface; `devrites-*` is internal

The prefix mirrors visibility:

- **`rite-*`** = user-invocable (`user-invocable: true`). The public slash-command
  surface — lifecycle phases plus utilities. `rite`, `rite-spec`, `rite-temper`,
  `rite-define`, `rite-vet`,
  `rite-plan`, `rite-build`, `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`,
  `rite-ship`, `rite-autocomplete`, `rite-status`, `rite-resolve`, `rite-prototype`,
  `rite-handoff`, `rite-zoom-out`, `rite-pressure-test`. `/rite-seal` **decides**
  GO/NO-GO and writes the verdict; `/rite-ship` **executes** the irreversible git
  ladder and **closes** the task (archives the workspace, clears `.devrites/ACTIVE`).
  `/rite-autocomplete` drives the whole lifecycle unattended.
- **`devrites-*`** = model-invoked (`user-invocable: false`, no slash command).
  Internal specialists that fire on trigger from the `rite-*` skills or auto-select.
  `devrites-interview`, `devrites-source-driven`, `devrites-doubt`,
  `devrites-ux-shape` (plans UX/UI into `design-brief.md` at `/rite-spec` when UI is
  detected — the build target), `devrites-frontend-craft`, `devrites-prose-craft`
  (human-voice prose for artifacts + replies; the catch pass in `/rite-polish`),
  `devrites-browser-proof`,
  `devrites-debug-recovery`, `devrites-api-interface`, `devrites-audit` (dispatches the
  security / perf / simplify reviewer subagent on an axis argument).
  The `devrites-` prefix avoids collisions with bundled Claude Code skill
  names (`prototype`, `handoff`, `diagnose`, etc.) — peer skill packs may
  collide on the bare names internally even though these never appear in the
  user's slash menu. Parallel reviewer fan-out at `/rite-seal` is no longer
  a skill — it lives as a shared reference file
  (`.claude/skills/devrites-lib/reference/parallel-dispatch.md`), pointed at by both
  `/rite-review` and `/rite-seal`.

The `/rite` menu carries user-facing routing. `user-invocable:` in each
`SKILL.md` is the source of truth; the prefix is a naming convention that matches it.

## When to bring in a reviewer (no prompt needed)
The seal/review fan-out is **roster-driven**: sealing a feature dispatches every roster reviewer
whose trigger the diff meets (frontend on UI, security on input/auth/data, performance on a perf
budget or hot-path, devex on a developer-facing surface) and accounts for the rest — the triggers
and the no-silent-skip gate are the single source in
[`parallel-dispatch.md`](../../../devrites-lib/reference/parallel-dispatch.md). Three reviewers
fire **outside** that fan-out:
1. Standing a non-trivial decision (boundary, data model, auth, public API, migration) →
   `devrites-doubt` → `devrites-doubt-reviewer`.
2. A developer-facing surface also gets `devrites-devex-reviewer` **at `/rite-vet` to predict** the
   DX scorecard — the seal side then measures and reconciles the boomerang (`developer-experience.md`).
3. Closing a feature at `/rite-ship`, cadence-gated → `devrites-retrospector` (below). Not a per-diff
   reviewer: it fires only when there's enough cross-feature signal to mine.

## How skills reach an agent (both harnesses)

A spawned agent runs in a **fresh context** and does **not** auto-discover skills the way the main
thread does. Naming a skill in an agent's body ("apply `devrites-frontend-craft`") is a prompt
reference, not a load — the agent must actually **invoke** the skill, and how it can differs by
harness:

- **Claude Code** gates skills behind the agent's `tools:` allowlist. An agent that lists `tools:`
  without `Skill` **cannot invoke any skill**. Two ways to grant access: add **`Skill`** to `tools:`
  (on-demand invocation — right for the stack-agnostic wright, so a backend slice pays no cost), or a
  **`skills:`** frontmatter field (preloads the full skill content at startup — right for a focused
  reviewer that *always* needs it, e.g. `devrites-frontend-reviewer` preloads `devrites-frontend-craft`).
- **Codex** does not gate skills per-agent: skills are auto-discovered from `.agents/skills/` and
  subagents inherit that access. So the same agent body that names the skill lets a Codex agent invoke
  it (`$devrites-frontend-craft`, or implicitly). The Claude-only `skills:`/`Skill` metadata is dropped
  in translation — do **not** map it to Codex `skills.config`, which *disables* skills, not preloads them.

The durable, cross-harness lever is the **agent body**: name the skill and instruct the agent to
*invoke* it (carried to Codex via the installer's `developer_instructions`). The `SubagentStart` hook
(`devrites-engine hook subagent-orient`, Claude) and `AGENTS.md` (Codex) inject the operating-rule discipline
into every DevRites subagent, since a fresh subagent doesn't run the rite-\* skills that load the rules.

**Reaching every agent, not just one skill.** A fresh subagent doesn't load the `.claude/skills/devrites-lib/reference/standards/`
framework either — so a reviewer whose whole job is a rule (`security.md`, `performance.md`,
`testing.md`, `code-review.md`, …) must **Read that rule file at the start of its review**, the
rule-analog of invoking a skill. Without it the reviewer judges from a remembered summary and misses
every sharpening the rule has grown. So the discipline is uniform across the fan-out:

- **wright** → reads its in-scope rules (`coding-style`, `testing`, `error-handling`, `patterns`,
  `security`/`performance` when relevant) **and** invokes its craft skills on demand
  (`devrites-frontend-craft` / `-source-driven` / `-api-interface`).
- **frontend reviewer** → preloads the `devrites-frontend-craft` **skill** (its axis has one).
- **every other reviewer** → Reads its governing **rule** first (code-reviewer→`code-review`+`coding-style`
  +`patterns`; security→`security`; performance→`performance`; test→`testing`; spec→`spec-grammar`;
  simplifier→`coding-style`+`patterns`; devex→`developer-experience`; forge-judge→`coding-style`+`testing`
  +`principles`). Promote a reviewer to a preloaded skill only when a canonical one appears for its axis.

The rule paths are written as `.claude/skills/devrites-lib/reference/standards/<x>.md`; the installer rewrites them to
`.agents/skills/devrites-lib/reference/standards/<x>.md` for Codex, so one body line serves both harnesses.

## Rules
- Run independent reviewers **in parallel** at the seal, then reconcile; surface
  disagreements explicitly rather than averaging them away.
- Reviewer read-only is **wired at the tool layer**: each reviewer carries a shared
  deny-mutating-Bash hook (`devrites-engine hook reviewer-readonly`) — via subagent frontmatter on Claude
  Code (project-local install), and globally with agent-type gating in `.codex/hooks.json` on
  Codex. It runs **observe-by-default** (logs a would-block) and **denies** under
  `DEVRITES_REVIEWER_RO=enforce`. A fresh-context reviewer reads untrusted source — a silent write
  path would be a prompt-injection surface, so enable enforce once the log shows no false positives.
- **Reviewer** agents are **read-only** and return labeled findings (Critical / Important /
  Suggestion / Nit / FYI). Keep review **feature-scoped**.
- The **executor** agent (`devrites-slice-wright`) is the one **write-capable** agent: it writes
  code + tests for a single slice and returns a structured artifact, but it never writes the
  `.devrites/` workspace files (the orchestrator does) and never runs in parallel with another
  writer.
- Give each agent the contract (workspace + diff for a reviewer; the slice contract for the
  wright) without the author's reasoning — fresh, undirected context is the point.
- **Orchestration depth ≤ 1 — agents don't invoke agents.** The `rite-*` skill (driven by the
  user) is the orchestrator; it dispatches reviewers and the wright, collects their findings, and
  reconciles. A reviewer never spawns another reviewer, and the wright never spawns a sub-writer —
  the fan-out is one level, always from the orchestrator. Four shapes this rules out: a
  "meta-orchestrator" agent whose only job is routing (that's the skill's job), an agent that calls
  a second agent, a sequential chain where each agent paraphrases the last one's hand-off (lossy —
  the orchestrator passes the real artifact, not a summary), and deep agent trees. Parallel fan-out
  is many dispatches in **one** orchestrator turn, not nested ones; on Claude Code the platform
  also enforces this (a subagent can't spawn subagents), so the invariant and the runtime agree.
