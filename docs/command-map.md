# DevRites Command Map

Reference for every shipped skill and agent — what triggers it, what it reads,
what it writes, and how the pieces interact.

- **Workflow diagram** (top-level flow) → [`flow.md`](flow.md).
- **Architecture rationale** → [`architecture.md`](architecture.md).
- **Worked examples** → [`usage.md`](usage.md).

## Naming convention

`devrites-` is a **namespace prefix** chosen for collision avoidance against
bundled Claude Code skill names (`prototype`, `handoff`, `triage`, `diagnose`,
…). It does **not** signal "internal" — visibility is governed by the
`user-invocable:` flag in each `SKILL.md`. Three `devrites-*` skills are
public (`zoom-out`, `prototype`, `handoff`); the rest are model-invoked.

## Public commands (`user-invocable: true`)

| Command | Phase | Argument | What it does | Reads | Writes |
|---|---|---|---|---|---|
| [`/rite`](../pack/.claude/skills/rite/SKILL.md) | menu | `[subcommand]` | Compact menu + suggested next command. Pure router; does **not** read state — that's `/rite-status`. | — | — |
| [`/rite-spec`](../pack/.claude/skills/rite-spec/SKILL.md) | spec | `<feature>` | **Start here.** Deep investigation → writes `spec.md` (placement, what-it-resolves, gaps closed with options, design references). Creates the workspace. | codebase + codegraph/graphify | `spec.md`, `references/`, `references.md`, `brief.md`, `questions.md`, `decisions.md`, `assumptions.md`, `state.md` |
| [`/rite-define`](../pack/.claude/skills/rite-define/SKILL.md) | plan | `[slug]` | Turns the approved `spec.md` into plan + vertical task slices + state. | `spec.md` + references | `plan.md`, `tasks.md`, `state.md`, `decisions.md` |
| [`/rite-plan`](../pack/.claude/skills/rite-plan/SKILL.md) | plan | `[mode]` | Decompose / reslice / repair / re-order / split / unblock an active plan. | spec/plan/tasks/state/drift + diff | `plan.md`, `tasks.md`, `state.md`, `decisions.md` |
| [`/rite-build`](../pack/.claude/skills/rite-build/SKILL.md) | build | `[slice]` | Implement **exactly one** vertical slice, then stop. | workspace + diff | code + `state.md`, `evidence.md`, `touched-files.md` |
| [`/rite-prove`](../pack/.claude/skills/rite-prove/SKILL.md) | prove | `[scope]` | Tests + build + runtime + browser proof of the completed feature. | workspace + diff | `evidence.md`, `browser-evidence.md`, `state.md` |
| [`/rite-polish`](../pack/.claude/skills/rite-polish/SKILL.md) | polish | `[target \| mode]` | Orchestrator. Reads `reference/code.md` always (Phase 1 + 2); reads `reference/ui.md` when UI is touched (Phase 3 + 4). Mode tokens: `bolder \| quieter \| distill \| harden \| normalize-only`. | workspace + design system + diff | `polish-report.md`, `browser-evidence.md` |
| [`/rite-review`](../pack/.claude/skills/rite-review/SKILL.md) | review | `[scope]` | Feature-scoped multi-axis review. Parallel Spec + Standards sub-agents (`devrites-spec-reviewer`, `devrites-code-reviewer`). | workspace + diff | `review.md`, `evidence.md`, `state.md` |
| [`/rite-seal`](../pack/.claude/skills/rite-seal/SKILL.md) | seal | — | Final GO / NO-GO. Walks acceptance, fans out reviewers, type-`GO` before irreversible git. | all artifacts + diff | `seal.md` |
| [`/rite-status`](../pack/.claude/skills/rite-status/SKILL.md) | status | `[slug]` | Active feature: phase, run mode (AFK/HITL), status, next action, evidence, open questions by gate, risks, handoff readiness. Reads via `scripts/load-state.sh` (portable). | workspace + `.devrites/AFK` | — |
| [`/rite-resolve`](../pack/.claude/skills/rite-resolve/SKILL.md) | resume | `<qid> "<answer>"` \| `--drop <qid>` \| `--batch <file>` | Answer / drop / batch-resolve open `questions.md` entries; clears `state.md` `Awaiting human` and sets `Status: running`. Canonical writer for `status: open → answered`. | `questions.md` + `state.md` | `questions.md`, `state.md` |
| [`/rite-zoom-out`](../pack/.claude/skills/rite-zoom-out/SKILL.md) | utility | — | One-pass structural map of an unfamiliar area (modules, in-callers, out-calls, decisions) in project vocabulary. Prefers codegraph/graphify. | codebase + ADRs/CONTEXT.md | — |
| [`/rite-prototype`](../pack/.claude/skills/rite-prototype/SKILL.md) | utility | `[question]` | Throwaway code answering ONE design question. Logic harness OR 2–4 UI variations on one route. Captures verdict to `decisions.md`. | spec / surrounding code | prototype scratch + `decisions.md` |
| [`/rite-handoff`](../pack/.claude/skills/rite-handoff/SKILL.md) | utility | `[next-session-focus]` | Compacts the chat into a handoff doc. Syncs chat-only context into workspace canonical files. References existing artifacts by path. | chat + workspace | `handoff.md` + sync into canonical files |
| [`/rite-pressure-test`](../pack/.claude/skills/rite-pressure-test/SKILL.md) | utility | `[idea]` | Pressure-test a rough idea: 3–5 genuinely different options → converge on one with trade-off + hinge. | spec / surrounding code | `decisions.md` (optional) |

