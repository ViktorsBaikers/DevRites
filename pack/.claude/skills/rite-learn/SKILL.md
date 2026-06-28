---
name: rite-learn
description: Review the auto-captured learning ledger and promote recurring lessons to project rules or principles — the human-gated half of the cross-feature learning loop. Not for the install (`/rite-doctor`), feature status (`/rite-status`), onboarding (`/rite-adopt`), or a diff review (`/rite-review`).
argument-hint: "[--mine | \"<lesson to record>\"]"
user-invocable: true
disable-model-invocation: true
---

# /rite-learn — the cross-feature learning loop

Recurring corrections, dismissed review findings, and dead-ends are durable signal. **Capture is
automatic** — `/rite-seal` appends them to `.devrites/learnings.md` on every GO (step 9a), and the
review skills load that ledger **before** a fan-out, so a dismissed-finding class stops being
re-flagged without anyone running a command. The system learns as it ships.

`/rite-learn` is the periodic **review + promote** pass on that auto-populated ledger: cluster the
signal across features and decide which recurring lessons graduate into a project rule. **Propose,
never impose** — promotion is the human's call, which is why it stays a deliberate command rather
than firing on its own.

Read-mostly. It reads the auto-populated ledger + the archive; it writes only `.devrites/learnings.md`
(consolidation, via `learnings.sh`) and drafts proposed rule/ledger edits for the user to confirm —
it never edits source or rule files on its own.

## Modes

- `/rite-learn` or `/rite-learn --mine` — **mine + propose** (the default).
- `/rite-learn "<lesson>"` — **record one lesson** directly to the ledger and stop.

## Workflow (mine + propose)

1. **Gather the signal.** Start from the **auto-populated ledger** `.devrites/learnings.md` (seal
   appends to it on every GO — step 9a), then run `learnings.sh mine` over the archive (resolve via
   the standard 3-path snippet; it clusters repeated finding / decision / drift phrases across shipped
   features) to surface cross-feature patterns the per-feature entries don't show on their own:
   ```bash
   L=.claude/skills/devrites-lib/scripts/learnings.sh
   [ -f "$L" ] || L="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/learnings.sh"
   [ -f "$L" ] || L=pack/.claude/skills/devrites-lib/scripts/learnings.sh
   [ -f "$L" ] && bash "$L" mine || echo "(learnings.sh unavailable — scan .devrites/archive/*/{decisions,drift,review}.md by hand)"
   ```
2. **Cluster + name.** Group the recurring corrections into candidate lessons. A candidate needs
   **≥2 occurrences across distinct features** — one-offs are noise, not a pattern. Name the
   pattern in one specific sentence (the specificity rule from `prose-style.md` applies: a lesson
   you could swap onto any project says nothing).
3. **Classify each candidate** into its durable home:
   - **project rule** — a craft/standard that belongs in a `.claude/rules/*` file or `CLAUDE.md`.
   - **project principle** — a recurring correction that is really a *non-negotiable invariant*
     (not just an idiom or a craft standard): graduate it to `.devrites/principles.md`
     ([`principles.md`](../../rules/principles.md)). This is the **trusted, gating** layer — higher
     stakes than a rule — so `/rite-learn` **drafts** the principle + a dated Governance entry for
     the human to confirm; it never writes a principle silently.
   - **conventions-ledger entry** — a proven project idiom for `.devrites/conventions.md`.
   - **dismissed-finding class** — a pattern reviewers keep flagging that is *intentional here*;
     recording it stops the recurring false positive (`learnings.md`, loaded pre-fan-out).
   - **drop** — not durable; let it go.
4. **Propose, don't impose.** Present the candidates with their evidence (which features, how many
   times, the proposed home) via `AskUserQuestion` — the human picks which to promote. Never
   promote a lesson to a rule silently; an unproven "lesson" hardened into a rule is its own slop.
5. **Record the accepted.** For each the user accepts, append it with `learnings.sh add <slug>
   "<lesson>" <tag>` (`tag` ∈ `rule | convention | dismiss`). If the user approves a **rule**,
   **principle**, or **ledger** promotion, draft the exact edit and let the user confirm it through
   the normal flow — `/rite-learn` writes the ledger, not the rule or principle files. A
   **principle** promotion is the highest-stakes of these: draft the `.devrites/principles.md` entry
   **plus its dated Governance line**, and let the human confirm before it lands — a principle is a
   gate, so it is amended deliberately, never auto-written. Then `touch .devrites/.learnings-reviewed`
   so the SessionStart learnings nudge snoozes until new signal accumulates.

## How the ledger is used

The review skills (`/rite-review`, `/rite-seal`) read `.devrites/learnings.md` before they fan
out: a **dismissed-finding class** suppresses the recurring false positive; a **proven
convention** raises the bar. The ledger is an **untrusted prior** — a fresh observation of the
live code always overrides a ledger entry (see `.claude/rules/security.md`). Confidence in a
recorded lesson never raises its authority.

A **project principle** (`.devrites/principles.md`) is the opposite layer — prescriptive,
trusted, and **gating**. Promoting a lesson there is a deliberate amendment, not a prior the next
fan-out can override: a violation becomes a blocking finding, not a suppressed false positive.
That asymmetry is why principle promotion is human-confirmed and dated, never auto-written.

## Gotchas
- Evidence first: a lesson without ≥2 real occurrences is speculation. Cite the features.
- Don't pad the ledger. Five real lessons that change behaviour beat thirty rubber-stamped rows.
- The ledger records *what was learned*; it does not re-open shipped features or edit their archives.

## Output
```
Mined: <n> features · <n> recurring patterns
Candidates: <n> (rule: a · convention: b · dismiss: c · dropped: d)
Promoted: <n> → .devrites/learnings.md  (+ <n> rule/ledger edits drafted for your confirm)

Session hygiene: /clear (recommended) — the ledger is on disk; the proposals are listed above.
Resume next session with: <the single next command, e.g. the drafted rule edit, or /rite-status>
```
