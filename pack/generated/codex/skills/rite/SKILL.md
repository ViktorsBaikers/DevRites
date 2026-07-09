---
name: rite
description: User-invoked DevRites menu and router; no args renders the menu, a verb dispatches to the matching `rite-<verb>` skill.
argument-hint: "[verb [args...]]"
user-invocable: true
disable-model-invocation: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite — DevRites menu + router

You are the DevRites entry point. Two modes:

- **No args** → render the menu (below), then stop. Do not execute a workflow phase. Do not read `state.md` / run evidence checks / list artifacts — that's `$rite-status`.
- **Verb arg** → dispatch to the matching `rite-<verb>` skill (see "Dispatch" below). The router is a pass-through: `$rite spec foo` ≡ `$rite-spec foo`; the called skill owns the output.

## Dispatch

If `$ARGUMENTS` starts with a verb in this table, **load the matching skill and execute its workflow** with the remainder of `$ARGUMENTS` as that skill's argument. Try post-install path first, fall back to pre-install:

```bash
V=<verb>; ARGS="<remaining args>"
F=.agents/skills/rite-$V/SKILL.md
[ -f "$F" ] || F=.agents/skills/rite-$V/SKILL.md
# Then Read "$F" and follow its workflow with $ARGS as that skill's $ARGUMENTS.
```

| Verb | Equivalent shortcut | Skill |
|---|---|---|
| `spec [feature]` | `$rite-spec` | start a feature — investigate + write spec.md |
| `adopt [area]` | `$rite-adopt` | onboard an existing codebase — reverse-derive spec.md + seed conventions |
| `temper [--mode]` | `$rite-temper` | optional strategic spec review (scope mode + pre-mortem) before define |
| `define` | `$rite-define` | turn the spec into plan + task slices |
| `vet [--cross-model]` | `$rite-vet` | optional engineering plan review (scope · architecture · tests · perf) before build |
| `plan [mode]` | `$rite-plan` | reshape / reslice / repair an active plan |
| `build [slice]` | `$rite-build` | implement exactly one vertical slice, then stop |
| `converge [slug]` | `$rite-converge` | recovery — assess live code vs intent, append remaining work as new slices |
| `prove` | `$rite-prove` | full tests + browser proof |
| `polish [mode]` | `$rite-polish` | code + UI polish |
| `review [scope]` | `$rite-review` | multi-axis feature review |
| `seal` | `$rite-seal` | final GO / NO-GO decision |
| `ship` | `$rite-ship` | type-GO + commit/push/tag, then archive the task |
| `status [slug]` | `$rite-status` | active feature, next action, evidence |
| `doctor` | `$rite-doctor` | health check — install integrity, stale ACTIVE, orphaned gates, hook wiring, merge/rebase state |
| `learn [--mine \| "<lesson>"]` | `$rite-learn` | review the captured learning ledger → promote recurring lessons to project rules / principles |
| `use <slug>` | (inline) | switch the active feature — re-point `.devrites/ACTIVE` |
| `resolve <qid> "<answer>"` | `$rite-resolve` | answer a HITL gate |
| `prototype [question]` | `$rite-prototype` | throwaway prototype |
| `handoff [focus]` | `$rite-handoff` | compact chat → handoff doc |
| `zoom-out` | `$rite-zoom-out` | structural map of unfamiliar code |
| `pressure-test` | `$rite-pressure-test` | diverge → converge on a rough idea |
| `autocomplete [idea] [--ship]` | `$rite-autocomplete` | run the whole lifecycle unattended |
| `quick [change]` | `$rite-quick` | express lane — one small reversible change, build → prove → ship |
| `frame [task]` | `$rite-frame` | pre-flight goal-reframe + four-failure-mode self-audit for ad-hoc / express work |

The `/rite-<verb>` standalones remain user-invocable as direct shortcuts; both forms hit the same skill. Use whichever reads more naturally — the menu form (`$rite spec`) for discovery, the shortcut (`$rite-spec`) for muscle memory.

`use <slug>` is handled **inline** — there is no `rite-use` skill. Confirm
`.devrites/work/<slug>/` exists, then re-point `.devrites/ACTIVE` to `<slug>` and report
the now-active feature. It is cheap context-switching only — no re-spec, no phase run. If
the workspace is missing, list the slugs under `.devrites/work/` and stop.

