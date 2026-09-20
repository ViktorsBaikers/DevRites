# Analysis: Workflow Artifact Identity

## Deterministic cross-artifact result

- Spec gate: passed.
- Decision coverage: CLEAR after DRIFT-003 and DRIFT-004 fold-back.
- Plan approval: recorded.
- Workspace schema after hardening: passed.
- Legacy `devrites-engine analyze` command: unavailable in current engine; no pass claimed.
- REQ-001–REQ-009, AC-001–AC-007, EDGE-001–EDGE-007, PROH-001–PROH-012, SLICE-001, EVID-001–EVID-008, and T-001–T-017: no orphan found.

## Terminology and module discipline

`module`, `interface`, `implementation`, `depth`, `seam`, `adapter`, `leverage`, and `locality` match `CONTEXT.md` and codebase-design discipline. Existing `workflow-artifacts.md` remains sole deep module. Deletion redistributes admission, identity, owner/concurrency, operation, proof, retry, evidence, diagnostics, resume, and product-separation rules across ten canonical adapters and two generated hosts. Interface is exact grammar plus finite tables; test crosses same seam.

No second module, generic materializer, engine command, compatibility wrapper, dependency, schema, feature flag, active hook, semantic Go policy, actual workflow executable, or completed-workspace backfill is introduced.

## Initial independent findings and fold-back

| Finding | Severity / confidence | Resolution |
| --- | --- | --- |
| Stage/backup writes lacked durable complete-write state and interruption coverage | Critical 9/10 | strict create/write/mode/file-sync/dir-sync intents; full-write loop; invalid-progress/error/death matrix; declared partial recovery |
| `FAILED` had no retry transition or finite exhaustion | Critical 10/10 | immutable epochs; accepted correction + re-preflight; death-safe handoff; same-fingerprint count 3; no attempt 4; exhaustion GC |
| `CLEANED` conflicted with missing retained source | Critical 9/10 | source GC required before terminal states; source absence after `CLEANED` expected; source-free idempotent verify |
| Whole `evidence.md` replacement could erase lifecycle proof | Critical 10/10 | exact marker-owned section; generation/preimage compare; prefix/suffix/EVID preservation; one candidate binding |
| Default generator path could delete full generated tree | Critical 10/10 | private same-filesystem output only; complete staged-tree validation; recoverable exact 22 install; outside manifest equality |
| Test-local executor risked becoming second authority | Important 9/10 | canonical machine-readable tables; parsed filesystem driver; independent fixed tuple oracle; bidirectional mutations |
| Concurrent roots were undefined | Important 10/10 | persistent no-follow mode-0600 owner lock; nonblocking exclusion; lock held through final journal generation |
| Proof timeout and process descendant cleanup were undefined | Important 9/10 | fresh process groups; command/aggregate bounds; terminate/grace/force-kill/reap; bounded output |
| Retained bundles could accumulate | Important 9/10 | namespace/cardinality limit; success/exhaustion GC; one canonical/recognized temp only |
| Admission/journal serialization, byte encoding, caller route actions, and diagnostics were underspecified | DevEx blockers 9–10/10 | exact grammars/golden vector; ordered classifier; route/action/cursor table; ten adapter rows; finite reason/boundary taxonomy |
| Root-caller path was not measurable | DevEx blocker 9/10 | one-target Prove walkthrough with verbatim output, interrupted resume, stale authority, cursor return, and observed TTHW |

All repairs preserve accepted behavior and one-slice/16-authored/22-generated scope. DRIFT-003 and DEC-009–DEC-014 record technical hardening; no human-owned gate or acceptance expansion.

## Prior-candidate composition

Build-entry overlap freeze found 13 Workflow Artifact destinations already owned by the sealed uncommitted Acceptance-preserving Reslice candidate: five authored preimages (`pack/.claude/skills/rite-autocomplete/SKILL.md`, `pack/.claude/skills/rite-autocomplete/reference/loop.md`, `pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md`, `pack/.claude/skills/rite-vet/SKILL.md`, and `tests/instruction-size-baseline.json`) plus the Claude/Codex generated mirrors of the four skill paths. Workspace Observation has zero destination overlap.

DRIFT-004 and DEC-015 replace the false byte-identical claim and initial authored-only count with an exact composition contract. All shared bytes are preimages; the sole wright must edit the five authored paths additively, derive the eight mirrors only through normal private-stage generation, preserve Reslice linkage, packet, route, action, stop, and baseline semantics, pass `tests/acceptance-preserving-reslice-policy-test.sh` before `COMMITTED`, and leave prior workspace records plus all 30 non-overlap candidate paths exact. The previous standalone Reslice candidate digest becomes historical after shared-byte change and cannot be cited as current proof. No destination, acceptance criterion, product behavior, or writer authority was added.

## Build-entry preflight

Observed:

- Bash 5.3.15, Python 3.14.6, Node 24.18.0, Go 1.26.4.
- Workspace schema: `OK: 1 workspace(s) validated` after fold-back.
- Behavioral schema: 13 files, 72 scenarios, zero failures before source change.
- Instruction baseline: 217 instruction files, 854997/855000 bytes before source change.
- Shell, Node, and Python script syntax: passed before source change.
- Race-enabled engine compilation with no selected tests: passed before source change.
- Generator contract: `DEVRITES_HOST_ARTIFACT_DIR` is consumed by generator and exercised by host test.
- Authoritative module manifest is `engine/go.mod`; root `go.mod` is absent and not treated as prerequisite.

