# Menu reference

Phase-ordered command reference for `/rite`. Load only if the user wants detail on
what each command does or how phases connect.

## Phases and commands

| Phase | Command | Use when |
|---|---|---|
| Spec | `/rite-spec <feature>` | **Start here.** Investigate deeply → write spec.md. Asks you with options; gathers any design references you attach (optional). |
| Temper | `/rite-temper` | _Optional, before define._ Strategic review of the spec — scope mode (expand/selective/hold-rigor/reduce) + pre-mortem; hardens the spec. Best on big/risky features; mandatory in `/rite-autocomplete`. |
| Plan | `/rite-define` | Turn the approved spec into plan + vertical task slices + state. |
| Re-plan | `/rite-plan` | The active plan is too big, wrong, stale, ambiguous, or blocked. |
| Build | `/rite-build` | Implement the next single vertical slice. Stops after one slice. |
| Prove | `/rite-prove` | Prove the current scope: tests, build, runtime, browser evidence. |
| Polish | `/rite-polish` | Code polish always; UI normalize + ship-quality polish if UI is in scope. Modes: `bolder/quieter/distill/harden/normalize-only`. |
| Review | `/rite-review` | Feature-scoped review before sealing. |
| Seal | `/rite-seal` | Final GO / NO-GO decision. |
| Status | `/rite-status` | See where the active feature stands. |

## Typical orderings

- **Every feature**: `/rite-spec` (spec) → *(big feature? `/rite-temper` — strategic review)* →
  `/rite-define` (plan) → `/rite-build` ×N (all slices) → `/rite-prove` (once all built) →
  `/rite-polish` (always: code + UI if UI) → `/rite-review` → `/rite-seal`.
- **Drift mid-build**: stop → drift question → `/rite-plan` (repair) → resume build.

## Rules this menu obeys

- `/rite` never edits code or runs a phase workflow.
- It reads `.devrites/ACTIVE` and `state.md` for status only.
- It suggests; the user (or Claude, when appropriate) invokes the real skill.
