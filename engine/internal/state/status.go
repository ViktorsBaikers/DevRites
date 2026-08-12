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
	policy, _ := PolicyFor(f.Phase)
	required := make(map[Section]bool, len(policy.RequiredSections))
	for _, section := range policy.RequiredSections {
		required[section] = true
	}
	requiredFiles := make(map[string]bool, len(policy.RequiredArtifacts))
	for _, artifact := range policy.RequiredArtifacts {
		requiredFiles[string(artifact)] = true
	}
	return &Report{
		Feature:       f,
		Required:      required,
		Missing:       missingSectionsForPolicy(f, policy),
		RequiredFiles: requiredFiles,
		MissingFiles:  missingArtifactsForPolicy(f, policy),
	}
}

// MissingFor returns the sections required to complete phase p that the feature
// does not yet have real content for, in canonical Sections order. Gates use it
// to check completeness against a phase other than the feature's current one
// (e.g. seal always checks the full seal-phase set), so its result is
// independent of f.Phase.
func MissingFor(f *Feature, p Phase) []Section {
	policy, _ := PolicyFor(p)
	return missingSectionsForPolicy(f, policy)
}

func missingSectionsForPolicy(f *Feature, policy PhasePolicy) []Section {
	var missing []Section
	for _, section := range policy.RequiredSections {
		if !f.Present[section] {
			missing = append(missing, section)
		}
	}
	return missing
}

// MissingWorkspaceFiles returns the concrete lifecycle artifacts required for
// p that do not contain real content. It is the authoritative runtime
// completeness view; Section completeness remains only a compact legacy view.
func MissingWorkspaceFiles(f *Feature, p Phase) []string {
	policy, _ := PolicyFor(p)
	return missingArtifactsForPolicy(f, policy)
}

func missingArtifactsForPolicy(f *Feature, policy PhasePolicy) []string {
	var missing []string
	for _, artifact := range policy.RequiredArtifacts {
		name := string(artifact)
		if !f.PresentFiles[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// Complete reports whether every concrete canonical-workspace file required by
// the current phase has real content.
func (r *Report) Complete() bool {
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
	} else {
		fmt.Fprintf(&b, "result: incomplete (missing files: %s)\n", strings.Join(r.MissingFiles, ", "))
	}
	return b.String()
}
