# Menu reference

Phase-ordered command reference for `$rite`. Load only if the user wants detail on
what each command does or how phases connect.

## Phases and commands

| Phase | Command | Use when |
|---|---|---|
| Spec | `$rite-spec <feature>` | **Start here.** Investigate deeply → write spec.md. Asks you with options; gathers any design references you attach (optional). |
| Adopt | `$rite-adopt` | Onboard an existing codebase instead of starting fresh — reverse-derive spec.md + seed the conventions ledger. |
| Temper | `$rite-temper` | _Optional, before define._ Strategic review of the spec — scope mode (expand/selective/hold-rigor/reduce) + pre-mortem; hardens the spec. Best on big/risky features; mandatory in `$rite-autocomplete`. |
| Plan | `$rite-define` | Turn the approved spec into plan + vertical task slices + state. |
| Vet | `$rite-vet` | _Optional, before build._ Engineering review of the plan — scope · architecture · tests · perf; hardens the plan. Best on big/risky features; mandatory in `$rite-autocomplete`. |
| Re-plan | `$rite-plan` | The active plan is too big, wrong, stale, ambiguous, or blocked. |
| Build | `$rite-build` | Implement the next single vertical slice. Stops after one slice. |
| Converge | `$rite-converge` | _Recovery._ Code drifted from / falls short of intent (resumed cold, adopted, stalled build) — assess live code vs spec/plan/tasks and append the remaining work as new slices for `$rite-build`. |
| Prove | `$rite-prove` | Prove the current scope: tests, build, runtime, browser evidence. |
| Polish | `$rite-polish` | Code polish always; UI normalize + ship-quality polish if UI is in scope. Modes: `bolder/quieter/distill/harden/normalize-only`. |
| Review | `$rite-review` | Feature-scoped review before sealing. |
| Seal | `$rite-seal` | Final GO / NO-GO decision (no git). |
| Ship | `$rite-ship` | Type-GO → commit/push/tag, then archive the task + clear ACTIVE. |
| Status | `$rite-status` | See where the active feature stands. |
| Doctor | `$rite-doctor` | Health check — install integrity, stale ACTIVE, orphaned gates, hook wiring, merge/rebase state. |
| Learn | `$rite-learn` | Review the captured learning ledger → promote recurring lessons to project rules / principles. |

## Typical orderings

- **Every feature**: `$rite-spec` (spec) → *(big feature? `$rite-temper` — strategic review)* →
  `$rite-define` (plan) → *(big feature? `$rite-vet` — engineering review)* →
  `$rite-build` ×N (all slices) → `$rite-prove` (once all built) →
  `$rite-polish` (always: code + UI if UI) → `$rite-review` → `$rite-seal` → `$rite-ship`.
- **Existing codebase**: `$rite-adopt` (onboard → spec.md + conventions) → continue at `$rite-define`.
- **Drift mid-build**: stop → drift question → `$rite-plan` (repair) → resume build.
- **Resumed / adopted / stalled**: `$rite-converge` (assess live code vs intent → append the
  remaining slices) → `$rite-build` ×N → continue at `$rite-prove`.

## Rules this menu obeys

- `$rite` never edits code or runs a phase workflow.
- It reads `.devrites/ACTIVE` and `state.md` for status only.
- It suggests; the user (or Claude, when appropriate) invokes the real skill.
