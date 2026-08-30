package state

//go:generate go run ./cmd/workflowmanifest -out workflow_manifest.json

// SchemaVersion versions the persisted workflow/state manifest and the
// workspace contract the engine accepts.
const SchemaVersion = 3

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

// ArtifactPath is the logical identity of an observed workspace artifact.
type ArtifactPath string

// PhasePolicy is the complete deterministic policy for one target Phase.
type PhasePolicy struct {
	Target              Phase
	ResumeVerb          string
	TransitionRight     string
	RequiredSections    []Section
	RequiredArtifacts   []ArtifactPath
	ProofRequired       bool
	BlocksOpenQuestions bool
	Shippable           bool
}

var (
	sectionsSpec     = []Section{SectionSpec}
	sectionsPlan     = []Section{SectionSpec, SectionPlan}
	sectionsBuild    = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks}
	sectionsProof    = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof}
	sectionsComplete = []Section{SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof, SectionStatus}

	artifactsFrame   = []ArtifactPath{"state.md"}
	artifactsSpec    = []ArtifactPath{"brief.md", "spec.md", "state.md", "decisions.md", "assumptions.md", "questions.md"}
	artifactsClarify = append(append([]ArtifactPath(nil), artifactsSpec...), "decision-coverage.md")
	artifactsPlan    = append(append([]ArtifactPath(nil), artifactsClarify...), "architecture.md", "plan.md", "tasks.md", "traceability.md")
	artifactsVetted  = append(append([]ArtifactPath(nil), artifactsPlan...), "eng-review.md", "test-plan.md")
	artifactsProof   = append(append([]ArtifactPath(nil), artifactsVetted...), "evidence.md", "touched-files.md")
	artifactsFinal   = append(append([]ArtifactPath(nil), artifactsProof...), "review.md", "seal.md")
)

// Completeness is phase-relative: a section not yet required (e.g. proof during
// spec) never blocks. Requirements are additive down the arc. Define is active
// authoring; Plan is the approved/repaired checkpoint that resumes at Vet.
var orderedPhasePolicies = []PhasePolicy{
	{Target: PhaseFrame, ResumeVerb: "frame", TransitionRight: "Frame an unstructured request before lifecycle work.", RequiredArtifacts: artifactsFrame},
	{Target: PhaseSpec, ResumeVerb: "spec", TransitionRight: "Author the product specification.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsSpec},
	{Target: PhaseClarify, ResumeVerb: "clarify", TransitionRight: "Close decision coverage in the written specification.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsClarify, BlocksOpenQuestions: true},
	{Target: PhaseTemper, ResumeVerb: "temper", TransitionRight: "Optionally challenge the clarified specification strategy.", RequiredSections: sectionsSpec, RequiredArtifacts: artifactsClarify, BlocksOpenQuestions: true},
	{Target: PhaseDefine, ResumeVerb: "define", TransitionRight: "Author and approve the initial implementation plan.", RequiredSections: sectionsPlan, RequiredArtifacts: artifactsPlan, BlocksOpenQuestions: true},
	{Target: PhasePlan, ResumeVerb: "vet", TransitionRight: "Hold the approved or repaired plan checkpoint for engineering review.", RequiredSections: sectionsPlan, RequiredArtifacts: artifactsPlan, BlocksOpenQuestions: true},
	{Target: PhaseVet, ResumeVerb: "vet", TransitionRight: "Review implementation readiness before build.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
	{Target: PhaseBuild, ResumeVerb: "build", TransitionRight: "Implement the next approved vertical slice.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
	{Target: PhaseConverge, ResumeVerb: "converge", TransitionRight: "Recover unmet clarified intent into new slices.", RequiredSections: sectionsBuild, RequiredArtifacts: artifactsVetted, BlocksOpenQuestions: true},
	{Target: PhaseProve, ResumeVerb: "prove", TransitionRight: "Produce acceptance evidence for the implementation.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
	{Target: PhasePolish, ResumeVerb: "polish", TransitionRight: "Apply the bounded quality pass.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
	{Target: PhaseReview, ResumeVerb: "review", TransitionRight: "Review the proven implementation.", RequiredSections: sectionsProof, RequiredArtifacts: artifactsProof, ProofRequired: true, BlocksOpenQuestions: true},
	{Target: PhaseSeal, ResumeVerb: "seal", TransitionRight: "Decide the final GO or NO-GO.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
	{Target: PhaseShip, ResumeVerb: "ship", TransitionRight: "Perform authorized release and close-out mutations.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
	{Target: PhaseDone, TransitionRight: "Represent archived completion with no resume command.", RequiredSections: sectionsComplete, RequiredArtifacts: artifactsFinal, ProofRequired: true, BlocksOpenQuestions: true, Shippable: true},
}

var phasePolicyIndex = func() map[Phase]int {
	index := make(map[Phase]int, len(orderedPhasePolicies))
	for i, policy := range orderedPhasePolicies {
		index[policy.Target] = i
	}
	return index
}()

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

func copyPhasePolicy(policy PhasePolicy) PhasePolicy {
	policy.RequiredSections = append([]Section(nil), policy.RequiredSections...)
	policy.RequiredArtifacts = append([]ArtifactPath(nil), policy.RequiredArtifacts...)
	return policy
}

// PolicyFor returns the complete policy for target. Unknown phases have no fallback.
func PolicyFor(target Phase) (PhasePolicy, bool) {
	index, ok := phasePolicyIndex[target]
	if !ok {
		return PhasePolicy{}, false
	}
	return copyPhasePolicy(orderedPhasePolicies[index]), true
}

// PhasePolicies returns all policies in lifecycle order.
func PhasePolicies() []PhasePolicy {
	policies := make([]PhasePolicy, len(orderedPhasePolicies))
	for i, policy := range orderedPhasePolicies {
		policies[i] = copyPhasePolicy(policy)
	}
	return policies
}
