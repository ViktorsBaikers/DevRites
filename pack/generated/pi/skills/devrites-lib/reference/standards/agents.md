# Agent orchestration

> Applies when: dispatching or admitting any devrites-* role; the source-writing boundary.

Follow DevRites policy and [`depth profiles`](../orchestration-profiles.md).

## Authority

<!-- root-only -->
- Root owns scope, questions/decisions/results, `.devrites/**`, phase transitions — not product source/tests; vetted executable workflow artifacts follow [`workflow-artifacts.md`](workflow-artifacts.md).
- Every named role runs; unavailable → HITL, never skip/substitute.
<!-- /root-only -->
- Only bounded wright writes product source/tests; others inspect an immutable candidate.
- Leaves never invoke agents, ask humans, change phase, push, install/deploy, migrate live data, or act irreversibly; they return evidence/proposals for root acceptance. Sole exception: one local unpushed transfer commit by an eligible native-worktree `devrites-slice-wright` — transport, not shipping authority or a checkpoint.

<!-- root-only -->
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
<!-- /root-only -->

<!-- root-only -->
## Native invocation

Skills name exact fresh roles, omit native fields;
hosts spawn/wait/deliver. Root MUST NOT advance/claim completion before admitting required results;
running/orphaned/unavailable = `gap` — no root/generic substitute.
Where a feature workspace exists, every fresh dispatch to a named `devrites-*`
role — wave or solo — builds its read-set with `devrites-engine context --skills-root .pi/skills <slug>
--phase <p> --role <role>` and tracks the launch through `devrites-engine
dispatch <slug> open|start|seal|return` (a solo dispatch is a wave of one;
start/return auto-record metrics). Without a workspace the plain host dispatch
contract above applies and no engine tracking is required.
Workflow-named skills (e.g. devrites-source-driven) are invoked inline in the root context, never dispatched as agents.
<!-- /root-only -->

## Source-writing boundary

Claude grants only wright `acceptEdits`; Codex root is workspace-capable (children cannot elevate). Wright alone `:workspace`; others `:read-only`. Wright gets the smallest exact project-relative source/test list — no directories/globs, traversal, or `.devrites/**`; no scope widening; root rejects `git diff --name-only` extras. Never patch product source/tests in root, bypass/substitute wright, accept drift, or recreate a dispatch bridge.

