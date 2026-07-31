---
name: rite-adopt
description: Adopt an existing codebase into DevRites by reverse-engineering current behavior and establishing a baseline workspace.
argument-hint: "[path or area to adopt] [+ what you want to build next]"
user-invocable: true
---

# /rite-adopt: onboard existing code

Reverse-engineer existing behavior into the same `spec.md` contract used by new
features. Adoption documents the repository; it does not rewrite it or create a
parallel convention system.

## Workflow

1. Read the core standards and resolve the repository/sub-area plus the next
   objective. Ask once when either is materially ambiguous.
2. Follow [`reference/adoption.md`](reference/adoption.md): inspect current
   behavior, architecture and placement, callers, reusable seams, repository/CI
   commands, visible code/test patterns, and non-obvious constraints. Read
   existing `AGENTS.md`, `CLAUDE.md`, product, and design guidance.
3. Create the workspace and write `spec.md`, `decisions.md`, `assumptions.md`,
   `questions.md`, and `state.md`. The spec separates the current baseline from
   the measurable next objective.
4. For a structured baseline, follow the native ledger workflow in
   [`ledger.md`](../rite-polish/reference/ledger.md): read current specs, preview
   ADDED entries, confirm paths, refuse escape/symlink/clobber/ignored targets,
   write, lint, and require an empty re-preview.
5. If a repeatedly observed operating rule should govern future work, propose
   one exact edit to the nearest existing `AGENTS.md`, `CLAUDE.md`, or scoped
   document. Apply only after review. Do not create convention bands, scores, or
   a command cache.
6. A genuine non-negotiable invariant may be proposed for
   `.devrites/principles.md`; the human must ratify it.
7. Set the next step to `/rite-clarify` and stop. Do not plan or build.

## Output

```text
Done: adopted <scope> into <slug>.
Changed: <workspace artifacts>; project guidance <proposed | unchanged>
Evidence: <source/CI paths inspected>
Open: <questions | none>
Next: /rite-clarify
```
