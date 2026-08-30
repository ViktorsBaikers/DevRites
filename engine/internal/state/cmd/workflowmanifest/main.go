// Command workflowmanifest writes the deterministic cross-language derivative
// of state.PhasePolicies. The typed Go registry remains the authority.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/devrites/devrites/internal/state"
)

type workflowPhase struct {
	ID                  state.Phase          `json:"id"`
	ResumeVerb          string               `json:"resumeVerb,omitempty"`
	TransitionRight     string               `json:"transitionRight"`
	RequiredSections    []state.Section      `json:"requiredSections,omitempty"`
	WorkspaceRequired   []state.ArtifactPath `json:"workspaceRequired"`
	ProofRequired       bool                 `json:"proofRequired,omitempty"`
	BlocksOpenQuestions bool                 `json:"blocksOpenQuestions,omitempty"`
	Shippable           bool                 `json:"shippable,omitempty"`
}

func workflowPhases() []workflowPhase {
	policies := state.PhasePolicies()
	phases := make([]workflowPhase, len(policies))
	for i, policy := range policies {
		phases[i] = workflowPhase{
			ID:                  policy.Target,
			ResumeVerb:          policy.ResumeVerb,
			TransitionRight:     policy.TransitionRight,
			RequiredSections:    policy.RequiredSections,
			WorkspaceRequired:   policy.RequiredArtifacts,
			ProofRequired:       policy.ProofRequired,
			BlocksOpenQuestions: policy.BlocksOpenQuestions,
			Shippable:           policy.Shippable,
		}
	}
	return phases
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

// run is the testable entry point. It returns the process exit code.
func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("workflowmanifest", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	outPath := flags.String("out", "workflow_manifest.json", "output path")
	check := flags.Bool("check", false, "fail when the output is stale")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	document := struct {
		GeneratedBy      string                 `json:"generatedBy"`
		SchemaVersion    int                    `json:"schemaVersion"`
		AuthorityPolicy  state.AuthorityPolicy  `json:"authorityPolicy"`
		Phases           []workflowPhase        `json:"phases"`
		CursorKeyAliases []state.CursorKeyAlias `json:"cursorKeyAliases"`
	}{
		GeneratedBy:      "go generate ./internal/state; DO NOT EDIT",
		SchemaVersion:    state.SchemaVersion,
		AuthorityPolicy:  state.WorkflowAuthorityPolicy(),
		Phases:           workflowPhases(),
		CursorKeyAliases: state.CursorKeyAliases(),
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	data = append(data, '\n')
	if *check {
		current, err := os.ReadFile(*outPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if !bytes.Equal(current, data) {
			fmt.Fprintf(stderr, "%s is stale; run go generate ./internal/state\n", *outPath)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*outPath, data, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
