#!/usr/bin/env bash
# Cloud Agent install: prepare the DevRites development environment.
# Idempotent and non-interactive; safe to run repeatedly.
set -euo pipefail

cd "$(dirname "$0")/.."

# 1. System packages the default image lacks.
#    - gawk: install.sh's archive preflight uses POSIX interval regexes
#      (e.g. `\.{1,2}`) that panic under Debian/Ubuntu's default mawk
#      ("REcompile() - panic"); GitHub CI runners default to gawk, so match them.
#    - shellcheck: scripts/validate.sh enforces the same error-level shellcheck
#      gate that CI runs on every shell script.
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gawk shellcheck
sudo update-alternatives --set awk /usr/bin/gawk

# 2. Node toolchain: husky + commitlint + semantic-release + the test runner deps.
npm ci

# 3. Warm the Go toolchain pinned by engine/go.mod (go 1.26 / toolchain go1.26.7)
#    and build the control-plane engine once, so the first `npm test` or engine
#    invocation does not pay the toolchain download + compile cost.
( cd engine && CGO_ENABLED=0 go build -o /tmp/devrites-engine-warm . && rm -f /tmp/devrites-engine-warm )
