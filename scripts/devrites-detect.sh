#!/usr/bin/env bash
# scripts/devrites-detect.sh — deterministic anti-slop detector.
#
# Greps the current git diff (or a passed file list) for ~25 actionable
# anti-slop patterns from rite-polish/reference/anti-ai-slop.md +
# devrites-frontend-craft/reference/quality-standards.md. No LLM needed.
#
# Exits 0 if no findings, 1 if any (so a pre-push hook can block).
# Use --advisory to always exit 0 (CI does this).
#
# Usage:
#   scripts/devrites-detect.sh                # diff vs upstream / HEAD~
#   scripts/devrites-detect.sh file1 file2    # check specific files
#   scripts/devrites-detect.sh --advisory     # report-only, never fail

set -euo pipefail

ADVISORY=0
FILES=()

for arg in "$@"; do
  case "$arg" in
    --advisory) ADVISORY=1 ;;
    *)          FILES+=("$arg") ;;
  esac
done

# Resolve which files to scan: explicit args, else diff against the base.
if [[ ${#FILES[@]} -eq 0 ]]; then
  BASE=""
  for candidate in origin/main origin/master main master HEAD~1; do
    if git rev-parse --verify "$candidate" >/dev/null 2>&1; then
      BASE="$candidate"
      break
    fi
  done
  if [[ -n "$BASE" ]]; then
    mapfile -t FILES < <(git diff --name-only --diff-filter=ACMR "$BASE" 2>/dev/null || true)
  fi
fi

# Filter into two scan lists: code/UI source (most rules) and markdown
# (em-dash overuse only). Meta / reference docs are excluded from the markdown
# scan because they discuss the patterns under audit.
SCAN=()
SCAN_MD=()
for f in "${FILES[@]}"; do
  [[ -z "$f" ]] && continue
  [[ ! -f "$f" ]] && continue
  case "$f" in
    *.lock|*.png|*.jpg|*.jpeg|*.gif|*.svg|*.ico|*.woff*|*.ttf|*.zip|*.gz) continue ;;
    *vendor/*|*node_modules/*|*dist/*|*build/*|*.min.*)                   continue ;;
  esac
  case "$f" in
    *.md)
      case "$f" in
        REVIEW.md|CHANGELOG.md|res.md)                                 continue ;;
        .review-notes/*|*/.review-notes/*)                             continue ;;
        pack/.claude/rules/*|*/pack/.claude/rules/*)                   continue ;;
        pack/.claude/skills/*|*/pack/.claude/skills/*)                 continue ;;
      esac
      SCAN_MD+=("$f")
      ;;
    *)
      SCAN+=("$f")
      ;;
  esac
done

if [[ ${#SCAN[@]} -eq 0 && ${#SCAN_MD[@]} -eq 0 ]]; then
  echo "devrites-detect: no files to scan."
  exit 0
fi

# Each rule: pattern (ERE), severity, message.
# Severity = critical (auth/safety), high (slop tell), low (style nit).
RULES=(
  # --- UI anti-slop (CSS-ish) ---
  '#000([^0-9a-f]|$)|#fff([^0-9a-f]|$)|#000000|#ffffff'                            'high'  'Pure #000 / #fff — use near-black / near-white tokens.'
  '#6366f1|#8b5cf6|#a855f7|#d946ef|#ec4899'                                        'high'  'Purple/blue gradient hex range — likely default AI palette.'
  'background-clip:[[:space:]]*text|-webkit-background-clip:[[:space:]]*text'      'high'  'Gradient text decoration — likely AI slop.'
  'backdrop-filter:[[:space:]]*blur'                                               'high'  'Glassmorphism `backdrop-filter: blur(...)` — banned default surface.'
  'border-left:[[:space:]]*[1-6]px[[:space:]]+solid|border-l-[2-8]'                'low'   'Side-stripe accent border — common templating tell; confirm vs design system.'
  'cubic-bezier\([^)]*0\.34[[:space:]]*,[[:space:]]*1\.56'                         'high'  'Bounce/elastic easing curve — banned unless design system uses it.'
  'animation-duration:[[:space:]]*[5-9][0-9]{2}ms[[:space:]]*;[[:space:]]*animation-delay'  'low'   'Long animation durations stacked — verify motion budget.'
  'z-index:[[:space:]]*[0-9]{4,}'                                                  'high'  'Raw z-index in the 1000s — use semantic z scale.'
  'font-family:[[:space:]]*"?(DM Sans|Plus Jakarta|Fraunces|Newsreader)'           'high'  'Reflex font choice — use the project type system.'
  'text-transform:[[:space:]]*uppercase[[:space:]]*;[^}]*letter-spacing'           'low'   'All-CAPS with letter-spacing — verify register; banned for body text.'
  # --- Code anti-slop ---
  '\bconsole\.log\('                                                               'high'  'console.log left in code — remove before polish.'
  'if[[:space:]]*\([[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*&&[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*\.length[[:space:]]*>[[:space:]]*0' 'high' 'Over-defensive `x && x.length > 0` — validate at boundaries.'
  '\bcatch[[:space:]]*\([[:space:]]*\)[[:space:]]*\{[^}]*\}'                       'high'  'Blanket catch with no narrow handling — likely swallowing errors.'
  '\bcatch[[:space:]]*\([[:space:]]*[A-Za-z][A-Za-z0-9_]*[[:space:]]*\)[[:space:]]*\{[[:space:]]*\}'  'high'  'Empty catch body — silently swallows errors. Catch narrow; recover or rethrow.'
  '//[[:space:]]*helper function|//[[:space:]]*increment'                          'low'   'Tutorial-style comment — restates code; remove.'
  '\bfunction[[:space:]]+(process|handle|do)(Data|Item|Thing|It)?\b'               'high'  'Generic AI naming (`processData` / `handleItem` / `doIt`) — name for intent.'
  '\bfunction[[:space:]]+get[A-Z][a-zA-Z]*\([^)]*\)[[:space:]]*\{[[:space:]]*return[[:space:]]+[A-Z][a-zA-Z]+\.find'  'low'   'Useless wrapper around a single ORM call — inline.'
  '//[[:space:]]*TODO:[[:space:]]*(improve|cleanup|refactor)[[:space:]]+(this|later)'  'low'   '"TODO: improve this later" — without owner/issue, this rots.'
  '\btemp\s*=|\bresult\s*=|\bdata\s*=[^=]'                                         'low'   'Generic local name (temp/result/data) — name for intent.'
  '\binterface[[:space:]]+I[A-Z][a-zA-Z]+\b'                                       'low'   '`IFoo` prefix on interfaces — most modern style guides prefer `Foo`.'
  '/\*\s*eslint-disable\s*\*/'                                                     'high'  'Global eslint-disable — explain or remove.'
  # --- Security ---
  '\bnew Function\('                                                               'critical' '`new Function(...)` — dynamic code construction.'
  '\beval\s*\('                                                                    'critical' '`eval(...)` — avoid.'
  '\bdangerouslySetInnerHTML\b'                                                    'high'  'dangerouslySetInnerHTML — confirm XSS-safe.'
  'process\.env\.[A-Z_]+(.{0,40}(?:console|res\.send|res\.json))'                  'high'  'env var possibly echoed to response/log — re-check for secret leakage.'
  # --- Performance pitfalls ---
  '\.forEach\([^)]+\)\s*\.\s*forEach\('                                            'low'   'Nested `forEach` — accidental quadratic loop?'
)

# Markdown-only rules. Run against SCAN_MD, not SCAN.
# Em-dash overuse: 2+ em-dashes (U+2014) on the same line — the per-line proxy
# for the "multiple em-dashes per paragraph" rule in anti-ai-slop.md.
MD_RULES=(
  $'.*\xe2\x80\x94.*\xe2\x80\x94'  'high'  'Em-dash overuse on one line (2+ em-dashes) — a common AI tell; prefer comma, period, or parenthetical.'
)

FOUND=0

# Code rules against code files.
if [[ ${#SCAN[@]} -gt 0 ]]; then
  N=${#RULES[@]}
  i=0
  while (( i < N )); do
    pat="${RULES[$i]}"
    sev="${RULES[$((i+1))]}"
    msg="${RULES[$((i+2))]}"
    i=$((i+3))

    HITS="$(grep -HnIE --color=never "$pat" "${SCAN[@]}" 2>/dev/null || true)"
    [[ -z "$HITS" ]] && continue

    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      FOUND=$((FOUND + 1))
      printf '[%s] %s — %s\n' "$sev" "$line" "$msg"
    done <<<"$HITS"
  done
fi

# Markdown rules against markdown files.
if [[ ${#SCAN_MD[@]} -gt 0 ]]; then
  N=${#MD_RULES[@]}
  i=0
  while (( i < N )); do
    pat="${MD_RULES[$i]}"
    sev="${MD_RULES[$((i+1))]}"
    msg="${MD_RULES[$((i+2))]}"
    i=$((i+3))

    HITS="$(grep -HnIE --color=never "$pat" "${SCAN_MD[@]}" 2>/dev/null || true)"
    [[ -z "$HITS" ]] && continue

    while IFS= read -r line; do
      [[ -z "$line" ]] && continue
      FOUND=$((FOUND + 1))
      printf '[%s] %s — %s\n' "$sev" "$line" "$msg"
    done <<<"$HITS"
  done
fi

TOTAL=$(( ${#SCAN[@]} + ${#SCAN_MD[@]} ))

echo
if [[ $FOUND -eq 0 ]]; then
  echo "devrites-detect: clean — scanned $TOTAL files (${#SCAN[@]} code, ${#SCAN_MD[@]} md)."
  exit 0
fi

echo "devrites-detect: $FOUND finding(s) across $TOTAL files (${#SCAN[@]} code, ${#SCAN_MD[@]} md)."

if [[ $ADVISORY -eq 1 ]]; then
  exit 0
fi

exit 1
