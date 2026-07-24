---
name: rite-autocomplete
description: Run the DevRites lifecycle end-to-end unattended; --ship confirms the final gate. Use when the user says "autocomplete", "one-shot this feature", or "ship it autonomously". Not for a single phase.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N]"
user-invocable: true
---

# /rite-autocomplete: full lifecycle, unattended

Runs every DevRites phase in order without pausing for discretionary input. It asks
clarifying questions **before** unattended work begins. Safety gates remain active:
hard irreversible-risk,<!-- pack-scan-ignore: negated statement: gates are NOT disabled -->
blocking / escalating gates, and any NO-GO still pause.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.claude/skills/devrites-lib/reference/standards/core.md` and `.claude/skills/devrites-lib/reference/standards/afk-hitl.md` first.

## Operating rules
- **Use one initial human window.** Run spec and topology-first clarify; arm AFK only after
  `Decision coverage: CLEAR`. Later discretionary calls are recorded, not asked
  ([decision policy](reference/decision-policy.md)).
- **Safety gates are not bypassable.** AFK never auto-passes destructive migration /
  auth-authz change / public-API break / external-contract change; blocking
  and escalating gates and any open `gate: validating` always pause. `--ship` auto-confirms
  the **final** type-GO only: nothing else. A change that violates a declared project
  principle (`.devrites/principles.md`) with no recorded exception pauses too: autocomplete
  never grants a principle exception on its own (`principles.md`: that's a human decision).
  Red checks never advance the loop: autocomplete runs the shared bounded debug recovery, then
  stops as `blocked` if the objective failure remains; it asks only if recovery exposes a
  human-owned decision.
- **Set the loop budget from the plan's slice count.** After `/rite-vet`
  (not `/rite-define`: vet may split or add slices, so the count isn't final until then),
  set the AFK budget to however many slices the plan has, so the loop builds exactly the
  planned slices and stops when they are done. `--max-slices N` is an optional lower
  safety cap for a partial run; omit it to run the whole plan. The planned slice count
  keeps the default run finite.
- **Record each discretionary choice.** Pick the option recommended by the relevant
  specialist or reviewer and record the rationale. Do not choose arbitrarily.
- **Strategic review runs, but never auto-grows scope.** After `/rite-clarify`, run `/rite-temper`
  (significance-gated; it skips low-stakes specs in one line). Unattended it auto-applies only
  `hold-rigor` + `reduce-to-MVP` (these never grow acceptance); **any `expand` is a blocking
  pause**, and irreversible-risk findings always pause. Autocomplete hardens and may *prune* the
  spec on its own; it never *expands* the build's scope without the human.
- **Engineering review runs on every plan, but never auto-grows scope.** After `/rite-define`,
  run `/rite-vet` on **every** feature (depth scales: a light pass on simple plans, full rigor on
  big/risky; never skipped). Unattended it auto-applies only *hardening* findings: added test
  requirements, error-handling / failure-mode coverage, tightened scope, reuse-over-rebuild,
  ordering / parallel-lane fixes (these never grow acceptance); **any finding that grows scope,
  adds a slice, or changes acceptance is a blocking pause**, and irreversible-risk findings always
  pause. Cross-model is off unless `--cross-model` was armed.

## Workflow
1. **Orient + parse args.** Run `devrites-engine preamble` for deterministic workspace orientation.
   The idea + flags: `--ship` / `--yolo` (auto-confirm the final
   type-GO), `--max-slices N` (optional lower safety cap for a partial run; default =
   the plan's slice count, i.e. run all planned slices).
2. **Specify and clarify up front.** Use `devrites-interview`, `/rite-spec`, and
   `/rite-clarify` as one interactive window. Clear specs ask zero questions; Partial/Missing
   coverage never arms AFK. **Completion:** `decision-coverage.md` records `CLEAR`.
3. **Arm AFK after clarity.** Require `Decision coverage: CLEAR`, then write `.devrites/AFK`
   with `allow_gates: [advisory]`; set the slice budget
   from the plan's count after `/rite-vet` (the slice count is only final post-vet), or from
   an explicit `--max-slices` ([reference/loop.md](reference/loop.md)). validating / blocking / escalating +
   irreversible-risk still pause. Also `touch .devrites/CHECKPOINT`: unattended runs use
   checkpoint mode so each proven slice is committed locally as a crash-survivable `WIP`
   ([rite-build/reference/checkpoint.md](../rite-build/reference/checkpoint.md));
   `/rite-ship` collapses them into the one feature commit.
4. **Drive the phases** ([reference/loop.md](reference/loop.md)). The canonical arc is
   `/rite-spec` → **`/rite-clarify`** → **`/rite-temper`** → `/rite-define` →
   **`/rite-vet`** → `/rite-build` (repeat until all slices
   built; `devrites-engine tick-afk` each) → `/rite-prove` → `/rite-polish` → `/rite-review` → `/rite-seal`.
   Run each by Reading its `SKILL.md` and executing its workflow; state is carried by the
   workspace files, not chat.
5. **Apply stop conditions at every gate** ([reference/stop-conditions.md](reference/stop-conditions.md)):
   on hard-risk / blocking / escalating / NO-GO / budget-exhausted / still-low-confidence
   → write `state.md` (`Status`, `Next step`), surface *why*, and **STOP**.
6. **Seal GO → ship.** With `--ship`, proceed to `/rite-ship` and auto-confirm the
   type-GO. Without it, render the type-GO prompt and stop for the human.

> **Mid-flight discipline.** Stop rather than auto-passing a blocking gate, answering a
> human-owned material question, or continuing past red tests.

## Output
A compact phase log followed by the final status. **Progress first for the final
status**: run `devrites-engine progress`, then use the shared typed states from
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md):
`Shipped`, `Stopped`, `Awaiting human`, `NO-GO`, or `GO`.

Keep the log terse:
```
Autocomplete: <slug>
spec <done|stopped> · clarify <clear|stopped> · temper <done|skipped|stopped> · define <done|stopped> · vet <ready|stopped> · build <n/N|stopped> · prove <done|stopped> · polish <done|stopped> · review <done|stopped> · seal <GO|NO-GO|stopped>
```

Final state examples: `Shipped: <feature>`, `Stopped: <reason>`, `Awaiting human:
<qid> · <gate> · <slice/phase>`, `NO-GO: <verdict>`, or `GO: feature cleared to ship`.
Do not write a narrative recap.

## Clean baseline and checkpoint mode
- Before an autonomous run, require a clean or explicitly accepted baseline: refuse unrelated dirty work and record expected planning artifacts.
- Arm `.devrites/CHECKPOINT` for the run so each proven slice can be checkpointed local-only; `/rite-ship` owns collapsing those checkpoints.
- Stop on risky steps, red gates, NO-GO, stale evidence, or budget exhaustion.
- Autocomplete gets one approved pass through the lifecycle; `/rite-build` still builds exactly one slice per invocation.
