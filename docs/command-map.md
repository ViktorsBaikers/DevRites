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
`rite-pr-feedback`, plus the evidence-gated workspace compatibility command
`rite-upgrade`. The host may invoke `devrites-*` specialists through the model.
`devrites-lib` is the non-workflow library exception.

## Surface lifecycle

- **Promoted:** shipped `rite-*`, `devrites-*`, generated host artifacts, and docs
  in this map.
- **Draft:** research or local experiments that are not copied into `pack/`.
- **Deprecated:** compatibility shims with a removal note and a replacement path.
- **Research:** source-intake notes under `docs/research/`; not installed.

Only promoted surfaces are shipped by the npm package.

## Engine command ownership

The Go surface is intentionally closed and deterministic:

| Lane | Commands | Owner |
|---|---|---|
| Candidate and deterministic checks | `check candidate`, `check readiness [--emit-binding]`, `check seal` | Candidate validates/hashes the strict project manifest; readiness checks phase files or emits the vetted Build-input binding; seal checks final files/open gates, that binding, then exact candidate bindings. |
| Atomic workspace state | `state resolve`, `state close` | Go owns answer/drop/batch resolution and transactional close. |
| Security | `secret-scan` | `/rite-ship`, safe hooks, or an operator scans staged blobs, stdin, or touched files. |
| Install/operator | `install`, `update`, `uninstall`, `version` | npm/bootstrap or a human operator supplies local candidates; the engine performs deterministic local changes. |
| Native policy | no engine command | Skills/root own spec grammar re-read, qid allocation, Clarify cursor edits, AFK/recovery accounting, and read-only `/rite-doctor`. |

`devrites-engine help` is exhaustive. `check candidate <slug>` prints exactly
`candidate-sha256: <64 lowercase hex>` and `candidate-files: <row count>` on a
pass; usage/root errors exit `2` and candidate blocks exit `3`. There are no old
aliases, semantic readiness commands, reviewer-prose parsers, capability-ledger engine,
compatibility telemetry, or migration command.

Claude Code and Codex own native dispatch. Installed skills and exact agents
own semantic readiness, traceability, acceptance/evidence quality, doubt,
review reconciliation, test-quality assessment, capability interpretation,
workspace compatibility, and
recovery routing. Repository tools and CI own test, build, lint, typecheck,
schema, and release execution.

`secret-scan --staged` reads exact NUL-enumerated Git index blobs with
replacement objects disabled. `--stdin` accepts text through process stdin.
Scans cap captured input at 64 MiB and entries/findings at 4,096; findings expose
metadata only and source or limit errors fail closed.

The shell/npm updater acquires the exact-release pack/binary, then invokes
the offline local `devrites-engine update`. Its `--check` compares installed and
local candidate versions; engine release selectors are unsupported.
`/rite-upgrade` separately audits an older active workspace. Only a cited
current-contract defect may route a repair through its existing Clarify, Plan
repair, Converge, Vet, Prove, Polish, Review, or Seal owner.
See [`cli.md`](cli.md).

## Public commands (`user-invocable: true`)

