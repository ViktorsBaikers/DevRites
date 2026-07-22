# Cleanup + classify

## Classify the failure (for the report)

- `test-right / code-wrong` (fix code)
- `test-wrong` (possible **spec drift** → `/rite-plan repair`)
- `environment / setup`
- `flaky / ordering`
- `external-dependency-down` (blocker)

## Cleanup checklist: required before declaring done

- [ ] Original repro no longer reproduces (re-run the Phase 1 loop).
- [ ] Regression test passes (or absence of seam is documented).
- [ ] All `[DEBUG-...]` instrumentation removed (`grep` the prefix).
- [ ] Throwaway harnesses deleted (or moved to a clearly marked debug location).
- [ ] The correct hypothesis is stated in `evidence.md` + the commit/PR message: next debugger learns.
