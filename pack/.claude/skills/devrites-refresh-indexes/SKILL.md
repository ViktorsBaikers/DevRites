---
name: devrites-refresh-indexes
description: Refresh stale optional indexes. Use when structural lookup disagrees with live code or rite-doctor requests a synchronous reindex.
user-invocable: false
---

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
