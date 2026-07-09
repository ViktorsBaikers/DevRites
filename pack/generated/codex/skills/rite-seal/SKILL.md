---
name: rite-seal
description: Decide GO / NO-GO on the active feature. Use when the user says "seal this" or "GO / NO-GO". Hands off to $rite-ship for the actual commit/push/close. Not for the irreversible ship itself (use $rite-ship), inline review, or unpolished features.
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


# $rite-seal — GO / NO-GO

The decision gate before shipping. **Read the active workspace first**; if none, tell
the user to run `$rite-spec <feature>`. Produces `seal.md` with a clear verdict.
`$rite-seal` **decides**; the irreversible git commit/push/tag and the task close-out
live in `$rite-ship`, which refuses to run without a GO recorded here.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. The other rule files load on demand —
pull these via `Read` before sealing:
- `agents.md` — review-subagent fan-out at seal.
- `code-review.md` — severity labels (Critical / Important / Suggestion / Nit / FYI).
- `principles.md` — declared project invariants (`.devrites/principles.md`) are a pass/fail gate; a diff that violates one with no recorded, human-approved exception is a NO-GO.
- `documentation.md` — record decisions in `decisions.md` before sealing.
- `observability.md` — a runtime surface that ships blind is an Important finding.
- `deprecation.md` — when the diff removes / migrates code, API, or data (read with the
  risk-and-rollback step below).
- `definition-of-done.md` — standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


## Operating rules
- Evidence over confidence — a criterion is met only if evidence proves it.
- Spec Drift Guard applies: unresolved drift is a NO-GO.
- **Honest verdict.** DO NOT round a NO-GO up to GO to be agreeable.

## Severity gate (the ship/no-ship rule)

Read `review.md` and the latest reviewer outputs.

| State | Gate result |
|---|---|
| `Critical == 0` and `Important == 0` and acceptance proven and drift resolved | **GO** (proceed) |
| `Critical == 0` and `Important > 0` and acceptance proven and drift resolved | Render interactive prompt: *"`Important > 0` open. Proceed to seal? [y/N]"*. Default **N**. If the user types `y`, GO; otherwise NO-GO with the open Important findings listed as blockers-by-policy. |
| `Critical > 0` | **NO-GO**, no exceptions. List every Critical with `file:line` and fix direction. |
| Any acceptance criterion unproven, or any bespoke `Prohibitions (must-NOT)` `resolved/test` row lacks linked proof | **NO-GO**, list the unproven criteria/prohibitions. |
| Fan-out roster incomplete — `devrites-engine footprint roster` rc=3 (a reviewer neither dispatched nor skip-recorded) | **NO-GO** on the review's *completeness*, not a verdict on the diff: dispatch the missing reviewer or record a justified skip, then re-seal. Same standing as an unproven criterion. |
| Silent review axis — `devrites-engine review-integrity` rc=1 (an adversarial axis in `review.md` reported nothing and justified nothing) | **Important** — a suspected rubber-stamp, on the review's *completeness* not the diff. Re-run that axis or record its `No-findings:` justification (`code-review.md` § Zero findings is suspicious), then re-seal. |
| Visual Verdict `FAIL` on an acceptance-mapped UI criterion (`browser-evidence.md`) | **NO-GO** — an unmet acceptance criterion. A declared-state `FAIL` is Important (the `Important > 0` row). UI build with a `design-brief.md` but no Visual Verdict → Important evidence gap. No brief → not applicable. |
| Diff violates a declared project principle (`.devrites/principles.md`) with no recorded, human-approved exception | **NO-GO**, list each violated principle with `file:line`. Same standing as an unproven criterion (absent / empty file → none declared → not a blocker). |
| Unresolved drift in `drift.md` | **NO-GO**, route through `$rite-plan` first. |
| Any `questions.md` entry with `gate: validating` and `status: open` | **NO-GO** regardless of behavior impact — an open validating gate is merge-blocking by definition. A slice marked `built (pending review)` is not done. |
| A stood decision (boundary / data-model / auth / public-API / migration / branching) in the `decisions.md` `## Decisions stood` ledger with no recorded `devrites-doubt` verdict — `doubt: MISSING` / `doubt-coverage` rc=3 | **Important** — **NO-GO** when the undoubted decision is irreversible-risk (auth / public-API / migration), the same standing as an unproven acceptance criterion. Severity rides the unverified **decision**, not the exit code: `doubt-coverage` rc=1 (zero doubt dispatched) is a **prompt to verify**, not itself a finding — confirm against the ledger, where every-slice-trivial (`- none`) passes and a skipped triggering decision is the finding. |

## Workflow

Run the full execution contract in
[`reference/phase-contract.md`](reference/phase-contract.md). It is not optional:
it contains the orientation, evidence, risk, reviewer fan-out, roster,
review-integrity, verdict, convention, and learning steps for this gate.

The operating rules and severity gate above plus the phase contract together
define `$rite-seal`. If any supporting reference appears to conflict with this
root file, follow the stricter instruction.

## On GO → hand off to $rite-ship

`$rite-seal` makes the **decision** and stops. It does **not** run `git commit`,
`git push`, `git tag`, publish, or deploy — those moved to `$rite-ship`, which renders
the type-GO prompt and refuses to run without a GO recorded here. Keeping the decision
and the irreversible action as two separately-auditable steps is the point: a GO seal
is a verdict, not an authorization to push.

On **GO**: write `seal.md`, set `state.md` `Next step: $rite-ship`, and tell the user
the feature is cleared to ship. Do **not** set phase `done` — `$rite-ship` marks done
after the task is shipped and archived. The `Important > 0` interactive y/N earlier in
the gate is the one off-ramp seal still owns; the type-GO off-ramp now lives in ship.
For a **UI feature**, note in the hand-off that `$rite-ship` offers an optional
**design-memory** rollup — persist this feature's proven design language into a project
`DESIGN.md` so later features inherit it (`../rite-ship/reference/design-memory.md`).

## Output

Use the full output contract in [`reference/output.md`](reference/output.md).
It preserves the progress-first GO / NO-GO shape, uses the shared completion
reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)),
and keeps the boundary that `$rite-seal` decides only; `$rite-ship` executes.
