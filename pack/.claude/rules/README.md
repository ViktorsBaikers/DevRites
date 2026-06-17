# DevRites rules

Stack-agnostic engineering rules that DevRites installs to `.claude/rules/`. They encode
the standards the DevRites workflow holds code to — quality, safety, testing, and review
discipline that apply in any language.

These are **common** by design: nothing here assumes a specific framework or language.
Project-specific conventions always win where they exist (DevRites reads the codebase and
prefers what's already there).

## Loading model — progressive disclosure

To keep context lean, the rules follow Claude's progressive-disclosure pattern. There
are 16 rule files (plus this README index): each DevRites `rite-*` skill Reads
`.claude/rules/core.md` as its first step; the other 15 rule files load on demand by the
phase that needs them.

### Always-on (read by each `rite-*` skill as step 0)

| Rule | Covers |
|---|---|
| `core.md` | Operating rules, universal anti-rationalizations, one-line craft disciplines (fail-fast, reuse-first, test-behaviour-not-impl, trust boundary, measure-first), persistence + hygiene reminders. |

### On-demand (read by the phase / topic that needs it)

| Rule | Covers | Typical phase |
|---|---|---|
| `coding-style.md` | Naming, function shape, guard clauses, comments, simplicity, reuse-first. | `/rite-build`, `/rite-polish` Phase 1. |
| `error-handling.md` | Fail fast, no silent catches, meaningful messages, fail closed. | `/rite-build`, `/rite-polish` (backend), `/rite-review`. |
| `testing.md` | Pyramid, behavior over implementation, determinism. | `/rite-build`, `/rite-prove`, `/rite-review`. |
| `code-review.md` | Small PRs, severity labels, what to check, actionable feedback. | `/rite-review`, `/rite-seal`. |
| `security.md` | Untrusted input, least privilege, secrets, three-tier trust boundary, fail closed. | When input / auth / data / integrations are in scope. |
| `performance.md` | Measure first, common pitfalls, prove the win. | When perf is in scope. |
| `patterns.md` | SOLID, composition, loose coupling, avoid over-engineering. | `/rite-build`, simplification audit. |
| `git-workflow.md` | Conventional Commits, atomic commits, small PRs. | `/rite-ship` commit / push / tag steps. |
| `hooks.md` | Stage checks by cost, fast local hooks, secret scanning. | When configuring hooks; `/rite-ship` commit/git step points here. |
| `documentation.md` | Explain why, keep current, record decisions. | `/rite-spec`, `/rite-define`, `/rite-seal`. |
| `development-workflow.md` | Small batches, trunk-always-green, definition of done. | `/rite-define`, `/rite-plan`. |
| `agents.md` | DevRites review subagents + specialist skills, when to fan out. | `/rite-review`, `/rite-seal`. |
| `context-hygiene.md` | `/clear` vs `/compact`, lost-in-the-middle, phase-aware hygiene footer. | Phase-end hygiene footer; choosing `/clear` vs `/compact`. |
| `anti-patterns.md` | Pack-wide rationalizations + red flags the agent reaches for. | Loaded by each `rite-*/reference/anti-patterns.md`; loaded directly when reluctance is broader than the active phase. |
| `afk-hitl.md` | AFK vs HITL contract: `.devrites/AFK` sentinel format, `questions.md` schema, four-gate taxonomy (advisory / validating / blocking / escalating), AFK-never-silently-accepts boundaries. | `/rite-build`, `/rite-status`, `/rite-resolve`, `devrites-doubt`; anywhere a pause-or-proceed decision happens. |

How they're used: DevRites skills follow these rules; you and Claude can reference them
directly. They are guidance, not enforced gates — the enforced gates live in the
workflow skills (Spec Drift Guard, readiness gates, the seal).
