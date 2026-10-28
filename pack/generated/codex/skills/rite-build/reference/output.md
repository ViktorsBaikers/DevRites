# rite-build output

> Root-only: not inlined in --role bundles.

## Caller-owned continuation

Before standalone replies, return internally to active `$rite-autocomplete` or
another controlling rite after green proof, checkpoint, and accounting:
- Slices remain: re-read durable state and reselect/refill; no user-facing reply.
- All built: return internally to the caller to invoke `$rite-prove` under its own contract.

Human/safety/access/Git stops still use the shared reply contract.

## Standalone response

HITL returns after one green slice; AFK chains per `afk-discipline.md`.
Every wright returns after one slice. Build alone never invokes Prove.

```text
Done: built slice <n> — <name>[; all slices built].
Changed: <files>; state.md, touched-files.md, evidence.md
Evidence: acceptance met; tests <cmd> → pass; browser <summary|n/a>; drift <none|handled>
Open: <none|blockers|questions>
Next: $rite-build [all built: $rite-prove]
Record: .devrites/work/<slug>/evidence.md
↻ Hygiene: /clear between slices (none in `orient` `inflight_waves`); $rite-handoff if away > a few hours
```

Awaiting-human or stopped states use the shared reply contract.
