# All 29 skills

The pack ships **29 skills total** — 20 user-invocable `rite-*` workflow + utility skills, 9 model-invoked `devrites-*` specialists. Each skill is a structured workflow with its own operating rules, anti-rationalization tables, and red flags. Engineering rules live at `.claude/rules/`; each `rite-*` skill Reads `.claude/rules/core.md` as its first step (step 0) and pulls the other rule files on demand (no carrier skill, no session-start autoload). A 30th directory — the internal `devrites-lib` library skill (`user-invocable: false`, not a command) — ships the cross-cutting scripts the workflow skills run: the read-only orientation preamble and the `readiness` / `evidence-fresh` / `acceptance` gate scripts, plus the `tick-afk` / `resolve` / `close-out` state mutators. A unified `devrites` CLI and an MCP server expose those ops to any tool (see [`cli-mcp.md`](cli-mcp.md)).

**Naming convention.** `rite-*` is the user-facing slash-command surface (lifecycle phases plus utilities — `rite-prototype`, `rite-handoff`, `rite-zoom-out`, `rite-pressure-test`). `devrites-*` is internal (model-invoked, hidden from the menu) and collision-avoiding against bundled Claude Code skill names. Visibility is governed by each skill's `user-invocable:` flag; the prefix mirrors it.

---

## Failure-mode section convention

