#!/usr/bin/env bash
# Exercise validate-workflow-security.py with unsafe and safe fixtures. Unsafe
# workflows must produce findings; SHA-pinned actions with scoped permissions
# must pass. The repository workflows provide the regression case.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
VAL="python3 $HERE/../scripts/validate-workflow-security.py"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

finds() { # label file
  if $VAL "$2" >/dev/null 2>&1; then echo "FAIL [$1]: expected a finding"; fail=1; else echo "ok   [$1]"; fi
}
clean() { # label file
  if $VAL "$2" >/dev/null 2>&1; then echo "ok   [$1]"; else echo "FAIL [$1]: expected clean:"; $VAL "$2"; fail=1; fi
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
      - uses: actions/checkout@v7
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
    if: ${{ github.actor == 'dependabot[bot]' }}
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
    if: ${{ github.actor == 'dependabot[bot]' }}
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
    if: ${{ github.actor == 'dependabot[bot]' }}
    steps:
      - run: gh pr merge --auto "$PR_URL"
  unsafe:
    steps:
      - run: echo unguarded
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

clean "SHA-pinned + scoped"        "$TMP/clean.yml"
finds "unpinned third-party"       "$TMP/unpinned.yml"
finds "unquoted name colon"        "$TMP/unquoted-name-colon.yml"
finds "unpinned first-party"       "$TMP/unpinned-first-party.yml"
finds "unpinned action with spaced YAML key" "$TMP/unpinned-key-spacing.yml"
finds "no permissions block"       "$TMP/noperm.yml"
finds "one of two jobs inherits permissions" "$TMP/partially-scoped-jobs.yml"
clean "every job explicitly scopes permissions" "$TMP/all-jobs-scoped.yml"
finds "write-all over-broad"       "$TMP/writeall.yml"
finds "pull_request_target"        "$TMP/prtarget.yml"
clean "Dependabot-only target without checkout" "$TMP/dependabot-target.yml"
finds "Dependabot target with checkout" "$TMP/dependabot-target-checkout.yml"
finds "Dependabot target with unguarded job" "$TMP/dependabot-target-unguarded-job.yml"
finds "dispatch shell substitution in run" "$TMP/dispatch-shell-substitution.yml"
finds "dispatch input with spaced run key" "$TMP/dispatch-key-spacing.yml"
clean "dispatch input transported through env" "$TMP/dispatch-input-via-env.yml"

# The repository's workflows must pass too.
clean "repo workflows pass"        "$HERE/../.github/workflows"

if [ "$fail" -ne 0 ]; then echo "WORKFLOW-SECURITY TESTS: FAIL"; exit 1; fi
echo "WORKFLOW-SECURITY TESTS: PASS"
exit 0
