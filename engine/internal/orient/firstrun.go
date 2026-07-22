package orient

import (
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/lib"
)

// FirstRunFile names the marker, under the .devrites root, that records that the
// one-time blank-slate orientation has been shown. It is runtime state like
// ACTIVE: preserved on uninstall, so a reinstall never re-nags.
const FirstRunFile = ".first-run-shown"

// firstRunNudges maps each first-task token to the single orientation line shown
// on the first-ever session with no active feature. A token missing here (or a
// future new token) stays silent: the nudge is an enhancement, never a
// dependency, and prose is owned here rather than in the classifier.
var firstRunNudges = map[string]string{
	"greenfield":           "Greenfield project: start with /rite-spec <feature> to spec the first feature.",
	"brownfield-unadopted": "Existing codebase with no DevRites history: start with /rite-adopt to onboard it.",
	"dirty-worktree":       "Uncommitted changes in the worktree: /rite-frame the work in flight, or /rite-quick for a small fix.",
	"branch-ahead":         "Branch is ahead of its upstream: /rite-review then /rite-ship the unshipped work.",
	"clean-default":        "No active feature: /rite-spec <feature> starts the next one; /rite shows the menu.",
}

// FirstRunDigest returns the one-time blank-slate orientation for a workspace
// with no active feature. It fires at most once per project: the marker is
// written before the text is returned, and a marker that cannot be written
// keeps the digest silent so a broken filesystem never turns into a per-session
// nag. Fail-open throughout: every non-happy path is ("", false).
func FirstRunDigest(root string) (text string, has bool) {
	marker := filepath.Join(root, FirstRunFile)
	if _, err := os.Stat(marker); err == nil {
		return "", false
	}
	nudge, ok := firstRunNudges[lib.FirstTaskToken(root)]
	if !ok {
		return "", false
	}
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", false // marker unwritable (or raced): silent rather than nagging
	}
	if err := f.Close(); err != nil {
		return "", false
	}
	return "DevRites is installed but no feature is active. " + nudge +
		"\n(One-time orientation; /rite shows the full menu.)", true
}
