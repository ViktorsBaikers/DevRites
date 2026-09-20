# Questions: Workflow Artifact Identity

## Question register

| Question ID | Status | Gate | Question | Answer | Impact |
| --- | --- | --- | --- | --- | --- |
| Q-001 | answered | validating | Where does Workflow Artifact authority live? | Deepen existing `workflow-artifacts.md`; ten canonical callers become thin adapters; Claude/Codex remain generated adapters. | Binds REQ-001 and DEC-001. |
| Q-002 | answered | validating | Who authors workflow-only executable bytes? | Controlling root authors exact Vet-admitted bytes during disposable preflight; no product wright, planner, or reviewer authors them. | Binds REQ-002/003 and DEC-002/003. |
| Q-003 | answered | validating | What controls identity and resume? | Current Vet binding plus stable source resolver, no-follow held-descriptor bytes, frozen path/mode/hash, and durable current journal state; stale writer history and completed workspaces carry no authority. | Binds REQ-003–REQ-008 and DEC-003–DEC-006. |
| Q-004 | answered | validating | How does transaction survive failure and process termination? | Intent precedes operations; pre-proof installed failure rolls back then failure-cleans; success proves then cleans without rollback; source loss follows pre-install, install-through-proving, or post-proved class. | Binds REQ-005–REQ-007 and AC-003/004. |
| Q-005 | answered | validating | How is host behavior proved without false model claims? | Ten provider-neutral scenarios, deterministic filesystem/engine fixtures, normal generation, and parity; live pinned runs remain optional and claim-bounded. | Binds REQ-007/008 and DEC-007/008. |
| Q-006 | answered | validating | How should the exact pre-`COMMITTED` delivery be recovered after unrelated recursive outside-manifest drift makes normal rollback unreachable? | Preserve the full recursive success guarantee. Authorize one bounded rollback of only the validated 16 authored and 22 generated destinations from exact journal backups, record outside drift as failed evidence without altering it, then retry synchronously with repository-local bytecode disabled. | Binds DRIFT-013 and DEC-021; does not narrow REQ-009/AC-004. |
| Q-007 | answered | validating | How should delivery `61fe47e0…` release its 38 installed destinations after a transient full-suite failure and unrelated recursive outside drift made normal rollback unreachable? | Preserve the full recursive success guarantee. Authorize one bounded rollback of only this journal's validated 16 authored and 22 generated destinations from exact backups, record all outside drift without altering it, then retry as a distinct synchronous delivery after a new baseline. | Binds DRIFT-015 and DEC-023; does not generalize DEC-021 or narrow REQ-009/AC-004. |
| Q-008 | answered | validating | How should delivery `8c7bd216…` repair the 11 ignored bytecode modes changed by explicit proof compilation? | Restore only the 11 byte-identical journal-selected modes from `0644` to frozen `0600`, then use the retained candidate driver only if its exact recursive predecessor still exists. Preserve any later host-lifecycle differences rather than broadening rollback. | Binds DRIFT-021 and DEC-031; authority is exact to this journal and does not narrow REQ-009/AC-004. |
| Q-009 | answered | validating | How should failed delivery `be1bd63a…` restore its 38 destinations when its only outside-snapshot delta is the required DRIFT-026 successor? | Preserve and physically recheck the exact 33-record successor addition, restore only the journal-selected 16 authored and 22 generated preimages, validate the recursive snapshot-plus-successor union, then finalize this journal from `RESTORED` to `FAILED`. | Binds DRIFT-026 and DEC-034; authority is exact to this journal and never creates a candidate-directory exclusion. |
| Q-010 | answered | blocking | Successor proof observed `.devrites/ACTIVE` SHA-256 `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267` (`workflow-artifact-identity`) instead of frozen `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`. Accept the current hash as the new protected preimage, or keep the frozen hash? | Accept current ACTIVE hash as the new protected preimage. Do not rewrite ACTIVE. | Binds protected baseline and successor root proof. The 38-file candidate does not include ACTIVE. Feature forbids restoring user-owned ACTIVE. |
| Q-011 | answered | blocking | Q-010 cannot execute against candidate `76700e28…` because the dedicated test still requires frozen ACTIVE `9ef52cca…`. Restore ACTIVE for this proof, or authorize a new candidate? | Authorize a new candidate | Supersedes Q-010. Changing the test would change the 38-file candidate digest. Exact `devrites-slice-wright` is unavailable on this host. |
| Q-012 | answered | blocking | Q-011's test-only ACTIVE-hash candidate cannot reach CLEANED delivery. How should prove continue? | Plan-repair empty generated delta for unchanged generator inputs | DEC-048 revisits DEC-027 for that class. Spec AC-007 and test-plan already say differences subset of 22. Wright updates real-delivery admission only. |
| Q-013 | answered | blocking | DEC-042 install of `aff21c0c…` reached durable FAILED at delivery gate-1 (workspace schema: EVID-008 and EVID-010 unmapped). Destinations already match expected-post. Traceability now maps those IDs and schema passes. DEC-024 forbids deleting/renaming failed history, journal reuse, and comment-only digest churn. How should Prove continue? | Preserve FAILED `aff21c0c…` and authorize one bounded wright to add an asserting workspace-schema lock to the dedicated test, creating a distinct retry digest, then prepare+install with no post-prepare gap | Binds DEC-024 vs a workspace-only proving failure after Seal-correction delivery. |
| seal-important-accept | answered | validating | Important findings remain. Proceed to seal? [y/N] | N — keep NO-GO; fix nested `claude`/`codex` OUT_ROOT basename plant via bounded wright then `$rite-prove` | Operator chose fix path via `$rite-build`; binds DEC-056. |

