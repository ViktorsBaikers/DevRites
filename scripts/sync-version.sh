#!/usr/bin/env bash
# Sync the semantic-release-determined version into package.json and the README
# status line so the published version stays in step with the released tag.
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
];

for (const u of updates) {
  const p = path.resolve(u.file);
  const data = JSON.parse(fs.readFileSync(p, 'utf8'));
  u.set(data);
  fs.writeFileSync(p, JSON.stringify(data, null, 2) + '\n');
  console.log(`  → ${u.file}`);
}

// Update README status line so the published version shows on the repo
// landing page. Matches the previous status block (single- or two-line form)
// and rewrites it to a one-liner pointing at the new tag.
const readmePath = path.resolve('README.md');
const readme = fs.readFileSync(readmePath, 'utf8');
const statusLine =
  `**Status:** [\`v${version}\`](https://github.com/ViktorsBaikers/DevRites/releases/tag/v${version}) — ` +
  "see [`CHANGELOG.md`](CHANGELOG.md) for release notes.";
const re = /\*\*Status:\*\*[^\n]*(?:\n(?!\n)[^\n]*)*/;
if (re.test(readme)) {
  const updated = readme.replace(re, statusLine);
  if (updated !== readme) {
    fs.writeFileSync(readmePath, updated);
    console.log('  → README.md');
  }
} else {
  console.warn('  ! README.md status line not found — skipping');
}
NODE
