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
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers — NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite — DevRites menu + router

You are the DevRites entry point. Two modes:

- **No args** → run `devrites-engine first-task`, render one recommended-start line above the menu, then stop. Do not execute a phase or read `state.md` — status is `$rite-status`.
- **Verb arg** → pass-through dispatch to the matching `rite-<verb>` skill (`$rite spec foo` ≡ `$rite-spec foo`); the called skill owns the output.

When the user asks which rite fits, load [`devrites-lib/reference/intent-map.md`](../devrites-lib/reference/intent-map.md).
When they ask how phases connect, load [`reference/menu.md`](reference/menu.md).

## Dispatch

If `$ARGUMENTS` starts with a verb in this table, **load the matching skill and execute its workflow** with the remainder of `$ARGUMENTS` as that skill's argument. Try post-install path first, fall back to pre-install:

```bash
V=<verb>; ARGS="<remaining args>"
F=.agents/skills/rite-$V/SKILL.md
[ -f "$F" ] || F=.agents/skills/rite-$V/SKILL.md
# Then Read "$F" and follow its workflow with $ARGS as that skill's $ARGUMENTS.
```

What each verb does lives once, in the Menu below; this table is the dispatch map only.

| Verb | Skill |
|---|---|
| `spec [feature]` | `$rite-spec` |
| `adopt [area]` | `$rite-adopt` |
| `temper [--mode]` | `$rite-temper` |
| `define` | `$rite-define` |
| `vet [--cross-model]` | `$rite-vet` |
| `plan [mode]` | `$rite-plan` |
| `build [slice]` | `$rite-build` |
| `converge [slug]` | `$rite-converge` |
| `prove` | `$rite-prove` |
| `polish [mode]` | `$rite-polish` |
| `review [scope]` | `$rite-review` |
| `seal` | `$rite-seal` |
| `ship` | `$rite-ship` |
| `status [slug]` | `$rite-status` |
| `doctor [--code | --reindex]` | `$rite-doctor` |
| `learn [--mine \| "<lesson>"]` | `$rite-learn` |
| `explain [concept \| diff \| idea \| recap]` | `$rite-explain` |
| `pov [candidate]` | `$rite-pov` |
| `dogfood [--port N]` | `$rite-dogfood` |
| `pr-feedback [PR\|thread]` | `$rite-pr-feedback` |
| `customize [override <agent> \| extension <name>]` | `$rite-customize` |
| `use <slug>` | (inline) |
| `guide` | (inline) |
| `resolve <qid> "<answer>"` | `$rite-resolve` |
| `prototype [question]` | `$rite-prototype` |
| `handoff [focus]` | `$rite-handoff` |
| `zoom-out` | `$rite-zoom-out` |
| `pressure-test` | `$rite-pressure-test` |
| `autocomplete [idea] [--ship]` | `$rite-autocomplete` |
| `quick [change]` | `$rite-quick` |
| `frame [task]` | `$rite-frame` |

Both forms hit the same skill — the menu form for discovery, the `/rite-<verb>` shortcut for muscle memory.

`use <slug>` is handled **inline** — there is no `rite-use` skill. Confirm
`.devrites/work/<slug>/` exists, then re-point `.devrites/ACTIVE` to `<slug>` and report
the now-active feature. It is cheap context-switching only — no re-spec, no phase run. If
the workspace is missing, list the slugs under `.devrites/work/` and stop.

`guide` is also inline — a first-feature walkthrough that teaches the lifecycle by running
it. Agree on one **real, genuinely small** change, then dispatch the normal phases in order
(spec → define → build → prove → seal → ship). Per phase, exactly two narration beats:
before, what it will decide; after, what it wrote in `.devrites/work/<slug>/` and why. Walk
every phase — the small change is what makes the full ceremony affordable to watch. Pause
at each boundary for the user's go-ahead. Teach without lecturing.

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

1. **Verb in `$ARGUMENTS`** → dispatch per the table above.
2. **No args** → menu mode, as above.
3. **Unrecognized first token** → tell the user the known verbs and stop. Don't guess.
4. **No active feature** and the user asked "where am I" or named no verb → point at `$rite spec <feature>` (or `$rite-spec`). Don't summarize state yourself — `$rite status` (or `$rite-status`) owns that.

## Menu

```
DevRites — disciplined senior-engineer workflow
Recommended start: <greenfield: $rite spec <feature> | brownfield-unadopted: $rite adopt | active-feature: $rite status | dirty-worktree: $rite frame or $rite quick | branch-ahead: $rite ship/status | clean-default: $rite spec <feature>>
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
DOCTOR        $rite doctor ...         ≡    $rite-doctor      health check; --reindex refreshes structural indexes
LEARN         $rite learn ...          ≡    $rite-learn       review captured lessons → promote to project rules / principles
EXPLAIN       $rite explain ...        ≡    $rite-explain     grounded concept/diff/idea/recap explainer
POV           $rite pov ...            ≡    $rite-pov         decide adopt / trial / hold / reject for an external option
DOGFOOD       $rite dogfood ...        ≡    $rite-dogfood     browser QA by changed user journey
PR FEEDBACK   $rite pr-feedback ...    ≡    $rite-pr-feedback fix and resolve PR review threads
CUSTOMIZE     $rite customize ...      ≡    $rite-customize   author overrides/extensions without forking the pack
SWITCH        $rite use <slug>                                re-point .devrites/ACTIVE to another feature (inline)
GUIDE         $rite guide                                     first feature, guided — full lifecycle on one small real change (inline)
RESUME        $rite resolve ...        ≡    $rite-resolve     answer a HITL checkpoint
AUTO          $rite autocomplete ...   ≡    $rite-autocomplete  run the whole lifecycle unattended (--ship to push)
QUICK         $rite quick <change>     ≡    $rite-quick       express lane — one small reversible change (escalates if it grows)
UTILITY       $rite frame | prototype | handoff | zoom-out | pressure-test  (or direct shortcuts)
```

> **Small one-off change?** A typo, copy tweak, config bump, or one-function fix → **`$rite-quick`**
> (express lane: one contract → build → prove → ship, no full workspace). It escalates to
> `$rite-spec` the instant the change grows past small / reversible / unambiguous. The full
> lifecycle above is for real features — don't pay its ceremony for a one-off.

## Core operating rules (every DevRites skill enforces)

The operating rules live in `.agents/skills/devrites-lib/reference/standards/core.md`; DevRites skills Read it as
their first step, and the other rule files load on demand. See
[`.agents/skills/devrites-lib/reference/standards/README.md`](../devrites-lib/reference/standards/README.md) for the full index.