| Command | Phase | Argument | What it does | Reads | Writes |
|---|---|---|---|---|---|
| [`/rite`](../pack/.claude/skills/rite/SKILL.md) | menu | `[subcommand]` | Compact menu + suggested next command. Pure router; does **not** read state because `/rite-status` owns that job. | none | none |
| [`/rite-spec`](../pack/.claude/skills/rite-spec/SKILL.md) | spec | `<feature>` | **New feature.** Deep investigation → writes a product-focused `spec.md` (WHAT/WHY, requirements, ACs, boundaries, gaps, design references, and one capability-impact declaration). MODIFIED deltas preserve prior scenarios/claims unless an accepted decision authorizes removal. | codebase + native file/code search + shipped archive Markdown | `README.md`, `brief.md`, `spec.md`, `references/`, `references.md`, `questions.md`, `decisions.md`, `assumptions.md`, `state.md` |
| [`/rite-clarify`](../pack/.claude/skills/rite-clarify/SKILL.md) | clarify | `[slug]` | **Required, adaptive.** Topology-first coverage scan of the written spec; searches facts, closes human-owned decisions, audits assumptions, and takes a zero-question fast path when already clear. Writes a semantic `CLEAR` verdict reconciled against the spec, decisions, assumptions, and evidence; later-phase retrofits persist and restore their return cursor when the contract is unchanged. | spec workspace + code/docs/decisions | `decision-coverage.md`, `spec.md`, `decisions.md`, `assumptions.md`, `questions.md`, `state.md` |
| [`/rite-temper`](../pack/.claude/skills/rite-temper/SKILL.md) | temper | `[slug] [--mode]` | **Optional, before define.** Strategic review of the readied spec: scope mode (expand / selective / hold-rigor / reduce-to-MVP) + pre-mortem + 9-dimension floor-gate; folds decisions into the spec via the Spec Drift Guard. Significance-gated; **mandatory in `/rite-autocomplete`**. Reviewer: `devrites-strategy-reviewer`. | `spec.md` + decisions/assumptions + design-brief | `strategy.md`, `spec.md`, `decisions.md`, `assumptions.md` |
| [`/rite-define`](../pack/.claude/skills/rite-define/SKILL.md) | define → plan | `[slug]` | Authors architecture, plan, vertical `SLICE-###` tasks, and traceability. A changed provider/consumer boundary names one canonical shared contract plus consuming provider/consumer tests; no change uses a justified no-impact statement. | `spec.md` + `decision-coverage.md` (+ `strategy.md`) + references | `architecture.md`, `plan.md`, `tasks.md`, `traceability.md`, `state.md`, `decisions.md` |
| [`/rite-vet`](../pack/.claude/skills/rite-vet/SKILL.md) | vet | `[slug] [--cross-model] [--full]` | **Before build: every feature.** Engineering review of the defined plan: scope challenge (reuse / minimum-diff / complexity smell) + architecture / plan code-quality / test-coverage design / performance, confidence-banded with a quote-the-source verification gate; failure-mode and dependency-safety review. Hardens `plan.md` / `tasks.md` in place; writes the build-readable `test-plan.md` and a semantic `READY` verdict reconciled by the exact native reviewer; acceptance-changing deltas route via the Spec Drift Guard. Runs on every plan: depth scales to stakes (light pass on simple plans, full on big/risky), never skipped; **always in `/rite-autocomplete`**. Reviewer: `devrites-plan-reviewer` (+ optional `--cross-model`). | `plan.md` + `tasks.md` + `spec.md` (+ `strategy.md`) | `eng-review.md`, `test-plan.md`, `plan.md`, `tasks.md`, `decisions.md`, `state.md` |
| [`/rite-plan`](../pack/.claude/skills/rite-plan/SKILL.md) | repair → plan | `[mode]` | Reslice / repair / re-order / split / unblock an active plan and return to the `plan` checkpoint; `revise` is artifact-only and `/rite-vet` is the normal resume. | spec/plan/tasks/state/drift + diff | `plan.md`, `tasks.md`, `state.md`, `decisions.md` |
| [`/rite-build`](../pack/.claude/skills/rite-build/SKILL.md) | build | `[slice]` | In HITL, orchestrates one vertical slice through the sole wright; an explicit `.devrites/AFK` sentinel may chain bounded low-risk slices. Every task states exact project-relative paths; after return, the root rejects extra paths or weakened tests, runs repository proof, and maintains the strict candidate manifest. | workspace + diff + `.devrites/CHECKPOINT` | code + `state.md`, `evidence.md`, `traceability.md`, `touched-files.md` (+ local `WIP(<slug>)` commit in checkpoint mode) |
| [`/rite-converge`](../pack/.claude/skills/rite-converge/SKILL.md) | converge | `[slug]` | **Recovery.** Compare live code with clarified intent, append each unmet piece as a traceable `SLICE-###`, and invalidate the old vet verdict so changed work returns through `/rite-vet`. `tasks.md` stays byte-identical when already converged. | clarified spec + plan/tasks + principles + live code | `tasks.md` (appended), `traceability.md`, `state.md`, `eng-review.md` (invalidated), `decisions.md` |
| [`/rite-upgrade`](../pack/.claude/skills/rite-upgrade/SKILL.md) | compatibility | `[slug]` | **Conditional recovery.** Audit an older released workspace against current contracts. Only cited defects route through Clarify, Plan repair, Converge, Vet, Prove, Polish, Review, or Seal. Ambiguous candidate scope is a gap; age/cursor form alone is never a defect, and old passes are never synthesized. | active workspace + named current contracts | admitted unfinished artifacts via their owning rite |
| [`/rite-prove`](../pack/.claude/skills/rite-prove/SKILL.md) | prove | `[scope]` | Positive, discriminating tests + build/runtime/browser proof; checks the same candidate before/after commands and binds evidence to its digest. | `traceability.md` + workspace + candidate | `evidence.md`, `browser-evidence.md`, `traceability.md`, `state.md` |
| [`/rite-polish`](../pack/.claude/skills/rite-polish/SKILL.md) | polish | `[target \| mode]` | Runs code/UI polish, then applicable capability-ledger, design-memory, and ADR rollups; updates the manifest and affected proof before closing the candidate. | workspace + design system + candidate | `polish-report.md`, `browser-evidence.md`, `touched-files.md`, durable rollups |
| [`/rite-review`](../pack/.claude/skills/rite-review/SKILL.md) | review | `[scope]` | Feature-scoped parallel Spec + Standards review of the closed candidate; binds `review.md` to its digest. | workspace + candidate | `review.md`, `evidence.md`, `state.md` |
| [`/rite-seal`](../pack/.claude/skills/rite-seal/SKILL.md) | seal | none | Candidate-bound GO / NO-GO **decision**. Rechecks exact proof/review bindings and runs no git; on GO sets `Next step: /rite-ship`. | all artifacts + candidate | `seal.md`, `state.md` |
| [`/rite-ship`](../pack/.claude/skills/rite-ship/SKILL.md) | ship | `[slug]` | Candidate-read-only final phase. Requires Seal GO, verifies exact staged scope/bytes, renders type-`GO`, commits, reverifies the committed candidate before push/tag/PR, then archives and clears `ACTIVE`. | `seal.md` + candidate + Git index | `ship.md`, `state.md`, archive |
| [`/rite-autocomplete`](../pack/.claude/skills/rite-autocomplete/SKILL.md) | (orchestrator) | `[idea] [--ship\|--yolo] [--max-slices N] [--full] [--cross-model]` | Full lifecycle (spec → clarify → … → seal → ship). Spec + clarify form the one interactive window; AFK/checkpoint mode arms only after decision coverage is CLEAR. Pauses for genuine human-owned decisions/actions, NO-GO, or budget exhaustion; objective red checks use bounded technical recovery. Default stops at Seal GO; `--ship` (`--yolo`) reaches Ship preflight but still requires fresh literal `GO` and native approval; `--full` selects the Full profile and `--cross-model` arms Vet's second opinion. | idea + workspace | whole workspace (drives every phase) |
| [`/rite-quick`](../pack/.claude/skills/rite-quick/SKILL.md) | (express) | `<change>` | Express lane for a **small, reversible, unambiguous** change: one-line contract → TDD build → scoped prove → review-lite → ship, no full artifact tree. **Significance gate first**: auth / migration / public-API / destructive / multi-slice / ambiguous → escalates to `/rite-spec`. Triggers: "quick fix", "small change", "tiny tweak", "just do X". | the change + codebase | code + commit (optional `brief.md` / `evidence.md`) |
| [`/rite-frame`](../pack/.claude/skills/rite-frame/SKILL.md) | lens | `[ask \| diff]` | Pre-flight + self-audit lens for ad-hoc work the lifecycle gates never see: **FRAME** turns an imperative ask into a falsifiable success criterion + verify command before code; **AUDIT** checks a raw diff against the four LLM coding failure modes (silent assumption / overcomplication / out-of-scope edit / unverifiable goal). Top of `/rite-quick` or before a plain "just do X". | the ask / a raw diff | success criterion + verify command (inline) |
| [`/rite-adopt`](../pack/.claude/skills/rite-adopt/SKILL.md) | onboard | `[path]` | Reverse-derive the existing baseline and next objective; propose any durable project guidance in the nearest native instruction file. | codebase + project instructions | workspace spec/state artifacts + optional guidance proposal |
| [`/rite-status`](../pack/.claude/skills/rite-status/SKILL.md) | status | `[slug]` | Active feature: phase, run mode, status, persisted next action, evidence, questions, drift, risks, and handoff readiness. | direct workspace ledger/artifacts | none |
| [`/rite-resolve`](../pack/.claude/skills/rite-resolve/SKILL.md) | resume | `<qid> "<answer>"` \| `--drop <qid>` \| `--batch <file>` | Answer / drop / batch-resolve open `questions.md` entries; clears `state.md` `Awaiting human` and sets `Status: running`. Canonical writer for `status: open → answered`. | `questions.md` + `state.md` | `questions.md`, `state.md` |
| [`/rite-zoom-out`](../pack/.claude/skills/rite-zoom-out/SKILL.md) | utility | none | One-pass structural map of an unfamiliar area (modules, in-callers, out-calls, decisions) in project vocabulary. Prefers codegraph/graphify. | codebase + ADRs/CONTEXT.md | none |
| [`/rite-prototype`](../pack/.claude/skills/rite-prototype/SKILL.md) | utility | `[question]` | Throwaway code answering ONE design question. Logic harness OR 2 to 4 UI variations on one route. Captures verdict to `decisions.md`. | spec / surrounding code | prototype scratch + `decisions.md` |
| [`/rite-handoff`](../pack/.claude/skills/rite-handoff/SKILL.md) | utility | `[next-session-focus]` | Compacts the chat into a handoff doc. Syncs chat-only context into workspace canonical files. References existing artifacts by path. | chat + workspace | `handoff.md` + sync into canonical files |
| [`/rite-learn`](../pack/.claude/skills/rite-learn/SKILL.md) | utility | `["<lesson>"]` | Review native memory plus verified archive/ADR evidence; propose one durable instruction or ADR update for human approval. | native memory + reviewed Markdown | approved existing instruction/ADR only |
| [`/rite-customize`](../pack/.claude/skills/rite-customize/SKILL.md) | utility | `[instruction \| skill \| agent \| plugin]` | Guided authoring for a native project instruction, skill, agent, or explicitly connected plugin/MCP surface. | native host/project config | approved native artifact |
| [`/rite-explain`](../pack/.claude/skills/rite-explain/SKILL.md) | utility | `[concept \| diff: \| walkthrough: \| since: \| idea]` | The **human** half of the learning loop (complement of `/rite-learn`, which teaches the repo). Turns a concept, diff, idea, or window of the user's own recent work into a dense personal explainer; diff inputs can produce a concern-ordered human review walkthrough. Grounds off `seal.md` / `evidence.md` / the diff / the archive. Read-only against source. | workspace + archive + diff + code | `.devrites/explainers/<date>-<slug>/explainer.md` or `walkthrough.md` |
| [`/rite-pov`](../pack/.claude/skills/rite-pov/SKILL.md) | utility | `[candidate/link/question]` | Project-grounded verdict on one outside option after live repository and primary-source evidence. | live code/docs + external sources | `decisions.md` or ADR only on request |
| [`/rite-dogfood`](../pack/.claude/skills/rite-dogfood/SKILL.md) | utility | `[feature-slug\|branch] [--port N]` | Diff-scoped browser QA: map changed user journeys, run scenario matrix, fix small obvious breakages, write dogfood report. Explicit-only. | diff + app routes + browser | `.devrites/work/<slug>/dogfood.md` + safe fixes |
| [`/rite-pr-feedback`](../pack/.claude/skills/rite-pr-feedback/SKILL.md) | utility | `[PR number\|thread URL]` | Resolve GitHub PR review feedback: fetch unresolved threads, judge centrally, fix valid items, reply, resolve. Explicit-only. | PR threads + code | code/tests + PR replies/resolutions |
| [`/rite-pressure-test`](../pack/.claude/skills/rite-pressure-test/SKILL.md) | utility | `[idea]` | Pressure-test a rough idea: 3 to 5 genuinely different options → converge on one with trade-off + hinge. | spec / surrounding code | `decisions.md` (optional) |
| [`/rite-doctor`](../pack/.claude/skills/rite-doctor/SKILL.md) | diagnostic | none | Diagnose the DevRites binary, pack, schema, and native host permission/profile configuration. | install + host config | none |

