# `ai-spec.md` template

Create this annex only when the feature touches model calls, RAG, agents, evals,
or LLM output.

```markdown
# AI-SPEC: <Feature>

## AI surface
| Surface | Purpose | Owner |
| --- | --- | --- |
| <model call/RAG/agent/eval> | <why it exists> | <module/team> |

## Model/runtime choice
- Framework/model/provider: <choice or project default>
- Settings that matter: <temperature, tools, retrieval scope, limits>
- Fallback/degradation: <what users see when AI is unavailable>

## Domain evals
| Eval ID | Scenario / slice | Dataset provenance | Expected signal + threshold | Gate |
| --- | --- | --- | --- | --- |
| EVAL-001 | <representative/adversarial/empty-context case> | <held-out source/version> | <pass condition and baseline delta> | build/seal |

For RAG, cover retrieval relevance/context recall, context precision, answer
faithfulness, citation support, tenant/ACL isolation, poisoned or conflicting documents,
and insufficient-context fallback as applicable. A single polished example is not an eval.

## Guardrails
- Inputs: <validation, prompt-injection boundaries, tenant/data limits>
- Retrieval: <source provenance, indexing validation, tenant/ACL filter, freshness/deletion>
- Outputs: <schema checks, refusals, human review, citations only to supporting retrieved sources>
- Privacy/security: <data sent to model, retention, secrets policy>
- Unknown/insufficient context: <abstain, clarify, or bounded fallback; never fabricate>

## Monitoring
- Runtime metrics/logs: <latency, cost, retrieval/quality, refusal/error/fallback rates>
- Alerts or manual review: <trigger>
- Rollback/kill switch: <prompt/model/index version and reversible mechanism>
- Drift evaluation: <reference set, schedule/trigger, regression threshold, owner>
```
