package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

// State-parsing patterns ported verbatim from progress.sh's awk. Each name-strip
// sub is anchored (^ or $), so it matches at most once per line — ReplaceAllString
// is therefore equivalent to awk's single-match sub.
var (
	sliceCheckRe   = regexp.MustCompile(`^- \[[^\]]*\][[:space:]]*`)             // drop "- [x] "
	sliceNameRe    = regexp.MustCompile(`^Slice[[:space:]]*[0-9]+:[[:space:]]*`) // drop "Slice N: "
	sliceStateRe   = regexp.MustCompile(`[[:space:]]+(built|pending)[[:space:]]*$`)
	sliceSepRe     = regexp.MustCompile(`[[:space:]]+[^[:alnum:][:space:]]+[[:space:]]*$`) // drop " — "
	sliceCheckedRe = regexp.MustCompile(`\[[xX]\]`)
	sliceHeadingRe = regexp.MustCompile(`^##[[:space:]]+(SLICE-[0-9]{3})(?:[[:space:]]+(.*))?$`)
)

// Progress renders the active feature's position — the deterministic footer every
// workspace-operating rite-* skill prints last. Read-only. Prints nothing and
// exits 0 when there is no active workspace, so pre-spec callers stay silent. root
// is the .devrites directory; slug defaults to the active feature at <root>/ACTIVE.
func Progress(root string, args []string, stdout, stderr io.Writer) int {
	slug := ""
	if len(args) > 0 {
		slug = args[0]
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	workDir := featureDir(root, slug)
	statePath := filepath.Join(workDir, "state.md")
	if slug == "" || !isFile(statePath) {
		return 0 // silent — no active workspace
	}
	stateBytes, err := os.ReadFile(statePath)
	if err != nil {
		return 0
	}
	lines := splitLinesNoTrailing(stateBytes)

	phase := parsePhase(lines)
	if phase == "" {
		phase = "spec"
	}
	total, built, lastbuilt := tallySlices(lines)
	if total == 0 {
		total, built, lastbuilt = tallyCanonicalSlices(workDir)
	}

	// Header rule: pad "── rite-<phase> " with box glyphs to ≥44 BYTES. Under the
	// oracle's LC_ALL=C the bash `${#header}` counts bytes, which len() matches.
	header := "── rite-" + phase + " "
	for len(header) < 44 {
		header += "─"
	}
	fmt.Fprintf(stdout, "%s\n", header)

	// Slice meter: a 10-cell bar, filled = round(built/total*10). Skipped before
	// any slices exist.
	if total > 0 {
		filled := (built*10 + total/2) / total
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
		if built >= total {
			fmt.Fprintf(stdout, "Slices %d/%d  %s  ✅ ALL BUILT\n", built, total, bar)
		} else {
			tail := ""
			if lastbuilt != "" {
				tail = "  " + lastbuilt + " ✓"
			}
			fmt.Fprintf(stdout, "Slice %d/%d  %s%s\n", built, total, bar, tail)
		}
	}

	// Flow ribbon: the lifecycle spine; optional phases appear once their artifact
	// exists or while they are the active phase, so the cursor is never omitted.
	order := make([]string, 0, len(state.LifecyclePhases()))
	for _, lifecyclePhase := range state.LifecyclePhases() {
		name := string(lifecyclePhase)
		include := true
		switch lifecyclePhase {
		case state.PhaseFrame:
			include = phase == name
		case state.PhaseTemper:
			include = phase == name || isFile(filepath.Join(workDir, "strategy.md"))
		case state.PhaseDone:
			include = false
		case state.PhaseVet:
			include = phase == name || isFile(filepath.Join(workDir, "eng-review.md"))
		case state.PhaseConverge:
			include = phase == name
		}
		if include {
			order = append(order, name)
		}
	}

	cur := phase
	idx := -1
	if phase == "done" { // past ship
		idx = len(order)
	} else {
		for n, name := range order {
			if name == cur {
				idx = n
				break
			}
		}
	}
	ribbon := "Flow  "
	for n, name := range order {
		g := "○"
		if n < idx {
			g = "✓"
		} else if n == idx {
			g = "◉"
		}
		ribbon += " " + name + " " + g
	}
	fmt.Fprintf(stdout, "%s\n", ribbon)
	return 0
}

func tallyCanonicalSlices(workDir string) (total, built int, lastbuilt string) {
	raw, err := os.ReadFile(filepath.Join(workDir, "tasks.md"))
	if err != nil {
		return 0, 0, ""
	}
	name, sliceState := "", ""
	finalize := func() {
		if name == "" {
			return
		}
		total++
		if sliceState == "built" {
			built++
			lastbuilt = name
		}
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if match := sliceHeadingRe.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			finalize()
			name = match[1]
			if strings.TrimSpace(match[2]) != "" {
				name += " " + strings.TrimSpace(match[2])
			}
			sliceState = "pending"
			continue
		}
		if name != "" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "status:") {
			_, value, _ := strings.Cut(strings.TrimSpace(line), ":")
			sliceState = strings.ToLower(strings.TrimSpace(value))
		}
	}
	finalize()
	return total, built, lastbuilt
}

// parsePhase returns the first token of the canonical or legacy phase field.
func parsePhase(lines []string) string {
	value, ok := state.CursorField(lines, state.CursorPhase)
	if !ok {
		return ""
	}
	toks := strings.Fields(value)
	if len(toks) == 0 {
		return ""
	}
	return toks[0]
}

// tallySlices reads the "## Slice progress" block: total slice lines, how many are
// checked, and the display name of the last checked one — the awk in progress.sh.
func tallySlices(lines []string) (total, built int, lastbuilt string) {
	inp := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## Slice progress") {
			inp = true
			continue
		}
		if inp && strings.HasPrefix(line, "## ") {
			inp = false
		}
		if inp && strings.HasPrefix(line, "- [") {
			total++
			name := line
			name = sliceCheckRe.ReplaceAllString(name, "")
			name = sliceNameRe.ReplaceAllString(name, "")
			name = sliceStateRe.ReplaceAllString(name, "")
			name = sliceSepRe.ReplaceAllString(name, "")
			if sliceCheckedRe.MatchString(line) {
				built++
				lastbuilt = name
			}
		}
	}
	return total, built, lastbuilt
}