## Internal specialist skills (`user-invocable: false`, model-invoked)

The 10 specialist skills below are model-invoked. `devrites-lib` is the eleventh
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
| [`devrites-debug-recovery`](../pack/.claude/skills/devrites-debug-recovery/SKILL.md) | failing tests/build/runtime | 7-step: loop → reproduce → hypotheses → trace → instrument → fix → cleanup | caller + recovery count three failed attempts per causal fingerprint from context and Dead ends/evidence |
| [`devrites-api-interface`](../pack/.claude/skills/devrites-api-interface/SKILL.md) | cross-boundary slice | Stable API/contract design | FE/BE split |
| [`devrites-audit simplify`](../pack/.claude/skills/devrites-audit/SKILL.md) | `/rite-polish` Phase 1 | Chesterton's Fence, behavior-preserving simplification | dispatches `devrites-simplifier-reviewer` |
| [`devrites-audit security`](../pack/.claude/skills/devrites-audit/SKILL.md) | input/auth/data/integration in scope | OWASP Top 10, three-tier boundary | dispatches `devrites-security-auditor` |
| [`devrites-audit perf`](../pack/.claude/skills/devrites-audit/SKILL.md) | perf relevant or regression risk | Measure-first, CWV targets | dispatches `devrites-performance-reviewer` |
| [`devrites-lib/reference/parallel-dispatch.md`](../pack/.claude/skills/devrites-lib/reference/parallel-dispatch.md) | loaded inline by `/rite-seal` and `/rite-review` | Reference doc: host-neutral fresh-context dispatch + reconciliation rules for parallel reviewer fan-out | not a skill: a reference file |

