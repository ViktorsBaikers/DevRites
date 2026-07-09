// Package orient builds the SessionStart orientation digest injected at the top
// of a session, so the model knows a DevRites workflow is in flight and where it
// stands. It reads workspace state through the index (which may lazily heal the
// disposable state.db cache) and makes zero model or network calls. It is silent
// when there is nothing to say, mirroring the legacy devrites-orient.sh: an
// enhancement, never a dependency.
package orient

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/index"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/state"
)

// ActiveFile names the pointer file, under the .devrites root, that records the
// slug of the feature currently being worked. An absent or empty pointer means
// no feature is active and orientation stays silent.
const ActiveFile = "ACTIVE"

// Digest returns the orientation text for the workspace at root and whether
// there is anything to say. It returns ("", false, nil) — silent, not an error —
// when no feature is active or the active pointer is stale, so a session is never
// disrupted by a missing or half-set-up workspace.
func Digest(root string) (text string, has bool, err error) {
	slug, err := ActiveSlug(root)
	if err != nil {
		return "", false, err
	}
	if slug == "" {
		return "", false, nil
	}

	// A stale pointer (names a feature that no longer exists) stays silent
	// rather than erroring — orientation must never be a session dependency.
	report, err := index.Status(root, slug)
	if err != nil {
		return "", false, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "DevRites workflow active — feature %q at phase %q.\n\n", report.Slug, report.Phase)
	b.WriteString(report.Render())

	// Surface only the high-confidence, human-promoted learnings, capped so a
	// growing ledger never bloats the session. Bounds come from
	// DEVRITES_LEARNING_CONFIDENCE_THRESHOLD / DEVRITES_MAX_INJECTED_LEARNINGS.
	if picks := lib.TopLearnings(root, lib.InjectMax(), lib.InjectThreshold()); len(picks) > 0 {
		b.WriteString("\n\nHigh-confidence learnings (untrusted prior — live code always wins):\n")
		for _, p := range picks {
			b.WriteString(p)
			b.WriteByte('\n')
		}
	}

	b.WriteString("\nThis is the current state; the next host-specific DevRites command (/rite-* in Claude Code, $rite-* in Codex) will re-run its own step-0 preamble.")
	return b.String(), true, nil
}

// ActiveSlug reads and trims the active-feature pointer. A missing pointer file
// is not an error — it yields the empty slug (no active feature).
func ActiveSlug(root string) (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE")); raw != "" {
		path := raw
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(root), path)
		}
		return filepath.Base(filepath.Clean(path)), nil
	}
	raw, err := os.ReadFile(filepath.Join(root, ActiveFile))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// ResolveRoot is a thin re-export so hook callers can fail open on a
// non-DevRites directory without importing state directly.
var ResolveRoot = state.ResolveRoot
