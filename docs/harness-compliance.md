# Harness adapter compliance

Claude Code and Codex expose different native enforcement surfaces. This page
records where each host enforces a boundary, needs an adapter or instruction,
or deliberately stops because the surface is unavailable.

New hosts must pass
[`porting-to-a-new-harness.md`](porting-to-a-new-harness.md), including the
React todo-list routing transcript, before they are listed here.

The capability table is maintained directly as descriptive documentation.
Operational truth lives in the canonical Claude profiles and settings, generated
Codex profiles and configuration, workflow skills, and the
generation/install/permission/routing tests that exercise them.

| Surface | Claude Code | Codex | Phase impact | Fallback | Native doctor | Confidence | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Session context and compaction | Native | Native | all phases | explicit skill state read | no | high | Both hosts own transcript, session, resume, and compaction context; DevRites does not inject a duplicate lifecycle hook. |
| Root orchestrator permissions | Native | Instruction-backed | all source mutations | stop if the root edits source | yes | medium | Claude uses project plan mode; Codex needs a workspace-capable parent for native writer-child execution, so its root no-source-writing boundary is instruction-enforced. |
| Slice-wright exact paths | Instruction-backed | Instruction-backed | build safety | reject an out-of-scope returned diff | yes | medium | Both hosts run the exact write-capable slice-wright under native sandboxing. The task states exact paths; the root compares the returned file list and `git diff --name-only` with that contract. |
| Native reviewer permissions | Native | Native | review/seal safety | HITL stop; no root inline review | yes | high | Claude reviewer profiles use permissionMode plan; Codex reviewer profiles use `default_permissions = ":read-only"`. No engine reviewer classifier is involved. |
| Skill invocation | Native | Native | all public rites | explicit file read | yes | high | Claude invokes /rite natively. Codex invokes $rite natively from the generated/mirrored .agents/skills tree; the mirror delivers artifacts rather than adapting execution. |
| Candidate identity | Adapter-backed | Adapter-backed | Build through Ship | stop; never substitute a host hash | partial | high | Both hosts expose read-only `devrites-engine check candidate <slug>` with exact two-field output and preserve one binding across evidence, optional browser evidence, review, and seal. Build writes scope, Polish closes it, and Ship is candidate-read-only. |
| Reviewer subagent dispatch | Native | Native | review/seal confidence | HITL stop; no root inline review | yes | high | Both hosts dispatch the exact named project role. Codex loads developer instructions and the `:read-only` permission profile from the generated custom-agent TOML. |
| Standards step-0 load | Native | Instruction-backed | all phases | explicit standards read | partial | medium | Claude: skill Reads core.md. Codex: an AGENTS.md directive to read it. |
| Project activation | Native | Conditional | Codex-only startup | inspect config/agents, decide trust, then rerun `/rite-doctor` | yes | medium | Codex silently skips every .codex/ layer in an untrusted project; trust remains an operator decision after inspection. |

Tiers: **Native** (the harness exposes and delivers the surface directly; policy notes state whether it blocks) · **Adapter-backed** (supported through a translation shim) · **Instruction-backed** (no runtime surface: rides on a directive the model may under-fire) · **Conditional** (native but gated on an operator precondition) · **Unavailable** (deliberately absent; the workflow stops).

The Doctor column refers to `/rite-doctor`'s read-only root inspection. It is a
native skill procedure, not an engine diagnostic command.

## Evidence boundary

Generation and install tests verify the native profiles and configuration.
Permission and routing tests prove that both hosts dispatch the exact
slice-wright while every other specialist remains read-only. Claude grants that
writer `acceptEdits`; Codex grants it `:workspace`. Exact-path scope is
instruction-backed on both hosts and the root rejects a returned diff outside
the task contract. Candidate compatibility additionally requires the shared
read-only engine command, exact binding lines, and unchanged phase ownership;
an adapter may not introduce another schema, registry, or identity algorithm.
