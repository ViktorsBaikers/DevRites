---
name: rite-prove
description: Prove completed feature with full tests/build/typecheck/lint, end-to-end/browser evidence, screenshots, commands and outputs for seal. Not for single-slice proof.
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
> (Playwright MCP → DevTools MCP → `/run`+`/verify` → project E2E →
> manual), and writes `evidence.md` + `browser-evidence.md` keyed to the
> active `.devrites/work/<slug>/` (`.devrites/features/<slug>/` remains readable
> during migration). Use `/verify` or `/run` on their own
> when there is no DevRites feature workspace.

## Gate: all slices must be built first
Read `tasks.md` + `state.md`. **If ANY slice is still pending/unbuilt, STOP** and tell the
user to finish it with `/rite-build` — `/rite-prove` runs once, when the full task is
complete, not after each slice. (Each slice already got its own targeted tests during
`/rite-build`; this phase is the comprehensive proof of the assembled feature.)

**Never report a pass you didn't observe.** If a command couldn't run, say so and give exact manual steps.

**Re-runnable, scoped.** `/rite-prove` runs once when the full feature is assembled, but
it can be **re-run scoped** afterwards: when `/rite-polish` or `/rite-review` edit code,
the existing `evidence.md` no longer post-dates the change, so re-run `/rite-prove` over
the affected criteria/routes to refresh proof before `/rite-seal`.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.claude/skills/devrites-lib/reference/standards/core.md` first. The other rule files load on demand;
pull these via `Read` when relevant:
- `testing.md` — pyramid, determinism, no-flake discipline.
- `test-proof-checklist.md` — compact proof-quality gate for tests and recorded evidence.
- `browser-proof-checklist.md` — for UI scope, the required browser states and Visual Verdict evidence.
- `spec-grammar.md` — when the spec uses structured `### Requirement:` / `#### Scenario:`
  blocks, each scenario (WHEN/THEN) is one observable behavior to walk and prove.
- `performance.md` — measure first when perf is in scope.
- `observability.md` — when the change has a runtime surface (endpoint, job, integration,
  user flow): telemetry must be present **and observed to emit**, not assumed.
- `developer-experience.md` — when the change ships a developer-facing surface (API / CLI / SDK /
  webhook / config / error messages / getting-started): **measure** the DX scorecard (run the flow,
  time time-to-hello-world, capture the verbatim error text), don't assert it.
- `definition-of-done.md` — standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- Evidence over confidence. Feature scope only — fix within the feature or record a
  blocker; don't refactor unrelated code.
- Spec Drift Guard applies: if tests/evidence reveal the spec is wrong, stop and handle
  drift (`rite-build/reference/spec-drift-guard.md`).

## Workflow
0. Read `.claude/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above when relevant.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. **Confirm the gate** (all slices built). Read `spec.md` (acceptance criteria +
   "Commands discovered"), `tasks.md`, `state.md`, `test-plan.md` if present (the vetted
   coverage target from `/rite-vet`), and the full `git diff`.
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
   proven and which aren't. **If the spec uses the structured grammar** (`### Requirement:` /
   `#### Scenario:` blocks — `spec-grammar.md`), walk it **per scenario**: each `#### Scenario:`
   WHEN/THEN is one observable behavior that needs a passing asserting test (the WHEN is the
   arrange, the THEN the assert). A scenario with no covering result is an unproven gap =
   blocker, the same standing as an uncovered acceptance criterion. Re-run the grammar gate
   first so a requirement hand-edited to malformed since `/rite-spec` can't masquerade as proven
   (`--against` re-checks any delta classification against the ledger, same as the spec gate):
   ```bash
   devrites-engine spec-validate ".devrites/work/<slug>" --against .devrites/specs; echo "spec-validate rc=$?"
   ```
   If `test-plan.md` exists, also walk its acceptance→test map and
   per-gap requirements — a planned test (especially a regression-Critical) with no covering
   result is an unproven gap, not a pass. **Also walk the test-plan interaction inventory**
   (every interactive element + user flow): each must have a passing asserting test. An
   element/flow with no asserting result is an **unproven gap = blocker** — a NO-GO at
   `/rite-seal`, the same standing as an unproven acceptance criterion (`testing.md`
   "Completeness"). For UI, the browser proof (step 4) demonstrates the flows; the asserting
   tests prove the elements.
