# Parallel review dispatch

How `$rite-review` and `$rite-seal` fan out the fresh-context review subagents under
`.codex/agents/`. The **single source** for the dispatch + reconciliation contract: loaded on
demand by the calling skill (each points here); not a skill itself.

The seal/review fan-out **roster** is the seven post-build reviewers in the table below. This
file is the single source of which reviewer fires when. The seal and the multi-axis review need
them running **at the same time**, on the same workspace + diff, so the verdicts don't
contaminate each other. Every other agent under `.codex/agents/` fires from its own phase, not
here: the roster table below names them under **Not in this fan-out** so the set is unambiguous.

Pattern: delegate to specialized agents with isolated context, brief each one precisely, run
them concurrently, reconcile on return.

## The roster: every reviewer the fan-out accounts for

These seven are the **roster**. `$rite-seal` accounts for **all** of them: the always-on three
plus each conditional, either dispatched or skip-recorded; `$rite-review` runs the two always-on
axes. Each trigger is a *checkable signal*, not a judgement call, so a conditional reviewer is
either fired or consciously skipped: never silently dropped.

| Reviewer | Fires | Trigger: the checkable signal |
|---|---|---|
| `devrites-spec-reviewer` | always | `$rite-review` Spec axis; `$rite-seal` (carry review's verdict forward only if the diff is unchanged) |
| `devrites-code-reviewer` | always | `$rite-review` Code-review axis; `$rite-seal` (carry forward only if the diff is unchanged) |
| `devrites-test-analyst` | always | at `$rite-seal` |
| `devrites-frontend-reviewer` | conditional | the diff touches a UI file: component / template / stylesheet, per the project's UI paths |
| `devrites-security-auditor` | conditional | the diff touches input handling, auth / authz, data storage or access, an external integration, or a secret |
| `devrites-performance-reviewer` | conditional | `spec.md` states a perf budget, **OR** the diff adds a query / a loop over a growing set / hot-path or render work |
| `devrites-devex-reviewer` | conditional | the diff changes a developer-facing surface: public API, CLI, SDK/library export, webhook, config/env contract, error message, or getting-started path |

### Hit-rate gating: consult before dispatching conditionals

Before dispatching, run `devrites-engine reviewer-stats report`. It grades each reviewer from the
cross-feature dispatch ledger (`.devrites/reviewer-stats.jsonl`):

- `run` / `run (always-on)` / `run (insurance — never gated)`: dispatch per the trigger table above.
- `gate-candidate`: this **conditional** reviewer produced zero surviving findings in its last 10+
  dispatches on this project. Skip it as a *recorded* skip:
  `devrites-engine footprint log <slug> skip devrites-<x>-reviewer` with the reason
  `gated: zero surviving findings in <N> dispatches`. The roster gate stays satisfied: a gated
  skip is a conscious skip.

The verdict is engine-owned and deterministic; never gate by your own judgement of a reviewer's
past usefulness. Two hard bounds: the always-on axes (`spec`, `code-review`, `test-analyst`) and
the insurance reviewers (`security-auditor`, `doubt-reviewer`) are **never** gated: the engine
never grades them `gate-candidate`, and a caller must not skip them on hit-rate grounds. The user
can override gating for one run by asking for a full panel (`--full`): then dispatch every
triggered reviewer regardless of verdict.

After reconciliation, close the loop: record each **dispatched** reviewer's outcome so the
ledger stays live:

```bash
devrites-engine reviewer-stats record devrites-<x>-reviewer <surviving Critical+Important count> <slug>
```

Surviving means it stood after reconciliation and dismissals: a finding the caller dismissed as a
false positive does not count. Record `0` honestly; dry streaks are the signal.

**Not in this fan-out** (named so the roster is unambiguous): `devrites-simplifier-reviewer` fires
at `$rite-polish` Phase 1 (via `devrites-audit simplify`) and `devrites-doubt-reviewer` fires from
`devrites-doubt` when a decision is stood up: neither is a seal reviewer, so neither is part of
the seal accounting. `devrites-strategy-reviewer` (pre-plan, `$rite-temper`), `devrites-plan-reviewer`
(pre-build, `$rite-vet`), `devrites-forge-judge` (`$rite-build` forge), and `devrites-retrospector`
(`$rite-ship` close) are single-agent, phase-locked gates. They fire iff their phase runs, not here.

One entity, one name: the `devrites-code-reviewer`'s axis is the **Code-review axis** everywhere
(at both `$rite-review` and `$rite-seal`): don't rename it per caller.

## Dispatch shape

For each chosen subagent, the caller uses the harness's subagent primitive (the `Task` tool on
Claude Code, `spawn_agent` (with the matching `.codex/agents/devrites-*.toml` custom agent) on
Codex) with this prompt shape:

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

- **One Task call per subagent, awaited in the same turn.** Send them in a single message with multiple `Task` invocations so the runtime dispatches concurrently. Never background/detach reviewers in `$rite-autocomplete` or AFK; there is no event loop that guarantees a later result is reconciled.
- **No cross-pollination.** Each subagent gets only its narrow brief and the workspace path. Do not pass another subagent's findings into a sibling's prompt, that recreates the masking problem.
- **No author context.** Do not include the caller's analysis or the user's framing of the change; the point is a fresh, adversarial read.
- **Never coach a reviewer.** No "do not flag X", "treat Y as at most Minor", or "the plan
  chose this" in a dispatch prompt: pre-judging findings is how a known defect sails through.
  A plan-mandated quirk still gets reported; the caller (or the human) grades it, not the prompt.