## Agents (`.claude/agents/devrites-*`, fresh-context leaves)

**Seventeen role profiles:** both hosts have 16 read-only leaves plus the
write-capable `devrites-slice-wright`.

| Agent | Spawned by | Purpose |
|---|---|---|
| [`devrites-evidence-scout`](../pack/.claude/agents/devrites-evidence-scout.md) | `/rite-spec`, `/rite-clarify`, `/rite-converge` | Read-only bounded evidence dossier from live code, project records, or cited external facts |
| [`devrites-plan-drafter`](../pack/.claude/agents/devrites-plan-drafter.md) | `/rite-define`, `/rite-plan repair` | Read-only planning candidate; the root makes decisions and writes planning artifacts |
| [`devrites-proof-runner`](../pack/.claude/agents/devrites-proof-runner.md) | `/rite-prove`, affected re-proof | Validates immutable artifacts produced by root-executed proof commands; returns a proof report, never executes a gate or decides the verdict |
| [`devrites-upgrade-planner`](../pack/.claude/agents/devrites-upgrade-planner.md) | `/rite-upgrade` | Fresh read-only contract matrix with typed outcome, cited defects, protected identities, and canonical repair route |
| [`devrites-slice-wright`](../pack/.claude/agents/devrites-slice-wright.md) | `/rite-build` (the build core) | **Sole source/test writer role** on Claude and Codex: implement one exact path-bounded slice and return a typed artifact |
| [`devrites-strategy-reviewer`](../pack/.claude/agents/devrites-strategy-reviewer.md) | `/rite-temper` (pre-plan) | Spec-vs-rubric strategic review (ambition / scope / premise / pre-mortem / YAGNI / testability / irreversibility / cross-cutting / convention); read-only; **not** part of the seal fan-out |
| [`devrites-plan-reviewer`](../pack/.claude/agents/devrites-plan-reviewer.md) | `/rite-vet` (pre-build) | Plan-vs-rubric engineering review (architecture / scope-reuse / plan code-quality / test-coverage design / performance / reversibility / failure-mode coverage), confidence-banded with a quote-the-source verification gate; read-only; **not** part of the seal fan-out |
| [`devrites-spec-reviewer`](../pack/.claude/agents/devrites-spec-reviewer.md) | `/rite-review` Spec axis; `/rite-seal` | Does the diff implement the spec? Missing/partial/wrong criteria; scope creep |
| [`devrites-code-reviewer`](../pack/.claude/agents/devrites-code-reviewer.md) | `/rite-review` Standards axis; `/rite-seal` | Correctness / readability / architecture / maintainability |
| [`devrites-test-analyst`](../pack/.claude/agents/devrites-test-analyst.md) | `/rite-seal` | Do the tests prove the acceptance criteria? |
| [`devrites-frontend-reviewer`](../pack/.claude/agents/devrites-frontend-reviewer.md) | `/rite-seal` on UI features | UX, a11y, responsive, design-system, anti-AI-slop; reads the **Visual Verdict** |
| [`devrites-security-auditor`](../pack/.claude/agents/devrites-security-auditor.md) | `/rite-seal` when input/auth/data in scope | OWASP Top 10, trust boundary, secrets, deps |
| [`devrites-performance-reviewer`](../pack/.claude/agents/devrites-performance-reviewer.md) | `/rite-seal` when perf relevant | N+1s, hot paths, payload size |
| [`devrites-devex-reviewer`](../pack/.claude/agents/devrites-devex-reviewer.md) | `/rite-vet` (predict) + `/rite-seal` (measure) when a developer-facing surface is in scope | Developer-experience scorecard + predict-vs-measure boomerang (TTHW, getting-started, error-message quality, ergonomics, docs) |
| [`devrites-doubt-reviewer`](../pack/.claude/agents/devrites-doubt-reviewer.md) | `devrites-doubt` loop | Adversarial check of a single claim/decision |
| [`devrites-simplifier-reviewer`](../pack/.claude/agents/devrites-simplifier-reviewer.md) | `devrites-audit simplify` | Independent simplification judgment |
| [`devrites-retrospector`](../pack/.claude/agents/devrites-retrospector.md) | `/rite-learn` cross-feature review | Read-only native search over reviewed archive evidence; proposes specific instruction/ADR candidates |

