---
name: rite-autocomplete
description: Run the entire DevRites lifecycle end-to-end with no per-phase human iteration — spec → temper → define → build×N → prove → polish → review → seal → ship — choosing the best option at every soft gate and recording the rationale. A vague prompt triggers an up-front clarifying interview; after that it runs unattended, pausing only on hard irreversible-risk gates (auth / migration / public-API / red tests), blocking or escalating gates, or a NO-GO. Default stops at the final type-GO; `--ship` (alias `--yolo`) auto-confirms it for a zero-touch push. Use when the user says "autocomplete", "do the whole thing", "run the full cycle", "build this end to end", "one-shot this feature", or "ship it autonomously". Not for a single phase (use the specific /rite-* skill) or when the user wants to drive each step.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N]"
user-invocable: true
---

# /rite-autocomplete — full lifecycle, unattended

Drives every DevRites phase in order without stopping for discretionary input. The
prompt may be vague — autocomplete asks its clarifying questions **up front**, then
runs to completion. It does **not** disable the safety gates: hard irreversible-risk,
blocking / escalating gates, and any NO-GO still pause.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` and `.claude/rules/afk-hitl.md` first.

## Operating rules
- **One human window.** Clarifying questions are batched up front via
  `devrites-interview`. After that, discretionary decisions are made automatically and
  recorded in `decisions.md` — not asked. See [reference/decision-policy.md](reference/decision-policy.md).
- **Safety gates are not bypassable.** AFK never auto-passes destructive migration /
  auth-authz change / public-API break / external-contract change / red tests; blocking
  and escalating gates and any open `gate: validating` always pause. `--ship` auto-confirms
  the **final** type-GO only — nothing else.
- **Loop budget = the plan's own slice count, not a fixed number.** After `/rite-define`,
  set the AFK budget to however many slices the plan has, so the loop builds exactly the
  task's slices and stops when they're done. `--max-slices N` is an OPTIONAL *lower* safety
  cap (partial / babysat run); omit it to run the whole plan. The budget is finite
  (= planned slices), so a runaway is still bounded.
- **Best option, recorded.** For each discretionary choice, pick the option the relevant
  specialist / reviewer favours and record the rationale. Never silently coin-flip.
- **Strategic review runs, but never auto-grows scope.** After `/rite-spec`, run `/rite-temper`
  (significance-gated — it skips low-stakes specs in one line). Unattended it auto-applies only
  `hold-rigor` + `reduce-to-MVP` (these never grow acceptance); **any `expand` is a blocking
  pause**, and irreversible-risk findings always pause. Autocomplete hardens and may *prune* the
  spec on its own; it never *expands* the build's scope without the human.

## Workflow
1. **Parse args.** The idea + flags: `--ship` / `--yolo` (auto-confirm the final
   type-GO), `--max-slices N` (OPTIONAL *lower* safety cap for a partial run; default =
   the plan's slice count, i.e. run all planned slices).
2. **Clarify up front.** If the idea is underspecified, run `devrites-interview` to
   ~95% confidence — the only interactive window. If already clear, skip.
3. **Arm AFK.** Write `.devrites/AFK` with `allow_gates: [advisory]`; set the slice budget
   from the plan's count after `/rite-define` (or from an explicit `--max-slices`)
   ([reference/loop.md](reference/loop.md)). validating / blocking / escalating +
   irreversible-risk still pause.
4. **Drive the phases** ([reference/loop.md](reference/loop.md)): `/rite-spec` →
   **`/rite-temper`** → `/rite-define` → `/rite-build` (loop until all slices built;
   `tick-afk.sh` each) → `/rite-prove` → `/rite-polish` → `/rite-review` → `/rite-seal`. Run
   each by Reading its `SKILL.md` and executing its workflow; state is carried by the workspace
   files, not chat.
5. **Apply stop conditions at every gate** ([reference/stop-conditions.md](reference/stop-conditions.md)):
   on hard-risk / blocking / escalating / NO-GO / budget-exhausted / still-low-confidence
   → write `state.md` (`Status`, `Next step`), surface *why*, and **STOP**.
6. **Seal GO → ship.** With `--ship`, proceed to `/rite-ship` and auto-confirm the
   type-GO. Without it, render the type-GO prompt and stop for the human.

> **Mid-flight discipline.** When tempted to auto-pass a blocking gate "to keep moving",
> answer a material question yourself instead of pausing, or run past red tests — stop.
> Autonomy is for the routine path; the gates exist for everything else.

## Output
A phase-by-phase progress log, then the final status:
```
Autocomplete: <slug>
spec ✓  temper ✓ (mode <m> | skipped)  define ✓  build ✓ (N slices)  prove ✓  polish ✓  review ✓  seal: GO
→ SHIPPED  <sha> on <branch>            (--ship)
   — or —
→ STOPPED — awaiting human at <phase>: <reason>
   Resume: <single command>
```

End with `↻ Hygiene` per `.claude/rules/context-hygiene.md` (`/clear` between long phases).
