# Optional tooling: code intelligence, docs, memory

Every external tool in this file is optional. Detect what is installed, use the best fit,
and fall back to `Read` / `Grep` / `Glob`, which are always available. Never assume another
tool is installed, require an installation, or block a phase because a tool is missing.

DevRites runs in projects with different stacks and toolsets. Treat these tools as
available accelerators, never as dependencies.

## Code intelligence: structure, placement, callers, impact, blast-radius, trace

For structural questions such as "where is X", "what calls X", "what would changing X
break", or "how does X reach Y", use an available code-intelligence index. Follow this
order and skip any index that is not installed:

1. **codebase-memory-mcp: primary.** When available, answer the structural question here
   **first**: `search_graph`, `trace_path`, `detect_changes` (git-diff → affected symbols +
   blast radius), `get_architecture`, `get_code_snippet`, `query_graph`.
2. **Cross-check with codegraph *and* graphify when present.** Ask the same
   structural question of `codegraph` (`.codegraph/`, `codegraph_*`) **and** `graphify`
   (`graphify-out/`), and confirm they agree with the codebase-memory-mcp answer, especially
   for consequential claims (blast radius, every caller of a thing you're about to change,
   "nothing else uses this"). A disagreement between indexes is a signal, not noise: trust a
   fresh read of the **live code** over any index, and investigate the gap before relying on it.
3. **Use standard methods as the fallback.** When none of the three indexes is
   present, or when an index cannot pin an exact reference, use **LSP** (Claude Code Code
   Intelligence: go-to-definition, find-references, hover / signature, diagnostics, document &
   workspace symbols) plus **`Read` / `Grep` / `Glob`**, reading comprehensively rather than
   stopping at the first match (see `core.md` rule 1).

Use whatever subset is installed. Codebase Memory alone is sufficient; one additional
index still provides a cross-check. With no index, use standard methods. A missing index
never blocks a phase.

### Keeping the indexes fresh

An index is useful only when it matches the live code. After edits, a stale graph can
create the disagreement described in step 2. The
[`devrites-refresh-indexes`](../../../devrites-refresh-indexes/SKILL.md) Stop hook keeps
the three mechanical indexes current. At the end of a turn, it incrementally reindexes
whichever of codebase-memory-mcp, codegraph, and graphify track the repository. It runs in
a detached process, does nothing when no index is present, and is disabled by
`DEVRITES_REFRESH_INDEXES=off`. Use that skill to force a
synchronous refresh or rerun graphify's semantic pass after **doc** changes
(`/graphify --update`). Still trust a fresh read of the live code over any index when they disagree.

## Up-to-date library / framework docs: context7

When implementing against, choosing, or verifying an **external** library/framework whose
current API or version behaviour matters, use **context7 if available**: `resolve-library-id`
(library name + your question) → `query-docs` (the resolved id + the question).

context7 complements [`devrites-source-driven`](../../../devrites-source-driven/SKILL.md).
The project's **installed / pinned source still wins** for the version it runs. Use
context7 when local source/docs are missing or when you need current upstream behavior
that the installed copy may predate. Record the fact and source in `decisions.md` /
`evidence.md`. A context7 lookup is a cited source, not a memory.

## Up-to-date web facts: web search

When a **material decision** depends on a fact that neither the codebase nor installed
docs can answer, **search the web if a search tool is available**. This includes UX
patterns, standards, current practices, comparable products, pricing, and compatibility.
Include the finding in the option presented to the human. Order of
preference: **brave MCP is the primary** (`mcp__brave-search__brave_web_search`, or
`brave_local_search` for place/region queries); **fall back to the harness's native web search
only when brave MCP is unavailable**. Claude Code `WebSearch` / `WebFetch`, Codex `web_search`
(`--search` / `web_search = "live"` for fresh pages; its default `"cached"` mode serves an
OpenAI-indexed snapshot); else skip and log the open question. A web fact is a **cited
source**, not a memory. Record the claim and URL in `decisions.md` or the option's
rationale, just as for a context7 lookup.

If no search tool is present, continue without one and log the open question. Search
informs the human's decision; it does not replace that decision.

On Claude Code, `WebFetch` is cached per project and revalidated against the origin on
reuse. A cached response is replayed only after an HTTP 304, so do not skip verification
to save a request. The `devrites-source-cache` hooks provide this behavior
(`DEVRITES_SOURCE_CACHE=off` to disable); it pairs with
[`devrites-source-driven`](../../../devrites-source-driven/SKILL.md). (Claude-only. Codex has no
`WebFetch` tool to intercept, and its `web_search` already serves from a cached index, so the same
caching is built in there.)

## Architecture & decision memory: codebase-memory-mcp

When codebase-memory-mcp is available, use `get_architecture` for an overview
(languages, packages, routes, hotspots, clusters)
during `$rite-spec`, `$rite-clarify`, `$rite-define`, or `$rite-zoom-out`; `manage_adr` for an ADR-style record
at `$rite-define` / `$rite-seal`. These records complement `decisions.md`; the
workspace files remain canonical.

## Output hygiene

Per [`prose-style.md`](prose-style.md): don't name these tools to the user. Say what you
learned ("the change touches three call sites"), not which tool found it.
