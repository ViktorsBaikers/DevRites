---
name: rite-autocomplete
description: Run the full DevRites lifecycle unattended; --ship continues to the final Git approval boundary. Use for one-shot autonomous work; not for a single phase.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N] [--full] [--cross-model]"
user-invocable: true
---

# $rite-autocomplete: full lifecycle, unattended

Runs every phase unattended after initial clarification. Irreversible-risk,
blocking/escalating,<!-- pack-scan-ignore: negated statement: gates are NOT disabled -->
and NO-GO gates still pause. Use the **Standard** native execution profile by
default and **Full** for high-risk scope or explicit `--full`; profiles are defined in
[`devrites-lib/reference/orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md).

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` and `.agents/skills/devrites-lib/reference/standards/afk-hitl.md` first.
When the cursor or approved plan contains a consumptive action, also read
`.agents/skills/devrites-lib/reference/standards/one-shot-actions.md` before any
stop/continue or execution decision.
When a blocked cursor names missing executable controller/harness/bundle bytes or
a missing writer under the active feature workspace, read
`.agents/skills/devrites-lib/reference/standards/workflow-artifacts.md` before honoring
the terminal cursor or its recovery count.

## Operating rules
- **Use one initial human window.** Run spec and topology-first clarify; arm AFK
  only after `Decision coverage: CLEAR`. Record later discretionary calls
  ([decision policy](reference/decision-policy.md)).
- **Safety gates are not bypassable.** Irreversible risk, blocking/escalating,
  open validating gates, and unexcepted principle violations pause. `--ship` /
  `--yolo` never authorizes Git; it reaches only the exact-plan literal-GO and
  native-approval boundary. Red checks run bounded recovery, then block unless
  the remaining decision is human-owned.
- **Keep routine backtracking internal.** Always follow agent-owned backward edges
  through repair, Vet, bounded implementation correction, and re-proof inside
  the active run. A nested phase `STOP` or intermediate `Next step` is not a
  user handoff; pause only on the shared real stop conditions.
- **Treat technical readiness as routing, not completion.** `NEEDS_REPLAN` is a backward edge
  to Plan repair and narrow Vet; `NEEDS_CLARIFICATION` is likewise internal when
  decision coverage can be resolved from existing authority. Neither state may
  end Autocomplete unless its underlying fingerprint is actually exhausted or a
  human/safety/access decision is required.
- **Do not confuse an action budget with recovery exhaustion.** After a failed
  consumptive action, zero remaining real attempts blocks only another execution.
  New retained Critical/Important evidence starts offline repair and narrow Vet
  inside this Autocomplete invocation. Stop for a fresh GO only after that repair
  changes the failure condition and Vet is READY.
- **Budget from the post-vet slice count.** Vet may split/add slices.
  `--max-slices N` may lower the cap for a partial run; otherwise build all.
- **Parse flags only from this invocation.** `--ship`, `--yolo`, `--max-slices`,
  `--full`, and `--cross-model` are active only when their exact standalone tokens
  occur in `$ARGUMENTS`; their presence in this skill, examples, or earlier messages
  can never arm them. `--max-slices` must occur once and be followed by a positive
  base-10 integer; missing, repeated, malformed, or conflicting values stop before
  any sentinel or workspace write.
- **Record discretionary choices.** Use the specialist/reviewer recommendation
  and rationale; never choose arbitrarily.
- **Strategic review runs, but never auto-grows scope.** After `$rite-clarify`, run `$rite-temper`
  (significance-gated; it skips low-stakes specs in one line). Unattended it auto-applies only
  `hold-rigor` + `reduce-to-MVP` (these never grow acceptance); **any `expand` is a blocking
  pause**, and irreversible-risk findings always pause. Autocomplete hardens and may *prune* the
  spec on its own; it never *expands* the build's scope without the human.
- **Engineering review runs on every plan, but never auto-grows scope.** After `$rite-define`,
  run `$rite-vet` on **every** feature (depth scales: a light pass on simple plans, full rigor on
  big/risky; never skipped). Unattended it auto-applies only *hardening* findings: added test
  requirements, error-handling / failure-mode coverage, tightened scope, reuse-over-rebuild,
  dependency-order fixes (these never grow acceptance); acceptance-preserving
  reslicing or remediation remains agent-owned. **Any finding that grows product
  scope or changes acceptance is a blocking pause**, and irreversible-risk findings always
  pause. Cross-model is off unless `--cross-model` was armed.

## Workflow
1. **Orient + parse args.** Resolve the explicit or active slug, require its
   `state.md`, and read the cursor directly. Before honoring `blocked` with a
   terminal `next_action`, reconcile retained consumptive-action artifacts and
   per-fingerprint attempts from `drift.md` / `evidence.md`. A distinct retained
   fingerprint below its no-progress cap reopens caller-owned offline recovery;
   a stale pre-ownership workflow-artifact writer stop reopens under
   `workflow-artifacts.md` when no controlling-root materialization attempt exists.
   A `NEEDS_REPLAN` cold resume with a valid technical return cursor immediately
   invokes Plan repair and Recovery Vet before selecting a forward phase or
   emitting any reply.
   That materialization runs directly as non-consumptive prerequisite recovery,
   even with no pending product slice or AFK budget; it does not re-ask an answered
   human gate.
   reconstruct any missing return cursor only from the current phase plus the
   exact approved action in `test-plan.md` / evidence, never from chat or guesswork.
   The idea + flags: `--ship` / `--yolo` (continue through ship preflight, then stop
   for literal-GO and native approval), `--max-slices N` (optional lower safety cap for a partial run; default =
   the plan's slice count, i.e. run all planned slices). Parse only the current
   `$ARGUMENTS`; normalize the result to the idea, `ship_preflight: yes|no`,
   `max_slices: N|plan`, `profile: standard|full`, and `cross_model: yes|no`.
   **Completion:** that normalized state is unambiguous and no sentinel or workspace
   file has been written.
2. **Specify and clarify up front.** Use `devrites-interview`, `$rite-spec`, and
   `$rite-clarify` as one interactive window. Clear specs ask zero questions; Partial/Missing
   coverage never arms AFK. **Completion:** `decision-coverage.md` records `CLEAR`.
3. **Arm AFK after clarity.** Require `Decision coverage: CLEAR`, then apply the
   loop's [one-write AFK contract](reference/loop.md#arm-afk-once): preserve a valid
   existing sentinel byte-for-byte or create an advisory-only sentinel once; never
   rewrite it after Vet. Also `touch .devrites/CHECKPOINT`: unattended runs use
   checkpoint mode so each proven slice is committed locally as a crash-survivable
   `WIP` ([rite-build/reference/checkpoint.md](../rite-build/reference/checkpoint.md));
   `$rite-ship` collapses them into the one feature commit. **Completion:** AFK config is
   valid and advisory-only, any pre-existing bytes are unchanged, and the checkpoint
   sentinel exists.
4. **Drive the phases** ([reference/loop.md](reference/loop.md)). The canonical arc is
   `$rite-spec` → **`$rite-clarify`** → **`$rite-temper`** → `$rite-define` →
   **`$rite-vet`** → `$rite-build` (repeat until all slices
   built; root charges the AFK budget once per green built slice) → `$rite-prove`
   → `$rite-polish` → `$rite-review` → `$rite-seal`.
   Read each `SKILL.md` and execute it; workspace files carry state, not chat.
   Follow the loop's backward-edge contract whenever a later phase discovers an
   agent-owned earlier-phase gap; keep the original return cursor until that
   phase resumes.
   Immediately after `$rite-vet` and before the first Build dispatch, apply the loop's
   [mutable post-vet budget](reference/loop.md#derive-the-mutable-post-vet-budget)
   contract. **Completion:** the loop reaches Seal GO, or a stop condition is persisted
   before any later phase runs; the cursor names the last completed phase and the
   effective remaining budget.
5. **Apply stop conditions at every gate** ([reference/stop-conditions.md](reference/stop-conditions.md)):
   first route an agent-owned red technical result through the loop's bounded
   recovery; its `blocked` label alone is not a stop condition. After that, on
   hard-risk / human-owned blocking / escalating / NO-GO / budget-exhausted / still-low-confidence
   → write `state.md` (`Status`, `Next step`), surface *why*, and **STOP**.
   Exhausted agent-owned technical recovery uses the terminal `Next step: none`
   marker and never hands the user `$rite-plan unblock` or another routine
   phase command.
   **Completion:** either no stop condition is active, or the stopped cursor and reason
   are durable and no later phase has been invoked.
6. **Seal GO → ship boundary.** With `--ship` / `--yolo`, proceed through
   `$rite-ship` preflight, disclose the exact Git plan, then stop for the human's
   literal `GO` and any native host approval. Without the flag, stop at seal GO with
   `$rite-ship` as the resume command. **Completion:** the no-flag path has performed no
   Ship work; the flagged path has disclosed the exact plan and performed no Git action
   before fresh literal `GO` plus native approval.

> **Mid-flight discipline.** Stop rather than auto-passing a blocking gate, answering a
> human-owned material question, or continuing past red tests.

Keep the log terse:
```
Autocomplete: <slug>
spec <done|stopped> · clarify <clear|stopped> · temper <done|skipped|stopped> · define <done|stopped> · vet <ready|stopped> · build <n/N|stopped> · prove <done|stopped> · polish <done|stopped> · review <done|stopped> · seal <GO|NO-GO|stopped>
```

Final state examples: `Shipped: <feature>`, `Stopped: <reason>`, `Awaiting human:
<qid> · <gate> · <slice/phase>`, `NO-GO: <verdict>`, or `GO: feature cleared to ship`.
Do not write a narrative recap.

## Clean baseline and checkpoint mode
- Before autonomy, require a clean or accepted baseline; refuse unrelated dirty work and record plan artifacts.
- Arm `.devrites/CHECKPOINT`; `$rite-ship` collapses its local-only proven-slice checkpoints.
- Red gates block forward advancement and enter caller-owned bounded recovery.
  Stop only when the shared stop contract classifies risk/HITL or the exact
  fingerprint's recovery budget is exhausted.
- One autocomplete pass spans the lifecycle. Each wright returns after one slice;
  explicit `.devrites/AFK` lets the Build root chain green slices under cap/pause rules;
  otherwise HITL stops.
