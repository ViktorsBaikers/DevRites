# rite-build output

**Slices remain:**
```text
Done: built slice <n> — <name>.
Changed: <files>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd->pass>; browser <summary|n/a>; drift <none|handled>
Open: <none|blockers|questions>
Next: /rite-build
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear between slices; /rite-handoff if away > a few hours
```

**All built:**
```text
Done: built slice <n> — <name>; all slices built.
Changed: <files>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd->pass>; browser <summary|n/a>; drift <none|handled>
Open: <none|blockers|questions>
Next: /rite-prove
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before /rite-prove; /rite-handoff if away > a few hours
```

HITL returns after one green slice. Only explicit `.devrites/AFK` lets the
controlling root chain another pending slice after green-proof, cap, and pause
gates. Every `devrites-slice-wright` returns after exactly one slice.

At all built, Build never enters `/rite-prove` automatically; a higher-level
orchestrator may invoke it only under its own contract. Awaiting-human or
stopped states use the shared reply contract.
