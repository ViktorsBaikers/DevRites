package state

//go:generate go run ./cmd/workflowmanifest -out workflow_manifest.json

// SchemaVersion is the .devrites state-schema version this engine understands.
// A workspace map may declare its own schemaVersion in frontmatter; the engine
// refuses a version newer than this (see LoadFeature) and otherwise reads the
// files, which evolve additively.
const SchemaVersion = 1

const (
	WorkspaceMapFile = "README.md"
	EvidenceFile     = "evidence.md"
)

var workspaceMapFiles = []string{WorkspaceMapFile, "feature.md", "index.md"}

// WorkspaceMapFiles returns the canonical workspace map followed by readable aliases.
func WorkspaceMapFiles() []string {
	return append([]string(nil), workspaceMapFiles...)
}

// Section is one single-concern completeness file within a feature directory.
// Splitting a feature into small files (rather than one long document) keeps
// each file context-cheap and makes completeness self-evident.
type Section string

const (
	SectionSpec      Section = "spec"      // outcome, scope, constraints
	SectionPlan      Section = "plan"      // approach / implementation plan
	SectionDecisions Section = "decisions" // decision log
	SectionTasks     Section = "tasks"     // task breakdown with status
	SectionProof     Section = "proof"     // acceptance criteria / evidence
	SectionStatus    Section = "status"    // status checkpoint
)

// Sections is the canonical order sections are read and displayed in.
var Sections = []Section{
	SectionSpec,
	SectionPlan,
	SectionDecisions,
	SectionTasks,
	SectionProof,
	SectionStatus,
}

// sectionFiles lists the filenames that can satisfy each section, canonical name
// first, then supported aliases — the same mapping `devrites-engine migrate`
// normalizes (proof→evidence, status→state). A section
// counts as present if ANY of its files has real content, so the engine reads a
// live workspace before the pack sweep converges the filenames. The workspace
// map is not a section; it is handled separately in LoadFeature.
var sectionFiles = map[Section][]string{
	SectionSpec:      {"spec.md"},
	SectionPlan:      {"plan.md"},
	SectionDecisions: {"decisions.md"},
	SectionTasks:     {"tasks.md"},
	SectionProof:     {EvidenceFile, "proof.md"},
	SectionStatus:    {"state.md", "status.md"},
}

// LedgerFile is the working-state ledger the live pack writes. It carries the
// phase in its canonical cursor table (legacy "- Phase: <p>" remains readable)
// when no workspace map declares one, and it satisfies the status section.
const LedgerFile = "state.md"

// Phase is a workflow state. The order mirrors the rite-* arc.
type Phase string

const (
	PhaseFrame    Phase = "frame"    // problem framing
	PhaseSpec     Phase = "spec"     // specification
	PhaseTemper   Phase = "temper"   // strategic specification review
	PhaseDefine   Phase = "define"   // plan definition
	PhasePlan     Phase = "plan"     // approved plan
	PhaseVet      Phase = "vet"      // pre-build engineering review
	PhaseBuild    Phase = "build"    // implementation
	PhaseConverge Phase = "converge" // post-build gap closure
	PhaseProve    Phase = "prove"    // proof / testing
	PhasePolish   Phase = "polish"   // quality pass
	PhaseReview   Phase = "review"   // post-proof review
	PhaseSeal     Phase = "seal"     // completeness seal
	PhaseShip     Phase = "ship"     // shipping
	PhaseDone     Phase = "done"     // archived completion
)

// phaseDefinition is the single source of truth for lifecycle order and
// phase-specific behavior. Workflow topology is versioned application logic,
// rather than deploy-time configuration, so keeping it typed and ordered makes
// additions compile-visible and lets every consumer derive the same arc.
type phaseDefinition struct {
	phase               Phase
	resumeVerb          string
	required            []Section
	aliases             []string
	workspaceRequired   []string
	proofRequired       bool
	blocksOpenQuestions bool
	shippable           bool
}

var (
	sectionsSpec     = []Section{SectionSpec}
	sectionsPlan     = []Section{SectionSpec, SectionPlan}
	sectionsBuild    = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks}
	sectionsProof    = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof}
	sectionsComplete = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof, SectionStatus}

	workspaceFrame = []string{"state.md"}
	workspaceSpec  = []string{"brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"}
	workspacePlan  = []string{"brief.md", "spec.md", "architecture.md", "plan.md", "tasks.md", "traceability.md", "state.md", "decisions.md", "assumptions.md", "questions.md"}
	workspaceProof = append(append([]string(nil), workspacePlan...), "evidence.md", "touched-files.md")
)

