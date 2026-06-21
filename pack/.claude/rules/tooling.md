# Optional tooling — code intelligence, docs, memory

Every external tool DevRites can use is **optional**. Detect what's present, use the best fit
for the job, and **degrade gracefully to `Read` / `Grep` / `Glob`** (always available) when
nothing is. Never assume a tool is installed, never require installing one to run a phase, and
never block on a missing tool — the fallback path is a first-class path, not a failure.

DevRites is stack-agnostic and installs into arbitrary projects; an index or MCP server that
exists in one repo is absent in the next. Treat the tools below as accelerators you reach for
*when available*, not dependencies.

## Code intelligence — structure, placement, callers, impact, blast-radius, trace

For "where is X / what calls X / what would changing X break / how does X reach Y", reach for a
code-intelligence index when available. The three indexes below are **recommended, not
mandatory** — follow this order and skip any that isn't installed:

1. **codebase-memory-mcp — primary.** When available, answer the structural question here
   **first**: `search_graph`, `trace_path`, `detect_changes` (git-diff → affected symbols +
   blast radius), `get_architecture`, `get_code_snippet`, `query_graph`.
2. **Cross-verify with codegraph *and* graphify (both, when present).** Re-ask the same
   structural question of `codegraph` (`.codegraph/`, `codegraph_*`) **and** `graphify`
   (`graphify-out/`), and confirm they agree with the codebase-memory-mcp answer — especially
   for load-bearing claims (blast radius, every caller of a thing you're about to change,
   "nothing else uses this"). A disagreement between indexes is a signal, not noise: trust a
   fresh read of the **live code** over any index, and investigate the gap before relying on it.
3. **Standard methods — the always-available fallback.** When none of the three indexes is
   present — or to pin an exact reference an index is unsure of — use **LSP** (Claude Code Code
   Intelligence: go-to-definition, find-references, hover / signature, diagnostics, document &
   workspace symbols) plus **`Read` / `Grep` / `Glob`**, reading comprehensively rather than
   stopping at the first match (see `core.md` rule 1).

Use whatever subset is installed: codebase-memory-mcp alone is fine; codebase-memory-mcp plus
one of the others still cross-verifies; none present → standard methods. The fallback path is a
first-class path — never block a phase on a missing index.

## Up-to-date library / framework docs — context7

When implementing against, choosing, or verifying an **external** library/framework whose
current API or version behaviour matters, use **context7 if available**: `resolve-library-id`
(library name + your question) → `query-docs` (the resolved id + the question).

context7 pairs with [`devrites-source-driven`](../skills/devrites-source-driven/SKILL.md), it
doesn't replace it: the project's **installed / pinned source still wins** for the version the
project actually runs. Reach for context7 when the local source/docs are missing, or when you
need the *current upstream* behaviour the installed copy may predate. Record the fact + its
source in `decisions.md` / `evidence.md` the same way — a context7 lookup is a cited source,
not a memory.

## Architecture & decision memory — codebase-memory-mcp

Where a fast codebase map or a durable decision record helps, and codebase-memory-mcp is
available: `get_architecture` for an overview (languages, packages, routes, hotspots, clusters)
during `/rite-spec`, `/rite-define`, or `/rite-zoom-out`; `manage_adr` for an ADR-style record
at `/rite-define` / `/rite-seal`. This complements the workspace `decisions.md`; it never
replaces it — the workspace files remain the canonical source of truth.

## Output hygiene

Per [`prose-style.md`](prose-style.md): don't name these tools to the user. Say what you
learned ("the change touches three call sites"), not which tool found it.
