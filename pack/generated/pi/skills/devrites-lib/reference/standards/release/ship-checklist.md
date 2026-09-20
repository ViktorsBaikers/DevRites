# Ship checklist

> Applies when: the final pass/fail sweep at ship.

- Current `seal.md` GO; literal GO approved this attempt.
- Collapse/Git follow project convention.
- All strategy/decisions/review/seal residuals: one `ship.md` tracked path/ID or prior/explicit human-approved no-action; gaps block archive.
- Affected-surface matrix recorded: every dimension the change could touch (surface, OS, environment, data state) is marked `verified | fixture-covered | not-run:<reason> | n/a` — an unlisted dimension is an unexamined one.
- Attached or committed artifacts are sanitized: no secrets, personal paths, or machine-local identifiers in logs, captures, or fixtures.
- Success: archive; clear `ACTIVE`.

[`git-workflow.md`](../git-workflow.md), [`ci-cd.md`](../ci-cd.md), [`git-ship.md`](../../../../rite-ship/reference/git-ship.md).
