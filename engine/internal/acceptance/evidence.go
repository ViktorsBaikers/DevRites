package acceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Evidence state vocabulary. A runnable gate is met only when its recorded
// evidence carries the current definition digest; every other runnable state
// is unmet and schedules the gate for execution.
const (
	StateMet        = "met"
	StateUnmet      = "unmet"
	StateStaleUnmet = "stale-unmet"
	StateAbandoned  = "abandoned"
	StateUnapproved = "unapproved" // CHECK is not a vetted test-plan command
)

// EvidenceMarker is the versioned automatic-evidence prefix written by the
// runner. Ordinary prose evidence on a runnable gate never satisfies it; a
// manual gate keeps human attestation.
const EvidenceMarker = "automatic-evidence=v1"

var automaticEvidenceRE = regexp.MustCompile(`^automatic-evidence=v1; def=([0-9a-f]{64}); exit=0; expect=matched; output-sha256=([0-9a-f]{64}); output-bytes=([0-9]+); at=(\S+)$`)

// DefinitionDigest is the environment-independent SHA-256 binding of the
// parsed CHECK, EXPECT, and raw CWD fields. Editing any bound input renders
// previously recorded evidence stale. The digest is unkeyed drift detection,
// not tamper proof: a ledger editor can write a syntactically valid line.
func DefinitionDigest(gate *Gate) string {
	sum := sha256.Sum256([]byte("v1\x00" + gate.Check + "\x00" + gate.Expect + "\x00" + gate.Cwd))
	return hex.EncodeToString(sum[:])
}

// FormatEvidence renders the canonical automatic evidence line payload.
func FormatEvidence(gate *Gate, outputDigest string, outputBytes int, at string) string {
	return fmt.Sprintf("%s; def=%s; exit=0; expect=matched; output-sha256=%s; output-bytes=%d; at=%s",
		EvidenceMarker, DefinitionDigest(gate), outputDigest, outputBytes, at)
}

// OutputDigest is the recorded fingerprint of the canonical successful output
// string. Raw output is never persisted.
func OutputDigest(output string) string {
	sum := sha256.Sum256([]byte(output))
	return hex.EncodeToString(sum[:])
}

// GateState resolves a gate's state from the parsed ledger alone. No command
// executes: status is a pure function of definition binding plus checkbox.
func GateState(gate *Gate, doc *Document) string {
	if _, ok := doc.Abandoned[gate.ID]; ok {
		return StateAbandoned
	}
	if gate.Check == "" {
		// Manual gate: met when checked with any non-empty human evidence.
		evidence := strings.TrimSpace(gate.Evidence)
		if gate.Checked && evidence != "" && evidence != "pending" {
			return StateMet
		}
		return StateUnmet
	}
	if !gate.Checked {
		return StateUnmet
	}
	evidence := strings.TrimSpace(gate.Evidence)
	match := automaticEvidenceRE.FindStringSubmatch(evidence)
	if match == nil || match[1] != DefinitionDigest(gate) {
		return StateStaleUnmet
	}
	return StateMet
}

// Reduction is the non-executing ledger verdict shared by `gates status` and
// the seal gate. Unmet counts every gate that is neither met nor abandoned;
// Stale and Unapproved are diagnostic subsets of it.
type Reduction struct {
	Total      int
	Met        int
	Unmet      int
	Stale      int
	Abandoned  int
	Unapproved int
	Handoffs   []string // abandoned gate ids
	UnmetIDs   []string // all non-met, non-abandoned ids
	StaleIDs   []string
	UnapprIDs  []string
}

// AllMet reports whether every gate is met and none is abandoned.
func (r Reduction) AllMet() bool {
	return r.Total > 0 && r.Met == r.Total && r.Abandoned == 0
}

// HandoffRequired reports whether at least one gate was visibly abandoned: a
// terminal non-successful outcome that can never read as completion.
func (r Reduction) HandoffRequired() bool {
	return r.Abandoned > 0
}

// Reduce computes the ledger reduction. approved is the vetted (command, cwd)
// surface from test-plan.md; a runnable gate whose CHECK is absent also
// records an unapproved diagnostic but always counts as unmet.
func (d *Document) Reduce(approved []ApprovedCommand) Reduction {
	r := Reduction{Total: len(d.Gates)}
	for _, gate := range d.Gates {
		switch GateState(gate, d) {
		case StateMet:
			r.Met++
		case StateAbandoned:
			r.Abandoned++
			r.Handoffs = append(r.Handoffs, gate.ID)
		default:
			r.Unmet++
			r.UnmetIDs = append(r.UnmetIDs, gate.ID)
			if gate.Check == "" {
				continue
			}
			if GateState(gate, d) == StateStaleUnmet {
				r.Stale++
				r.StaleIDs = append(r.StaleIDs, gate.ID)
			}
			if approved != nil && !Approved(gate.Check, gate.Cwd, approved) {
				r.Unapproved++
				r.UnapprIDs = append(r.UnapprIDs, gate.ID)
			}
		}
	}
	return r
}

// normalizeCommand trims outer whitespace for the approval binding. Approval
// is exact-text equality, not shell equivalence: a semantically identical but
// differently spelled command is a different oracle requiring its own vet row.
func normalizeCommand(command string) string {
	return strings.TrimSpace(command)
}
