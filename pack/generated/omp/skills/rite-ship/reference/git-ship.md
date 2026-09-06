# The irreversible ship: type-GO + git ladder

`/rite-ship` alone runs irreversible actions. A seal GO is never authorization for Git.
Require fresh approval for one disclosed attempt plus native permission.

## The type-GO prompt (render verbatim, wait)

```
Feature:  <slug>
Verdict:  GO  (seal.md)
Acceptance criteria proven: <n / total>
Branch:   <target branch>
Candidate paths: <exact project-relative paths>
Checkpoint collapse: <exact `git reset --soft '<merge-base>'` command | skip>
Stage plan: <exact argv-safe `git add -- '<path>'` command for every manifest row>
Commit:   <exact git commit command>
Push:     <exact git push command | skip>
Tag:      <exact git tag command | skip>
PR:       <exact PR command, target, title, and body source | skip>

Type "GO" exactly to approve this attempt once. Anything else cancels.
```

Rules:
- Render every attempt; autocomplete flags only reach this boundary.
- Only literal `GO` proceeds. Anything else cancels, records
  `user declined irreversible step at <ts>` in `ship.md`, and stops.
- `GO` is one-use for the shown scope/commands; changed/retried plans re-prompt.
- Prior answers, AFK, seal GO, and native approval do not replace type-GO;
  type-GO never replaces/bypasses native permission or sandbox approval.

## The candidate-safe Git ladder

Follow `standards/git-workflow.md` and project convention; invent no release flow.
Candidate identity always comes from `devrites-engine check candidate`; do not
implement a Git-side hash.

### Before type-GO

Pre-GO is read-only. Do not run the disclosed collapse or staging commands.

0. **Protect existing index work.** Read the NUL-delimited staged state/path set.
   Refuse a pre-existing staged path outside the manifest; never unstage, stage,
   reset, or discard user work.
1. **Analyze checkpoint history without mutating it.** Read the upstream, merge
   base, commit OIDs, and full subjects. If no checkpoint commit exists, disclose
   `skip`. A collapse is eligible only when every commit in its range has the exact
   `WIP(<active-slug>):` prefix for the validated active slug. A different slug,
   missing colon, broad `WIP(` match, ordinary commit, or mixed history blocks;
   never reinterpret the range.
2. **Verify the candidate.** Rerun `devrites-engine check candidate <slug>` and
   `devrites-engine check seal <slug>`, compare every binding, require the
   candidate worktree paths to be unchanged, and inspect the existing staged
   scope. These checks do not prepare the index.
3. **Prepare the disclosure.** Build argv-safe commands from literal validated
   paths, without `eval` or command-string interpolation. Run only read-only
   secret scans of the existing staged state and proposed PR body. Render the
   literal prompt with the exact optional collapse command or `skip`, exact stage
   plan, commit, push, tag, and PR commands; then wait.

### After type-GO

1. **Optional checkpoint collapse.** If and only if the prompt disclosed it,
   recheck that `HEAD`, merge base, and every subject are unchanged and that each
   subject starts exactly `WIP(<slug>):` for the active slug, then run the exact
   disclosed `git reset --soft "$mb"`. Otherwise skip. Any different or mixed
   subject blocks without mutation.
2. **Exact staging.** Run `git add -- "$path"` once for each disclosed manifest
   row, passing each validated project-relative path as its own quoted argv
   element. Use no glob, directory, `-A`, `.env`, secret, out-of-scope path,
   `eval`, or interpolated command string.
3. **Compare staged scope.** Compare the NUL-delimited output of
   `git diff --cached --name-status --no-renames -z` to the manifest's exact
   state/path set. `present` maps only to `A` or `M`; `deleted` maps only to `D`.
   Reject missing, extra, renamed, duplicated, or differently classified paths.
4. **Compare staged bytes.** Require no index-to-worktree difference for any
   manifest path. A dirty manifest path stops; never stage or reinterpret it
   after this comparison.
5. **Revalidate immediately before commit.** With no intervening mutation,
   rerun `devrites-engine check candidate <slug>` and
   `devrites-engine check seal <slug>`, require the exact candidate digest and
   every evidence/browser/review/Seal binding, then run the staged secret scan.
   Any scope, byte, identity, binding, scan, history, or command drift invalidates
   the one-use approval. Stop and render a fresh type-GO prompt; do not repair the
   drift under the old approval.
6. **Commit** one logical change using Conventional Commit and a why-body.
   Non-trivial work adds applicable `Constraint:`, `Rejected:`, `Confidence:`,
   `Scope-risk:`, and `Not-tested:` workspace trailers; trivial edits omit them.
7. **Verify the commit.** Immediately after the authorized commit and before any
   push or tag, compare
   `git diff-tree --root --no-commit-id --name-status --no-renames -r -z HEAD`
   to the manifest using the same exact state/path mapping. Require candidate
   paths still match `HEAD`, then rerun `devrites-engine check candidate <slug>`
   and `devrites-engine check seal <slug>` and compare the exact digest again.
   Any mismatch stops; do not reinterpret it.
8. **Push** to the project-conventional target branch.
9. **Tag / PR** only when project convention requires it; otherwise skip.

Record SHA(s), branch, and tag/PR URL in `ship.md`.

## Pull request body

When this ship opens a PR, include only applicable sections:

- **Summary:** what shipped and acceptance proven.
- **Risk & rollback:** migration/auth/destructive risk and reversal from `seal.md`.
- **What to scrutinize:** highest-blast-radius review stops.
- **Evidence:** condensed proof plus the archived evidence bundle.

For an agent-owned staged rollout, apply [`rollout.md`](rollout.md). CI-owned deploys
skip that branch. Delete empty sections rather than writing `N/A`.
