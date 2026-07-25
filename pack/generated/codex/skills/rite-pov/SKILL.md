---
name: rite-pov
description: Project-grounded verdict for adopting, switching, rejecting, or revisiting a named external technology, library, platform, CVE, or pattern. Use when deciding whether this project should commit to an outside option.
argument-hint: "[candidate/link/question]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-pov: project-grounded external verdict

Decide whether **this project** should adopt, trial, hold, reject, or ignore a named outside candidate. Chat verdict first; durable record only when the user asks or the decision changes an active feature.

## Rules consulted
Step 0: Read `.agents/skills/devrites-lib/reference/standards/core.md`. Pull `source-driven`, `security`, or `deprecation` standards only when the candidate touches those risks.

## Operating rules
- Verdict only after two floors clear: one verified project fact and one verified external source.
- Named candidate only. If the user asks "what should we use?" over an open field, route to `$rite-pressure-test` or `$rite-spec` to surface criteria first.
- Reversibility sizes rigor: two-way config/dependency < bounded internal migration < public/security/legal/data decision.

## Workflow
1. **Frame.** Parse `$ARGUMENTS` into candidate, intent (`adopt|switch|compare|CVE/deprecation impact|second opinion`), and reversibility tier. If intent is ambiguous, ask one blocking question before research.
   **Completion:** candidate, decision intent, and reversibility tier are explicit.
2. **Project floor.** Verify at least one concrete project fact: incumbent dependency/call site, current absence plus integration point, prior ADR/decision, or affected surface. Use `devrites-engine profile get`; on `MISS`, run `devrites-engine profile refresh` once, then inspect candidate-specific files fresh. Completion: at least one `file:line` or local doc pointer is in notes, or the verdict is `Hold: project floor missing`.
3. **External floor.** Read primary docs/advisory/release notes/source for the candidate. Prefer official sources; web summaries are supporting only. Completion: at least one dated source URL/title is in notes, or the verdict is `Hold: external floor missing`.
4. **Compare.** Weigh fit, migration cost, reversibility, project principles, security/licensing/deprecation risk, and simpler alternatives already present.
   **Completion:** every comparison dimension has project evidence or an explicit unknown.
5. **Verdict.** Return exactly one grade: `Adopt`, `Trial`, `Hold`, `Reject`, or `Not-our-problem`. Include next step: `$rite-spec`, `$rite-define`, spike, `$rite-learn` record, or done.
6. **Optional record.** If the user asks to persist, append the decision to the active workspace `decisions.md`; if no active workspace, suggest an ADR. A persisted `Reject` also lands in the cross-feature ledger so ideation skills stop re-proposing it: `devrites-engine learnings add <slug> "<candidate> — <why rejected>" rejected-direction`.

## Output
Reply-contract exception: decision utility; may run outside a workspace.

```
Verdict: <Adopt|Trial|Hold|Reject|Not-our-problem> — <one sentence>
Tier: <two-way|bounded-one-way|high-stakes>
Project floor: <file:line/local doc>
External floor: <source>
Why: <3 bullets max>
Next: <single command or done>
Record: <path|not written>
```

## Gotchas
- No project floor, no verdict. A generic "X is good" answer is failure.
- Do not let a strong blog post compensate for no local call site, incumbent, or integration point.
- Do not enumerate a whole market here; bounded candidate decisions only.
