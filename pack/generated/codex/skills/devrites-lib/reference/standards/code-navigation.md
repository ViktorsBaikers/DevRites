# Code navigation

> Applies when: choosing structural search tools (code index vs grep) before exploring.

Decision tree for structural questions. Load with [`tooling.md`](tooling.md);
this file is the pack-canonical routing table agents cite in Build, Vet, and
Review. The one supported code index is **codebase-memory-mcp**. Every call
takes `project`: pass the absolute repository root (it resolves to the indexed
project that contains it); never guess a project name.

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

## Before cross-file edits

1. Run **one** primary route from the table (record tool + query in the artifact).
   **Compute the affected set before writing code, not after** — the blast-radius
   list is the edit plan: every file on it is a touch or a conscious skip.
2. Read symbol bodies you will change (`get_code_snippet` counts as read), comments
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
   the slice records "diagnostics clean". The converse holds: an error that the project
   compile/typecheck of the current bytes does not reproduce is stale. Record it once
   with that command's result and move on; never stub, suppress, or re-chase it.

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
  no import statement links. So do DB columns and wire formats that another
  language reads. Grep the literal route/command/topic/column when a
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

## Index location

The index lives in the server's store (`~/.cache/codebase-memory-mcp`, or
`$CBM_CACHE_DIR`), not in the repository. `devrites-engine check indexes` only
detects an in-repo `.codebase-memory/` artifact; `index_status` is
authoritative.

## Tool availability

- **MCP:** Claude hosts expose `mcp__codebase-memory-mcp__<tool>` (allowlisted
  in `.claude/settings.json` and agent `tools:` lines); generated omp agents
  list the read-only tools as `mcp__codebase_memory_mcp_<tool>`. Generated Pi
  agents list none: pi-subagents fails a child whose `tools:` names a tool it
  cannot load. A missing server offers no tools — use the fallback column.
- **CLI** when the host exposes no MCP tools (e.g. Devin agents via `exec`,
  Pi agents via `bash`):
  `codebase-memory-mcp cli <tool> --project <repo root> --format json` with
  kebab-case flags (`--function-name X --direction inbound --include-tests`);
  repeat a flag for array values. Exit 1 means the tool returned an error.
- **Read-only sandbox** (Codex read-only children) or no tool route at all: the
  CLI cannot run in a read-only sandbox (it writes daemon locks outside the
  workspace). Use **parent handoff**: the root runs the queries and puts the
  project, each query, its raw rows, cited paths, and coverage status in the
  dispatch packet. Raw query output is input, like a diff; never add the root's
  interpretation ([`agents.md`](agents.md) seeding rule). A child whose packet
  lacks what it needs returns the exact query for the root to run instead of
  guessing.
- **Never call** `delete_project`, `manage_adr` `update`/`set_sections`, or
  `ingest_traces`. Only the root calls `index_repository`.

## Anti-patterns

- Duplicate graph queries for reassurance.
- Whole-file reads when `get_code_snippet` or `get_file_outline` suffices.
- Grep-only blast-radius claims when an index is present.
- `trace_path` on a guessed name instead of the exact name from `search_graph`.
- Citing a graph result without `check_index_coverage` on the cited paths.
- Editing without reading the enclosing symbol body.
