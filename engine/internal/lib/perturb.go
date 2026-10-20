package lib

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/secretpattern"
)

// PerturbUsage is the help and usage-error text for `check perturb`.
const PerturbUsage = `usage: devrites-engine check perturb <slug> --patch <file> --expect <test-name> --gate <G-ID> [--timeout N]

Runs one safe-perturbation probe without touching the checkout. It copies the
current candidate (HEAD plus every touched-files.md manifest row) into a
detached temporary worktree, requires the copy to hash to the check candidate
digest, and applies the wright-supplied mutant --patch (project-relative or
absolute). It runs the gate's CHECK only when it is an approved test-plan.md
preflight row. RED counts only when a runner FAIL line (go test, pytest, cargo
test, jest/vitest) names --expect; a build or setup failure, a timeout, exit 0
(the mutant survived), or a RED naming only other tests is cannot_verify. It
then reverts the patch, requires the copy back at the candidate digest and the
gate GREEN (exit 0 plus EXPECT), and requires the checkout candidate unchanged.
The transcript is written under evidence/perturb/. Exit 0 verified; exit 3
cannot_verify or refused.
`

const perturbOutputTail = 64 << 10

var (
	perturbNameRE = regexp.MustCompile(`[^A-Za-z0-9._-]`)
	// perturbFailREs capture the failed test name from one runner FAIL line.
	perturbFailREs = []*regexp.Regexp{
		regexp.MustCompile(`^\s*--- FAIL: (\S+)`),                                 // go test
		regexp.MustCompile(`^FAILED (\S+)`),                                       // pytest short summary
		regexp.MustCompile(`^test (\S+) \.\.\. FAILED$`),                          // cargo test
		regexp.MustCompile(`^\s*[✕×] (.+?)(?:\s+\(?\d+(?:\.\d+)?\s?m?s\)?)?\s*$`), // jest / vitest
	}
)

// perturbProbe is one probe's record: everything the transcript binds.
type perturbProbe struct {
	patchFiles              string
	copyStart, copyRestored string
	red, green              *acceptance.RunOutcome
	redMatch                string
	redFailed               []string
	greenMet                bool
	reason                  string // "" when verified
}

// RunCheckPerturb implements `check perturb`: an engine-run, candidate-bound
// safe-perturbation probe (testing.md § Safe perturbation).
func RunCheckPerturb(root string, args []string, stdout, stderr io.Writer) int {
	var patchArg, expect, gateID string
	var positional []string
	timeout := acceptance.DefaultTimeout
	for i := 0; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--patch":
			patchArg = value
		case "--expect":
			expect = strings.TrimSpace(value)
		case "--gate":
			gateID = value
		case "--timeout":
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 {
				fmt.Fprint(stderr, PerturbUsage)
				return 2
			}
			timeout = time.Duration(n) * time.Second
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "check perturb: unknown flag %q\n", args[i])
				return 2
			}
			positional = append(positional, args[i])
			continue
		}
		i++
	}
	if len(positional) != 1 || patchArg == "" || expect == "" || gateID == "" {
		fmt.Fprint(stderr, PerturbUsage)
		return 2
	}
	slug := positional[0]
	refuse := func(err error) int {
		fmt.Fprintf(stderr, "check perturb: refused: %v\n", err)
		return 3
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return refuse(err)
	}
	identity, err := loadCandidateIdentity(root, slug)
	if err != nil {
		return refuse(err)
	}
	gate, err := perturbGate(root, slug, gateID)
	if err != nil {
		return refuse(err)
	}
	project := projectDir(root)
	patchPath := patchArg
	if !filepath.IsAbs(patchPath) {
		patchPath = filepath.Join(project, filepath.FromSlash(patchPath))
	}
	patch, err := readBoundedRegularFile(patchPath, maxCandidateArtifactBytes)
	if err != nil {
		return refuse(fmt.Errorf("--patch %s: %w", patchArg, err))
	}
	if len(bytes.TrimSpace(patch)) == 0 {
		return refuse(fmt.Errorf("--patch %s is empty", patchArg))
	}

	probe := runPerturbProbe(project, identity, gate, patch, expect, timeout)
	after := "unavailable"
	if current, err := loadCandidateIdentity(root, slug); err != nil {
		after += ": " + oneLine(err.Error())
	} else {
		after = current.digest
	}
	if probe.reason == "" && after != identity.digest {
		probe.reason = "the checkout candidate changed during the probe"
	}
	transcript, err := writePerturbTranscript(workspace, gate, expect, patchArg, patch, identity.digest, after, probe)
	if err != nil {
		fmt.Fprintf(stderr, "check perturb: transcript: %v\n", err)
		return 3
	}
	rel := workspaceRel(root, filepath.Dir(transcript)) + "/" + filepath.Base(transcript)
	if probe.reason != "" {
		fmt.Fprintf(stdout, "check perturb: cannot_verify: %s\ntranscript: %s\n", probe.reason, rel)
		return 3
	}
	fmt.Fprintf(stdout, "check perturb: verified: %s RED named %s, restored copy GREEN, candidate %s unchanged\ntranscript: %s\n", gate.ID, expect, identity.digest, rel)
	return 0
}

