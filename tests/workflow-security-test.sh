#!/usr/bin/env bash
# Exercise validate-workflow-security.py with unsafe and safe fixtures. Unsafe
# workflows must produce findings; SHA-pinned actions with scoped permissions
# must pass. The repository workflows provide the regression case.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
VALIDATOR="$HERE/../scripts/validate-workflow-security.py"
VAL=(python3 "$VALIDATOR")
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

finds() { # label file expected-finding-substring
  local out rc=0
  out="$(python3 "$VALIDATOR" "$2" 2>&1)" || rc=$?
  if [ "$rc" -eq 0 ]; then
    echo "FAIL [$1]: expected a finding"; fail=1
  elif printf '%s\n' "$out" | grep -q 'Traceback'; then
    echo "FAIL [$1]: validator crashed"; fail=1
  elif ! printf '%s\n' "$out" | grep '^FINDING ' | grep -qF -- "$3"; then
    echo "FAIL [$1]: no FINDING line containing '$3' (exit $rc)"; fail=1
  else
    echo "ok   [$1]"
  fi
}
clean() { # label file
  if "${VAL[@]}" "$2" >/dev/null 2>&1; then echo "ok   [$1]"; else echo "FAIL [$1]: expected clean:"; "${VAL[@]}" "$2"; fail=1; fi
}

SHA=773744901bac0e8cbb5a0dc842800d45e9b2b405

cat > "$TMP/clean.yml" <<EOF
name: ok
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses: actions/checkout@$SHA # v7
      - uses: marocchino/sticky-pull-request-comment@$SHA # SHA-pinned fixture
EOF

cat > "$TMP/unpinned.yml" <<'EOF'
name: bad
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses: marocchino/sticky-pull-request-comment@v2
EOF

cat > "$TMP/unquoted-name-colon.yml" <<'EOF'
name: bad
permissions:
  contents: read
jobs:
  a:
    steps:
      - name: Security scan: BLOCKING
        run: echo unreachable
EOF

cat > "$TMP/unpinned-first-party.yml" <<'EOF'
name: bad-first-party
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses: actions/checkout@v7
EOF

cat > "$TMP/unpinned-key-spacing.yml" <<'EOF'
name: bad-key-spacing
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses : actions/checkout@main
EOF

cat > "$TMP/noperm.yml" <<'EOF'
name: noperm
jobs:
  a:
    steps:
      - uses: actions/checkout@v7
EOF

cat > "$TMP/partially-scoped-jobs.yml" <<EOF
name: partially-scoped
jobs:
  scoped:
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@$SHA
  inherited:
    steps:
      - uses: actions/checkout@$SHA
EOF

cat > "$TMP/all-jobs-scoped.yml" <<EOF
name: all-jobs-scoped
jobs:
  first:
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@$SHA
  second:
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@$SHA
EOF

cat > "$TMP/writeall.yml" <<'EOF'
name: broad
permissions: write-all
jobs:
  a:
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
EOF

cat > "$TMP/prtarget.yml" <<'EOF'
name: risky
on: pull_request_target
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses: actions/checkout@v7
EOF

cat > "$TMP/dependabot-target.yml" <<'EOF'
name: safe-dependabot-write
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-checkout.yml" <<'EOF'
name: unsafe-dependabot-checkout
on: pull_request_target
permissions:
  contents: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - uses: actions/checkout@v7
EOF

cat > "$TMP/dependabot-target-unguarded-job.yml" <<'EOF'
name: unsafe-extra-job
on: pull_request_target
permissions:
  contents: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - run: gh pr merge --auto "$PR_URL"
  unsafe:
    steps:
      - run: echo unguarded
EOF

cat > "$TMP/dependabot-target-actor.yml" <<'EOF'
name: unsafe-dependabot-actor
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.actor == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-actor-or.yml" <<'EOF'
name: unsafe-actor-or
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' || github.actor == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-or-true.yml" <<'EOF'
name: unsafe-or-true
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' || true }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-and-true.yml" <<'EOF'
name: unsafe-and-true
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' && true }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-bare-or.yml" <<'EOF'
name: unsafe-bare-or
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: github.event.pull_request.user.login == 'dependabot[bot]' || true
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-step-only.yml" <<'EOF'
name: unsafe-step-only-gate
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    steps:
      - name: gated step only
        if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
        run: gh pr merge --auto "$PR_URL"
      - name: ungated step
        run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dependabot-target-bare.yml" <<'EOF'
