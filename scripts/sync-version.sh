#!/usr/bin/env bash
# Sync the semantic-release-determined version across every DevRites manifest
# so plugin.json, marketplace.json and package.json stay in lockstep.
#
# Usage: sync-version.sh <version>
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: sync-version.sh <version>" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Syncing version ${VERSION} across DevRites manifests"

node - "$VERSION" <<'NODE'
const fs = require('fs');
const path = require('path');
const version = process.argv[2];

const updates = [
  { file: 'package.json',                       set: (j) => { j.version = version; } },
  { file: '.claude-plugin/plugin.json',         set: (j) => { j.version = version; } },
  { file: '.claude-plugin/marketplace.json',    set: (j) => { j.plugins.forEach(p => { if (p.name === 'devrites') p.version = version; }); } },
];

for (const u of updates) {
  const p = path.resolve(u.file);
  const data = JSON.parse(fs.readFileSync(p, 'utf8'));
  u.set(data);
  fs.writeFileSync(p, JSON.stringify(data, null, 2) + '\n');
  console.log(`  → ${u.file}`);
}
NODE
