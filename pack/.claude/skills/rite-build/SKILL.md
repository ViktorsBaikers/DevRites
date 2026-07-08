---
name: rite-build
description: Implement exactly ONE vertical slice of the active feature, then stop with evidence. Use when the user says "build the next slice", "implement slice N", "continue the build", "code this slice". Not for bug fixes, prototypes, refactors outside scope, or two slices in a row.
argument-hint: "[slice number or name]"
user-invocable: true
---

# /rite-build — one verified slice

Build the next single slice, leave it working and proven, then **stop**. **Read the
active workspace first**; if none, tell the user to run `/rite-spec <feature>`.

This skill is the **orchestrator**: it owns the gates and the workspace; a fresh-context
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) owns the **writing**. You run
pre-flight (readiness, slice select, HITL pause), dispatch the wright for the build core, then
run the post-return gates (doubt, fail-on-red, record, stop). See
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
DevRites skills Read `.claude/skills/devrites-lib/reference/standards/core.md` as their first step (workflow step 0). The
following load on demand — **the wright reads them** (they are named in its contract) while it
writes; read them yourself for the doubt/record gates or in the inline fallback:
- `coding-style.md` — naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md` — fail fast, no silent catches, fail closed.
- `testing.md` — pyramid, behaviour over implementation, see-it-fail-first.
- `patterns.md` — composition over inheritance, avoid premature abstraction.
- `principles.md` — the project invariants (`.devrites/principles.md`) the slice must honor; the wright reads them as **binding**, not priors.
- `security.md` — when the slice touches user input, auth, data, or external integrations.

## Operating rules
- **One slice at a time. DO NOT** start the next slice without the user asking.
- Evidence over confidence. Prefer existing conventions. Feature scope only — no
  drive-by refactors.
- **Noticed, not touched.** An adjacent smell the wright sees outside `touched-files.md` is
  recorded as an FYI follow-up in `decisions.md`, never fixed inline — the slice's change summary
  states what it *deliberately left alone* ([`git-workflow.md`](../devrites-lib/reference/standards/git-workflow.md) "Things
  I didn't touch"), so the reviewer reads a feature-scoped diff, not a renovation. The `devrites-engine reconcile`
  gate (step 6) enforces this by exit code.
- **Don't re-run an unchanged check.** Re-running the same build/test command on code that hasn't
  changed since proves nothing new — it's motion, not evidence. Re-verify after an edit, not before.
- Surface material assumptions; ask before adding dependencies or a second design
  system. The [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing.** `devrites-slice-wright` enforces the anti-slop charter **at
  the source** — the canonical do-not list is `rite-polish/reference/anti-ai-slop.md` (the
  wright reads it; don't restate it here). It writes the code the *project* would write, in its
  idiom, reusing before building; **you verify the charter held on return** — you do not re-list
  it and you do not fix slop by editing source. Polish catches what slips; build prevents.
  The **prose you write yourself** — `evidence.md`, `decisions.md`, the slice report — follows
  the human-voice charter (`.claude/skills/devrites-lib/reference/standards/prose-style.md`; depth in `devrites-prose-craft`): no
  filler openers, no marketing adjectives, exact commands and identifiers kept verbatim.
- **Honor declared project principles.** The wright reads `.devrites/principles.md` and treats
  each invariant as **binding** (not a prior to weigh like a convention) — a slice it cannot build
  without breaking one is an **Escalation**, not a silent violation. On return **you verify no
  principle was broken**; a fresh violation is handled like any irreversible-risk item — a
  human-approved, scoped exception in the register or a stop, never folded into the slice. No
  `.devrites/principles.md` → none declared → nothing to honor.
- **You never edit source — the wright is the only writer of code + tests.** You write only
  `.devrites/` bookkeeping. On any red gate, doubt finding, or coverage gap your only remedies
  are **continue the same wright once** (it fixes in its own context) or **stop + escalate** —
  never patch the code yourself. The `devrites-engine reconcile` gate (step 6) enforces this by exit code:
  any source file changed outside the wright's claimed set is a hard STOP.
- **A `Forge: yes` slice competes candidates — one author still lands.** When `/rite-vet`
  flagged the slice a genuine architecture fork, step 3 runs K=2–3 candidate wrights in
  **isolated worktrees** and lands exactly one winner's diff; the single-writer invariant holds
  because no tree ever has two authors and only the winner reaches the working tree. You still
  never edit source, and reconcile runs against the winner's claimed set. The default slice is
  single-path — forge is the rare exception ([`reference/forge.md`](reference/forge.md)).

## Workflow ([one-slice-cycle](reference/one-slice-cycle.md))

> **Running the gate helpers.** Each gated `bash` block calls a named helper directly:
> `devrites <helper> [args]` runs the helper, **propagates its exit code** (so the `rc=$?`
> checks below still hold) and degrades to a printed note if the binary is absent. The
> `devrites` binary is the shared runner.

0. **Rules + AFK + readiness check.** Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Then **run the
   shared orientation preamble** — it prints `state.md`, the artifacts present, the run
   mode (HITL/AFK), and the open-question tally by gate, deterministically:
   ```bash
   devrites-engine preamble
   ```
   Then **run the readiness gate** — it enforces the step-0 stop conditions by exit code,
   not by memory:
   ```bash
   devrites-engine build-readiness; echo "readiness rc=$?"
   ```
   A non-zero `rc` is a hard STOP: `2` → `/rite-define` (plan not approved), `3` →
   `/rite-resolve` (awaiting human), `4` → `/rite-plan` (blocked). The prose below is the
   same gate for installs without the script.
   Orient from its digest. If `Status == awaiting_human` → **STOP**, tell the user to run
   `/rite-resolve <qid> "<answer>"`. If `state.md` has no `Plan approved: <iso>` field
   → **STOP**, tell the user the plan isn't approved yet (`/rite-define` writes it when
   the human confirms). If `.devrites/AFK` is present, re-derive the remaining AFK budget
   from `state.md`'s `AFK slices remaining: <n>` field (initialized from `.devrites/AFK`
   `max_slices` on the first AFK build); if it is `0` → **STOP** (forced HITL stop; raise
   the count in `state.md` or remove the sentinel to continue). See
   [`reference/afk-discipline.md`](reference/afk-discipline.md).
1. Read `spec.md`, `plan.md`, `tasks.md`, `assumptions.md`, `drift.md`, and `test-plan.md`
   if present (the vetted coverage target from `/rite-vet` — the slice's tests come from
   here when it exists). `state.md` and the open-`questions.md` tally are already in the
   preamble digest from step 0 — re-read `questions.md` only for the full text of a flagged
   blocking question.
   If a **blocking `[NEEDS CLARIFICATION]`** remains or the spec/plan readiness gates
   don't pass, stop → `/rite-spec` (to resolve) or `/rite-plan` (to repair). Don't build
   on an unresolved spec.
2. Select the next pending slice (or the one in `$ARGUMENTS`). **Restate its goal,
   acceptance criteria, and scope boundary** in one short block. Confirm it's still the
   right next slice. Write the slice's `Mode` to `state.md` as `Slice mode: <HITL|AFK>` on
   **every** selection (not only on the HITL pause path); `/rite-resolve` clears or updates
   it on resume.
2a. **HITL gate (pre-action pause).** Read the slice's `Mode`. If `HITL`, surface the
    checkpoint as a ranked **option set** and resolve it **before** any code lands, per
    [`reference/checkpoint-protocol.md`](reference/checkpoint-protocol.md). Branch on whether
    a human is here:
    - **Human present (interactive — no `.devrites/AFK`) → ask inline via `AskUserQuestion`.**
      This is the default for an interactive build. Render 2–4 options, the recommended one
      **first and labelled `(Recommended)`**, each with its dimension-tagged rationale + the
      trade-off it accepts, plus the `Something else — I'll describe it` escape hatch. The
      human picks; record the pick to `questions.md` (`answered`) + `decisions.md` (through the
      `devrites-engine resolve` writer), clear the gate, and **continue to step 3 in place** — do **not**
      STOP, do **not** route through `/rite-resolve`.
    - **Human absent / AFK (`.devrites/AFK` present) → auto-pick or persist + STOP.** For a gate
      in `allow_gates`, auto-pick the recommended option (option 1) and proceed; otherwise
      append the `questions.md` entry, write the `Awaiting human` block to `state.md`, set
      `Status: awaiting_human`, fire the `notify:` hook, then **STOP** — resume when the user
      runs `/rite-resolve <qid> "<answer>"`. `blocking` / `escalating` / irreversible-risk gates
      always take this stop path, never the AFK auto-pick.
