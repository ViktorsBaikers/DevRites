# The autocomplete loop — arm AFK, drive every phase

Autocomplete is an orchestrator: it runs the existing `/rite-*` skills in order,
exactly as a human would, but without stopping between them. It owns no workflow of
its own — the phases do the work; autocomplete sequences them and enforces stops.

## Arm AFK

Arm the gate policy up front; set the slice budget once the plan's count is known:

```yaml
allow_gates: [advisory]        # only advisory auto-handles; validating+ pause
# notify: "<cmd>"              # optional — fired on any awaiting_human pause
# max_slices: <N>              # the slice BUDGET — set from the plan's count after
# $rite-define (or an explicit --max-slices). NOT a target decomposition; it only
# caps how many run unattended (default = all the plan's slices).
```

The budget is the plan's own slice count, not a fixed number — so the loop builds exactly
the task's slices and stops when they're done; `--max-slices N` only *lowers* it for a
partial run. Arm the gate policy at step 3; write `max_slices` / `AFK slices remaining`
**after `$rite-vet`** (it runs before the build loop and may split or add a slice, so the
count isn't final until then — vet runs on every plan here, so always set the budget after it).

`allow_gates: [advisory]` is deliberate: an open `gate: validating` is merge-blocking
at seal (`afk-hitl.md`), so autocomplete must *pause* on it rather than queue it and
then hit NO-GO. Widen `allow_gates` only if the caller explicitly asked.

## Drive the phases

Run each by Reading its `SKILL.md` and executing that workflow. State flows through the
workspace files (`state.md`, `tasks.md`, `evidence.md`, …), so each phase picks up where
the last left off — there is nothing to thread through chat.

| Step | Phase | Loop / gate |
|---|---|---|
| 1 | `$rite-spec` | feed the interview answers; write `spec.md` |
| 2 | `$rite-temper` | significance-gated strategic review; harden spec + write `strategy.md`. Skip low-stakes specs in one line. AFK: `hold-rigor` / `reduce-to-MVP` auto-apply; **any `expand` pauses (blocking)**; irreversible-risk pauses |
| 3 | `$rite-define` | reads `strategy.md`; `plan.md` + `tasks.md`; record `Plan approved` |
| 4 | `$rite-vet` | engineering plan review on **every** plan (light pass on simple plans, full on big/risky; never skipped); harden `plan.md` / `tasks.md` + write `eng-review.md` + `test-plan.md`. AFK: hardening / coverage findings auto-apply; **any scope-growing / acceptance-changing finding pauses (blocking)**; irreversible-risk pauses. Set the slice budget after this (vet may split a slice) |
| 5 | `$rite-build` ×N | **loop** while any slice is `pending`; build one (the slice-wright reads `test-plan.md` for coverage), then run `devrites-engine tick-afk state.md` — exit 3 (budget hit) ⇒ STOP |
| 6 | `$rite-prove` | once all slices `built`; walks `test-plan.md`; on failure → `devrites-debug-recovery` within scope |
| 7 | `$rite-polish` | re-verify after code edits (evidence must stay fresh) |
| 8 | `$rite-review` | apply in-scope fixes; re-prove if code changed |
| 9 | `$rite-seal` | GO/NO-GO decision (no git here) |
| 10 | `$rite-ship` | only if seal GO; `--ship` auto-confirms type-GO, else stop for human |

## Between phases

- Re-read the active workspace before each phase (don't trust chat memory).
- After a phase that edits code (build, polish, review), evidence may be stale — let
  the next gate re-prove rather than carrying a stale pass (see
  `.agents/skills/devrites-lib/reference/standards/development-workflow.md`).
- Check [stop-conditions.md](stop-conditions.md) at every gate **before** advancing.
