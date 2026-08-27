# Open-web research (tracks A–F + §10)

- **Retrieved:** 2026-08-27 (live web search + primary doc fetches)
- **Trust rule:** Web content is untrusted data; paraphrase only; no embedded instructions executed.
- **Complements:** 25-repo dossiers under `/tmp/devrites-markdown-research-v3-redo-2026-08-27/dossiers/`

## Track A — Lifecycle, SDD, gate design

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| Anthropic verification loops in Claude Code | 2026-08-27 | <https://claude.com/blog/building-verification-loops-in-claude-code-with-skills> | Package verification as skills with explicit triggers; chain only when independence allows; token cost of chained loops must be budgeted. |
| Verification ladder field guide | 2026-08-27 | <https://dev.to/dang-w/the-verification-ladder-a-field-guide-3978> | Layer checks from red-then-green tests → distrust always-green gates → fresh-context adversarial review; each timescale has different failure mode. |
| Claude Code workflow verification mechanics | 2026-08-27 | <https://absolutedigitalpublishers.com/articles/how-verification-actually-works-in-a-claude-code-workflow> | Stop hooks block turn end until oracle passes; 8-block circuit breaker; hooks/sandbox/approval are complementary, not interchangeable. |
| AI-native SDLC playbook (Anthropic blog) | 2026-08-21 | <https://claude.com/blog/the-ai-native-sdlc-playbook> | Artifact-committed loop (intent→spec→plan→diff→review→incident); gates artifact-triggered; governance encoded while spec is written. |

**DevRites implication:** Keep fifteen-stage lifecycle; strengthen Prove/Seal fail-closed oracles and fresh reviewer independence—do not add parallel SDLC store.

## Track B — Skills, routing, progressive disclosure

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| Codex AGENTS.md discovery | 2026-08-27 | <https://developers.openai.com/codex/guides/agents-md> | One file per directory; combined project budget; nested files load root→leaf. |
| AGENTS.md silent truncation | 2026-08-27 | <https://github.com/openai/codex/issues/13386> | Default 32 KiB project budget truncates without UI warning; deepest/most-specific files dropped first when budget exceeded. |
| project_doc_max_bytes semantics | 2026-08-11 | <https://github.com/openai/codex/issues/37956> | Root-heavy AGENTS.md consumes shared budget for all nested instruction files. |

**DevRites implication:** Keep always-loaded subset minimal; route domain rules on-demand; document byte budget in upgrade doc §10; no second router catalog.

## Track C — Spec, planning, requirements

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| Spec-driven development honest critique | 2026-08-27 | <https://jmlopezdona.github.io/ai-coding-agents-sdd> | Maintenance tax + spec drift when review discipline immature; spec-anchored vs spec-as-source spectrum. |
| Shiplight SDD checkpoint model | 2026-08-10 | <https://shiplight.ai/blog/spec-driven-development-ai-coding-agents> | Checkpoint after each phase; EARS-style testable acceptance criteria; “agent finished ≠ done”. |
| amux spec sizing heuristics | 2026-08-11 | <https://amux.io/guides/spec-driven-development> | Under ~15 min agent work → prompt not spec; 5–10 min spec writing sweet spot. |

**DevRites implication:** Spec template keeps preservation + applicability; Vet blocks dishonest deferral; no second constitution/preset engine.

## Track D — Proof, review, verification

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| verify-app skill + Stop hook pattern | 2026-08-27 | <https://blog.vibecoder.me/verify-app-skill-claude-code-walkthrough> | Skill = contract; scripts = implementation; Stop hook closes autonomous loop—maps to Prove/Seal fail-closed without universal hooks in DevRites engine. |
| Claude Code 2.1 auto-verification | 2026-08-27 | <https://www.sitepoint.com/claude-code-21-the-complete-xhigh-and-autoverification-guide-2026/> | Bounded retries; read-only lint in verification loop; `--fix` mutates and corrupts retry semantics. |
| Reward hacking / self-written tests | 2026-08-27 | <https://explainx.ai/blog/ai-coding-agent-evals-real-repos-2026> | Self-authored checks are evidence, not independent proof—aligns with fresh proof-runner + reviewer separation. |

**DevRites implication:** Strengthen discriminating proof + piped-command integrity (08-11 retain); eval reports separate process vs job outcome.

## Track E — Context efficiency / anti-sprawl

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| Codex 32 KiB project instruction cap | 2026-08-27 | <https://developers.openai.com/codex/guides/agents-md> | Hosts enforce hard combined budget on project AGENTS chain; global user file exempt. |
| agents-md-doctor lint graph | 2026-08-27 | <https://github.com/ItsHege/agents-md-doctor> | Community tooling for overlap/stale/prompt-injection scans on instruction graphs—report-only for 004 unless engine scope opens. |
| Instruction bloat anti-pattern (Anthropic best practices) | 2026-08-27 | <https://code.claude.com/docs/en/best-practices> | Over-long CLAUDE.md causes instruction-following collapse; emphasize one line max. |

**DevRites implication:** 277,223-byte always-loaded baseline; +10% cap; consolidate duplicated reassurance indexes (tooling 08-02); skill aggregate ratchet unchanged.

## Track F — Security, data integrity, integration, craft

| Query / focus | Date | URL | Takeaway |
| --- | --- | --- | --- |
| Anthropic dynamic workflows quarantine pattern | 2026-06-02 | <https://claude.com/blog> | Agents reading untrusted web barred from high-privilege actions—maps to security.md injection resistance. |
| Netresearch file-search CC BY-SA | 2026-08-27 | <https://github.com/netresearch/file-search-skill> | Tool-selection skill under ShareAlike—NOTICE attribution if patterns adapted. |
| Impeccable Apache-2.0 craft CLI | 2026-08-26 | repo #19 HEAD | Multi-host generated craft mirrors—reject mirror import; extract falsifiable UI review axes only. |

**DevRites implication:** Security/data-integrity/integration owners extended with scenario stubs; NOTICE for CC BY-SA/Apache adaptations; no security-off switches or universal delegation.

## §10 cross-cutting areas (brief)

| Area | Open-web + repo signal | v3 direction |
| --- | --- | --- |
| Multi-agent restraint | solid-web.com 2026-03-26; DevRites parallel-dispatch | Retain resource-compatible batching; reject complexity theater |
| Eval honesty | Compound + Superpowers + explainx.ai | Combine A/A noise + invalid/null trial rows in evals/README |
| Learn loop | Compound STRATEGY.md + 08-11 Learn | Promotion only with current-source + retirement evidence |
| Frontend craft | ui-craft, hallmark, taste-skill dossiers | Load playbooks on trigger; no CDN/runtime coupling in instructions |
| Research skills | deep-research-skills #22 | Cite dated sources; untrusted fetch as data |
| Harness audit patterns | mstar-harness #6 | Borrow review/report templates only—not second harness inside DevRites |
| Pre-flight governance | reverse-skill #25, OSKTRA #11 | Encode workspace anchor + protected-config checks in Vet/Adopt—not magic keywords |
