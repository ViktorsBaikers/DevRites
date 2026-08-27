# Agent orchestration

Follow DevRites policy and [`depth profiles`](../orchestration-profiles.md).

## Authority

- Root owns scope, questions/decisions/results, `.devrites/**`, phase transitions — not product source/tests; vetted executable workflow artifacts follow [`workflow-artifacts.md`](workflow-artifacts.md).
- Only bounded wright writes product source/tests; others inspect an immutable candidate.
- Every named role runs; unavailable → HITL, never skip/substitute.
- Leaves never invoke agents, ask humans, change phase, push, install/deploy, migrate live data, or act irreversibly; they return evidence/proposals for root acceptance. Sole exception: one local unpushed transfer commit by an eligible native-worktree `devrites-slice-wright` — transport, not shipping authority or a checkpoint.

## Agents

| Agent |
| --- |
| `devrites-evidence-scout` |
| `devrites-plan-drafter` |
| `devrites-upgrade-planner` |
| `devrites-proof-runner` |
| `devrites-strategy-reviewer` |
| `devrites-plan-reviewer` |
| `devrites-doubt-reviewer` |
| `devrites-spec-reviewer` |
| `devrites-code-reviewer` |
| `devrites-test-analyst` |
| `devrites-frontend-reviewer` |
| `devrites-security-auditor` |
| `devrites-performance-reviewer` |
| `devrites-devex-reviewer` |
| `devrites-simplifier-reviewer` |
| `devrites-retrospector` |
| `devrites-slice-wright` |

Files own briefs; [`parallel-dispatch.md`](../parallel-dispatch.md) owns rosters.

## Native invocation

Skills name exact fresh roles, omit native fields;
hosts spawn/wait/deliver. Root MUST NOT advance/claim completion before admitting required results;
running/orphaned/unavailable = `gap` — no root/generic substitute.

## Source-writing boundary

Claude grants only wright `acceptEdits`; Codex root is workspace-capable (children cannot elevate). Wright alone `:workspace`; others `:read-only`. Wright gets the smallest exact project-relative source/test list — no directories/globs, traversal, or `.devrites/**`; no scope widening; root rejects `git diff --name-only` extras. Never patch product source/tests in root, bypass/substitute wright, accept drift, or recreate a dispatch bridge.

Isolated-worktree pilot only under [`wright-dispatch.md`](../../../rite-build/reference/wright-dispatch.md#isolated-writer-worktree-pilot): one writer, committed/clean baseline, non-submodule parent, exact transfer commit, candidate reconciliation — never parallel writers nor weaker exact-path admission. Root may materialize only exact Vet-ready workflow-artifact paths per [`workflow-artifacts.md`](workflow-artifacts.md) — not a writer dispatch or candidate mutation.

Each job gets objective/exclusions, exact paths/immutable candidate, rubric/result shape. Briefs MUST NOT seed verdict/severity cap/conclusion/suppression. Results state status/scope, outcome, commands/escalation; wright adds paths, changed files, gates, stood decisions; results never widen scope.

## Independence

- A fresh result sees scope/paths-diff/rubric only — never another result's or the root's conclusions, severities, expected verdicts, or edited context; seeding voids the packet.
- A parent-context pass contributes attributed evidence but is not independent: exclude it from independent accounting and name the lost coverage.
- Final severity is set at reconciliation after re-verifying the claimed consequence at the cited site (reviewer severity advisory); dismissals record a reason, and true facts about neighboring code route elsewhere instead of being dismissed.
- Conflicting required results are arbitrated by re-verifying evidence at the site; the deciding evidence is recorded, truly unresolved conflicts stay open blockers.

## Result admission

Each required reviewer/analyst/auditor starts with exactly one:
`Outcome: findings`, `Outcome: no-findings`, or `Outcome: gap`.

**Canonical finding shape (C2 — all `devrites-*-reviewer` / auditor agents):**

```text
Outcome: <findings | no-findings | gap>
Finding: <severity> | <file:line or artifact section> | <observed quote/result> | <impact> | <minimum fix>
```

- **`findings`:** each row uses the shape above; confidence 1–10 on Critical/Important.
  Critical/Important requires 7+, exact evidence, and concrete impact.
- **`no-findings`:** `No-findings:` names checks and inspected evidence. Bare
  pass, empty list, or “looks good” is malformed.
- **`gap`:** names missing/unreadable/stale input; skipped/failed required check;
  tool/reviewer failure; or another limit. Required gaps block.

Root treats results as claims; verifies proposed Critical/Important blockers
against candidate. Missing fields become
`Unverified: <claim> — missing <proof>` and stay `gap` until verified/rejected
with evidence—never silently dropped/demoted. Null/timeout/failure/malformed
output is `gap`, never `no-findings`. Conditional `Not-applicable` must name the
inspected scope and why its trigger did not fire.

## Reconciliation

Root rejects stale/out-of-scope work, dedupes by root cause, records
accepted conclusions, and reruns only affected proof/review. Route defects before
correction: product/acceptance → Spec Drift Guard/Clarify;
architecture/slicing/proof → Plan/Vet; source/test → wright;
external/pre-existing → blocker/defer. Never patch downstream to hide an upstream
defect.
