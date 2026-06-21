---
name: rite-build
description: Implement exactly ONE vertical slice of the active feature, then stop with evidence. A fresh-context `devrites-slice-wright` writes the slice (orient → TDD → verify, anti-slop, project idiom); this skill gates it (readiness, HITL/AFK, doubt loop, Spec Drift Guard) and records the evidence. Use when the user says "build the next slice", "implement slice N", "continue", "code this slice". Not for bug fixes, prototypes, refactors outside scope, or two slices in a row.
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

## Rules consulted (read on demand from `.claude/rules/`)
DevRites skills Read `.claude/rules/core.md` as their first step (workflow step 0). The
following load on demand — **the wright reads them** (they are named in its contract) while it
writes; read them yourself for the doubt/record gates or in the inline fallback:
- `coding-style.md` — naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md` — fail fast, no silent catches, fail closed.
- `testing.md` — pyramid, behaviour over implementation, see-it-fail-first.
- `patterns.md` — composition over inheritance, avoid premature abstraction.
- `security.md` — when the slice touches user input, auth, data, or external integrations.

## Operating rules
- **One slice at a time. DO NOT** start the next slice without the user asking.
- Evidence over confidence. Prefer existing conventions. Feature scope only — no
  drive-by refactors.
- Surface material assumptions; ask before adding dependencies or a second design
  system. The [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing.** `devrites-slice-wright` enforces the anti-slop charter **at
  the source** — the canonical do-not list is `rite-polish/reference/anti-ai-slop.md` (the
  wright reads it; don't restate it here). It writes the code the *project* would write, in its
  idiom, reusing before building; **you verify the charter held on return** — you do not re-list
  it and you do not fix slop by editing source. Polish catches what slips; build prevents.
  The **prose you write yourself** — `evidence.md`, `decisions.md`, the slice report — follows
  the human-voice charter (`.claude/rules/prose-style.md`; depth in `devrites-prose-craft`): no
  filler openers, no marketing adjectives, exact commands and identifiers kept verbatim.
- **You never edit source — the wright is the only writer of code + tests.** You write only
  `.devrites/` bookkeeping. On any red gate, doubt finding, or coverage gap your only remedies
  are **continue the same wright once** (it fixes in its own context) or **stop + escalate** —
  never patch the code yourself. The `reconcile.sh` gate (step 6) enforces this by exit code:
  any source file changed outside the wright's claimed set is a hard STOP.

## Workflow ([one-slice-cycle](reference/one-slice-cycle.md))
0. **Rules + AFK + readiness check.** Read `.claude/rules/core.md` first. Then **run the
   shared orientation preamble** — it prints `state.md`, the artifacts present, the run
   mode (HITL/AFK), and the open-question tally by gate, deterministically:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
   Then **run the readiness gate** — it enforces the step-0 stop conditions by exit code,
   not by memory:
   ```bash
   G=.claude/skills/devrites-lib/scripts/readiness.sh
   [ -f "$G" ] || G="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/readiness.sh"
   [ -f "$G" ] || G=pack/.claude/skills/devrites-lib/scripts/readiness.sh
   [ -f "$G" ] && { bash "$G"; echo "readiness rc=$?"; } || echo "(readiness gate unavailable — apply the prose checks below)"
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
2a. **HITL gate (pre-action pause).** Read the slice's `Mode`. If `HITL` → render the
    checkpoint per [`reference/checkpoint-protocol.md`](reference/checkpoint-protocol.md):
    append a `questions.md` entry with the slice's `Checkpoint:` + `Gate:` + `SLA:`,
    write the `Awaiting human` block to `state.md`, set `Status: awaiting_human`, run
    the `notify:` hook if `.devrites/AFK` defines one, then **STOP**. Resume happens
    when the user runs `/rite-resolve <qid> "<answer>"`.
