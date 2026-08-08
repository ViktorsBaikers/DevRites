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

A new real attempt is admissible only after the affected plan and fixtures are
re-vetted, the failure condition is shown changed, and any required fresh
authorization is obtained. Stop at that authorization boundary; never infer it
from successful offline repair. If the attempt ran without complete retention,
stop with the observed technical blocker because a blind retry cannot manufacture
the destroyed evidence.
