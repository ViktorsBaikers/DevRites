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
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-ship — ship + close the task

The final phase. `$rite-seal` **decides** GO/NO-GO; `$rite-ship` **executes** the
irreversible git actions and **closes** the feature. **Read the active workspace
first**; if none, tell the user to run `$rite-spec <feature>`.

Refuses to ship unless `seal.md` records a **GO** verdict.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. Then pull on demand:
- `git-workflow.md` — Conventional Commits, atomic commits, the never-commit list.
- `afk-hitl.md` — type-GO is the irreversible-action gate.

## Operating rules
- **Seal GO is a precondition.** No GO in `seal.md` → stop, point at `$rite-seal`.
- **Evidence must be fresh.** If any file in `touched-files.md` changed after
  `evidence.md` was written, the proof is stale → stop, point at `$rite-prove`. Enforced
  deterministically by `devrites-engine evidence-fresh` in step 1 (exit 3 = STALE), not by eyeballing
  mtimes (see `.agents/skills/devrites-lib/reference/standards/development-workflow.md`).
- **type-GO before anything irreversible.** Render the prompt verbatim and wait for
  the literal `GO`. Last safety net — render it every time, even under auto-trigger.
- **Never delete the audit trail.** Closing *archives* the workspace; it never erases
  the `.md` files.

## Workflow
1. **Run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
   Then read `seal.md`, `state.md`, `spec.md`, `touched-files.md`, `evidence.md`, and
   `design-brief.md` (if the feature is UI — the design-memory rollup in step 2a reads it).
   Confirm the verdict is **GO**, then run the deterministic evidence-freshness gate rather than
   eyeballing mtimes (mirrors `$rite-seal`):
   ```bash
   devrites-engine evidence-fresh; echo "evidence-fresh rc=$?"
   ```
   **Exit 3 → STALE proof: STOP**, point at `$rite-prove` (a polish/review edit made after
   `$rite-prove` invalidates the proof). Not GO → stop with the single resume command.
1a. **Health re-check (advisory).** Run the DevRites doctor before the irreversible ladder —
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
2a. **Design memory (optional, UI features only).** If the feature shipped UI, offer to roll
   its *proven* design language up into a project-level `DESIGN.md` so the next feature
   inherits the system instead of re-discovering it
   ([reference/design-memory.md](reference/design-memory.md)). **Opt-in and confirmed** —
   present the option set (default **skip**; persisting beyond feature scope is the user's
   call), and on yes append `DESIGN.md` to `touched-files.md` so it ships *in this commit*.
   Skip silently when there's no UI. Record the outcome in `ship.md`.
2b. **Capability ledger sync.** Fold this feature's *proven* behavior into the living
   `.devrites/specs/<capability>/spec.md` ledger so the next feature starts from the contract
   instead of re-deriving it from code ([reference/ledger.md](reference/ledger.md)). Preview first —
   `devrites-engine ledger diff .devrites/work/<slug>` — then **opt-in, confirmed** (default
   **sync**; a shipped feature's proven behavior belongs in the ledger — the escape hatch is skip,
   for an internal-only change with no capability contract). On yes: run
   `devrites-engine ledger sync .devrites/work/<slug>`, append each written
   `.devrites/specs/<capability>/spec.md` to `touched-files.md` so it ships *in this commit*, and
   record the synced capabilities in `ship.md`. Skip silently when the feature declares no
   requirements (a pure refactor / chore). The fold is gated on the GO + evidence-fresh confirmed
   in step 1, so the ledger only ever records proven truth.
2c. **Credential guard (blocking for HIGH).** Before the irreversible type-GO prompt, scan staged
   and touched files plus any PR body draft. HIGH blocks ship and tells the user to rotate/redact;
   MEDIUM asks for confirmation and records the exception in `ship.md`; LOW is FYI.
   ```bash
   devrites-engine secret-scan --staged; echo "secret-scan rc=$?"
   ```
   **rc=3 → STOP**: do not type-GO, commit, push, or archive.
