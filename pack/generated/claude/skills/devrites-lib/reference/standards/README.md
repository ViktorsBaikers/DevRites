# DevRites rules

Stack-agnostic engineering rules that DevRites installs to `.claude/skills/devrites-lib/reference/standards/`. They encode
the standards the DevRites workflow holds code to — quality, safety, testing, and review
discipline that apply in any language.

These are **common** by design: nothing here assumes a specific framework or language.
Project-specific conventions always win where they exist (DevRites reads the codebase and
prefers what's already there).

## Loading model — progressive disclosure

To keep context lean, the rules follow progressive disclosure: each DevRites `rite-*`
skill Reads `.claude/skills/devrites-lib/reference/standards/core.md` as its first step;
the remaining rule files load on demand by the phase that needs them.

### Always-on (read by each `rite-*` skill as step 0)

| Rule | Covers |
|---|---|
| `core.md` | Operating rules, universal anti-rationalizations, one-line craft disciplines (fail-fast, reuse-first, test-behaviour-not-impl, trust boundary, measure-first), persistence + hygiene reminders. |

### On-demand (read by the phase / topic that needs it)

| Rule | Covers | Typical phase |
|---|---|---|
| `coding-style.md` | Naming, function shape, guard clauses, comments, simplicity, reuse-first. | `/rite-build`, `/rite-polish` Phase 1. |
| `prose-style.md` | Human-voice writing for artifacts + replies; two registers (prose vs technical); the LLM-tell cut-list. Depth in the `devrites-prose-craft` skill. | Any phase that writes prose — `/rite-spec`, `/rite-define`, `/rite-review`, `/rite-seal`, `/rite-ship`; `/rite-polish` Phase 1 as the catch. |
| `error-handling.md` | Fail fast, no silent catches, meaningful messages, fail closed. | `/rite-build`, `/rite-polish` (backend), `/rite-review`. |
| `testing.md` | Pyramid, behavior over implementation, determinism. | `/rite-build`, `/rite-prove`, `/rite-review`. |
| `spec-grammar.md` | Optional, recommended structure for behavioral acceptance — `### Requirement:` (SHALL/MUST) + `#### Scenario:` (WHEN/THEN), validated deterministically by `devrites-engine spec-validate`. Progressive rigor; flat `AC-###` bullets stay valid. | `/rite-spec` readiness gate; `/rite-prove`, `/rite-review` consume the scenario hooks. |
| `code-review.md` | Small PRs, severity labels, what to check, actionable feedback. | `/rite-review`, `/rite-seal`. |
| `edge-case-trace.md` | Mechanical branch/boundary sweep, fixed-set siblings, and deletion-contract checks. | `/rite-review`, `/rite-seal`, `devrites-doubt`. |
| `security.md` | Untrusted input, least privilege, secrets, three-tier trust boundary, fail closed. | When input / auth / data / integrations are in scope. |
| `performance.md` | Measure first, common pitfalls, prove the win. | When perf is in scope. |
| `observability.md` | Structured logs, metrics/SLIs, traces, symptom-based alerts, verify-the-telemetry-fires — proof a feature works in prod. | When the change has a runtime surface (endpoint, job, integration, user flow); `/rite-prove`, `/rite-seal`. |
| `developer-experience.md` | DX as a measured axis — the predict-at-vet / measure-at-prove / reconcile-at-seal boomerang; scorecard (TTHW, getting-started, error-message quality, ergonomics, docs); severity by who-pays. Conditional + greenfield no-op. | When a developer-facing surface is in scope (public API, CLI, SDK/library, webhook, config, error messages, getting-started); `/rite-vet`, `/rite-prove`, `/rite-seal`. |
| `patterns.md` | SOLID, composition, loose coupling, avoid over-engineering. | `/rite-build`, simplification audit. |
| `git-workflow.md` | Conventional Commits, atomic commits, small PRs. | `/rite-ship` commit / push / tag steps. |
| `hooks.md` | Stage checks by cost, fast local hooks, secret scanning. | Reference-only — read when setting up the project's git hooks; not auto-loaded by any phase. |
| `ci-cd.md` | Shift-left + faster-is-safer, the no-skip gate pipeline, the CI-failure loop, Build Cop, feature-flags decouple deploy from release, secret tiers, the pipeline-speed ladder. | Reference — read when setting up or changing a build/deploy pipeline; `/rite-ship` when CI/CD is in scope. |
| `documentation.md` | Explain why, keep current, record decisions. | `/rite-spec`, `/rite-define`, `/rite-seal`. |
| `elicitation.md` | A move-set of named reasoning techniques (Steelman, Delphi, Red-Team/Blue-Team, Assumption Audit, Pre-Mortem…) selected per section by risk, to deepen a spec or plan on demand. | `/rite-temper`, `/rite-vet`; any section that needs sharper thinking. |
| `development-workflow.md` | Small batches, trunk-always-green, definition of done. | `/rite-define`, `/rite-plan`. |
| `principles.md` | The four knowledge layers; project invariants (`.devrites/principles.md`) as a *trusted, gating* pass/fail (vs conventions' untrusted prior); justified-exception register; dated-amendment governance. | `/rite-define`, `/rite-vet`, `/rite-build`, `/rite-review`, `/rite-seal`; seeded by `/rite-adopt`, grown by `/rite-learn`. |
| `deprecation.md` | Code-as-liability, Hyrum's law, prove-unused-before-remove, expand→contract, deprecate-before-delete. The safe path behind the irreversible-migration gate. | When removing / replacing / migrating code, a feature, an API, or data. |
| `agents.md` | DevRites review subagents + specialist skills, when to fan out. | `/rite-review`, `/rite-seal`. |
| `context-hygiene.md` | `/clear` vs `/compact`, lost-in-the-middle, phase-aware hygiene footer. | Phase-end hygiene footer; choosing `/clear` vs `/compact`. |
| `anti-patterns.md` | Pack-wide rationalizations + red flags the agent reaches for. | Loaded by each `rite-*/reference/anti-patterns.md`; loaded directly when reluctance is broader than the active phase. |
| `afk-hitl.md` | AFK vs HITL contract: `.devrites/AFK` sentinel format, `questions.md` schema, four-gate taxonomy (advisory / validating / blocking / escalating), AFK-never-silently-accepts boundaries. | `/rite-build`, `/rite-status`, `/rite-resolve`, `devrites-doubt`; anywhere a pause-or-proceed decision happens. |
| `tooling.md` | Optional external tools — code intelligence (codebase-memory-mcp first → cross-verify with codegraph + graphify → standard methods LSP / `Read`/`Grep`/`Glob`), up-to-date library docs (context7), architecture/ADR memory. Recommended, not required. | Any phase doing structural lookups (callers / impact / placement) or relying on external library/framework facts. |
| `skill-authoring.md` | Skill descriptions, progressive disclosure, completion criteria, and pruning rules for keeping DevRites skills predictable and cheap. | When creating or editing DevRites skills or reviewing skill pack quality. |
| `definition-of-done.md` | Done means acceptance proven, evidence recorded, drift resolved, and handoff/ship state clean. | `/rite-prove`, `/rite-seal`, `/rite-ship`. |
| `review-checklist.md` | Compact reviewer pass/fail checklist. | `/rite-review`, `/rite-seal`. |
| `test-proof-checklist.md` | Proof-quality checklist for tests and evidence. | `/rite-prove`, `/rite-seal`. |
| `browser-proof-checklist.md` | Browser/UI proof checklist. | UI features in `/rite-prove`, `/rite-polish`, `/rite-seal`. |
| `security-checklist.md` | Security review checklist. | Auth/input/data/integration changes. |

How they're used: DevRites skills follow these rules; you and Claude can reference them
directly. They are guidance, not enforced gates — the enforced gates live in the
workflow skills (Spec Drift Guard, readiness gates, the seal).
