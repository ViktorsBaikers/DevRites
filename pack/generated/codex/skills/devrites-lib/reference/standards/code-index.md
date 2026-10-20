# Code index

> Root-only: not inlined in --role bundles.

> Applies when: choosing structural search tools (code index vs grep) before exploring.

Decision tree for structural questions. Load with [`tooling.md`](tooling.md);
this file is the pack-canonical routing table agents cite in Build, Vet, and
Review. Edit-safety rules that hold with or without an index live in
[`code-navigation.md`](code-navigation.md). The one supported code index is
**codebase-memory-mcp**. Every call takes `project`: pass the absolute
repository root (it resolves to the indexed project that contains it); never
guess a project name.

## Preflight

Once per agent, before the first structural claim:

1. `index_status` (`project` = repo root). An error or "not indexed" means no
   index. The **root** session may run `index_repository` once (`repo_path` =
   repo root, default mode, never `persistence: true`, which writes
   `.codebase-memory/` into the repo), then re-check. A subagent never indexes:
   it records `index: unavailable` and uses the fallback column.
2. `status: ready` only means the graph has nodes; it is not freshness. When
   the response carries `watch.watched: true`, auto-sync keeps it current.
   Otherwise, if files changed after `indexed_at`, the root may re-run
   `index_repository`; everyone else relies on the coverage check below.

## Route by question

| Question | First call | Fallback |
| --- | --- | --- |
| Orient in an area, architecture | `get_architecture` (`aspects: ["overview"]`, `path`) | Glob + targeted `Read` |
| Where is X | `search_graph` (`name_pattern` regex, or BM25 `query`) | Grep |
| Does a similar implementation already exist? | `search_graph` with `semantic_query: [purpose words]`, then `name_pattern` variants | Grep on purpose-words |
| Callers, callees, call paths across files | `search_graph` for the exact name, then `trace_path` (`direction` `inbound`/`outbound`/`both`, `depth` 1-3) | LSP references + Grep |
| Which tests reach a symbol | `trace_path` (`direction: "inbound"`, `include_tests: true`); direct tests: `query_graph` on `TESTS` edges | Grep test files for the name |
| Value/taint flow | `trace_path` (`mode: "data_flow"`), then read each hop ([`tooling.md`](tooling.md) route table) | `cannot_verify` |
| Blast radius of a diff | `detect_changes` (`base_branch` = feature base, always explicit; `scope: "impact"`, `depth: 2`) | `git diff --name-only` + `trace_path` per changed symbol |
| Exact symbol body before edit | `get_code_snippet` (`qualified_name` from `search_graph`) | Targeted `Read` |
| File outline | `get_file_outline` | `Read` |
| Multi-hop structure | `query_graph` (read-only Cypher, always with `LIMIT`) | Split into `trace_path` calls |
| Rename, references, diagnostics | LSP (`lsp` rename/references) when available | `trace_path` + Grep |
| Known file, one-line fix | `Read` at offset/limit | — |
| Literal string, config value, non-code file | `Grep` | — |

Exact values matter: an unknown `mode` silently becomes `calls`, and an
unknown `detect_changes` `scope` silently returns files only. `semantic_query`
is an array, cannot combine with `query`, and needs a `full`/`moderate` index.
`detect_changes` covers commits since the merge base plus unstaged, staged, and
untracked files; `scope: "tests"` exists only from v0.12. Results page: check
`has_more` and continue with `offset`/`cursor`.

## Coverage before citing

After the candidate paths are known, call `check_index_coverage` once with every
path the claim relies on (`paths`, max 128), plus `scopes` for negative or
exhaustive claims ("no other callers", "dead"). `no_recorded_issue` is best
effort, not proof. For `freshness` other than `metadata_match`, or `status`
`partial`/`excluded`/`skipped`/`coverage_unavailable`, read or grep those files
directly and qualify the claim.

## Query budget

Spend the minimum that settles the question:

- Minimal output first — pass the smallest detail/scope option a tool offers;
  escalate to a fuller answer only when the minimal one cannot decide.
- One structural question ≈ five index calls. The sixth means the question needs
  re-scoping or a file read, not another query.
- Stop at sufficiency: an index answer that names the symbols and their source
  ends the search; re-verifying it with grep is waste, not diligence.

## Index location

The index lives in the server's store (`~/.cache/codebase-memory-mcp`, or
`$CBM_CACHE_DIR`), not in the repository. `devrites-engine check indexes` only
detects an in-repo `.codebase-memory/` artifact; `index_status` is
authoritative.

## Tool availability

- **MCP:** Claude hosts expose `mcp__codebase-memory-mcp__<tool>` (allowlisted
  in `.claude/settings.json` and agent `tools:` lines). omp subagents and
  Codex children inherit every MCP server of the parent session (omp names
  the tools `mcp__codebase_memory_mcp_<tool>`; a Codex agent file cannot add
  servers). Generated Pi agents list none: pi-subagents fails a child whose
  `tools:` names a tool it cannot load. A missing server offers no tools — use
  the fallback column.
- **CLI** when the host exposes no MCP tools (e.g. Devin agents via `exec`,
  Pi agents via `bash`):
  `codebase-memory-mcp cli <tool> --project <repo root> --format json` with
  kebab-case flags (`--function-name X --direction inbound --include-tests`);
  repeat a flag for array values. Exit 1 means the tool returned an error.
- **Read-only sandbox** or no tool route at all: the CLI cannot run in a
  read-only sandbox (it writes daemon locks outside the workspace). Use
  **parent handoff**: the root runs the queries and puts the
  project, each query, its raw rows, cited paths, and coverage status in the
  dispatch packet. Raw query output is input, like a diff; never add the root's
  interpretation ([`agents.md`](agents.md) seeding rule). A child whose packet
  lacks what it needs returns the exact query for the root to run instead of
  guessing.
- Forbidden index calls are listed in
  [`code-navigation.md`](code-navigation.md) § Anti-patterns.

## Anti-patterns

- Duplicate graph queries for reassurance.
- Whole-file reads when `get_code_snippet` or `get_file_outline` suffices.
- `trace_path` on a guessed name instead of the exact name from `search_graph`.
- Citing a graph result without `check_index_coverage` on the cited paths.
