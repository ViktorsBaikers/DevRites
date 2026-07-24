# DevRites command map

This page lists every shipped skill and agent, including its triggers, inputs,
outputs, and interactions.

DevRites ships through npm (`npx devrites ...`). Claude Code and Codex support
comes from generated host artifacts copied by the npm installer; DevRites is not
distributed as a Claude or Codex plugin.

- **Workflow diagram** (top-level flow) → [`flow.md`](flow.md).
- **Architecture rationale** → [`architecture.md`](architecture.md).
- **Worked examples** → [`usage.md`](usage.md).

## Naming convention

The `devrites-` prefix avoids collisions with bundled Claude Code skill names
such as `prototype`, `handoff`, `triage`, and `diagnose`. The prefix does not
mean that a skill is internal; each `SKILL.md` uses `user-invocable:` to set
visibility. Public utilities use the `rite-*` prefix: `rite-quick`,
`rite-frame`, `rite-adopt`, `rite-learn`, `rite-doctor`, `rite-customize`,
`rite-zoom-out`, `rite-prototype`, `rite-handoff`, `rite-pressure-test`,
`rite-autocomplete`, `rite-explain`, `rite-pov`, `rite-dogfood`, and
`rite-pr-feedback`. The host may invoke `devrites-*` specialists through the
model. `devrites-lib` is the non-workflow library exception.

## Surface lifecycle

- **Promoted:** shipped `rite-*`, `devrites-*`, generated host artifacts, and docs
  in this map.
- **Draft:** research or local experiments that are not copied into `pack/`.
- **Deprecated:** compatibility shims with a removal note and a replacement path.
- **Research:** source-intake notes under `docs/research/`; not installed.

Only promoted surfaces are shipped by the npm package.

## Engine command ownership

Not every engine command is a phase step. Assign each new command to one of
these lanes so its owner is clear:

| Lane | Commands | Owner |
|---|---|---|
| Workflow gates | `preamble`, `build-readiness`, `readiness-digest`, `spec-skeleton`, `spec-validate`, `check-acceptance`, `evidence-fresh`, `coverage`, `doubt-coverage`, `budget`, `test-integrity`, `mutation-gate`, `package-existence`, `review-integrity`, `footprint`, `reconcile`, `conventions`, `learnings`, `review-fingerprints`, `timeline`, `health`, `progress` | Called by the relevant `rite-*` workflow or shared reply contract. |
| Workspace utilities | `status`, `snapshot`, `analyze`, `archive-search`, `clarify-return`, `recovery`, `resolve`, `close-out`, `stuck`, `tick-afk`, `ledger`, `profile` | Called by a specific utility/phase when its condition is met. |
| Low-level completeness API | `readiness`, `seal` | Available for scripts/CI and documented engine use. Feature rites use stricter phase-specific gates (`build-readiness`, `/rite-seal` phase contract) instead of auto-running these weaker aggregate checks. |
| Install / operator / CI | `install`, `update`, `uninstall`, `doctor`, `migrate`, `validate-pack`, `harness-matrix`, `extensions`, `overrides`, `reviewers`, `hook`, `version` | Called by `npx devrites ...`, `/rite-doctor`, hooks, CI, or a human operator; do not auto-run during feature work just because the command exists. |

Workflow-owned commands should have a concrete call site in a `rite-*` skill,
phase contract, shared reply contract, or installed hook. Operator-owned commands
must say who runs them.

## Public commands (`user-invocable: true`)

