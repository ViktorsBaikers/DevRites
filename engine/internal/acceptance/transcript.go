package acceptance

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/secretpattern"
)

// CandidateDigest computes the feature's candidate identity. main wires it to
// the candidate hash in package lib, which imports this package; nil reports
// the identity as unavailable.
var CandidateDigest func(root, slug string) (string, error)

// maxTranscriptOutput caps the redacted output a transcript keeps.
const maxTranscriptOutput = 1 << 20

// Redaction can expand short credentials in metadata as well as output.
const maxTranscriptBytes = 2*MaxLedgerBytes + maxTranscriptOutput + (64 << 10)

var unsafeNameRE = regexp.MustCompile(`[^A-Za-z0-9._-]`)
var transcriptDigestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

func candidateLabel(root, slug string) string {
	if CandidateDigest == nil {
		return "unavailable: no candidate identity"
	}
	digest, err := CandidateDigest(root, slug)
	if err != nil {
		return "unavailable: " + transcriptMetadata(err.Error())
	}
	return digest
}

func transcriptMetadata(value string) string {
	redacted, _ := secretpattern.Redact(oneLine(value))
	return redacted
}

func transcriptGateName(id string) string {
	name := unsafeNameRE.ReplaceAllString(transcriptMetadata(id), "_")
	if name == "." || name == ".." || name == "" {
		return "_"
	}
	return name
}

// transcriptDirectory opens each directory without accepting symlink parents.
// Holding the resulting root also confines publication if a parent is renamed.
func transcriptDirectory(work, id string, create bool) (*os.Root, error) {
	root, err := os.OpenRoot(work)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"references", "gates", transcriptGateName(id)} {
		info, err := root.Lstat(name)
		if errors.Is(err, os.ErrNotExist) && create {
			if err = root.Mkdir(name, 0o700); err == nil || errors.Is(err, os.ErrExist) {
				info, err = root.Lstat(name)
			}
		}
		if err != nil || !info.IsDir() {
			_ = root.Close()
			if err == nil {
				err = fmt.Errorf("unsafe transcript directory %s", name)
			}
			return nil, err
		}
		next, err := root.OpenRoot(name)
		if err == nil {
			var opened os.FileInfo
			opened, err = next.Stat(".")
			if err == nil && !os.SameFile(info, opened) {
				err = fmt.Errorf("transcript directory changed: %s", name)
			}
		}
		_ = root.Close()
		if err != nil {
			if next != nil {
				_ = next.Close()
			}
			return nil, err
		}
		root = next
	}
	return root, nil
}

// writeTranscripts atomically publishes private, complete executed records.
// Reuse points to the original file, preserving its historical candidate labels.
// OUTPUT_SHA256 and OUTPUT_BYTES always describe original, not redacted output.
func writeTranscripts(work string, doc *Document, results []GateResult, before, after string) ([]string, error) {
	gates := map[string]*Gate{}
	for _, gate := range doc.Gates {
		gates[gate.ID] = gate
	}
	prefix := "nodigest"
	if transcriptDigestRE.MatchString(before) {
		prefix = before[:12]
	}
	var paths []string
	for _, result := range results {
		gate := gates[result.ID]
		if gate == nil {
			continue
		}
		if result.Reused && result.Origin != "" {
			paths = append(paths, result.Origin)
			continue
		}
		if !result.Ran {
			continue
		}
		dir, err := transcriptDirectory(work, gate.ID, true)
		if err != nil {
			return paths, err
		}
		file, err := publishTranscript(dir, gate, result, prefix, before, after)
		_ = dir.Close()
		if err != nil {
			return paths, err
		}
		paths = append(paths, filepath.Join(work, "references", "gates", transcriptGateName(gate.ID), file))
	}
	return paths, nil
}

func publishTranscript(dir *os.Root, gate *Gate, result GateResult, prefix, before, after string) (string, error) {
	attempt := 1
	for {
		file := fmt.Sprintf("%s-a%d.log", prefix, attempt)
		if _, err := dir.Lstat(file); errors.Is(err, os.ErrNotExist) {
			break
		} else if err != nil {
			return "", err
		}
		attempt++
	}
	var body strings.Builder
	fmt.Fprintf(&body, "DEVRITES-GATE-TRANSCRIPT v2\nGATE: %s\nTITLE: %s\nCOMMAND: %s\nCWD: %s\nDEFINITION_SHA256: %s\n",
		transcriptMetadata(gate.ID), transcriptMetadata(gate.Title), transcriptMetadata(gate.Check), transcriptMetadata(valueOr(gate.Cwd, ".")), DefinitionDigest(gate))
	fmt.Fprintf(&body, "INPUTS: %s\nINPUTS_SHA256: %s\nCANDIDATE_BEFORE: %s\nCANDIDATE_AFTER: %s\n",
		transcriptMetadata(valueOr(gate.Inputs, "none")), valueOr(result.Inputs, "none"), transcriptMetadata(before), transcriptMetadata(after))
	output, masked := secretpattern.Redact(result.Outcome.Output)
	truncated := len(output) > maxTranscriptOutput
	if truncated {
		output = output[:maxTranscriptOutput]
	}
	verdict := "fail"
	if result.Passed && !result.Discarded && result.WriteErr == nil {
		verdict = "pass"
	}
	fmt.Fprintf(&body, "ATTEMPT: %d\nSTARTED: %s\nENDED: %s\nEXIT: %s\nEXPECT: %s\nRESULT: %s\n",
		attempt, result.Started.Format(time.RFC3339Nano), result.Ended.Format(time.RFC3339Nano), transcriptMetadata(exitLabel(result.Outcome)), transcriptMetadata(gate.Expect), verdict)
	fmt.Fprintf(&body, "OUTPUT_SHA256: %s\nOUTPUT_BYTES: %d\nREDACTIONS: %d\nTRUNCATED: %t\n--- OUTPUT ---\n%s\n--- END OUTPUT ---\n",
		OutputDigest(result.Outcome.Output), result.Outcome.OutputBytes, masked, truncated, output)
	checksum := OutputDigest(body.String())
	fmt.Fprintf(&body, "TRANSCRIPT_SHA256: %s\n", checksum)
	file := fmt.Sprintf("%s-a%d.log", prefix, attempt)
	return file, fsutil.WriteFileAtomicIn(dir, file, []byte(body.String()), 0o600)
}

