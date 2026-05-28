---
name: rite-prove
description: Prove the completed feature — full tests + build + typecheck + lint + browser-proof ladder — walking `spec.md` acceptance criteria one by one. Use when the user says "prove this", "run the full tests", "is this feature really done", "check it end-to-end". Not for single-slice or pre-completion proof.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-prove — prove the completed feature

Turn "I think it works" into recorded evidence for the **whole feature**. Read the active
workspace first; if none, run `/rite-spec <feature>`.

> **Differs from built-in `/verify` and `/run` in:** `/verify` proves a
> single change; `/run` launches the app. `/rite-prove` is feature-scoped:
> it walks `spec.md` acceptance criteria one-by-one, runs the full relevant
> test suite + build/typecheck/lint, ascends the browser-proof ladder
> (browser-harness → DevTools MCP → `/run`+`/verify` → project E2E →
> manual), and writes `evidence.md` + `browser-evidence.md` keyed to the
> active `.devrites/work/<slug>/`. Use `/verify` or `/run` on their own
> when there is no DevRites feature workspace.

## Gate: all slices must be built first
Read `tasks.md` + `state.md`. **If ANY slice is still pending/unbuilt, STOP** and tell the
user to finish it with `/rite-build` — `/rite-prove` runs once, when the full task is
complete, not after each slice. (Each slice already got its own targeted tests during
`/rite-build`; this phase is the comprehensive proof of the assembled feature.)

**Never report a pass you didn't observe.** If a command couldn't run, say so and give exact manual steps.

## Rules consulted (read on demand from `pack/.claude/rules/`)
`core.md` is already loaded. Pull these via `Read` when relevant:
- `testing.md` — pyramid, determinism, no-flake discipline.
- `performance.md` — measure first when perf is in scope.

## Operating rules
- Evidence over confidence. Feature scope only — fix within the feature or record a
  blocker; don't refactor unrelated code.
- Spec Drift Guard applies: if tests/evidence reveal the spec is wrong, stop and handle
  drift (`rite-build/reference/spec-drift-guard.md`).

## Workflow
1. **Confirm the gate** (all slices built). Read `spec.md` (acceptance criteria +
   "Commands discovered"), `tasks.md`, `state.md`, and the full `git diff`.
2. **Discover commands** if not recorded —
   [test-command-discovery](reference/test-command-discovery.md): README, package
   scripts, Makefile, CI configs, Gemfile/Rakefile, pyproject, go.mod, Cargo.toml.
3. **Run the full relevant test suite** for the feature (not a single slice), then the
   relevant **build / typecheck / lint**.
4. **UI feature?** Run the browser proof ladder over the feature's routes —
   [proof-ladder](reference/proof-ladder.md) + [browser-proof](reference/browser-proof.md)
   (`devrites-browser-proof`): routes, viewports, screenshots (opened + described),
   console, network, interaction paths, and design-reference match if references exist.
5. **Map results to acceptance** — walk `spec.md` acceptance criteria; note which are now
   proven and which aren't.
6. **On failure** → [failure-triage](reference/failure-triage.md) +
   `devrites-debug-recovery`. Reproduce → isolate → fix within scope → re-run; if a fix
   would exceed scope, record a blocker.
7. Update `evidence.md`, `browser-evidence.md` (if UI), and `state.md`.

> **Mid-flight discipline.** When tempted to claim an un-observed pass, skip a rung of the browser-proof ladder, or proceed with slices pending — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output
```
Proved: <feature>
Acceptance criteria proven: <n / total>
Tests:  <cmd → pass/fail (counts)>
Build:  <cmd → pass/fail>   Lint: <cmd → pass/fail>
Browser: <ladder rung used + summary | n/a>
Unresolved failures / blockers: <none | list>
Next: /rite-polish   (finish the feature → /rite-review → /rite-seal)
↻ Hygiene: /clear before /rite-polish (evidence.md + browser-evidence.md captured; debug trails noisy). See rules/context-hygiene.md.
```
— (see hard rule above).
