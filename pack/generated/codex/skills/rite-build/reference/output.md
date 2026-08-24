# rite-build output

**Slices remain:**
```text
Done: built slice <n> — <name>.
Changed: <files>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd->pass>; browser <summary|n/a>; drift <none|handled>
Open: <none|blockers|questions>
Next: $rite-build
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear between slices; $rite-handoff if away > a few hours
```

**All built:**
```text
Done: built slice <n> — <name>; all slices built.
Changed: <files>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd->pass>; browser <summary|n/a>; drift <none|handled>
Open: <none|blockers|questions>
Next: $rite-prove
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear before $rite-prove; $rite-handoff if away > a few hours
```

HITL returns after one green slice; AFK may chain under green-proof/cap/pause.
Wright returns after one slice. Build never auto-enters `$rite-prove`.
