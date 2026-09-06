# Fix + regression test

Write the regression test **before the fix**, *but only if there is a correct
seam for it*.

A correct seam exercises the **real failure pattern** as it occurs at the call
site. If the only available seam is too shallow (unit test that can't replicate
the chain that triggered the failure), a regression test there gives false
confidence.

## No correct seam? That itself is the finding

Note it in `evidence.md` and append a follow-up in the active feature's
`decisions.md` (or run `$rite-plan repair` if the spec is affected): the
codebase architecture is preventing this class of failure from being locked
down. Frame it as a *deepening opportunity* ("this module needs a seam at <X>
so this failure can be regression-tested") so the next `$rite-plan` repair or
architecture cleanup pass has a concrete target.

Do **not** invent an artificial seam just to host a test: a shallow seam that
doesn't exercise the real call chain gives false confidence and is worse than
no test at all.

## When a correct seam exists

1. Turn the minimised repro into a failing test at that seam.
2. Watch it fail.
3. Apply the fix.
4. Watch it pass.
5. Re-run the Phase 1 loop against the original (un-minimised) scenario.