New dedicated test/corpus are Build-owned outputs. Their exact commands, cwd, expected signals, self-built-engine fallback, provenance, and failure matrix are closed in `test-plan.md`.

## Candidate protection

Verified unchanged:

- `.gitignore`: `2af674b07482b76740df8ac9fe46913ab73ef1751e4c1496e3342d86b2230781`
- `.devrites/ACTIVE`: `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`
- Workspace Observation manifest: `2dca74484895de119cd935db6c3692782df9173eef199c88a7d5a65898332ec9`

Workspace remains detached from `.devrites/ACTIVE`. No source, test, generated, Git, remote, release, Ship, or close-out mutation occurred during Vet.

## Narrow recheck and final fold-back

Plan recheck confirmed five Critical repairs plus proof timeout, but found exact implementation gaps: ambiguous oracle schema/self-attestation; unspecified lock bootstrap/primitive; pre-journal stale canonical after binding rollover; root-owned generated installation crossing writer seam; internally impossible limits. DevEx recheck confirmed grammar/encoding/routes/diagnostics, but found module Go prerequisite mismatch and no distinct TTHW command/interval.

Final technical fold-back, with no third reviewer loop:

- exact five-field oracle and separate OS/engine observer process; every table column/oracle field mutates independently;
- umask-077 mkdirat/openat bootstrap and only Python `fcntl.flock(LOCK_EX|LOCK_NB)`; first-create/interruption/alternate-lock/unsupported-host fixtures;
- lock-held `.stale-cleanup` plus atomic stale-cleaning rename and idempotent pre-journal binding-rollover GC;
- bounded wright authors the candidate and the hash-bound driver is the sole 16-authored/22-generated filesystem writer through the private feature-local delivery journal; root freezes/verifies only; DEC-042 allows exact coordinator launch, not destination writes or recovery, under the measured host-window exception;
- checked relational limit minima/headroom/overflow fixtures;
- observed module-selected Go 1.26.5 as already-available prerequisite;
- exact `bash tests/workflow-artifact-identity-test.sh --prove-walkthrough`; TTHW command entry to first `CLEANED` cursor, 90000 ms prediction; recovery cases outside interval.

No acceptance, slice, 16/22 destination, dependency, engine behavior, product, or release scope changed. DRIFT-004 adds only the prior Reslice composition gate. Final deterministic schema, coverage, protected-hash, and readiness-digest gates remain before Build.

## 2026-08-22 post-prove re-vet

Frozen planning identity before fold: plan `b9c4b044…`, tasks `7b60d9d3…`, spec `106b7621…`. Product candidate unchanged `bee44b1ada3b975839e90c79d9d04a10cf7b3b48b1ec3deed8f0f99931441fd9` (38 files).

Deterministic:
- `devrites-engine snapshot` / `analyze` / `readiness-digest` unavailable (unknown command).
- `python3 scripts/validate-workspace-schema.py .devrites/work/workflow-artifact-identity` → `OK: 1 workspace(s) validated`.
- Legacy `devrites analyze workflow-artifact-identity` → verdict clear (warns on missing `[ACn]` tags; this spec uses `AC-001` WHEN/THEN).
- Principles file absent → gate passes.
- Terminology: module/interface/adapter/seam match CONTEXT.md; no second module or engine command.

Independent plan review (iter 1): Outcome findings. Floor thin on Architecture → BLOCKED. Accepted technical folds (DEC-051): Shared contract proof table; exact adapter JSON comment grammar already enforced by dedicated test; N/A admission sentence; inspect-and-OUT SHA freeze of `README.md` / `docs/skills.md`; protected ACTIVE `fc0dd2b2…`. Rejected: 38-file allowlist growth; operator-doc edits; TTHW 90s→60s or excluding self-build; spec demo example expansion; new dedicated-test fixture.

Independent DevEx predict: Important copy-paste spec-example gap rejected as this slug admits no active target; `--prove-walkthrough` remains the measured root-caller surface. Suggestions recorded, not folded.

Narrow recheck: Outcome no-findings; accepted findings closed; floor adequate → PASS. Schema OK after fold. Engineering binding `Readiness inputs SHA-256: 7e93bcdedaace7e309af08cc8084ca1f9dc0c9b8e0f3a3c06abd9bdf8238fbdb`. Coverage digest command unavailable; `decision-coverage.md` remains CLEAR without that field.

## 2026-08-22 post-EVID-014 readiness re-vet

Independent `devrites-plan-reviewer`: Outcome no-findings; floor adequate → PASS. No acceptance, slice, destination, or product-scope change. Deterministic `check readiness` PASS after recording live `Readiness inputs SHA-256: 40372e0b1eaa9ec63eeaf252ec4b2ce4574ef3f1814355725652056a63059225`. `devrites-engine snapshot` / `analyze` / `readiness-digest` remain unavailable; binding via `check readiness --emit-binding`.
