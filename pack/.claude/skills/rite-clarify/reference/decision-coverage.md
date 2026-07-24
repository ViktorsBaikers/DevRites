# Decision coverage contract

`decision-coverage.md` proves a topology-first scan ran; reference canonical IDs instead of
duplicating their prose.

## Coverage taxonomy

For each material surface scan: outcome/scope/must-NOT; behavior, permissions, and all states;
domain/data lifecycle/invariants/order/concurrency; interfaces/integrations and failure
contracts; security/privacy/compliance/accessibility/performance/reliability/compatibility;
configuration/observability/support/rollout/rollback; and executable acceptance proof,
environment, credentials/approvals, and evidence limits.

## Status vocabulary

- `closed`: the canonical artifact records the decision.
- `agent-owned`: a reversible technical choice is explicitly delegated to define/vet.
- `not-applicable`: evidence shows the dimension does not apply.
- `deferred-nonblocking`: pre-code evidence is impossible or the item is an explicit
  non-goal; owner and validation gate are named.

`Partial`, `Missing`, open, unknown, or ownerless rows never produce `CLEAR`.

## Artifact shape

```markdown
# Decision coverage

Decision coverage: <CLEAR | NEEDS CLARIFICATION>
Coverage inputs SHA-256: <exact value from `devrites-engine readiness-digest coverage <slug>`>

## Topology
| Surface | Kind | Related IDs | Evidence |
| --- | --- | --- | --- |

## Coverage matrix
| Surface | Dimension | Status | Canonical reference | Owner / validation gate | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |

## Assumption audit
| Assumption | Evidence | Confidence | Owner | Validation | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |

## Residual uncertainty
| Item | Why nonblocking | Owner | Validation gate |
| --- | --- | --- | --- |

## Readiness verdict
No Partial, Missing, unowned material assumption, or unresolved blocking/escalating
question remains.
```

Combine rows only when one decision and owner genuinely close them.

Run the digest command only after `brief.md`, `spec.md`, `decisions.md`,
`assumptions.md`, and `questions.md` are final for this clarification pass, then
copy its complete field line verbatim. The build gate recomputes the digest and
rejects stale coverage. Any open blocking, validating, or escalating question
also prevents `CLEAR`.
