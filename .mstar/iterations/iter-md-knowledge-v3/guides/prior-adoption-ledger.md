# Prior-adoption ledger (08-02 / 08-11 / benchmark only)

- **Sources allowed:** `docs/markdown-instruction-upgrade-2026-08-02.md`, `docs/markdown-instruction-upgrade-2026-08-11.md`, `docs/upstream-workflow-benchmark-2026-08-01.md`, related `docs/research/*.md`, live pack inspection.
- **Excluded:** quarantined `/tmp/devrites-quarantine-08-27-docs/*`, prior `/tmp/devrites-markdown-research-v3` notes as ledger authority.
- **Recorded:** 2026-08-27 @ `main` `1da70ceced71b7e6c27cc204a06ff3b2926f932a`
- **`source_doc` column:** `08-02`, `08-11`, or `benchmark-08-01` only (per spec §003→004). Upstream repo cross-refs live in `upstream_note`.

| concept | source_doc | upstream_note | disposition | canonical_owner | action_v3 | overturn_rationale |
| --- | --- | --- | --- | --- | --- | --- |
| Four-layer precedence (authority/evidence/method/advice) | 08-02 | | adopted | `pack/.claude/skills/devrites-lib/reference/standards/core.md` | retain | |
| Controlling request vs embedded untrusted data | 08-02 | | adopted | `context-hygiene.md`, `security.md` | retain | |
| Observable outcomes + preserving REQ/AC in Spec | 08-02 | gstack review patterns | adopted | `rite-spec/reference/spec-template.md` | strengthen | Add explicit failing case when preservation row missing evidence |
| Six-item ORIENT evidence gate (not five lookups) | 08-02 | | adopted | `devrites-slice-wright.md` | retain | |
| Resource-compatible parallel dispatch | 08-02 | | adopted | `parallel-dispatch.md` | retain | |
| Qualified backstop + cannot_verify | 08-02 | GSD edge disposition | adopted | `acceptance-proof.md`, `devrites-proof-runner.md` | strengthen | Prove must name missing evidence class, not self-report |
| Explicit → active lifecycle → one implicit route | 08-02 | OMC routing | adopted | `intent-map.md`, `skill-authoring.md` | strengthen | Add AC-9 tie-breakers for rite-quick vs rite-build |
| Load-path placement gate (core / shared / local) | 08-02 | ECC catalog discipline | adopted | `skill-authoring.md` | retain | |
| Normative rewrite preservation map | 08-02 | BMAD preservation map | adopted | `skill-authoring.md` | retain | |
| Spec-quality checklist (requirements-as-tests) | 08-02 | Spec Kit checklist | partial | `spec-checklists.md` | strengthen | Add falsifiable brownfield preservation probes |
| Diagnostic redaction before persistence | 08-11 | | adopted | `security.md`, debug recovery refs | retain | |
| ADR admission rejects derivable/reversible choices | 08-11 | | adopted | Define template, ADR promotion | retain | |
| Supporting + contrary evidence in Pressure Test | 08-11 | | adopted | `rite-pressure-test` skill | retain | |
| Characterize-before-modify on load-bearing seams | 08-11 | | adopted | Adopt, Define | strengthen | Require see-it-fail evidence stub in plan |
| Pipeline exit integrity (no masked producer failure) | 08-11 | | adopted | testing / Vet standards | retain | |
| Bounded condition polling (no blind sleep) | 08-11 | | partial | debug recovery | strengthen | Add max-wait + last-signal artifact |
| External source provenance on imported guidance | 08-11 | | adopted | skill-authoring, Customize | retain | |
| A/A noise floor + process vs job eval axes | 08-11 | | partial | `evals/README.md` | strengthen | Sanitized per-trial invalid/null outcomes |
| Decision-horizon register (Define/Plan/Vet) | 08-11 | | adopted | Define plan template, Vet | retain | |
| Learn/doc promotion with retirement evidence | 08-11 | | partial | Documentation, rite-learn | strengthen | Require discoverability + conflict check |
| Reviewer finding fingerprints / second wave | 08-11 | | rejected | Review/Seal | reject | Duplicates fresh required roles; 08-11 deferred then rejected |
| Browser daemon / sink telemetry stack | 08-02 | benchmark runtime coupling | rejected | none | reject | Runtime coupling; out of Markdown scope |
| Second spec store (OpenSpec lifecycle) | 08-11 | 08-02 OpenSpec rejection reaffirmed | rejected | none | reject | Conflicts with `.devrites/` workspace model |
| Universal TDD / worktrees / fixed review rounds | 08-02 | Superpowers TDD/worktree | rejected | none | reject | Conflicts with evidence-scaled DevRites lifecycle |
| BMAD personas / module graph | 08-02 | | rejected | none | reject | Duplicates DevRites agents |
| OMC magic modes / model tiers | 08-02 | | rejected | none | reject | Unsafe routing |
| ECC catalog import (66/817 skills) | 08-11 | benchmark sprawl note | rejected | none | reject | Sprawl + drift |
| Compound automatic learning ledger | 08-11 | | rejected | none | reject | Parallel store; Learn owns promotion |
| gstack always-invoke fallback | 08-02 | | rejected | none | reject | Over-triggering |
| OpenSpec cross-repo automatic writers | 08-11 | | rejected | none | reject | Violates single writer |
| Fixed numeric coverage / rollout thresholds | 08-11 | agent-skills rollout thresholds | rejected | none | reject | False confidence without project baselines |
| Live-host eval runner with coverage debt | benchmark-08-01 | | deferred | none | defer | Needs engine owner + cost model |
| GSD forensics command / capability runtime | 08-11 | | deferred | debug recovery | defer | No new public command without engine |
| Lexical routing diagnostics harness | 08-11 | | deferred | evals | defer | Forbidden non-Markdown runtime in 004 |
| Reviewer hit-rate / dedup state | 08-11 | | deferred | Review | defer | No demonstrated recall gap on fixtures |
| Sink-byte redaction automation | 08-02 | | deferred | security | defer | Engine/script scope |
| Context telemetry / percentage budgets | 08-11 | GSD + ECC telemetry ideas | deferred | none | defer | Engine enforcement track |
