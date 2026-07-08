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

## Change summary — prove the scope
When you hand back a change, state what you deliberately left alone, not only what you touched.
A **"Things I didn't touch (intentionally)"** line — the adjacent smells you noticed and declined
to fix — proves the diff is feature-scoped ([`core.md`](core.md) rule 7) and matches
`touched-files.md`, rather than an unsolicited renovation the reviewer has to untangle from the
real change. Noticed-but-not-fixed becomes an FYI follow-up, never a silent addition to this diff.

## Versioning & changelog
- **The version is a promise.** Semver encodes what a consumer may rely on. A "patch" that
  changes behaviour someone depended on is a major wearing a disguise — every observable behaviour
  is a contract (Hyrum's law, [`deprecation.md`](deprecation.md)). When unsure whether a change is
  breaking, assume it is; a surprise major is far cheaper than a broken consumer.
- **The tag is the single source of truth** for the version. Derive it from the tag; never
  hand-edit a version copied across `package.json`, a constant, and a header — three copies drift,
  one tag can't.
- **A changelog is not the git log.** Curate it by user impact — what a consumer must know to
  upgrade — not by commit. Write the entry in the same change that earns it, while the impact is
  fresh; a changelog reconstructed later is a guess.

## Never commit
- Secrets, credentials, tokens, or `.env` files. If one lands in history, rotate it and
  scrub it — deleting the file in a later commit is not enough.
- Generated artifacts, dependencies, or large binaries that belong in ignore rules.
