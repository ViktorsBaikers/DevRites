---
name: rite
description: DevRites menu + router. Without args, renders the menu. With a verb (e.g. `spec`, `build`, `review`), dispatches to the matching `rite-<verb>` skill — `/rite spec foo` is equivalent to `/rite-spec foo`. Use when the user types `/rite` or names a phase verb.
argument-hint: "[verb [args...]]"
user-invocable: true
disable-model-invocation: true
---

# /rite — DevRites menu + router

You are the DevRites entry point. Two modes:

- **No args** → render the menu (below), then stop. Do not execute a workflow phase. Do not read `state.md` / run evidence checks / list artifacts — that's `/rite-status`.
- **Verb arg** → dispatch to the matching `rite-<verb>` skill (see "Dispatch" below). The router is a pass-through: `/rite spec foo` ≡ `/rite-spec foo`; the called skill owns the output.

## Dispatch

If `$ARGUMENTS` starts with a verb in this table, **load the matching skill and execute its workflow** with the remainder of `$ARGUMENTS` as that skill's argument. Try post-install path first, fall back to pre-install:

```bash
V=<verb>; ARGS="<remaining args>"
F=.claude/skills/rite-$V/SKILL.md
[ -f "$F" ] || F=pack/.claude/skills/rite-$V/SKILL.md
# Then Read "$F" and follow its workflow with $ARGS as that skill's $ARGUMENTS.
```

| Verb | Equivalent shortcut | Skill |
|---|---|---|
| `spec [feature]` | `/rite-spec` | start a feature — investigate + write spec.md |
| `adopt [area]` | `/rite-adopt` | onboard an existing codebase — reverse-derive spec.md + seed conventions |
| `temper [--mode]` | `/rite-temper` | optional strategic spec review (scope mode + pre-mortem) before define |
| `define` | `/rite-define` | turn the spec into plan + task slices |
| `vet [--cross-model]` | `/rite-vet` | optional engineering plan review (scope · architecture · tests · perf) before build |
| `plan [mode]` | `/rite-plan` | reshape / reslice / repair an active plan |
| `build [slice]` | `/rite-build` | implement exactly one vertical slice, then stop |
| `prove` | `/rite-prove` | full tests + browser proof |
| `polish [mode]` | `/rite-polish` | code + UI polish |
| `review [scope]` | `/rite-review` | multi-axis feature review |
| `seal` | `/rite-seal` | final GO / NO-GO decision |
| `ship` | `/rite-ship` | type-GO + commit/push/tag, then archive the task |
| `status [slug]` | `/rite-status` | active feature, next action, evidence |
| `use <slug>` | (inline) | switch the active feature — re-point `.devrites/ACTIVE` |
| `resolve <qid> "<answer>"` | `/rite-resolve` | answer a HITL gate |
| `prototype [question]` | `/rite-prototype` | throwaway prototype |
| `handoff [focus]` | `/rite-handoff` | compact chat → handoff doc |
| `zoom-out` | `/rite-zoom-out` | structural map of unfamiliar code |
| `pressure-test` | `/rite-pressure-test` | diverge → converge on a rough idea |
| `autocomplete [idea] [--ship]` | `/rite-autocomplete` | run the whole lifecycle unattended |
| `quick [change]` | `/rite-quick` | express lane — one small reversible change, build → prove → ship |

The `/rite-<verb>` standalones remain user-invocable as direct shortcuts; both forms hit the same skill. Use whichever reads more naturally — the menu form (`/rite spec`) for discovery, the shortcut (`/rite-spec`) for muscle memory.

`use <slug>` is handled **inline** — there is no `rite-use` skill. Confirm
`.devrites/work/<slug>/` exists, then re-point `.devrites/ACTIVE` to `<slug>` and report
the now-active feature. It is cheap context-switching only — no re-spec, no phase run. If
the workspace is missing, list the slugs under `.devrites/work/` and stop.

Specialist triggers (model-invoked inside the above):
`devrites-frontend-craft` (UI) · `devrites-browser-proof` (UI verify) ·
`devrites-source-driven` (uncertain library) · `devrites-doubt` (non-trivial
decision) · `devrites-api-interface` (cross-boundary) ·
`devrites-audit <security|perf|simplify>` (single-axis review/polish pass) ·
`devrites-debug-recovery` (failures). Parallel reviewer fan-out at seal lives
inline in `/rite-seal` (see its `reference/parallel-dispatch.md`).