| Command | Phase | Argument | What it does | Reads | Writes |
|---|---|---|---|---|---|
| [`/rite`](../pack/.claude/skills/rite/SKILL.md) | menu | `[subcommand]` | Compact menu + suggested next command. Pure router; does **not** read state because `/rite-status` owns that job. | none | none |
| [`/rite-spec`](../pack/.claude/skills/rite-spec/SKILL.md) | spec | `<feature>` | **New feature.** Deep investigation → writes a product-focused `spec.md` (WHAT/WHY, requirements, ACs, boundaries, gaps closed with options, design references). Checks the shipped archive for prior art before speccing. Creates the workspace map. | codebase + codegraph/graphify + shipped archive (`devrites-engine archive-search`) | `README.md`/`feature.md`, `brief.md`, `spec.md`, `references/`, `references.md`, `questions.md`, `decisions.md`, `assumptions.md`, `state.md` |
| [`/rite-clarify`](../pack/.claude/skills/rite-clarify/SKILL.md) | clarify | `[slug]` | **Required, adaptive.** Topology-first coverage scan of the written spec; searches facts, closes human-owned decisions, audits assumptions, and takes a zero-question fast path when already clear. Writes a semantic `CLEAR` verdict bound to all decision inputs; later-phase retrofits persist and restore their return cursor when the contract is unchanged. | spec workspace + code/docs/decisions | `decision-coverage.md`, `spec.md`, `decisions.md`, `assumptions.md`, `questions.md`, `state.md` |
| [`/rite-temper`](../pack/.claude/skills/rite-temper/SKILL.md) | temper | `[slug] [--mode]` | **Optional, before define.** Strategic review of the readied spec: scope mode (expand / selective / hold-rigor / reduce-to-MVP) + pre-mortem + 9-dimension floor-gate; folds decisions into the spec via the Spec Drift Guard. Significance-gated; **mandatory in `/rite-autocomplete`**. Reviewer: `devrites-strategy-reviewer`. | `spec.md` + decisions/assumptions + design-brief | `strategy.md`, `spec.md`, `decisions.md`, `assumptions.md` |
| [`/rite-define`](../pack/.claude/skills/rite-define/SKILL.md) | define → plan | `[slug]` | Authors and approves architecture, plan, vertical `SLICE-###` tasks, and traceability, then leaves the workspace at the `plan` checkpoint for `/rite-vet`. | `spec.md` + `decision-coverage.md` (+ `strategy.md`) + references | `architecture.md`, `plan.md`, `tasks.md`, `traceability.md`, `state.md`, `decisions.md` |
| [`/rite-vet`](../pack/.claude/skills/rite-vet/SKILL.md) | vet | `[slug] [--cross-model] [--full]` | **Before build: every feature.** Engineering review of the defined plan: scope challenge (reuse / minimum-diff / complexity smell) + architecture / plan code-quality / test-coverage design / performance, confidence-banded with a quote-the-source verification gate; failure-mode + parallelization map. Hardens `plan.md` / `tasks.md` in place; writes the build-readable `test-plan.md` and semantic, input-digest-bound `READY` verdict; acceptance-changing deltas route via the Spec Drift Guard. Runs on every plan: depth scales to stakes (light pass on simple plans, full on big/risky), never skipped; **always in `/rite-autocomplete`**. Reviewer: `devrites-plan-reviewer` (+ optional `--cross-model`). | `plan.md` + `tasks.md` + `spec.md` (+ `strategy.md`) | `eng-review.md`, `test-plan.md`, `plan.md`, `tasks.md`, `decisions.md`, `state.md` |
| [`/rite-plan`](../pack/.claude/skills/rite-plan/SKILL.md) | repair → plan | `[mode]` | Reslice / repair / re-order / split / unblock an active plan and return to the `plan` checkpoint; `revise` is artifact-only and `/rite-vet` is the normal resume. | spec/plan/tasks/state/drift + diff | `plan.md`, `tasks.md`, `state.md`, `decisions.md` |
| [`/rite-build`](../pack/.claude/skills/rite-build/SKILL.md) | build | `[slice]` | Orchestrate **exactly one** vertical slice through the sole wright. The root writes an exact `.wright-allowlist`, retains the original slice baseline through snapshot → reconcile check → test/package integrity → close, and refreshes only the dispatch boundary on bounded retries. Objective failures stay agent-owned; only product/scope/policy choices, irreversible risk, or human-only access/actions become questions. | workspace + diff + `.devrites/CHECKPOINT` | code + `.wright-allowlist`, `recovery-attempts.jsonl`, `state.md`, `evidence.md`, `traceability.md`, `touched-files.md` (+ local `WIP(<slug>)` commit in checkpoint mode) |
| [`/rite-converge`](../pack/.claude/skills/rite-converge/SKILL.md) | converge | `[slug]` | **Recovery.** Compare live code with clarified intent, append each unmet piece as a traceable `SLICE-###`, and invalidate the old vet verdict so changed work returns through `/rite-vet`. `tasks.md` stays byte-identical when already converged. | clarified spec + plan/tasks + principles + live code | `tasks.md` (appended), `traceability.md`, `state.md`, `eng-review.md` (invalidated), `decisions.md` |
| [`/rite-prove`](../pack/.claude/skills/rite-prove/SKILL.md) | prove | `[scope]` | Tests + build + runtime + browser proof of the completed feature. | `traceability.md` + workspace + diff | `evidence.md`, `browser-evidence.md`, `traceability.md`, `state.md` |
| [`/rite-polish`](../pack/.claude/skills/rite-polish/SKILL.md) | polish | `[target \| mode]` | Orchestrator. Reads `reference/code.md` always (Phase 1 + 2); reads `reference/ui.md` when UI is touched (Phase 3 + 4). Mode tokens: `bolder \| quieter \| distill \| harden \| normalize-only`. | workspace + design system + diff | `polish-report.md`, `browser-evidence.md` |
| [`/rite-review`](../pack/.claude/skills/rite-review/SKILL.md) | review | `[scope]` | Feature-scoped multi-axis review. Parallel fresh-context Spec + Standards agents (`devrites-spec-reviewer`, `devrites-code-reviewer`). | workspace + diff | `review.md`, `evidence.md`, `state.md` |
| [`/rite-seal`](../pack/.claude/skills/rite-seal/SKILL.md) | seal | none | GO / NO-GO **decision**, hands off to `/rite-ship`. Walks acceptance vs evidence, fans out reviewers, writes the verdict. Runs no git; on GO sets `state.md` `Next step: /rite-ship`. Triggers: "GO / NO-GO", "is it safe to merge", "decide if we can ship". | all artifacts + diff | `seal.md`, `state.md` |
| [`/rite-ship`](../pack/.claude/skills/rite-ship/SKILL.md) | ship | `[slug]` | Final phase. Requires a GO in `seal.md` → collapses any `WIP(<slug>)` checkpoints → renders type-`GO` + runs the irreversible git ladder (commit → push → tag/PR) + closes the task (archive workspace → `.devrites/archive/<slug>/`, clear `ACTIVE`, phase `done`). Triggers: "ship it", "ship this", "push it out", "close the task". | `seal.md` + all artifacts + diff | `ship.md`, `state.md`, archive |
| [`/rite-autocomplete`](../pack/.claude/skills/rite-autocomplete/SKILL.md) | (orchestrator) | `[idea] [--ship\|--yolo] [--max-slices N]` | Full lifecycle (spec → clarify → … → seal → ship). Spec + clarify form the one interactive window; AFK/checkpoint mode arms only after decision coverage is CLEAR. Pauses for genuine human-owned decisions/actions, NO-GO, or budget exhaustion; objective red checks use bounded technical recovery. Default stops at final type-`GO`; `--ship` (`--yolo`) auto-confirms it. | idea + workspace | whole workspace (drives every phase) |
| [`/rite-quick`](../pack/.claude/skills/rite-quick/SKILL.md) | (express) | `<change>` | Express lane for a **small, reversible, unambiguous** change: one-line contract → TDD build → scoped prove → review-lite → ship, no full artifact tree. **Significance gate first**: auth / migration / public-API / destructive / multi-slice / ambiguous → escalates to `/rite-spec`. Triggers: "quick fix", "small change", "tiny tweak", "just do X". | the change + codebase | code + commit (optional `brief.md` / `evidence.md`) |
| [`/rite-frame`](../pack/.claude/skills/rite-frame/SKILL.md) | lens | `[ask \| diff]` | Pre-flight + self-audit lens for ad-hoc work the lifecycle gates never see: **FRAME** turns an imperative ask into a falsifiable success criterion + verify command before code; **AUDIT** checks a raw diff against the four LLM coding failure modes (silent assumption / overcomplication / out-of-scope edit / unverifiable goal). Top of `/rite-quick` or before a plain "just do X". | the ask / a raw diff | success criterion + verify command (inline) |
| [`/rite-adopt`](../pack/.claude/skills/rite-adopt/SKILL.md) | onboard | `[path]` | Bring an EXISTING codebase under DevRites: reverse-derive a `spec.md` of current behavior + placement + architecture, seed the conventions ledger from observed idioms, then hand off to the lifecycle. Triggers: "adopt this project", "onboard this codebase", "we already have code". | the codebase | `spec.md`, `.devrites/conventions.md`, `decisions.md`, `state.md` |
| [`/rite-status`](../pack/.claude/skills/rite-status/SKILL.md) | status | `[slug]` | Active feature: phase, run mode (AFK/HITL), status, next action, evidence, open questions by gate, risks, handoff readiness. Reads via the shared `devrites-lib` preamble (portable). | workspace + `.devrites/AFK` | none |
| [`/rite-resolve`](../pack/.claude/skills/rite-resolve/SKILL.md) | resume | `<qid> "<answer>"` \| `--drop <qid>` \| `--batch <file>` | Answer / drop / batch-resolve open `questions.md` entries; clears `state.md` `Awaiting human` and sets `Status: running`. Canonical writer for `status: open → answered`. | `questions.md` + `state.md` | `questions.md`, `state.md` |
| [`/rite-zoom-out`](../pack/.claude/skills/rite-zoom-out/SKILL.md) | utility | none | One-pass structural map of an unfamiliar area (modules, in-callers, out-calls, decisions) in project vocabulary. Prefers codegraph/graphify. | codebase + ADRs/CONTEXT.md | none |
| [`/rite-prototype`](../pack/.claude/skills/rite-prototype/SKILL.md) | utility | `[question]` | Throwaway code answering ONE design question. Logic harness OR 2 to 4 UI variations on one route. Captures verdict to `decisions.md`. | spec / surrounding code | prototype scratch + `decisions.md` |
| [`/rite-handoff`](../pack/.claude/skills/rite-handoff/SKILL.md) | utility | `[next-session-focus]` | Compacts the chat into a handoff doc. Syncs chat-only context into workspace canonical files. References existing artifacts by path. | chat + workspace | `handoff.md` + sync into canonical files |
| [`/rite-learn`](../pack/.claude/skills/rite-learn/SKILL.md) | utility | `[--mine \| "<lesson>"]` | Mine archived features for recurring mistakes / dismissed-finding classes; propose project-local lessons into `.devrites/learnings.md` (loaded by the review skills before a fan-out). | archive + workspace | `.devrites/learnings.md` |
| [`/rite-customize`](../pack/.claude/skills/rite-customize/SKILL.md) | utility | `[override <agent> \| extension <name>]` | Guided authoring for project-local reviewer overrides and extensions; writes the smallest artifact, then runs the matching validator. Explicit-only. | `.devrites/overrides`, `.devrites/extensions`, pack agents | `.devrites/overrides/<agent>.md` or `.devrites/extensions/<name>/...` |
| [`/rite-explain`](../pack/.claude/skills/rite-explain/SKILL.md) | utility | `[concept \| diff: \| walkthrough: \| since: \| idea]` | The **human** half of the learning loop (complement of `/rite-learn`, which teaches the repo). Turns a concept, diff, idea, or window of the user's own recent work into a dense personal explainer; diff inputs can produce a concern-ordered human review walkthrough. Grounds off `seal.md` / `evidence.md` / the diff / the archive. Read-only against source. | workspace + archive + diff + code | `.devrites/explainers/<date>-<slug>/explainer.md` or `walkthrough.md` |
| [`/rite-pov`](../pack/.claude/skills/rite-pov/SKILL.md) | utility | `[candidate/link/question]` | Project-grounded verdict on adopting, switching, rejecting, or ignoring a named external technology/library/platform/pattern. Clears a project floor and external floor before grading. | repo profile + code/docs + external sources | `decisions.md` or ADR only on request |
| [`/rite-dogfood`](../pack/.claude/skills/rite-dogfood/SKILL.md) | utility | `[feature-slug\|branch] [--port N]` | Diff-scoped browser QA: map changed user journeys, run scenario matrix, fix small obvious breakages, write dogfood report. Explicit-only. | diff + app routes + browser | `.devrites/work/<slug>/dogfood.md` + safe fixes |
| [`/rite-pr-feedback`](../pack/.claude/skills/rite-pr-feedback/SKILL.md) | utility | `[PR number\|thread URL]` | Resolve GitHub PR review feedback: fetch unresolved threads, judge centrally, fix valid items, reply, resolve. Explicit-only. | PR threads + code | code/tests + PR replies/resolutions |
| [`/rite-pressure-test`](../pack/.claude/skills/rite-pressure-test/SKILL.md) | utility | `[idea]` | Pressure-test a rough idea: 3 to 5 genuinely different options → converge on one with trade-off + hinge. | spec / surrounding code | `decisions.md` (optional) |
| [`/rite-doctor`](../pack/.claude/skills/rite-doctor/SKILL.md) | diagnostic | `[--code \| --reindex]` | Diagnose DevRites install, workspace, and optional index health. `--reindex` explicitly runs the internal synchronous refresh. Triggers: "rite doctor", "is DevRites healthy", "reindex". | install + workspace + optional indexes | none |

