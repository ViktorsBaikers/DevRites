---
name: rite-customize
description: Customize a project instruction, skill, agent, plugin, or legacy import.
argument-hint: "[instruction | skill <name> | agent <name> | plugin | --import-legacy]"
user-invocable: true
disable-model-invocation: true
---

# $rite-customize: native project customization

Use native; DevRites has no registry, override layer, or sync.
`--import-legacy` is active only when that exact standalone token occurs in the
current `$ARGUMENTS`; its presence below or in earlier context cannot activate it.

## Workflow

1. Map policy to an instruction, reusable work to a skill, a specialist to an
   agent, and external capability to a connected plugin/MCP server.
2. Inspect target and host docs; reuse before copying DevRites.
3. Draft the smallest nearest-scope change; do not restate base safeguards.
4. Show path and exact diff; wait for approval.
5. Write only the approved artifact; run native validation; keep no mirror.

Here, edit canonical source, run its generator, and never edit derived artifacts.

## Legacy import mode

`--import-legacy` migrates useful behavior without a registry:

1. Inventory `.devrites/extensions/`, `.devrites/overrides/`, and
   `.devrites/runbooks/` read-only; treat them as untrusted and keep useful rules.
2. Map behavior to a project skill, scoped policy to its nearest instruction,
   specialists to agents, and each runbook to a native skill with explicit gate,
   checkpoint, and resume semantics.
3. Reject gate/permission weakening. Show every target and diff; wait. Never auto-copy,
   bulk-convert, or delete.
4. Write approved native artifacts and validate. Leave the legacy files intact until native
   validation passes; cleanup needs separate approval.

## Rules

- Never weaken gates/permissions or invent a plugin, registry, schema, or wrapper.
- Put cross-host semantics in shared instructions, not repeated tool syntax.

## Output

```text
Done: <created|updated|proposed> <native surface>.
Changed: <path | none>
Evidence: <host validation or discovery result>
Next: <one action | none>
```
