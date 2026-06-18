# Close-out — archive the workspace, free the active slot

Closing a feature means it stops being the *active* work, not that its record is
deleted. DevRites keeps the full audit trail; it just moves out of the live path.

## What close-out does

1. **Mark done.** Set `state.md` → `Phase: done`, `Status: done`, and a `Next step`
   of `/rite-spec <next feature>`.
2. **Archive.** Run the deterministic script:
   ```bash
   bash .claude/skills/devrites-lib/scripts/close-out.sh <slug>
   ```
   It moves `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` (every `.md`
   intact) and clears `.devrites/ACTIVE` **only if** ACTIVE still points at `<slug>`.
   It refuses to clobber an existing `.devrites/archive/<slug>/` (exit 5).
3. **Confirm.** ACTIVE is now empty, so the next `/rite-spec` starts a clean feature
   and `/rite-status` reports "no active feature".

## Why archive, not delete

- The `.md` files (`spec`, `decisions`, `assumptions`, `evidence`, `seal`, `ship`, …)
  are the project's record of *why* the feature is the way it is. A future
  `/rite-zoom-out` or incident review reads them.
- Deleting them would make the workflow's own "evidence over confidence" rule a lie.

## Re-opening an archived feature

To resume archived work, move it back:
`mv .devrites/archive/<slug> .devrites/work/<slug>` and write `<slug>` into
`.devrites/ACTIVE`. Nothing is lost — close-out is reversible by design.