Every skill carries a **failure-mode section** — the highest-signal content for keeping
the model honest in that phase (per Anthropic's skill-authoring guidance: capture the
model's actual failure points and grow the list over time). The canonical heading is
**`## Gotchas`**. Existing skills express the same intent under equivalent headings, all
of which satisfy the convention:

- **Lifecycle skills** → a `> **Mid-flight discipline.**` blockquote pointing at
  `reference/anti-patterns.md` (rite-spec, -temper, -define, -vet, -build, -prove,
  -polish, -review, -seal, -ship, -plan, -resolve, -autocomplete).
- **Specialists / utilities** → `## Hard rules` (browser-proof, debug-recovery),
  `## NEVER` (ux-shape), `## Rules` (doubt, source-driven), `## Boundaries`
  (pressure-test), `## Don't ask` (interview), `## When NOT to use` (zoom-out),
  `## What NOT to include` (handoff), `## Scope reminders` (audit), `## Anti-AI-slop`
  (frontend-craft), the numbered rule list (prototype), and `## Gotchas`
  (api-interface, rite, status).

**New skills SHOULD use `## Gotchas`** — 2–3 traps the model most reaches for in that
phase (the rationalizations in `pack/.claude/rules/anti-patterns.md`, specialized), not a
restatement of the positive steps. A mechanical rename of the equivalents above to a
single `## Gotchas` heading is a possible future normalization; the content already holds.

## Commands quick reference

Every user-invocable skill responds to **both** `/rite <verb>` (menu form — type `/rite` for the discoverable entry point) and `/rite-<verb>` (direct shortcut). Both hit the same skill; use whichever reads more naturally.

| Menu form | Direct shortcut | Phase | Does |
|---|---|---|---|
| `/rite` | — | menu | Compact menu + dispatches verb args to the matching `rite-<verb>` skill. Without args, renders the menu. |
| `/rite spec <feature>` | `/rite-spec <feature>` | spec | **Start here.** Deep investigation → writes `spec.md`: understands the ask fully, decides **placement** (where it lives), names what it resolves, closes gaps with you (questions with options), and gathers any design references you optionally attach (screenshots / Figma / links / video — there may be none). |
| `/rite temper` | `/rite-temper` | temper | **Optional, before define.** Strategic review of the readied spec — pick a scope mode (expand / selective / hold-rigor / reduce-to-MVP), run a pre-mortem, harden the spec, write `strategy.md`, and fold the decisions back via the Spec Drift Guard. Significance-gated (skips small work); **mandatory inside `/rite-autocomplete`**. |
| `/rite define` | `/rite-define` | plan | Turns the approved spec into `plan.md` + vertical task slices + state. |
| `/rite vet` | `/rite-vet` | vet | **Before build — every feature.** Engineering review of the defined plan — scope challenge (reuse / minimum-diff / complexity smell), then architecture / plan code-quality / test-coverage design / performance through senior-engineer lenses, every finding confidence-banded with a quote-the-source verification gate. Maps failure modes + parallel lanes, hardens `plan.md` / `tasks.md`, writes the build-readable `test-plan.md`; acceptance-changing deltas fold back via the Spec Drift Guard. `--cross-model` adds a different-model second opinion. Runs on every plan — depth scales to stakes (light pass on simple, full on big/risky), never skipped; **always inside `/rite-autocomplete`**. |
| `/rite plan` | `/rite-plan` | re-plan | Decompose / reslice / repair an active plan. |
| `/rite build` | `/rite-build` | build | Implement **exactly one** vertical slice, then stop — dispatches a fresh-context `devrites-slice-wright` to write it; gates and records the result. |
| `/rite prove` | `/rite-prove` | prove | Tests + build + runtime + browser evidence for the current scope. |
| `/rite polish` | `/rite-polish` | polish | Code polish always; UI normalize + polish if UI in scope. Modes: `bolder / quieter / distill / harden / normalize-only`. |
| `/rite review` | `/rite-review` | review | Feature-scoped multi-axis review (parallel Spec + Standards axes). |
| `/rite seal` | `/rite-seal` | seal | Final GO / NO-GO decision gate — walks acceptance vs evidence, fans out reviewers, writes the verdict to `seal.md`. Decides only; hands off to `/rite-ship`. |
| `/rite ship` | `/rite-ship` | ship | Final phase. Requires a GO in `seal.md` → renders type-`GO`, runs the irreversible git ladder (commit → push → tag/PR), writes `ship.md`, then closes the task (archives the workspace, clears `ACTIVE`). |
| `/rite autocomplete` | `/rite-autocomplete` | (orchestrator) | Runs the whole lifecycle unattended (spec → … → seal → ship), choosing the best option at each soft gate. `--ship` / `--yolo` auto-confirms the final type-`GO`. |
| `/rite status` | `/rite-status` | status | Where the active feature stands. |
| `/rite resolve <qid> "<answer>"` | `/rite-resolve <qid> "<answer>"` | resume | Answer a HITL checkpoint (or `--drop <qid>` / `--batch <file>`); clears `state.md` `Awaiting human` and resumes. |
| `/rite zoom-out` | `/rite-zoom-out` | utility | Step up an abstraction layer when stuck — returns a map of the area (modules, callers, decisions) in the project's vocabulary. |
| `/rite prototype` | `/rite-prototype` | utility | Throwaway code that answers ONE design question before committing — routes between a Logic harness (state/data model) and 2–4 radically different UI variations on one route. |
| `/rite handoff` | `/rite-handoff` | utility | Compact the chat session into a handoff doc a fresh agent can pick up — syncs to the workspace, references existing artifacts by path. |
| `/rite pressure-test` | `/rite-pressure-test` | utility | Diverge → converge on a rough idea before specifying. |

The 9 model-invoked internal specialists (hidden from the menu): `devrites-interview`, `-source-driven`, `-doubt`, `-ux-shape`, `-frontend-craft`, `-browser-proof`, `-debug-recovery`, `-api-interface`, `-audit` (security / perf / simplify). The `/rite` menu carries the routing, the five user-invocable utilities all use the `rite-*` prefix (`rite-prototype`, `rite-handoff`, `rite-zoom-out`, `rite-pressure-test`, `rite-autocomplete`), and the `/rite-polish` phases live as references inside the same skill rather than as `rite-polish-code` / `rite-polish-ui`. Triggers and interactions are documented in [`command-map.md`](command-map.md); diagrams in [`flow.md`](flow.md).

---

## Phase-by-phase catalogue

### Menu & status — Find your place

| Skill | What It Does | Use When |
|---|---|---|
| [`rite`](../pack/.claude/skills/rite/SKILL.md) | Compact menu + active-feature status + suggested next command. Does not run a workflow. | You type `/rite`, ask "what DevRites commands exist", or "where am I". |
| [`rite-status`](../pack/.claude/skills/rite-status/SKILL.md) | Read-only report: phase, run mode (AFK/HITL), status, active slice, next action, evidence, open questions by gate, drift, risks. | You ask "where am I", "what's the status", "what's next". |
| [`rite-resolve`](../pack/.claude/skills/rite-resolve/SKILL.md) | Answer / drop / batch-resolve open `questions.md` entries; clears `state.md` `Awaiting human` and resumes the workflow. | A HITL checkpoint is open or AFK left blocking questions; you type `/rite-resolve <qid> "<answer>"`. |

### Spec — Understand the ask before any code

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-spec`](../pack/.claude/skills/rite-spec/SKILL.md) | Deep investigation → writes `spec.md`. Decides **placement**, names what it resolves, closes gaps with you, gathers design refs. | You start a feature, have a vague idea, attach screenshots/Figma/video, or say "spec this". |
| [`devrites-ux-shape`](../pack/.claude/skills/devrites-ux-shape/SKILL.md) | **Plans UX/UI before code** — writes the feature-level `design-brief.md` (design direction, key states, interaction model, Figma/image visual-direction probe) that the build targets. Woven into spec/build, not a separate phase. | `/rite-spec` detects UI, or you say "shape the UX" / "plan the UI before coding" / "design direction". |
| [`devrites-interview`](../pack/.claude/skills/devrites-interview/SKILL.md) | One-question-at-a-time interview until ~95% confidence. | The ask is underspecified, or user says "interview me" / "grill me". |
| [`rite-pressure-test`](../pack/.claude/skills/rite-pressure-test/SKILL.md) | Structured divergent → convergent thinking; rough concept → buildable proposal. | The idea itself needs exploration before specifying; users say "ideate", "stress-test my plan", "I have a vague idea". |

### Temper — strategic review before planning *(optional; mandatory in `/rite-autocomplete`)*

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-temper`](../pack/.claude/skills/rite-temper/SKILL.md) | Strategic review of a readied `spec.md`: pick a scope mode (expand / selective / hold-rigor / reduce-to-MVP), run a pre-mortem, score 9 dimensions on a floor-gate, then write `strategy.md` and fold decisions into `spec.md` / `decisions.md` / `assumptions.md` via the Spec Drift Guard. Adversarial fresh-context loop via [`devrites-strategy-reviewer`](../pack/.claude/agents/devrites-strategy-reviewer.md). Ambition on outcomes, minimalism on the surface. | A big / risky feature (auth · data model · public API · migration · multi-slice · ambiguous scope), or you say "think bigger" / "scope check" / "pre-mortem". Skips low-stakes specs in one line. |

