# DevRites rules

Stack-agnostic rules: `.claude/skills/devrites-lib/reference/standards/`.
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
| `spec-grammar.md` | Structuring high-risk behavioral requirements or capability deltas. |
| `code-review.md` | Reviewing a change or sealing review findings. |
| `edge-case-trace.md` | Sweeping branches, boundaries, fixed-set siblings, or deletion contracts. |
| `security.md` | Handling input, auth, data, secrets, dependencies, or integrations. |
| `performance.md` | A measured performance concern is in scope. |
| `observability.md` | A changed runtime path must be diagnosed in production. |
| `developer-experience.md` | A public API, CLI, SDK, webhook, config, error, or getting-started surface changes. |
| `patterns.md` | Choosing or simplifying architecture. |
| `git-workflow.md` | Preparing commits, branches, tags, or changelog entries. |
| `hooks.md` | Creating or changing repository hooks. |
| `ci-cd.md` | Creating or changing a build/deploy pipeline. |
| `documentation.md` | Behavior, commands, contracts, or durable decisions change. |
| `elicitation.md` | Temper or Vet needs a sharper reasoning move for one section. |
| `development-workflow.md` | Planning batch size, integration, or the standing done bar. |
| `principles.md` | Authoring or checking project invariants and approved exceptions. |
| `deprecation.md` | Removing, replacing, or migrating behavior, code, APIs, or data. |
| `agents.md` | Dispatching, awaiting, validating, or reconciling fresh-context agents. |
| `context-hygiene.md` | Choosing `/clear`, `/compact`, or a handoff. |
| `anti-patterns.md` | A pack-wide rationalization or red flag appears. |
| `afk-hitl.md` | A pause, question, resume, or AFK decision is possible. |
| `one-shot-actions.md` | A proof/action may be attempted once, needs fresh retry authorization, consumes external state/quota, or can delete its own failure evidence. |
| `tooling.md` | Structural lookup, current external facts, or architecture memory is needed. |
| `skill-authoring.md` | Creating, editing, routing, evaluating, or pruning a DevRites skill. |
| `definition-of-done.md` | Prove, Seal, Ship, or Quick must decide whether work is finished. |
| `review-checklist.md` | A compact review pass/fail sweep is enough. |
| `test-proof-checklist.md` | Test and evidence quality needs a compact sweep. |
| `browser-proof-checklist.md` | UI behavior needs browser proof. |
| `security-checklist.md` | Auth, input, data, or integration work needs a compact security sweep. |

These guide judgment; workflows and engine gates enforce.