// perturbGate returns the live runnable gate id, refusing a CHECK/CWD that is
// not an approved test-plan.md preflight row: a probe never runs unvetted code.
func perturbGate(root, slug, id string) (*acceptance.Gate, error) {
	raw, err := readWorkspaceArtifact(root, slug, acceptance.GatesFile)
	if err != nil {
		return nil, err
	}
	doc := acceptance.ParseLedger(string(raw))
	if len(doc.Errors) > 0 {
		return nil, fmt.Errorf("%s: %s", acceptance.GatesFile, doc.Errors[0])
	}
	for _, gate := range doc.Gates {
		if gate.ID != id {
			continue
		}
		if _, abandoned := doc.Abandoned[id]; abandoned {
			return nil, fmt.Errorf("gate %s is abandoned", id)
		}
		if gate.Check == "" {
			return nil, fmt.Errorf("gate %s is a manual gate with no CHECK", id)
		}
		plan, _ := readWorkspaceArtifact(root, slug, "test-plan.md") // unreadable approves nothing
		if !acceptance.Approved(gate.Check, gate.Cwd, acceptance.ApprovedCommands(plan)) {
			return nil, fmt.Errorf("gate %s CHECK/CWD is not an approved test-plan.md preflight row", id)
		}
		return gate, nil
	}
	return nil, fmt.Errorf("%s has no gate %s", acceptance.GatesFile, id)
}

