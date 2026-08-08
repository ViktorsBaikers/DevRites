# One-shot evidence completeness

An action is **consumptive** when a failed attempt is not safely equivalent to a
normal rerun. This includes commands limited to one attempt, commands whose retry
needs fresh human authorization, actions that spend external quota or mutate
privileged/external state so a rerun is not equivalent, and actions whose cleanup
can destroy the failure state needed for diagnosis. Successful cleanup does not
make a consumptive action repeatable.

Workflow-artifact materialization is reversible offline work when every admitted
target has a bound preimage or absence marker, rollback is local to the active
feature workspace, and no privileged/external real action executes. It is not a
consumptive action and must not receive a one-shot authorization budget. Failures
of its materializer, atomic replacement, rollback, or offline proof use the normal
causal-fingerprint recovery cap in `afk-hitl.md`.

## Pre-attempt gate

Before Vet can emit READY, and again immediately before Prove executes the action,
the approved `test-plan.md` must bind all of the following:

1. **Durable retention:** an operator-controlled evidence artifact outside the
   disposable runtime/cleanup tree, created before the first side effect, written
   durably before cleanup, least-privilege, and bounded by schema, size, and
   cardinality.
2. **Trust-safe diagnostics:** known semantic values use the normal validator;
   unknown but lexically well-formed non-secret values survive in bounded sanitized
   fields; malformed, hostile, or secret-bearing values become fixed reason codes
   rather than retained raw input.
3. **Terminal completeness:** every success, nonzero exit, rejection, timeout,
   signal, and cleanup failure either names the retained artifact or proves that
   no diagnostic state exists. Failure retention preserves the original safe
   failure family and cause through clean convergence.
4. **Discriminating proof:** fixtures cover success, a known failure, an unknown
   well-formed failure, malformed/hostile input, and cleanup after failure. They
   prove cleanup cannot delete or overwrite the retained failure evidence.
5. **Causal actionability:** every failure record includes a stable non-secret `boundary_id`
   whose finite map is injective: one retained fingerprint identifies
   one actionable failure seam and correction class. Broad operation/cause labels
   are insufficient when multiple emit sites can produce them. `test-plan.md`
   enumerates every emit site, its boundary ID, expected retained relation, and
   offline decision.
6. **Collision proof:** inject a failure at every mapped seam and require its exact
   boundary ID. Execute a negative mutant that aliases two seams to one retained
   fingerprint and prove the validator/reviewer rejects it.
7. **Recovery sufficiency:** the retained bounded evidence is enough to choose an
   offline correction or a truthful terminal classification without consuming
   another attempt.

Missing or stale evidence is an agent-owned technical plan gap: Vet returns
`NEEDS REPLAN`, and Prove returns to Vet inline without executing the action. Never
weaken the trust validator or spend the attempt merely to discover what the
retention design should have preserved.

## Failure handling

After a consumptive action fails, its retained artifact is the reproduction input.
Do not rerun the action during triage.

Keep two budgets separate:

- **Action authorization:** the failed execution consumes only the authorization
  for that consumptive execution. Zero remaining action attempts prohibits another
  real execution; it does not exhaust offline diagnosis or correction.
- **Causal-fingerprint recovery:** when the retained artifact supplies a new
  Critical/Important failed invariant, the controlling caller immediately runs
  bounded offline triage, repair, fixtures, and narrow Vet in the same invocation.
  Count only no-progress corrections of that exact fingerprint under
  `afk-hitl.md`; do not stop merely because the action authorization was consumed.

Cold resume does not make that fingerprint old or exhausted. Derive its offline
no-progress count from `drift.md` and `evidence.md`; while the count is below the
cap, resume recovery even if a prior writer stored `blocked` / `Next step: none`.
That terminal cursor is valid only for missing retention, a human/safety gate, or
an actually exhausted fingerprint.

A new real attempt is normally admissible only after the affected plan and fixtures
are re-vetted, the failure condition is shown changed, and any required fresh
authorization is obtained. Stop at that authorization boundary; never infer it
from successful offline repair.

## Diagnostic amplification

If a retained artifact is absent or maps one fingerprint to multiple causal
boundaries, do not guess a runtime correction. The fact that evidence from the
past attempt is irretrievable does not prove that a safe future evidence-acquisition
design is unavailable.

When an in-scope trusted seam can add the missing stable discriminator, classify
the ambiguity as an agent-owned **diagnostic-amplification plan gap**. Without
executing the action, repair the diagnostic schema, finite boundary map, per-seam
fault fixtures, cleanup-survival proof, and collision mutant; then run narrow Vet.
Once READY, stop for fresh authorization before exactly one diagnostic-amplification attempt
bound to that vetted design and artifact identity. This exception does not
claim the runtime failure condition changed: the independently proven change is the
evidence-acquisition invariant, and the attempt's acceptance signal is the promised
unique retained boundary (or action success).

Terminal `Next: none` is valid only when no safe in-scope diagnostic-amplification
seam exists, the required change is human/risk/scope owned, or the exact evidence-gap
fingerprint exhausts bounded recovery. A missing old artifact alone is not terminal,
and an amplification attempt never inherits authorization from the failed action.