## Internal specialist skills (`user-invocable: false`, model-invoked)

The 11 specialist skills below are model-invoked. `devrites-lib` is the twelfth
internal skill, but it sets `disable-model-invocation: true` and serves only as
the shared reference library.

| Skill | Triggered by | Role | Notable |
|---|---|---|---|
| [`devrites-interview`](../pack/.claude/skills/devrites-interview/SKILL.md) | `/rite-spec`, underspecified ask | One-Q-at-a-time protocol | best-guess + confidence stop |
| [`devrites-source-driven`](../pack/.claude/skills/devrites-source-driven/SKILL.md) | uncertain framework/library fact | Consult docs/source, record citation | writes `evidence.md` / `decisions.md` |
| [`devrites-doubt`](../pack/.claude/skills/devrites-doubt/SKILL.md) | non-trivial decision in build/review | CLAIM → EXTRACT → DOUBT → RECONCILE → STOP | adversarial; the root gates genuine human-owned uncertainty |
| [`devrites-ux-shape`](../pack/.claude/skills/devrites-ux-shape/SKILL.md) | UI detected in `/rite-spec` | Plan UX/UI before code → `design-brief.md` (direction, states, interaction, visual-direction probe) | the build target; refs: brief-template/visual-direction-probe |
| [`devrites-frontend-craft`](../pack/.claude/skills/devrites-frontend-craft/SKILL.md) | UI detected in build/polish | Build **to** `design-brief.md`: register, refine-per-slice, states, anti-slop | refs: shape/craft/design-references |
| [`devrites-prose-craft`](../pack/.claude/skills/devrites-prose-craft/SKILL.md) | a phase writes prose; `/rite-polish` Phase 1 catch | Human-voice writing: strip LLM tells, keep precise lists/terms | refs: banned-phrases, structures, examples |
| [`devrites-browser-proof`](../pack/.claude/skills/devrites-browser-proof/SKILL.md) | UI in prove/polish | Browser proof ladder + evidence schema + the structured **Visual Verdict** table | harness preferred |
| [`devrites-refresh-indexes`](../pack/.claude/skills/devrites-refresh-indexes/SKILL.md) | Stop hook or explicit `/rite-doctor --reindex` call | Keep codebase-memory-mcp / codegraph / graphify current after edits | internal synchronous force; no-ops when no index |
| [`devrites-debug-recovery`](../pack/.claude/skills/devrites-debug-recovery/SKILL.md) | failing tests/build/runtime | 7-step: loop → reproduce → hypotheses → trace → instrument → fix → cleanup | durable three-failure budget per root-cause fingerprint |
| [`devrites-api-interface`](../pack/.claude/skills/devrites-api-interface/SKILL.md) | cross-boundary slice | Stable API/contract design | FE/BE split |
| [`devrites-audit simplify`](../pack/.claude/skills/devrites-audit/SKILL.md) | `/rite-polish` Phase 1 | Chesterton's Fence, behavior-preserving simplification | dispatches `devrites-simplifier-reviewer` |
| [`devrites-audit security`](../pack/.claude/skills/devrites-audit/SKILL.md) | input/auth/data/integration in scope | OWASP Top 10, three-tier boundary | dispatches `devrites-security-auditor` |
| [`devrites-audit perf`](../pack/.claude/skills/devrites-audit/SKILL.md) | perf relevant or regression risk | Measure-first, CWV targets | dispatches `devrites-performance-reviewer` |
| [`devrites-lib/reference/parallel-dispatch.md`](../pack/.claude/skills/devrites-lib/reference/parallel-dispatch.md) | loaded inline by `/rite-seal` and `/rite-review` | Reference doc: host-neutral fresh-context dispatch + reconciliation rules for parallel reviewer fan-out | not a skill: a reference file |

