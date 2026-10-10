#!/usr/bin/env bash
# check-no-global-writes.sh: static guard that the installer/uninstaller never
# writes to the user's global ~/.claude or ~/.codex, and that the global guard is present.
# Exits non-zero on violation.

set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0

# Array, not a space-joined string: a checkout path with spaces must not split.
SCRIPTS=("$ROOT/install.sh" "$ROOT/uninstall.sh" "$ROOT/update.sh" "$ROOT"/scripts/*.sh)

# 1) the no-global guard marker must exist in install.sh
if ! grep -q 'GUARD:no-global' "$ROOT/install.sh"; then
  echo "FAIL: install.sh is missing the GUARD:no-global guard"
  fail=1
else
  echo "ok: install.sh contains the no-global guard"
fi

# write_hits: reads grep -n records on stdin and prints those that write. PATH_RE
# (env) names the protected path. A record is a write when it matches a plain write
# verb or redirection anywhere, or when one command of it (split at unquoted | ; &)
# names PATH_RE and is touch, dd, truncate or install (install always writes its
# destination), or sed with an in-place flag of its own: -i/-I in any short-flag
# bundle, or a --in-place prefix of --i or longer. The verb is found after leading
# ( { $( backtick, a case-arm label, a shell keyword, or a sudo/env/nice/xargs/time/
# command/exec wrapper with its options, or after a `case WORD in` head. The body of
# every $( ) and backtick span, also inside double quotes, is checked as its own command.
# Limit: matching is per physical line, so a path built in a separate assignment is not seen.
write_hits() {
  PATH_RE="$1" awk '
    function isflag(t,  n, i, c) {
      n = t; sub(/=.*/, "", n)
      if (n ~ /^--/) return length(n) >= 3 && index("--in-place", n) == 1
      if (n !~ /^-/) return 0
      for (i = 2; i <= length(n); i++) {
        c = substr(n, i, 1)
        if (c == "i" || c == "I") return 1
        if (c == "e" || c == "f" || c == "l") return 0
      }
      return 0
    }
    function flush() { if (intok) tok[++nt] = cur; cur = ""; intok = 0 }
    function argopt(w) {
      if (w == "sudo") return "ughpCDRrtTU"
      if (w == "env") return "uCS"
      if (w == "nice") return "n"
      if (w == "xargs") return "InPLsdEaJ"
      return ""
    }
    function check(  k, t, v, w, i) {
      flush()
      if (nt == 0 || raw !~ ENVIRON["PATH_RE"]) return 0
      k = 1
      while (k <= nt) {
        t = tok[k]
        while (sub(/^([({!`]|\$\()/, "", t)) ;
        if (t == "") { k++; continue }
        if (t == "case") { while (k <= nt && tok[k] != "in") k++; k++; continue }
        if (t ~ /^[A-Za-z_][A-Za-z0-9_]*=/ || t ~ /\)$/ || t ~ /^(if|then|elif|else|do|while|until)$/) { k++; continue }
        if (t ~ /^(sudo|env|command|nohup|time|exec|builtin|nice|xargs)$/) {
          w = t; k++
          while (k <= nt && tok[k] ~ /^-/) {
            if (length(tok[k]) == 2 && index(argopt(w), substr(tok[k], 2, 1))) k++
            k++
          }
          continue
        }
        break
      }
      v = t; sub(/.*\//, "", v)
      if (v == "install" || v == "touch" || v == "dd" || v == "truncate") return 1
      if (v == "sed") for (i = k + 1; i <= nt; i++) if (isflag(tok[i])) return 1
      return 0
    }
    function reset() { raw = ""; nt = 0; cur = ""; intok = 0 }
    function scan(line,  n, i, c, d, q) {
      q = ""; reset(); n = length(line)
      for (i = 1; i <= n; i++) {
        c = substr(line, i, 1)
        if (c == "\\" && q != "\047") { i++; d = substr(line, i, 1); cur = cur d; intok = 1; raw = raw c d; continue }
        if (q != "") { if (c == q) q = ""; else cur = cur c; raw = raw c; continue }
        if (c == "\"" || c == "\047") { q = c; intok = 1; raw = raw c; continue }
        if (c == "|" || c == ";" || c == "&") { if (check()) return 1; reset(); continue }
        if (c == " " || c == "\t") { flush(); raw = raw c; continue }
        cur = cur c; intok = 1; raw = raw c
      }
      return check()
    }
    function spans(s,  n, i, j, c, depth, st, body) {
      n = length(s)
      for (i = 1; i <= n; i++) {
        c = substr(s, i, 1)
        if (c == "\\") { i++; continue }
        if (c == "`") {
          st = i + 1
          for (j = st; j <= n && substr(s, j, 1) != "`"; j++) ;
        } else if (c == "$" && substr(s, i + 1, 1) == "(") {
          st = i + 2; depth = 1
          for (j = st; j <= n; j++) {
            c = substr(s, j, 1)
            if (c == "(") depth++
            else if (c == ")" && --depth == 0) break
          }
        } else continue
        body = substr(s, st, j - st)
        if (scan(body) || spans(body)) return 1
        i = j
      }
      return 0
    }
    BEGIN { simple = "(^|[^A-Za-z0-9_])(mkdir|cp|tee|rsync|mv|ln)([^A-Za-z0-9_]|$)|>>?" }
    {
      line = $0; sub(/^[0-9]+:/, "", line)
      if ($0 ~ simple || scan(line) || spans(line)) print
    }'
}

# 2) no write operation may target ~/.claude, ~/.codex, or their $HOME forms.
GLOBAL_RE='(\$HOME/\.(claude|codex)|~/\.(claude|codex))'
for f in "${SCRIPTS[@]}"; do
  [ -f "$f" ] || continue
  # lines that contain a global-claude path AND a write verb, excluding comments
  hits="$(grep -nE "$GLOBAL_RE" "$f" | grep -vE '^\s*[0-9]+:\s*#' | write_hits "$GLOBAL_RE" || true)"
  if [ -n "$hits" ]; then
    echo "FAIL: possible global write in ${f#$ROOT/}:"
    printf '%s\n' "$hits" | sed 's/^/    /'
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "ok: no write operations target global agent homes"

# 3) no write operation may target a system bin directory EXCEPT the one sanctioned
#    engine-binary path (the devrites-engine control-plane binary; issue 10). Anything else
#    writing to /usr/bin, /usr/local/bin, /bin, /sbin, or ~/.local/bin is a global
#    write the installer must not do. The carve-out is exactly a `devrites-engine` (or
#    `devrites.exe`) leaf under /usr/local/bin or ~/.local/bin: never /usr/bin.
BIN_RE='((\$HOME|~)/\.local/bin|/usr/local/bin|/usr/bin|/bin|/sbin)'
# Sanctioned: the write's bin path ends in .../devrites-engine (optionally .exe) and sits
# under /usr/local/bin or ~/.local/bin. The trailing boundary keeps `devrites-lib`
# or `devritesX` from sneaking through.
SANCTIONED_RE='((\$HOME|~)/\.local/bin|/usr/local/bin)/devrites-engine(\.exe)?([^A-Za-z0-9._-]|$)'
for f in "${SCRIPTS[@]}"; do
  [ -f "$f" ] || continue
  hits="$(grep -nE "$BIN_RE" "$f" | grep -vE '^\s*[0-9]+:\s*#' | write_hits "$BIN_RE" \
    | grep -vE "$SANCTIONED_RE" || true)"
  if [ -n "$hits" ]; then
    echo "FAIL: possible global bin write in ${f#$ROOT/} (only /usr/local/bin/devrites-engine or ~/.local/bin/devrites-engine is sanctioned):"
    printf '%s\n' "$hits" | sed 's/^/    /'
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "ok: no write operations target system bin dirs (except the sanctioned devrites-engine binary)"

if [ "$fail" -eq 0 ]; then
  echo "PASS: no global-write risks detected"
else
  echo "FAILED: global-write check"
fi
exit "$fail"
