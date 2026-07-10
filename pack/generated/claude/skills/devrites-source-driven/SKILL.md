---
name: devrites-source-driven
description: Verify uncertain framework/library behavior against official docs or installed source before relying on it. Use when the user says "check the docs", "verify this assumption", or hits an unfamiliar API. Not for internal code.
user-invocable: false
---

# devrites-source-driven — verify, don't guess

A confident wrong assumption about a library is a bug waiting to ship. When behavior
matters and isn't certain, check the source of truth.

## When to trigger
- You're about to rely on an API signature, default, config key, or behavior you're not
  sure of.
- The docs in memory might be stale (the project pins a different version).
- Tests can't easily prove the assumption, but it drives the implementation.
- An error message points at framework behavior you don't fully understand.

## How
1. **Find the version** the project actually uses (lockfile, manifest) — behavior is
   version-specific.
2. **Consult the source of truth**, in order: the installed package's own source/types
   in `node_modules`/gem/site-packages; context7 if available (`resolve-library-id` →
   `query-docs`) for current upstream docs; official docs for *that version*; the project's
   existing usage of the same API.
3. **Confirm the specific fact** — the signature, the default, the edge behavior — not a
   general impression.
4. **Record it** in `decisions.md` (or `evidence.md`): the fact, the version, and the
   source (path or URL). Future phases trust the record instead of re-checking.

## Delegate reading legwork bigger than one fact
When the question is an *area*, not a fact — surveying a library's API surface, an unfamiliar
subsystem's docs, a migration guide — dispatch a **background agent** instead of reading inline:
it investigates against the same source-of-truth order above, cites each claim (path or URL +
version), and writes one note to the active workspace's `references/` directory, linked from
`references.md`. You keep working while it reads; later phases trust the cited note. The inline
flow above stays the path for confirming a single fact.

## Rules
- Prefer the **installed** source over remembered docs — it can't be out of date.
- Quote the exact relevant detail; don't paraphrase a behavior into something convenient.
- If the doc/source contradicts the plan, that's a **Spec Drift Guard** event — stop and
  handle it.
- Don't rabbit-hole: confirm the one fact you need, record it, return.

## Re-fetching is cheap (and still fresh)
Fetching the same doc URL twice costs almost nothing: a WebFetch is transparently cached per
project and, on reuse, revalidated against the origin — the cached reading is replayed **only**
when the server confirms the page is unchanged (HTTP 304). A 304 is a fresh verification, not a
memory read, so citing a revalidated page is as sound as re-fetching it. Fetch freely; don't
skip a check to save a round trip. (Mechanism: the `devrites-source-cache` hooks; off via
`DEVRITES_SOURCE_CACHE=off`. Web-search + cache policy lives in [`tooling.md`](../devrites-lib/reference/standards/tooling.md).)
