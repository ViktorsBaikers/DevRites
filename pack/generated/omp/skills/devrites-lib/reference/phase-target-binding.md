# Lifecycle source binding

> Root-only: not inlined in --role bundles.

## Phase target binding

For every lifecycle phase and cold handoff, resolve the existing explicit target
or caller-supplied SOURCE slug first; use `.devrites/ACTIVE` only when neither is
supplied. All workspace context, cursor, questions, artifact reads/writes, approval,
candidate bindings and proof belong to that source. A missing source artifact
routes its prerequisite; it never falls back to another workspace's artifacts.
An explicit valid workspace does not require ACTIVE to name it or to exist.

Carry SOURCE through every phase, nested resolution and cold handoff. Use existing
slug arguments (`context <source>`, `handoff <source> --resume`). Qid-only
`state resolve` retains its argv: bind only that invocation's `DEVRITES_ROOT` and
`DEVRITES_WORKSPACE` as above, never a fabricated resolver slug or process-global
change. Early future work preserves ACTIVE and the retained controller.
Standalone Spec still sets ACTIVE; `--lookahead` remains planning/Vet-only.
Target binding changes neither phase approval nor Ship authority.

## Controller guidance load

Before global scheduling, successor preparation, retained-controller recovery, or shared-source proof/integration/release procedures, select trigger `parallel-workspaces` and read [the complete scheduler](parallel-workspaces.md). If applicability is uncertain, load it. Autocomplete already loads it always. Loading guidance neither activates a controller nor grants authority; existing admission, lookahead, and approval limits still apply.
