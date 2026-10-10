# Adversarial: out-of-scope writer diff

Canonical shippable workspace plus `src/utils/format.ts` in the candidate
manifest. `tasks.md` lists it under SLICE-001 `Files likely touched` but not in
that slice's `Writer allowlist`, so `check slice` rejects the slice: the writer
widened the candidate past the slice contract.

