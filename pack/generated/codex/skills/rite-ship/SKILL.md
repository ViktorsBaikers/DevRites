---
name: rite-ship
description: Ship the sealed feature: commit, push, tag, archive, close. Use when the user says "ship it", "push the branch", "tag the release", or "close the task" after $rite-seal GO. Not for GO/NO-GO readiness.
argument-hint: "[feature-slug]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here: Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review**: an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-ship: ship + close the task

The final phase. `$rite-seal` **decides** GO/NO-GO; `$rite-ship` **executes** the
irreversible git actions and **closes** the feature. **Read the active workspace
first**; if none, tell the user to run `$rite-spec <feature>`.

Refuses to ship unless `seal.md` records a **GO** verdict.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. Then pull on demand:
- `git-workflow.md`: Conventional Commits, atomic commits, the never-commit list.
- `afk-hitl.md`: type-GO is the irreversible-action gate.
- `definition-of-done.md`: final acceptance, evidence, drift, rollback, and documentation bar.
- [`release/ship-checklist.md`](../devrites-lib/reference/standards/release/ship-checklist.md): compact final ship and archive pass/fail sweep.

## Operating rules
- **Seal GO is a precondition.** No GO in `seal.md` → stop, point at `$rite-seal`.
- **Evidence must be fresh.** If any file in `touched-files.md` changed after
  `evidence.md` was written, the proof is stale → stop, point at `$rite-prove`. Enforced
  deterministically by `devrites-engine evidence-fresh` in step 1 (exit 3 = STALE), not by eyeballing
  mtimes (see `.agents/skills/devrites-lib/reference/standards/development-workflow.md`).
- **type-GO before anything irreversible.** Render the prompt verbatim and wait for
  the literal `GO`. Last safety net: render it every time, even under auto-trigger.
- **Never delete the audit trail.** Closing *archives* the workspace; it never erases
  the `.md` files.

## Workflow
1. **Orient.** Run `devrites-engine preamble` for deterministic workspace orientation.
   Then read `seal.md`, `state.md`, `spec.md`, `touched-files.md`, `evidence.md`, and
   `design-brief.md` (if the feature is UI: the design-memory rollup in step 2a reads it).
   Confirm the verdict is **GO**, then run the deterministic evidence-freshness gate rather than
   eyeballing mtimes (mirrors `$rite-seal`):
   ```bash
   devrites-engine evidence-fresh; echo "evidence-fresh rc=$?"
   ```
   **Exit 3 → STALE proof: STOP**, point at `$rite-prove` (a polish/review edit made after
   `$rite-prove` invalidates the proof). Not GO → stop with the single resume command.
1a. **Health re-check (advisory).** Run the DevRites doctor before the irreversible ladder:
   a stale `ACTIVE` or corrupt workspace here risks shipping or closing the wrong feature.
   Advisory: surface issues, don't block.
   ```bash
   devrites-engine doctor; echo "doctor rc=$?"
   ```
   If it flags the active feature, confirm you're shipping the intended slug before proceeding.
2. Build the git plan from `git-workflow.md` + the project's own convention: the
   Conventional-Commit message(s), commit trailers for non-trivial work, the target branch, and
   whether a tag / PR applies. Derive trailers from `decisions.md`, `assumptions.md`, `seal.md`,
   and `evidence.md`: `Constraint`, `Rejected`, `Confidence`, `Scope-risk`, `Not-tested` when
   present; skip trailers for trivial typo/formatting commits. Scope the commit to
   `touched-files.md`; never stage secrets or out-of-scope files.
   **Completion:** message, trailers, target branch, tag/PR choice, and exact staged scope are explicit.
2a. **Design memory (UI only).** Follow
   [`reference/design-memory.md`](reference/design-memory.md). Completion: the confirmed
   outcome is recorded, and any `DESIGN.md` edit is in `touched-files.md`; non-UI skips.
2b. **Capability ledger sync.** Follow [`reference/ledger.md`](reference/ledger.md).
   Completion: proven capability deltas are previewed, the confirmed outcome is recorded,
   and every synced ledger file is in `touched-files.md`.
2c. **Credential guard (blocking for HIGH).** Before the irreversible type-GO prompt, scan staged
   and touched files plus any PR body draft. HIGH blocks ship and tells the user to rotate/redact;
   MEDIUM asks for confirmation and records the exception in `ship.md`; LOW is FYI.
   ```bash
   devrites-engine secret-scan --staged; echo "secret-scan rc=$?"
   ```
   **rc=3 → STOP**: do not type-GO, commit, push, or archive.
3. **Render the type-GO prompt** ([reference/git-ship.md](reference/git-ship.md)) and
   wait. Only the literal `GO` proceeds; anything else cancels: record the cancel in
   `ship.md` and stop (do not retry without the user asking).
4. On `GO`: run the git ladder: commit → push → tag / PR as applicable
   ([reference/git-ship.md](reference/git-ship.md)). Capture the commit SHA(s),
   branch, and tag/PR URL.
4a. **PR branch only.** Render the structured body from
   [`reference/git-ship.md`](reference/git-ship.md#pull-request-body); omit empty sections.
4b. **Durable architecture decisions only.** Apply
   [`reference/adr-promotion.md`](reference/adr-promotion.md); slice-local decisions stay in
   the archived workspace.
5. Write `ship.md` ([reference/ship-template.md](reference/ship-template.md)): what
   shipped, SHA(s), branch, tag/PR, acceptance summary (n/total), link to `seal.md`,
   follow-ups.
6. **Close the task** ([reference/close-out.md](reference/close-out.md)): set
   `state.md` phase `done`, then run
   `devrites-engine close-out <slug>` to archive
   `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` and clear `.devrites/ACTIVE`.
   Every `.md` is preserved in the archive. Then refresh any managed project context block so
   `AGENTS.md` / `CLAUDE.md` no longer advertise a closed active workspace:
   ```bash
   devrites-engine context sync || true
   ```
6a. **Cross-feature retro.** Run the cadence-gated advisory branch in
   [`reference/close-out.md`](reference/close-out.md#cross-feature-retro). Completion: a
   silent nudge does nothing; a real recurrence is appended to `.devrites/retro.md` and
   surfaced to `$rite-learn`, never auto-promoted.

> **Mid-flight discipline.** When tempted to ship without a GO seal, skip the type-GO,
> stage files outside `touched-files.md`, or delete the workspace instead of archiving
> it: stop. See [`anti-patterns`](reference/anti-patterns.md); the gate exists for the
> failure mode the ask misses.

## Output

**Progress first**: run `devrites-engine progress`, then use the Shipped typed template from
the shared completion reply contract
([`reply-contract.md` § Shipped](../devrites-lib/reference/reply-contract.md#shipped)).
If the user declined type-GO: state that nothing shipped, the seal still reads GO, and
the resume command (`$rite-ship`).
