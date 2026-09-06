# rite-ship: anti-patterns

Load this before any irreversible action: `git commit`, `push`, `tag`, publish, deploy,
or when tempted to skip the close-out.

Pack-wide rationalizations + red flags: see
[standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Seal said GO, so I can just push." | A GO verdict is a decision, not authorization. **Render the type-GO prompt every time** and wait for the literal `GO`. |
| "I'll push the tag now; type-GO is a formality." | Type-GO is the last safety net before an irreversible action. Always render it: even under auto-trigger. |
| "`git add -A` is faster than listing touched files." | Stage only what's in `touched-files.md`. `-A` sweeps secrets, scratch files, and out-of-scope edits into the ship. |
| "The evidence is probably still good." | Recheck the canonical candidate digest and every artifact binding. Any mismatch stops. |
| "The ledger/DESIGN/ADR update is just ship bookkeeping." | Those are candidate paths. Route the change through Polish, affected Prove, Review, and Seal; Ship is read-only. |
| "Delete the workspace, the feature's done." | Closing **archives** the workspace (`.devrites/archive/<slug>/`); it never deletes the audit trail. |

## Red Flags

- About to run `git push` / `git tag` without rendering the type-GO prompt.
- Shipping without a `GO` verdict recorded in `seal.md`.
- Staging files not listed in `touched-files.md`, or any secret / `.env`.
- Changing a candidate path or the manifest after Seal.
- Pushing or tagging before the committed NUL-delimited state/path and digest checks pass.
- `rm`-ing `.devrites/work/<slug>/` instead of archiving it.
- Marking the feature `done` but leaving `.devrites/ACTIVE` pointing at it.
