package acceptance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

const preconditionsUsage = "usage: devrites-engine check preconditions <slug> [--slice SLICE-###] [--timeout N]"

var preDispatchRunAtRE = regexp.MustCompile(`(?i)^pre-dispatch SLICE-\d+$`)

// PreflightGateID names a preflight row's transcript directory. The command
// and cwd digest keeps two rows sharing a Gate cell apart and retires a
// transcript the moment its row's command changes.
func PreflightGateID(row PreflightRow) string {
	sum := sha256.Sum256([]byte(row.Command + "\x00" + row.Cwd))
	return "preflight-" + valueOr(row.Gate, "row") + "-" + hex.EncodeToString(sum[:4])
}

// PreflightRowsPassed reports whether every row has a passing transcript in
// the feature workspace (see PreflightPassed).
func PreflightRowsPassed(root, slug string, rows []PreflightRow) (bool, error) {
	work, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if !PreflightPassed(work, row) {
			return false, nil
		}
	}
	return true, nil
}

// PreflightPassed reports whether the row's newest transcript under
// <work>/references/gates/ records RESULT: pass. The transcript records its
// candidate but the lookup does not compare it: Vet writes workspace artifacts
// after the run, and only a change to the row's command or cwd retires a pass.
func PreflightPassed(work string, row PreflightRow) bool {
	dir := filepath.Join(work, "references", "gates", unsafeNameRE.ReplaceAllString(PreflightGateID(row), "_"))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	newest, newestTime := "", time.Time{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		if newest == "" || info.ModTime().After(newestTime) {
			newest, newestTime = entry.Name(), info.ModTime()
		}
	}
	if newest == "" {
		return false
	}
	// #nosec G304 -- transcript path built from the workspace and a sanitized row id
	body, err := os.ReadFile(filepath.Join(dir, newest))
	// Only the header counts: command output could print the verdict line.
	header, _, _ := strings.Cut(string(body), "--- OUTPUT ---")
	return err == nil && strings.Contains(header, "\nRESULT: pass\n")
}

// RunPreconditions is `check preconditions <slug> [--slice S]`: it executes
// the Build-entry preflight rows whose `Run at` is `vet` (or, with --slice,
// `pre-dispatch <S>`), writes one candidate-bound transcript per row, and exits
// ExitBlocked on any failure. Rows are the approved surface itself, so nothing
// outside test-plan.md runs; `action` rows and rows without a `Run at` cell
// never run here.
func RunPreconditions(root string, args []string, stdout, stderr io.Writer) int {
	var slug, slice string
	timeout := DefaultTimeout
	for i := 0; i < len(args); i++ {
		switch {
		case (args[i] == "--slice" || args[i] == "--timeout") && i+1 < len(args):
			if args[i] == "--slice" {
				slice = args[i+1]
			} else if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				timeout = time.Duration(n) * time.Second
			} else {
				fmt.Fprintln(stderr, preconditionsUsage)
				return ExitUsage
			}
			i++
		case strings.HasPrefix(args[i], "-") || slug != "":
			fmt.Fprintln(stderr, preconditionsUsage)
			return ExitUsage
		default:
			slug = args[i]
		}
	}
	runAt := "vet"
	if slice != "" {
		runAt = "pre-dispatch " + slice
	}
	if slug == "" || (slice != "" && !preDispatchRunAtRE.MatchString(runAt)) {
		fmt.Fprintln(stderr, preconditionsUsage)
		return ExitUsage
	}
	work, ledgerPath, testPlanPath, project, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check preconditions: %v\n", err)
		return ExitBlocked
	}
	var plan []byte
	if err := state.WithFeatureLock(root, slug, func() error {
		var err error
		// #nosec G304 -- workspace artifact path resolved from the operator root
		plan, err = os.ReadFile(testPlanPath)
		return err
	}); err != nil {
		fmt.Fprintf(stderr, "check preconditions: %v\n", err)
		return ExitBlocked
	}
	rows, _ := PreflightRows(plan)
	fmt.Fprintf(stdout, "feature: %s\nrun-at: %s\n", slug, runAt)

	// The gates lock serializes execution and transcript attempt numbering
	// with `gates run` and any concurrent run of the same row.
	failed, ran := false, false
	lockErr := state.WithLock(ledgerPath+".lock", func() error {
		var err error
		failed, ran, err = runPreflightRows(root, slug, work, project, runAt, rows, timeout, stdout)
		return err
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "check preconditions: %v\n", lockErr)
		return ExitBlocked
	}
	switch {
	case failed:
		fmt.Fprintln(stdout, "result: fail")
		return ExitBlocked
	case !ran:
		fmt.Fprintln(stdout, "result: none")
	default:
		fmt.Fprintln(stdout, "result: pass")
	}
	return ExitOK
}

// runPreflightRows executes the rows whose Run at equals runAt and writes their
// transcripts. It reports whether any row failed or carried an invalid Run at,
// and whether any row ran.
func runPreflightRows(root, slug, work, project, runAt string, rows []PreflightRow, timeout time.Duration, stdout io.Writer) (bool, bool, error) {
	before := candidateLabel(root, slug)
	failed := false
	doc := &Document{}
	var results []GateResult
	for _, row := range rows {
		lower := strings.ToLower(row.RunAt)
		if lower != "" && lower != "vet" && lower != "action" && !preDispatchRunAtRE.MatchString(row.RunAt) {
			fmt.Fprintf(stdout, "row %s: invalid Run at %q (want vet, pre-dispatch SLICE-###, or action)\n", valueOr(row.Gate, row.Command), row.RunAt)
			failed = true
			continue
		}
		if !strings.EqualFold(row.RunAt, runAt) {
			continue
		}
		gate := &Gate{ID: PreflightGateID(row), Title: row.Gate, Check: row.Command, Cwd: row.Cwd,
			Expect: "exit 0 (Expected: " + valueOr(row.Expected, "unstated") + ")"}
		cwd := project
		if row.Cwd != "" {
			cwd = filepath.Join(project, filepath.FromSlash(row.Cwd))
		}
		started := time.Now().UTC()
		outcome := Execute(context.Background(), row.Command, cwd, timeout)
		passed := outcome.Err == nil && outcome.ExitCode == 0
		doc.Gates = append(doc.Gates, gate)
		results = append(results, GateResult{ID: gate.ID, Ran: true, Passed: passed, Outcome: outcome, Started: started, Ended: time.Now().UTC()})
		if passed {
			fmt.Fprintf(stdout, "row %s: pass\n", gate.ID)
		} else {
			fmt.Fprintf(stdout, "row %s: fail (%s)\n", gate.ID, exitLabel(outcome))
			failed = true
		}
	}
	if len(results) > 0 {
		paths, err := writeTranscripts(work, doc, results, before, candidateLabel(root, slug))
		for _, path := range paths {
			fmt.Fprintf(stdout, "transcript: %s\n", path)
		}
		if err != nil {
			return failed, true, fmt.Errorf("transcript: %w", err)
		}
	}
	return failed, len(results) > 0, nil
}