### Plan — Decompose into vertical slices

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-define`](../pack/.claude/skills/rite-define/SKILL.md) | Approved spec → `plan.md` + vertical task slices + `state.md`. | `spec.md` exists and is approved. |
| [`rite-plan`](../pack/.claude/skills/rite-plan/SKILL.md) | Decompose / reslice / repair an active plan after drift. | Spec Drift Guard fires, or you need to repair an existing plan. |

### Vet — Lock in the engineering plan before building

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-vet`](../pack/.claude/skills/rite-vet/SKILL.md) | Engineering review of `plan.md` + `tasks.md`: scope challenge (reuse / minimum-diff / complexity smell), then architecture / plan code-quality / test-coverage design / performance through senior-engineer lenses, every finding confidence-banded with a quote-the-source verification gate. Maps failure modes + parallel lanes, hardens the plan, writes the build-readable `test-plan.md`; acceptance-changing deltas fold back via the Spec Drift Guard. Adversarial fresh-context loop via [`devrites-plan-reviewer`](../pack/.claude/agents/devrites-plan-reviewer.md) on the full pass; optional `--cross-model` second opinion. | **Every defined plan, before build** — depth scales to stakes: a light pass on a simple, reversible plan; the full pass on a big / risky one (migration · auth · public API · data model · multi-slice · >8 files · new dependency). Never skipped; always in `/rite-autocomplete`. |

