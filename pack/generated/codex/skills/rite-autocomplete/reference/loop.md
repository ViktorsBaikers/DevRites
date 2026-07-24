# The autocomplete loop: arm AFK, drive every phase

Autocomplete runs the existing `/rite-*` skills in order without stopping between
routine phases. Each phase keeps its own workflow; autocomplete only sequences phases
and enforces stop conditions.

## Arm AFK

Arm the gate policy up front; set the slice budget once the plan's count is known:

```yaml
allow_gates: [advisory]        # only advisory auto-handles; validating+ pause
# notify: "<cmd>"              # optional — fired on any awaiting_human pause
# max_slices: <N>              # the slice BUDGET — set from the plan's count after
# $rite-define (or an explicit --max-slices). NOT a target decomposition; it only
# caps how many run unattended (default = all the plan's slices).
```

The budget is the plan's slice count, so the loop builds the planned slices and stops
when they are done. `--max-slices N` only lowers that number for a
partial run. Arm the gate policy at step 3; write `max_slices` / `AFK slices remaining`
**after `$rite-vet`** (it runs before the build loop and may split or add a slice, so the
count isn't final until then: vet runs on every plan here, so always set the budget after it).

`allow_gates: [advisory]` prevents an open `gate: validating` from being queued until
seal, where it would force NO-GO under `afk-hitl.md`. Autocomplete pauses on it instead. Widen
`allow_gates` only when the caller explicitly asks.

## Drive the phases

Read each phase's `SKILL.md` and execute that workflow. Workspace files such as
`state.md`, `tasks.md`, and `evidence.md` carry state between phases; chat does not.

| Step | Phase | Loop / gate |
|---|---|---|
| 1 | `$rite-spec` | interactive window: investigate, feed intent answers, write `spec.md` |
| 2 | `$rite-clarify` | same interactive window: topology-first scan; write `decision-coverage.md`; proceed only on `CLEAR`, then arm AFK |
| 3 | `$rite-temper` | significance-gated strategic review; harden spec + write `strategy.md`. Skip low-stakes specs in one line. AFK: `hold-rigor` / `reduce-to-MVP` auto-apply; **any `expand` pauses (blocking)**; irreversible-risk pauses |
| 4 | `$rite-define` | reads `decision-coverage.md` + `strategy.md`; writes `plan.md` + `tasks.md`; records `Plan approved` |
| 5 | `$rite-vet` | engineering/readiness review on **every** plan (light pass on simple plans, full on big/risky; never skipped); harden `plan.md` / `tasks.md` + write `eng-review.md` (`Implementation readiness: READY`) + `test-plan.md`. AFK: hardening / coverage findings auto-apply; **any scope-growing / acceptance-changing finding pauses (blocking)**; irreversible-risk pauses. Set the slice budget after this (vet may split a slice) |
| 6 | `$rite-build` ×N | **loop** while any slice is `pending`; build one (the slice-wright reads `test-plan.md` for coverage), then run `devrites-engine tick-afk state.md`: exit 3 (budget hit) ⇒ STOP |
| 7 | `$rite-prove` | once all slices `built`; walks `test-plan.md`; on failure → `devrites-debug-recovery` within scope |
| 8 | `$rite-polish` | re-verify after code edits (evidence must stay fresh) |
| 9 | `$rite-review` | apply in-scope fixes; re-prove if code changed |
| 10 | `$rite-seal` | GO/NO-GO decision (no git here) |
| 11 | `$rite-ship` | only if seal GO; `--ship` auto-confirms type-GO, else stop for human |

## Between phases

- Re-read the active workspace before each phase (don't trust chat memory).
- After a phase that edits code (build, polish, review), evidence may be stale: let
  the next gate re-prove rather than carrying a stale pass (see
  `.agents/skills/devrites-lib/reference/standards/development-workflow.md`).
- Check [stop-conditions.md](stop-conditions.md) at every gate **before** advancing.
