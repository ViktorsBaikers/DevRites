#!/usr/bin/env bash
# Fixture test: check-no-global-writes.sh must reject single-line global writes
# made with touch, install, dd, truncate and sed -i, and pass read-only uses and the real tree.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
CHECK="$ROOT/scripts/check-no-global-writes.sh"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

bash "$CHECK" >/dev/null || { echo "FAIL: real tree should pass"; exit 1; }

for line in \
  'touch "$HOME/.claude/x"' \
  'install -d "$HOME/.codex/x"' \
  'dd of="$HOME/.claude/x" if=/dev/null' \
  'truncate -s 0 ~/.claude/x' \
  'sed -i s/a/b/ ~/.codex/x' \
  'install -m 755 -d "$HOME/.claude/skills"' \
  'install -dm755 "$HOME/.claude/skills"' \
  'sed -E -i s/a/b/ ~/.codex/x' \
  'sed -e s/a/b/ -i ~/.codex/x' \
  'sed --in-place s/a/b/ ~/.codex/x' \
  'install --directory "$HOME/.claude/skills"' \
  "sed 's/a/b/;s/c/d/' -i ~/.claude/x" \
  "sed -e 's/a/&b/' -i ~/.claude/x" \
  'install --dir ~/.claude/a' \
  'install --di "$HOME/.claude/a"' \
  'install --directory=$HOME/.claude/a' \
  'sed --in s/a/b/ ~/.claude/x' \
  'sed --in-pl s/a/b/ ~/.claude/x' \
  'sed --i s/a/b/ ~/.claude/x' \
  'sed --in-place=.bak s/a/b/ ~/.claude/x' \
  "sed -I '' s/a/b/ ~/.claude/x" \
  'sed -I.bak s/a/b/ ~/.claude/x' \
  "sed -EI '' s/a/b/ ~/.claude/x" \
  'install -m 644 src ~/.claude/x' \
  'install -D src "$HOME/.claude/x"' \
  'install -t ~/.claude/skills a b' \
  'install -Dm644 src ~/.codex/x' \
  'FOO=1 touch ~/.claude/x' \
  'sudo install src ~/.claude/x' \
  'true && touch ~/.claude/x' \
  'ls | sed -i s/a/b/ ~/.claude/x' \
  "echo 'a;b' | sed -i s/a/b/ ~/.claude/x" \
  '(touch "$HOME/.claude/x")' \
  "(sed -i 's/a/b/' ~/.claude/x)" \
  '(install -d "$HOME/.codex")' \
  '(dd of=~/.claude/x if=/dev/null)' \
  '(truncate -s 0 ~/.claude/x)' \
  '{ touch ~/.claude/x; }' \
  'claude) touch "$HOME/.claude/x" ;;' \
  "*) sed -i '' 's/a/b/' ~/.claude/x ;;" \
  'a|claude) touch ~/.claude/x ;;' \
  'then touch ~/.claude/x' \
  '! touch ~/.claude/x' \
  'if touch ~/.claude/x; then' \
  'while touch ~/.claude/x; do' \
  'time touch ~/.claude/x' \
  'nice -n 5 touch ~/.claude/x' \
  'xargs touch ~/.claude/x' \
  'xargs -I {} touch ~/.claude/x' \
  'command touch ~/.claude/x' \
  'exec touch ~/.claude/x' \
  '$(touch ~/.claude/x)' \
  'out=$(touch ~/.claude/x)' \
  'x=`touch ~/.claude/x`' \
  'echo "$(touch ~/.claude/x)"' \
  'v="$(sed -i '"''"' s/a/b/ "$HOME/.codex/x")"' \
  'out="$(install -d "$HOME/.claude/x")"' \
  'x="`touch ~/.claude/x`"' \
  'echo "a $(echo "$(touch ~/.claude/x)")"' \
  'case "$h" in claude) touch "$HOME/.claude/x" ;; esac' \
  'case "$h" in codex) sed -i s/a/b/ ~/.codex/x ;; esac' \
  'sudo -u "$SUDO_USER" touch "$HOME/.claude/x"' \
  'env -u FOO touch ~/.claude/x' \
  'env -i FOO=1 touch ~/.claude/x'; do
  rm -rf "$T/root"
  mkdir -p "$T/root/scripts"
  cp "$CHECK" "$T/root/scripts/"
  printf '#!/usr/bin/env bash\n# GUARD:no-global\n%s\n' "$line" >"$T/root/install.sh"
  if bash "$T/root/scripts/check-no-global-writes.sh" >"$T/out.txt" 2>&1; then
    echo "FAIL: should reject: $line"
    cat "$T/out.txt"
    exit 1
  fi
  grep -q 'possible global write' "$T/out.txt" || { echo "FAIL: wrong failure for: $line"; cat "$T/out.txt"; exit 1; }
done

for line in \
  'sed -n 1p ~/.claude/x' \
  'sed -n p ~/.claude/x | grep -i foo' \
  'cat ~/.claude/x | sed s/a/b/' \
  'echo "install into ~/.claude"' \
  "echo 'dd is a tool'; ls ~/.claude" \
  "sed -e 's/-i/b/' ~/.claude/x" \
  'claude) ls ~/.claude ;;' \
  "claude) sed -n p ~/.claude/x ;;" \
  '(cat ~/.claude/x)' \
  'x=$(cat ~/.claude/x)' \
  'echo "$(cat ~/.claude/x)"' \
  'case "$h" in claude) ls ~/.claude ;; esac' \
  'sudo -u user ls ~/.claude' \
  'env -u FOO ls ~/.claude'; do
  rm -rf "$T/root"
  mkdir -p "$T/root/scripts"
  cp "$CHECK" "$T/root/scripts/"
  printf '#!/usr/bin/env bash\n# GUARD:no-global\n%s\n' "$line" >"$T/root/install.sh"
  bash "$T/root/scripts/check-no-global-writes.sh" >"$T/out.txt" 2>&1 || { echo "FAIL: should pass: $line"; cat "$T/out.txt"; exit 1; }
done

echo "PASS: check-no-global-writes rejects single-line global writes"
