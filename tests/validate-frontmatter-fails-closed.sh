#!/usr/bin/env bash
# validate-frontmatter-fails-closed.sh: assert scripts/validate-frontmatter.py
# - refuses with a "PyYAML required" message and a non-zero exit when PyYAML cannot be
#   imported, instead of guessing at the frontmatter; and
# - with PyYAML, exits 1 and names the file when the frontmatter is not valid YAML even
#   though name and description are otherwise valid, and still accepts valid documents.
# Neither case is skipped: a missing PyYAML or python3 fails the run. Runs on temp
# fixtures only; the pack is never read or written.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PY="${VALIDATE_FRONTMATTER_PY:-$ROOT/scripts/validate-frontmatter.py}"
MSG="PyYAML required: pip install -r scripts/requirements-ci.txt"

if ! command -v python3 >/dev/null 2>&1; then
  echo "FAIL: python3 not found"
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

mkdir "$TMP/noyaml"
printf 'raise ImportError("PyYAML hidden for the refusal run")\n' > "$TMP/noyaml/yaml.py"

printf -- '---\nname: probe-unparsable\ndescription: [unclosed flow sequence here\nuser-invocable: true\n---\nbody\n' > "$TMP/unparsable-flow.md"
printf -- '---\nname: probe-mapping\ndescription: {unclosed: flow mapping\nuser-invocable: true\n---\nbody\n' > "$TMP/unparsable-mapping.md"
printf -- '---\nname: probe-quote\ndescription: "unclosed quote here\nuser-invocable: true\n---\nbody\n' > "$TMP/unparsable-quote.md"
printf -- '---\nname: probe-tab\ndescription: A well-formed description for the tab probe.\n\tbad: tab-indented\n---\nbody\n' > "$TMP/unparsable-tab.md"
# mkfm NAME DESCRIPTION-LINE: valid name and user-invocable around one description line.
mkfm() { printf -- '---\nname: probe-%s\n%s\nuser-invocable: true\n---\nbody\n' "$1" "$2" > "$TMP/$1.md"; }
mkfm nospace 'description:foo'
mkfm tabvalue "$(printf 'description: a\tb')"
mkfm dashvalue 'description: - item-like value'
mkfm questionvalue 'description: ? key-like value'
mkfm badescape 'description: "bad \q escape"'
mkfm starvalue 'description: *alias'
mkfm bangvalue 'description: !tag value'
mkfm pipevalue 'description: |not-a-header'
mkfm gtvalue 'description: >not-a-header'
mkfm percentvalue 'description: %directive'
mkfm atvalue 'description: @reserved'
mkfm backtickvalue 'description: `reserved'
printf -- '---\nname: probe-validlist\ndescription: A well-formed description for the list control.\nargument-hint:\n    - Read\n    - Grep\n---\nbody\n' > "$TMP/valid-list.md"
printf -- '---\nname: probe-validblock\ndescription: A well-formed description for the block control.\nargument-hint: |\n    first\n      deeper\n\n    last\n---\nbody\n' > "$TMP/valid-block.md"
printf -- '---\nname: probe-valid\ndescription: A canonical skill used as the acceptance control.\nuser-invocable: true\n---\nbody\n' > "$TMP/valid-control.md"
printf -- '---\nname: probe-quoted\ndescription: "A quoted description with a colon: inside."\nargument-hint: "[arg]"\nuser-invocable: true\n---\nbody\n' > "$TMP/valid-quoted.md"

BAD="unparsable-flow unparsable-mapping unparsable-quote unparsable-tab nospace tabvalue dashvalue questionvalue badescape starvalue bangvalue pipevalue gtvalue percentvalue atvalue backtickvalue"
GOOD="valid-control valid-quoted valid-list valid-block"

# Without PyYAML the validator must refuse every file, valid or not.
assert_refuses_without_pyyaml() {
  out="$(PYTHONPATH="$TMP/noyaml" python3 "$PY" "$TMP/$1.md" 2>&1)"; status=$?
  if [ "$status" -eq 0 ]; then
    echo "FAIL: [no-pyyaml] $1: expected non-zero exit, got 0"
    fail=1
  elif ! printf '%s\n' "$out" | grep -qF "$MSG"; then
    echo "FAIL: [no-pyyaml] $1: exit $status but output lacks: $MSG"
    printf '%s\n' "$out" | sed 's/^/      | /'
    fail=1
  else
    echo "ok: [no-pyyaml] $1: refused (exit $status, message present)"
  fi
}

# Guards the refusal run: it is only meaningful if `import yaml` really fails there.
if PYTHONPATH="$TMP/noyaml" python3 -c 'import yaml' 2>/dev/null; then
  echo "FAIL: [no-pyyaml] import yaml still succeeds, refusal not exercised"
  fail=1
fi
for name in unparsable-flow valid-control; do assert_refuses_without_pyyaml "$name"; done

if ! python3 -c 'import yaml' 2>/dev/null; then
  echo "FAIL: PyYAML not installed; run: pip install -r scripts/requirements-ci.txt"
  fail=1
else
  # Guards the fixtures: a document PyYAML accepts would not exercise the YAML-error path.
  assert_pyyaml_rejects() {
    if python3 - "$TMP/$1.md" <<'PY'
import sys, yaml
lines = open(sys.argv[1], encoding="utf-8").read().splitlines()
body = []
for line in lines[1:]:
    if line.strip() == "---":
        break
    body.append(line)
try:
    yaml.safe_load("\n".join(body))
except yaml.YAMLError:
    sys.exit(0)
sys.exit(1)
PY
    then
      echo "ok: $1: PyYAML rejects the fixture"
    else
      echo "FAIL: $1: PyYAML accepts the fixture, so it does not exercise the YAML-error path"
      fail=1
    fi
  }

  assert_rejected_named() {
    out="$(python3 "$PY" "$TMP/$1.md" 2>&1)"; status=$?
    if [ "$status" -ne 1 ]; then
      echo "FAIL: [pyyaml] $1: expected exit 1, got $status"
      printf '%s\n' "$out" | sed 's/^/      | /'
      fail=1
    elif ! printf '%s\n' "$out" | grep -qF "ERROR $TMP/$1.md"; then
      echo "FAIL: [pyyaml] $1: exit 1 but no ERROR line naming the file"
      printf '%s\n' "$out" | sed 's/^/      | /'
      fail=1
    else
      echo "ok: [pyyaml] $1: rejected (exit 1, ERROR names the file)"
    fi
  }

  assert_accepted() {
    if out="$(python3 "$PY" "$TMP/$1.md" 2>&1)"; then
      echo "ok: [pyyaml] $1: accepted (exit 0)"
    else
      echo "FAIL: [pyyaml] $1: valid frontmatter rejected"
      printf '%s\n' "$out" | sed 's/^/      | /'
      fail=1
    fi
  }

  for name in $BAD; do assert_pyyaml_rejects "$name"; assert_rejected_named "$name"; done
  for name in $GOOD; do assert_accepted "$name"; done
fi

if [ "$fail" -eq 0 ]; then
  echo "VALIDATE-FRONTMATTER FAILS-CLOSED TESTS: PASS"
else
  echo "VALIDATE-FRONTMATTER FAILS-CLOSED TESTS: FAIL"
fi
exit "$fail"