3. **Snapshot the tree, then dispatch the build core to `devrites-slice-wright`** — one `Task`
   call, fresh context. **First**, capture the pre-dispatch tree so the reconcile gate (step 6)
   can prove you never touched source — run this immediately before the `Task` call:
   ```bash
   RC=.claude/skills/devrites-lib/scripts/reconcile.sh
   [ -f "$RC" ] || RC="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/reconcile.sh"
   [ -f "$RC" ] || RC=pack/.claude/skills/devrites-lib/scripts/reconcile.sh
   [ -f "$RC" ] && bash "$RC" snapshot || echo "(reconcile gate unavailable — verify by hand that only the wright wrote source)"
   ```
   Then log the dispatch so the stuck-loop detector can catch a slice that keeps being
   re-dispatched without progress (it pauses the build even under AFK):
   ```bash
   ST=.claude/skills/devrites-lib/scripts/stuck.sh
   [ -f "$ST" ] || ST="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/stuck.sh"
   [ -f "$ST" ] || ST=pack/.claude/skills/devrites-lib/scripts/stuck.sh
   [ -f "$ST" ] && bash "$ST" log "$(cat .devrites/ACTIVE 2>/dev/null)" dispatch "<slice id>" || true
   ```
   Then assemble the slice contract and send it per
   [`reference/wright-dispatch.md`](reference/wright-dispatch.md): the slice goal, acceptance
   criteria, and **scope boundary**; the paths it may touch (`touched-files.md`); the context
   paths to read (`spec.md`, `plan.md`, `decisions.md`, `assumptions.md`, plus `test-plan.md`
   when present — its per-gap test requirements + regression-criticals for this slice are the
   coverage the wright must write — and `design-brief.md`
   when the slice touches UI per [frontend-trigger](reference/frontend-trigger.md)); and the
   `.claude/rules/` files in scope. The wright **orients** on the project's idiom (using a
   code-intelligence index **if available** — `codebase-memory-mcp` first, cross-checked with
   `codegraph` (`.codegraph/` / `codegraph_*`) + `graphify` (`graphify-out/`), else standard
   methods (LSP / `Read`/`Grep`/`Glob`); see `.claude/rules/tooling.md` — for
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
   doesn't grade its own decisions. The doubt loop honours `.devrites/AFK` (see its AFK
   exception): findings below the slice's gate ceiling become advisory entries in `questions.md`;
   destructive / auth / public-API concerns always pause regardless. A non-empty `Escalation` in
   the artifact is handled here too: irreversible-risk / blockers → blocking question + set
   `Status: awaiting_human`; a scope-changing answer → `/rite-plan repair` (Spec Drift Guard),
   never silently into the slice. **If an irreversible-risk item shows up under the wright's
   `Decisions stood` rather than `Escalation`**, treat that misclassification as itself a
   blocking protocol violation — pause and re-dispatch with the item flagged out-of-bounds, do
   **not** doubt-and-accept it. (The wright's return is the not-yet-load-bearing moment — the
   slice isn't `built` or merged yet — so this post-return doubt is still pre-commit.)
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
   `.devrites/work/<slug>/.reconcile-claimed`, then:
   ```bash
   RC=.claude/skills/devrites-lib/scripts/reconcile.sh
   [ -f "$RC" ] || RC="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/reconcile.sh"
   [ -f "$RC" ] || RC=pack/.claude/skills/devrites-lib/scripts/reconcile.sh
   [ -f "$RC" ] && { bash "$RC" check; echo "reconcile rc=$?"; } || echo "(reconcile gate unavailable — confirm by hand only the wright wrote source)"
   ```
   **Exit 5 → hard STOP:** a source file changed outside the wright's claimed set — code was
   edited by something other than the wright (A1 breach). Revert it and re-dispatch the wright;
   do **not** mark the slice `built`.

   **Then run the test-integrity gate (anti-reward-hacking)** — prove the slice didn't reach
   green by weakening its tests:
   ```bash
   TI=.claude/skills/devrites-lib/scripts/test-integrity.sh
   [ -f "$TI" ] || TI="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/test-integrity.sh"
   [ -f "$TI" ] || TI=pack/.claude/skills/devrites-lib/scripts/test-integrity.sh
   [ -f "$TI" ] && { bash "$TI"; echo "test-integrity rc=$?"; } || echo "(test-integrity gate unavailable — confirm by hand no test was deleted/skipped/loosened)"
   ```
   **Exit 3 → hard STOP:** a test was deleted, skipped, or de-asserted since the slice base — the
   slice went green by weakening its tests, a Critical protocol violation. Revert the weakening and
   re-dispatch the wright; do **not** mark the slice `built`.

   Then, from the wright's artifact, update `state.md`,
   `evidence.md`, `touched-files.md` (and `browser-evidence.md` for UI). If the wright reported
   an approach it tried and backed out of, record it under a `## Dead ends` section in
   `decisions.md` so a retry or the next agent doesn't repeat it.
   **If the wright's `Conventions` field reports a contradiction** (the live code now
   disagrees with a held convention), record the drift — you own this bookkeeping:
   ```bash
   C=.claude/skills/devrites-lib/scripts/conventions.py
   [ -f "$C" ] || C="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/conventions.py"
   [ -f "$C" ] || C=pack/.claude/skills/devrites-lib/scripts/conventions.py
   command -v python3 >/dev/null 2>&1 && [ -f "$C" ] && \
     python3 "$C" contradict --key <key> --slug <slug> --evidence "<what the live code does>" \
       --drift-file .devrites/work/<slug>/drift.md || true
   ```
   It lowers the convention's band (or retires it) and appends a `convention-drift` entry to
   `drift.md`. The ledger is **promoted only at `/rite-seal` on GO** — build only records
   contradictions, never new conventions. **Evidence is the
   wright's real command output, not its say-so.** Capture per
   [evidence-standard](reference/evidence-standard.md). Append a footprint record for this
   slice's wright dispatch (deterministic run-weight bookkeeping the seal reports):
   ```bash
   FP=.claude/skills/devrites-lib/scripts/footprint.sh
   [ -f "$FP" ] || FP="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/footprint.sh"
   [ -f "$FP" ] || FP=pack/.claude/skills/devrites-lib/scripts/footprint.sh
   [ -f "$FP" ] && bash "$FP" log <slug> wright "<slice id>" || true
   ```
   If `.devrites/AFK` is present, decrement
   the budget by running `bash .claude/skills/devrites-lib/scripts/tick-afk.sh <state.md path>` —
   it decrements `state.md`'s `AFK slices remaining` field, prints the new value, and exits `3`
   when it hits 0. **Exit 3 → STOP** (forced HITL stop; the cap is exhausted). Never rewrite
   `.devrites/AFK` `max_slices` in place — it is read-only initial budget.
