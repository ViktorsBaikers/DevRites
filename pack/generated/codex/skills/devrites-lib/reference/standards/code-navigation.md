# Code navigation

> Applies when: editing across files or claiming a blast radius, with or without a code index.

Edit-safety rules that hold whatever tool finds the code. Index routing
(preflight, route table with a fallback for every question, coverage, query
budget, host access) lives in [`code-index.md`](code-index.md); load it with
[`tooling.md`](tooling.md) when choosing a structural search route.

## Before cross-file edits

1. Run **one** primary route from the [`code-index.md`](code-index.md) route
   table; use its fallback column only when no index is available (record
   tool + query in the artifact).
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

## Anti-patterns

- Editing without reading the enclosing symbol body.
- Grep-only blast-radius claims when an index is present.
- Calling `delete_project`, `manage_adr` `update`/`set_sections`, or
  `ingest_traces`; only the root calls `index_repository`, never with
  `persistence: true`, and a subagent never indexes.
