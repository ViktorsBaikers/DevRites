# Coding style

Write code the next engineer can read without you in the room. Match the project's
existing idiom first; these rules fill the gaps.

## Names reveal intent
- A name should answer *what is this and why*. `daysSinceLastModification`, not `d`.
- Avoid generic verbs that hide intent: `process()`, `handle()`, `doWork()`, `manage()`.
- Be consistent: the same concept gets the same word across the codebase. Don't alias
  `user` / `account` / `member` for one thing.

## Functions do one thing
- One responsibility per function; if you need "and" to describe it, split it.
- Keep functions short enough to hold in your head. Long functions hide bugs.
- Limit parameters; a long parameter list usually wants a struct/object or a split.
- Make edge cases explicit rather than implicit in clever control flow.

## Guard clauses over nested pyramids
Handle the unwanted cases up front and return early; keep the success path flat.
```
# instead of nesting the whole body in if/else, exit early:
if (!user) return Unauthorized
if (!user.active) return Forbidden
# ...happy path here, un-nested
```

## Comments explain *why*, not *what*
- Self-explanatory code beats a comment restating it. Rename before you comment.
- Reserve comments for intent, trade-offs, non-obvious constraints, and "here be
  dragons" warnings. Delete commented-out code — that's what version control is for.

## Simplicity
- Prefer the simplest thing that works. Don't add abstraction before you have two real
  callers; premature generalization is a cost, not a saving.
- Don't be clever at the expense of clear. Shorter-but-cryptic is not simpler.
- Delete dead code you created; don't leave TODOs or stray debug logs in shipped code.

## Reuse before you write
Before adding a new util, helper, hook, type, component, or formatter, **search** for an
existing one. **Reuse → extend → build new**, in that order. Don't re-implement what the
project (or stdlib) already provides. If forcing reuse would distort the existing thing's
shape, build a sibling and consolidate later — duplication is cheaper than the wrong
abstraction (AHA).
