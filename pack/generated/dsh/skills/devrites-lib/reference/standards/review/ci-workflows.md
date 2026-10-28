# Review: CI workflows

> Applies when: reviewing `.github/workflows/*.yml`, `*.gitlab-ci.yml`, and
> equivalent pipeline definitions. Load with [`default.md`](default.md).

## Defect probes

- **Unpinned actions/images.** `uses: org/action@main`/`@master` or a mutable
  tag for third-party actions — supply-chain risk. Pin to a SHA or a semver
  tag per project policy; flag mutable refs on *untrusted* publishers.
- **`pull_request_target` / `issue_comment` triggers running untrusted code.**
  Checking out PR head code or evaluating `${{ github.event.* }}` fields in a
  privileged context (write token, secrets): trace execution, available authority,
  and reachable effect; assign severity under [`security.md`](../security.md).
- **Expression injection.** `${{ github.event.issue.title }}`,
  `github.head_ref`, PR body, or any attacker-controlled field interpolated
  into `run:` can turn data into executable syntax. Establish attacker control
  and impact before assigning severity; pass data through env vars and quote its
  use instead of interpolating it as script source.
- **Permissions.** Missing `permissions:` block (defaults to broad write on
  some configurations); `contents: write` or `packages: write` wider than the
  job needs; `id-token: write` without an OIDC consumer.
- **Secret exposure.** `secrets.X` in `run:` output or a step that echoes
  env/vars; secrets passed to `pull_request_target` or fork-triggered jobs;
  a secret in a URL or file that gets uploaded as an artifact.
- **Artifact poisoning.** `actions/upload-artifact` of a path that includes
  attacker-controlled files later consumed by a privileged job; cross-workflow
  `actions/download-artifact` without integrity or source checks.
- **Concurrency gaps.** Missing `concurrency:` on deploy/release jobs —
  overlapping runs corrupt state; `cancel-in-progress: true` on a deploy that
  must finish.
- **`if:` conditions on the wrong level.** A job-level `if` intended per-step;
  `always()` masking a needed `success()`/`failure()` gate.
- **Environment/approval bypass.** Deploy steps without `environment:` where
  the project requires protection rules; self-hosted runner labels on
  untrusted-triggered jobs.
- **Cache poisoning.** Cache keys that include attacker-controlled refs;
  restoring a cache built from PR code into a release/signing job.

## Do not flag

- `uses: actions/*@vN` semver tags on first-party/trusted publishers per
  project policy — mutable SHA-pinning is a hardening Suggestion, not Critical,
  unless the policy requires pinning.
- Missing `permissions:` when `GITHUB_TOKEN` default is read-only in the repo
  settings — verify the repo setting before flagging.
- `${{ secrets.X }}` as a step `env:` value consumed by a trusted action —
  flag only paths to attacker-visible output.
- Long `run:` blocks that a lint step (`actionlint`, `yamllint`) already
  covers — do not duplicate lint findings.
