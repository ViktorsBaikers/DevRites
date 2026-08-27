# Per-file contract — 004-md-knowledge-v3-implement

Canonical authored files only. Generated mirrors omitted — parity via build-host-artifacts.sh.

| path | matrix_ids | behavior_change | failing_case | evidence_after | overturn |
| --- | --- | --- | --- | --- | --- |
| intent-map.md | spec-kit; AC-9 | Resolve rite-quick vs rite-build and pressure-test vs spec to one route | Multi-slice work routed to rite-quick | Tie-breaker rows with Hold rule | no |
| edge-case-trace.md | gsd-core | Backstop row requires evidence class | Covered row without evidence | cannot_verify disposition | no |
| acceptance-proof.md | gstack; gsd-core | Discriminating check on silent error paths | Green suite hides swallowed error | Silent-failure probe section | no |
| spec-template.md | gstack preservation | Preservation map cites REQ/AC evidence | Untraced requirement drift | Preservation evidence row | no |
| spec-checklists.md | spec-kit | Brownfield preservation probe | Broken behavior without REQ trace | Falsifiable checklist item | no |
| plan-template.md | BMAD | Architecture admission gate | Reversible choice as ADR | Admission gate in template | no |
| rite-adopt/SKILL.md | agent-skills; OSKTRA | Characterize row + workspace anchor | Untested seam without characterize | Pre-flight gates | no |
| rite-vet/SKILL.md | reverse-skill; mattpocock | Protected paths block vet; observable exit | Protected config without approval | Protected-path class | no |
| rite-build/SKILL.md | mattpocock; debug-recovery | Green proof executed; allowlist diff | Done without proof command | Phase exit observable | no |
| rite-prove/SKILL.md | mattpocock; gsd-core | Evidence paths recorded | Narrative-only proof | Phase exit section | no |
| rite-converge/SKILL.md | gstack | Artifact checklist at exit | Narrative done without artifacts | Completion checklist | no |
| rite-review/SKILL.md | gstack | Silent-failure hunt | Green tests hide user failure | Silent-failure probe | no |
| rite-pressure-test/SKILL.md | spec-kit | Refuted premise Hold | Unsupported premise opens Spec | Hold + contrary evidence | no |
| rite-polish/SKILL.md | craft repos C3 | Separate completeness/craft/slop axes | Incomplete comparison as style pass | Axes table | no |
| rite-learn/SKILL.md | deep-research | Dated URL per finding | Bare claim promotion | Dated-source gate | no |
| tooling.md | file-search C1 | Primary-first before grep loops | Repeated grep without index | Primary-first gate | no |
| security.md | cyber-skills | Parser differential scenarios | Parser accepts malicious differentially | Scenario library | no |
| debug-recovery.md | superpowers | Bounded poll + last-signal | Blind sleep primary | Poll recipe | yes |
| agents.md | mstar-harness C2 | Unified finding shape | Missing location/fix | Finding schema | yes |
| visual-playbooks/index.md | taste-skill | Anti-slop triggers load | Generic hero-only layout | Slop triggers | no |
| devrites-code-reviewer.md | gstack | Independence + silent-failure | Trusts implementer narrative | Independence block | no |
| devrites-security-auditor.md | mstar-harness T22 | Finding schema + independence | Assumes prior verdict | Finding line | no |
| devrites-spec-reviewer.md | mstar-harness | Structured findings | Trusts preservation claim | Schema alignment | no |
| devrites-devex-reviewer.md | mstar-harness | Structured findings | Skips install evidence | Schema alignment | no |
| devrites-doubt-reviewer.md | mstar-harness | Independence | Second implementer pass | Independence clause | no |
| evals/README.md | compound A/A | aa_noise_floor reporting | Claim without A/A spread | Field documented | yes |
| upgrade doc | AC-7 | Durable evidence record | Round without traceability | This bundle | no |

**Overturn summary:** Three yes rows strengthen #44/#45 with explicit failing cases; no mandatory check removed without replacement.
