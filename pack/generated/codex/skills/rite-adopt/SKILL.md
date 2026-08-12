---
name: rite-adopt
description: Adopt an existing codebase into DevRites by reverse-engineering current behavior and establishing a baseline workspace.
argument-hint: "[path or area to adopt] [+ what you want to build next]"
user-invocable: true
---

# $rite-adopt: onboard existing code

Reverse-engineer current behavior into `spec.md`; do not rewrite product code or create a
parallel convention system.

## Workflow

1. Read core; resolve repository/sub-area and next objective. Ask once only if material.
2. Follow [`reference/adoption.md`](reference/adoption.md): inspect current
   behavior, architecture and placement, callers, reusable seams, repository/CI
   commands, visible code/test patterns, non-obvious constraints, and any touched
   load-bearing seam that needs `characterize-before-modify`. Read existing
   `AGENTS.md`, `CLAUDE.md`, product, and design guidance.
3. Create the workspace and write `spec.md`, `decisions.md`, `assumptions.md`,
   `questions.md`, and `state.md`. The spec separates the current baseline from
   the measurable next objective.
4. For a structured baseline, follow the native ledger workflow in
   [`ledger.md`](../rite-polish/reference/ledger.md): read current specs, preview
   ADDED entries, confirm paths, refuse escape/symlink/clobber/ignored targets,
   write, lint, and require an empty re-preview.
5. For a verified non-obvious policy or recurring/costly mistake, prefer existing hook/lint/
   CI/type/schema enforcement; else propose one evidence-cited nearest-scope instruction edit.
   Apply only after review; never create convention bands, scores, or a command cache.
6. A genuine non-negotiable invariant may be proposed for
   `.devrites/principles.md`; the human must ratify it.
7. Set the next step to `$rite-clarify` and stop. Do not plan or build.

## Output

```text
Done: adopted <scope> into <slug>.
Changed: <workspace artifacts>; project guidance <proposed | unchanged>
Evidence: <source/CI paths inspected>
Open: <questions | none>
Next: $rite-clarify
```
