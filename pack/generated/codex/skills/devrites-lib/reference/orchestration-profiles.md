# Native execution profiles

Profiles scale **change depth**, not orchestration machinery. The host discovers
the exact project specialist named by a workflow, dispatches it in fresh context,
waits for the result, validates the result shape, and reconciles the evidence.
DevRites does not provide an engine broker, agent API version, tier protocol,
receipt, polling loop, or fallback agent.

| Profile | Use when | Depth |
|---|---|---|
| **Quick** | One small, reversible, unambiguous change | One bounded implementation pass and focused proof; escape to Standard as soon as the significance gate fails. |
| **Standard** | Default feature work | Full required lifecycle, named independent passes, focused then feature-level proof. |
| **Full** | Security, migration, public contracts, high blast radius, or explicit `--full` | Same phases and exact roles as Standard, with every applicable conditional axis, broader failure-mode proof, and stricter final review. |

A profile never skips a workflow-named specialist, changes source-write
ownership, relaxes a gate, or authorizes an irreversible action. Missing or
incompatible exact roles stop for HITL; the root never substitutes a generic
agent or performs the specialist's work inline.

The complete 17-role catalog and source-writing boundary live in
[`standards/agents.md`](standards/agents.md). Review-specific applicability
lives in [`parallel-dispatch.md`](parallel-dispatch.md).
