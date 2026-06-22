---
name: devrites-audit
description: Read-only audit dispatch for the active feature on the requested axis — security (OWASP, trust boundary, secrets), perf (measure-first, N+1, CWV), or simplify (Chesterton's Fence, deletion test). Use when the user says "security review", "is this safe", "is this fast enough", "perf check", "N+1", "simplify this", "Chesterton's Fence". Not for write actions or whole-project audits.
argument-hint: "<security | perf | simplify>"
user-invocable: false
---

# devrites-audit — read-only audit dispatch

Dispatch one read-only review subagent against the active feature's workspace + diff. The subagent runs in **fresh context** (no author anchoring) and returns labeled findings. The caller (`/rite-polish` Phase 1, `/rite-review`, or the user) acts on them — this skill returns the subagent's report verbatim.

This is the **inline single-axis pass** used during build / polish — one axis at a time, on demand, where a quick read keeps a slice honest. It is intentionally distinct from the seal/review **gate**, where the reviewer agents fan out in parallel across all relevant axes in their own fresh contexts (see `/rite-seal`). Same agents, different role: the audit is a cheap mid-flight check; the seal fan-out is the blocking gate. Both reading the same agent disciplines is the point, not a divergence.

Why a subagent rather than inline: an adversarial reviewer with no author context is more likely to find what's wrong. Anthropic bug [#49559](https://github.com/anthropics/claude-code/issues/49559) can leave `context: fork` silently inline, so `Task` dispatch is the reliable path.

## Axis selection

`$ARGUMENTS` picks the axis. If the caller did not pass one, infer from intent and confirm with the user before dispatch.

| Axis | Subagent (`.claude/agents/`) | Discipline |
|---|---|---|
| `security` | `devrites-security-auditor` | OWASP Top 10; three-tier trust boundary (untrusted → boundary → trusted); secrets handling; dependency risk. A real auth-bypass / data-exposure / injection is **Critical → NO-GO** at seal. |
| `perf` | `devrites-performance-reviewer` | Measure-first: no claim without a number or a specified measurement. N+1s, hot-path work, payload/bundle size, Core Web Vitals risks. Breach of a stated `spec.md` budget is **Important/Critical**. |
| `simplify` | `devrites-simplifier-reviewer` | Behavior-preserving simplification: guard clauses, Extract Method, simplify conditionals, the deletion-test heuristic, Chesterton's Fence; plus the AI-codegen over-engineering smells — single-use factory / needless indirection, defensive try-catch bloat + redundant logging, dependency creep where an in-repo option exists, a 100-line function where 20 would do. Findings are **Suggestion / Nit / FYI** — no behavior change. |

## Gather

1. Read `.devrites/ACTIVE` to resolve the active feature `<slug>`.
2. Confirm `.devrites/work/<slug>/touched-files.md` and `spec.md` exist. If missing → **STOP** and tell the caller the feature has no recorded diff or spec yet.

## Dispatch

Use the `Task` tool to launch the chosen subagent with this prompt shape (axis-specific reads in `Read:`):

```
Audit the active DevRites feature on the <axis> axis.

Workspace: .devrites/work/<slug>/
Read:
  - spec.md   (acceptance criteria; for perf: any perf budget; for security: data model + affected areas)
  - decisions.md   (if present)
  - evidence.md    (existing measurements, for perf)
  - touched-files.md
Run `git diff` and read the listed touched files. Apply your documented
discipline and return labeled findings (Critical / Important / Suggestion /
Nit / FYI) using your documented output format. ONE FINDING PER LINE,
cite file:line.

Feature scope only. No edits. Do not summarize or re-rank — the caller
reconciles.
```

Rules for the dispatch:

- **One subagent per call.** This skill is not a fan-out; multi-axis fan-out is `/rite-seal`'s job (see `.claude/skills/rite-seal/reference/parallel-dispatch.md`).
- **No author context.** Do not pass the caller's analysis or framing of the change to the subagent — fresh, adversarial read is the point.
- **No cross-pollination.** If the caller wants more than one axis, dispatch each axis in its own `Task` call in a single message so the runtime parallelizes; each subagent gets only its own brief.

## Return

Pass the subagent's findings report back to the caller **verbatim**. Do not re-label, re-rank, or summarize. The caller (`/rite-polish` for `simplify`, `/rite-review` for `security`/`perf`) decides what to act on within feature scope, and surfaces any **Critical** to `/rite-seal` as a NO-GO blocker.

## Fallback

If the `Task` tool is unavailable in the current environment, fall back to a read-only inline audit using the discipline documented in the corresponding agent file (`.claude/agents/devrites-{security-auditor,performance-reviewer,simplifier-reviewer}.md`). **Flag clearly that this was an inline fallback**, not an independent review. The seal weighs the fallback differently — see [`rite-seal/reference/risk-and-rollback.md`](../rite-seal/reference/risk-and-rollback.md).

## Scope reminders

- Active feature + touched files only. Out-of-scope risks become FYI follow-ups.
- Read-only. No edits.
- For `perf`: no number → no claim. Speculative micro-opts are Suggestion at most.
- For `simplify`: behavior-preserving only. Anything that needs new tests is out of scope here — route to `/rite-plan reslice`.
- Critical findings block the seal.