name: safe-bare-gate
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: github.event.pull_request.user.login == 'dependabot[bot]'
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/dependabot-target-quoted-job-id.yml" <<'EOF'
name: unsafe-quoted-job-id
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
  "unsafe":
    steps:
      - run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dependabot-target-flow-job.yml" <<'EOF'
name: unsafe-flow-job
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
  unsafe: {runs-on: ubuntu-latest, steps: [{run: gh pr merge --auto "$PR_URL"}]}
EOF

cat > "$TMP/dependabot-target-env-named-if.yml" <<'EOF'
name: unsafe-env-named-if
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
   env:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
   steps:
   - run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dependabot-target-step-if-at-job-indent.yml" <<'EOF'
name: unsafe-step-if-at-job-indent
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
   steps:
   -
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    run: echo gated
   - run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dependabot-target-hash-suffix.yml" <<'EOF'
name: unsafe-hash-suffix
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}#|| true
    steps:
      - run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dependabot-target-bare-hash-suffix.yml" <<'EOF'
name: unsafe-bare-hash-suffix
on: pull_request_target
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}#
    steps:
      - run: gh pr merge --auto "$PR_URL"
EOF

cat > "$TMP/dispatch-shell-substitution.yml" <<'EOF'
name: unsafe-dispatch-shell-substitution
on:
  workflow_dispatch:
    inputs:
      model:
        default: '$(touch "$RUNNER_TEMP/dispatched")'
permissions:
  contents: read
jobs:
  live:
    runs-on: [self-hosted, linux]
    environment: live-evals
    steps:
      - name: Unsafe direct interpolation
        run: |
          python3 runner.py --model "${{ github.event.inputs.model }}"
EOF

cat > "$TMP/dispatch-input-via-env.yml" <<'EOF'
name: safe-dispatch-input-via-env
on:
  workflow_dispatch:
    inputs:
      model:
        default: ''
permissions:
  contents: read
jobs:
  live:
    runs-on: [self-hosted, linux]
    environment: live-evals
    steps:
      - name: Quoted environment transport
        env:
          MODEL: ${{ inputs.model }}
        run: python3 runner.py --model "$MODEL"
EOF

cat > "$TMP/dispatch-key-spacing.yml" <<'EOF'
name: unsafe-dispatch-key-spacing
on:
  workflow_dispatch:
    inputs:
      model:
        default: ''
permissions:
  contents: read
jobs:
  live:
    steps:
      - run : python3 runner.py --model "${{ inputs.model }}"
EOF

cat > "$TMP/escaped-trigger-list-actor.yml" <<'EOF'
name: esc-u
on: ["pull_request_\u0074arget"]
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    runs-on: ubuntu-latest
    if: ${{ github.actor == 'dependabot[bot]' }}
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
        with:
          ref: ${{ github.event.pull_request.head.sha }}
      - run: make
EOF

cat > "$TMP/escaped-trigger-mapping.yml" <<'EOF'
name: esc-x
on:
  "pull_request_\x74arget":
    types: [opened]
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    runs-on: ubuntu-latest
    if: github.actor == 'dependabot[bot]'
    steps:
      - run: echo hi
EOF

cat > "$TMP/escaped-trigger-string-fold.yml" <<'EOF'
name: esc-nl
on: "pull_request_\
  target"
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
        with:
          ref: ${{ github.event.pull_request.head.sha }}
EOF

cat > "$TMP/quoted-on-list-trigger.yml" <<'EOF'
name: on-list
"on": [push, pull_request_target]
permissions:
  contents: write
  pull-requests: write
jobs:
  m:
    runs-on: ubuntu-latest
    if: github.actor == 'dependabot[bot]'
    steps:
      - run: echo
EOF

cat > "$TMP/on-mapping-trigger.yml" <<'EOF'
name: on-map
on:
  pull_request_target: {types: [opened]}
permissions:
  contents: write
  pull-requests: write
