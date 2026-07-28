---
name: rite-handoff
description: User-invoked handoff writer: sync chat-only context into `.devrites/` and write a fresh-agent handoff.
user-invocable: true
disable-model-invocation: true
argument-hint: "[what the next session will focus on]"
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
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