## Internal skills (`user-invocable: false`, model-invoked)

| Skill | Triggered by | Role | Notable |
|---|---|---|---|
| [`devrites-interview`](../pack/.claude/skills/devrites-interview/SKILL.md) | `/rite-spec`, underspecified ask | One-Q-at-a-time protocol | best-guess + confidence stop |
| [`devrites-source-driven`](../pack/.claude/skills/devrites-source-driven/SKILL.md) | uncertain framework/library fact | Consult docs/source, record citation | writes `evidence.md` / `decisions.md` |
| [`devrites-doubt`](../pack/.claude/skills/devrites-doubt/SKILL.md) | non-trivial decision in build/review | CLAIM → EXTRACT → DOUBT → RECONCILE → STOP | adversarial; ask user if uncertain |
| [`devrites-frontend-craft`](../pack/.claude/skills/devrites-frontend-craft/SKILL.md) | UI detected in build/polish | Register detection, shape-before-code, states, anti-slop | refs: shape/craft/design-references |
| [`devrites-browser-proof`](../pack/.claude/skills/devrites-browser-proof/SKILL.md) | UI in prove/polish | Browser proof ladder + evidence schema | harness preferred |
| [`devrites-debug-recovery`](../pack/.claude/skills/devrites-debug-recovery/SKILL.md) | failing tests/build/runtime | 6-phase: loop → reproduce → hypotheses → instrument → fix → cleanup | references split per phase |
| [`devrites-api-interface`](../pack/.claude/skills/devrites-api-interface/SKILL.md) | cross-boundary slice | Stable API/contract design | FE/BE split |
| [`devrites-audit simplify`](../pack/.claude/skills/devrites-audit/SKILL.md) | `/rite-polish` Phase 1 | Chesterton's Fence, behavior-preserving simplification | dispatches `devrites-simplifier-reviewer` |
| [`devrites-audit security`](../pack/.claude/skills/devrites-audit/SKILL.md) | input/auth/data/integration in scope | OWASP Top 10, three-tier boundary | dispatches `devrites-security-auditor` |
| [`devrites-audit perf`](../pack/.claude/skills/devrites-audit/SKILL.md) | perf relevant or regression risk | Measure-first, CWV targets | dispatches `devrites-performance-reviewer` |
| [`reference/parallel-dispatch.md`](../pack/.claude/skills/rite-seal/reference/parallel-dispatch.md) (sibling in `rite-review/reference/`) | loaded inline by `/rite-seal` and `/rite-review` | Reference doc — dispatch shape + reconciliation rules for the parallel reviewer fan-out via the `Task` tool | not a skill — a reference file |

## Agents (`.claude/agents/devrites-*`, fresh-context reviewers)

