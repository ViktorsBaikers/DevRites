# Git workflow

History is documentation. Keep it clean, atomic, and readable.

## Conventional Commits
Format: `type(scope): subject`
- **type**: `feat fix docs style refactor perf test build ci chore revert` (lower-case).
- **scope**: the area changed (lower-case).
- **subject**: imperative mood, no leading capital, no trailing period
  ("add export endpoint", not "Added export endpoint.").
- Keep the header short (~50, hard cap ~72). Put the *why* in the body, wrapped, after a
  blank line. Reference issues in the footer.

```
feat(export): stream large CSV exports

Buffering the whole file OOM-ed on >100k rows. Stream rows to the
response instead so memory stays flat.

Refs: #123
```

## Atomic commits
- One logical change per commit; it should build and pass tests on its own.
- Don't mix refactor + behavior change in one commit — split them so each is reviewable
  and revertible.

## Small, focused branches/PRs
- One concern per branch. Short-lived; rebase/merge per the project's convention.
- Keep PRs small (see `code-review.md`) so they get a real review.

## Never commit
- Secrets, credentials, tokens, or `.env` files. If one lands in history, rotate it and
  scrub it — deleting the file in a later commit is not enough.
- Generated artifacts, dependencies, or large binaries that belong in ignore rules.
