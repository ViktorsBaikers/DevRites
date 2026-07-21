# Menu reference

Phase-ordered command reference for `/rite`. Load only if the user wants detail on
what each command does or how phases connect.

## Phases and commands

| Phase | Command | Use when |
|---|---|---|
| Spec | `/rite-spec <feature>` | **New feature.** Investigate deeply → write spec.md. Asks with options; gathers attached design references (optional). |
| Adopt | `/rite-adopt` | Onboard an existing codebase instead of starting fresh — reverse-derive spec.md + seed the conventions ledger. |
| Temper | `/rite-temper` | _Optional, before define._ Strategic review of the spec — scope mode (expand/selective/hold-rigor/reduce) + pre-mortem; hardens the spec. Best on big/risky features; mandatory in `/rite-autocomplete`. |
| Plan | `/rite-define` | Turn the approved spec into plan + vertical task slices + state. |
| Vet | `/rite-vet` | _Required before build._ Review every plan — scope · architecture · tests · perf; light for simple/reversible, full for high stakes. |
| Re-plan | `/rite-plan` | The active plan is too big, wrong, stale, ambiguous, or blocked. |
| Build | `/rite-build` | Implement the next single vertical slice. Stops after one slice. |
| Converge | `/rite-converge` | _Recovery._ Code drifted from / falls short of intent (resumed cold, adopted, stalled build) — assess live code vs spec/plan/tasks and append the remaining work as new slices for `/rite-build`. |
| Prove | `/rite-prove` | Prove the current scope: tests, build, runtime, browser evidence. |
| Polish | `/rite-polish` | Code polish always; UI normalize + ship-quality polish if UI is in scope. Modes: `bolder/quieter/distill/harden/normalize-only`. |
| Review | `/rite-review` | Feature-scoped review before sealing. |
| Seal | `/rite-seal` | Final GO / NO-GO decision (no git). |
| Ship | `/rite-ship` | Type-GO → commit/push/tag, then archive the task + clear ACTIVE. |
| Status | `/rite-status` | See where the active feature stands. |
| Doctor | `/rite-doctor` | Health check — install integrity, stale ACTIVE, orphaned gates, hook wiring, merge/rebase state. |
| Learn | `/rite-learn` | Review the captured learning ledger → promote recurring lessons to project rules / principles. |

## Typical orderings

- **Every feature**: `/rite-spec` (spec) → *(big feature? `/rite-temper` — strategic review)* →
  `/rite-define` (plan) → `/rite-vet` (engineering review; light or full) →
  `/rite-build` ×N (all slices) → `/rite-prove` (once all built) →
  `/rite-polish` (always: code + UI if UI) → `/rite-review` → `/rite-seal` → `/rite-ship`.
- **Existing codebase**: `/rite-adopt` → `/rite-define` → `/rite-vet` → build.
- **Drift mid-build**: stop → drift question → `/rite-plan` (repair) → resume build.
- **Resumed / adopted / stalled**: `/rite-converge` (assess live code vs intent → append the
  remaining slices) → `/rite-build` ×N → continue at `/rite-prove`.

## Rules this menu obeys

- `/rite` never edits code or runs a phase workflow.
- Menu mode runs `devrites-engine first-task`; `/rite-status` owns workspace status.
- It suggests; the user (or Claude, when appropriate) invokes the real skill.