## Q-010
status: answered
slice: SLICE-001
gate: blocking
question: Successor proof cannot start because `.devrites/ACTIVE` is `workflow-artifact-identity` (SHA-256 `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`) instead of the frozen protected preimage `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`.
options: |
  1. Accept current ACTIVE hash as the new protected preimage (Recommended) — logic: candidate never owned ACTIVE · risk: do not rewrite the user-owned file · architecture: update only the workspace runner expected hash, then rerun successor proof
  2. Keep the frozen hash — you restore ACTIVE to the frozen bytes for the proof window; agent will not write ACTIVE
proposed: Accept current ACTIVE hash `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267` as the new protected preimage; do not rewrite ACTIVE.
raised_at: 2026-08-22T00:00:00Z
answered_at: 2026-08-22T00:00:00Z
answer: Accept current ACTIVE hash

## Q-011
status: answered
slice: SLICE-001
gate: blocking
supersedes: Q-010
question: Q-010 cannot be executed against frozen candidate `76700e28…`. `tests/workflow-artifact-identity-test.sh` still requires `.devrites/ACTIVE` SHA-256 `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`. Changing that test would create a new candidate.
options: |
  1. Restore ACTIVE to the frozen bytes for this proof window (Recommended) — logic: successor proof of 76700e28 needs the candidate's own protected check · risk: you write ACTIVE, agent will not · architecture: no candidate change
  2. Authorize a new candidate that updates only the dedicated test's expected ACTIVE hash — logic: keep current slug permanently · architecture: not successor proof of 76700e28 · risk: new identity, Vet/proof restart
proposed: Restore ACTIVE to frozen bytes for this proof window; agent will not write ACTIVE.
raised_at: 2026-08-22T00:00:00Z
answered_at: 2026-08-22T00:00:00Z
answer: Authorize a new candidate

## Q-012
status: answered
slice: SLICE-001
gate: blocking
supersedes: Q-011
question: Q-011 authorized a new candidate that updates only `tests/workflow-artifact-identity-test.sh` `LIVE_PROTECTED_SHA256` for `.devrites/ACTIVE` to `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`. That edit is on disk (driver SHA-256 `2346a80ee6f3a8978e4774e9ad33bd9d061c738f6d151e2938e5863301235569`; engine candidate `ef07a774abe21eaedfb4244248f2dac0986d74ec2486daf5081b099547a7a08a`). Delivery cannot CLEANED: umask-022 private generation of current canonical sources differs from live `pack/generated` at 0 files, and DEC-027 rejects both an empty generated delta and fabricated generated differences. Journal `570f004b8893a67b718b395620544f8f53461e93f4e36e74ded761f9968312a7` remains SNAPSHOTTING after DEC-042 install failed at `delivery outside manifest binding`; the sole outside mismatch was `.code-review-graph/graph.db` (`5a096e71…` → `d52381e5…`). Generated replacements were not started. Root did not alter graph.db or destinations.
options: |
  1. Plan-repair empty generated delta for unchanged generator inputs (Recommended) — logic: honors Q-011 test-only delta · risk: revisits DEC-027 for a defined class, not a silent skip · architecture: keep nonempty replacement proof in fixtures/historical deliveries; wright changes only the real-delivery admission `bool(differences)` then a distinct uninterrupted prepare+install
  2. Expand Q-011 with a real canonical generator-input edit — logic: keeps DEC-027 nonempty-delta rule · risk: a second authored change Q-011 did not name; fabricating a module tweak is also rejected · architecture: human names the exact canonical path/bytes
  3. Restore ACTIVE to frozen `9ef52cca…` / `thin-engine-native-codex` for 76700e28 proof — logic: original Q-011 option 1 · risk: you write user-owned ACTIVE; workspace leaves this feature; the Q-011 test edit must be reverted by wright · architecture: agent will not write ACTIVE