3. **Snapshot the tree, then dispatch the build core to `devrites-slice-wright`** — one `Task`
   call, fresh context. **First**, capture the pre-dispatch tree so the reconcile gate (step 6)
   can prove you never touched source — run this immediately before the `Task` call:
   ```bash
   devrites-engine reconcile snapshot
   ```
   Then log the dispatch so the stuck-loop detector can catch a slice that keeps being
   re-dispatched without progress (it pauses the build even under AFK):
   ```bash
   devrites-engine stuck log "$(cat .devrites/ACTIVE 2>/dev/null)" dispatch "<slice id>"
   ```
   **Forge branch — only if the selected slice is `Forge: yes`.** Instead of the single dispatch
   below, run the competitive build per [`reference/forge.md`](reference/forge.md): K=2–3 candidate
   wrights, each in an **isolated git worktree** on the distinct strategy `/rite-vet` named, then a
   fresh-context [`devrites-forge-judge`](../../agents/devrites-forge-judge.md) scores them against
   acceptance + `test-plan.md` + `.devrites/principles.md` + the anti-slop charter, and you land
   exactly **one** winner's diff in the working tree, graft any cheap runner-up improvement by
   continuing the winning wright once, and write `forge-report.md`. Forge returns the **same shape**
   a single wright does (one structured artifact, for the winner), so steps 4–7 (doubt, fail-on-red,
   reconcile against the winner's claimed set, record, stop) run **unchanged**. If you cannot give
   the judge an objective scorecard (the slice lacks acceptance / `test-plan.md` coverage) or cannot
   name two genuinely different strategies, the flag is stale — clear it and build single-path. The
   single-writer invariant is intact: each candidate owns its own tree, exactly one author's diff
   lands. Then **stop** here for this slice; the steps below are the default single-path dispatch.

   Then assemble the slice contract and send it per
   [`reference/wright-dispatch.md`](reference/wright-dispatch.md): the slice goal, acceptance
   criteria, and **scope boundary**; the paths it may touch (`touched-files.md`); the context
   paths to read (`spec.md`, `plan.md`, `decisions.md`, `assumptions.md`, `.devrites/principles.md`
   when present — the binding invariants the slice must honor — plus `test-plan.md`
   when present — its per-gap test requirements + regression-criticals for this slice are the
   coverage the wright must write — and `design-brief.md`
   when the slice touches UI per [frontend-trigger](reference/frontend-trigger.md)); and the
   `.claude/skills/devrites-lib/reference/standards/` files in scope. The wright **orients** on the project's idiom (using a
   code-intelligence index **if available** — `codebase-memory-mcp` first, cross-checked with
   `codegraph` (`.codegraph/` / `codegraph_*`) + `graphify` (`graphify-out/`), else standard
   methods (LSP / `Read`/`Grep`/`Glob`); see `.claude/skills/devrites-lib/reference/standards/tooling.md` — for
   placement/callers/impact), writes the **failing test first** when
   behaviour changes ([tdd](reference/tdd.md)), implements the **smallest complete** version in
   the project's style (applying `devrites-frontend-craft` to `design-brief.md` for UI, and
   `devrites-source-driven` — with context7 if available — for uncertain framework facts), runs the slice's **targeted tests**
   (plus typecheck / lint / build where the project has them), and returns a structured artifact
   — **code + tests only; it does not write the workspace files.** If the slice is UI but no `design-brief.md` exists (e.g. a spec written before
   shaping), shape it via `devrites-ux-shape` before the wright codes. If the `Task` tool is
   unavailable, run the wright's discipline **inline** as a flagged fallback (see the reference)
   — same one-slice cycle, no isolation; in that case write
   `.devrites/work/<slug>/.reconcile-inline` so the reconcile gate skips (you are legitimately
   the writer in this fallback).
