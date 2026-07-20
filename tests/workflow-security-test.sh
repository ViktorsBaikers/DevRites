#!/usr/bin/env bash
# workflow-security-test.sh — fixtures for validate-workflow-security.py:
# unpinned/broad/pull_request_target → findings; SHA-pinned + scoped → none.
# Plus a regression: the repo's own workflows must pass.
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

cat > "$TMP/unpinned-first-party.yml" <<'EOF'
name: bad-first-party
permissions:
  contents: read
jobs:
  a:
    steps:
      - uses: actions/checkout@v7
EOF

cat > "$TMP/noperm.yml" <<'EOF'
name: noperm
jobs:
  a:
    steps:
      - uses: actions/checkout@v7
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

clean "SHA-pinned + scoped"        "$TMP/clean.yml"
finds "unpinned third-party"       "$TMP/unpinned.yml"
finds "unpinned first-party"       "$TMP/unpinned-first-party.yml"
finds "no permissions block"       "$TMP/noperm.yml"
finds "write-all over-broad"       "$TMP/writeall.yml"
finds "pull_request_target"        "$TMP/prtarget.yml"
clean "Dependabot-only target without checkout" "$TMP/dependabot-target.yml"
finds "Dependabot target with checkout" "$TMP/dependabot-target-checkout.yml"
finds "Dependabot target with unguarded job" "$TMP/dependabot-target-unguarded-job.yml"

# regression: the repo's real workflows must pass
clean "repo workflows pass"        "$HERE/../.github/workflows"

if [ "$fail" -ne 0 ]; then echo "WORKFLOW-SECURITY TESTS: FAIL"; exit 1; fi
echo "WORKFLOW-SECURITY TESTS: PASS"
exit 0
