# Native review dispatch

> Root-only: not inlined in --role bundles.

Host dispatch; DevRites governs.

## Roster

Review: Spec/Code, triggered frontend/security/performance, and the seal roles that can
judge the frozen digest (test-analyst, doubt, devex); Seal: all applicable roles, citing
same-candidate Review accounts.

| Reviewer | Fires | Trigger |
|---|---|---|
| `devrites-spec-reviewer` | always | acceptance/scope |
| `devrites-code-reviewer` | always | implementation correctness |
| `devrites-test-analyst` | seal | completed feature |
| `devrites-frontend-reviewer` | conditional | UI/route/screen/style/design-token change |
| `devrites-security-auditor` | conditional | input/auth/data/permission/dependency/integration/secret |
| `devrites-performance-reviewer` | conditional | performance budget or hot path/query/growing set |
| `devrites-devex-reviewer` | conditional | public API/CLI/SDK/webhook/config/error/docs/onboarding |

Also: `devrites-simplifier-reviewer`, `devrites-doubt-reviewer`,
`devrites-strategy-reviewer`, `devrites-plan-reviewer`, or `devrites-retrospector`.

If the candidate diff has no matching surfaces, record
`Not-applicable: no relevant paths in diff` for `frontend-reviewer`,
`security-auditor`, and `performance-reviewer`; do not dispatch them.

## Host reservations and subwaves

Apply this protocol in every phase, including Spec/Clarify preparation, Build,
review, proof and recovery. Root supplies `DEVRITES_HOST_CAPACITY=<positive integer>`
per engine invocation from the actual native host's total concurrent actor budget.
It is an internal capability, not a user parallelism option; never derive it from
`--parallel` or `max_parallel`. Absent means 1; malformed values fail closed.
Account for every retained/open/unreturned role across sources and phases, including
unknown outcomes and restarts between native spawn and recorded start. Workspace
pool slots and writer leases do not substitute for these actor reservations.

Before `open`, determine the complete required roster and available host slots.
Reserve all roles in each wave together. If the roster will not fit, split it
**before opening** into complete, independently sealable subwaves with fresh IDs.
Keep the full phase review barrier across subwaves: all required accounts must
return before reconciliation or correction. Never open a wave whose remaining
roles cannot launch, then wait for a return that requires that wave to seal.
Render each subwave's packets against the same protected candidate and preserve
coverage; constrained capacity changes scheduling, not required roles.

For each role, run `devrites-engine dispatch <source> check-spawn --wave <w>
[--role <r>]` immediately before the actual native spawn. It rechecks retained
host reservations and immutable pool source/parent/position binding. Then spawn
through the native host and immediately record `dispatch <source> start --wave <w>
--role <r> --handle <id>`. If authority changes after launch, still record the
actual late start: it is a host fact, not renewed admission. Changed eligibility
never prevents settlement; return/abandon old actors through normal terminal
reconciliation. A successful check is not proof that a host launched the actor.