// runPerturbProbe builds the isolated candidate copy, runs the mutant (RED)
// and restored (GREEN) gate there, and removes the copy. The checkout is only
// read.
func runPerturbProbe(project string, identity candidateIdentity, gate *acceptance.Gate, patch []byte, expect string, timeout time.Duration) perturbProbe {
	var probe perturbProbe
	tmp, err := os.MkdirTemp("", "devrites-perturb-")
	if err != nil {
		probe.reason = "isolation unavailable: " + oneLine(err.Error())
		return probe
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	dir := filepath.Join(tmp, "copy")
	if err := parallel.AddDetachedWorktree(project, dir, "HEAD"); err != nil {
		probe.reason = "isolation unavailable: " + oneLine(err.Error())
		return probe
	}
	defer func() { _ = parallel.RemoveWorktree(project, dir) }()
	// Manifest rows and gate CWD are project-relative; the project may sit
	// below the git toplevel. Patches stay toplevel-relative, so git apply runs at dir.
	prefix, err := runGitCommand(project, nil, "rev-parse", "--show-prefix")
	if err != nil {
		probe.reason = "isolation unavailable: " + oneLine(err.Error())
		return probe
	}
	base := filepath.Join(dir, filepath.FromSlash(strings.TrimSpace(string(prefix))))
	rows, err := overlayCandidate(project, base, identity.rows)
	if err != nil {
		probe.reason = "candidate copy failed: " + oneLine(err.Error())
		return probe
	}
	if probe.copyStart, err = hashCandidate(base, rows); err != nil || probe.copyStart != identity.digest {
		probe.reason = "the isolated copy does not hash to the candidate digest"
		return probe
	}
	if stat, err := runGitCommandIO(dir, nil, patch, nil, "apply", "--numstat", "-"); err == nil {
		probe.patchFiles = perturbPatchFiles(stat)
	}
	if _, err := runGitCommandIO(dir, nil, patch, nil, "apply", "-"); err != nil {
		probe.reason = "the mutant patch does not apply to the candidate copy: " + oneLine(err.Error())
		return probe
	}
	cwd := filepath.Join(base, filepath.FromSlash(gate.Cwd))
	red := acceptance.Execute(context.Background(), gate.Check, cwd, timeout)
	probe.red = &red
	probe.redMatch, probe.redFailed, probe.reason = classifyPerturbRed(red, expect)
	if probe.reason != "" {
		return probe
	}
	if _, err := runGitCommandIO(dir, nil, patch, nil, "apply", "-R", "-"); err != nil {
		probe.reason = "the mutant patch does not revert: " + oneLine(err.Error())
		return probe
	}
	if err := validateCandidateRows(base, rows); err == nil {
		probe.copyRestored, _ = hashCandidate(base, rows)
	}
	if probe.copyRestored != identity.digest {
		probe.reason = "the reverted copy does not hash to the candidate digest"
		return probe
	}
	green := acceptance.Execute(context.Background(), gate.Check, cwd, timeout)
	probe.green = &green
	expectation, err := acceptance.CompileExpect(gate.Expect)
	probe.greenMet = err == nil && green.Err == nil && green.ExitCode == 0 && expectation.Matches(green.Output)
	if !probe.greenMet {
		probe.reason = "the restored copy is not GREEN (exit 0 plus EXPECT)"
	}
	return probe
}

// overlayCandidate writes every manifest row onto the HEAD worktree at dir
// (copying present files, deleting deleted ones) and returns rows validated
// against the copy, ready for hashCandidate.
func overlayCandidate(project, dir string, rows []candidateRow) ([]candidateRow, error) {
	copied := make([]candidateRow, len(rows))
	for i, row := range rows {
		copied[i] = candidateRow{state: row.state, path: row.path, slice: row.slice, reason: row.reason}
		// A symlinked parent in the HEAD tree would redirect the write out of the copy.
		if _, _, err := candidatePathInfo(dir, row.path); err != nil {
			return nil, fmt.Errorf("%s: %w", row.path, err)
		}
		target := filepath.Join(dir, filepath.FromSlash(row.path))
		if row.state == "deleted" {
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			continue
		}
		content, err := readBoundedRegularFile(filepath.Join(project, filepath.FromSlash(row.path)), maxCandidateFileBytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", row.path, err)
		}
		mode := os.FileMode(0o644)
		if row.executable {
			mode = 0o755
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, content, mode); err != nil {
			return nil, err
		}
		if err := os.Chmod(target, mode); err != nil { // WriteFile keeps an existing file's mode
			return nil, err
		}
	}
	if err := validateCandidateRows(dir, copied); err != nil {
		return nil, err
	}
	return copied, nil
}

// classifyPerturbRed accepts the mutant run as RED only when a runner FAIL
// line names expect; otherwise reason says why it is cannot_verify.
func classifyPerturbRed(outcome acceptance.RunOutcome, expect string) (match string, failed []string, reason string) {
	if outcome.Err != nil {
		return "", nil, "the mutant run did not complete: " + oneLine(outcome.Err.Error())
	}
	if outcome.ExitCode == 0 {
		return "", nil, "the mutant survived: the gate exited 0"
	}
	for _, line := range strings.Split(outcome.Output, "\n") {
		line = strings.TrimRight(line, "\r")
		for _, re := range perturbFailREs {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			failed = append(failed, m[1])
			if match == "" && perturbNames(m[1], expect) {
				match = strings.TrimSpace(line)
			}
			break
		}
	}
	switch {
	case match != "":
		return match, failed, ""
	case len(failed) == 0:
		return "", nil, fmt.Sprintf("exit %d with no runner FAIL line naming a test (build, setup, or unrecognized runner failure)", outcome.ExitCode)
	}
	return "", failed, fmt.Sprintf("the RED names other tests (%s), not %s", strings.Join(failed, ", "), expect)
}

// perturbNames reports whether a runner's failed-test name is expect: exact,
// or its last pytest/cargo `::` or vitest ` > ` segment.
func perturbNames(name, expect string) bool {
	return name == expect || strings.HasSuffix(name, "::"+expect) || strings.HasSuffix(name, " > "+expect)
}

func perturbPatchFiles(numstat []byte) string {
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(numstat)), "\n") {
		if fields := strings.SplitN(line, "\t", 3); len(fields) == 3 {
			files = append(files, fields[2])
		}
	}
	return strings.Join(files, ", ")
}

