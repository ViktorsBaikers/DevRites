# Parallel review dispatch

How `/rite-review` and `/rite-seal` fan out the fresh-context review subagents under `.claude/agents/`. Loaded on demand by the calling skill — not a skill itself.

DevRites ships ten fresh-context review subagents under `.claude/agents/` (plus the write-capable `devrites-slice-wright`, which is not a reviewer). Eight are post-build reviewers used at the seal / multi-axis review; the other two are gate-specific and *not* part of the seal fan-out — `devrites-strategy-reviewer` is **pre-plan** (it judges the spec for `/rite-temper`) and `devrites-plan-reviewer` is **pre-build** (it judges the plan for `/rite-vet`). The seal and the multi-axis review need most of the post-build reviewers running **at the same time**, on the same workspace + diff, so the verdicts don't contaminate each other.

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

Before judging the diff, derive the expected behaviour from the spec
yourself, then compare it against what the code does. Anchor every finding
to file:line plus the spec criterion or command output that proves it —
an unanchored finding is a Suggestion at most. The order or length of the
diff is not evidence.

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
4. **One scale.** All subagents use the same five-label scale (Critical / Important / Suggestion / Nit / FYI). Reject any subagent output that invents its own. **Exception:** `devrites-simplifier-reviewer` deliberately emits only Suggestion / Nit / FYI (it is non-blocking by design) — that is a valid subset of the scale, not an invented one; do not reject it during reconciliation.
5. **Consensus roll-up (after the verbatim per-axis record).** Keep every axis's findings verbatim under its `## <axis>` heading (above), then add one deduped roll-up the gate reads: where **≥2 axes flag the same `file:line`**, raise it to the top and mark it *consensus* — independent corroboration raises confidence. A lone low-confidence finding with no `file:line` or evidence anchor drops out of the roll-up (it stays in its per-axis section). The roll-up reduces noise without hiding any axis — the verbatim sections are the audit trail; the roll-up is the actionable summary the gate acts on.

## Fallback

If the `Task` tool is unavailable in the current environment, the caller runs the relevant subagent discipline **inline** in its own context and flags the result as a fallback (not an independent review). The seal weighs the fallback differently — see [`./risk-and-rollback.md`](./risk-and-rollback.md).
