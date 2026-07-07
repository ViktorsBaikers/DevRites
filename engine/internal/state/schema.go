package state

// SchemaVersion is the .devrites state-schema version this engine understands.
// A feature.md may declare its own schemaVersion in frontmatter; the engine
// refuses a version newer than this (see LoadFeature) and otherwise reads the
// files, which evolve additively.
const SchemaVersion = 1

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

// Phase is a workflow state. The order mirrors the rite-* arc.
type Phase string

const (
	PhaseFrame Phase = "frame" // problem framing
	PhaseSpec  Phase = "spec"  // specification
	PhasePlan  Phase = "plan"  // planning
	PhaseBuild Phase = "build" // implementation
	PhaseProve Phase = "prove" // proof / testing
	PhaseVet   Phase = "vet"   // review
	PhaseSeal  Phase = "seal"  // completeness seal
	PhaseShip  Phase = "ship"  // shipping
)

// requiredByPhase lists the sections that must have real content to complete a
// phase. Completeness is phase-relative: a section not yet required (e.g. proof
// during the spec phase) never blocks. The set is additive down the arc.
var requiredByPhase = map[Phase][]Section{
	PhaseFrame: {},
	PhaseSpec:  {SectionSpec},
	PhasePlan:  {SectionSpec, SectionPlan},
	PhaseBuild: {SectionSpec, SectionPlan, SectionDecisions, SectionTasks},
	PhaseProve: {SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof},
	PhaseVet:   {SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof},
	PhaseSeal:  {SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof, SectionStatus},
	PhaseShip:  {SectionSpec, SectionPlan, SectionDecisions, SectionTasks, SectionProof, SectionStatus},
}

// KnownPhase reports whether p is a phase the engine understands.
func KnownPhase(p Phase) bool {
	_, ok := requiredByPhase[p]
	return ok
}

// RequiredSections returns the sections required to complete the given phase,
// in canonical Sections order.
func RequiredSections(p Phase) []Section {
	want := requiredByPhase[p]
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