// writePerturbTranscript records one immutable probe file under
// <work>/evidence/perturb/, binding the patch, command, both runs, the copy
// digests, and the checkout candidate before and after.
func writePerturbTranscript(workspace string, gate *acceptance.Gate, expect, patchArg string, patch []byte, before, after string, probe perturbProbe) (string, error) {
	dir := filepath.Join(workspace, "evidence", "perturb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	prefix := perturbNameRE.ReplaceAllString(gate.ID, "_") + "-" + before[:12]
	attempt := 1
	if entries, err := os.ReadDir(dir); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), prefix+"-a") {
				attempt++
			}
		}
	}
	verdict, reason := "verified", "none"
	if probe.reason != "" {
		verdict, reason = "cannot_verify", probe.reason
	}
	var body strings.Builder
	fmt.Fprintf(&body, "DEVRITES-PERTURB-TRANSCRIPT v1\nGATE: %s\nCOMMAND: %s\nCWD: %s\nEXPECT_TEST: %s\n", gate.ID, gate.Check, cmp.Or(gate.Cwd, "."), expect)
	fmt.Fprintf(&body, "PATCH: %s\nPATCH_SHA256: %x\nPATCH_FILES: %s\n", patchArg, sha256.Sum256(patch), cmp.Or(probe.patchFiles, "unknown"))
	fmt.Fprintf(&body, "CANDIDATE_BEFORE: %s\nCOPY_START: %s\nCOPY_RESTORED: %s\nCANDIDATE_AFTER: %s\n", before, cmp.Or(probe.copyStart, "none"), cmp.Or(probe.copyRestored, "none"), after)
	fmt.Fprintf(&body, "RED_EXIT: %s\nRED_MATCH: %s\nRED_FAILED: %s\n", perturbExit(probe.red), cmp.Or(probe.redMatch, "none"), cmp.Or(strings.Join(probe.redFailed, ", "), "none"))
	fmt.Fprintf(&body, "GREEN_EXIT: %s\nGREEN_EXPECT: %s\nGREEN_MET: %t\nVERDICT: %s\nREASON: %s\n", perturbExit(probe.green), gate.Expect, probe.greenMet, verdict, reason)
	redacted, _ := secretpattern.Redact(string(patch))
	fmt.Fprintf(&body, "--- PATCH ---\n%s\n--- END PATCH ---\n", strings.TrimRight(redacted, "\n"))
	for _, run := range []struct {
		name    string
		outcome *acceptance.RunOutcome
	}{{"RED", probe.red}, {"GREEN", probe.green}} {
		if run.outcome == nil {
			continue
		}
		output, _ := secretpattern.Redact(run.outcome.Output)
		if len(output) > perturbOutputTail {
			output = output[len(output)-perturbOutputTail:]
		}
		fmt.Fprintf(&body, "--- %s OUTPUT (tail) ---\n%s\n--- END %s OUTPUT ---\n", run.name, output, run.name)
	}
	file := filepath.Join(dir, fmt.Sprintf("%s-a%d.log", prefix, attempt))
	if err := os.WriteFile(file, []byte(body.String()), 0o644); err != nil {
		return "", err
	}
	return file, nil
}

func perturbExit(outcome *acceptance.RunOutcome) string {
	switch {
	case outcome == nil:
		return "not run"
	case outcome.TimedOut:
		return "timeout"
	case outcome.Err != nil:
		return "error: " + oneLine(outcome.Err.Error())
	}
	return strconv.Itoa(outcome.ExitCode)
}
