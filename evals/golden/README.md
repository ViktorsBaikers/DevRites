# Golden outcome fixtures

`shippable-feature/` is the only canonical base. It contains every non-empty
artifact required by the current workflow manifest, valid readiness digests,
and the final-only seal and review files.

`run-outcome-evals.sh` copies that base for each negative row and proves that
exactly one causal path changed. The Go seal gate supplies the frozen DIAG v1
reason ID. The shell grader keeps only final semantic checks that the structural
gate does not own: GO verdict, checked and matching acceptance IDs, blockers,
review, open questions, and final phase/status.

Build readiness accepts only `vet` or `build`, while a finished outcome uses a
shippable phase. The runner therefore creates a temporary `build` projection
from the same canonical base before testing coverage and engineering digest
drift. Evidence freshness also runs separately with controlled temporary
mtimes; Git does not preserve useful proof timestamps.

`blocked-feature/` and `near-miss-unproven-ac/` remain compatibility examples.
They must stay NO-GO, but their overlapping failures are not matrix proof.