### Build — One verified slice at a time

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-build`](../pack/.claude/skills/rite-build/SKILL.md) | Orchestrates **exactly one** vertical slice: pre-flight gates, then dispatches a fresh-context [`devrites-slice-wright`](../pack/.claude/agents/devrites-slice-wright.md) for the build core, then doubts / gates / records with evidence. | A plan exists and the next slice is ready. |
| [`devrites-source-driven`](../pack/.claude/skills/devrites-source-driven/SKILL.md) | Consult official docs / source before relying on framework behavior; record the source. | API, config, or framework behavior is assumed rather than known. |
| [`devrites-api-interface`](../pack/.claude/skills/devrites-api-interface/SKILL.md) | Design stable API / interface contracts — REST/GraphQL, module boundaries, type contracts, FE/BE split. | A slice crosses a boundary or defines a public interface. |
| [`devrites-debug-recovery`](../pack/.claude/skills/devrites-debug-recovery/SKILL.md) | Reproduce → ranked hypotheses → instrument → fix in scope → regression-test. Stops guess-and-retry. | Tests, builds, or runtime checks fail. |

### Prove — Real evidence, including the browser

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-prove`](../pack/.claude/skills/rite-prove/SKILL.md) | Tests + build + runtime + browser evidence for the current scope. | All slices built; ready for full verification. |
| [`devrites-browser-proof`](../pack/.claude/skills/devrites-browser-proof/SKILL.md) | Browser proof ladder: browser-harness → Chrome DevTools MCP → `/run`+`/verify` → project E2E → manual. | Scope touches UI. |

### Polish — Normalize, then ship-quality detail pass

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-polish`](../pack/.claude/skills/rite-polish/SKILL.md) | Orchestrator. Always runs code polish (`reference/code.md`); detects UI scope and runs UI normalize + polish (`reference/ui.md`) when needed. Accepts mode tokens (`bolder`, `quieter`, `distill`, `harden`, `normalize-only`). | All slices proven; ready to finish. |
| [`devrites-frontend-craft`](../pack/.claude/skills/devrites-frontend-craft/SKILL.md) | Senior frontend craft — register, shape-before-code, all states, design system, anti-AI-slop, Core Web Vitals + WCAG 2.2. | A slice touches UI. |

### Review — Adversarial, scoped to the feature

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-review`](../pack/.claude/skills/rite-review/SKILL.md) | Multi-axis feature-scoped review. Parallel **Spec axis** (`devrites-spec-reviewer`) + **Standards axis** (`devrites-code-reviewer`) sub-agents; severity-labeled findings. | Polish done; ready for a final pass. |
| [`devrites-doubt`](../pack/.claude/skills/devrites-doubt/SKILL.md) | In-flight adversarial review of risky decisions — branching logic, boundaries, data/auth/API changes, migrations. | About to stand a "this is safe / scales / matches spec" claim. |
| [`devrites-audit`](../pack/.claude/skills/devrites-audit/SKILL.md) | Read-only audit dispatch — picks the right reviewer subagent on the requested axis (`security` / `perf` / `simplify`). Single-axis only; multi-axis parallel fan-out lives inline in `/rite-seal` (see `rite-seal/reference/parallel-dispatch.md`). | Polish Phase 1 (`simplify`); review involves user input / auth / data / external integrations / secrets / permissions (`security`); performance is relevant or regression risk visible (`perf`). |

### Seal — GO / NO-GO decision

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-seal`](../pack/.claude/skills/rite-seal/SKILL.md) | Pure decision gate. Walks acceptance vs evidence, fans out reviewers, writes the GO / NO-GO verdict to `seal.md`. Decides only — no git; on GO it sets `state.md` `Next step: /rite-ship`. | Review is clean; you ask "GO / NO-GO", "is it safe to merge", "decide if we can ship". |

### Ship — Execute + close the task

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-ship`](../pack/.claude/skills/rite-ship/SKILL.md) | Final phase. Requires a GO in `seal.md` → renders type-`GO`, runs the irreversible git ladder (commit → push → tag/PR per project convention), writes `ship.md`, then closes the task: sets phase `done`, archives `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` (all `.md` preserved), clears `.devrites/ACTIVE`. | Seal returned GO; you say "ship it", "ship this", "push it out", "close the task". |