## Agents (`.claude/agents/devrites-*`, fresh-context leaves)

**Seventeen roles:** 16 read-only leaves plus one source/test writer,
`devrites-slice-wright`.

| Agent | Spawned by | Purpose |
|---|---|---|
| [`devrites-evidence-scout`](../pack/.claude/agents/devrites-evidence-scout.md) | `/rite-spec`, `/rite-clarify`, `/rite-converge` | Read-only bounded evidence dossier from live code, project records, or cited external facts |
| [`devrites-plan-drafter`](../pack/.claude/agents/devrites-plan-drafter.md) | `/rite-define`, `/rite-plan repair` | Read-only planning candidate; the root makes decisions and writes planning artifacts |
| [`devrites-proof-runner`](../pack/.claude/agents/devrites-proof-runner.md) | `/rite-prove`, affected re-proof | Read-only tree plus non-destructive command execution; returns a proof report, never the verdict |
| [`devrites-slice-wright`](../pack/.claude/agents/devrites-slice-wright.md) | `/rite-build` (the build core) | **Sole source/test writer**: implement one exact allowlisted slice (orient → TDD → verify); returns a typed artifact and writes no bookkeeping |
| [`devrites-strategy-reviewer`](../pack/.claude/agents/devrites-strategy-reviewer.md) | `/rite-temper` (pre-plan) | Spec-vs-rubric strategic review (ambition / scope / premise / pre-mortem / YAGNI / testability / irreversibility / cross-cutting / convention); read-only; **not** part of the seal fan-out |
| [`devrites-plan-reviewer`](../pack/.claude/agents/devrites-plan-reviewer.md) | `/rite-vet` (pre-build) | Plan-vs-rubric engineering review (architecture / scope-reuse / plan code-quality / test-coverage design / performance / reversibility / failure-mode coverage), confidence-banded with a quote-the-source verification gate; read-only; **not** part of the seal fan-out |
| [`devrites-forge-judge`](../pack/.claude/agents/devrites-forge-judge.md) | `/rite-build` on a `Forge: yes` slice | Comparative judge of K=2 to 3 competing candidate builds (acceptance / test strength / principle fit / simplicity / reuse / anti-slop); picks the single winner to land, names grafts; read-only |
| [`devrites-spec-reviewer`](../pack/.claude/agents/devrites-spec-reviewer.md) | `/rite-review` Spec axis; `/rite-seal` | Does the diff implement the spec? Missing/partial/wrong criteria; scope creep |
| [`devrites-code-reviewer`](../pack/.claude/agents/devrites-code-reviewer.md) | `/rite-review` Standards axis; `/rite-seal` | Correctness / readability / architecture / maintainability |
| [`devrites-test-analyst`](../pack/.claude/agents/devrites-test-analyst.md) | `/rite-seal` | Do the tests prove the acceptance criteria? |
| [`devrites-frontend-reviewer`](../pack/.claude/agents/devrites-frontend-reviewer.md) | `/rite-seal` on UI features | UX, a11y, responsive, design-system, anti-AI-slop; reads the **Visual Verdict** |
| [`devrites-security-auditor`](../pack/.claude/agents/devrites-security-auditor.md) | `/rite-seal` when input/auth/data in scope | OWASP Top 10, trust boundary, secrets, deps |
| [`devrites-performance-reviewer`](../pack/.claude/agents/devrites-performance-reviewer.md) | `/rite-seal` when perf relevant | N+1s, hot paths, payload size |
| [`devrites-devex-reviewer`](../pack/.claude/agents/devrites-devex-reviewer.md) | `/rite-vet` (predict) + `/rite-seal` (measure) when a developer-facing surface is in scope | Developer-experience scorecard + predict-vs-measure boomerang (TTHW, getting-started, error-message quality, ergonomics, docs) |
| [`devrites-doubt-reviewer`](../pack/.claude/agents/devrites-doubt-reviewer.md) | `devrites-doubt` loop | Adversarial check of a single claim/decision |
| [`devrites-simplifier-reviewer`](../pack/.claude/agents/devrites-simplifier-reviewer.md) | `devrites-audit simplify` | Independent simplification judgment |
| [`devrites-retrospector`](../pack/.claude/agents/devrites-retrospector.md) | `/rite-ship` close (cadence-gated) | Cross-feature retrospective: mines the shipped archive for recurring patterns + trends; **drafts** graduation candidates for `/rite-learn`; read-only |