5a. **Assertion-strength spot check (critical paths).** For each regression-Critical /
   irreversible / data-loss path, confirm the covering test actually *can* fail: reject
   tautological assertions (`toBeDefined`-only, asserting the mock), and **fault-inject** —
   break the code on purpose (flip a comparison, drop a guard) and confirm the test goes red,
   then revert. Run the project's mutation-testing tool over the touched criticals if it has
   one. A test that stays green on deliberately broken code is an unproven gap, not a pass
   (`testing.md` "Assertion strength"). Record what was fault-checked in `evidence.md`.
   Run the deterministic gates rather than eyeballing:
   ```bash
   devrites-engine test-integrity; echo "test-integrity rc=$?"   # exit 3 = a test was weakened to pass → NO-GO
   devrites-engine mutation-gate   # changed-files mutation score → band the seal verdict
   devrites-engine package-existence; echo "package-existence rc=$?"   # exit 3 = a new import isn't declared in any manifest (hallucinated/typo-squatted dep) → NO-GO
   ```
   For a parser / serializer / encoder / auth-token / pure-transform criterion, add a **round-trip
   or metamorphic property** check (`decode(encode(x))==x`, `parse∘print==id`) over generated inputs —
   example tests miss the edge cases these explore. If the same unit regenerated from a paraphrased
   spec (or a second sample) **diverges in behaviour on shared inputs**, treat that as a low-confidence
   signal: under AFK it blocks an auto-GO and routes to HITL.
5b. **Observability check (runtime surface only).** If the feature added an endpoint, job,
   queue consumer, external integration, user-facing flow, or a new error path, apply the
   on-call test (`observability.md`): are the signals needed to debug a prod failure present —
   structured logs on the failure path, a metric/counter on errors, a trace id across any
   boundary? Then **observe them fire**: trigger the path and confirm the log line / metric /
   span actually emits, and record that observation in `evidence.md`. Instrumentation never seen
   emitting is unproven, not done. Skip entirely for pure-internal / docs / config / type-only
   changes — don't instrument a typo fix.
5c. **Developer-experience measure (developer-facing surface only).** If the feature ships a public
   API, CLI, SDK/library, webhook, config/env contract, error/exit path, or the getting-started flow
   (`developer-experience.md`), **exercise it** rather than reading it: run the getting-started steps
   on a clean state and **time time-to-hello-world**; invoke the CLI `--help` / call the endpoint /
   import the package; trigger the failure path and capture the **verbatim** error text. For a docs or
   quickstart page, capture it through the browser-proof ladder (`devrites-browser-proof`) and describe
   the screenshot. Write the **measured** scorecard to `devex.md` (beside the `/rite-vet` prediction the
   boomerang reconciles at `/rite-seal`) and the headline numbers + error strings to `evidence.md`. A
   scorecard from "the code looks fine" is Source mode, not proof. Skip entirely when no developer-facing
   surface is in scope — don't DX-measure an internal refactor.
6. **On failure** → [failure-triage](reference/failure-triage.md) +
   `devrites-debug-recovery`. Reproduce → isolate → fix within scope → re-run; if a fix
   would exceed scope, record a blocker.
7. Update `evidence.md`, `browser-evidence.md` (if UI), `traceability.md`, and
   `state.md`. `proof.md` remains a readable migration alias only; new proof goes
   to `evidence.md`.

> **Mid-flight discipline.** When tempted to claim an un-observed pass, skip a rung of the browser-proof ladder, or proceed with slices pending — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: feature proof complete for <slug>.
Changed: evidence.md, browser-evidence.md <updated|n/a>, devex.md <updated|n/a>, state.md
Evidence: acceptance <n>/<total>; scenarios <n>/<total|n/a>; tests/build/lint/browser <pass|fail summary>
Open: <none | unresolved blockers | unproven criteria>
Next: /rite-polish
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before /rite-polish
```
— (see hard rule above).
