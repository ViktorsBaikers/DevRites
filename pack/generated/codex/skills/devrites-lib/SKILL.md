---
name: devrites-lib
description: Internal shared DevRites helper library. Documents cross-cutting engine commands and references; not a user workflow. Do not invoke directly.
user-invocable: false
disable-model-invocation: true
---

# devrites-lib: shared workflow contracts

Do **not** invoke this skill directly.

## Ownership boundary

- The host owns instruction loading, exact-agent dispatch, scheduling, waiting,
  follow-up, results, and history. Skills state the role/result, never native
  fields or dispatch receipts.
- Dispatch every workflow-required role fresh. A missing role stops for HITL;
  never skip, substitute, or perform its work in the root.
- The root owns DevRites state/artifacts but never source/tests. Codex grants it
  workspace permission only because children cannot elevate; writing follows
  [`reference/standards/agents.md`](reference/standards/agents.md).
- The engine owns the retained deterministic structure, atomic-write, install,
  evidence-freshness, and safety primitives only.

## Workspace orientation

Use the supplied slug or read `.devrites/ACTIVE`. Require its authoritative
`state.md`; read it directly, then only needed phase artifacts. Never infer
lifecycle state from chat or optional `README.md`.

## Shared references

- [`reference/standards/agents.md`](reference/standards/agents.md): native custom-agent roles,
  immutable inputs, result contracts, and source-boundary review.
- [`reference/candidate-integrity.md`](reference/candidate-integrity.md): the
  content-bound candidate lifecycle from Build through Ship
  ([`workspace-artifact-schema.md`](reference/workspace-artifact-schema.md)).
- [`reference/reply-contract.md`](reference/reply-contract.md): compact user-facing
  completion states. The host renders the response normally.
- [`reference/visual-playbooks/index.md`](reference/visual-playbooks/index.md): progressive
  visual HTML playbook router (load matching ids only; dual-read outline).

## Deterministic engine surface

The engine is limited to:

- `check readiness` for structure, `check candidate` for content-bound identity,
  and `check seal` for structure plus exact artifact bindings; `secret-scan` and
  `version` remain read-only helpers.
- Atomic state: `state resolve` for answer/drop/batch and transactional
  `state close`.
- Offline, local `install`, `update`, and `uninstall`; their shell/npm callers
  acquire the candidate bundle, source, and binary before invoking the engine.

Exact native agents/checklists own semantics; the host filesystem owns ledger
reads/preview/confirmed no-clobber writes, spec grammar re-reading, question-id
allocation, clarify cursor edits, AFK accounting, recovery accounting, and
read-only diagnostics. Repository scripts and CI own gates.