// Completeness is phase-relative: a section not yet required (e.g. proof during
// spec) never blocks. Requirements are additive down the arc. Plan resumes at
// define because plan is the artifact state produced by rite-define; done has no
// resume command.
var phaseDefinitions = []phaseDefinition{
	{phase: PhaseFrame, resumeVerb: "frame", workspaceRequired: workspaceFrame},
	{phase: PhaseSpec, resumeVerb: "spec", required: sectionsSpec, aliases: []string{"specced", "specifying"}, workspaceRequired: workspaceSpec},
	{phase: PhaseTemper, resumeVerb: "temper", required: sectionsSpec, aliases: []string{"tempered", "tempering"}, workspaceRequired: workspaceSpec},
	{phase: PhaseDefine, resumeVerb: "define", required: sectionsPlan, aliases: []string{"defined", "defining"}, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhasePlan, resumeVerb: "define", required: sectionsPlan, aliases: []string{"planned", "planning"}, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhaseVet, resumeVerb: "vet", required: sectionsBuild, aliases: []string{"vetted", "vetting"}, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhaseBuild, resumeVerb: "build", required: sectionsBuild, aliases: []string{"building", "wip", "in", "in-progress"}, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhaseConverge, resumeVerb: "converge", required: sectionsBuild, aliases: []string{"converged", "converging"}, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhaseProve, resumeVerb: "prove", required: sectionsProof, aliases: []string{"proving", "proven", "testing"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhasePolish, resumeVerb: "polish", required: sectionsProof, aliases: []string{"polished", "polishing"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhaseReview, resumeVerb: "review", required: sectionsProof, aliases: []string{"reviewed", "reviewing"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhaseSeal, resumeVerb: "seal", required: sectionsComplete, aliases: []string{"sealed", "sealing"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true, shippable: true},
	{phase: PhaseShip, resumeVerb: "ship", required: sectionsComplete, aliases: []string{"shipped", "shipping"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true, shippable: true},
	{phase: PhaseDone, required: sectionsComplete, aliases: []string{"closed", "complete", "completed"}, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true, shippable: true},
}

// WorkflowPhase is the read-only cross-format view used to generate the compact
// manifest consumed by non-Go release tooling.
type WorkflowPhase struct {
	ID                  Phase    `json:"id"`
	ResumeVerb          string   `json:"resumeVerb,omitempty"`
	Aliases             []string `json:"aliases,omitempty"`
	WorkspaceRequired   []string `json:"workspaceRequired"`
	ProofRequired       bool     `json:"proofRequired,omitempty"`
	BlocksOpenQuestions bool     `json:"blocksOpenQuestions,omitempty"`
	Shippable           bool     `json:"shippable,omitempty"`
}

// WorkflowPhases returns copied metadata suitable for deterministic generation.
func WorkflowPhases() []WorkflowPhase {
	out := make([]WorkflowPhase, 0, len(phaseDefinitions))
	for _, definition := range phaseDefinitions {
		out = append(out, WorkflowPhase{
			ID:                  definition.phase,
			ResumeVerb:          definition.resumeVerb,
			Aliases:             append([]string(nil), definition.aliases...),
			WorkspaceRequired:   append([]string(nil), definition.workspaceRequired...),
			ProofRequired:       definition.proofRequired,
			BlocksOpenQuestions: definition.blocksOpenQuestions,
			Shippable:           definition.shippable,
		})
	}
	return out
}

// LifecyclePhases returns the ordered lifecycle. The returned slice is a copy,
// so callers cannot mutate the registry.
func LifecyclePhases() []Phase {
	phases := make([]Phase, len(phaseDefinitions))
	for i, definition := range phaseDefinitions {
		phases[i] = definition.phase
	}
	return phases
}

func definitionFor(p Phase) (phaseDefinition, bool) {
	for _, definition := range phaseDefinitions {
		if definition.phase == p {
			return definition, true
		}
	}
	return phaseDefinition{}, false
}

// KnownPhase reports whether p is a phase the engine understands.
func KnownPhase(p Phase) bool {
	_, ok := definitionFor(p)
	return ok
}

// ResumeVerb returns the public rite verb that resumes p. Terminal and unknown
// phases return an empty string.
func ResumeVerb(p Phase) string {
	definition, ok := definitionFor(p)
	if !ok {
		return ""
	}
	return definition.resumeVerb
}

// ShippablePhase reports whether p is allowed to claim a sealed/shipped result.
func ShippablePhase(p Phase) bool {
	definition, ok := definitionFor(p)
	return ok && definition.shippable
}

// PhaseForName resolves a canonical phase ID or compatibility alias. Callers
// should normalize surrounding syntax before querying it.
func PhaseForName(name string) (Phase, bool) {
	for _, definition := range phaseDefinitions {
		if string(definition.phase) == name {
			return definition.phase, true
		}
		for _, alias := range definition.aliases {
			if alias == name {
				return definition.phase, true
			}
		}
	}
	return "", false
}

// RequiredSections returns the sections required to complete the given phase,
// in canonical Sections order.
func RequiredSections(p Phase) []Section {
	definition, ok := definitionFor(p)
	if !ok {
		return nil
	}
	want := definition.required
	set := make(map[Section]bool, len(want))
	for _, s := range want {
		set[s] = true
	}
	out := make([]Section, 0, len(want))
	for _, s := range Sections {
		if set[s] {
			out = append(out, s)
		}
	}
	return out
}
