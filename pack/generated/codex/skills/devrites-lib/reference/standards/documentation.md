# Documentation

Document intent/decisions; public inputs, outputs, errors, and gotchas; non-obvious
constraints; and real build/test/run commands. Update with behavior; prefer one runnable
example. Do not restate code or types.

## Record decisions

- Capture significant context, decision, consequences, accepted trade-off, change trigger,
  and why viable alternatives lost. DevRites uses `decisions.md` for feature decisions.
- ADRs move `PROPOSED → ACCEPTED → SUPERSEDED / DEPRECATED`. Preserve accepted history;
  a replacement ADR cites and supersedes the old one rather than rewriting it.

## Promote durable guidance

Promotion is maintenance of an existing authority, not a new memory system.

1. **Trigger:** the same reviewed correction appears in at least two distinct features, or
   one explicit product/architecture decision has durable rationale. A one-off, generic,
   stale, or merely inferred observation does not trigger promotion.
2. **Ground:** verify each current claim against live authoritative repository sources.
   Report the source and currentness signal. Unverifiable means `unknown`, not false.
3. **Scope:** state when the guidance applies and does not apply. Reject a candidate whose
   observable trigger cannot be named.
4. **Own and expose:** choose one existing canonical owner (`AGENTS.md`/`CLAUDE.md`, a scoped
   standard, or an ADR) and name the phases, agents, or contributors that discover it and
   how (direct read, index link, or existing on-demand route).
5. **Reconcile:** search current guidance for duplicates, contradictions, and supersession.
   Update, narrow, replace, or retire contradicted guidance at its owner; do not append a
   competing rule. Record the conflict/retirement disposition.
6. **Approve:** show evidence and the exact durable edit before writing; user approval is
   required. Never create a learning ledger, index, queue, score, or parallel authority.

Long reference material stays behind its existing on-demand route.