jobs:
  m:
    runs-on: ubuntu-latest
    steps:
      - run: echo
EOF

cat > "$TMP/escaped-trigger-gated.yml" <<'EOF'
name: safe-escaped-trigger
on: ["pull_request_\u0074arget"]
permissions:
  contents: write
  pull-requests: write
jobs:
  merge:
    if: ${{ github.event.pull_request.user.login == 'dependabot[bot]' }}
    steps:
      - uses: dependabot/fetch-metadata@773744901bac0e8cbb5a0dc842800d45e9b2b405
EOF

cat > "$TMP/commitlint-edited-other-trigger.yml" <<'EOF'
name: edited-on-wrong-trigger
on:
  pull_request:
    types: [opened]
  issues:
    types: [edited]
permissions:
  contents: read
jobs:
  lint:
    steps:
      - env:
          TITLE: ${{ github.event.pull_request.title }}
        run: npx --no-install commitlint --verbose
EOF

cat > "$TMP/commitlint-edited-pull-request.yml" <<'EOF'
name: edited-on-pull-request
on:
  pull_request:
    types: [opened, edited]
permissions:
  contents: read
jobs:
  lint:
    steps:
      - env:
          TITLE: ${{ github.event.pull_request.title }}
        run: npx --no-install commitlint --verbose
EOF

clean "SHA-pinned + scoped"        "$TMP/clean.yml"
finds "unpinned third-party"       "$TMP/unpinned.yml" "is not pinned to a full commit SHA"
finds "unquoted name colon"        "$TMP/unquoted-name-colon.yml" "name has an unquoted colon"
finds "unpinned first-party"       "$TMP/unpinned-first-party.yml" "is not pinned to a full commit SHA"
finds "unpinned action with spaced YAML key" "$TMP/unpinned-key-spacing.yml" "is not pinned to a full commit SHA"
finds "no permissions block"       "$TMP/noperm.yml" "jobs without explicit permissions: a."
finds "one of two jobs inherits permissions" "$TMP/partially-scoped-jobs.yml" "jobs without explicit permissions: inherited"
clean "every job explicitly scopes permissions" "$TMP/all-jobs-scoped.yml"
finds "write-all over-broad"       "$TMP/writeall.yml" "permissions: write-all grants too much access"
finds "pull_request_target"        "$TMP/prtarget.yml" "pull_request_target exposes secrets"
clean "Dependabot-only target without checkout" "$TMP/dependabot-target.yml"
finds "Dependabot target gated on github.actor" "$TMP/dependabot-target-actor.yml" "pull_request_target exposes secrets"
finds "Dependabot gate widened with || github.actor" "$TMP/dependabot-target-actor-or.yml" "pull_request_target exposes secrets"
finds "Dependabot gate widened with || true" "$TMP/dependabot-target-or-true.yml" "pull_request_target exposes secrets"
finds "Dependabot gate padded with && true" "$TMP/dependabot-target-and-true.yml" "pull_request_target exposes secrets"
finds "Dependabot bare gate widened with || true" "$TMP/dependabot-target-bare-or.yml" "pull_request_target exposes secrets"
finds "Dependabot gate on a step only" "$TMP/dependabot-target-step-only.yml" "pull_request_target exposes secrets"
clean "Dependabot bare job-level gate" "$TMP/dependabot-target-bare.yml"
finds "Dependabot target with checkout" "$TMP/dependabot-target-checkout.yml" "pull_request_target exposes secrets"
finds "Dependabot gate with a quoted ungated job ID" "$TMP/dependabot-target-quoted-job-id.yml" "pull_request_target exposes secrets"
finds "Dependabot gate with a flow-style ungated job" "$TMP/dependabot-target-flow-job.yml" "pull_request_target exposes secrets"
finds "Dependabot gate text in an env var named if" "$TMP/dependabot-target-env-named-if.yml" "pull_request_target exposes secrets"
finds "Dependabot gate on a step whose if sits at 4 spaces" "$TMP/dependabot-target-step-if-at-job-indent.yml" "pull_request_target exposes secrets"
finds "Dependabot gate with }}#|| true suffix" "$TMP/dependabot-target-hash-suffix.yml" "pull_request_target exposes secrets"
finds "Dependabot gate with }}# suffix" "$TMP/dependabot-target-bare-hash-suffix.yml" "pull_request_target exposes secrets"
finds "Dependabot target with unguarded job" "$TMP/dependabot-target-unguarded-job.yml" "pull_request_target exposes secrets"
finds "escaped pull_request_target in a list with an actor gate" "$TMP/escaped-trigger-list-actor.yml" "pull_request_target exposes secrets"
finds "escaped pull_request_target mapping key without a gate" "$TMP/escaped-trigger-mapping.yml" "pull_request_target exposes secrets"
finds "escaped pull_request_target string folded across lines" "$TMP/escaped-trigger-string-fold.yml" "pull_request_target exposes secrets"
finds "pull_request_target in a quoted on list" "$TMP/quoted-on-list-trigger.yml" "pull_request_target exposes secrets"
finds "pull_request_target as an on mapping key" "$TMP/on-mapping-trigger.yml" "pull_request_target exposes secrets"
clean "escaped pull_request_target with the exact gate" "$TMP/escaped-trigger-gated.yml"
finds "edited listed under a different trigger" "$TMP/commitlint-edited-other-trigger.yml" "trigger on pull_request type 'edited'"
clean "edited listed under pull_request" "$TMP/commitlint-edited-pull-request.yml"
finds "dispatch shell substitution in run" "$TMP/dispatch-shell-substitution.yml" "workflow_dispatch input appears directly in run"
finds "dispatch input with spaced run key" "$TMP/dispatch-key-spacing.yml" "workflow_dispatch input appears directly in run"
clean "dispatch input transported through env" "$TMP/dispatch-input-via-env.yml"

