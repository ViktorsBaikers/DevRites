package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/secretpattern"
)

// CandidateDigest computes the feature's candidate identity. main wires it to
// the candidate hash in package lib, which imports this package; nil reports
// the identity as unavailable.
var CandidateDigest func(root, slug string) (string, error)

// maxTranscriptOutput caps the redacted output a transcript keeps.
const maxTranscriptOutput = 1 << 20

var unsafeNameRE = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func candidateLabel(root, slug string) string {
	if CandidateDigest == nil {
		return "unavailable: no candidate identity"
	}
	digest, err := CandidateDigest(root, slug)
	if err != nil {
		return "unavailable: " + oneLine(err.Error())
	}
	return digest
}

// writeTranscripts records one immutable file per executed or reused gate under
// <work>/references/gates/<gate-id>/, binding the exact command, cwd, attempt,
// exit, EXPECT verdict and redacted output to the candidate identity read
// before and after the run. OUTPUT_SHA256 equals the ledger evidence digest.
func writeTranscripts(work string, doc *Document, results []GateResult, before, after string) ([]string, error) {
	gates := map[string]*Gate{}
	for _, gate := range doc.Gates {
		gates[gate.ID] = gate
	}
	prefix := "nodigest"
	if len(before) >= 12 && !strings.HasPrefix(before, "unavailable") {
		prefix = before[:12]
	}
	var paths []string
	for _, result := range results {
		gate := gates[result.ID]
		if gate == nil || (!result.Ran && !result.Reused) {
			continue
		}
		name := unsafeNameRE.ReplaceAllString(gate.ID, "_")
		if name == "." || name == ".." {
			name = "_"
		}
		dir := filepath.Join(work, "references", "gates", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return paths, err
		}
		var body strings.Builder
		fmt.Fprintf(&body, "DEVRITES-GATE-TRANSCRIPT v1\nGATE: %s\nTITLE: %s\nCOMMAND: %s\nCWD: %s\n", gate.ID, gate.Title, gate.Check, valueOr(gate.Cwd, "."))
		fmt.Fprintf(&body, "INPUTS: %s\nINPUTS_SHA256: %s\nCANDIDATE_BEFORE: %s\nCANDIDATE_AFTER: %s\n", valueOr(gate.Inputs, "none"), valueOr(result.Inputs, "none"), before, after)
		var file string
		if result.Reused {
			file = filepath.Join(dir, prefix+"-reused.log")
			fmt.Fprintf(&body, "RESULT: reused (inputs unchanged; not executed)\nPRIOR_EVIDENCE: %s\n", gate.Evidence)
		} else {
			attempt := 1
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), prefix+"-a") {
						attempt++
					}
				}
			}
			file = filepath.Join(dir, fmt.Sprintf("%s-a%d.log", prefix, attempt))
			output, masked := secretpattern.Redact(result.Outcome.Output)
			truncated := len(output) > maxTranscriptOutput
			if truncated {
				output = output[:maxTranscriptOutput]
			}
			verdict := "fail"
			if result.Passed {
				verdict = "pass"
			}
			fmt.Fprintf(&body, "ATTEMPT: %d\nSTARTED: %s\nENDED: %s\nEXIT: %s\nEXPECT: %s\nRESULT: %s\n",
				attempt, result.Started.Format(time.RFC3339Nano), result.Ended.Format(time.RFC3339Nano), exitLabel(result.Outcome), gate.Expect, verdict)
			fmt.Fprintf(&body, "OUTPUT_SHA256: %s\nOUTPUT_BYTES: %d\nREDACTIONS: %d\nTRUNCATED: %t\n--- OUTPUT ---\n%s\n--- END OUTPUT ---\n",
				OutputDigest(result.Outcome.Output), result.Outcome.OutputBytes, masked, truncated, output)
		}
		if err := os.WriteFile(file, []byte(body.String()), 0o644); err != nil {
			return paths, err
		}
		paths = append(paths, file)
	}
	return paths, nil
}

func exitLabel(outcome RunOutcome) string {
	switch {
	case outcome.TimedOut:
		return "timeout"
	case outcome.Err != nil:
		return "error: " + oneLine(outcome.Err.Error())
	}
	return fmt.Sprint(outcome.ExitCode)
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
