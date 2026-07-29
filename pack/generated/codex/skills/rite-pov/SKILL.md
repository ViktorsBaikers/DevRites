---
name: rite-pov
description: Project-grounded verdict for adopting, switching, rejecting, or revisiting a named external technology, library, platform, CVE, or pattern. Use when deciding whether this project should commit to an outside option.
argument-hint: "[candidate/link/question]"
user-invocable: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. A missing visible `agent_type` field is still V2—not capability loss, V1, or HITL—so send it anyway. If the named call rejects it, stop before any generic/default spawn. Codex loads the role TOML's `developer_instructions` natively; DevRites verifies the durable rollout, wait, completion, and delivered result.
- Only after the runtime explicitly identifies MultiAgent V1, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On explicitly identified MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If the required dispatch for the explicitly identified runtime is unavailable or rejected, stop for HITL. Never switch runtime lanes. Never execute a DevRites specialist role in the root context.
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