# Release auth: NPM_TOKEN may only come from GitHub Actions secrets (no literals).
CIYML="$HERE/../.github/workflows/ci.yml"
if grep -n "NPM_TOKEN: \${{ secrets.NPM_TOKEN }}" "$CIYML" >/dev/null; then
  echo "ok   [ci.yml wires secrets.NPM_TOKEN for semantic-release]"
else
  echo "FAIL [ci.yml missing secrets.NPM_TOKEN for semantic-release]"; fail=1
fi
if grep -nE '^[[:space:]]*NPM_TOKEN:' "$CIYML" | grep -v 'secrets\.NPM_TOKEN' >/dev/null; then
  echo "FAIL [ci.yml appears to hardcode an NPM_TOKEN value]"; fail=1
else
  echo "ok   [ci.yml does not hardcode NPM_TOKEN]"
fi
if grep -n 'secrets.NPM_TOKEN' "$HERE/../docs/release.md" >/dev/null; then
  echo "ok   [docs/release.md documents secrets.NPM_TOKEN publish fallback]"
else
  echo "FAIL [docs/release.md missing secrets.NPM_TOKEN publish fallback]"; fail=1
fi

# setup-node v6 treats `cache` as a package-manager name. `cache: false` becomes
# Caching for 'false' is not supported and aborts semantic-release on main.
if awk '
  /uses:[[:space:]]*actions\/setup-node@/ {in_node=1; next}
  in_node && /^[[:space:]]*-[[:space:]]/ {in_node=0}
  in_node && /^[[:space:]]*[A-Za-z0-9_-]+:/ && $0 !~ /^[[:space:]]{2,}(with|node-version|registry-url|cache|cache-dependency-path|package-manager-cache|check-latest|token|always-auth|scope|architecture|mirror|mirror-url):/ {in_node=0}
  in_node && /^[[:space:]]*cache:[[:space:]]*false[[:space:]]*$/ {bad=1}
  END {exit bad ? 0 : 1}
' "$CIYML"; then
  echo "FAIL [ci.yml setup-node uses invalid cache: false]"; fail=1
else
  echo "ok   [ci.yml setup-node does not use cache: false]"
fi
if awk '
  /^[[:space:]]*name:[[:space:]]*semantic-release[[:space:]]*$/ {in_rel=1; next}
  in_rel && /^[A-Za-z0-9_-]+:/ {in_rel=0}
  in_rel && /^[[:space:]]*package-manager-cache:[[:space:]]*false[[:space:]]*$/ {found=1}
  END {exit found ? 0 : 1}
