# Orchestration

DevRites defines workflow semantics; Codex and Claude provide native
orchestration.

```text
AGENTS.md / CLAUDE.md / skills
              ↓
host interprets the requested workflow
              ↓
host runs .codex/agents/<role>.toml or .claude/agents/<role>.md
              ↓
host waits, follows up, and delivers results
              ↓
root reconciles semantics; engine checks deterministic structure/state/safety
```

## Authority

- The public `rite-*` skill is the root orchestrator. It owns scope, human
  questions, decisions, result reconciliation, `.devrites/**` artifacts, phase
  transitions, and explicit irreversible-action approval.
- Build maintains the strict candidate manifest. Prove, Review, and Seal bind
  their artifacts to the engine-computed digest; Polish owns final durable
  rollups and affected re-proof; Ship is candidate-read-only.
- The root never writes source or tests. Claude enforces that with plan mode;
  Codex enforces it as workflow policy because the parent must be
  workspace-capable for its writer child to execute.
- The role catalog has one source/test writer, `devrites-slice-wright`. Claude
  and Codex make only that exact specialist writable.
- Leaves never invoke leaves, ask the human, change phase, commit, push, deploy,
  migrate live data, or write canonical workflow state.

Skills name the exact role and bounded task in normal language. The host owns
internal spawn fields, scheduling, waiting, follow-ups, and result delivery.
DevRites does not parse rollout logs or maintain dispatch receipts.

Quick, Standard, and Full select evidence/review depth only; they are not agent
API versions or execution tiers. The shared contract lives in
[`orchestration-profiles.md`](../pack/.claude/skills/devrites-lib/reference/orchestration-profiles.md).
Persisted state schemas remain explicitly versioned because readers need
stable data contracts; no compatibility protocol is
inserted between a skill and the host's native agent call.

## Native permission boundary

Claude project settings use `permissions.defaultMode: plan`; its slice-wright
profile uses `permissionMode: acceptEdits`. Codex uses a workspace-capable root
because children cannot elevate above the parent permission ceiling. Its exact
slice-wright uses `default_permissions = ":workspace"` and the other 16
specialists use `default_permissions = ":read-only"`.

If the exact role is unavailable, the root stops instead of substituting a
generic agent or doing specialist/writer work inline.

## Result closure

Each leaf's final response is its complete admission packet. It repeats every
finding, outcome, evidence citation, limitation, and gap that the root must
reconcile; an earlier message followed by a bare `done` or a link is malformed.
The root owns collection and accounts for every required or applicable role.
Timeout, absence, failure, or malformed output remains a gap, and a successful
sibling cannot erase it.

An evidenced clean result is valid. Reviewers must account for their rubric and
inspected scope, but they never manufacture a finding to fill a quota.

## Slice-wright lifecycle

For Claude or Codex, the root states the smallest exact project-relative source
and test paths directly in the dispatch task. Directories, globs, traversal,
symlink escapes, duplicates, and `.devrites/**` are invalid.

1. The root records the pre-dispatch `git diff --name-only`.
2. The host runs the exact `devrites-slice-wright` and waits for its result.
3. The root compares the returned file list and `git diff --name-only` with the
   task contract and rejects any extra path.
4. The root inspects the test diff for deletion, skipping, focus markers, or
   loosened assertions.
5. Accepted changes receive repository/CI proof and applicable read-only review.
6. The root records exact candidate rows and canonical evidence/state. Proof
   must name a positive, discriminating assertion and decisive observed signal;
   skipped/zero/assertion-free/tautological/unexecuted/exit-only results cannot
   establish behavior.

Exact-path scope is instruction-backed on both hosts; native sandboxes provide
the broader writer/read-only split. An unauthorized delta is rejected and the
same bounded writer must restore it before work continues. The root never
widens the contract or rewrites source. Parallel source writers in one worktree
are forbidden.

## Engine boundary

The engine owns deterministic:

- local managed install, update, and uninstall against caller-supplied
  candidates;
- strict project-candidate validation/digesting, content-bound readiness, and final
  structural plus exact-binding checks;
- atomic answer/drop/batch resolve and transactional close mutations;
- secret scanning and version reporting.

Native hosts, skills, exact agents, repository tools, or CI own:

- agent discovery, dispatch, scheduling, waiting, follow-up, and results;
- semantic readiness, traceability, acceptance/evidence quality, doubt, review
  reconciliation, test-quality assessment, capability interpretation, semantic
  upgrade, and recovery routing;
- normative spec grammar re-read, qid allocation, Clarify cursor edits, AFK
  budget accounting, recovery attempt accounting, and read-only install/host
  diagnosis;
- exact-release bundle/binary acquisition before invoking local engine install
  operations;
- repository inspection, search, and test/build/lint/release execution;
- repository JSON/schema/generated-artifact validation;
- session history, compaction, status prose, progress, plugins, and memory.

The engine has no agent bridge, semantic readiness protocol/digests, heuristic
prose parser, capability-ledger interpreter, compatibility telemetry, migration
command, or old aliases.

Provider/consumer changes use the existing plan and traceability artifacts: one
canonical contract plus provider- and consumer-side asserting tests that both
consume it. Spec capability impact and lossless MODIFIED folding remain with
Spec/Polish. These rules add no dispatch protocol, schema registry, or phase.

Any harness adapter must expose the same read-only `check candidate` command,
preserve exact binding lines and the Build/Prove/Polish/Review/Seal/Ship role
boundaries, and stop rather than substitute another hash or inline specialist.

See [ADR-0015](adr/0015-read-only-root-native-orchestration.md),
[ADR-0017](adr/0017-native-codex-writer-agent.md),
[ADR-0018](adr/0018-native-sandbox-instruction-writer-boundary.md),
[ADR-0020](adr/0020-thin-engine-native-orchestration-boundary.md),
[ADR-0022](adr/0022-native-orchestration-thin-engine.md), and
[`standards/agents.md`](../pack/.claude/skills/devrites-lib/reference/standards/agents.md).