Only the root dispatches; leaves never dispatch other leaves. It dispatches the
exact named project role; if that role is unavailable, the workflow stops for
HITL. The root never executes a specialist role. Reviewer profiles are
natively read-only. Claude keeps its root in plan mode; Codex uses a
workspace-capable root so its child can write, while workflow policy forbids
that root from editing source/tests. Both hosts expose only
`devrites-slice-wright` as a writable specialist. See
[`standards/agents.md`](../pack/.claude/skills/devrites-lib/reference/standards/agents.md#source-writing-boundary).

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
| Frontend/UI detected (TSX/JSX/Vue/Svelte/Astro/Angular/ERB, CSS/Tailwind/tokens, components/forms/states) | `devrites-ux-shape` in spec (writes `design-brief.md`: direction, **calibration** density/motion, states), `devrites-frontend-craft` in build (builds to it; extracts to a supplied Figma/image target), `devrites-browser-proof` in prove, `rite-polish` Phase 3 + 4 (`reference/ui.md`) in polish, UX/a11y axes at review/seal, optional **design-memory** rollup → project `DESIGN.md` in Polish before Review |
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

/rite-autocomplete drives the reversible sequence unattended; --ship reaches
only the exact-plan Ship approval boundary and never authorizes Git.
```

- Every phase **reads the active workspace first**; if none, it stops and tells
  the user to run `/rite-spec <feature>`.
- **Spec Drift Guard** lives in build/prove/polish/review/seal: on drift, stop,
  record in `drift.md`, and classify. Objective implementation and tool defects
  use bounded recovery; a wrong durable plan uses `/rite-plan repair`; only a
  product, policy, or irreversible-risk choice asks the user.
- `/rite-seal` fans out to `.claude/agents/devrites-*` reviewers **in
  parallel** for independent, fresh-context judgment, then writes the GO /
  NO-GO verdict: it runs no git. On GO it hands off to `/rite-ship`, which
  renders type-`GO`, runs the irreversible git ladder, and closes the task by
  archiving the workspace to `.devrites/archive/<slug>/` and clearing
  `.devrites/ACTIVE`.
