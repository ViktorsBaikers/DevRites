# The irreversible ship — type-GO + git ladder

`$rite-ship` is the only DevRites phase that runs actions that cannot be undone
silently. The seal verdict authorizes the *decision*; this prompt authorizes the
*action*.

## The type-GO prompt (render verbatim, wait)

```
About to: <git commit + git push [+ git tag vX.Y.Z | open PR]>
Feature:  <slug>
Verdict:  GO  (seal.md)
Acceptance criteria proven: <n / total>
Branch:   <target branch>

Type "GO" exactly to proceed. Anything else cancels.
```

Rules for the prompt:
- Render it **every time**, even with auto-trigger enabled — this is the last net.
- Only the literal string `GO` (no quotes) proceeds. `y`, `yes`, `go` (lowercase),
  `ok`, `sure`, `do it`, or anything else → cancel, record the cancel in `ship.md`
  as "user declined irreversible step at <ts>", and stop.
- `$rite-autocomplete --ship` (or `--yolo`) is the *only* caller permitted to satisfy
  this prompt automatically. Every other path requires the human's literal `GO`.

## The git ladder

Follow `.agents/skills/devrites-lib/reference/standards/git-workflow.md` and the project's own convention. Do not invent
a release flow the project doesn't use.

0. **Collapse checkpoints** (only if checkpoint mode ran — see
   [rite-build/reference/checkpoint.md](../../rite-build/reference/checkpoint.md)). Local
   `WIP(<slug>)` commits are scratch; fold them into the working tree so the ship is one
   atomic feature commit. `reset --soft` moves only the branch pointer — index and worktree
   are untouched, so no change is ever lost:
   ```bash
   base=$(git rev-parse --abbrev-ref '@{upstream}' 2>/dev/null) || base=main
   mb=$(git merge-base HEAD "$base" 2>/dev/null)
   if [ -n "$mb" ] && git log --format='%s' "$mb..HEAD" | grep -q '^WIP('; then
     git log --format='%s' "$mb..HEAD" | grep -qv '^WIP(' \
       && echo "history since $base mixes WIP + real commits — surface and confirm before collapsing" \
       || git reset --soft "$mb"   # fold every WIP commit since the branch point → staged
   fi
   ```
   No WIP commits (checkpoint mode was off) → this is a silent no-op.
1. **Stage** only the files in `touched-files.md`. Never `git add -A`; never stage
   secrets, `.env`, or out-of-scope files (see the never-commit list in `git-workflow.md`).
2. **Commit** with a Conventional Commit message derived from the feature (`feat(scope):
   …` / `fix(scope): …`). Atomic — one logical change. Put the *why* in the body.
   For non-trivial features, append trailers from the workspace record:
   `Constraint:` from binding principles/irreversible risks, `Rejected:` from material alternatives,
   `Confidence:` from `seal.md`, `Scope-risk:` from the blast-radius scan, and `Not-tested:` for
   known proof gaps. Skip trailers for typos and pure formatting.
3. **Push** to the target branch (the feature branch, or per the project's trunk
   convention).
4. **Tag / PR** only if the project does it: cut a tag when the project tags releases,
   or open a PR when the project reviews via PRs. Otherwise skip — pushing the branch
   is the ship.

Capture the resulting commit SHA(s), branch, and tag/PR URL for `ship.md`.

## Pull request body

When this ship opens a PR, include only applicable sections:

- **Summary** — what shipped and acceptance proven.
- **Risk & rollback** — migration/auth/destructive risk and reversal from `seal.md`.
- **What to scrutinize** — highest-blast-radius review stops.
- **Evidence** — condensed proof plus the archived evidence bundle.

For an agent-owned staged rollout, apply [`rollout.md`](rollout.md). CI-owned deploys
skip that branch. Delete empty sections rather than writing `N/A`.
