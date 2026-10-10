#!/usr/bin/env bash
# The Temper skill must not let an unattended (AFK) run grow scope: `expand` and extra
# acceptance escalate through AskUserQuestion and the fold holds until a human answers.
# Checks the shipped rite-temper skill tree and the autocomplete decision policy, then proves
# each check bites on a mutated copy.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== rite-temper-afk-expansion-gate-test (target: $T) =="

REF=pack/.claude/skills/rite-temper/reference
POLICY=pack/.claude/skills/rite-autocomplete/reference/decision-policy.md
LOOP=pack/.claude/skills/rite-autocomplete/reference/loop.md

GROW='expan(d|sion)|(extra|additional|added|new|more) acceptance'
PROHIBITION='Recommended temper `expand` and extra acceptance are never auto-picked: they stop for a human through `AskUserQuestion`; do not invent product intent.'

# sentences <file> → one sentence per line: newlines folded, markdown emphasis and unicode
# hyphens normalised, split after . ; | (a dot only counts before whitespace or at the end).
sentences() {
  tr '\n' ' ' <"$1" | perl -pe 's/\xe2\x80[\x90-\x93]/-/g; s/\*//g; s/\b(e\.g|i\.e)\./$1/g; s/([;|]|\.(?=\s|$))/$1\n/g; s/^\s+//mg'
  echo
}

# grants <root> → sentences under pack/.claude that grant auto-applying expand/extra acceptance.
# A negation counts only when it directly governs the auto verb ("never auto-picked",
# "not auto-apply", "never be automatically applied"); that span is removed and the rest of the
# sentence is still tested. The one red-flag bullet in an anti-patterns.md that names the
# forbidden outcome is the only other sentence skipped.
grants() {
  find "$1/pack/.claude" -type f -print0 | while IFS= read -r -d '' f; do
    grep -qI . "$f" || continue
    sentences "$f" | F="${f#"$1"/}" GROW="$GROW" perl -ne '
      BEGIN { $neg = qr/(?:never|not|cannot|n\x27t)\s+(?:(?:be|ever)\s+)*(?:auto[- ]?\w+|automatically)/i;
              $verb = qr/auto[- ]?(?:appl|pick|grow|select|add|accept)|automatically/i; $grow = qr/$ENV{GROW}/i; }
      next if $ENV{F} =~ m{/anti-patterns\.md$} && /An `expand` decision auto-applied in AFK, an unapproved/;
      ($t = $_) =~ s/$neg/ /g;
      print "$ENV{F}: $_" if $t =~ $verb && $t =~ $grow;'
  done
}

# strip_hidden <file> → the file without what a reader never sees as prose: fenced code blocks
# (an unclosed fence runs to the end), closed HTML comments and ~~struck~~ spans.
strip_hidden() {
  perl -0pe 's/^ {0,3}(?:(```+)[^`\n]*|(~~~+)[^\n]*)\n.*?(?:^ {0,3}(?:\1|\2)[ \t]*$|\z)//msg;
            s/<!--.*?-->//sg; s/~~(?:(?!\n[ \t]*\n).)+?~~//sg' "$1"
}

