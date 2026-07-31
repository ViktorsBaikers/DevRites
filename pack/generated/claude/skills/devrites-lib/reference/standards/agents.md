# Agent orchestration

Follow DevRites policy and [`depth profiles`](../orchestration-profiles.md).

## Authority

- Root owns scope, questions/decisions/results, `.devrites/**`, and phase
  transitions—not source/tests.
- Only bounded wright writes source/tests; others inspect an immutable
  candidate.
- Every named role runs; unavailable → HITL, never skip/substitute.
- Leaves never invoke agents, ask humans, change phase, commit/push,
  install/deploy, migrate live data, or act irreversibly; return evidence/proposals
  for root acceptance.

## Agents

| Agent |
|---|
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

Skills name exact fresh roles, omit native fields; hosts spawn/wait/deliver.
Root MUST NOT advance/claim completion before admitting required results.
Running/orphaned/unavailable = `gap`; no root/generic substitute.

## Source-writing boundary

Claude grants only wright `acceptEdits`; Codex root is workspace-capable because
children cannot elevate. Wright alone is `:workspace`; others are `:read-only`.

Give wright the smallest exact project-relative file list—no directories/globs,
traversal/`.devrites/**`. No scope widening. Root rejects
`git diff --name-only` extras. Never patch in root, bypass/substitute wright,
accept drift, or recreate a dispatch bridge.

## Inputs and results

Each job gets objective/exclusions, exact paths/immutable candidate, rubric/result
shape. Briefs MUST NOT seed verdict/severity cap/conclusion/suppression.
Results state status/scope,
outcome, commands/escalation; wright adds paths, changed files, gates, stood
decisions. Results never widen scope.

## Result admission

Each required reviewer/analyst/auditor starts with exactly one:
`Outcome: findings`, `Outcome: no-findings`, or `Outcome: gap`.

- **`findings`:** each states severity, confidence 1–10, exact artifact section or
  `file:line`, observed quote/command/result/measurement, reachable failure or
  contract impact, and smallest correction. Critical/Important requires 7+, exact
  evidence, and concrete impact.
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
