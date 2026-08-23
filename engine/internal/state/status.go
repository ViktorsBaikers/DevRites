package state

import (
	"fmt"
	"strings"
)

// Report is the computed completeness status of a feature at its current phase.
type Report struct {
	Slug          string
	Phase         Phase
	Required      map[Section]bool
	Missing       []Section // required-but-empty sections, in policy order
	RequiredFiles map[string]bool
	MissingFiles  []string // required-but-empty workspace files, in lifecycle order
	present       map[Section]bool
	diagnostics   []ArtifactDiagnostic
}

// Status computes phase-relative completeness from one retained workspace observation.
func Status(root, slug string) (*Report, error) {
	return statusWithCallback(root, slug, nil)
}

func statusWithCallback(root, slug string, callback observationCallback) (*Report, error) {
	observation, err := observeWorkspace(root, slug, callback)
	if err != nil {
		return nil, err
	}
	phase, err := observation.DeclaredPhase()
	if err != nil {
		return nil, fmt.Errorf("compute status: %w", err)
	}
	policy, _ := PolicyFor(phase)
	return newObservationReport(observation, phase, policy), nil
}

func newObservationReport(observation *WorkspaceObservation, phase Phase, policy PhasePolicy) *Report {
	present := observationSectionPresence(observation)
	required := requiredSections(policy)
	requiredFiles := requiredArtifactSet(policy)
	missingArtifacts, diagnostics := observation.Missing(policy.RequiredArtifacts)
	missingFiles := make([]string, len(missingArtifacts))
	for i, artifact := range missingArtifacts {
		missingFiles[i] = string(artifact)
	}
	return &Report{
		Slug:          observation.Slug(),
		Phase:         phase,
		Required:      required,
		Missing:       missingObservationSections(present, policy.RequiredSections),
		RequiredFiles: requiredFiles,
		MissingFiles:  missingFiles,
		present:       present,
		diagnostics:   diagnostics,
	}
}

func observationSectionPresence(observation *WorkspaceObservation) map[Section]bool {
	present := make(map[Section]bool, len(Sections))
	for _, section := range Sections {
		for _, name := range sectionFiles[section] {
			fact, ok := observation.Fact(ArtifactPath(name))
			if ok && fact.State() == ArtifactPresent {
				present[section] = true
				break
			}
		}
	}
	return present
}

func missingObservationSections(present map[Section]bool, required []Section) []Section {
	var missing []Section
	for _, section := range required {
		if !present[section] {
			missing = append(missing, section)
		}
	}
	return missing
}

func requiredSections(policy PhasePolicy) map[Section]bool {
	required := make(map[Section]bool, len(policy.RequiredSections))
	for _, section := range policy.RequiredSections {
		required[section] = true
	}
	return required
}

func requiredArtifactSet(policy PhasePolicy) map[string]bool {
	required := make(map[string]bool, len(policy.RequiredArtifacts))
	for _, artifact := range policy.RequiredArtifacts {
		required[string(artifact)] = true
	}
	return required
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
	for _, section := range Sections {
		state := "empty"
		if r.present[section] {
			state = "present"
		}
		mark := ""
		if r.Required[section] {
			mark = "required"
		}
		line := fmt.Sprintf("  %-10s %-8s %s", section, state, mark)
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteByte('\n')
	}
	for _, diagnostic := range r.diagnostics {
		fmt.Fprintf(&b, "artifact: %s: %s (%s)\n", diagnostic.Path, diagnostic.State, diagnostic.Code)
	}
	if r.Complete() {
		b.WriteString("result: complete\n")
	} else {
		fmt.Fprintf(&b, "result: incomplete (missing files: %s)\n", strings.Join(r.MissingFiles, ", "))
	}
	return b.String()
}