Specialist triggers (model-invoked inside the above):
`devrites-frontend-craft` (UI) · `devrites-browser-proof` (UI verify) ·
`devrites-source-driven` (uncertain library) · `devrites-doubt` (non-trivial
decision) · `devrites-api-interface` (cross-boundary) ·
`devrites-audit <security|perf|simplify>` (single-axis review/polish pass) ·
`devrites-debug-recovery` (failures). Parallel reviewer fan-out at seal is a shared
reference (see [`devrites-lib/reference/parallel-dispatch.md`](../devrites-lib/reference/parallel-dispatch.md)).

## What to output

Reply-contract exception: `$rite` is the menu/router, not a workspace completion step.
Called phase skills own the shared completion reply contract
([`reply-contract.md`](../devrites-lib/reference/reply-contract.md)).

1. **Verb in `$ARGUMENTS`** → dispatch per the table above. The called skill owns the response.
2. **No args** → render the menu below, then stop.
3. **Unrecognized first token** → tell the user the known verbs and stop. Don't guess.
4. **No active feature** and the user asked "where am I" or named no verb → point at `$rite spec <feature>` (or `$rite-spec`). Don't summarize state yourself — `$rite status` (or `$rite-status`) owns that.

## Gotchas
- No args → render the menu and stop. Don't execute a phase, read `state.md`, or summarize status — that's `$rite-status`.
- Unrecognized first token → list the known verbs and stop; never guess which phase the user meant.
- Pure pass-through: dispatch to the `rite-<verb>` skill and let it own the output; don't do the phase's work in the router.

## Menu

```
DevRites — disciplined senior-engineer workflow
                              menu form           direct shortcut
SPEC          $rite spec               ≡    $rite-spec        investigate deeply → write spec.md
ADOPT         $rite adopt              ≡    $rite-adopt       onboard existing code → reverse-derive spec.md + seed conventions
TEMPER        $rite temper             ≡    $rite-temper      optional — strategic review: scope mode + pre-mortem, harden the spec
PLAN          $rite define             ≡    $rite-define      turn the spec into plan + task slices + state
VET           $rite vet                ≡    $rite-vet         optional — engineering plan review: scope · architecture · tests · perf, harden the plan
REPLAN        $rite plan               ≡    $rite-plan        decompose / reslice / repair an active plan
BUILD         $rite build              ≡    $rite-build       implement exactly one verified vertical slice, then stop
PROVE         $rite prove              ≡    $rite-prove       tests + build + runtime + browser evidence
POLISH        $rite polish             ≡    $rite-polish      code polish always; UI normalize + polish if UI
REVIEW        $rite review             ≡    $rite-review      feature-scoped multi-axis review
SEAL          $rite seal               ≡    $rite-seal        final GO / NO-GO decision (no git)
SHIP          $rite ship               ≡    $rite-ship        type-GO + commit/push/tag, then archive + clear ACTIVE
STATUS        $rite status             ≡    $rite-status      active feature, next action, evidence, risks
DOCTOR        $rite doctor             ≡    $rite-doctor      health check — install · stale ACTIVE · orphaned gates · hook wiring · merge/rebase
LEARN         $rite learn ...          ≡    $rite-learn       review captured lessons → promote to project rules / principles
SWITCH        $rite use <slug>                                re-point .devrites/ACTIVE to another feature (inline)
RESUME        $rite resolve ...        ≡    $rite-resolve     answer a HITL checkpoint
AUTO          $rite autocomplete ...   ≡    $rite-autocomplete  run the whole lifecycle unattended (--ship to push)
QUICK         $rite quick <change>     ≡    $rite-quick       express lane — one small reversible change (escalates if it grows)
UTILITY       $rite frame | prototype | handoff | zoom-out | pressure-test  (or direct /rite-* shortcuts)
```

> **Small one-off change?** A typo, copy tweak, config bump, or one-function fix → **`$rite-quick`**
> (express lane: one contract → build → prove → ship, no full workspace). It escalates to
> `$rite-spec` the instant the change grows past small / reversible / unambiguous. The full
> lifecycle above is for real features — don't pay its ceremony for a one-off.

## Core operating rules (every DevRites skill enforces)

The operating rules live in `.agents/skills/devrites-lib/reference/standards/core.md`; DevRites skills Read it as
their first step, and the other rule files load on demand. See
[`.agents/skills/devrites-lib/reference/standards/README.md`](../devrites-lib/reference/standards/README.md) for the full index.