Isolated-worktree pilot only under [`wright-dispatch.md`](../../../rite-build/reference/wright-dispatch.md#isolated-writer-worktree-pilot): one writer, committed/clean baseline, non-submodule parent, exact transfer commit, candidate reconciliation — never parallel writers nor weaker exact-path admission. Root may materialize only exact Vet-ready workflow-artifact paths per [`workflow-artifacts.md`](workflow-artifacts.md) — not a writer dispatch or candidate mutation.

Each job gets objective/exclusions, exact paths/immutable candidate, rubric/result shape, a **stop condition** (when to return), and the exact result shape; a dispatch missing them is malformed and re-issued. Briefs MUST NOT seed verdict/severity cap/conclusion/suppression. Results state status/scope, outcome, commands/escalation; wright adds paths, changed files, gates, stood decisions; results never widen scope.

**Same-tree concurrent sessions claim first.** Worktree dispatch isolates
parallel batch siblings, but two agent *sessions* (two hosts, two features, or
a human plus an agent) can share one working tree. Before a same-tree writer
starts, its `dispatch open` records an advisory claim with `--claim-session <id>
--claim <path>...` and fails, creating no wave, while a live foreign session holds
a path (without a wave: `claim add --session <id> <paths>`, keep its `<claim-id>`,
then `claim check`); a held claim from a
live foreign session is never merged later or overridden. Unattended, skip only
work that needs those paths, run independent work, and re-check `claim list`
before stopping; stop and report the holder only when nothing independent remains.
On every host-confirmed terminal path after `add` or a claimed `dispatch open` ([unknown outcomes keep it](../parallel-dispatch.md#cancellation-and-terminal-reconciliation)),
the wave's completing `return` releases its claim, `dispatch abandon --release`
releases it on any other (a stop or a refused launch), and a `claim add` claim runs
`devrites-engine claim release --session <id> --id <claim-id>`. Claims expire
by TTL only as crash recovery; `claim list` shows who holds what. Claims are
advisory coordination, never a substitute for the one-writer-per-worktree rule.
Claims cover product paths only; root-authored `.devrites/**` files use
[`workflow-artifacts.md`](workflow-artifacts.md) admission instead.
Expiry is not proof the holder stopped: size `--claim-ttl` or `claim add --ttl <minutes>` to the
expected write (engine max 240) and re-add before it lapses; before writing, also
read `claim list` — a foreign `expired` claim on overlapping paths whose
session is not proven terminal counts as held (stop and report). `dead` (engine-proven
exited holder) or `released` is terminal: reconcile per
[§ Cancellation](../parallel-dispatch.md#cancellation-and-terminal-reconciliation). **Failing case:**
session B's `claim check` passes at minute 35 over session A's still-running wright.

## Independence

- A fresh result sees scope/paths-diff/rubric only — never another result's or the root's conclusions, severities, expected verdicts, chosen verification methods, or edited context; seeding voids the packet. Canonical workspace artifacts in the engine-generated `--role` read-set (state cursor, decisions, `strategy.md` scores, prior review records) are inputs, not seeding: a verdict they record is a historical claim the reviewer re-verifies, never this pass's conclusion. The same holds for status, readiness-checklist, or gate lines inside a read-set artifact (e.g. a `spec.md` `Status:`/`Spec gate:` line from an older workspace). Never hand-edit installed read-sets or swap them for excerpts to strip that history.
- Never message a running read-only leaf: a mid-run note voids it like a seeded packet.
  Root proof that lands after dispatch is weighed at reconciliation, not forwarded; a
  leaf gap it would close (e.g. "no independent runtime run") closes there.
- A recheck packet lists only fingerprints that same role raised, plus the correction
  diff. Another role's finding goes to that role's recheck, never a sibling's packet.
- **Verify the packet is clean before admitting the account.** Confirm the returned packet/trace carries no host-injected context: an automatic memory/summary banner naming an earlier phase verdict (e.g. `past Temper PASS … next is /rite-define`) or harness reminder text seeds the pass and voids it. Cancel that account — never admit it — name the lost coverage, and re-dispatch with the host's context injection disabled for that child. Tool config changes that do not stop native child auto-injection are not a fix; verify on the next spawn instead of assuming.
- A parent-context pass contributes attributed evidence but is not independent: exclude it from independent accounting and name the lost coverage.
- Final severity is set at reconciliation after re-verifying the claimed consequence at the cited site (reviewer severity advisory); dismissals record a reason, and true facts about neighboring code route elsewhere instead of being dismissed.
- Conflicting required results are arbitrated by re-verifying evidence at the site; the deciding evidence is recorded, truly unresolved conflicts stay open blockers. Two results describing the same defect merge only when both defect and fix path match; otherwise the more conservative classification stands, and severity never widens without new site evidence — an overridden or demoted finding stays visible with the reason, never silently merged away.

## Result admission

Each required reviewer/analyst/auditor starts with exactly one:
`Outcome: findings`, `Outcome: no-findings`, or `Outcome: gap`.

**Canonical finding shape (C2 — all `devrites-*-reviewer` / auditor agents):**

```text
Outcome: <findings | no-findings | gap>
Counts: critical=<n> important=<n> suggestion=<n> nit=<n> fyi=<n> gap=<n>
Finding: <severity> conf <n> | <file:line or artifact section> | <observed quote/result> | when <input/state/sequence> → <effect> | breaks <ID or standard §> "<quoted line>" | <minimum fix>
Basis: <files read · commands run to reach this finding>
```

- **`findings`:** each row uses the shape above; confidence 1–10 on Critical/Important.
  Critical/Important requires 7+, exact evidence, concrete impact, and a non-empty
  `Basis` — a correct verdict reached by an unrecorded inspection path is unproven.
  A Critical/Important row carries `conf <n>` in its severity cell and writes impact
  as `when <input/state/sequence> → <effect>`; missing either, root admits it as
  `Unverified: <claim> — missing trigger` (stays `gap`).
- **Proportionality:** a Critical/Important row's `breaks` quotes the line it violates:
  a spec ID (`AC`/`REQ`/`INV`/`PROH`/`DEC`/`EDGE`) or a shipped standard rule
  (`<standard>.md § <section>`: security, regression, principles). Root anchors the
  quote like the `file:line` quote. A repair contract, recheck packet, earlier
  finding or plan item (`plan.md`, `tasks.md`, `architecture.md`, `test-plan.md`)
  only restates root's own claim and is never an anchor; quote the spec ID or
  standard rule it traces to. No anchor, or one
  the defect does not break,
  admits the row as Suggestion logged `Deferred: <claim> — no spec anchor` for an
  optional follow-up slice; it never drives a repair round. Hardening no spec line
  asks for is a Suggestion, however plausible. The `when` must also be reachable:
  a hostile actor enters only through a trust boundary the feature really has
  (users, network, external systems, untrusted files). A trigger that needs one it
  never faces — the code under test forging its own evidence, a tampering second
  user on a single-owner dev host, a swapped remote daemon under a local dev tool —
  is a Suggestion too. So is an environment state the feature's declared
  environment does not have (a mount option, kernel setting or daemon location
  absent from the target host): guard against it only when the spec names it.
  A finding never pulls a deferred slice's scope forward.
- **`no-findings`:** `No-findings:` names checks and inspected evidence. Bare
  pass, empty list, or “looks good” is malformed.
- **`gap`:** names missing/unreadable/stale input; skipped/failed required check;
  tool/reviewer failure; or another limit. Required gaps block. A reviewer gap
  that asks for more evidence without showing a defect is required only when it
  quotes the anchor that demands that evidence, as Proportionality does: a spec ID,
  an owner decision, or a shipped standard rule. Without one, root logs it
  `Deferred: <claim> — no anchor`. Advisory gaps follow
  [security.md § Reachability sets severity](security.md#reachability-sets-severity).
- **Transport failure is not a finding.** A read-only leaf aborted, timed out, lost,
  or with host-rejected output is `gap` quoting the host error. Re-dispatch one fresh
  child with the same brief and read-set — never a resume of the failed child. The
  retry spends any user-granted budget; ask only if it would exceed it. A second
  transport failure of the same role is a host defect: stop for HITL with both errors
  and `/rite-doctor`. Never design capture or retention machinery around it.

Root treats results as claims; verifies proposed Critical/Important blockers
against candidate, first anchoring each row's quote at its cited `file:line` on the
frozen digest: exact → admit; found at exactly one other location → relocate,
keeping both locations; not found or found at several → `Unverified: <claim> —
anchor lost|ambiguous`, a gap that is never a finding or a dismissal. A leaf result
produced by spawning a nested agent is `gap`, and each hidden child counts toward
the concurrent pool and `max_agents`. Missing fields become
`Unverified: <claim> — missing <proof>` and stay `gap` until verified/rejected
with evidence—never silently dropped/demoted. Null/timeout/failure/malformed
output is `gap`, never `no-findings`. Conditional `Not-applicable` must name the
inspected scope and why its trigger did not fire. **Failing case:** parent admits a
wright "done" with no `git diff --name-only` ⊆ path-contract (allowlist) observation.

### Return budget

Every read-only leaf returns rows, not prose, so the root keeps its context for
decisions and nothing is lost in a retelling. Exempt: `devrites-plan-drafter` (its
return is the bundle the root writes) and the wright's change evidence.

- Lead with the result line (`Outcome:` for reviewers, the role's own status line
  otherwise), then `Counts:` with the number of rows per severity or kind.
- A reviewer, auditor or test-analyst head is strict: line 1 is
  `Outcome: findings|no-findings|gap` (no title line above it), line 2 is
  `Counts: key=N ...` with keys from the role's closed vocabulary:
  `critical important suggestion nit fyi gap`, plus `contract mechanism` for
  `devrites-plan-reviewer`, only `suggestion nit fyi gap` for
  `devrites-simplifier-reviewer`. `findings` needs a non-`gap` count above 0;
  `no-findings` needs every count at 0. `dispatch return --result <file>` stores
  the return and records a malformed or inconsistent head as `gap`, never
  `no-findings`.
- No restated task, preamble, approach narrative or closing summary. An optional
  `Notes:` block carries at most five lines that no row can hold.
- Point, don't paste: cite `file:line` with at most one quoted line (160
  characters); for a command, give the exact command, its exit code and the one
  decisive output line. Never paste whole files, diffs, logs or test output.
- Each row stands alone: a reader of that row alone can re-check it from its pointer.
- The root records admitted rows verbatim, adds its own verification beside them, and
  never rewrites, merges or summarizes a row; deduplication references rows by
  position or ID instead of rewording them.

<!-- root-only -->
## Reconciliation

Root rejects stale/out-of-scope work, dedupes by root cause, records
accepted conclusions, and reruns only affected proof/review. Reconciliation runs
only after every dispatch reaches a terminal classification
([parallel-dispatch.md](../parallel-dispatch.md)); findings that do not survive stay
visible as `withdrawn: <n> (<reason>)` in the account rather than being silently
dropped, so a later reader can audit what was discarded and why.
Route defects before correction: product/acceptance → Spec Drift Guard/Clarify;
architecture/slicing/proof → Plan/Vet; source/test → wright;
external/pre-existing → blocker/defer. Never patch downstream to hide an upstream
defect.
<!-- /root-only -->
