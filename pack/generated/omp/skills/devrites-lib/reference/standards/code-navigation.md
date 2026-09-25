# Code navigation

> Applies when: choosing structural search tools (code index vs grep) before exploring.

Decision tree for structural questions. Load with [`tooling.md`](tooling.md);
this file is the pack-canonical routing table agents cite in Build, Vet, and
Review.

## Route by question

| Question | First tool | Fallback |
| --- | --- | --- |
| Explore unfamiliar area, architecture, blast radius | `codegraph_explore` or codebase-memory `search_graph` / `get_architecture` | code-review-graph `detect_changes_tool` / `get_impact_radius_tool` |
| Callers, callees, call paths across files (value/taint flow: [`tooling.md`](tooling.md) route table) | codebase-memory `trace_path` | code-review-graph `query_graph_tool` |
| PR/review context for a diff | code-review-graph `get_review_context_tool` | `detect_changes_tool` |
| Exact symbol body before edit | `read_symbol` / `module_report` (pi-lens) | `get_code_snippet` then targeted `Read` |
| Rename, references, diagnostics | LSP (`lsp` rename/references) when available | graph trace + grep |
| Known file, one-line fix | `Read` at offset/limit | — |
| Literal string / config value | `Grep` | — |
| Does a similar implementation already exist? | embedding/similarity tool when present | `search_graph` on purpose-words and name variants, then targeted `Read` |

## Before cross-file edits

1. Run **one** primary route from the table (record tool + query in the artifact).
   **Compute the affected set before writing code, not after** — the blast-radius
   list is the edit plan: every file on it is a touch or a conscious skip.
2. Read symbol bodies you will change (`read_symbol` counts as read), comments
   included — a why-comment is often the only record of a decision.
3. After edits: LSP diagnostics + project tests. Compiler errors are the oracle
   for callers the graph listed — fix them in dependency order, direct
   dependents first. Errors absent from the list mean the list was incomplete
   (see blind spots below); record the gap. Diagnostics count only when bound to the
   current bytes: the server received the post-edit sync or save, the project is loaded
   and ready, and the file is within its size limits. A `0` from a cold, not-ready, or
   oversized server, a pre-edit result, or a timed-out probe is `stale` (or `unavailable:
   <reason>` when none ran), never clean; fall back to the project typecheck/compile command. **Failing case:**
   the language server returns 0 diagnostics before the project finishes compiling and
   the slice records "diagnostics clean".

Skipping step 1 on a multi-file slice is a Build orient gap unless recorded
`cannot_verify`.

## Query budget

Spend the minimum that settles the question:

- Minimal output first — pass the smallest detail/scope option a tool offers;
  escalate to a fuller answer only when the minimal one cannot decide.
- One structural question ≈ five index calls. The sixth means the question needs
  re-scoping or a file read, not another query.
- Stop at sufficiency: an index answer that names the symbols and their source
  ends the search; re-verifying it with grep is waste, not diligence.

## What the static graph cannot see

Import/call edges are only the statically visible share of the dependency
surface. A blast-radius answer that ignores the rest is a false negative
factory:

- **Cross-language edges:** HTTP/RPC call sites (`fetch("/api/…")` → the route
  handler file), subprocess/CLI invocations, and queue topic names couple files
  no import statement links. Grep the literal route/command/topic when a
  producer or consumer file changes.
- **Dynamic binding:** `import()`/dynamic `require`, reflection, string-keyed
  DI/registries, plugin/extension loading, ORM model registration,
  `require.context`-style glob imports. A symbol with zero static dependents
  may be load-bearing at runtime.
- **Co-change coupling:** files that change together in `git log` but share no
  import edge are coupled by convention (paired configs, parallel
  implementations, duplicated schemas). Before a structural edit, run
  `git log --name-only` on the target file's recent history and treat repeat
  companions as soft dependents.
- **Type-only edges** (`import type`, generics-only references) inform impact
  and dead-code reads but do not execute — do not count them as runtime
  coupling (see [`architecture-health.md`](architecture-health.md)).

The compiler and the test suite are the oracle for everything in this list:
zero static dependents + zero compiler errors + green tests is evidence;
zero static dependents alone is a hypothesis.

## Index freshness

| Index path | Provider |
| --- | --- |
| `.codegraph/` | CodeGraph |
| `.code-review-graph/` | code-review-graph MCP |
| `.codebase-memory/` | codebase-memory MCP |
| `graph.json` | Graphify (`graphify query` / `graphify path A B`) |
Run `devrites-engine check indexes` for a quick presence check. It detects
**in-repo artifacts only** — codebase-memory may be indexed in the server's
central store with no `.codebase-memory/` dir; when the MCP server answers
(`list_projects`/`index_status`), trust it over the file probe. Stale or missing
index: build/refresh per provider docs, or fall back to LSP + grep and note the gap.

## Tool availability

- On Claude hosts, provider tools arrive as `mcp__<server>__<tool>` — e.g.
  `mcp__codegraph__codegraph_explore`, `mcp__code-review-graph__query_graph_tool`,
  `mcp__codebase-memory-mcp__search_graph`. DevRites agent profiles and
  `.claude/settings.json` already allowlist these servers; a server that is not
  installed simply offers no tools — fall back per the table, no error path needed.
- Server names are install-configured (`codebase-memory-mcp` and `codebase-memory`
  are both allowlisted). If your server uses another name, add it to
  `settings.json` allow and the agent `tools:` lines.
- CodeGraph and Graphify also run as CLIs (`codegraph explore`,
  `graphify query` / `graphify path A B`) — allowlisted via `Bash(...)` rules.
- On non-Claude hosts the `mcp__*` entries are ignored by generation; use each
  host's own nav extensions or the CLI forms. All listed tools are read-only
  navigation — do not call index rebuild, refactor-apply, or mutation tools.

## Anti-patterns

- Duplicate graph queries for reassurance.
- Whole-file reads when `read_symbol` or `module_report` suffices.
- Grep-only blast-radius claims when an index is present.
- Editing without reading the enclosing symbol body.