| Agent | Spawned by | Purpose |
|---|---|---|
| [`devrites-spec-reviewer`](../pack/.claude/agents/devrites-spec-reviewer.md) | `/rite-review` Spec axis; `/rite-seal` | Does the diff implement the spec? Missing/partial/wrong criteria; scope creep |
| [`devrites-code-reviewer`](../pack/.claude/agents/devrites-code-reviewer.md) | `/rite-review` Standards axis; `/rite-seal` | Correctness / readability / architecture / maintainability |
| [`devrites-test-analyst`](../pack/.claude/agents/devrites-test-analyst.md) | `/rite-seal` | Do the tests actually prove the acceptance criteria? |
| [`devrites-frontend-reviewer`](../pack/.claude/agents/devrites-frontend-reviewer.md) | `/rite-seal` on UI features | UX, a11y, responsive, design-system, anti-AI-slop |
| [`devrites-security-auditor`](../pack/.claude/agents/devrites-security-auditor.md) | `/rite-seal` when input/auth/data in scope | OWASP Top 10, trust boundary, secrets, deps |
| [`devrites-performance-reviewer`](../pack/.claude/agents/devrites-performance-reviewer.md) | `/rite-seal` when perf relevant | N+1s, hot paths, payload size |
| [`devrites-doubt-reviewer`](../pack/.claude/agents/devrites-doubt-reviewer.md) | `devrites-doubt` loop | Adversarial check of a single claim/decision |
| [`devrites-simplifier-reviewer`](../pack/.claude/agents/devrites-simplifier-reviewer.md) | `devrites-audit simplify` | Independent simplification judgment |

## Engineering rules (`pack/.claude/rules/`)

Progressive-disclosure rules. `core.md` always loads; the rest are referenced
on demand. Full index in [`pack/.claude/rules/README.md`](../pack/.claude/rules/README.md).

- `core.md` (always-on) — operating rules + universal anti-rationalizations + 1-line craft disciplines + persistence-before-stopping summary.
- `coding-style.md` · `error-handling.md` · `testing.md` · `code-review.md` · `security.md` · `performance.md` · `patterns.md` · `git-workflow.md` · `hooks.md` · `documentation.md` · `development-workflow.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md`
- `anti-patterns.md` — pack-wide rationalizations + red flags. Loaded by each per-phase `rite-*/reference/anti-patterns.md`; can be loaded directly for cross-phase reluctance.

## Trigger conditions (auto-selection)

| Trigger | Routes to |
|---|---|
| Frontend/UI detected (TSX/JSX/Vue/Svelte/Astro/Angular/ERB, CSS/Tailwind/tokens, components/forms/states) | `devrites-frontend-craft` in build, `devrites-browser-proof` in prove, `rite-polish` Phase 3 + 4 (`reference/ui.md`) in polish, UX/a11y axes at review/seal |
| Uncertain library / framework behavior | `devrites-source-driven` |
| Non-trivial decision (boundary, data model, auth, public API, migration, "this scales/safe") | `devrites-doubt` (+ `devrites-doubt-reviewer`) |
| User input / auth / storage / external integration / secrets / permissions | `devrites-audit security` (+ `devrites-security-auditor`) |
| Performance requirement or suspected regression | `devrites-audit perf` (+ `devrites-performance-reviewer`) |
| Failing tests / build / runtime / browser checks | `devrites-debug-recovery` |
| Slice crosses a boundary or defines a public interface | `devrites-api-interface` |
| Unfamiliar area, "zoom out", "map this" | `/rite-zoom-out` (uses codegraph/graphify) |

## Code-graph integration

Skills that prefer a code-intelligence index (`codegraph_*` / `graphify-out/`)
when available, falling back to file reads otherwise:

- `/rite-spec`, `/rite-define`, `/rite-plan` — placement / impact / callers during investigation
- `/rite-build` — `touched-files.md` + impact when loading slice context
- `/rite-review` — blast-radius checks on the diff
- `/rite-seal` — final blast-radius check
- `/devrites-doubt` — "where does this claim reach" via `codegraph_impact` / `codegraph_callers`
- `/rite-zoom-out` — `codegraph_context` + `codegraph_explore`
- `/devrites-frontend-craft` — component / token lookups

## Interactions (typical flow)

See [`flow.md`](flow.md) for the Mermaid diagrams. The text path:

```
/rite-spec → /rite-define → /rite-build ×N → /rite-prove → /rite-polish → /rite-review → /rite-seal
   │            │                │  ▲              │                          │
   │            │                │  └ Spec Drift Guard → /rite-plan repair ────┘
   │            │                └ devrites-frontend-craft / source-driven / doubt
   └ (no workspace) → summary    devrites-* internal skills fire on triggers above
```

- Every phase **reads the active workspace first**; if none, it stops and tells
  the user to run `/rite-spec <feature>`.
- **Spec Drift Guard** lives in build/prove/polish/review/seal: on drift,
  stop, record in `drift.md`, classify, ask the user if product behavior
  changes, then `/rite-plan repair` before resuming.
- `/rite-seal` fans out to `.claude/agents/devrites-*` reviewers **in
  parallel** for independent, fresh-context judgment.