Autocomplete follows the [rolling pool and writer priority](parallel-workspaces.md#continuous-global-frontier):
admitted writers get host slots first; preparation uses remaining slots. Retain
honest resource deferrals and refresh after each actor settlement, not whole-wave
completion. On resume, reconstruct retained reservations before any new opening.
Serialize root authority/AFK edits and use controller-locked engine mutations;
leaves return source-owned artifacts rather than writing shared workflow state.

## Dispatch contract

1. Freeze one candidate with `devrites-engine candidate freeze <slug> --wave <w>
   (--slice <SLICE-ID> | --paths <path>...) [--base <ref>]` (writes
   `packets/<w>.diff` and `packets/<w>.sha256`); give each reviewer the same spec, paths, and diff.
   Parallelize only when writes/state/locks/tool sessions/scarce resources
   (ports/processes/quotas/FDs/browsers/MCP) are compatible. Frozen read-only
   usually is; pressure requires batches or serial work.
   One fresh-context preflight before fan-out checks host configuration/evidence for
   inherited history/memory/sibling summaries. Use supported clean-context settings;
   unverified independence is a gap. Do not launch a probe agent per task;
   still validate each result's context and candidate.
2. Build each reviewer's read-set once with
   `devrites-engine context --skills-root .devin/skills <slug> --phase <p> --role <role> --trigger <set>` and
   render each reviewer, auditor, and test-analyst packet (other roles, such as
   `proof-runner`, keep their own) with `devrites-engine packet <slug> --wave <w> --phase <p>
   --role <role> [--slice <SLICE-ID>] [--focus <file>]`: it names the bundle and frozen
   diff by reference, copies the digests, and runs the `open` lints. Build roles add `--slice <SLICE-ID>` (slice-scoped, slice-named bundle).
   Packet rendering and dispatch verify each named diff's recorded SHA-256.
   Manually authored `.diff`/`.patch` packets need explicit
   `diff <64hex> <path>` binding rows too; an unbound diff is refused.
   Evaluate the skill's trigger conditions and pass each that fires — a role's
   Trigger column maps to the phase manifest's names: `security-auditor` →
   `security`; at Review, `frontend-reviewer` → `ui` and `performance-reviewer` →
   `performance`. Seal declares neither; those roles load their own standards.
   The `agents` dispatch contract needs no flag: `--role` auto-fires it
   (`auto=[agents]` in the output). The command prints `unselected=[...]`; an
   omitted applicable trigger is a gap, not a shortcut.
3. Reserve a complete subwave under [host reservations](#host-reservations-and-subwaves).
   Open a single-use wave; every retry needs a fresh ID:
   `devrites-engine dispatch <slug> open --phase <p> --wave <w> --role <r>...`.
   Reviewer/auditor/test-analyst waves add `--packet <file>` naming any rendered
   `packets/<w>.<role>.md`; give each agent its own packet unedited. `open` rejects
   seeded gate results, outcomes, counts, or pass claims. Build reviewers add
   `--slice <SLICE-ID>`; Review/Seal packets bind the frozen diff and a manifest
   covering every checkpointed path. Same-tree writers add `--claim-session <id>
   --claim <path>...` per [`wright-dispatch.md`](../../rite-build/reference/wright-dispatch.md#run).
   Run `dispatch <slug> check-spawn --wave <w> --role <r>` immediately before each
   native spawn; launch all roles in that reserved subwave nonblocking before waiting.
   Never shell-launch host CLIs (`omp -p`, `claude -p`): that bridge's handle dies
   with the session and its return is a gap. Immediately record every actual handle
   with `dispatch <slug> start --wave <w> --role <r> --handle <id>`; the last seals
   the wave (explicit `seal` then does nothing). Return only after seal, with the
   recorded handle: `dispatch <slug> return --wave <w> --role <r> --handle <id>`.
   Reviewers add `--result <file>`: retain it verbatim at
   `packets/<wave>-<role>.return.md` (keep an existing file), derive outcome and
   `Finding:` keys. If the host cannot provide a file, use explicit
   `--outcome findings|no-findings|gap`. Status is nonzero for open/sealed/abandoned
   waves; start/return record metrics. If handles/seal cannot be obtained,
   `dispatch ... abandon --wave <w> --reason <reason>` before recording the gap;
   add `--release` only after host-confirmed stop/refused launch.
4. Ask for each exact named agent in fresh context; omit native syntax and
   `agent_type`/`task_name`/`fork_turns` from the instruction.
5. Every dispatch reaches a terminal classification before reconciliation:
   `Outcome: findings`, `Outcome: no-findings`, `Outcome: gap`, or — for a
   conditional step that could not run at all — its NOT-RUN marker recorded in
   `review.md`/`seal.md` per [core.md § Gate contract](standards/core.md#gate-contract);
   an unavailable required role stops for HITL. Reconciling over a partial roster
   is forbidden. Admit one account per role label per candidate: only the return
   bound to the sealed wave's handle counts; a double delivery or a return from an
   abandoned or re-issued wave is recorded `duplicate`/`stale`, never as a second
   account or corroboration. **Failing case:** abandoned-wave and re-dispatched
   code-reviewers both return, and "two reviewers agree" dismisses a spec finding.
   Findings later withdrawn at reconciliation stay visible: the account records
   `withdrawn: <n> (<reason>)` rather than silently dropping them.
6. Apply [[`agents.md`](standards/agents.md) § Result admission](standards/agents.md#result-admission)
   and dedupe by evidence/root cause. Account for all seven (and, in `seal.md`, `devrites-proof-runner` and `devrites-doubt-reviewer`) in
   `review.md`/`seal.md`, each [engineering lane](#engineering-lanes) separately:
   `Outcome:` plus admitted account, or
   `Not-applicable: <inspected-scope reason>`. Seal cites, rather than re-dispatches, any
   role `review.md` admitted on the identical candidate under
   [evidence validity](candidate-integrity.md#evidence-validity); a changed digest or newly
   triggered surface needs a fresh account. Persist no dispatch telemetry.
7. Missing profile, null/failure, or malformed output is a `gap`; never
   infer approval or substitute inline review.

On exhaustion/contention: stop spawning; collect running results; use complete
subwaves or serial work. Never restart/orphan the cohort or infer approval.
Reviewers are read-only; accounts store evidence, never telemetry. Same-file writers
stay serial. Capacity rejection is backpressure, not permission to shrink a roster.

## Vet affected recovery

Before a Vet recovery dispatch, root validates the return cursor, accepted prior
findings and exact owner, repaired candidate identity, changed clauses/paths/criteria,
affected drift/evidence, and delta self-check under [Vet steps 1b and 7](../../rite-vet/SKILL.md).
Keep prior admitted wave/role/handle/return provenance in the root-owned
`eng-review.md` recovery record, never in reviewer inputs. Missing provenance uses
the initial applicable review; changed global contracts or unknown impact require
full applicable review. Do not manufacture a recovery claim to narrow that review.

For each exact owning reviewer requiring a fresh read-only recheck, freeze the
repaired candidate and cover all that owner's open fingerprints in one dispatch.
Build context, render the packet, then open a fresh wave in this order:

```text
devrites-engine context --skills-root .devin/skills <slug> --phase vet --role <owner> --recheck packets/<fold>.<owner>.inventory.md [--slice <id> ...] [--trigger <applicable-set>]
devrites-engine packet <slug> --wave <fresh-wave> --phase vet --role <owner> --recheck [--slice <id> ...] --focus packets/<fold>.<owner>.focus.md
devrites-engine dispatch <slug> open --phase vet --wave <fresh-wave> --role <owner> [--slice <id> ...] --packet packets/<fresh-wave>.<owner>.md
```

Repeat `--slice <id>` once per affected slice in the same order in all three
commands; never pass a joined slice token. Zero slices means plan-level affected
recovery, not an initial Vet. Select every applicable context trigger. If context
requires a trigger retry, preserve the recovery inventory, each selected slice,
and all other selected options while adding the suggested applicable triggers.
Continue with the dispatch contract's native launch, handle, seal, and return admission.

The inventory supplies neutral reproduction, each owner fingerprint, correction
scope, exact source/rule citations, and dependency-plus-traceability closure.
Focus rows stay `- <ID> <path:line> <text>`. Neither packet, inventory, nor focus
may contain prior verdicts, prior scores, or `*.return.md` links. Root checks
semantic completeness; engine flag-shape, readability, and integrity checks do not.
Each account must cover dependents of changed clauses, not only edited slices,
and return an explicit correction-created `Regression:` verdict. The plan reviewer
returns affected dimension bands, with unchanged dimensions outside that account,
not newly passed. Root retains valid prior coverage and uses the lowest applicable
band across current accounts without seeding reviewer scores. Every unbuilt
slice still needs write-set closure. Vet's mechanism-only exception, same-seam
rules, and correction/fingerprint and per-slice caps remain unchanged; this
sequence adds no reviewer or discretionary retry round.

## Engineering lanes

Classify each changed path by the runtime it executes in — from entry points, build
targets, and the import graph, never filename or language: `frontend`
(client-rendered, browser, or native UI runtime), `backend` (server, worker, CLI, or
data runtime), `boundary` (a serialized contract between them), or `other` (build
tooling, libraries, game engines: own profile, no forced split; native app UI is
`frontend`). Record the lane map
in `review.md`/`seal.md`.

- **Frontend and backend both changed:** dispatch `devrites-code-reviewer` twice in
  same wave when capacity permits (`dispatch open ... --role code-reviewer.frontend
  --role code-reviewer.backend`), otherwise use complete host-reserved subwaves
  under the capacity rule below. Distinct handles and recorded start times, not
  wave names, prove concurrency. Build each bundle with `devrites-engine
  context --skills-root .devin/skills <slug> --phase <p> --role code-reviewer --trigger <set> --out
  ctx/<p>-code-reviewer.<lane>.bundle.md`; the default name collides. Each packet
  names its lane, the lane's path set (the whole diff stays readable for context),
  the triggered profile ([`review/README.md`](standards/review/README.md)), and the
  contracts crossing the lane. Otherwise one code-reviewer, lane recorded.
- **Reconcile per contract:** each lane returns `Contract:` rows; root pairs both
  sides of each contract. A contract assessed from one side only is a `gap`; a
  missing crossing test is Important.
- **Accounting:** each lane is its own account; a combined verdict never hides a
  failed or unassessed lane, and a blocked question on one lane does not stop the
  other. A mixed file may be reviewed by both lanes; writes stay with one wright.
  `devrites-frontend-reviewer` (craft/a11y) supplements, never substitutes, the
  frontend lane.
- **Capacity:** temporary backpressure keeps the batch rule above. If the host
  cannot launch concurrent children at all, record `lane-concurrency: unavailable
  <reason>` and ask the user once before dispatching lanes serially; record the
  answer in `decisions.md`.

**Failing case:** one code-reviewer covers a full-stack diff from the handler side;
the client renames a field its mocked test still passes, and the verdict reads clean.

## Cancellation and terminal reconciliation

Cancellation acknowledgement or timeout leaves execution unknown. Keep handles,
path/resource ownership, and capacity until the native host confirms termination
and returned work is reconciled. Query/wait; never replace children, release slots,
or reuse resources prematurely. If unproven, record a gap and retain ownership;
earlier-session claim proof follows. Outcome classification alone proves no death.

On resume (compaction or a new session), run `devrites-engine dispatch <slug>
status`, `parallel status`, and `claim list` before any dispatch. A read-only
wave whose handles this session cannot query: `abandon` it, record each unreturned
role `gap`, and queue a fresh wave on the current digest. Its unknown actors still
reserve host capacity until terminal reconciliation; abandon alone frees nothing.
Admit replacement only when capacity and independence permit; late returns are
`stale`. A writer retains claim/lease: no relaunch under a fresh wave,
same session, or expired TTL until release or proven termination. For an
unqueryable writer, `claim list` `dead` proves its holding session exited on this
host; `claim list --all --session <id>` shows `released` (bare `list` counts it).
After that proof, `dispatch <slug> abandon --wave <w> --release --reason <reason>`
stamps `terminal_at` even without a claim, not a return. Record `gap`, reclaim
paths, and re-dispatch. `live`/`expired` stays held; cancellation acknowledgement,
timeout, ordinary abandonment, or replacement cannot prove writer death.
A wright that never returned an
admitted result leaves unadmitted bytes: task-path changes since its pre-dispatch
`git diff --name-only` baseline are never cited as built nor discarded, and are
named in the re-dispatch contract; changes outside task paths stop for the human
(Converge side: [`rite-converge`](../../rite-converge/SKILL.md)).

Finalization settles every launched role, including proof/custom/writer roles.
Abandoned proof/reviewer replacement needs a newer settled account, same canonical
phase/role/exact unordered slice set, and distinct handle. Writer termination
admits no bytes and bypasses no candidate, admission, review or proof gate.

## Scale limits

The role table bounds rosters. Use at most ten concurrent read-only reviewers,
counting children any leaf spawned, and fit them within the shared actual host
actor budget alongside writers and all other phases. Refuse a larger swarm and
role substitution: queue complete subwaves while preserving required coverage.
Overlapping reviewers increase duplicate findings and reconciliation work.

## Children are tools, not peers

A dispatched agent is a tool invocation: the parent owns the prompt, the
expected artifact, and admission. Reviewers do not negotiate with each other
or with the implementer. A mesh without a parent is not a DevRites profile.
**Failing case:** two reviewers agree a compromise in a shared channel and the
parent records that compromise as independent review.

Arbitration/independence → [agents.md § Independence](standards/agents.md#independence); writer batches → [`parallel-batch.md`](../../rite-build/reference/parallel-batch.md).
