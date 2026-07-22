package lib

import (
	"crypto/sha1" // #nosec G505 -- non-cryptographic tool-call fingerprint; must match `sha1sum` output
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// awkNumRe recognises an awk "numeric string" (optional surrounding blanks, sign,
// int/float/exponent): the class that makes a comparison numeric. awkLeadNumRe
// captures the leading numeric prefix strtod would convert (so "4x" → 4).
var (
	awkNumRe     = regexp.MustCompile(`^[ \t\n]*[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)([eE][+-]?[0-9]+)?[ \t\n]*$`)
	awkLeadNumRe = regexp.MustCompile(`^[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)([eE][+-]?[0-9]+)?`)
)

// awkNum returns a token's numeric value (strtod-style leading-number coercion,
// 0 when there is none) and whether the whole token is a numeric string.
func awkNum(s string) (val float64, isNumericString bool) {
	isNumericString = awkNumRe.MatchString(s)
	if m := awkLeadNumRe.FindString(strings.TrimLeft(s, " \t\n")); m != "" {
		val, _ = strconv.ParseFloat(m, 64)
	}
	return val, isNumericString
}

// sameWindow reports whether the last `win` actions are identical, reproducing
// awk's `for (i=NR-win+1; i<=NR; i++)` loop over the numeric window value: a
// non-integer window never matches (fractional array indices are empty), and a
// window ≤ 0 makes the loop empty so `same` stays true.
func sameWindow(h []string, winVal float64) bool {
	n := len(h)
	w := int(winVal)
	if float64(w) != winVal {
		return false
	}
	if w <= 0 {
		return true
	}
	if w > n {
		return false // guarded (the too-few check catches this for numeric windows)
	}
	for i := n - w; i < n; i++ {
		if h[i] != h[n-1] {
			return false
		}
	}
	return true
}

// Stuck is the unattended-build loop detector.
// `log` appends one (tool,target) action record; `check` flags the semantic-repeat
// patterns that mean the agent is burning tokens in a loop: the same action
// repeating, an A<->B ping-pong, or one action dominating the recent window.
// Matching is by (tool,target) sha1, timestamps ignored. root is the .devrites
// directory; the log lives at <root>/features/<slug>/action.log.
//
//	0  ok: not stuck (or logging, or no workspace: never fail the caller on log)
//	2  bad args
//	3  STUCK: the recent window is a loop
func Stuck(root string, args []string, stdout, stderr io.Writer) int {
	cmd := argAt(args, 0)
	slug := argAt(args, 1)
	if cmd == "" || slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine stuck log|check <slug> [...]")
		return 2
	}
	dir := featureDir(root, slug)
	logPath := filepath.Join(dir, "action.log")

	switch cmd {
	case "log":
		tool := argAt(args, 2)
		target := argAt(args, 3)
		if tool == "" {
			fmt.Fprintln(stderr, "usage: devrites-engine stuck log <slug> <tool> [target]")
			return 2
		}
		if !isDir(dir) {
			return 0 // no workspace: never fail the caller
		}
		record := fmt.Sprintf("%d %s %s\n", time.Now().Unix(), tool, sha1Hex(tool+"|"+target))
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return 0
		}
		defer f.Close()
		_, _ = f.WriteString(record)
		return 0

	case "check":
		rawWin := "4" // bash `win="${3:-4}"`
		if w := argAt(args, 2); w != "" {
			rawWin = w
		}
		data, err := os.ReadFile(logPath)
		if err != nil {
			fmt.Fprintln(stdout, "stuck: no actions logged: not stuck.")
			return 0
		}
		return checkStuck(actionHashes(data), rawWin, stdout)

	default:
		fmt.Fprintln(stderr, "usage: devrites-engine stuck log|check <slug> [...]")
		return 2
	}
}

// actionHashes returns the 3rd (sha1) field of every record line, mirroring the
// awk `h[NR]=$3` (an empty string for a line with fewer than three fields).
func actionHashes(data []byte) []string {
	lines := splitLinesNoTrailing(data)
	hashes := make([]string, len(lines))
	for i, line := range lines {
		if fields := strings.Fields(line); len(fields) >= 3 {
			hashes[i] = fields[2]
		}
	}
	return hashes
}

// checkStuck evaluates the hash column for identical-repeat,
// A<->B ping-pong (last 6), and one-action-dominates (≥6 of last 8). The window
// arg is handled with awk-compatible semantics (see awkNum / sameWindow) because
// the prior shell helper passed it verbatim to `awk -v win=`.
func checkStuck(h []string, rawWin string, stdout io.Writer) int {
	n := len(h)
	winVal, isNumericString := awkNum(rawWin)

	// `NR < win`: a numeric comparison when the token is a numeric string, else a
	// string comparison of str(NR) against the raw token: awk's comparison rule.
	tooFew := false
	if isNumericString {
		tooFew = float64(n) < winVal
	} else {
		tooFew = strconv.Itoa(n) < rawWin
	}
	if tooFew {
		fmt.Fprintf(stdout, "stuck: only %d action(s): not stuck.\n", n)
		return 0
	}
	// The last `win` actions are identical. The message echoes the raw token, as
	// awk prints the `win` variable verbatim (so "04"/"+4" render unchanged).
	if sameWindow(h, winVal) {
		fmt.Fprintf(stdout, "stuck: last %s actions identical: STUCK.\n", rawWin)
		return 3
	}
	// A<->B ping-pong over the last 6 actions.
	if n >= 6 {
		pp := true
		for i := n - 6; i < n; i++ {
			expect := h[n-2]
			if (n-1-i)%2 == 0 {
				expect = h[n-1]
			}
			if h[i] != expect {
				pp = false
				break
			}
		}
		if pp && h[n-1] != h[n-2] {
			fmt.Fprintln(stdout, "stuck: A<->B ping-pong over last 6: STUCK.")
			return 3
		}
	}
	// One action dominates the last 8.
	if n >= 8 {
		counts := map[string]int{}
		for i := n - 8; i < n; i++ {
			counts[h[i]]++
		}
		top := 0
		for _, c := range counts {
			if c > top {
				top = c
			}
		}
		if top >= 6 {
			fmt.Fprintf(stdout, "stuck: one action %d/8 of recent window: STUCK.\n", top)
			return 3
		}
	}
	fmt.Fprintln(stdout, "stuck: recent window varied: not stuck.")
	return 0
}

// sha1Hex returns the lowercase hex sha1 of s, matching `sha1sum`/`shasum`.
func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s)) // #nosec G401 -- dedup fingerprint, not a security boundary; persisted records depend on sha1
	return hex.EncodeToString(sum[:])
}

// argAt returns args[i] or "": the Go analogue of bash's ${N:-} positional reads.
func argAt(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}
