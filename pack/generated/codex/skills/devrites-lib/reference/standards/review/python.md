# Review: Python

> Applies when: reviewing `*.py` files. Load with [`default.md`](default.md).

## Defect probes

- **Mutable default argument.** `def f(x=[])` / `def f(x={})` — the default is
  created once and shared across calls. Flag unless the body never mutates it
  AND the intent is a shared memo (rare; usually a defect).
- **Bare / broad `except`.** `except:` or `except Exception` that swallows the
  error, `pass`, or re-raises a different exception losing the original
  (`raise X` inside `except` without `from`).
- **Resource lifecycle.** `open()` without `with`; subprocess without
  `timeout`; a `requests` call without `timeout` — it can hang forever.
- **`==` vs `is`.** `is` used for value comparison (`x is "foo"`, `x is 5`);
  `==` used against `None`/singletons where `is` is required for correctness
  against `__eq__` overrides.
- **Late-binding closure.** Loop variable captured in a lambda/def inside a
  loop without a default-arg or `functools.partial` bind — all closures see the
  final value.
- **Mutable class attribute.** A list/dict/set defined at class level meant to
  be per-instance — shared mutation across instances.
- **Shadowed builtin / import side effects.** A variable named `id`, `type`,
  `list`, `input` masking a builtin used later in scope; a module that runs
  real work (network, filesystem) at import time.
- **`str.format`/f-string on untrusted input** into `eval`, `exec`,
  `subprocess(shell=True)`, `os.system`, or SQL — injection, always
  Critical/Important.
- **Type coercion traps.** `datetime` naive vs aware mixing; `int`/`str`
  comparison (`"5" > 10` raises in py3, silent in dict keys); `dict.get`
  default masking a missing required key.
- **Async misuse.** `async def` called without `await`/task scheduling;
  blocking call (`time.sleep`, sync requests) inside `async def`; shared state
  mutated across `await` points assuming atomicity.

## Do not flag

- `except Exception` that re-raises, logs with traceback, or converts to a
  domain error preserving `__cause__`.
- `print` in a CLI script's `main` — flag it in library code, not entry points.
- Missing type hints in a codebase that does not enforce them — check
  `pyproject`/`mypy` config before flagging.
- `getattr`/`hasattr` defensive access where the attribute is genuinely
  optional.
- Snake_case/camelCase mix matching the file's own convention — flag only when
  the diff introduces a *new* convention.
