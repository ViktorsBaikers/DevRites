# Reviewer replay

Replays one reviewer role against frozen pre-repair slice diffs whose real findings
are known, and scores recall per role. It prices a change to a reviewer definition,
bundle, or recheck rule at a few hundred role runs instead of a full lifecycle per
data point. It is a measurement tool: it changes no reviewer, gate, or acceptance check.

The repository ships one synthetic case, `cases/example-quota-boundary`, so CI can
prove the grader. Real cases come from your own finished features and stay on your
machine unless you choose to commit them.

## Case format

One directory per case under the cases directory (`--cases`, default `evals/replay/cases`):

| File | Holds |
| --- | --- |
| `diff.patch` | The slice diff exactly as the reviewer first saw it, before any repair |
| `slice.md` | The slice block (goal, write paths) and the acceptance criteria it cites |
| `expected.json` | Role, cluster key, and the findings that were later fixed |

```json
{
  "role": "devrites-code-reviewer",
  "cluster": "<feature>/<SLICE-ID>",
  "findings": [
    {"file": "internal/quota/quota.go", "line": 8, "id": "AC-001", "severity": "Important", "security": false}
  ]
}
```

- `role` names a `<role>.md` file in the agents directory the runner is given.
- `cluster` groups cases that share a feature and slice; the grader's standard error
  treats each cluster as one unit, so two cases cut from one slice are not counted
  as independent evidence.
- Each finding needs `file`, `line` (in the + side of `diff.patch`), and `severity`
  (`Critical`, `Important`, `Suggestion`, `Nit`, `FYI`). `id` (an AC, INV, or
  standard ID the finding cited) and `security` are optional.

## Freeze cases from your own runs

1. Pick a slice whose reviewer return listed `Finding:` rows that a later repair
   fixed. Workspace packets under `.devrites/work/<slug>/` hold the return and,
   when kept, the pre-repair diff; otherwise rebuild it with
   `git diff <slice-base> <pre-repair-commit> -- <write paths>`.
2. Save that diff as `diff.patch`. Never use the repair diff or a later-round return.
3. Copy the slice block and the acceptance criteria it cites into `slice.md`.
4. Label only the findings the repair actually fixed, at their pre-repair `file:line`.
   Leave out findings that were rejected or never addressed; the grader reports
   extra returned findings as unlabeled instead of guessing whether they were valid.
5. Strip secrets, personal paths, and third-party product names from every file.
6. Run `python3 -I scripts/run-reviewer-replay.py --cases <dir> --validate-only`.

Aim for 30-50 cases across several features. With 40 cases and 3 trials, the 95%
interval on a recall difference is roughly +/-10 points, so the replay catches
large regressions, not small ones.

## Run an arm

Each run starts paid native sessions, so a human runs it; CI never does. Build the
candidate's host artifacts first (`bash scripts/build-host-artifacts.sh`), then point
`--agents` at that arm's generated agents directory. Preview the calls with `--dry-run`.

```bash
python3 -I scripts/run-reviewer-replay.py --cases <dir> --arm A --dry-run

python3 -I scripts/run-reviewer-replay.py --cases <dir> \
  --host claude --model '<pinned-model>' --arm A --agents <arm-A>/pack/generated/claude/agents \
  --trials 3 --parallel 6 --max-seconds 600 --max-cost-usd 40 \
  --claude-api-key-file '<0600-key-file>' --out <runs>/A
```

`--host codex` takes `--codex-home` instead of the key file and cost bound. Each
session is tool-free and sees only the role definition, `slice.md`, and
`diff.patch`. The runner allows at most 6 sessions at once and 300 calls per arm,
splits the cost bound evenly per call, and keeps only the `Outcome:`, `Counts:`, and
`Finding:` lines of each return (`<out>/<case>/trial-<n>.md`). `run.json` records
host, build, model, arm, agents and cases digests, and per-trial latency, tokens,
cost, and `timeout`/`host_error` results. Run every arm on the same cases, host,
model, and trial count.

## Grade

```bash
python3 -I scripts/grade-reviewer-replay.py --cases <dir> --run <runs>/A
python3 -I scripts/grade-reviewer-replay.py --cases <dir> --a <runs>/A --b <runs>/A2
python3 -I scripts/grade-reviewer-replay.py --cases <dir> --a <runs>/A --b <runs>/B --aa-noise-floor <half-width>
```

A returned `Finding:` row hits an expected finding when it names the same path (or
a whole-segment suffix of it, such as `quota.go`) with a line within +/-10, or cites
the expected `id`. A trial's recall is hits over expected findings. A return with no
`Finding:` rows scores 0. Rows that hit nothing are counted as `unlabeled`, never as
false positives.

Paired mode reports, per role, both arms' mean recall, the B-A difference, its
clustered standard error and 95% interval, and the correlation between arms. Cases
missing from either arm are listed under `unpaired`. The gate:

1. Grade two same-build arms first (A/A). Their `ci_half_width` is the noise floor.
2. Pass it as `--aa-noise-floor` when grading B against A. Without it the gate
   reads `needs_aa` and never passes.
3. A role passes when the interval's lower bound is at least
   `-max(0.05, aa_noise_floor)` and no must-catch finding regressed: a `Critical`
   or `security` finding that A hits in at least 2 of 3 trials must not fall to 0 in B.
4. A role never passes on fewer than 2 clusters (its standard error is 0 there): it
   reads `insufficient_clusters`. A role with any `unpaired` case reads `incomplete`,
   so a B arm that times out or errors on a case cannot pass by omission.

Report the effect size and interval, never "no change". The replay does not
measure tool use, repository navigation, rechecks, lifecycle outcomes, or precision.
`tests/reviewer-replay-grader-test.sh` proves the matcher and the paired math.
