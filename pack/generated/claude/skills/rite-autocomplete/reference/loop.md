# The autocomplete loop: arm AFK, drive every phase

Autocomplete sequences existing `/rite-*` workflows and enforces stop
conditions without pausing between routine phases.

## Arm AFK once

Arm the gate policy up front without mutating it later:

```yaml
allow_gates: [advisory]        # only advisory auto-handles; validating+ pause
# notify: "<cmd>"              # optional — fired on any awaiting_human pause
# max_slices: <N>              # include only for an explicit/configured cap
```

Read an existing sentinel first. Preserve it byte-for-byte when valid; stop if its gate
ceiling exceeds `advisory` or its `max_slices` conflicts with the invocation. If absent,
write it once after clarity with `allow_gates: [advisory]` and an explicit
`--max-slices N` only when supplied. The sentinel is read-only: never rewrite it after
`/rite-vet` to inject a discovered plan count.

### Derive the mutable post-vet budget

After `/rite-vet`, count remaining pending slices and derive the run budget as the
minimum of that count, an explicit `--max-slices`, and a configured sentinel cap. Before
the first build dispatch, pre-seed `state.md` `afk_slices_remaining` when absent. If a
valid remaining counter already exists, retain the lower of it and the derived budget;
never increase or reinitialize it. This keeps the crash-survivable budget in its mutable owner
without changing AFK configuration mid-run.

`allow_gates: [advisory]` prevents an open `gate: validating` from being queued until
seal, where it would force NO-GO under `afk-hitl.md`. Autocomplete pauses on it instead. Widen
`allow_gates` only outside autocomplete through an explicit human configuration change;
autocomplete itself never widens the ceiling.

## Drive the phases

Read each phase's `SKILL.md` and execute that workflow. Workspace files such as
`state.md`, `tasks.md`, and `evidence.md` carry state between phases; chat does not.

| Step | Phase | Loop / gate |
|---|---|---|
| 1 | `/rite-spec` | interactive window: investigate, feed intent answers, write `spec.md` |
| 2 | `/rite-clarify` | same interactive window: topology-first scan; write `decision-coverage.md`; proceed only on `CLEAR`, then arm AFK |
| 3 | `/rite-temper` | significance-gated strategic review; harden spec + write `strategy.md`. Skip low-stakes specs in one line. AFK: `hold-rigor` / `reduce-to-MVP` auto-apply; **any `expand` pauses (blocking)**; irreversible-risk pauses |
| 4 | `/rite-define` | reads `decision-coverage.md` + `strategy.md`; writes `plan.md` + `tasks.md`; records `Plan approved` |
| 5 | `/rite-vet` | engineering/readiness review on **every** plan (light pass on simple plans, full on big/risky; never skipped); harden `plan.md` / `tasks.md` + write `eng-review.md` (`Implementation readiness: READY`) + `test-plan.md`. AFK: hardening / coverage findings auto-apply; **any scope-growing / acceptance-changing finding pauses (blocking)**; irreversible-risk pauses. Derive and pre-seed the state-owned slice budget after this (vet may split a slice); never rewrite the AFK sentinel |
| 6 | `/rite-build` ×N | **loop** while any slice is `pending`; build one, then let the root charge exactly one budget unit with the built-state record. Zero ⇒ STOP before another dispatch. |
| 7 | `/rite-prove` | once all slices `built`; walks `test-plan.md`; on failure → `devrites-debug-recovery` within scope |
| 8 | `/rite-polish` | re-verify after code edits (evidence must stay fresh) |
| 9 | `/rite-review` | apply in-scope fixes; re-prove if code changed |
| 10 | `/rite-seal` | GO/NO-GO decision (no git here) |
| 11 | `/rite-ship` | only if seal GO; `--ship` / `--yolo` never authorizes Git and only continues to the exact-plan literal-GO/native-approval boundary |

## Between phases

- Re-read the active workspace before each phase (don't trust chat memory).
- After a phase that edits code (build, polish, review), evidence may be stale: let
  the next gate re-prove rather than carrying a stale pass (see
  `.claude/skills/devrites-lib/reference/standards/development-workflow.md`).
- Check [stop-conditions.md](stop-conditions.md) at every gate **before** advancing.