proposed: Plan-repair empty generated delta for unchanged generator inputs; then wright admission + distinct prepare/install with no post-prepare gap.
raised_at: 2026-08-22T00:00:00Z
answered_at: 2026-08-22T00:00:00Z
answer: Plan-repair empty generated delta for unchanged generator inputs (Recommended)

## Q-013
status: answered
slice: SLICE-001
gate: blocking
question: DEC-042 install of authored aggregate `aff21c0c80e683395937b81e16bd1b4c1dbd5c950ad3e9e2edf781959147f5a1` reached durable `FAILED` at delivery gate-1. Exact error: `traceability.md: evidence ID EVID-008 from evidence/browser proof is not mapped` and the same for `EVID-010`. Gate-0 toolchain passed. Destinations already match this journal's expected-post (preimage equals desired bytes; empty generated delta). Root mapped EVID-008 and EVID-010 in `traceability.md`; `python3 scripts/validate-workspace-schema.py .devrites/work/workflow-artifact-identity` now exits 0. DEC-024 forbids deleting/renaming failed history, retrying the failed journal, and comment-only digest churn, so the same digest cannot re-prepare a new sidecar.
options: |
  1. Preserve FAILED `aff21c0c…` and authorize one bounded wright to add an asserting workspace-schema lock to `tests/workflow-artifact-identity-test.sh` (not a comment), creating a distinct retry digest, then prepare+install with no post-prepare gap (Recommended) — logic: same DEC-024 retry shape as prior failed deliveries · architecture: failed history stays at its digest-keyed directory · risk: the 38-file engine candidate changes again and must be re-frozen
  2. Amend DEC-024 for this class only: destinations already equal expected-post and the only remaining defect was `.devrites/**` evidence mapping — allow one re-prepare/reuse of `aff21c0c…` after the mapping — logic: no authored churn · architecture: revisits journal-identity rule · risk: weakens failed-history preservation
  3. Stop and `$rite-plan unblock` — logic: DEC-024 has no legal retry · risk: Seal-correction candidate remains unproved
proposed: Preserve FAILED `aff21c0c…`. Wright adds an asserting workspace-schema check to the dedicated test so the retry digest is distinct; then one uninterrupted prepare+install.
raised_at: 2026-08-22T09:40:00Z
answered_at: 2026-08-22T09:45:00Z
answer: Preserve FAILED aff21c0c… and authorize one bounded wright to add an asserting workspace-schema lock to tests/workflow-artifact-identity-test.sh (not a comment), creating a distinct retry digest, then prepare+install with no post-prepare gap

## seal-important-accept
status: answered
slice: SLICE-001
gate: validating
question: Important findings remain on candidate `8c8cb87c…` (nested `claude`/`codex` pathname plant under open artifacts cwd). Proceed to seal? [y/N]
options: |
  1. N — keep NO-GO; fix via bounded wright then `$rite-prove` (Recommended) — logic: DEC-049 forbids sealing named Important · security: same-UID plant of OUT_ROOT basenames still divertable · architecture: plan Inspected-and-OUT blocks editing `scripts/build-host-artifacts.sh`; fix stays in dedicated test harness
  2. Y — accept only this named Important; rewrite seal Verdict GO — risk: residual plant window remains · business: unlocks `$rite-ship` without closing the finding
proposed: N — keep NO-GO; fix nested OUT_ROOT basename plant via bounded wright then `$rite-prove`.
raised_at: 2026-08-22T20:16:07Z
answered_at: 2026-08-22T23:30:00Z
answer: N — keep NO-GO; fix nested `claude`/`codex` OUT_ROOT basename plant via bounded wright then `$rite-prove`
