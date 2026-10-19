# Feature map contract

`verify-<app>/features/` indexes what the app does from the user's side and how to prove each
part works. Keep implementation details out; name only user paths, stable handles, required
state, commands, and observable proof.

## `features/README.md`

- **Baseline preconditions:** launch URL or command, a disposable data dir per run, seed data,
  tools on `PATH`, and the Doctor check that must pass. Never drive an instance this run did not start.
- **Driving conventions:** start each recipe from baseline; prefer roles and accessible names
  over CSS or DOM position; treat commands as literal; restore seeded data after a mutation.
- **Proof and skip reporting:** UI proof is an accessibility snapshot plus a screenshot showing
  the app identity; CLI proof is command, stdout, stderr, and exit code; mutation proof adds a
  read-only second view of the stored value. An unreachable path reports the attempted command
  and the unmet precondition; a skipped entry point is never reported as verified through another.
- **Features:** one linked line per feature file.

## One file per feature

An H1 title and one paragraph of user-visible behavior, then exactly these H2s in order:

1. `Sub-features`: short IDs, one line each.
2. `How to get to it (user POV)`: every entry point (button, shortcut, command).
3. `Driving it with <harness>`: starts with `Preconditions:`, then labeled bullets that pair a
   user action with the exact command and the observable result.
4. `Gotchas`: traps that waste or invalidate a run (focus-dependent shortcuts, debounce,
   excluded data, default output formats).

A missing seed precondition makes the journey fail with the reason in `Gotchas`. Never weaken
the journey to pass.

## Example

```markdown
# Search notes

Search finds notes by title or body and tells "no matches" apart from "search unavailable".

## Sub-features
- `search-match` returns title and body matches without changing data.
- `search-empty` shows a complete empty state.

## How to get to it (user POV)
- Choose `Search` in the toolbar, or press `/` outside a text field.
- Run `notes search <query>`.

## Driving it with control-notes
Preconditions:
- Doctor passes; the data dir holds `Quarterly plan` with body `Draft budget`.

- **Title match.** Run `control-notes browser fill --role searchbox --name "Search notes" --value "quarterly"`. Results list `Quarterly plan` only.
- **Empty state.** Fill `volcano`. A status named `No matching notes` appears.
- **CLI.** Run `control-notes cli -- notes search "quarterly" --format json`. Exit `0`; stdout has one object titled `Quarterly plan`.
- **Proof.** Save `artifacts/search/results.aria.txt` and `artifacts/search/results.png`.

## Gotchas
- `/` inside a text field types a slash instead of opening search.
- Results update after a debounce; wait for the list or empty status, not a fixed sleep.
```
