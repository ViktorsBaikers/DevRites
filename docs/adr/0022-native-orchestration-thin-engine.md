# ADR-0022: Native orchestration with a thin deterministic engine

- **Status:** Accepted
- **Date:** 2026-07-31

## Context

Claude Code and Codex already load project skills and exact custom agents. They
own agent discovery, dispatch, scheduling, waiting, follow-up, and result
delivery. Re-implementing that protocol in Go adds a second authority and
creates version, receipt, polling, and fallback machinery with no durable
product state to protect.

Semantic engine gates had the same problem. Heuristics that parsed readiness,
traceability, acceptance, review prose, assertion counts, doubt accounts, or
capability deltas duplicated judgments already made from fuller context by the
active skill and exact reviewers. Repository test, build, lint, typecheck, and
release validation likewise belong to the repository and CI.

Release history also showed that the proposed compatibility surface was too
broad. Released v1.0.0–v2.6.1 writers used
`.devrites/work/<slug>/state.md` bullet cursors with `Phase`, `Next step`, and
`qid`; v3 writers kept `work/<slug>/` and used the canonical map plus table
cursor fields `phase`, `next_action`, and `question_id`. `.devrites/features`,
alternate map/cursor/proof filenames, status-as-phase, and phase aliases were
not official released writer outputs. Early v3 migration code could create
extra copies, but it retained the canonical workspace and did not make those
copies authoritative.

Write-on-read local compatibility telemetry could not answer whether the global
installed population still used a format. It also made an otherwise read-only
operation mutate private project state. A journaled preview/apply/rollback
migration system therefore preserved speculative formats while adding more
risk and code than a small tolerant reader.

## Decision

- Claude Code and Codex own agent discovery, dispatch, scheduling, waiting,
  follow-up, and result delivery. Installed skills and exact custom agents own
  all semantic judgment: readiness, traceability, acceptance and evidence
  quality, doubt, review reconciliation, test-quality assessment, capability
  interpretation, semantic upgrade, and recovery routing.
- The Go engine retains only deterministic cross-host primitives:
  install/update/uninstall, `snapshot`, structural `check readiness`, structural
  plus evidence-freshness `check seal`, grammar `check spec`, atomic
  `state resolve|clarify|tick-afk|recovery|close`, `secret-scan`, `doctor`, and
  `version`.
- The engine has no agent bridge, semantic readiness V2 or digests, heuristic
  prose parser, capability-ledger engine, compatibility telemetry, migration
  command, or old command aliases/tombstones.
- Runtime compatibility is a direct, non-mutating read of only the official
  bullet and table `state.md` cursor forms described above. Semantic upgrades
  are preservation-first native workflow edits, not structural engine
  migrations.
- Explicit schema versions remain where persisted readers need them. The
  `devrites.workspace.v1` snapshot and current state schemas remain supported;
  orchestration does not acquire a DevRites protocol version.
- Native writer permissions and exact-path contracts remain in force.
  Structural gates remain independent of the model. Seal GO, AFK, or an
  autocomplete flag never authorizes an irreversible action: the exact current
  commit/push/tag/PR plan still needs fresh user approval for that attempt.

This ADR supersedes ADR-0019's semantic seal/test/reviewer/doubt gate clauses,
ADR-0020's broader gate and safe-migration engine scope, and ADR-0021's
telemetry plus preview/apply/rollback design. It retains their native
orchestration, writer/permission, structural-gate, root-safety, and fresh
irreversible-action approval rules.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep semantic Go gates as a second opinion | String and count heuristics see less context than the assigned native reviewers and create conflicting authorities. |
| Keep the transaction migration platform for possible legacy layouts | Release evidence supports only two cursor encodings; a tolerant reader is smaller and non-mutating. |
| Keep write-on-read telemetry until usage reaches zero | Local records cannot measure global usage and violate read-only expectations. |
| Move deterministic state and secret safety into prompts | Atomicity, root containment, evidence freshness, and secret handling need testable cross-host enforcement. |
| Retain old aliases for a transition | It preserves the oversized public surface and hides callers that must migrate. Unknown-command failure is explicit. |

## Consequences

The engine has one narrow, stable reason to exist: deterministic state and
safety primitives shared by both hosts. Skills and exact agents can evolve
semantic workflows without a Go protocol migration. Repository CI remains the
source of observed engineering proof.

Older official workspaces remain readable without a write, import step, or
migration journal. Unsupported pre-release aliases fail visibly instead of
silently becoming a permanent compatibility contract. Removing public commands
is intentionally breaking; canonical skills and documentation must move in the
same release.
