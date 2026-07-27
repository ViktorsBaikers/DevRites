# Parallel review dispatch

This file defines reviewer rosters for `$rite-review` and `$rite-seal`. The shared
packet, result, capability ladder, concurrency limit, retry, and reconciliation rules live in
[`standards/agents.md`](standards/agents.md); do not restate them here.

All reviewers read the same **immutable candidate**, but no more than three run
concurrently. For a larger panel, use awaited batches of at most three; after resource
or process errors, reduce the batch or run serially. A batch boundary does not permit a
write.

## Roster

`$rite-seal` accounts for all seven roles. `$rite-review` runs the first two.
Conditional triggers use the listed signals.

| Reviewer | Fires | Trigger |
|---|---|---|
| `devrites-spec-reviewer` | always | review Spec axis; seal may carry it forward only on the identical candidate |
| `devrites-code-reviewer` | always | review Code-review axis; seal may carry it forward only on the identical candidate |
| `devrites-test-analyst` | always at seal | completed feature |
| `devrites-frontend-reviewer` | conditional | diff touches a component, template, stylesheet, story, route, screen, or design token |
| `devrites-security-auditor` | conditional | input, auth/authz, data access/storage, external integration, permission, dependency, or secret |
| `devrites-performance-reviewer` | conditional | spec has a performance budget, or diff adds a query, growing-set loop, render/hot-path work, or material payload |
| `devrites-devex-reviewer` | conditional | public API, CLI, SDK/export, webhook, config/env, error message, docs, or getting-started path |

Not in this roster: `devrites-simplifier-reviewer` (polish), `devrites-doubt-reviewer`
(doubt), `devrites-strategy-reviewer` (temper), `devrites-plan-reviewer` (vet),
`devrites-forge-judge` (build), and `devrites-retrospector` (ship).

## Hit-rate gating

Before conditionals, run:

```bash
devrites-engine reviewer-stats report
```

- `run`, `run (always-on)`, or `run (insurance — never gated)`: apply the roster.
- `gate-candidate`: a conditional produced zero surviving findings in its last 10+
  dispatches. Record a skip with `gated: zero surviving findings in <N> dispatches`.
- `--full`: dispatch every triggered reviewer despite gating.

Always-on reviewers and insurance reviewers (security and doubt) are never hit-rate
gated. After reconciliation, record each dispatched reviewer's surviving
Critical+Important count:

```bash
devrites-engine reviewer-stats record devrites-<x>-reviewer <count> <slug>
```

## Reviewer packet

Create `agent-packet/v1` per `standards/agents.md`. In addition to the common fields:

```yaml
objective: Audit the active feature on <axis>; derive expected behavior independently.
inputs:
  - path: .devrites/work/<slug>/spec.md
    purpose: acceptance contract
  - path: .devrites/work/<slug>/touched-files.md
    purpose: feature boundary
  - path: <scratch_root>/candidate.diff
    purpose: reviewed candidate
scope:
  in: [<axis-specific workspace paths>, <touched source/test paths>]
  out: [author reasoning, sibling findings, unrelated repository debt]
  allowed_repo_writes: []
```

Tell the role to apply its documented discipline and return `payload.type:
review-findings`, one finding per line, labeled Critical / Important / Suggestion /
Nit / FYI and anchored to `file:line` plus the spec criterion or observed command.
The exact result `CANNOT-VERIFY: <requirement> — <why>` is never a pass.

Before dispatching `devrites-devex-reviewer` in measured mode, the root runs the
documented quickstart in an isolated clean checkout and adds immutable commands,
timings, output, candidate identity, and log hashes to the packet. Before
dispatching `devrites-performance-reviewer`, the root supplies the immutable diff
and any already-authorized measurement artifacts. Reviewers validate these inputs;
their read-only identity never executes quickstarts, builds, browser runs, or
assignment-only shell setup.

Dispatch rules:

- Fresh-context dispatch through the capability ladder; one packet per reviewer.
- No author context, sibling findings, severity coaching, or plan justification.
- Feature scope is `touched-files.md` plus the frozen diff.
- Await every required batch before any write or verdict. Never background/detach.

## Account for every reviewer

Log exact agent names as decisions are made:

```bash
devrites-engine footprint log <slug> reviewer devrites-<x>-reviewer
devrites-engine footprint log <slug> skip devrites-<x>-reviewer
devrites-engine footprint roster <slug>
```

`roster` rc=3 means an unaccounted reviewer: dispatch or record why it does not apply.
rc=1 means an always-on skip; this is valid only for an unchanged-candidate
carry-forward. A required reviewer that cannot be dispatched blocks the phase.

## Reconciliation

1. Preserve each valid report verbatim under `## <axis>` in `review.md`/`seal.md`.
2. Surface contradictions explicitly; the root decides them.
3. Keep the shared five-label scale. The simplifier's Suggestion/Nit/FYI subset is valid.
4. Resolve every `CANNOT-VERIFY` before the gate or retain it as a gap.
5. Add a deduped roll-up only after the verbatim record. Mark matching `file:line`
   findings from two or more axes as consensus; do not hide lone findings.
6. Apply the caller's gate; do not invent a composite score.

Every panel reviewer runs at the ceiling or inherited tier. If the named role is
unavailable, use the universal ladder. If no fresh-agent rung is available, stop for
HITL; never execute the reviewer in the root context.
