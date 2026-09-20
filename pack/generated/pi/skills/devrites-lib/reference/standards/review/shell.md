# Review: Shell

> Applies when: reviewing `*.sh`, `*.bash`, `*.zsh`, PKGBUILD, or `RUN` blocks.
> Load with [`default.md`](default.md).

## Defect probes

- **Unquoted expansion.** `$var`, `$(cmd)`, `$@` unquoted where word splitting
  or glob expansion corrupts the value — paths with spaces are the canonical
  failure. `"$@"` is the only correct arg-forwarding form.
- **Missing `set -euo pipefail`** in a new script, or a pipeline whose middle
  stage can fail silently (`cmd1 | cmd2` — `pipefail` or check
  `PIPESTATUS`/`PIPESTATUS`).
- **`cd` without failure handling.** `cd dir && rm -rf ...` is safe; `cd dir`
  then a bare destructive command runs in the wrong directory on `cd` failure.
- **`rm -rf` on a variable path.** Unquoted or possibly-empty expansion —
  `rm -rf "$DIR/"` where `DIR` can be empty or `/` is Critical.
- **`eval` or `bash -c` on composed input.** String-built commands break on
  quoting and inject on untrusted values.
- **Test operators.** `[ $x = y ]` unquoted (fails on empty/spaces); `==`
  inside `[ ]` (POSIX wants `=`); `[` vs `[[` mixed; `-n`/`-z` on unquoted vars.
- **Signal/cleanup.** Temp files, background jobs, or traps not cleaned on
  error/EXIT; `trap` overwriting an earlier trap.
- **Exit-code assumptions.** `$?` read after another command has run;
  `local x=$(cmd)` masks `cmd`'s exit code under `set -e` (assignment succeeds
  even when `cmd` fails).
- **Portability claims.** Bashisms (`[[`, arrays, `<<<`, `${var^^}`) in a
  `#!/bin/sh` script — check the shebang before flagging.
- **Here-doc / newline handling.** `while read` without `IFS=` or `-r` mangling
  input; `read` inside a pipeline subshell losing variable assignments.

## Do not flag

- Unquoted `$var` where splitting is intended (`$CFLAGS`, arg lists) — flag
  only when a file path or user value is at risk.
- Missing `set -u` in a file that intentionally probes unset vars.
- `[ ]` POSIX tests in a file targeting `/bin/sh` — correct for the target.
- `echo` vs `printf` when the output is a fixed literal — flag `echo` only on
  variable/escape-bearing output.
- Shellcheck-suppressed lines with a named reason — judge the reason.
