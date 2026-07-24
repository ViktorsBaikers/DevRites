package state

import (
	"fmt"
	"strings"
)

// Report is the computed completeness status of a feature at its current phase.
// It embeds the loaded Feature (Slug, Phase, Present) and adds the phase-relative
// required set and the missing sections.
type Report struct {
	*Feature
	Required      map[Section]bool
	Missing       []Section // legacy required-but-empty sections, in Sections order
	RequiredFiles map[string]bool
	MissingFiles  []string // required-but-empty workspace files, in lifecycle order
}

// Status computes the status report for feature <slug> under root by reading
// the files directly. It reports completeness relative to the feature's current
// phase only.
func Status(root, slug string) (*Report, error) {
	f, err := LoadFeature(root, slug)
	if err != nil {
		return nil, fmt.Errorf("compute status: %w", err)
	}
	return NewReport(f), nil
}

// NewReport computes the phase-relative required set and missing sections for a
// loaded Feature.
func NewReport(f *Feature) *Report {
	required := make(map[Section]bool)
	for _, s := range RequiredSections(f.Phase) {
		required[s] = true
	}
	requiredFiles := make(map[string]bool)
	var missingFiles []string
	if !f.LegacyLayout {
		for _, name := range RequiredWorkspaceFiles(f.Phase) {
			requiredFiles[name] = true
		}
		missingFiles = MissingWorkspaceFiles(f, f.Phase)
	}
	return &Report{
		Feature:       f,
		Required:      required,
		Missing:       MissingFor(f, f.Phase),
		RequiredFiles: requiredFiles,
		MissingFiles:  missingFiles,
	}
}

// MissingFor returns the sections required to complete phase p that the feature
// does not yet have real content for, in canonical Sections order. Gates use it
// to check completeness against a phase other than the feature's current one
// (e.g. seal always checks the full seal-phase set), so its result is
// independent of f.Phase.
func MissingFor(f *Feature, p Phase) []Section {
	required := make(map[Section]bool)
	for _, s := range RequiredSections(p) {
		required[s] = true
	}
	var missing []Section
	for _, s := range Sections {
		if required[s] && !f.Present[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

// MissingWorkspaceFiles returns the concrete lifecycle artifacts required for
// p that do not contain real content. It is the authoritative runtime
// completeness view; Section completeness remains only a compact legacy view.
func MissingWorkspaceFiles(f *Feature, p Phase) []string {
	var missing []string
	for _, name := range RequiredWorkspaceFiles(p) {
		if !f.PresentFiles[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// Complete reports whether every concrete canonical-workspace file required by
// the current phase has real content. Compatibility features/ workspaces keep
// their legacy section-based contract until migration moves them to work/.
func (r *Report) Complete() bool {
	if r.LegacyLayout {
		return len(r.Missing) == 0
	}
	return len(r.MissingFiles) == 0
}

// Render produces the deterministic, greppable status text, including a
// trailing newline. Each section line reads "<name> <present|empty> [required]".
func (r *Report) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "feature: %s\n", r.Slug)
	fmt.Fprintf(&b, "phase: %s\n", r.Phase)
	for _, s := range Sections {
		state := "empty"
		if r.Present[s] {
			state = "present"
		}
		mark := ""
		if r.Required[s] {
			mark = "required"
		}
		line := fmt.Sprintf("  %-10s %-8s %s", s, state, mark)
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteByte('\n')
	}
	if r.Complete() {
		b.WriteString("result: complete\n")
	} else if r.LegacyLayout {
		missing := make([]string, len(r.Missing))
		for i, section := range r.Missing {
			missing[i] = string(section)
		}
		fmt.Fprintf(&b, "result: incomplete (missing: %s)\n", strings.Join(missing, ", "))
	} else {
		fmt.Fprintf(&b, "result: incomplete (missing files: %s)\n", strings.Join(r.MissingFiles, ", "))
	}
	return b.String()
}
