# Duplicate code

> Applies when: a review or polish pass needs near-duplicate leads that survive
> renaming, or a dup finding needs a dismissal that persists.

`devrites-engine check dup` is the deterministic scanner. It normalizes source
(strips comments, collapses strings, canonicalizes identifiers and literals),
matches token shingles between files, clusters overlapping matches, and prints
advisory clusters — **leads**, not findings. The engine never judges whether a
cluster is worth merging; that decision is semantic and belongs to the
reviewer. Successful scans exit 0 even with clusters; usage and operational
errors, including incomplete scans, exit 2.

## Running it

```text
devrites-engine check dup [slug] --worktree|--staged|--base <ref>|--all
    [--min-lines n] [--threshold f] [--ignore-file <path>]
    [--limit n] [--exclude <csv>]
```

- `--staged` / `--worktree` / `--base <ref>`: scan the repo, then print only
  clusters with a unit overlapping the diff's changed hunks, marked `*`. Pick
  the mode matching the diff under review: `--base <ref>` when the feature's
  commits are landed (the common `/rite-review` case), `--staged`/`--worktree`
  for uncommitted diffs. `--staged` reads index blobs for every comparator and
  excludes untracked files, regardless of working-tree edits or deletions.
  Keep the index stable during the scan. Other modes read working-tree bytes.
- `--all`: whole-repo triage, includes untracked source files. Use in
  `/rite-polish` or `/devrites-audit` passes.
- `--min-lines` (default 8): minimum meaningful overlap. Lower it for tiny
  helper-heavy languages, raise it to cut boilerplate noise.
- `--threshold` (default 0.6): minimum similarity score per reported pair.
- `--exclude`: extra path prefixes to skip (fixtures, generated code).
- `dup: scanned … units` in the header is the coverage line — a scan of 3
  files in a 300-file repo proves nothing.

## Evidence limits

This is a bounded heuristic, not exhaustive duplicate detection. The scanner
limits eligible files (8,192), source bytes (64 MiB total, 1 MiB per file),
Git listing/diff output (16 MiB), repeated-shingle postings, pair anchors, and
reported clusters. Normalization loses language semantics; a successful scan
with no leads does not prove the repository has no duplication.

No eligible input and intentional generated exclusions are reported explicitly
with exit 0, without claiming a successful source scan. Unreadable/non-regular
source, unresolved index entries, Git failures, and input resource limits exit 2.
A partial scan may print available leads, but never an `ok` verdict or a success
metric. Generated exclusions are counted separately from unavailable input.
Repeated options, conflicting modes, missing/flag-like values, and nonfinite
thresholds fail before scanning or metrics. Diff helpers and text conversion
are disabled so configured executables cannot change the inspected evidence.

## Output shape

```text
dup: scanned 42 files (2 skipped), 17 units; ignore-file .devrites/dup-ignore
dup: 2 cluster(s) (1 ignored)
cluster 1 (a1b2c3d4e5f6) score 0.93:
  * engine/a.go:10-58
    engine/b.go:120-168
cluster 2 (f6e5d4c3b2a1) score 0.81:
    pkg/x/util.py:30-70
    pkg/y/util.py:14-54
```

`*` marks a unit that overlaps the diff's changed lines. The hex after
`cluster N` is the cluster's stable hash.

## Triage each cluster

For every reported cluster a reviewer returns exactly one verdict:

| Verdict | When | Action |
|---|---|---|
| **merge** | Same logic, same contract, no behavioral reason to differ | Finding: extract a shared helper or call the canonical implementation. Name which unit survives. |
| **keep** | Convergent code (test scaffolding, protocol mirrors, intentional fork points) | Record the cluster hash in the ignore ledger with a one-line reason. |
| **watch** | Similar now, likely to diverge — copied seed that is about to change | FYI finding noting the drift risk; do not ignore. |

Ranking boosts distant matches across directories or far apart within one file.
Distance is a triage hint, not proof that two contracts are interchangeable.
Check whether fixes to one copy should also apply to the other.

## Ignore ledger

`.devrites/dup-ignore` (project-root default; `--ignore-file` overrides) holds
one cluster hash per line, `#` comments allowed:

```text
a1b2c3d4e5f6  # test scaffolding mirrors on purpose — do not merge
```

The hash derives from normalized code *content*, not line numbers. It survives
renames, comment churn, literal tweaks, and line moves, but **changes when the
code's structure changes** — a dismissed cluster that gets edited resurfaces
for re-triage. This is intentional: a ledger entry must never permanently
blind the scan.

Rules:

- Never add a hash without reading both units — an unread dismissal is a gap.
- Write the reason on the same line after `#`; a bare hash is an unexplained
  dismissal and counts as a skipped check.
- Do not commit-ignore clusters introduced by the *current* diff — a `*` unit
  needs a verdict (merge/keep/watch), not suppression.

## Reporting dup findings

Dup findings use the C2 shape like any other finding, plus the cluster hash so
the dismissal ledger can reference it:

```text
Finding: Suggestion | engine/a.go:10 | cluster a1b2c3d4e5f6 duplicates engine/b.go:120 (score 0.93) | fixes to one will drift from the other | extract shared helper `dupNormalize` into engine/internal/lib
```

A reviewer that ran `check dup` and found nothing reports it under
`No-findings:` / `Basis:` (`dup: 0 clusters over 42 files`) — not by silence.