Only the root dispatches; leaves never dispatch other leaves. The first
fallback is a generic `explorer` or `worker` that reads the same role contract,
but only when the host preserves the required read-only or exact-write
boundary. If no safe fresh-context option is available, the root runs the work
inline and labels the result `independence: fallback`. Declared leaf identity
and scope guards fail closed. A missing or crashed engine blocks the tool call
instead of granting permission to continue.

## Engineering rules (`pack/.claude/skills/devrites-lib/reference/standards/`)

Workspace-operating lifecycle skills read `core.md` in step 0, while compact
utilities load their narrower contract. Other rules load on demand. The full
index is in
[`pack/.claude/skills/devrites-lib/reference/standards/README.md`](../pack/.claude/skills/devrites-lib/reference/standards/README.md).

- `core.md` (always-on): operating rules + universal anti-rationalizations + 1-line craft disciplines + persistence-before-stopping summary.
- On-demand rules and checklists (read by the phase that needs them): `coding-style.md` · `prose-style.md` · `error-handling.md` · `testing.md` · `spec-grammar.md` · `code-review.md` · `edge-case-trace.md` · `principles.md` · `security.md` · `performance.md` · `observability.md` · `developer-experience.md` · `patterns.md` · `git-workflow.md` · `ci-cd.md` · `hooks.md` · `documentation.md` · `development-workflow.md` · `deprecation.md` · `elicitation.md` · `agents.md` · `context-hygiene.md` · `anti-patterns.md` · `afk-hitl.md` · `tooling.md` · `skill-authoring.md` · `definition-of-done.md` · `review-checklist.md` · `test-proof-checklist.md` · `browser-proof-checklist.md` · `security-checklist.md`
- `anti-patterns.md`: pack-wide rationalizations + red flags. Loaded by each per-phase `rite-*/reference/anti-patterns.md`; can be loaded directly for cross-phase reluctance.