' "$CIYML"; then
  echo "ok   [semantic-release disables package-manager-cache]"
else
  echo "FAIL [semantic-release missing package-manager-cache: false]"; fail=1
fi

# Nightly fuzz is bounded and not a required PR check.
FUZZ="$HERE/../.github/workflows/engine-fuzz.yml"
if grep -q '^[[:space:]]*pull_request:' "$FUZZ"; then
  echo "FAIL [engine-fuzz.yml must not run on pull_request]"; fail=1
else
  echo "ok   [engine-fuzz.yml is schedule/dispatch only]"
fi
if grep -q 'fuzztime=20s' "$FUZZ" && grep -q 'FuzzWithinResolved' "$FUZZ" && grep -q 'FuzzWorkspaceSchemaRow' "$FUZZ"; then
  echo "ok   [engine-fuzz.yml names each fuzz target at 20s]"
else
  echo "FAIL [engine-fuzz.yml missing named 20s fuzz targets]"; fail=1
fi
if grep -q 'persist-credentials: false' "$FUZZ"; then
  echo "ok   [engine-fuzz.yml persist-credentials false]"
else
  echo "FAIL [engine-fuzz.yml missing persist-credentials: false]"; fail=1
fi

# CodeQL covers installer/scripts JS without a compiled build.
CODEQL="$HERE/../.github/workflows/codeql.yml"
if grep -q 'javascript-typescript' "$CODEQL" && grep -q 'build-mode: none' "$CODEQL"; then
  echo "ok   [codeql.yml analyzes javascript-typescript]"
else
  echo "FAIL [codeql.yml missing javascript-typescript / build-mode none]"; fail=1
fi
# Scan scope is the analyze-js `paths:` list only; `paths-ignore:` entries do not count.
CODEQL_SCAN_PATHS=$(awk '
  /^[[:space:]]*analyze-js:[[:space:]]*(#.*)?$/ && !job { match($0, /[^ ]/); jind = RSTART; job = 1; next }
  job && !/^[[:space:]]*(#.*)?$/ { match($0, /[^ ]/); if (RSTART <= jind) exit }
  job && !inlist && /^[[:space:]]*paths:[[:space:]]*(#.*)?$/ { match($0, /[^ ]/); ind = RSTART; inlist = 1; next }
  inlist && /^[[:space:]]*(#.*)?$/ { next }
  inlist { match($0, /[^ ]/); if (RSTART <= ind || $0 !~ /^[[:space:]]*-/) exit; print }
' "$CODEQL")
if printf '%s\n' "$CODEQL_SCAN_PATHS" | grep -Eq '^[[:space:]]*-[[:space:]]*bin[[:space:]]*(#.*)?$' &&
   printf '%s\n' "$CODEQL_SCAN_PATHS" | grep -Eq '^[[:space:]]*-[[:space:]]*scripts[[:space:]]*(#.*)?$'; then
  echo "ok   [codeql.yml JS scan is scoped to bin and scripts]"
else
  echo "FAIL [codeql.yml JS paths are not scoped to bin and scripts]"; fail=1
fi

# Windows -race is nightly/dispatch only, not a required PR check.
WINRACE="$HERE/../.github/workflows/engine-windows-race.yml"
if grep -q '^[[:space:]]*pull_request:' "$WINRACE"; then
  echo "FAIL [engine-windows-race.yml must not run on pull_request]"; fail=1
else
  echo "ok   [engine-windows-race.yml is schedule/dispatch only]"
fi
if grep -q -- '-race' "$WINRACE" && grep -q 'windows-latest' "$WINRACE" && grep -q 'persist-credentials: false' "$WINRACE"; then
  echo "ok   [engine-windows-race.yml runs go test -race on windows]"
else
  echo "FAIL [engine-windows-race.yml missing windows -race job]"; fail=1
fi

# The repository's workflows must pass too.
clean "repo workflows pass"        "$HERE/../.github/workflows"

if [ "$fail" -ne 0 ]; then echo "WORKFLOW-SECURITY TESTS: FAIL"; exit 1; fi
echo "WORKFLOW-SECURITY TESTS: PASS"
exit 0
