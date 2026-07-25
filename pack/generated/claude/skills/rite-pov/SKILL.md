---
name: rite-pov
description: Project-grounded verdict for adopting, switching, rejecting, or revisiting a named external technology, library, platform, CVE, or pattern. Use when deciding whether this project should commit to an outside option.
argument-hint: "[candidate/link/question]"
user-invocable: true
required-agent-roles: none
---

# /rite-pov: project-grounded external verdict

Decide whether **this project** should adopt, trial, hold, reject, or ignore a named outside candidate. Chat verdict first; durable record only when the user asks or the decision changes an active feature.

## Rules consulted
Step 0: Read `.claude/skills/devrites-lib/reference/standards/core.md`. Pull `source-driven`, `security`, or `deprecation` standards only when the candidate touches those risks.

## Operating rules
- Verdict only after two floors clear: one verified project fact and one verified external source.
- Named candidate only. If the user asks "what should we use?" over an open field, route to `/rite-pressure-test` or `/rite-spec` to surface criteria first.
- Reversibility sizes rigor: two-way config/dependency < bounded internal migration < public/security/legal/data decision.

## Workflow
1. **Frame.** Parse `$ARGUMENTS` into candidate, intent (`adopt|switch|compare|CVE/deprecation impact|second opinion`), and reversibility tier. If intent is ambiguous, ask one blocking question before research.
   **Completion:** candidate, decision intent, and reversibility tier are explicit.
2. **Project floor.** Verify at least one concrete project fact: incumbent dependency/call site, current absence plus integration point, prior ADR/decision, or affected surface. Use `devrites-engine profile get`; on `MISS`, run `devrites-engine profile refresh` once, then inspect candidate-specific files fresh. Completion: at least one `file:line` or local doc pointer is in notes, or the verdict is `Hold: project floor missing`.
3. **External floor.** Read primary docs/advisory/release notes/source for the candidate. Prefer official sources; web summaries are supporting only. Completion: at least one dated source URL/title is in notes, or the verdict is `Hold: external floor missing`.
4. **Compare.** Weigh fit, migration cost, reversibility, project principles, security/licensing/deprecation risk, and simpler alternatives already present.
   **Completion:** every comparison dimension has project evidence or an explicit unknown.
5. **Verdict.** Return exactly one grade: `Adopt`, `Trial`, `Hold`, `Reject`, or `Not-our-problem`. Include next step: `/rite-spec`, `/rite-define`, spike, `/rite-learn` record, or done.
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
