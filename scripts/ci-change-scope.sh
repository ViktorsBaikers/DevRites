#!/usr/bin/env bash
# Emit GitHub Actions outputs for CI path scoping on pull requests.
# On push/workflow_dispatch, always runs the full matrix.
set -euo pipefail

run_engine=true
run_pack_evals=true
run_full=true
run_tests=true
run_deps=true
run_docs=true

# Docs-only PRs skip the shell suite shards (job-level `if:` still reports
# success for required checks). run_full=false only when every path is
# allowlisted, and `validate` also runs when run_docs=true, so in practice only
# LICENSE-only or .scratch/-only PRs skip validate. Deny-by-default allowlist:
# any path outside these prefixes keeps validate. docs/engine/ is read by
# engine tests, so it never counts as docs-only. run_docs is true for any
# README/CHANGELOG/CONTRIBUTING/docs/ change.
docs_only_allowlist='^(README\.md|CHANGELOG\.md|LICENSE|CONTRIBUTING\.md|docs/|\.scratch/)'

changed=()
listed=false

if [[ "${GITHUB_EVENT_NAME:-}" == "pull_request" ]]; then
  if [[ -n "${DEVRITES_CI_CHANGED_PATHS:-}" ]]; then
    mapfile -t changed < <(printf '%s\n' "$DEVRITES_CI_CHANGED_PATHS")
    listed=true
  else
    base="${GITHUB_BASE_REF:-main}"
    git fetch --no-tags origin "${base}" 2>/dev/null || true
    if git rev-parse --verify --quiet "origin/${base}^{commit}" >/dev/null 2>&1 &&
      git merge-base "origin/${base}" HEAD >/dev/null 2>&1; then
      if diff_out=$(git diff --name-only "origin/${base}...HEAD"); then
        listed=true
        if [[ -n "$diff_out" ]]; then
          mapfile -t changed < <(printf '%s\n' "$diff_out")
        fi
      fi
    fi
  fi
  # Unknown file list: keep the full matrix. An empty or failed
  # origin/<base>...HEAD (no merge-base on a shallow clone) is not "no engine
  # changes" — that skip is what let gosec fail only after merge to main.
  if [[ "$listed" == true ]]; then
    engine=false
    pack=false
    deps=false
    docs=false
    for path in "${changed[@]}"; do
      [[ -z "$path" ]] && continue
      case "$path" in
        engine/*|.github/workflows/ci.yml|scripts/build-binaries.sh|scripts/build-release-tarball.sh)
          engine=true ;;
        pack/*|evals/*|scripts/validate.sh|scripts/run-evals.sh|scripts/run-outcome-evals.sh|scripts/run-behavioral-evals.sh|scripts/check-*)
          pack=true ;;
      esac
      # Engine tests and the npm launcher read these pack docs and launcher.
      case "$path" in
        bin/devrites.mjs|pack/.claude/skills/devrites-lib/reference/*|pack/.claude/skills/rite-spec/reference/*)
          engine=true ;;
      esac
      case "$path" in
        README.md|CHANGELOG.md|CONTRIBUTING.md|docs/*) docs=true ;;
      esac
      # Engine tests read docs/engine/.
      case "$path" in
        docs/engine/*) engine=true ;;
      esac
      # Dependency advisory gates (npm audit + OSV) judge the PR's own
      # dependency inputs; scheduled deps-scan.yml catches newly published
      # advisories against unchanged ones.
      case "$path" in
        package.json|package-lock.json|engine/go.mod|engine/go.sum|osv-scanner.toml|scripts/npm-audit-exceptions.json|scripts/check-npm-audit.mjs|scripts/ci-install-validate-tools.sh)
          deps=true ;;
      esac
    done
    if [[ "$engine" == false ]]; then
      run_engine=false
    fi
    if [[ "$pack" == false ]]; then
      run_pack_evals=false
    fi
    if [[ "$deps" == false ]]; then
      run_deps=false
    fi
    if [[ "$docs" == false ]]; then
      run_docs=false
    fi
    if [[ "${#changed[@]}" -gt 0 ]]; then
      docs_only=true
      for path in "${changed[@]}"; do
        [[ -z "$path" ]] && continue
        if ! [[ "$path" =~ $docs_only_allowlist ]] || [[ "$path" == docs/engine/* ]]; then
          docs_only=false
        fi
      done
      if [[ "$docs_only" == true ]]; then
        run_full=false
        run_tests=false
      fi
    fi
  fi
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "run_engine=${run_engine}" >>"$GITHUB_OUTPUT"
  echo "run_pack_evals=${run_pack_evals}" >>"$GITHUB_OUTPUT"
  echo "run_full=${run_full}" >>"$GITHUB_OUTPUT"
  echo "run_tests=${run_tests}" >>"$GITHUB_OUTPUT"
  echo "run_deps=${run_deps}" >>"$GITHUB_OUTPUT"
  echo "run_docs=${run_docs}" >>"$GITHUB_OUTPUT"
else
  echo "run_engine=${run_engine}"
  echo "run_pack_evals=${run_pack_evals}"
  echo "run_full=${run_full}"
  echo "run_tests=${run_tests}"
  echo "run_deps=${run_deps}"
  echo "run_docs=${run_docs}"
fi
