---
name: devrites-refresh-indexes
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- For every DevRites specialist or writer dispatch, first call `spawn_agent` with the named `devrites-<role>` custom role. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If `spawn_agent` is callable but a named read-only role is unavailable, use generic `explorer` only when the host proves that run has a runtime-enforced read-only sandbox. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. A missing read-only custom role is not evidence that spawning is unavailable.
- Never dispatch generic `worker` for `devrites-slice-wright` unless the host proves that worker run carries exact DevRites identity and the same `.wright-allowlist` enforcement as the named role. Codex reports a generic run as `agent_type=worker`, so the generated global hooks cannot prove that binding. Reject that unsafe rung and use the documented labelled inline wright path with `.reconcile-inline` plus the full reconcile gate.
- If the host cannot prove the generic explorer is runtime read-only, reject that rung too. Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, and apply every fallback risk gate. An unbound generic wright or unconfined generic explorer is such a safety rejection, not evidence that no agents exist.
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
