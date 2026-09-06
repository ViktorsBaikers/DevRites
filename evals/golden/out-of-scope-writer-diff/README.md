# Adversarial: out-of-scope writer diff

Canonical shippable workspace plus `src/utils/format.ts` in the candidate
manifest. That path is not named in `tasks.md`. Outcome evals must treat this
as a failure: the writer widened the candidate past the slice contract.