## Trigger conditions (auto-selection)

| Trigger | Routes to |
|---|---|
| Frontend/UI detected (TSX/JSX/Vue/Svelte/Astro/Angular/ERB, CSS/Tailwind/tokens, components/forms/states) | `devrites-ux-shape` in spec (writes `design-brief.md`: direction, **calibration** density/motion, states), `devrites-frontend-craft` in build (builds to it; extracts to a supplied Figma/image target), `devrites-browser-proof` in prove, `rite-polish` Phase 3 + 4 (`reference/ui.md`) in polish, UX/a11y axes at review/seal, optional **design-memory** rollup → project `DESIGN.md` at ship |
| Uncertain library / framework behavior | `devrites-source-driven` |
| Non-trivial decision (boundary, data model, auth, public API, migration, "this scales/safe") | `devrites-doubt` (+ `devrites-doubt-reviewer`) |
| User input / auth / storage / external integration / secrets / permissions | `devrites-audit security` (+ `devrites-security-auditor`) |
| Performance requirement or suspected regression | `devrites-audit perf` (+ `devrites-performance-reviewer`) |
| Failing tests / build / runtime / browser checks | `devrites-debug-recovery` |
| Slice crosses a boundary or defines a public interface | `devrites-api-interface` |
| Explicit `/rite-zoom-out` / `$rite-zoom-out` | read-only structural map (uses codegraph/graphify) |
| Every defined plan before build; "engineering review", "review the architecture", "lock in the plan", "test coverage check" | `/rite-vet` (+ `devrites-plan-reviewer`; depth scales to stakes, never skipped; always in `/rite-autocomplete`) |

