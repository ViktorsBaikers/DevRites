package state

//go:generate go run ./cmd/workflowmanifest -out workflow_manifest.json

// SchemaVersion versions the persisted workflow/state manifest.
const SchemaVersion = 2

const EvidenceFile = "evidence.md"

// Section is one single-purpose completeness file in a feature directory. Small
// files use less context and make missing content easier to spot.
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

// sectionFiles maps each completeness section to its one canonical file.
var sectionFiles = map[Section][]string{
	SectionSpec:      {"spec.md"},
	SectionPlan:      {"plan.md"},
	SectionDecisions: {"decisions.md"},
	SectionTasks:     {"tasks.md"},
	SectionProof:     {EvidenceFile},
	SectionStatus:    {LedgerFile},
}

// LedgerFile is the working-state ledger the live pack writes. It is the phase
// authority through either the canonical cursor table or the released bullet
// form ("- Phase: <p>"), and it satisfies the status section.
const LedgerFile = "state.md"

// Phase is a workflow state. The order mirrors the rite-* arc.
type Phase string

const (
	PhaseFrame    Phase = "frame"    // problem framing
	PhaseSpec     Phase = "spec"     // specification
	PhaseClarify  Phase = "clarify"  // decision-coverage closure
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

// phaseDefinition owns lifecycle order and phase behavior. Workflow topology is
// versioned application logic rather than deployment configuration, so the
// compiler sees changes and every consumer derives the same ordering.
type phaseDefinition struct {
	phase               Phase
	resumeVerb          string
	transitionRight     string
	required            []Section
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

	workspaceFrame   = []string{"state.md"}
	workspaceSpec    = []string{"brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"}
	workspaceClarify = append(append([]string(nil), workspaceSpec...), "decision-coverage.md")
	workspacePlan    = append(append([]string(nil), workspaceClarify...), "architecture.md", "plan.md", "tasks.md", "traceability.md")
	workspaceVetted  = append(append([]string(nil), workspacePlan...), "eng-review.md", "test-plan.md")
	workspaceProof   = append(append([]string(nil), workspaceVetted...), "evidence.md", "touched-files.md")
	workspaceFinal   = append(append([]string(nil), workspaceProof...), "review.md", "seal.md")
)

// Completeness is phase-relative: a section not yet required (e.g. proof during
// spec) never blocks. Requirements are additive down the arc. Define is active
// authoring; Plan is the approved/repaired checkpoint that resumes at Vet.
var phaseDefinitions = []phaseDefinition{
	{phase: PhaseFrame, resumeVerb: "frame", transitionRight: "Frame an unstructured request before lifecycle work.", workspaceRequired: workspaceFrame},
	{phase: PhaseSpec, resumeVerb: "spec", transitionRight: "Author the product specification.", required: sectionsSpec, workspaceRequired: workspaceSpec},
	{phase: PhaseClarify, resumeVerb: "clarify", transitionRight: "Close decision coverage in the written specification.", required: sectionsSpec, workspaceRequired: workspaceClarify, blocksOpenQuestions: true},
	{phase: PhaseTemper, resumeVerb: "temper", transitionRight: "Optionally challenge the clarified specification strategy.", required: sectionsSpec, workspaceRequired: workspaceClarify, blocksOpenQuestions: true},
	{phase: PhaseDefine, resumeVerb: "define", transitionRight: "Author and approve the initial implementation plan.", required: sectionsPlan, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhasePlan, resumeVerb: "vet", transitionRight: "Hold the approved or repaired plan checkpoint for engineering review.", required: sectionsPlan, workspaceRequired: workspacePlan, blocksOpenQuestions: true},
	{phase: PhaseVet, resumeVerb: "vet", transitionRight: "Review implementation readiness before build.", required: sectionsBuild, workspaceRequired: workspaceVetted, blocksOpenQuestions: true},
	{phase: PhaseBuild, resumeVerb: "build", transitionRight: "Implement the next approved vertical slice.", required: sectionsBuild, workspaceRequired: workspaceVetted, blocksOpenQuestions: true},
	{phase: PhaseConverge, resumeVerb: "converge", transitionRight: "Recover unmet clarified intent into new slices.", required: sectionsBuild, workspaceRequired: workspaceVetted, blocksOpenQuestions: true},
	{phase: PhaseProve, resumeVerb: "prove", transitionRight: "Produce acceptance evidence for the implementation.", required: sectionsProof, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhasePolish, resumeVerb: "polish", transitionRight: "Apply the bounded quality pass.", required: sectionsProof, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhaseReview, resumeVerb: "review", transitionRight: "Review the proven implementation.", required: sectionsProof, workspaceRequired: workspaceProof, proofRequired: true, blocksOpenQuestions: true},
	{phase: PhaseSeal, resumeVerb: "seal", transitionRight: "Decide the final GO or NO-GO.", required: sectionsComplete, workspaceRequired: workspaceFinal, proofRequired: true, blocksOpenQuestions: true, shippable: true},
	{phase: PhaseShip, resumeVerb: "ship", transitionRight: "Perform authorized release and close-out mutations.", required: sectionsComplete, workspaceRequired: workspaceFinal, proofRequired: true, blocksOpenQuestions: true, shippable: true},
	{phase: PhaseDone, transitionRight: "Represent archived completion with no resume command.", required: sectionsComplete, workspaceRequired: workspaceFinal, proofRequired: true, blocksOpenQuestions: true, shippable: true},
}

// WorkflowPhase is the read-only cross-format view used to generate the compact
// manifest consumed by non-Go release tooling.
type WorkflowPhase struct {
	ID                  Phase     `json:"id"`
	ResumeVerb          string    `json:"resumeVerb,omitempty"`
	TransitionRight     string    `json:"transitionRight"`
	RequiredSections    []Section `json:"requiredSections,omitempty"`
	WorkspaceRequired   []string  `json:"workspaceRequired"`
	ProofRequired       bool      `json:"proofRequired,omitempty"`
	BlocksOpenQuestions bool      `json:"blocksOpenQuestions,omitempty"`
	Shippable           bool      `json:"shippable,omitempty"`
}

// AuthorityPolicy owns the small cross-format trust and tracking assertions
// that otherwise drift between current docs.
type AuthorityPolicy struct {
	PrinciplesTrust string   `json:"principlesTrust"`
	TrackedState    []string `json:"trackedState"`
	LocalState      []string `json:"localState"`
}

// WorkflowAuthorityPolicy returns copied policy metadata for the manifest.
func WorkflowAuthorityPolicy() AuthorityPolicy {
	return AuthorityPolicy{
		PrinciplesTrust: "Project principles may become project policy only after explicit provenance and validation; arbitrary project-local Markdown is never inherently trusted executable instruction.",
		TrackedState:    []string{".devrites/specs/"},
		LocalState:      []string{".devrites/work/", ".devrites/archive/", ".devrites/ACTIVE"},
	}
}

// WorkflowPhases returns copied metadata suitable for deterministic generation.
func WorkflowPhases() []WorkflowPhase {
	out := make([]WorkflowPhase, 0, len(phaseDefinitions))
	for _, definition := range phaseDefinitions {
		out = append(out, WorkflowPhase{
			ID:                  definition.phase,
			ResumeVerb:          definition.resumeVerb,
			TransitionRight:     definition.transitionRight,
			RequiredSections:    append([]Section(nil), definition.required...),
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

// PhaseForName resolves a canonical phase ID.
func PhaseForName(name string) (Phase, bool) {
	for _, definition := range phaseDefinitions {
		if string(definition.phase) == name {
			return definition.phase, true
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

// RequiredWorkspaceFiles returns the canonical per-file completeness contract
// for phase p. Unlike the legacy Section view, this includes every durable
// workflow artifact (for example decision-coverage.md and test-plan.md), so
// runtime gates cannot silently ignore files added to the lifecycle registry.
func RequiredWorkspaceFiles(p Phase) []string {
	definition, ok := definitionFor(p)
	if !ok {
		return nil
	}
	return append([]string(nil), definition.workspaceRequired...)
}

// WorkspaceFiles returns every lifecycle-owned workspace filename once, in
// first-required order. It is derived from phaseDefinitions so the runtime
// presence model and the generated workflow manifest share one authority.
func WorkspaceFiles() []string {
	seen := make(map[string]bool)
	var out []string
	for _, definition := range phaseDefinitions {
		for _, name := range definition.workspaceRequired {
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
