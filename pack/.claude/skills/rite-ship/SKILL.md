---
name: rite-ship
description: Ship the sealed feature and close the task — render type-GO, run the irreversible git commit/push/tag (or open the PR), write ship.md, then archive the workspace and clear .devrites/ACTIVE. Use when the user says "ship it", "ship this", "push it out", "close the task", "finish and archive", or right after /rite-seal returns GO. Not for the GO/NO-GO decision itself (use /rite-seal) or an unsealed feature.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-ship — ship + close the task

The final phase. `/rite-seal` **decides** GO/NO-GO; `/rite-ship` **executes** the
irreversible git actions and **closes** the feature. **Read the active workspace
first**; if none, tell the user to run `/rite-spec <feature>`.

Refuses to ship unless `seal.md` records a **GO** verdict.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. Then pull on demand:
- `git-workflow.md` — Conventional Commits, atomic commits, the never-commit list.
- `afk-hitl.md` — type-GO is the irreversible-action gate.

## Operating rules
- **Seal GO is a precondition.** No GO in `seal.md` → stop, point at `/rite-seal`.
- **Evidence must be fresh.** If any file in `touched-files.md` changed after
  `evidence.md` was written, the proof is stale → stop, point at `/rite-prove`. Enforced
  deterministically by `evidence-fresh.sh` in step 1 (exit 3 = STALE), not by eyeballing
  mtimes (see `.claude/rules/development-workflow.md`).
- **type-GO before anything irreversible.** Render the prompt verbatim and wait for
  the literal `GO`. Last safety net — render it every time, even under auto-trigger.
- **Never delete the audit trail.** Closing *archives* the workspace; it never erases
  the `.md` files.

## Workflow
1. **Run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
   Then read `seal.md`, `state.md`, `spec.md`, `touched-files.md`, `evidence.md`. Confirm
   the verdict is **GO**, then run the deterministic evidence-freshness gate rather than
   eyeballing mtimes (mirrors `/rite-seal`):
   ```bash
   E=.claude/skills/devrites-lib/scripts/evidence-fresh.sh
   [ -f "$E" ] || E="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/evidence-fresh.sh"
   [ -f "$E" ] || E=pack/.claude/skills/devrites-lib/scripts/evidence-fresh.sh
   [ -f "$E" ] && { bash "$E"; echo "evidence-fresh rc=$?"; } || echo "(evidence-fresh gate unavailable — compare mtimes by hand)"
   ```
   **Exit 3 → STALE proof: STOP**, point at `/rite-prove` (a polish/review edit made after
   `/rite-prove` invalidates the proof). Not GO → stop with the single resume command.
2. Build the git plan from `git-workflow.md` + the project's own convention: the
   Conventional-Commit message(s), the target branch, and whether a tag / PR applies.
   Scope the commit to `touched-files.md`; never stage secrets or out-of-scope files.
3. **Render the type-GO prompt** ([reference/git-ship.md](reference/git-ship.md)) and
   wait. Only the literal `GO` proceeds; anything else cancels — record the cancel in
   `ship.md` and stop (do not retry without the user asking).
4. On `GO`: run the git ladder — commit → push → tag / PR as applicable
   ([reference/git-ship.md](reference/git-ship.md)). Capture the commit SHA(s),
   branch, and tag/PR URL.
5. Write `ship.md` ([reference/ship-template.md](reference/ship-template.md)): what
   shipped, SHA(s), branch, tag/PR, acceptance summary (n/total), link to `seal.md`,
   follow-ups.
6. **Close the task** ([reference/close-out.md](reference/close-out.md)): set
   `state.md` phase `done`, then run
   `bash .claude/skills/devrites-lib/scripts/close-out.sh <slug>` to archive
   `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` and clear `.devrites/ACTIVE`.
   Every `.md` is preserved in the archive.

> **Mid-flight discipline.** When tempted to ship without a GO seal, skip the type-GO,
> stage files outside `touched-files.md`, or delete the workspace instead of archiving
> it — stop. See [`anti-patterns`](reference/anti-patterns.md); the gate exists for the
> failure mode the ask misses.

## Output

**Footer first** — render the flow ribbon by running the progress footer (`progress.sh`, resolved like the step-0 preamble — canonical snippet in `devrites-lib/SKILL.md`); at ship it reads `ship ✓` across the spine. Keep the fact lines below it terse (`key value · key value`). Then:
```
Shipped: <feature>
Commit:  <sha> on <branch>    Tag/PR: <ref | none>
Acceptance: <n/total> proven
Archived: .devrites/archive/<slug>/    ·    ACTIVE cleared
ship.md:  .devrites/archive/<slug>/ship.md
```
If the user declined type-GO: state that nothing shipped, the seal still reads GO, and
the resume command (`/rite-ship`).

End with `↻ Hygiene: /clear` — the feature is closed; start the next with
`/rite-spec <feature>`. See `.claude/rules/context-hygiene.md`.
