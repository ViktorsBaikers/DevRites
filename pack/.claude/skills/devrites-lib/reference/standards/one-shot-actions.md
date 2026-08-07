# One-shot evidence completeness

An action is **consumptive** when a failed attempt is not safely equivalent to a
normal rerun. This includes commands limited to one attempt, commands whose retry
needs fresh human authorization, actions that spend external quota or mutate
privileged/external state so a rerun is not equivalent, and actions whose cleanup
can destroy the failure state needed for diagnosis. Successful cleanup does not
make a consumptive action repeatable.

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
5. **Recovery sufficiency:** the retained bounded evidence is enough to choose an
   offline correction or a truthful terminal classification without consuming
   another attempt.

Missing or stale evidence is an agent-owned technical plan gap: Vet returns
`NEEDS REPLAN`, and Prove returns to Vet inline without executing the action. Never
weaken the trust validator or spend the attempt merely to discover what the
retention design should have preserved.

## Failure handling

After a consumptive action fails, its retained artifact is the reproduction input.
Do not rerun the action during triage. A new attempt is admissible only after the
evidence identifies a changed invariant or external condition, the affected plan
and fixtures are re-vetted, and any required fresh authorization is obtained. If
the attempt ran without complete retention, stop with the observed technical
blocker; a blind retry cannot manufacture the destroyed evidence.
