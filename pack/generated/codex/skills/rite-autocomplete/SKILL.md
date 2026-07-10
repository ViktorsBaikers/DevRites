---
name: rite-autocomplete
description: Run the DevRites lifecycle end-to-end unattended; --ship confirms the final gate. Use when the user says "autocomplete", "one-shot this feature", or "ship it autonomously". Not for a single phase.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers — NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-autocomplete — full lifecycle, unattended

Drives every DevRites phase in order without stopping for discretionary input. The
prompt may be vague — autocomplete asks its clarifying questions **up front**, then
runs to completion. It does **not** disable the safety gates: hard irreversible-risk,<!-- pack-scan-ignore: negated statement — gates are NOT disabled -->
blocking / escalating gates, and any NO-GO still pause.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` and `.agents/skills/devrites-lib/reference/standards/afk-hitl.md` first.

## Operating rules
- **One human window.** Clarifying questions are batched up front via
  `devrites-interview`. After that, discretionary decisions are made automatically and
  recorded in `decisions.md` — not asked. See [reference/decision-policy.md](reference/decision-policy.md).
- **Safety gates are not bypassable.** AFK never auto-passes destructive migration /
  auth-authz change / public-API break / external-contract change / red tests; blocking
  and escalating gates and any open `gate: validating` always pause. `--ship` auto-confirms
  the **final** type-GO only — nothing else. A change that violates a declared project
  principle (`.devrites/principles.md`) with no recorded exception pauses too — autocomplete
  never grants a principle exception on its own (`principles.md`: that's a human decision).
- **Loop budget = the plan's own slice count, not a fixed number.** After `$rite-vet`
  (not `$rite-define` — vet may split or add slices, so the count isn't final until then),
  set the AFK budget to however many slices the plan has, so the loop builds exactly the
  task's slices and stops when they're done. `--max-slices N` is an OPTIONAL *lower* safety
  cap (partial / babysat run); omit it to run the whole plan. The budget is finite
  (= planned slices), so a runaway is still bounded.
- **Best option, recorded.** For each discretionary choice, pick the option the relevant
  specialist / reviewer favours and record the rationale. Never silently coin-flip.
- **Strategic review runs, but never auto-grows scope.** After `$rite-spec`, run `$rite-temper`
  (significance-gated — it skips low-stakes specs in one line). Unattended it auto-applies only
  `hold-rigor` + `reduce-to-MVP` (these never grow acceptance); **any `expand` is a blocking
  pause**, and irreversible-risk findings always pause. Autocomplete hardens and may *prune* the
  spec on its own; it never *expands* the build's scope without the human.
- **Engineering review runs on every plan, but never auto-grows scope.** After `$rite-define`,
  run `$rite-vet` on **every** feature (depth scales — a light pass on simple plans, full rigor on
  big/risky; never skipped). Unattended it auto-applies only *hardening* findings — added test
  requirements, error-handling / failure-mode coverage, tightened scope, reuse-over-rebuild,
  ordering / parallel-lane fixes (these never grow acceptance); **any finding that grows scope,
  adds a slice, or changes acceptance is a blocking pause**, and irreversible-risk findings always
  pause. Cross-model is off unless `--cross-model` was armed.

## Workflow
1. **Orient + parse args.** Run `devrites-engine preamble` for deterministic workspace orientation.
   The idea + flags: `--ship` / `--yolo` (auto-confirm the final
   type-GO), `--max-slices N` (OPTIONAL *lower* safety cap for a partial run; default =
   the plan's slice count, i.e. run all planned slices).
2. **Clarify up front.** If the idea is underspecified, run `devrites-interview` to
   ~95% confidence — the only interactive window. If already clear, skip.
3. **Arm AFK.** Write `.devrites/AFK` with `allow_gates: [advisory]`; set the slice budget
   from the plan's count after `$rite-vet` (the slice count is only final post-vet), or from
   an explicit `--max-slices` ([reference/loop.md](reference/loop.md)). validating / blocking / escalating +
   irreversible-risk still pause. Also `touch .devrites/CHECKPOINT` — an unattended run is the
   case checkpoint mode earns its keep, so each proven slice is committed local-only as
   crash-survivable `WIP` ([rite-build/reference/checkpoint.md](../rite-build/reference/checkpoint.md));
   `$rite-ship` collapses them into the one feature commit.
4. **Drive the phases** ([reference/loop.md](reference/loop.md)): `$rite-spec` →
   **`$rite-temper`** → `$rite-define` → **`$rite-vet`** → `$rite-build` (loop until all slices
   built; `devrites-engine tick-afk` each) → `$rite-prove` → `$rite-polish` → `$rite-review` → `$rite-seal`.
   Run each by Reading its `SKILL.md` and executing its workflow; state is carried by the
   workspace files, not chat.
5. **Apply stop conditions at every gate** ([reference/stop-conditions.md](reference/stop-conditions.md)):
   on hard-risk / blocking / escalating / NO-GO / budget-exhausted / still-low-confidence
   → write `state.md` (`Status`, `Next step`), surface *why*, and **STOP**.
6. **Seal GO → ship.** With `--ship`, proceed to `$rite-ship` and auto-confirm the
   type-GO. Without it, render the type-GO prompt and stop for the human.

> **Mid-flight discipline.** When tempted to auto-pass a blocking gate "to keep moving",
> answer a material question yourself instead of pausing, or run past red tests — stop.
> Autonomy is for the routine path; the gates exist for everything else.

## Output
A compact phase-by-phase log, then the final status. **Progress first for the final
status** — run `devrites-engine progress`, then use the shared typed states from
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md):
`Shipped`, `Stopped`, `Awaiting human`, `NO-GO`, or `GO`.

Keep the log terse:
```
Autocomplete: <slug>
spec <done|stopped> · temper <done|skipped|stopped> · define <done|stopped> · vet <done|stopped> · build <n/N|stopped> · prove <done|stopped> · polish <done|stopped> · review <done|stopped> · seal <GO|NO-GO|stopped>
```

Final state examples: `Shipped: <feature>`, `Stopped: <reason>`, `Awaiting human:
<qid> · <gate> · <slice/phase>`, `NO-GO: <verdict>`, or `GO: feature cleared to ship`.
Do not write a narrative recap.

## Clean baseline and checkpoint mode
- Before an autonomous run, require a clean or explicitly accepted baseline: refuse unrelated dirty work and record expected planning artifacts.
- Arm `.devrites/CHECKPOINT` for the run so each proven slice can be checkpointed local-only; `$rite-ship` owns collapsing those checkpoints.
- Stop on risky steps, red gates, NO-GO, stale evidence, or budget exhaustion.
- Autocomplete gets one approved pass through the lifecycle; `$rite-build` still builds exactly one slice per invocation.