4. **Doubt the decisions it stood up.** For each entry in the wright's `Decisions stood`
   (branching, boundary crossing, data model, auth, public API, migration, user-flow change,
   "this is safe/scales") apply `devrites-doubt` **before accepting the slice** — the writer
   doesn't grade its own decisions. Each `devrites-doubt` invocation **dispatches the
   `devrites-doubt-reviewer` subagent** (doing the adversarial pass inline is the writer grading
   itself — the thing this step exists to forbid). **Completion criterion (checkable):** step 4
   is done only when **every** `Decisions stood` entry carries a recorded `devrites-doubt`
   verdict — `accept`, or `reject` + the required changes — in `decisions.md` (accepted
   trade-offs) / `questions.md` (open gates). A `Decisions stood` entry with **no verdict on
   record** means doubt did not run for it: **do not enter step 5 and do not mark the slice
   `built`.** Log each dispatch so the seal can prove doubt ran (the footprint already counts a
   `doubt` kind):
   ```bash
   devrites-engine footprint log <slug> doubt "<decision id>"
   ``` The doubt loop honours `.devrites/AFK` (see its AFK
   exception): findings below the slice's gate ceiling become advisory entries in `questions.md`;
   destructive / auth / public-API concerns always pause regardless. A non-empty `Escalation` in
   the artifact is handled here too: irreversible-risk / blockers → blocking question + set
   `Status: awaiting_human`; a scope-changing answer → `/rite-plan repair` (Spec Drift Guard),
   never silently into the slice. **If an irreversible-risk item shows up under the wright's
   `Decisions stood` rather than `Escalation`**, treat that misclassification as itself a
   blocking protocol violation — pause and re-dispatch with the item flagged out-of-bounds, do
   **not** doubt-and-accept it. (The wright's return is the not-yet-load-bearing moment — the
   slice isn't `built` or merged yet — so this post-return doubt is still pre-commit.)
   **Principle check (same standing):** a wright return that breaks a declared principle
   (reported in its `Principles` field, or that you detect against `.devrites/principles.md`) is
   handled here like an irreversible-risk item — block, route to a human-approved scoped
   exception in the register or stop; never doubt-and-accept a principle violation into the slice.
5. **Fail-on-red.** If the wright's `Gates` were red (targeted tests / types / lint) or it
   couldn't verify: do **not** mark the slice `built`, and **do not fix the code yourself**.
   First remedy — **continue the same wright once** (`SendMessage` to it, carrying the failing
   gate + its real output) so it fixes in its own context. This retry is for **objective
   failures only** — red gate / type / lint / missing test coverage / UI browser-proof fail —
   never a contested decision (that routes to `/rite-plan repair`). **Still red after the one
   retry** → escalate: AFK → append a blocking question to `questions.md` (gate=blocking,
   slice's SLA) + set `Status: awaiting_human`; HITL → pause as a blocking gate. Either way,
   `Next step: /rite-plan unblock` until resolved.
6. **Record — you are the canonical writer.** **First, run the reconcile gate (A1):** write the
   wright's reported `Files changed` paths (one per line) to
   `.devrites/work/<slug>/.reconcile-claimed` (or the active legacy
   `.devrites/features/<slug>/` workspace during migration), then:
   ```bash
   devrites-engine reconcile check; echo "reconcile rc=$?"
   ```
   **Exit 5 → hard STOP:** a source file changed outside the wright's claimed set — code was
   edited by something other than the wright (A1 breach). Revert it and re-dispatch the wright;
   do **not** mark the slice `built`.

   **Then run the test-integrity gate (anti-reward-hacking)** — prove the slice didn't reach
   green by weakening its tests:
   ```bash
   devrites-engine test-integrity; echo "test-integrity rc=$?"
   ```
   **Exit 3 → hard STOP:** a test was deleted, skipped, or de-asserted since the slice base — the
   slice went green by weakening its tests, a Critical protocol violation. Revert the weakening and
   re-dispatch the wright; do **not** mark the slice `built`.

   **Then run the package-existence gate (anti-hallucination)** — every new third-party import must
   be declared in a project manifest, not just imported:
   ```bash
   devrites-engine package-existence; echo "package-existence rc=$?"
   ```
   **Exit 3 → STOP:** an imported package is not declared in any manifest (`package.json`, `go.mod`,
   `requirements.txt`, `pyproject.toml`, `Pipfile`, `Cargo.toml`) — the classic shape of a
   hallucinated or typo-squatted dependency. Confirm the name on the registry and declare it via the
   package manager, or remove the import; do **not** mark the slice `built`. The gate is deterministic
   and fail-open (not a git repo / no manifest / stdlib-only import → rc 0).

   Then, from the wright's artifact, update `state.md`,
   `evidence.md`, `touched-files.md` (and `browser-evidence.md` for UI). **Persist every
   `Decisions stood` entry from the wright's artifact to a `## Decisions stood` section in
   `decisions.md`, one line each ending `— doubt: <accept | reject-resolved | MISSING>`** — the
   verdict from step 4, or `MISSING` if step 4 was skipped for that entry. Write this ledger at
   record-time **independent of the doubt step**, so a stood decision lands on disk even when it
   was never doubted: `/rite-seal` cross-checks the ledger, and a `doubt: MISSING` entry is the
   skipped-doubt finding that would otherwise leave no trace (the per-slice skip the footprint
   count alone can't catch). If the wright stood no decisions, write `## Decisions stood` with
   `- none` so the absence is recorded, not assumed. If the wright reported
   an approach it tried and backed out of, record it under a `## Dead ends` section in
   `decisions.md` so a retry or the next agent doesn't repeat it.
   **If the wright's `Conventions` field reports a contradiction** (the live code now
   disagrees with a held convention), record the drift — you own this bookkeeping:
   ```bash
   devrites-engine conventions contradict --key <key> --slug <slug> \
     --evidence "<what the live code does>" --drift-file .devrites/work/<slug>/drift.md
   ```
   It lowers the convention's band (or retires it) and appends a `convention-drift` entry to
   `drift.md`. The ledger is **promoted only at `/rite-seal` on GO** — build only records
   contradictions, never new conventions. **Evidence is the
   wright's real command output, not its say-so.** Capture per
   [evidence-standard](reference/evidence-standard.md). Append a footprint record for this
   slice's wright dispatch (deterministic run-weight bookkeeping the seal reports):
   ```bash
   devrites-engine footprint log <slug> wright "<slice id>"
   ```
   If `.devrites/AFK` is present, decrement
   the budget by running `devrites-engine tick-afk <state.md path>` —
   it decrements `state.md`'s `AFK slices remaining` field, prints the new value, and exits `3`
   when it hits 0. **Exit 3 → STOP** (forced HITL stop; the cap is exhausted). Never rewrite
   `.devrites/AFK` `max_slices` in place — it is read-only initial budget.
7. **STOP.** Render the progress footer, then report and recommend the next step. Run the
   shared footer (mirror of the step-0 preamble) so the slice meter + flow ribbon are
   deterministic, not model-typed:
   ```bash
   devrites-engine progress
   ```
   It reads `state.md` (already updated in step 6), so the meter reflects the slice you
   just built. **When every slice is built** (`✅ ALL BUILT`) say so explicitly — the build
   phase is complete, the next phase is `/rite-prove` — don't leave completion implicit.

> **Mid-flight discipline.** The wright (or you, in the inline fallback) must resist doing two slices, skipping TDD, adding a defensive check, or wandering outside `touched-files.md`; you must resist skipping the post-return `devrites-doubt` because the wright "seems confident" — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

The `devrites-engine progress` footer (step 7) prints the deterministic header, slice meter,
and flow ribbon. Put the compact status lines below it using the shared completion
reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).

**Slice built; slices remain:**
```
Done: built slice <n> — <name>.
Changed: <files touched>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd -> pass>; browser <summary | n/a>; drift <none|handled>
Open: <none | blockers | questions>
Next: /rite-build
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear between slices; /rite-handoff if away > a few hours
```

**Slice built; all slices built:**
```
Done: built slice <n> — <name>; all slices built.
Changed: <files touched>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd -> pass>; browser <summary | n/a>; drift <none|handled>
Open: <none | blockers | questions>
Next: /rite-prove
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before /rite-prove; /rite-handoff if away > a few hours
```

For awaiting-human or stopped states, use the typed templates from the reply contract.
Keep fact lines terse. The meter and ribbon carry progress; don't duplicate them in
prose.

For a **forged slice**, the `Built` line names the competition and points at the record, e.g.
`Done: built slice 3 — csv-streaming (forged: 3 candidates, winner B).` and
`Record: .devrites/work/<slug>/forge-report.md`.

**DO NOT continue to the next slice automatically** — even at `✅ ALL BUILT`, `/rite-prove` is the user's call.
