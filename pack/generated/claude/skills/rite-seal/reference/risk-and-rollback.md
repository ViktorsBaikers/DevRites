# Risk & rollback

Before GO, know how to undo the change and what could go wrong in production.

## Risk scan
Rank risks; for each, note likelihood × impact and any mitigation:
- **Data**: schema changes, data migration, destructive operations, backfills.
- **Security**: new trust boundaries, auth/authz changes, secret handling, new deps.
- **Compatibility**: API contract changes, breaking changes for existing clients,
  feature interactions.
- **Operational**: new external dependency, config/env requirements, rate limits.
- **UX**: changed flows users rely on; unverified UI.

## Rollback plan (required for risky changes)
For each risky step, state how to back it out:
- **Migrations**: is there a reversible `down` / a documented manual revert? Is data
  loss possible on rollback?
- **Feature flag**: can it be disabled without a deploy?
- **Revert boundary**: can the change be `git revert`-ed cleanly, or does it entangle
  with other work?
- **Data**: is there a backup / a way to restore prior state?

## Blocking rules
- A **destructive or data-migration change with no rollback plan** is a **NO-GO**.
- A new external dependency with no failure handling is at least **Important**.
- Record the chosen rollback path in `seal.md` → "Rollback / Recovery".

## Fresh-context availability

A required review must run in a fresh spawned agent:

- If the named role is unavailable, record the roster skip reason and keep seal
  **NO-GO**.
- Never run the reviewer in the root context or log root work as a dispatched review.
- AFK never auto-accepts a missing reviewer.
- Security or irreversible-risk scope remains NO-GO while independent review is
  unavailable.

## Destructive operations
Confirm any destructive step with the user before shipping. Verify backups exist where
relevant. Never treat an irreversible action as routine.
