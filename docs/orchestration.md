# Orchestration patterns

DevRites delegates bounded work to agents while keeping authority in the root
orchestrator. The governing contract is
[`standards/agents.md`](../pack/.claude/skills/devrites-lib/reference/standards/agents.md);
this page summarizes that model.

## Authority and topology

- The active public `rite-*` skill is the root. It alone owns human questions,
  decisions, routing, gates, reconciliation, canonical `.devrites/**` writes,
  phase changes, and irreversible or external actions.
- Orchestration is flat depth one: only the root dispatches. Leaves never
  dispatch leaves.
- Normally at most three read-only leaves run concurrently against one frozen
  candidate. The root awaits every required return before mutation.
- There are **18 named roles**: 17 read-only leaves and one source/test writer,
  `devrites-slice-wright`.

| Group | Roles |
|---|---|
| Bounded work leaves | `devrites-evidence-scout`, `devrites-plan-drafter`, `devrites-proof-runner`, `devrites-upgrade-planner` |
| Strategic/claim challenge | `devrites-strategy-reviewer`, `devrites-plan-reviewer`, `devrites-doubt-reviewer` |
| Review/audit | `devrites-spec-reviewer`, `devrites-code-reviewer`, `devrites-test-analyst`, `devrites-frontend-reviewer`, `devrites-security-auditor`, `devrites-performance-reviewer`, `devrites-devex-reviewer`, `devrites-simplifier-reviewer` |
| Comparison/history | `devrites-forge-judge`, `devrites-retrospector` |
| Sole writer | `devrites-slice-wright` |

Every dispatch uses file-backed `agent-packet/v1` and `agent-result/v1`
envelopes. They record exact inputs, scope, budgets, immutable baseline
identity, side effects, and terminal status. The root rejects malformed, stale,
or out-of-scope returns before adding anything to canonical state.

## Dispatch and enforcement

Use the first safe option available:

1. the named project role; Codex V2 uses its exact `agent_type`, a unique
   `task_name`, and a durable rollout proving native rules, wait, and result;
2. a generic V1 `explorer`/`worker` only when the host still enforces read-only
   or exact wright scope (or isolated staging);
3. a HITL stop when neither safe fresh-context option is callable.

Specialist work never runs in the root context. Declared leaf identity is
fail-closed: a missing or crashed `devrites-engine` guard blocks the tool call.
Every installed skill also carries `required-agent-roles` frontmatter. Codex
arms those unconditional roles when the user invokes the skill and blocks Stop
until each role has a confirmed start, wait, and non-empty result. `none`
explicitly declares that the skill has no unconditional fresh-agent requirement;
conditional fan-out still follows the phase's documented triggers.
Leaves never ask the human, write `.devrites/**`, change phase, commit, push,
install, deploy, migrate live data, or perform irreversible actions.

## Single-writer build lifecycle

The root derives normalized project-relative source/test paths and writes them,
one per line, to `.devrites/work/<slug>/.wright-allowlist`. It is an exact
authorization manifest and permits no directories, globs, traversal,
duplicates, or `.devrites/**`. The wright's returned file list records what
changed but cannot grant access.

Each dispatch follows the retained-baseline sequence:

1. `reconcile snapshot` captures the original dirty-tree baseline, private Git
   objects, exact allowlist, and canonical-state fingerprint.
2. The sole wright returns code/tests; the root validates its typed identity and
   exact changed-file set.
3. `reconcile check` rejects anything outside the root allowlist.
4. `test-integrity` and `package-existence` run against the same retained
   baseline.
5. After proof and decision checks pass, `reconcile close` retires the private
   window; only then does the root write canonical records.

On retry, the root may add only an accepted still-in-slice path, then runs
`reconcile snapshot` again. This refreshes the dispatch boundary while
preserving the original slice baseline. The same objective root cause has a
durable three-failure budget in `recovery-attempts.jsonl`; exhaustion records a
technical blocker with reproduction and dead ends, not a retry-approval
question.

Build asks the human only for product/scope/policy choices, irreversible risk,
or human-only access/actions. Tests, types, lint, runtime/browser failures,
missing coverage, and workflow-tool defects stay agent-owned.

## Isolation

Parallel shared-tree writers are forbidden. A vetted `Forge: yes` slice may
compare two or three complete strategies only through
`devrites-engine forge`:

1. `plan` validates the scorecard, pins the physical repository/base, writes
   the sole `devrites-forge/v1` ownership manifest, and creates derived
   worktrees outside `.devrites` **before** reconciliation snapshots.
2. A real host adapter declares `manifest-env-v1`. Each candidate is bound
   all-or-none to the manifest run, candidate, worktree cwd, branch, live
   worker ID, PID, and the engine's `forge process-token` value.
3. `record` and `extract` make every terminal full-tree delta immutable; the
   read-only judge records one winner; `merge` preflights and lands exactly it.
4. Normal reconcile, integrity, doubt, and proof gates record verification
   before `cleanup`. The human `forge-report.md` is written afterward and owns
   nothing.

An ineligible layout returns a typed serial degradation before side effects.
After planning, errors preserve manifest-owned state for recovery. Cleanup and
reap enumerate only manifests, preserve live, dirty, mismatched, or ambiguous
state, and never delete candidate branches. Ordinary features may also run in
separate user worktrees so their `ACTIVE` cursors and source trees do not
collide.

## See also

- [`architecture.md`](architecture.md): full layer model.
- [`command-map.md`](command-map.md): every role and phase owner.
- [`flow.md`](flow.md): lifecycle and state diagrams.
- [`ADR-0010`](adr/0010-agent-first-fresh-context-orchestration.md): decision
  record and rejected alternatives.
