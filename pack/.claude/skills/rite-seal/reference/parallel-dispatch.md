# Parallel review dispatch

How `/rite-review` and `/rite-seal` fan out the fresh-context review subagents under `pack/.claude/agents/`. Loaded on demand by the calling skill — not a skill itself.

DevRites ships eight fresh-context review subagents under `.claude/agents/`. The seal and the multi-axis review need most of them running **at the same time**, on the same workspace + diff, so the verdicts don't contaminate each other.

Pattern: delegate to specialized agents with isolated context, brief each one precisely, run them concurrently, reconcile on return.

## When to use which subagent

| Subagent | Always | Conditional |
|---|---|---|
| `devrites-spec-reviewer` | `/rite-review` Spec axis; `/rite-seal` | — |
| `devrites-code-reviewer` | `/rite-review` Standards axis; `/rite-seal` | — |
| `devrites-test-analyst` | `/rite-seal` | — |
| `devrites-frontend-reviewer` | — | UI files in the diff |
| `devrites-security-auditor` | — | input / auth / data / external integrations / secrets in scope |
| `devrites-performance-reviewer` | — | perf budget in `spec.md` or visible regression risk |
| `devrites-doubt-reviewer` | — | a non-trivial decision is being stood up (called from `devrites-doubt`) |
| `devrites-simplifier-reviewer` | — | `/rite-polish` Phase 1 audit (called from `devrites-audit simplify`) |

## Dispatch shape

For each chosen subagent, the caller uses the `Task` tool with this prompt shape:

```
Audit the active DevRites feature.

Workspace: .devrites/work/<slug>/
Read (yourself, fresh context):
  - spec.md  (+ acceptance criteria)
  - touched-files.md
  - the git diff
  - <any axis-specific files: decisions.md, evidence.md, references/...>

Apply your documented discipline. Return labeled findings (Critical /
Important / Suggestion / Nit / FYI) using your documented output format,
ONE FINDING PER LINE, cite file:line.

Feature scope only. No edits. Do not summarize or re-rank — the caller
reconciles.
```

Rules:

- **One Task call per subagent.** Send them in a single message with multiple `Task` invocations so the runtime dispatches concurrently.
- **No cross-pollination.** Each subagent gets only its narrow brief and the workspace path. Do not pass another subagent's findings into a sibling's prompt — that recreates the masking problem.
- **No author context.** Do not include the caller's analysis or the user's framing of the change; the point is a fresh, adversarial read.
- **Feature scope only.** Each subagent must stay inside `touched-files.md` + the diff.

## Reconciliation

When the subagents return:

1. **Quote verbatim.** Place each subagent's findings under its own `## <axis>` heading in `review.md` / `seal.md`. Do not merge, re-rank, or summarize.
2. **Surface contradictions explicitly.** "Spec axis says complete, Standards axis says untestable" is a finding, not noise. The caller decides at the gate.
3. **Severity is the gate, not a score.** Sum the labels (`Critical / Important / Suggestion / Nit / FYI`) and apply the caller's gate (`/rite-seal` blocks on `Critical == 0`; `/rite-review` reports counts).
4. **One scale.** All subagents use the same five-label scale. Reject any subagent output that invents its own.

## Fallback

If the `Task` tool is unavailable in the current environment, the caller runs the relevant subagent discipline **inline** in its own context and flags the result as a fallback (not an independent review). The seal weighs the fallback differently — see [`./risk-and-rollback.md`](./risk-and-rollback.md).