## What to output

1. **Verb in `$ARGUMENTS`** → dispatch per the table above. The called skill owns the response.
2. **No args** → render the menu below, then stop.
3. **Unrecognized first token** → tell the user the known verbs and stop. Don't guess.
4. **No active feature** and the user asked "where am I" or named no verb → point at `/rite spec <feature>` (or `/rite-spec`). Don't summarize state yourself — `/rite status` (or `/rite-status`) owns that.

## Gotchas
- No args → render the menu and stop. Don't execute a phase, read `state.md`, or summarize status — that's `/rite-status`.
- Unrecognized first token → list the known verbs and stop; never guess which phase the user meant.
- Pure pass-through: dispatch to the `rite-<verb>` skill and let it own the output; don't do the phase's work in the router.

## Menu

```
DevRites — disciplined senior-engineer workflow
                              menu form           direct shortcut
SPEC          /rite spec               ≡    /rite-spec        investigate deeply → write spec.md
ADOPT         /rite adopt              ≡    /rite-adopt       onboard existing code → reverse-derive spec.md + seed conventions
TEMPER        /rite temper             ≡    /rite-temper      optional — strategic review: scope mode + pre-mortem, harden the spec
PLAN          /rite define             ≡    /rite-define      turn the spec into plan + task slices + state
VET           /rite vet                ≡    /rite-vet         optional — engineering plan review: scope · architecture · tests · perf, harden the plan
REPLAN        /rite plan               ≡    /rite-plan        decompose / reslice / repair an active plan
BUILD         /rite build              ≡    /rite-build       implement exactly one verified vertical slice, then stop
PROVE         /rite prove              ≡    /rite-prove       tests + build + runtime + browser evidence
POLISH        /rite polish             ≡    /rite-polish      code polish always; UI normalize + polish if UI
REVIEW        /rite review             ≡    /rite-review      feature-scoped multi-axis review
SEAL          /rite seal               ≡    /rite-seal        final GO / NO-GO decision (no git)
SHIP          /rite ship               ≡    /rite-ship        type-GO + commit/push/tag, then archive + clear ACTIVE
STATUS        /rite status             ≡    /rite-status      active feature, next action, evidence, risks
SWITCH        /rite use <slug>                                re-point .devrites/ACTIVE to another feature (inline)
RESUME        /rite resolve ...        ≡    /rite-resolve     answer a HITL checkpoint
AUTO          /rite autocomplete ...   ≡    /rite-autocomplete  run the whole lifecycle unattended (--ship to push)
QUICK         /rite quick <change>     ≡    /rite-quick       express lane — one small reversible change (escalates if it grows)
UTILITY       /rite prototype | handoff | zoom-out | pressure-test  (or direct /rite-* shortcuts)
```

> **Small one-off change?** A typo, copy tweak, config bump, or one-function fix → **`/rite-quick`**
> (express lane: one contract → build → prove → ship, no full workspace). It escalates to
> `/rite-spec` the instant the change grows past small / reversible / unambiguous. The full
> lifecycle above is for real features — don't pay its ceremony for a one-off.

## Core operating rules (every DevRites skill enforces)

The minimal version lives in `.claude/rules/core.md`; DevRites skills Read it as
their first step, and the other rule files load on demand. The rule list:

1. **Right step, right time** — smallest relevant workflow; don't load everything.
2. **No silent assumptions** — surface material assumptions; ask when the
   answer changes scope, architecture, data model, UX, security, migration
   risk, or acceptance.
3. **No guessing through confusion** — if requirements / code / tests / docs
   conflict, stop, name the conflict, present options, wait for resolution
   when the answer changes the product.
4. **Spec is living, not sacred** — change spec/plan only through the
   Spec Drift Guard; never code against a known-wrong plan.
5. **One slice at a time** — `/rite-build` does one slice then stops.
6. **Evidence over confidence** — tests, builds, runtime, screenshots beat
   assertions; record commands and output.
7. **Feature scope only** — review / simplify / polish / security stay within
   the active feature and touched files.
8. **Prefer existing conventions** — follow the project's architecture,
   components, tokens, tests, and commands.
9. **Verify uncertain facts at the source** — when framework / library
   behaviour matters and isn't certain, check the docs or source and record it.

Detail in [reference/menu.md](reference/menu.md) when needed.
