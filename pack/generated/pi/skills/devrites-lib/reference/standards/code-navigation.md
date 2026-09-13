# Code navigation

Decision tree for structural questions. Load with [`tooling.md`](tooling.md);
this file is the pack-canonical routing table agents cite in Build, Vet, and
Review.

## Route by question

| Question | First tool | Fallback |
| --- | --- | --- |
| Explore unfamiliar area, architecture, blast radius | `codegraph_explore` or codebase-memory `search_graph` / `get_architecture` | code-review-graph `detect_changes_tool` / `get_impact_radius_tool` |
| Callers, callees, data flow across files | codebase-memory `trace_path` | code-review-graph `query_graph_tool` |
| PR/review context for a diff | code-review-graph `get_review_context_tool` | `detect_changes_tool` |
| Exact symbol body before edit | `read_symbol` / `module_report` (pi-lens) | `get_code_snippet` then targeted `Read` |
| Rename, references, diagnostics | LSP (`lsp` rename/references) when available | graph trace + grep |
| Known file, one-line fix | `Read` at offset/limit | — |
| Literal string / config value | `Grep` | — |

## Before cross-file edits

1. Run **one** primary route from the table (record tool + query in the artifact).
2. Read symbol bodies you will change (`read_symbol` counts as read).
3. After edits: LSP diagnostics + project tests.

Skipping step 1 on a multi-file slice is a Build orient gap unless recorded
`cannot_verify`.

## Index freshness

| Index path | Provider |
| --- | --- |
| `.codegraph/` | CodeGraph |
| `.code-review-graph/` | code-review-graph MCP |
| `.codebase-memory/` | codebase-memory MCP |
Run `devrites-engine check indexes` for a quick presence check. Stale or missing index:
build/refresh per provider docs, or fall back to LSP + grep and note the gap.

## Anti-patterns

- Duplicate graph queries for reassurance.
- Whole-file reads when `read_symbol` or `module_report` suffices.
- Grep-only blast-radius claims when an index is present.
- Editing without reading the enclosing symbol body.