### Utility — Stand-alone helpers

| Skill | What It Does | Use When |
|---|---|---|
| [`rite-autocomplete`](../pack/.claude/skills/rite-autocomplete/SKILL.md) | Runs the whole lifecycle unattended (spec → … → seal → ship), choosing the best option at each soft gate and recording the rationale in `decisions.md`. A vague prompt triggers an up-front `devrites-interview`; after that it runs without per-phase iteration, pausing only on hard irreversible-risk / blocking / escalating gates, an open validating gate, NO-GO, or budget exhaustion. Default stops at the final type-`GO`; `--ship` (alias `--yolo`) auto-confirms it. | "Autocomplete", "do the whole thing", "run the full cycle", "one-shot this feature". |
| [`rite-zoom-out`](../pack/.claude/skills/rite-zoom-out/SKILL.md) | Map an unfamiliar area — modules, callers, callees, decisions — in the project's domain glossary. | You say "zoom out", "I don't know this area", "map this area", "bigger picture". |
| [`rite-prototype`](../pack/.claude/skills/rite-prototype/SKILL.md) | Throwaway code answering ONE design question — logic harness OR 2–4 UI variations on one route. | "Prototype this", "let me play with it", "try a few designs", question is undecidable on paper. |
| [`rite-handoff`](../pack/.claude/skills/rite-handoff/SKILL.md) | Compact chat session → handoff doc. References existing `.devrites/work/<slug>/` artifacts by path. | "Handoff", "summarize this session", before `/clear`, fresh agent will continue. |
| [`rite-learn`](../pack/.claude/skills/rite-learn/SKILL.md) | Cross-feature learning loop — mine shipped features for recurring mistakes + dismissed-finding classes (`learnings.sh mine`), propose project-local lessons into `.devrites/learnings.md` (loaded by the review skills before a fan-out). | After several features ship; "what have we learned", "harvest lessons", "why does this keep coming up". |

### Foundation — Engineering rules

| Skill | What It Does | Use When |
|---|---|---|
| (engineering rules) | Live at `.claude/rules/` post-install — each `rite-*` skill Reads `.claude/rules/core.md` as its first step (step 0); the other 19 rule files (`coding-style.md`, `prose-style.md`, `error-handling.md`, `testing.md`, `code-review.md`, `security.md`, `performance.md`, `observability.md`, `patterns.md`, `git-workflow.md`, `hooks.md`, `documentation.md`, `development-workflow.md`, `deprecation.md`, `agents.md`, `context-hygiene.md`, `anti-patterns.md`, `afk-hitl.md`, `tooling.md`) load on demand per skill body. No carrier skill, no session-start autoload. | n/a — step-0 core / on-demand by path. |

---

## Review agents — Fresh-context reviewers

Spawned by `/rite-review`, `/rite-seal` (post-build), `/rite-temper` (pre-plan), and `/rite-vet` (pre-build) for independent judgment. Read-only; given only the artifact + rubric, never the author's reasoning.

