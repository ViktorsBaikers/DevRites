# Review: TypeScript / JavaScript

> Applies when: reviewing `*.ts`, `*.tsx`, `*.js`, `*.jsx`, `*.mjs`, `*.cjs`, or the
> script blocks of `*.vue`, `*.svelte`, `*.astro`. Load with [`default.md`](default.md).

## Defect probes

- **Unhandled promise.** An async call whose rejection has no `.catch`/`await`
  + `try` — an unhandled rejection is a silent failure or a crash depending on
  the runtime. Trace deliberate propagation to the caller's error boundary.
  `Promise.all` rejects early but does not cancel siblings; `allSettled` observes
  every outcome, not cancellation. Check explicit cancellation and cleanup when
  the contract requires them.
- **`await` in a loop** — check measured latency and intended concurrency bounds.
  Independence alone does not justify unbounded parallelism; sequential work can
  deliberately limit rate, memory, or resource use.
- **`any`/`as` escape.** `as X` or `any` hiding an invariant the compiler
  cannot check; a cast that will silently pass wrong-shaped data at a trust
  boundary (API response, storage, IPC).
- **`==` vs `===`** — `==` with null/undefined is a deliberate idiom
  (`x == null`); other loose comparisons on user input are defects.
- **Mutated prop/state.** Direct mutation of React state or props; a state
  update built from stale state (`setCount(count + 1)` inside an interval)
  instead of the functional form.
- **Effect deps.** `useEffect`/`useMemo`/`useCallback` with a stale or missing
  dependency that changes behavior; a dep array suppressed by a comment without
  a reason.
- **Event/listener leak.** `addEventListener`, subscription, `setInterval`,
  socket, or observer without a matching cleanup on unmount/teardown.
- **Stale response.** A fetch keyed on changing input (query text, route param,
  selected id) in an effect, watcher, or store needs an ignore flag, request id, or
  sequence guard so a late response for old input cannot overwrite newer state.
  Abort alone is insufficient: a response already resolving, or a non-abortable
  source, still lands. Proof: resolve request B before A with deferred promises; the
  UI shows B. A render-only test is inadmissible.
- **Client async lifecycle.** Probe each, with the proof it needs:
  - work for an old parameter or an unmounted component is cancelled or ignored —
    change the param mid-flight and unmount; no state update or request follows;
  - double submit — two rapid activations produce one side effect (or the server
    dedupes by idempotency key);
  - optimistic update — on a failed request the UI returns to server state and
    surfaces the error; a silent rollback or none at all is a defect;
  - impossible states — independent booleans (`loading`, `error`, `data`) that can
    be true together; a test drives failure-after-success and asserts one state renders;
  - index keys on a list that inserts, removes, or reorders — insert at the top
    with a filled row input; the value stays with its item (static lists: do not flag);
  - SSR non-determinism — time, random, or locale/timezone output in server-rendered
    markup; render with a different clock or timezone on the client and the markup
    matches, or the value is client-only.
- **Prototype/key injection.** `obj[key] = v` or deep-merge with `key` from
  untrusted input — `__proto__`/`constructor` keys pollute. Check for
  `Object.create(null)`, `Map`, or a key allowlist.
- **JSON boundary.** Bound untrusted input and validate the parsed shape. Trace
  parse failures to an intentional error boundary; a local catch is not always
  required. `JSON.stringify` throws on cycles; `undefined` object properties are
  omitted, array entries become `null`, and a root `undefined` yields no JSON
  string. Flag a mismatch with the serialization contract.
- **Floating point / `NaN`.** Numeric parsing that accepts a forbidden prefix,
  radix, or truncated suffix; omitting `parseInt`'s radix alone is not a defect.
  `NaN` compared with `===`; money arithmetic on `number` where rounding error
  compounds.

## Do not flag

- `console.log` in CLI/tooling scripts — flag it in shipped libraries and
  request handlers.
- Missing `await` where the caller deliberately fire-and-forgets AND the
  rejection is handled or acceptable (e.g. telemetry) — flag only silent drops.
- Do not demand a lint-policy change merely because `exhaustive-deps` is off.
  A demonstrated stale closure remains a defect regardless of that setting.
- Non-null assertion (`!`) where an earlier guard establishes non-null —
  check the guard before flagging.
- `var`/function-style code in a file whose existing convention predates the
  diff — flag new code only.
