#!/usr/bin/env bash
# Emit GitHub Actions outputs for CI path scoping on pull requests.
# On push/workflow_dispatch, always runs the full matrix.
set -euo pipefail

run_engine=true
run_pack_evals=true
run_full=true
run_tests=true

# Docs-only PRs skip validate and the shell suite (job-level `if:` still
# reports success for required checks). Deny-by-default allowlist: any path
# outside these prefixes keeps validate.
docs_only_allowlist='^(README\.md|CHANGELOG\.md|LICENSE|CONTRIBUTING\.md|docs/|\.scratch/)'

# Engine-only PRs still run validate (OSV/go.mod, workflow pins) and the
# engine jobs, but skip the 8 shell shards. Deny-by-default: any path
# outside engine/ keeps the shards.
engine_only_allowlist='^engine/'

changed=()

if [[ "${GITHUB_EVENT_NAME:-}" == "pull_request" ]]; then
  if [[ -n "${DEVRITES_CI_CHANGED_PATHS:-}" ]]; then
    mapfile -t changed < <(printf '%s\n' "$DEVRITES_CI_CHANGED_PATHS")
  else
    base="${GITHUB_BASE_REF:-main}"
    git fetch --no-tags --depth=1 origin "${base}" 2>/dev/null || true
    if git rev-parse "origin/${base}" >/dev/null 2>&1; then
      mapfile -t changed < <(git diff --name-only "origin/${base}...HEAD")
    fi
  fi
  engine=false
  pack=false
  for path in "${changed[@]}"; do
    [[ -z "$path" ]] && continue
    case "$path" in
      engine/*|.github/workflows/ci.yml|scripts/build-binaries.sh|scripts/build-release-tarball.sh)
        engine=true ;;
      pack/*|evals/*|scripts/validate.sh|scripts/run-evals.sh|scripts/run-outcome-evals.sh|scripts/run-behavioral-evals.sh|scripts/check-*)
        pack=true ;;
    esac
  done
  if [[ "$engine" == false ]]; then
    run_engine=false
  fi
  if [[ "$pack" == false ]]; then
    run_pack_evals=false
  fi
  if [[ "${#changed[@]}" -gt 0 ]]; then
    docs_only=true
    engine_only=true
    for path in "${changed[@]}"; do
      [[ -z "$path" ]] && continue
      if ! [[ "$path" =~ $docs_only_allowlist ]]; then
        docs_only=false
      fi
      if ! [[ "$path" =~ $engine_only_allowlist ]]; then
        engine_only=false
      fi
    done
    if [[ "$docs_only" == true ]]; then
      run_full=false
      run_tests=false
    elif [[ "$engine_only" == true ]]; then
      run_tests=false
    fi
  fi
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "run_engine=${run_engine}" >>"$GITHUB_OUTPUT"
  echo "run_pack_evals=${run_pack_evals}" >>"$GITHUB_OUTPUT"
  echo "run_full=${run_full}" >>"$GITHUB_OUTPUT"
  echo "run_tests=${run_tests}" >>"$GITHUB_OUTPUT"
else
  echo "run_engine=${run_engine}"
  echo "run_pack_evals=${run_pack_evals}"
  echo "run_full=${run_full}"
  echo "run_tests=${run_tests}"
fi
