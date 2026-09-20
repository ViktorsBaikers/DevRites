#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/good/nested" "$TMP/bad"
printf '{"name":"devrites","items":[1,true,null]}\n' > "$TMP/good/package.json"
printf '{"nested":{"ok":true}}\n' > "$TMP/good/nested/settings.json"
python3 "$ROOT/scripts/validate-pack-json.py" "$TMP/good" >/dev/null
python3 "$ROOT/scripts/validate-pack-json.py" \
  "$TMP/good/package.json" "$TMP/good/nested/settings.json" >/dev/null

printf '{"duplicate":1,"duplicate":2}\n' > "$TMP/bad/duplicate.json"
if python3 "$ROOT/scripts/validate-pack-json.py" "$TMP/bad" >/dev/null 2>&1; then
  echo "validate-pack-json-test: duplicate keys were accepted" >&2
  exit 1
fi

printf '{"broken":]\n' > "$TMP/bad/syntax.json"
if python3 "$ROOT/scripts/validate-pack-json.py" "$TMP/bad" >/dev/null 2>&1; then
  echo "validate-pack-json-test: malformed JSON was accepted" >&2
  exit 1
fi

python3 "$ROOT/scripts/validate-pack-json.py" \
  "$ROOT/pack/.claude" "$ROOT/pack/generated" >/dev/null
echo "validate-pack-json-test: PASS"