## Code-graph integration

These skills prefer a code-intelligence index (`codegraph_*` or
`graphify-out/`) when available and fall back to file reads:

- `/rite-spec`, `/rite-clarify`, `/rite-define`, `/rite-plan`: placement / impact / callers during investigation and decision coverage
- `/rite-vet`: reuse-vs-rebuild, blast-radius, and placement-realism checks during the scope challenge + architecture axis
- `/rite-build`: `touched-files.md` + impact when loading slice context
- `/rite-review`: blast-radius checks on the diff
- `/rite-seal`: final blast-radius check
- `devrites-doubt`: "where does this claim reach" via `codegraph_impact` / `codegraph_callers`
- `/rite-zoom-out`: `codegraph_context` + `codegraph_explore`
- `devrites-frontend-craft`: component / token lookups

## Interactions (typical flow)

See [`flow.md`](flow.md) for the Mermaid diagrams. The text form is:

```
/rite-frame → /rite-spec → /rite-clarify → /rite-temper → /rite-define → /rite-vet → /rite-build ×N → /rite-converge → /rite-prove → /rite-polish → /rite-review → /rite-seal → /rite-ship
   │            │                │  ▲              │                          │             (decide)   (execute+close)
   │            │                │  └ Spec Drift Guard → /rite-plan repair ────┘
   │            │                └ devrites-frontend-craft / source-driven / doubt
   └ (no workspace) → summary    devrites-* internal skills fire on triggers above

/rite-autocomplete drives the entire sequence above unattended (best option per soft gate; --ship auto-confirms ship).
```

- Every phase **reads the active workspace first**; if none, it stops and tells
  the user to run `/rite-spec <feature>`.
- **Spec Drift Guard** lives in build/prove/polish/review/seal: on drift,
  stop, record in `drift.md`, classify, ask the user if product behavior
  changes, then `/rite-plan repair` before resuming.
- `/rite-seal` fans out to `.claude/agents/devrites-*` reviewers **in
  parallel** for independent, fresh-context judgment, then writes the GO /
  NO-GO verdict: it runs no git. On GO it hands off to `/rite-ship`, which
  renders type-`GO`, runs the irreversible git ladder, and closes the task by
  archiving the workspace to `.devrites/archive/<slug>/` and clearing
  `.devrites/ACTIVE`.