7. **STOP.** Render the progress footer, then report and recommend the next step. Run the
   shared footer (mirror of the step-0 preamble) so the slice meter + flow ribbon are
   deterministic, not model-typed:
   ```bash
   PR=.claude/skills/devrites-lib/scripts/progress.sh
   [ -f "$PR" ] || PR="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/progress.sh"
   [ -f "$PR" ] || PR=pack/.claude/skills/devrites-lib/scripts/progress.sh
   [ -f "$PR" ] && bash "$PR" || echo "(progress footer unavailable on this install)"
   ```
   It reads `state.md` (already updated in step 6), so the meter reflects the slice you
   just built. **When every slice is built** (`✅ ALL BUILT`) say so explicitly — the build
   phase is complete, the next phase is `/rite-prove` — don't leave completion implicit.

> **Mid-flight discipline.** The wright (or you, in the inline fallback) must resist doing two slices, skipping TDD, adding a defensive check, or wandering outside `touched-files.md`; you must resist skipping the post-return `devrites-doubt` because the wright "seems confident" — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

The `progress.sh` footer (step 7) prints the first three lines — the header rule, the
**slice meter** (how many of N are built), and the **flow ribbon**. Your fact lines sit
under it. Two shapes:

**Slices still pending:**
```
── rite-build ──────────────────────────────
Slice 3/5  ██████░░░░  csv-streaming ✓
Flow   spec ✓ define ✓ build ◉ prove ○ polish ○ review ○ seal ○ ship ○
Built     slice 3 — csv-streaming   (by devrites-slice-wright | inline fallback)
Acceptance ✓ met · Tests <cmd → pass> · Browser <summary | n/a> · Drift <none | handled>
Next  ▸ /rite-build  (slice 4 — pagination)
↻ Hygiene  /clear between slices (state.md + touched-files.md + evidence.md carry forward); /rite-handoff if away > a few hours.
```

**All slices built (build phase complete — say it):**
```
── rite-build ──────────────────────────────
Slices 5/5  ██████████  ✅ ALL BUILT
Flow   spec ✓ define ✓ build ◉ prove ○ polish ○ review ○ seal ○ ship ○
Built     slice 5 — error-states   (by devrites-slice-wright | inline fallback)
Acceptance ✓ met · Tests <cmd → pass> · Browser <summary | n/a> · Drift <none | handled>
✅ Feature implemented — every slice built. Build phase done.
Next  ▸ /rite-prove  (prove the completed feature)
↻ Hygiene  /clear (state.md + evidence.md captured); /rite-handoff if away > a few hours.
```

Keep fact lines terse — one `key value` per fact, `·` between, no prose. The meter, the
`✅ ALL BUILT` marker, and the ribbon carry the progress; don't restate them in words.

**DO NOT continue to the next slice automatically** — even at `✅ ALL BUILT`, `/rite-prove` is the user's call.