# policy_rules <policy> → one line per violated rule in what autocomplete may pick. Only visible
# prose counts: the one verbatim prohibition sentence is the sole place `expand` or extra
# acceptance may appear, and it must itself be visible.
policy_rules() {
  local p="$1" defaults rule vis
  vis="$T/policy.visible"
  strip_hidden "$p" >"$vis"
  defaults="$(awk '/^## /{on = /^## Defaults autocomplete may assume/} on' "$vis")"
  [ -n "$defaults" ] || echo "decision-policy.md: no 'Defaults autocomplete may assume' section"
  printf '%s\n' "$defaults" | grep -qiE "$GROW" && echo "decision-policy.md: defaults section admits expand/extra acceptance"
  rule="$(sentences "$vis" | grep -E '\(Recommended\)|proposed:')"
  [ -n "$rule" ] || echo "decision-policy.md: no (Recommended)/proposed: rule"
  printf '%s\n' "$rule" | grep -qiE "$GROW" && echo "decision-policy.md: (Recommended)/proposed: rule admits expand/extra acceptance"
  P="$PROHIBITION" GROW="$GROW" perl -0e '
    $_ = <STDIN>; tr/\n/ /; s/ +/ /g; $p = $ENV{P};
    if (!s/(?:^|(?<=[.!?] ))\Q$p\E(?= |$)//) { print "decision-policy.md: prohibition on auto-picking expand/extra acceptance missing\n"; exit }
    s/[*`]//g;
    print "decision-policy.md: expand/extra acceptance mentioned outside the prohibition\n" if /$ENV{GROW}/i' <"$vis"
}

# check <root> → prints one line per violated rule; exit 1 when any.
check() {
  local r="$1" bad=0 f grant pol
  if grant="$(grants "$r")" && [ -n "$grant" ]; then
    printf '%s\n' "$grant"; echo "grant: auto-apply paired with expand/extra acceptance"; bad=1
  fi
  pol="$(policy_rules "$r/$POLICY")"
  [ -z "$pol" ] || { printf '%s\n' "$pol"; bad=1; }
  f="$r/$REF/significance.md"
  grep -q 'AskUserQuestion' "$f" || { echo "significance.md: no AskUserQuestion escalation"; bad=1; }
  grep -q 'holds the fold' "$f" || { echo "significance.md: fold not held"; bad=1; }
  grep -qE '`expand` and any extra acceptance \*\*do not\*\* apply unattended' "$f" \
    || { echo "significance.md: expand/acceptance not excluded from unattended"; bad=1; }
  grep -qF '`expand` and extra acceptance stop for a human' "$r/$LOOP" \
    || { echo "loop.md: temper row does not stop expand/extra acceptance for a human"; bad=1; }
  grep -q 'AskUserQuestion' "$r/$REF/scope-modes.md" \
    || { echo "scope-modes.md: EXPAND row has no AFK escalation"; bad=1; }
  grep -q 'resolved human `questions.md` qid' "$r/$REF/strategy-template.md" \
    || { echo "strategy-template.md: growth not tied to a human qid"; bad=1; }
  return $bad
}

out="$(check "$ROOT")" && ok "shipped rite-temper tree holds the AFK expansion gate" || no "gate violated: $out"

# Mutations: each must make check() fail.
mutate() { # name file sed-expr
  local d="$T/$1"
  mkdir -p "$d/pack/.claude/skills" && cp -R "$ROOT/pack/.claude/skills/rite-temper" "$ROOT/pack/.claude/skills/rite-autocomplete" "$d/pack/.claude/skills/"
  sed -i.bak "$3" "$d/$2" && rm -f "$d/$2.bak"
  check "$d" >/dev/null && no "mutation $1 not caught" || ok "mutation $1 caught"
}
mutate regrant "$REF/significance.md" 's/`expand` and any extra acceptance \*\*do not\*\* apply unattended/auto-apply the recommended mode, including `expand`/'
mutate noask "$REF/significance.md" 's/AskUserQuestion/prompt/g'
mutate nohold "$REF/significance.md" 's/holds the fold/folds/'
mutate scope-noask "$REF/scope-modes.md" 's/AskUserQuestion/prompt/g'
mutate adr-ok "$REF/strategy-template.md" 's/resolved human `questions.md` qid/`decisions.md` ADR/g'
mutate policy-regrant "$POLICY" 's/`expand` and extra acceptance are never auto-picked/`expand` auto-applies/'
mutate policy-weakened "$POLICY" 's/are never auto-picked/are auto-picked/'
mutate policy-prohibition-deleted "$POLICY" '/are never auto-picked/d'
mutate policy-defaults-expand "$POLICY" 's/^- Prefer the smallest vertical slice that proves the acceptance criterion\./- Prefer the smallest vertical slice, and take the recommended `expand`./'
mutate policy-defaults-acceptance "$POLICY" 's/^- Prefer the smallest vertical slice that proves the acceptance criterion\./- Add extra acceptance when the spec looks thin./'
mutate loop-expand-unattended "$LOOP" 's/harden or reduce; `expand` and extra acceptance stop for a human;/harden, reduce, or expand;/'
mutate policy-rule-expand "$POLICY" "s/is autocomplete's to resolve: pick/is autocomplete's to resolve, even \`expand\`: pick/"

# Appending text to a copied file must also fail the check.
append() { # name file text
  local d="$T/$1"
  mkdir -p "$d/pack/.claude/skills" && cp -R "$ROOT/pack/.claude/skills/rite-temper" "$ROOT/pack/.claude/skills/rite-autocomplete" "$d/pack/.claude/skills/"
  printf '\n%s\n' "$3" >>"$d/$2"
  check "$d" >/dev/null && no "mutation $1 not caught" || ok "mutation $1 caught"
}
append policy-baseline-grant "$POLICY" 'Recommended temper\s+`expand`
auto-applies; do not invent product intent.'
append skill-grant pack/.claude/skills/rite-autocomplete/SKILL.md 'Recommended temper\s+`expand` auto-applies.'
append inflection-grant "$REF/scope-modes.md" 'Extra acceptance is auto-applying under AFK.'

# Hidden-text and stray-mention escapes in the policy: each must fail.
rewrite() { # name file perl-expr (P holds the prohibition)
  local d="$T/$1"
  mkdir -p "$d/pack/.claude/skills" && cp -R "$ROOT/pack/.claude/skills/rite-temper" "$ROOT/pack/.claude/skills/rite-autocomplete" "$d/pack/.claude/skills/"
  P="$PROHIBITION" perl -0pi -e "$3" "$d/$2"
  check "$d" >/dev/null && no "mutation $1 not caught" || ok "mutation $1 caught"
}
DROP='s/Recommended temper\s+`expand`.*?intent\.//s;'
rewrite policy-table-row "$POLICY" 's/(\| Two reasonable designs[^\n]*\n)/$1| Temper `expand` | take the recommended mode | note in `decisions.md` |\n/'
rewrite policy-stray-sentence "$POLICY" 's/\z/\nTemper `expand` with a recommended option is autocomplete\x27s to resolve.\n/'
rewrite policy-pick-section "$POLICY" 's/\z/\n## Temper picks\n\nPick the recommended `expand` mode.\n/'
rewrite policy-extra-acceptance "$POLICY" 's/\z/\nAdd additional acceptance when the spec looks thin.\n/'
rewrite policy-duplicate-prohibition "$POLICY" 's/\z/\n$ENV{P}\n/'
rewrite policy-prohibition-fenced "$POLICY" "$DROP"'s/\z/\n```\n$ENV{P}\n```\n/'
rewrite policy-prohibition-tilde-fenced "$POLICY" "$DROP"'s/\z/\n~~~\n$ENV{P}\n~~~\n/'
rewrite policy-prohibition-unclosed-fence "$POLICY" "$DROP"'s/\z/\n```\n$ENV{P}\n/'
rewrite policy-prohibition-struck "$POLICY" 's/(Recommended temper\s+`expand`.*?intent\.)/~~$1~~/s'
rewrite policy-prohibition-commented "$POLICY" 's/(Recommended temper\s+`expand`.*?intent\.)/<!-- $1 -->/s'
rewrite policy-prohibition-prefixed "$POLICY" 's/continue\. Recommended temper/continue. Not so: recommended temper/'
rewrite policy-prohibition-fenced-kept-struck "$POLICY" 's/(Recommended temper\s+`expand`.*?intent\.)/~~$1~~/s;s/\z/\n```\n$ENV{P}\n```\n/'

# Grant phrasings where a negation or exception elsewhere in the sentence must not excuse it.
S=pack/.claude/skills/rite-autocomplete/SKILL.md
append negation-elsewhere "$S" 'Temper auto-applies `expand` without waiting for a human.'
append unless-clause "$S" 'Unless irreversible risk fires, Temper auto-applies `expand` and extra acceptance.'
append never-pause "$S" 'Auto-apply the recommended mode, including `expand`, and never pause for the human.'
append not-stop "$S" 'Temper does not stop: it auto-applies `expand`.'
append never-then-grant "$S" 'Never auto-apply a risky mode, but auto-apply `expand`.'
append automatically-trailing "$S" 'Apply the recommended `expand` mode automatically.'
append auto-selects "$S" 'Temper auto-selects `expand` and adds acceptance criteria.'
append accept-automatically "$S" 'Accept `expand` and extra acceptance automatically.'
append md-filename-dot "$S" 'Auto-apply the recommended mode (see scope-modes.md), including `expand`.'
append eg-dot "$S" 'Auto-apply the recommended mode, e.g. `expand`.'
append expansion-word "$S" 'Temper auto-applies more acceptance criteria and expansion.'
append upper-case "$S" 'Auto-apply EXPAND.'
append spaced-verb "$S" 'auto apply expand'
append unicode-hyphen "$S" "$(printf 'Auto\342\200\221apply `expand`.')"
append anti-patterns-grant pack/.claude/skills/rite-temper/reference/anti-patterns.md 'Recommended temper\s+`expand` auto-applies.'
append yaml-grant pack/.claude/skills/rite-autocomplete/reference/extra.yaml 'temper_expand: auto-apply'
append txt-grant pack/.claude/skills/rite-autocomplete/reference/extra.txt 'Recommended temper\s+`expand` auto-applies.'

exit $fail
