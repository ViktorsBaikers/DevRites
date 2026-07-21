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
| Eval ID | Scenario | Expected signal | Gate |
| --- | --- | --- | --- |
| EVAL-001 | <domain-specific case> | <pass condition> | build/seal |

## Guardrails
- Inputs: <validation, prompt-injection boundaries, tenant/data limits>
- Outputs: <schema checks, refusals, human review, citations>
- Privacy/security: <data sent to model, retention, secrets policy>

## Monitoring
- Runtime metrics/logs: <latency, cost, quality, refusal/error rates>
- Alerts or manual review: <trigger>
- Rollback/kill switch: <mechanism>
```
