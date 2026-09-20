# DevRites rules

Stack-agnostic rules: `.omp/skills/devrites-lib/reference/standards/`.
[`core.md` § Precedence](core.md#precedence) governs: repository conventions
select technical form only inside its safety, source-writing, and evidence gates.

## Loading model

Each workspace rite reads [`core.md`](core.md); then load only the current
topic's owner.

| Rule | Load when |
|---|---|
| `core.md` | Every workflow phase: operating, persistence, evidence, and precedence rules. |
| `coding-style.md` | Writing or simplifying code. |
| `prose-style.md` | Writing prose artifacts or user replies. |
| `error-handling.md` | Adding or reviewing failure paths. |
| `testing.md` | Designing tests or judging proof quality. |
| `gates.md` | Authoring, executing, or judging the `gates.md` acceptance ledger — runnable oracles, evidence binding, ABANDON handoffs. |
| `verification-methods.md` | Choosing checks and counting coverage — method distinctness, the unit×check×environment denominator, status vs result, capability fallbacks, recipient-path execution. |
| `spec-grammar.md` | Structuring high-risk behavioral requirements or capability deltas. |
| `code-review.md` | Reviewing a change or sealing review findings. |
| `review/README.md` | A diff touches source files — per-language defect-probe checklists (always with `code-review.md`). |
| `duplicate-code.md` | Triage of near-duplicate clusters, the `check dup` scan, or the dup ignore ledger. |
| `edge-case-trace.md` | Resolving relevant edge/prohibition classes and their evidence disposition. |
| `security.md` | Handling input, auth, data, secrets, dependencies, or integrations. |
| `repository-topology.md` | Work spans a monorepo member, nested root, multiple languages/services, or repositories. |
| `data-integrity.md` | Durable writes, schemas, migrations/backfills, concurrency, retention, or tenant data are touched. |
| `integration-reliability.md` | APIs, webhooks, queues/jobs, caches, or cross-service failure behavior is touched. |
| `performance.md` | A measured performance concern is in scope. |
| `observability.md` | A changed runtime path must be diagnosed in production. |
| `developer-experience.md` | A public API, CLI, SDK, webhook, config, error, or getting-started surface changes. |
| `patterns.md` | Choosing or simplifying architecture. |
| `architecture-health.md` | An audit needs a structural read on the whole codebase — coupling/cohesion/cycle score from a dependency index. |
| `git-workflow.md` | Preparing commits, branches, tags, or changelog entries. |
| `hooks.md` | Creating or changing repository hooks. |
| `ci-cd.md` | Creating or changing a build/deploy pipeline. |
| `documentation.md` | Behavior, commands, contracts, or durable decisions change. |
| `elicitation.md` | Temper or Vet needs a sharper reasoning move for one section. |
| `development-workflow.md` | Planning batch size, integration, or the standing done bar. |
| `principles.md` | Authoring or checking project invariants and approved exceptions. |
| `deprecation.md` | Removing, replacing, or migrating behavior, code, APIs, or data. |
| `agents.md` | Dispatching, awaiting, validating, or reconciling fresh-context agents. |
| `loop-operations.md` | Running a goal-, time-, or event-activated loop through native host scheduling. |
| `workflow-artifacts.md` | Materializing executable proof/controller/harness files under the active `.devrites/work/<slug>/`. |
| `context-hygiene.md` | Choosing `/clear`, `/compact`, or a handoff. |
| `anti-patterns.md` | A pack-wide rationalization or red flag appears. |
| `afk-hitl.md` | A pause, question, resume, or AFK decision is possible. |
| `one-shot-actions.md` | A proof/action may be attempted once, needs fresh retry authorization, consumes external state/quota, or can delete its own failure evidence. |
| `tooling.md` | Structural lookup, current external facts, or architecture memory is needed. |
| `code-navigation.md` | Choosing graph, LSP, grep, or read routes before cross-file edits. |
| `skill-authoring.md` | Creating, editing, routing, evaluating, or pruning a DevRites skill. |
| `definition-of-done.md` | Prove, Seal, Ship, or Quick must decide whether work is finished. |
| `review-checklist.md` | A compact review pass/fail sweep is enough. |
| `test-proof-checklist.md` | Test and evidence quality needs a compact sweep. |
| `browser-proof-checklist.md` | UI behavior needs browser proof. |
| `security-checklist.md` | Auth, input, data, or integration work needs a compact security sweep. |
| `acceptance-preserving-reslice.md` | Classifying or reviewing a Reslice in Plan, Vet, or Autocomplete. |
| `audit-coverage.md` | An audit covers a surface larger than one feature diff — ledger, hunter/critic waves, three-verdict findings. |
| `architecture-health.md` | An audit needs a whole-codebase structural score from a code index — coupling, cohesion, cycles, god files, orphans, depth. |
| `assumption-checkpoints.md` | Scope generalizes an existing boundary: a second case, a newly optional field, or a constant turned parameter. |
| `debug-recovery.md` | Waiting on async readiness (server start, job completion, browser signal) needs a bounded condition-based poll. |
| `windows.md` | A deferral marker (TODO/FIXME/skipped test/stub) is introduced or waived — the `check windows` ledger and grammar. |
| `glossary.md` | A DevRites term is unclear; resolve it once here instead of inferring. |

The table is a load trigger, not an exemption. [`core.md` § Rule summary](core.md#rule-summary-load-the-full-file-when-in-scope)
makes every *applicable* owner mandatory. **Failing case:** a security or data-integrity
gate is skipped because the standard is "on-demand / modular."

These guide judgment; workflows and engine gates enforce.
