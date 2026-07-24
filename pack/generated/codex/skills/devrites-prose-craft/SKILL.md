---
name: devrites-prose-craft
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- For every DevRites specialist or writer dispatch, first call `spawn_agent` with the named `devrites-<role>` custom role. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If `spawn_agent` is callable but a named read-only role is unavailable, use generic `explorer` only when the host proves that run has a runtime-enforced read-only sandbox. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. A missing read-only custom role is not evidence that spawning is unavailable.
- Never dispatch generic `worker` for `devrites-slice-wright` unless the host proves that worker run carries exact DevRites identity and the same `.wright-allowlist` enforcement as the named role. Codex reports a generic run as `agent_type=worker`, so the generated global hooks cannot prove that binding. Reject that unsafe rung and use the documented labelled inline wright path with `.reconcile-inline` plus the full reconcile gate.
- If the host cannot prove the generic explorer is runtime read-only, reject that rung too. Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, and apply every fallback risk gate. An unbound generic wright or unconfined generic explorer is such a safety rejection, not evidence that no agents exist.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-prose-craft: prose that reads human

Write for a teammate who will rely on the result. Remove model-shaped filler without sanding
off the author's voice or weakening technical content.

## When this fires
- A text-generating phase composes an artifact: `$rite-spec` (overview, rationale),
  `$rite-define` / `$rite-plan` (plan narrative), `$rite-temper` / `$rite-vet` (review prose),
  `$rite-review` / `$rite-seal` (findings + verdict prose), `$rite-ship` (commit/PR body),
  `devrites-doubt` / `rite-handoff` (notes).
- Any phase composes a substantive **user-facing reply**. Deterministic progress footers are
  script-rendered and exact by design.
- `$rite-polish` Phase 1 as the **catch** pass on prose that slipped through at write time.

## Two modes
- **Rewrite (default).** When DevRites writes the artifact/reply or polishes it, fix the prose
  in place.
- **Detect-only.** When auditing prose you shouldn't silently change, such as a user's existing
  `spec.md` at `$rite-adopt` or text under `$rite-review`, list the tells with quotes and leave
  the text untouched. Mirrors `devrites-audit`'s read-only stance.

Order findings by severity: **P0** credibility-killers (vague attribution, a marketing
adjective standing in for evidence, a false/unsourced claim) → **P1** obvious tells (negative
parallelism, filler openers, em-dash tics) → **P2** polish (rhythm, word choice). A quick pass
fixes P0 + P1 and stops; a full pass takes P2 too.

## Process

1. **Calibrate.** Choose rewrite or detect-only mode, split prose from technical sections, and
   set the voice baseline: an explicit user sample first, then neighboring project prose, then
   the source text itself. Fall back to flat, direct prose. Done when every section has a
   register and one voice baseline.
2. **Protect.** Apply the preservation contract in
   [`prose-style.md`](../devrites-lib/reference/standards/prose-style.md) before changing words.
   Scan for clusters of tells; one isolated marker is not a verdict. Done when every claim,
   constraint, identifier, example, quotation, and deliberate uncertainty is accounted for.
3. **Rewrite.** Remove P0/P1 tells and P2 when the caller asked for a full pass. Match the
   baseline's vocabulary and cadence without inventing facts, opinions, or quirks. Done when the
   same reader can make the same decisions from the rewrite as from the source.
4. **Audit.** Compare source and result, read the prose aloud, and run the relevant references
   below. Done when the preservation contract passes, the technical register remains exact,
   and no in-scope tell remains.

## References

- Always read [`prose-style.md`](../devrites-lib/reference/standards/prose-style.md) for the two
  registers, preservation contract, and core checks.
- Load [`reference/banned-phrases.md`](reference/banned-phrases.md) when word choice or tone is
  the problem.
- Load [`reference/structures.md`](reference/structures.md) when sentence shape, rhythm, or
  formatting is the problem.
- Load [`reference/examples.md`](reference/examples.md) when the target artifact's shape is
  unclear.

## Boundaries

This skill edits language, not facts. When style and fidelity conflict, fidelity wins.

## Output

- **Rewrite:** replace the target prose; add no editing commentary unless the caller's contract
  asks for it.
- **Detect-only:** return ordered P0/P1/P2 findings with the exact phrase and a concrete fix.
