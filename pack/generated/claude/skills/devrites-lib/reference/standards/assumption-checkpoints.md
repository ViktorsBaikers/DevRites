# Assumption-delta checkpoints

> Applies when: planning or vetting a slice whose scope generalizes an existing
> boundary — the seam where a quiet assumption loses its monopoly.

Most quietly imported architectural debt comes from the seam: a later change adds a
*second case* — platform, auth method, tenant, region, source of truth — and nobody
re-asks whether the original abstraction still names the right thing. The change that
adds the second case is the cheap moment to ask.

## The detector

Scan the phase scope, spec delta, or task list for transition signals:

| Signal | Shape | The question it raises |
|---|---|---|
| `pluralization` | A second X where exactly one existed (platform, provider, tenant, region, source of truth) | Does the identity model still name the right noun? |
| `optionality` | A required or `only`-assumed field becomes optional | Is that field still the right anchor, or did the anchor move? |
| `choice` | A derived value becomes chosen; a constant becomes a parameter | Has a config decision silently become a modeling decision? |

Code blocks, comments, and prose that merely *mention* a trigger word do not count.

## Firing the checkpoint

A firing signal surfaces **one** question before the plan is finalized — not a
redesign, one question: *does the name/model chosen when N=1 still hold at N=2?*

- `detected:false` — skip the checkpoint; do not raise it.
- `skipped` — the scope was never examined. A skipped check asserts *nothing*: never
  record "no assumption changed" from a skipped result; skipped ≠ negative.
- `detected:true` — ask the one question and record the answer in `decisions.md` or
  `assumptions.md`; an answered checkpoint that leaves no trace re-fires next round.

## Promote, don't add alongside

When a generalization is real, the default-correct move is to **promote** the general
representation to primary and demote the old specific one to a detail of one variant —
not to add the new shape alongside the still-required old one, which silently
contradicts the generalized intent.

**Failing case:** `region` added as an optional column beside `primaryRegion`; every
write now guesses which is authoritative.
