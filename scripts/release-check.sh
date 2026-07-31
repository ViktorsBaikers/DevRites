#!/usr/bin/env bash
# Gather release evidence without modifying the checkout. These are the same
# checks users run through npx; there is no Claude or Codex plugin path.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
cd "$ROOT"

ok() { printf 'ok: %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing tool: $1" >&2; exit 1; }; }

need bash
need node
need python3

bash scripts/validate.sh >/tmp/devrites-release-validate.log
ok "scripts/validate.sh"

bash tests/npx-pack-smoke.sh >/tmp/devrites-release-npx-pack-smoke.log
ok "npx-pack-smoke"

bash tests/install-smoke.sh >/tmp/devrites-release-install.log
bash tests/update-smoke.sh >/tmp/devrites-release-update.log
bash tests/uninstall-smoke.sh >/tmp/devrites-release-uninstall.log
ok "install/update/uninstall smoke"

bash scripts/run-behavioral-evals.sh >/tmp/devrites-release-behavioral.log
ok "behavioral eval schema"

bash tests/release-tarball-test.sh >/tmp/devrites-release-tarball.log
ok "release tarball reproducible and confined"

if compgen -G "dist/devrites-v*.tar.gz" >/dev/null; then
  for tarball in dist/devrites-v*.tar.gz; do
    [[ -f "$tarball.sha256" ]] || { echo "missing checksum: $tarball.sha256" >&2; exit 1; }
  done
  ok "release tarball checksums present"
else
  echo "skip: no dist/devrites-v*.tar.gz tarball built yet"
fi

python3 - <<'PY' >/tmp/devrites-release-plugin-refs.log
from pathlib import Path
bad = []
for root in [Path('README.md'), Path('install.sh'), Path('update.sh'), Path('bin'), Path('package.json')]:
    files = [root] if root.is_file() else [p for p in root.rglob('*') if p.is_file()]
    for p in files:
        text = p.read_text(errors='ignore').splitlines()
        for i, line in enumerate(text, 1):
            l = line.lower()
            if not (('plugin' in l or 'marketplace' in l) and ('install' in l or 'distributed' in l or 'shipped' in l)):
                continue
            if any(neg in l for neg in [' not ', ' no ', 'not through', 'not shipped', 'not distributed', 'no claude/codex']):
                continue
            bad.append(f'{p}:{i}:{line}')
if bad:
    print('\n'.join(bad))
raise SystemExit(1 if bad else 0)
PY
if [[ -s /tmp/devrites-release-plugin-refs.log ]]; then
  echo "unexpected plugin/marketplace install language (DevRites ships by npx only):" >&2
  cat /tmp/devrites-release-plugin-refs.log >&2
  exit 1
fi
ok "install docs stay npx-only, no Claude/Codex plugin path"

printf '\nrelease-check: OK\n'
