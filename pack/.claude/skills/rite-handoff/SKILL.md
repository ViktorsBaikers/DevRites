---
name: rite-handoff
description: Compact the current chat into a handoff doc a fresh agent can pick up cold — syncs chat-only context into the canonical `.devrites/` files, references artifacts by path. Use when the user says "handoff", "prep for tomorrow", "I'm leaving for a while", "summarize for the next agent". Not for routine `/clear` between phases.
user-invocable: true
argument-hint: "[what the next session will focus on]"
---

# /rite-handoff — chat-only context, into a fresh-agent doc

DevRites' workspace `.devrites/work/<slug>/` already captures everything that *should*
persist (spec, plan, tasks, decisions, evidence, drift, review). This skill captures
what the **chat** is holding that is **not** in the workspace, so a fresh agent — or
the same user after `/clear` — can pick the work up without re-reading the transcript.

Read `.claude/rules/core.md` first — its "Persistence before stopping" discipline is
exactly what this skill executes. The other rule files load on demand.

## Where to write

- **Active feature exists** → `.devrites/work/<slug>/handoff.md` (overwrites the previous
  handoff; the workspace is the canonical home for this slug).
- **No active feature** → OS temp dir (`$TMPDIR` / `/tmp` / `%TEMP%`) as
  `rite-handoff-<ISO-timestamp>.md`. Print the absolute path after writing.

## Before you write — sync, don't duplicate

For each of these, write the content into its **canonical home** first, then merely
*note* in the handoff that the sync happened:

- **Open mid-flight question** → append to `questions.md` (with best-guess attached if
  possible).
- **Decision discussed but not yet recorded** → append to `decisions.md` with the *why*,
  not just the *what*.
- **Assumption** about behaviour / API / user intent → append to `assumptions.md`.
- **Drift event raised but not fully resolved** → update `drift.md` with the current
  resolution status (open / asked-user / repaired).
- **Files modified in chat not yet in `touched-files.md`** → append to
  `touched-files.md`.

The handoff doc itself then says "synced N entries into decisions.md", not the entries
themselves. The workspace is the canonical store; the handoff is the chat-only delta.

## What to include in the handoff

1. **Suggested next action.** ONE command, not three options. If the active feature is
   mid-phase, name the next `rite-*` command. If a question blocks progress, point at
   it. If the user passed `[what the next session will focus on]`, tailor this section
   to the named focus.
2. **What just happened in this chat** (3–5 bullets). Distil — do not transcribe.
3. **External references** that exist only in chat: URLs, Figma links, screenshot paths,
   video timestamps the user pasted. List as references; do not embed.
4. **Live assumptions the agent is acting on** that the workspace doesn't reflect yet
   (after the sync above, this section should be near-empty — flag anything that
   remains).
5. **How to resume** — fixed boilerplate (see template below).

## What NOT to include

- Anything already in `spec.md`, `plan.md`, `tasks.md`, `state.md`, `decisions.md`,
  `evidence.md`, `review.md` — link by path instead.
- `git diff` output — the next agent runs `git diff` themselves.
- The full conversation transcript — distil.
- Secrets (API keys, tokens, PII, credentials). Redact aggressively.

## Output template

Loaded on demand from [`reference/handoff-template.md`](reference/handoff-template.md).
Fill in each section + write to `.devrites/work/<slug>/handoff.md` (or to a temp file
when no active feature).

After writing, **print the absolute path** of the handoff file so the user (or the next
agent) can open it without searching.