- **Feature scope only.** Each subagent must stay inside `touched-files.md` + the diff.
- **Can't-verify is a verdict, not a pass.** A reviewer that cannot verify a spec requirement
  from the diff + its allowed reads returns it as a `CANNOT-VERIFY: <requirement> — <why>` line.
  The caller resolves each one itself before the gate; an unresolved CANNOT-VERIFY on an
  acceptance-mapped requirement stands as a gap, not a pass.

## Account for every reviewer: the roster gate

The fan-out is done only when **every roster reviewer is accounted for**: dispatched, or
consciously skipped with a one-line reason. A conditional reviewer that genuinely does not apply
(no UI in the diff → `frontend-reviewer`) is a *recorded* skip, not a silent no-op. That record
is the difference between "reviewed" and "declared done after firing three of seven": the silent
skip is exactly how a needed reviewer never runs.

Record each decision to the footprint as you make it (via `devrites-engine footprint`, as in `$rite-build`). Log the reviewer's
**exact agent name**: the `.codex/agents/` stem, e.g. `devrites-frontend-reviewer`; the `roster`
gate matches on it (stripping the `devrites-` prefix), so a freehand label like `frontend` or
`Spec axis` will not match and the gate reads that reviewer as unaccounted:

- dispatched → `devrites-engine footprint log <slug> reviewer devrites-<x>-reviewer`  (the dispatch record itself)
- skipped → `devrites-engine footprint log <slug> skip devrites-<x>-reviewer`  # e.g. `skip devrites-frontend-reviewer`: no UI in the diff

Then, **before the verdict**, prove the roster is complete:

```bash
devrites-engine footprint roster <slug>   # rc=0 complete · rc=3 a reviewer was neither dispatched nor skipped · rc=1 an always-on reviewer was skipped
```

`rc=3` is the silent omission the gate exists to catch: resolve it by dispatching the reviewer or
recording why it does not apply: never by proceeding with it unaccounted. `rc=1` means an
always-on axis (Spec / Code-review) was skip-recorded; that is legitimate only as a carry-forward
of `$rite-review`'s verdict on an **unchanged** diff: confirm that, don't wave it through.

## Reconciliation

When the subagents return:

1. **Quote verbatim.** Place each subagent's findings under its own `## <axis>` heading in `review.md` / `seal.md`. Do not merge, re-rank, or summarize. `devrites-code-reviewer` runs its **full** documented discipline (tests-first, correctness, readability, architecture, maintainability, standards); the inline lead **reconciles** the returned reports. It does not re-run those same axes itself.
2. **Surface contradictions explicitly.** "Spec axis says complete, Code-review axis says untestable" is a finding, not noise. The caller decides at the gate.
3. **Severity is the gate, not a score.** Sum the labels (`Critical / Important / Suggestion / Nit / FYI`) and apply the caller's gate (`$rite-seal` blocks on `Critical == 0`; `$rite-review` reports counts).
4. **One scale.** All subagents use the same five-label scale (Critical / Important / Suggestion / Nit / FYI). Reject any subagent output that invents its own. **Exception:** `devrites-simplifier-reviewer` deliberately emits only Suggestion / Nit / FYI (it is non-blocking by design), that is a valid subset of the scale, not an invented one; do not reject it during reconciliation.
5. **Consensus roll-up (after the verbatim per-axis record).** Keep every axis's findings verbatim under its `## <axis>` heading (above), then add one deduped roll-up the gate reads: where **≥2 axes flag the same `file:line`**, raise it to the top and mark it *consensus*: independent corroboration raises confidence. A lone low-confidence finding with no `file:line` or evidence anchor drops out of the roll-up (it stays in its per-axis section). The roll-up reduces noise without hiding any axis: the verbatim sections are the audit trail; the roll-up is the actionable summary the gate acts on.

## Model tier

Every reviewer in this fan-out runs at **ceiling tier**: the orchestrator's own model, inherited
by declaring no `model:` in the agent definition (see
[`model-tiers.md`](model-tiers.md)). Adversarial review is exactly where a cheaper model costs the
most: a missed Critical is far more expensive than the tokens saved. Do not downgrade a reviewer to
save cost. Extraction-tier savings belong to scouts (archive-search, footprint), not to the panel.

## Fallback

If the harness has **no subagent primitive at all** (neither Claude's `Task` nor Codex's
`spawn_agent`: an absent tool on one harness while the other's equivalent exists does NOT
qualify), the caller runs the relevant subagent discipline **inline** in its own context and flags the result as a fallback (not an independent review). The seal weighs the fallback differently: see [`../../rite-seal/reference/risk-and-rollback.md`](../../rite-seal/reference/risk-and-rollback.md). This is the [`model-tiers.md`](model-tiers.md) degradation rule applied to the review panel: no subagent primitive → run inline under the same discipline and budget.
