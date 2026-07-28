---
name: devrites-refresh-indexes
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
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


# devrites-refresh-indexes: keep the code-intelligence indexes fresh

`tooling.md` says to cross-verify structural claims across codebase-memory-mcp, codegraph, and
graphify, and that **a disagreement between indexes is a signal**: a fresh read of live code
beats any index. A *stale* index manufactures exactly that disagreement. This keeps the three
mechanical indexes current after edits so the next lookup is trustworthy.

All three are **optional** (per `tooling.md`): each is refreshed only if its binary is on PATH
**and** it already tracks this repo. None present → this is a silent no-op. Never installs
anything, never blocks. Same incremental shape for all three; no LLM needed for code.

| Index | Tracks repo when | Refresh |
|---|---|---|
| **codebase-memory-mcp** (primary) | binary present + repo in `list_projects` (or `.codebase-memory/`) | `codebase-memory-mcp cli index_repository '{"repo_path":"<root>"}'` |
| **codegraph** | `.codegraph/` exists + `codegraph` on PATH | `codegraph sync` |
| **graphify** | `graphify-out/` exists + `graphify` on PATH | `graphify update .` |

## Automatic (already wired: no action needed)

The `Stop` hook `devrites-engine hook refresh-indexes` runs at end of turn. It self-guards: exits
instantly unless an index tracks the repo, exits instantly if no source file changed since the
last refresh, else stamps + locks and spawns a **detached** worker so the turn never blocks.
ON by default; disable with `DEVRITES_REFRESH_INDEXES=off`.

## Manual / thorough refresh (this skill)

Force a synchronous refresh now and print the report: resolve the hook across install layouts:

```bash
devrites-engine hook refresh-indexes --force .
```

Then the one case the hook can't cover:

- **Docs / papers / images changed** (not just code) → `graphify update` only re-runs AST. Run
  the full semantic re-extraction: `/graphify --update` (re-extracts only changed files).

## Notes

- codegraph and codebase-memory-mcp each have their own background watcher; the explicit
  reindex is the belt-and-suspenders fallback for when a watcher isn't running. graphify has no
  default watcher, so this is its primary freshness path.
- The index lags writes by ~1s after a refresh: don't re-query in the same instant you edit.
- Output hygiene (`prose-style.md`): don't name these tools to the user: say what changed
  ("re-indexed the edited files"), not which tool did it.
