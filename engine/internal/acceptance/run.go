package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

// MaxLedgerBytes bounds one ledger read; the artifact budget keeps real
// ledgers far below it.
const MaxLedgerBytes = 8 << 20

// ErrOracleMoved marks a discarded result: the gate's bound oracle changed
// between execution and writeback, so the run must not certify it.
var ErrOracleMoved = errors.New("gate oracle changed during the run")

// GateResult is one executed gate's outcome line for the report.
type GateResult struct {
	ID        string
	Ran       bool
	Passed    bool
	Detail    string // exit/match/timeout/approval diagnostic
	Discarded bool   // result dropped: ledger changed under the run
	WriteErr  error  // evidence writeback failed for a reason other than drift
	Reused    bool   // met gate kept because its INPUTS are unchanged
	Inputs    string // INPUTS content digest at run time, when declared
	// Outcome, Started and Ended describe the execution this result judged;
	// they are zero when the gate did not execute.
	Outcome        RunOutcome
	Started, Ended time.Time
}

var (
	uncheckedRE = regexp.MustCompile(`^(\s*)- \[ \] `)
	checkedRE   = regexp.MustCompile(`^(\s*)- \[[xX]\] `)
)

// ReadLedgerFile reads a ledger within the size bound. Caller-resolved path.
func ReadLedgerFile(ledgerPath string) (string, error) {
	info, err := os.Lstat(ledgerPath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("gate ledger is not a regular file: %s", ledgerPath)
	}
	if info.Size() > MaxLedgerBytes {
		return "", fmt.Errorf("gate ledger exceeds %d bytes: %s", MaxLedgerBytes, ledgerPath)
	}
	// #nosec G304 -- ledger path resolved inside the operator workspace
	raw, err := os.ReadFile(ledgerPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// RunGates executes the runnable gates of ledgerPath. reverify re-runs every
// runnable gate including already-met ones; the normal mode runs only unmet
// runnable gates. approved is the vetted command set from test-plan.md; a
// CHECK outside it never executes, and neither does one outside a surface that
// could not be determined — an absent preflight is a refusal, not permission.
// Each pass writes definition-bound evidence; each failure clears the box and
// restores EVIDENCE: pending.
//
// Gates sharing one CHECK and CWD execute once per call; each matches its own
// EXPECT against that shared output. writeLock, when non-nil, wraps every
// ledger writeback so the caller need not hold the feature lock while checks
// execute; writeResult re-verifies the gate's oracle before it writes.
func RunGates(ctx context.Context, root, ledgerPath string, doc *Document, approved []ApprovedCommand, reverify bool, timeout time.Duration, writeLock func(func() error) error) []GateResult {
	var results []GateResult
	outcomes := map[string]RunOutcome{}
	spans := map[string][2]time.Time{}
	write := func(gate *Gate, evidence string, checked bool) error {
		if writeLock == nil {
			return writeResult(ledgerPath, gate, evidence, checked)
		}
		return writeLock(func() error { return writeResult(ledgerPath, gate, evidence, checked) })
	}
	for _, gate := range doc.Gates {
		if _, abandoned := doc.Abandoned[gate.ID]; abandoned {
			results = append(results, GateResult{ID: gate.ID, Detail: "abandoned; handoff required"})
			continue
		}
		if gate.Check == "" {
			results = append(results, GateResult{ID: gate.ID, Detail: "manual gate; evidence is judged, not executed"})
			continue
		}
		inputs := ""
		if gate.Inputs != "" {
			// An unreadable or unmatched spec yields "", which never matches, so the gate re-executes.
			inputs, _ = InputsDigest(root, gate.Inputs)
		}
		if GateState(gate, doc) == StateMet {
			if gate.Inputs == "" && !reverify {
				results = append(results, GateResult{ID: gate.ID, Passed: true, Detail: "already met"})
				continue
			}
			// A gate that declares INPUTS is reused, under run and reverify alike,
			// exactly while the files it depends on are byte-identical; any change re-executes it.
			if inputs != "" && inputs == recordedInputs(gate) {
				results = append(results, GateResult{ID: gate.ID, Passed: true, Reused: true, Inputs: inputs, Detail: "reused: inputs unchanged"})
				continue
			}
		}
		if !Approved(gate.Check, gate.Cwd, approved) {
			results = append(results, GateResult{ID: gate.ID, Detail: "CHECK/CWD is not an approved test-plan.md preflight row; not executed"})
			continue
		}
		expectation, err := CompileExpect(gate.Expect)
		if err != nil {
			results = append(results, GateResult{ID: gate.ID, Detail: err.Error()})
			continue
		}
		cwd := root
		if gate.Cwd != "" {
			cwd = filepath.Join(root, filepath.FromSlash(gate.Cwd))
		}
		key := gate.Check + "\x00" + cwd
		outcome, ran := outcomes[key]
		if !ran {
			started := time.Now().UTC()
			outcome = Execute(ctx, gate.Check, cwd, timeout)
			outcomes[key] = outcome
			spans[key] = [2]time.Time{started, time.Now().UTC()}
		}
		span := spans[key]
		if outcome.Err == nil && outcome.ExitCode == 0 && expectation.Matches(outcome.Output) {
			evidence := withInputs(FormatEvidence(gate, OutputDigest(outcome.Output), outcome.OutputBytes, time.Now().UTC().Format(time.RFC3339)), inputs)
			writeErr := write(gate, evidence, true)
			results = append(results, GateResult{
				ID: gate.ID, Ran: true, Passed: true,
				Discarded: errors.Is(writeErr, ErrOracleMoved), WriteErr: writeErr,
				Detail: resultDetail(writeErr, "pass"),
				Inputs: inputs, Outcome: outcome, Started: span[0], Ended: span[1],
			})
		} else {
			writeErr := write(gate, "pending", false)
			results = append(results, GateResult{
				ID: gate.ID, Ran: true,
				Discarded: errors.Is(writeErr, ErrOracleMoved), WriteErr: writeErr,
				Detail: resultDetail(writeErr, failDetail(outcome, expectation)),
				Inputs: inputs, Outcome: outcome, Started: span[0], Ended: span[1],
			})
		}
	}
	return results
}

func resultDetail(writeErr error, ok string) string {
	if writeErr == nil {
		return ok
	}
	if errors.Is(writeErr, ErrOracleMoved) {
		return ok + "; evidence discarded: " + writeErr.Error()
	}
	return ok + "; evidence write failed: " + writeErr.Error()
}

func failDetail(outcome RunOutcome, expectation Expectation) string {
	if outcome.Err != nil {
		return outcome.Err.Error()
	}
	if outcome.ExitCode != 0 {
		return fmt.Sprintf("exit %d", outcome.ExitCode)
	}
	return fmt.Sprintf("EXPECT %q did not match output", expectation.Literal)
}

// writeResult re-reads the ledger, verifies the gate's bound oracle is
// unchanged since it ran, then flips the checkbox and writes the evidence
// line atomically. A definition that moved in flight discards the result
// instead of certifying a different oracle.
func writeResult(ledgerPath string, ran *Gate, evidence string, checked bool) error {
	text, err := ReadLedgerFile(ledgerPath)
	if err != nil {
		return err
	}
	doc := ParseLedger(text)
	var current *Gate
	for _, gate := range doc.Gates {
		if gate.ID == ran.ID {
			current = gate
			break
		}
	}
	if current == nil {
		return fmt.Errorf("%w: gate %s disappeared", ErrOracleMoved, ran.ID)
	}
	if current.Check != ran.Check || current.Expect != ran.Expect || current.Cwd != ran.Cwd || current.Inputs != ran.Inputs {
		return fmt.Errorf("%w: gate %s definition edited", ErrOracleMoved, ran.ID)
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	masked, err := markdowntext.Structural([]byte(text))
	if err != nil {
		return err
	}
	maskedLines := strings.Split(strings.ReplaceAll(string(masked), "\r\n", "\n"), "\n")
	row := current.Line - 1
	if checked {
		lines[row] = uncheckedRE.ReplaceAllString(lines[row], "${1}- [x] ")
	} else {
		lines[row] = checkedRE.ReplaceAllString(lines[row], "${1}- [ ] ")
	}

	// Attribute-block boundaries come from the masked text so a fenced
	// EVIDENCE example can never be mistaken for this gate's attribute.
	evidenceLine := "  EVIDENCE: " + evidence
	end := row + 1
	evidenceRow := -1
	for end < len(maskedLines) {
		line := maskedLines[end]
		if attrMatch := attrLineRE.FindStringSubmatch(line); attrMatch != nil {
			if attrMatch[2] == "EVIDENCE" {
				evidenceRow = end
			}
			end++
			continue
		}
		if strings.TrimSpace(line) == "" {
			end++
			continue
		}
		break
	}
	if evidenceRow >= 0 {
		lines[evidenceRow] = evidenceLine
	} else {
		insertAt := row + 1
		for insertAt < len(maskedLines) && attrLineRE.MatchString(maskedLines[insertAt]) {
			insertAt++
		}
		lines = append(lines[:insertAt], append([]string{evidenceLine}, lines[insertAt:]...)...)
	}
	out := strings.Join(lines, "\n")
	if doc.Newline() == "\r\n" {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return state.AtomicWrite(ledgerPath, []byte(out), 0o644)
}