3. **Render the type-GO prompt** ([reference/git-ship.md](reference/git-ship.md)) and
   wait. Only the literal `GO` proceeds; anything else cancels — record the cancel in
   `ship.md` and stop (do not retry without the user asking).
4. On `GO`: run the git ladder — commit → push → tag / PR as applicable
   ([reference/git-ship.md](reference/git-ship.md)). Capture the commit SHA(s),
   branch, and tag/PR URL.
4a. **When opening a PR, render a structured body** — not just the commit message:
   **Summary** (what shipped + acceptance n/total) · **Risk & rollback** (the migration /
   destructive / auth touches + how to revert, from `seal.md`'s risk scan — when this ship drives a
   *live staged rollout* the agent owns, follow [reference/rollout.md](reference/rollout.md) for the
   rollout thresholds, rollback-time budgets, and first-hour runbook; skip it when CI deploys on merge)
   · **What to scrutinize**
   (point reviewers at the highest-blast-radius hunks) · **Evidence** (a condensed `evidence.md` +
   the seal's reconciled reviewer-verdict digest, linking the full `.devrites/archive/<slug>/`
   bundle). **Delete any N/A section** — an empty Risk block is noise.
4b. **Promote architecturally-significant decisions to ADRs.** For each `decisions.md` ADR that
   records a *durable* architecture / interface choice (not a slice-local call), append it to a
   persistent `docs/adr/ADR-NNN.md` — Nygard shape (Context · Decision · Status `accepted` ·
   Consequences), **append-only**, never rewritten; supersede with a new ADR that links the old.
   The per-feature `decisions.md` is archived with the workspace; the ADR keeps the load-bearing
   *why* discoverable in the repo. Skip if the project keeps no `docs/adr/` and the user doesn't want one.
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
6a. **Cross-feature retro (automatic, cadence-gated, advisory).** The just-shipped feature is now in
   the archive, so this is where the **cross-feature** learning loop closes on its own — the synthesis
   that otherwise waits for a human to run `$rite-learn`. Run the cheap cadence gate first; it stays
   silent unless a finding/drift class recurs across **>=2 shipped features** with new signal since the
   last review (so it never fires on an early or one-off ship):
   ```bash
   devrites-engine learnings nudge
   ```
   **If the nudge emits** (a recurring pattern crossed the threshold), dispatch the read-only
   `devrites-retrospector` (`.codex/agents/`) over `.devrites/archive/` for the cross-feature
   synthesis. Persist its digest to `.devrites/retro.md` (append a dated entry — the project-level
   retro ledger, never rewritten) and surface the **graduation candidates** with a one-line pointer
   to `$rite-learn`, which is where the human confirms a promotion to a rule / principle / convention.
   **Propose, never impose:** retro **drafts**; it never auto-writes a rule or principle (a principle
   is a gate, amended deliberately and dated — `principles.md` governance), and it never blocks the
   ship, which has already happened. If the nudge is silent, skip — no retro this close. Then `touch
   .devrites/.learnings-reviewed` only when the human acts on it via `$rite-learn`, not here.

> **Mid-flight discipline.** When tempted to ship without a GO seal, skip the type-GO,
> stage files outside `touched-files.md`, or delete the workspace instead of archiving
> it — stop. See [`anti-patterns`](reference/anti-patterns.md); the gate exists for the
> failure mode the ask misses.

## Output

**Progress first** — run `devrites-engine progress`, then use the Shipped typed template from
the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Final shipped shape:
```
Shipped: <feature>
Commit: <sha> on <branch>
Tag/PR: <value | none>
Acceptance: <n>/<total> proven
Archived: .devrites/archive/<slug>/ · ACTIVE cleared
Record: .devrites/archive/<slug>/ship.md
↻ Hygiene: /clear
```
If the user declined type-GO: state that nothing shipped, the seal still reads GO, and
the resume command (`$rite-ship`).
