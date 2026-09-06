# Reuse first: frontend application

Frontend application of the canonical rule in
[`coding-style.md`](../../devrites-lib/reference/standards/coding-style.md#reuse-before-you-write):
reuse → extend → build new, AHA caveat below. Before creating any new component, style,
token, icon, hook, util, or helper, search the project first.

This applies to UI **and** non-UI code: utilities, helpers, types, validators, schemas,
formatters, hooks, query helpers: anything that might already exist.

## The decision (in order)
1. **Search first.** Apply `../../devrites-lib/reference/standards/tooling.md`: use the
   primary available code index to find similar definitions, cross-check only a named
   unresolved predicate, then fall back to grep/glob over `components/`, design tokens, hooks/,
   utils/, lib/. Look for things doing the *same job*, not just the same name.
2. **Exact fit → REUSE.** Compose / import the existing thing. No copy, no fork.
3. **Close fit → EXTEND.** Add a variant/prop/option that the existing component or util
   can carry without distortion. Adding a prop to a button is fine; adding ten props that
   each branch the internals is not.
4. **No fit → BUILD NEW**, in the project's idiom. If a similar need is likely to recur,
   propose adding it to the shared system (component library / utils module). If it's
   truly one-off, build it locally: don't pre-abstract.

## When to *avoid* reuse (the AHA caveat)
Reuse is good; **forcing** it isn't. **Avoid Hasty Abstractions**: if making the existing
thing fit means warping its API or breaking its contract for one caller, **don't**.
Duplication is cheaper than the wrong abstraction. Build a sibling component/util and let
a real pattern emerge from two or three callers before consolidating.

## Record the decision (per slice)
On each slice, note what was reused / extended / created new, so the seal can see the
consistency story and reviewers can spot silent duplication.

```
Existing reused: <Button, useToast, formatCents>
Extended: <Card — added variant="elevated">
Created new: <ExportProgress (no prior equivalent; propose for shared lib)>
```

## Anti-patterns to name
- Silently re-implementing an existing component / hook / util because it was easier than
  finding it. Search first.
- Copy-pasting a component from `components/` into the feature folder. That's a fork.
- Adding a second icon set / second toast library / second date util because the existing
  one didn't quite fit: almost always extend the existing or ask first.
- Forcing reuse via 8 boolean props that branch the internals. That's an abstraction
  collapsing under its own weight. Split it.

## Search hints
- **UI**: `components/ui/`, design-system package imports neighboring features use,
  `tokens.*` / Tailwind config / theme files, icon barrel files, layout primitives.
- **Non-UI**: `lib/`, `utils/`, `helpers/`, `services/`, `validators/`, `schemas/`,
  `hooks/`, language stdlib equivalents.
