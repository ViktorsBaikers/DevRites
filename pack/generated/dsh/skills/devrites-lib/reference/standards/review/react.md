# Review: React / Next.js

> Applies when: the resolved manifest of the reviewed package declares `react`.
> Load with [`default.md`](default.md) and [`ts-js.md`](ts-js.md). Vue, Svelte,
> Angular, and vanilla code never load this file.

Resolve each tag yourself from the package's resolved lockfile entry or installed
package (a leaf never spawns `devrites-source-driven`; an open version question returns
to the root as a gap). An unresolved predicate means the
probe is skipped and listed in `Basis` as `predicate unresolved: <tag>`, never applied
on a guess. Each probe is a lead: establish the contract, reachable path, and effect
before reporting; a finding names its tag and how the version was resolved.

Tags: `[any]` · `[react≥19]` · `[compiler on]` (optimizing compiler enabled in build
config) · `[next app-router]` · `[next≥15.x]` (minor stated per probe; verify
installed version). `(perf)` marks probes the performance reviewer also applies.

## Defect probes

- `[any]` **Derived state through an effect.** Props or state copied into state by
  an effect render one frame stale and can desync. Proof: change the source; the
  derived value is correct in the same render.
- `[any]` **Action modelled as state + effect.** A submit or click stored as a flag
  and executed by an effect repeats on remount or dependency change. Proof: remount
  under strict mode; the side effect runs once.
- `[any]` **Component type defined in render.** A component declared in another's
  body remounts each render, dropping focus and state. Proof: type into a nested
  input across a parent re-render; focus and value survive.
- `[react≥19]` **setState after `await` in an async transition** with no sequence
  guard: an earlier transition resolving late overwrites a newer one. Proof: the
  deferred-promise ordering test of the stale-response probe in `ts-js.md`.
- `[compiler on]` (perf) **Memo added or removed** without evidence: an addition
  needs profiler evidence; a removal needs a test that effect run counts stay unchanged.
- `[next app-router]` **`'use server'` exports are public entry points.** Each
  authenticates, authorizes, and validates inside the action — judge under
  [`security.md`](../security.md). Proof: call it with no session and as another user.
- `[next app-router]` (perf) **Server→client props** carry unused or sensitive fields
  into the serialized page payload.
- `[next app-router]` **`ssr: false` inside a Server Component** fails the build
  (verify installed version).
- `[any]` **Mutable module-level request or user data** in a long-lived SSR process.
  Proof: two overlapping requests as different users show no cross-contamination.
- `[next≥15.1]` **`after()` used for invalidation or a must-not-lose write**: it runs
  after the response is sent and may run for failed responses (verify installed version).
- `[any]` **`suppressHydrationWarning` on a non-leaf node** hides real mismatches; a
  pre-hydration inline script without the page's CSP nonce.
- `[any]` (perf) **Barrel imports** only when the bundler does not optimize them and a
  bundle measurement shows the cost.

## Do not flag

- Missing `memo`/`useMemo`/`useCallback` without profiler evidence.
- `useContext` where `use()` exists; `forwardRef` on React 18 (still required there).
- Migration to a newer API with no behavior change.
- A probe whose predicate is unresolved: list it in `Basis`, do not report it.
