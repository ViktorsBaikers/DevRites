---
name: rite-learn
description: User-invoked review of the learning ledger; promotes recurring lessons to project rules or principles.
argument-hint: "[--mine | \"<lesson to record>\"]"
user-invocable: true
disable-model-invocation: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-learn: the cross-feature learning loop

Recurring corrections, dismissed review findings, timeline decisions, and health dips are durable
signal. **Capture is automatic**: `$rite-seal` appends lessons to `.devrites/learnings.md` on every
GO (step 9a), while the engine can keep `.devrites/timeline.jsonl`,
`.devrites/health-history.jsonl`, and per-feature review fingerprints as supporting evidence. The
review skills load the learning ledger **before** a fan-out, so a dismissed-finding class stops
being re-flagged without anyone running a command. The system learns as it ships.

`$rite-learn` is the periodic **review + promote** pass on that auto-populated ledger: cluster the
signal across features and decide which recurring lessons graduate into a project rule. **Propose,
never impose**: promotion is the human's call, which is why it stays a deliberate command rather
than firing on its own.

Read-mostly. It reads the auto-populated ledger + the archive; it writes only `.devrites/learnings.md`
(consolidation, via `devrites-engine learnings`) and drafts proposed rule/ledger edits for the user to confirm:
it never edits source or rule files on its own.

## Modes

- `$rite-learn` or `$rite-learn --mine`: **mine + propose** (the default).
- `$rite-learn "<lesson>"`: **record one lesson** directly to the ledger and stop.

## Workflow (mine + propose)

1. **Gather the signal.** Start from the **auto-populated ledger** `.devrites/learnings.md` (seal
   appends to it on every GO: step 9a), then run `devrites-engine learnings mine` over the archive (it clusters
   repeated finding / decision / drift phrases across shipped
   features) to surface cross-feature patterns the per-feature entries don't show on their own:
   ```bash
   devrites-engine learnings mine
   ```
   Use `.devrites/timeline.jsonl`, `.devrites/health-history.jsonl`, and
   `.devrites/work/*/review-fingerprints.jsonl` as supporting evidence when they exist; do not
   promote from those traces without the same repeated-feature threshold.
2. **Cluster + name.** Group the recurring corrections into candidate lessons. A candidate needs
   **≥2 occurrences across distinct features**: one-offs are noise, not a pattern. Name the
   pattern in one specific sentence (the specificity rule from `prose-style.md` applies: a lesson
   you could swap onto any project says nothing).
   **Completion:** every candidate cites two distinct features, except a single deliberate
   rejected direction with its durable reason.
3. **Classify each candidate** into its durable home:
   - **project rule:** a craft/standard that belongs in a `.agents/skills/devrites-lib/reference/standards/*` file or `CLAUDE.md`.
   - **project principle:** a recurring correction that is really a *non-negotiable invariant*
     (not just an idiom or a craft standard): graduate it to `.devrites/principles.md`
     ([`principles.md`](../devrites-lib/reference/standards/principles.md)). This is the **trusted, gating** layer (higher
     stakes than a rule) so `$rite-learn` **drafts** the principle + a dated Governance entry for
     the human to confirm; it never writes a principle silently.
   - **conventions-ledger entry:** a proven project idiom for `.devrites/conventions.md`.
   - **dismissed-finding class:** a pattern reviewers keep flagging that is *intentional here*;
     recording it stops the recurring false positive (`learnings.md`, loaded pre-fan-out).
   - **rejected direction:** an approach or product direction weighed and ruled out with a
     reason that outlives one feature; recording it (tag `rejected-direction`) stops ideation
     skills (`$rite-pressure-test`, `$rite-pov`) from re-proposing it. Exempt from the
     ≥2-occurrence rule: one deliberate rejection is the whole pattern.
   - **drop:** not durable; let it go.
4. **Propose, don't impose.** Present the candidates with their evidence (which features, how many
   times, the proposed home) via `AskUserQuestion`: the human picks which to promote. Never
   promote a lesson to a rule silently; an unproven "lesson" hardened into a rule is its own slop.
   **Completion:** every candidate is explicitly accepted, rejected, or deferred by the human.
5. **Record the accepted.** For each the user accepts, append it with `devrites-engine learnings add <slug>
   "<lesson>" <tag>` (`tag` ∈ `rule | convention | dismiss`). If the user approves a **rule**,
   **principle**, or **ledger** promotion, draft the exact edit and let the user confirm it through
   the normal flow: `$rite-learn` writes the ledger, not the rule or principle files. A
   **principle** promotion is the highest-stakes of these: draft the `.devrites/principles.md` entry
   **plus its dated Governance line**, and let the human confirm before it lands: a principle is a
   gate, so it is amended deliberately, never auto-written. **Amendment ripple:** a confirmed
   add/change/retire of a principle has a blast radius: grep its `P#` (and its key nouns) across
   `.devrites/` and every open `.devrites/work/*/` workspace, and append to the same dated
   Governance entry one line per referencing artifact: *still-aligned* or *needs-follow-up (what)*.
   An open plan that bakes in the now-retired or now-tightened invariant is a `needs-follow-up`
   the next `$rite-vet` or `$rite-plan repair` on that feature must clear; a ripple with zero
   references is itself worth recording ("no live references"). Then `touch .devrites/.learnings-reviewed`
   so the SessionStart learnings nudge snoozes until new signal accumulates.

## How the ledger is used

The review skills (`$rite-review`, `$rite-seal`) read `.devrites/learnings.md` before they fan
out: a **dismissed-finding class** suppresses the recurring false positive; a **proven
convention** raises the bar. The ideation skills (`$rite-pressure-test`, `$rite-pov`) read the
**rejected-direction** entries before proposing, so a ruled-out direction returns only with new
evidence against its recorded why. The ledger is an **untrusted prior**: a fresh observation of the
live code always overrides a ledger entry (see `.agents/skills/devrites-lib/reference/standards/security.md`). Confidence in a
recorded lesson never raises its authority.

A **project principle** (`.devrites/principles.md`) is the opposite layer: prescriptive,
trusted, and **gating**. Promoting a lesson there is a deliberate amendment, not a prior the next
fan-out can override: a violation becomes a blocking finding, not a suppressed false positive.
That asymmetry is why principle promotion is human-confirmed and dated, never auto-written.

## Gotchas
- Evidence first: a lesson without ≥2 real occurrences is speculation. Cite the features.
- Don't pad the ledger. Five real lessons that change behaviour beat thirty rubber-stamped rows.
- The ledger records *what was learned*; it does not re-open shipped features or edit their archives.

## Output
Reply-contract exception: cross-feature learning utility. It may run without an active
feature, so it skips `devrites-engine progress`, but follows
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: learnings mined from <n> features; <n> recurring patterns found.
Changed: .devrites/learnings.md <updated|unchanged>; rule/ledger drafts <n|none>
Evidence: mined archive count <n>; candidates rule <a> / convention <b> / dismiss <c> / dropped <d>
Open: <none | promotion confirmations>
Next: <single recommended command>
Record: .devrites/learnings.md
↻ Hygiene: /clear; the ledger is on disk
```