| Agent | Purpose |
|---|---|
| [`devrites-strategy-reviewer`](../pack/.claude/agents/devrites-strategy-reviewer.md) | **Pre-plan** spec-vs-rubric strategic review (ambition / scope / premise / pre-mortem / YAGNI / testability / irreversibility / cross-cutting / convention) — via `/rite-temper`, not the seal fan-out. |
| [`devrites-plan-reviewer`](../pack/.claude/agents/devrites-plan-reviewer.md) | **Pre-build** plan-vs-rubric engineering review (architecture / scope-reuse / plan code-quality / test-coverage design / performance / reversibility / failure-mode coverage), confidence-banded with a quote-the-source verification gate — via `/rite-vet`, not the seal fan-out. |
| [`devrites-forge-judge`](../pack/.claude/agents/devrites-forge-judge.md) | **Build-time** comparative judge of K=2–3 competing candidate builds of one slice (acceptance / test strength / principle fit / simplicity / reuse / anti-slop) — via `/rite-build` on a `Forge: yes` slice; picks the single winner to land, names grafts. |
| [`devrites-spec-reviewer`](../pack/.claude/agents/devrites-spec-reviewer.md) | Does the diff implement the spec? Missing / partial / wrong criteria; scope creep. |
| [`devrites-code-reviewer`](../pack/.claude/agents/devrites-code-reviewer.md) | Correctness / readability / architecture / maintainability. |
| [`devrites-test-analyst`](../pack/.claude/agents/devrites-test-analyst.md) | Do the tests actually prove the acceptance criteria? |
| [`devrites-frontend-reviewer`](../pack/.claude/agents/devrites-frontend-reviewer.md) | UX, a11y, responsive, design-system, anti-AI-slop. |
| [`devrites-security-auditor`](../pack/.claude/agents/devrites-security-auditor.md) | OWASP Top 10, trust boundary, secrets, deps. |
| [`devrites-performance-reviewer`](../pack/.claude/agents/devrites-performance-reviewer.md) | N+1s, hot paths, payload size. |
| [`devrites-doubt-reviewer`](../pack/.claude/agents/devrites-doubt-reviewer.md) | Adversarial check of a single claim/decision. |
| [`devrites-simplifier-reviewer`](../pack/.claude/agents/devrites-simplifier-reviewer.md) | Independent simplification judgment (called by `devrites-audit simplify`). |

Seal fan-out diagram: [`flow.md` § /rite-seal fan-out](flow.md#4-rite-seal-fan-out).

---

## Executor agent — Fresh-context writer

The system's one **write-capable** agent — the mirror of the read-only reviewers above. Dispatched by `/rite-build` for the build core, in a clean context, with only the slice contract.

| Agent | Purpose |
|---|---|
| [`devrites-slice-wright`](../pack/.claude/agents/devrites-slice-wright.md) | Turn ONE slice contract into the smallest complete, idiomatic, proven implementation — orient → TDD → verify, anti-AI-slop, feature scope only. Writes code + tests; returns a structured artifact for the orchestrator to doubt, gate, and record. Writes no `.devrites/` bookkeeping; single-threaded (one per slice). |

Dispatch contract + return shape + fallback: [`rite-build/reference/wright-dispatch.md`](../pack/.claude/skills/rite-build/reference/wright-dispatch.md).

---

## `/rite-polish` orchestrator

`/rite-polish` always runs code polish; it detects UI scope from the diff and runs UI normalize + polish only when needed. Each half lives as a progressive-disclosure reference inside the skill (`reference/code.md`, `reference/ui.md`) so the orchestrator body stays small and each phase loads only when its trigger fires.

```mermaid
flowchart LR
    P[/rite-polish/] -->|always| C[reference/code.md<br/>Phase 1 + 2]
    P -->|UI touched?| U[reference/ui.md<br/>Phase 3 + 4]
    C --> O([polish-report.md])
    U --> O
    classDef o fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef s fill:#312e81,stroke:#818cf8,color:#eef2ff
    class P o
    class C,U s
```

Argument modes (`bolder`, `quieter`, `distill`, `harden`, `normalize-only`) pass through to Phase 4 (`reference/ui.md`).

## `/rite-review` parallel axes & `/rite-seal` fan-out

`/rite-review` runs **Spec axis** (`devrites-spec-reviewer`) and **Standards axis** (`devrites-code-reviewer`) as parallel sub-agents so neither masks the other. `/rite-seal` fans out the full reviewer set in parallel (`devrites-{spec,code,test,frontend,security,performance}-reviewer`) and reconciles findings before the GO / NO-GO gate. Severity labels (Critical / Important / Suggestion / Nit / FYI) drive the gate; there's no advisory number — the gate is `Critical == 0` plus proven acceptance plus resolved drift. Diagrams: [`flow.md`](flow.md#3-rite-review-parallel-axes).