// passingOrigin accepts only an executed, complete pass bound to ledger proof.
// Candidate labels are historical context, not a dependency of scoped INPUTS.
func passingOrigin(work string, gate *Gate) string {
	evidence := automaticEvidenceRE.FindStringSubmatch(strings.TrimSpace(gate.Evidence))
	if evidence == nil || evidence[1] != DefinitionDigest(gate) {
		return ""
	}
	dir, err := transcriptDirectory(work, gate.ID, false)
	if err != nil {
		return ""
	}
	defer func() { _ = dir.Close() }()
	listing, err := dir.Open(".")
	if err != nil {
		return ""
	}
	entries, err := listing.ReadDir(-1)
	_ = listing.Close()
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		raw, err := readTranscript(dir, entry.Name())
		if err == nil && validOrigin(string(raw), gate, evidence) {
			return filepath.Join(work, "references", "gates", transcriptGateName(gate.ID), entry.Name())
		}
	}
	return ""
}

func readTranscript(dir *os.Root, name string) ([]byte, error) {
	info, err := dir.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxTranscriptBytes {
		return nil, fmt.Errorf("unsafe or oversized transcript")
	}
	file, err := dir.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) {
		return nil, fmt.Errorf("transcript changed while opening")
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxTranscriptBytes+1))
	if err != nil {
		return nil, err
	}
	current, err := dir.Lstat(name)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxTranscriptBytes || !current.Mode().IsRegular() || !os.SameFile(opened, current) || current.Size() != int64(len(raw)) {
		return nil, fmt.Errorf("transcript changed while reading")
	}
	return raw, nil
}

func validOrigin(raw string, gate *Gate, evidence []string) bool {
	const footer = "TRANSCRIPT_SHA256: "
	end := strings.LastIndex(raw, footer)
	if end < 0 || raw[end:] != footer+OutputDigest(raw[:end])+"\n" {
		return false
	}
	header, output, ok := strings.Cut(raw[:end], "\n--- OUTPUT ---\n")
	if !ok || !strings.HasSuffix(output, "\n--- END OUTPUT ---\n") {
		return false
	}
	output = strings.TrimSuffix(output, "\n--- END OUTPUT ---\n")
	lines := strings.Split(header, "\n")
	if lines[0] != "DEVRITES-GATE-TRANSCRIPT v2" {
		return false
	}
	fields := map[string]string{}
	for _, line := range lines[1:] {
		key, value, ok := strings.Cut(line, ": ")
		if _, duplicate := fields[key]; !ok || duplicate {
			return false
		}
		fields[key] = value
	}
	for key, want := range map[string]string{
		"GATE": transcriptMetadata(gate.ID), "TITLE": transcriptMetadata(gate.Title),
		"COMMAND": transcriptMetadata(gate.Check), "CWD": transcriptMetadata(valueOr(gate.Cwd, ".")),
		"EXPECT": transcriptMetadata(gate.Expect), "DEFINITION_SHA256": evidence[1],
		"INPUTS": transcriptMetadata(valueOr(gate.Inputs, "none")), "INPUTS_SHA256": valueOr(evidence[4], "none"),
		"RESULT": "pass", "EXIT": "0", "OUTPUT_SHA256": evidence[2], "OUTPUT_BYTES": evidence[3],
	} {
		if got, exists := fields[key]; !exists || got != want {
			return false
		}
	}
	started, err := time.Parse(time.RFC3339Nano, fields["STARTED"])
	if err != nil || started.IsZero() {
		return false
	}
	ended, err := time.Parse(time.RFC3339Nano, fields["ENDED"])
	if err != nil || ended.Before(started) {
		return false
	}
	attempt, err := strconv.Atoi(fields["ATTEMPT"])
	if err != nil || attempt < 1 || fields["CANDIDATE_BEFORE"] == "" || fields["CANDIDATE_AFTER"] == "" {
		return false
	}
	redactions, err := strconv.Atoi(fields["REDACTIONS"])
	if err != nil || redactions < 0 {
		return false
	}
	truncated, err := strconv.ParseBool(fields["TRUNCATED"])
	return err == nil && len(output) <= maxTranscriptOutput && (!truncated || len(output) == maxTranscriptOutput)
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
