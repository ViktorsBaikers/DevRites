# `--cross-model` — the outside voice (a genuinely different model)

`devrites-plan-reviewer` gives an *independent* read (fresh context, no authoring reasoning) —
but it's still the same model family. `--cross-model` adds a second opinion from a **different**
AI system, because two models agreeing on a plan is stronger signal than one model reviewing it
thoroughly. It is **opt-in** for `/rite-vet` (the flag) and **off by default in
`/rite-autocomplete`** unless the flag was armed — it costs latency and a second runtime.

## When it runs
Only when `$ARGUMENTS` contains `--cross-model` **and** a different-model reviewer is available.
The default path (no flag) relies on `devrites-plan-reviewer` for independence — that is a
complete review on its own; cross-model is an enhancement, not a requirement.

## Dispatch
After the in-model reviewer loop converges (or in parallel with its final iteration), hand the
**same fresh inputs** to a different model — preferentially the `codex:codex-rescue` agent (it
runs Codex through the shared runtime). Give it only the contract, never the authoring reasoning:

> Independent engineering review of a defined implementation plan, before any code.
> Read only `.devrites/work/<slug>/plan.md`, `tasks.md`, and `spec.md`. Judge: architecture &
> boundaries, scope discipline & reuse, test-coverage design, performance, reversibility, and
> failure-mode coverage. For every finding, quote the line that motivates it and give a 1-10
> confidence; suppress anything you can't quote. Return labeled findings (Critical / Important /
> Suggestion) with a concrete fix. Do not edit anything. Do not approve — find what will cost a redo.

If the agent / runtime is unavailable, **skip and say so** in `eng-review.md` ("cross-model:
requested but unavailable — in-model review only"); do not block on it and do not silently treat
the absence as a pass.

## Integration rule (the important one)
Cross-model findings are **informational until the human approves each one** — even when you
agree with them, even when they overlap the in-model reviewer. Cross-model **consensus** is a
strong signal: surface it as such ("both reviewers flagged Slice 3's missing timeout test"), but
the human still makes the call via `AskUserQuestion`. In AFK, the same gate ceiling applies:
a cross-model finding that *hardens* the plan auto-applies and is recorded; one that *grows
scope* or changes acceptance is a blocking pause.

## Recording
In `eng-review.md` §6, note: which model ran, the overlap with the in-model reviewer (consensus
findings), and any **unique** findings the outside voice caught that the in-model pass missed —
those are the highest-value output of running it at all.
