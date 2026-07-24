# Decisions

## Decision log
| Decision ID | Status | Context | Options | Decision | Consequences | Related IDs |
| --- | --- | --- | --- | --- | --- | --- |
| DEC-001 | accepted | The transactions route already owns caller scoping. | reuse / duplicate | Reuse the route and stream CSV. | One authorization boundary; bounded memory. | REQ-001, REQ-002, REQ-003 |
