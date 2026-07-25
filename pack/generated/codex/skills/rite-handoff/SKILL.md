---
name: rite-handoff
description: User-invoked handoff writer: sync chat-only context into `.devrites/` and write a fresh-agent handoff.
user-invocable: true
disable-model-invocation: true
argument-hint: "[what the next session will focus on]"
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


# $rite-handoff: chat-only context, into a fresh-agent doc

DevRites' workspace `.devrites/work/<slug>/` already captures everything that *should*
persist (spec, plan, tasks, decisions, evidence, drift, review). This skill captures
what the **chat** is holding that is **not** in the workspace, so a fresh agent (or
the same user after `/clear`) can pick the work up without re-reading the transcript.

Read `.agents/skills/devrites-lib/reference/standards/core.md` first: its "Persistence before stopping" discipline is
exactly what this skill executes. The other rule files load on demand.

Then run `devrites-engine preamble` for deterministic workspace orientation.

## Where to write

- **Active feature exists** → `.devrites/work/<slug>/handoff.md` (overwrites the previous
  handoff; the workspace is the canonical home for this slug).
- **No active feature** → OS temp dir (`$TMPDIR` / `/tmp` / `%TEMP%`) as
  `rite-handoff-<ISO-timestamp>.md`. Print the absolute path after writing.

## Before you write: sync, don't duplicate

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
   to the named focus. Where you were mid-thought, **quote the most recent next-step
   verbatim** from the chat (the exact sentence describing what to do next) so no nuance
   is lost to paraphrase.
2. **What just happened in this chat** (3-5 bullets). Distil: do not transcribe.
3. **External references** that exist only in chat: URLs, Figma links, screenshot paths,
   video timestamps the user pasted. List as references; do not embed.
4. **Live assumptions the agent is acting on** that the workspace doesn't reflect yet
   (after the sync above, this section should be near-empty: flag anything that
   remains).
5. **How to resume:** fixed boilerplate (see template below).

## What NOT to include

- Anything already in `spec.md`, `plan.md`, `tasks.md`, `state.md`, `decisions.md`,
  `evidence.md`, `review.md`: link by path instead.
- `git diff` output: the next agent runs `git diff` themselves.
- The full conversation transcript: distil.
- Secrets (API keys, tokens, PII, credentials). Redact aggressively.
- **New ideas the user didn't confirm.** A handoff records what *happened* and what's *next*.
  Leave out fresh suggestions, scope the user didn't agree to, and redesigns you thought of. Capture
  the state, not your opinions about it.

## Output template

Loaded on demand from [`reference/handoff-template.md`](reference/handoff-template.md).
Fill in each section and write to `.devrites/work/<slug>/handoff.md` (or to a temp
file when no active feature). Then use the compact reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)):

```
Done: handoff written for <slug | session>.
Changed: handoff.md plus synced workspace artifacts <n>
Evidence: not applicable; chat-only context persisted
Open: <none | unsynced caveats>
Next: <single resume command>
Record: <absolute path to handoff.md>
↻ Hygiene: /clear
```

Print the absolute path in `Record:` so the user or next agent can open it without searching.

## Session hygiene
Close with the one-line hygiene advisory + the single resume command. This skill *is* the
pre-`/clear` bridge, so it's where the advisory matters most (`context-hygiene.md`):
```
↻ Hygiene: /clear
```
