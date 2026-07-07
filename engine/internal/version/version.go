// Package version records the engine binary's own version and knows how to
// compare semantic versions for skew detection. The value is stamped at build
// time via -ldflags "-X .../internal/version.Version=vX.Y.Z"; unstamped
// developer builds report "dev", which compares as incomparable (never a false
// skew warning).
package version

import (
	"strconv"
	"strings"
)

// Version is the binary's version, overridden at release build time by ldflags.
var Version = "dev"

// Compare returns -1, 0, or +1 as semantic version a is less than, equal to, or
// greater than b, and ok=false when either side is not a plain dotted-number
// version (e.g. "dev"). A leading "v" is tolerated; pre-release/build suffixes
// are ignored. Callers use ok=false to mean "not comparable — do not warn".
func Compare(a, b string) (cmp int, ok bool) {
	pa, oka := parse(a)
	pb, okb := parse(b)
	if !oka || !okb {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		switch {
		case pa[i] < pb[i]:
			return -1, true
		case pa[i] > pb[i]:
			return 1, true
		}
	}
	return 0, true
}

// parse extracts [major, minor, patch] from a version string, tolerating a
// leading "v" and ignoring anything from the first '-' or '+'. Fewer than three
// components pad with zeros; a non-numeric component makes it incomparable.
func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return out, false
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
