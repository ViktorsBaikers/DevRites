# Per-language review checklists

> Applies when: reviewing a diff that touches source files. Companion to
> [`code-review.md`](../code-review.md) — these are file-level probes, not a
> second severity scale.

Load `default.md` for every reviewed file, plus the file matching the file's
language. A file with no listed language gets `default.md` only.

| File pattern | Checklist |
|---|---|
| every source file | [`default.md`](default.md) |
| `*.go` | [`go.md`](go.md) |
| `*.py` | [`python.md`](python.md) |
| `*.ts`, `*.tsx`, `*.js`, `*.jsx`, `*.mjs`, `*.cjs` | [`ts-js.md`](ts-js.md) |
| `*.sh`, `*.bash`, `*.zsh`, PKGBUILD, Dockerfile RUN blocks | [`shell.md`](shell.md) |
| `.github/workflows/*.yml`, `.github/workflows/*.yaml`, `*.gitlab-ci.yml` | [`ci-workflows.md`](ci-workflows.md) |

Each checklist lists investigation probes, not automatic findings. Establish the
applicable contract, reachable path, and concrete effect before reporting a defect.
The "do not flag" sections prevent unsupported style or idiom complaints; they do
not excuse a demonstrated correctness or security failure. Assign severity under
the canonical review and security standards, never from a pattern match alone.
