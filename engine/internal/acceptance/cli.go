package acceptance

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// Exit codes match the engine CLI conventions.
const (
	ExitOK      = 0
	ExitUsage   = 2
	ExitBlocked = 3
)

const gatesUsage = `usage: devrites-engine gates <subcommand> <slug>

Subcommands:
  scaffold <slug>            Create gates.md from spec.md AC ids (only when absent)
  status <slug>              Reduce the ledger without executing; exit 3 unless ALL MET
  run <slug> [--timeout N]   Execute unmet runnable gates and record evidence
  reverify <slug> [--timeout N]  Re-execute every runnable gate, demoting stale passes
  lint <slug> [--strict]     Audit oracle quality without executing
  attest <slug> <id> <note>  Record human evidence on a manual gate
  abandon <slug> <id> <why>  Record a terminal ABANDON handoff on a gate

Exit codes: 0 ok, 2 usage, 3 blocked`

// Run is the engine entrypoint for `gates …`; root is the resolved DevRites
// root from the caller's single root resolution.
func Run(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, gatesUsage)
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "-h", "-help", "--help", "help":
		fmt.Fprintln(stdout, gatesUsage)
		return ExitOK
	case "scaffold":
		return cmdScaffold(root, rest, stdout, stderr)
	case "status":
		return cmdStatus(root, rest, stdout, stderr)
	case "run":
		return cmdRun(root, rest, false, stdout, stderr)
	case "reverify":
		return cmdRun(root, rest, true, stdout, stderr)
	case "lint":
		return cmdLint(root, rest, stdout, stderr)
	case "attest":
		return cmdAttest(root, rest, stdout, stderr)
	case "abandon":
		return cmdAbandon(root, rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "gates: unknown subcommand %q\n\n%s\n", sub, gatesUsage)
		return ExitUsage
	}
}

func featurePaths(root, slug string) (work, ledger, testPlan, project string, err error) {
	work, err = devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", "", "", "", fmt.Errorf("feature %q: %w", slug, err)
	}
	return work,
		filepath.Join(work, GatesFile),
		filepath.Join(work, "test-plan.md"),
		filepath.Dir(root),
		nil
}

// loadTestPlan returns the vetted command surface; an unreadable plan approves
// nothing, so runnable gates stay unapproved rather than running unvetted code.
func loadTestPlan(testPlanPath string) []ApprovedCommand {
	// #nosec G304 -- workspace artifact path resolved from the operator root
	raw, err := os.ReadFile(testPlanPath)
	if err != nil {
		return nil
	}
	return ApprovedCommands(raw)
}

func parseSlug(args []string) (string, bool) {
	return args[0], len(args) == 1 && args[0] != "" && !strings.HasPrefix(args[0], "-")
}

// oneLine flattens an argument to a single ledger line so a pasted newline
// cannot inject rows into the artifact.
func oneLine(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' }), " ")
}

func cmdScaffold(root string, args []string, stdout, stderr io.Writer) int {
	slug, ok := parseSlug(args)
	if !ok {
		fmt.Fprintln(stderr, "usage: devrites-engine gates scaffold <slug>")
		return ExitUsage
	}
	work, ledgerPath, _, _, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates scaffold: %v\n", err)
		return ExitBlocked
	}
	var wrote bool
	lockErr := state.WithFeatureLock(root, slug, func() error {
		if _, statErr := os.Lstat(ledgerPath); statErr == nil {
			return fmt.Errorf("%s already exists; scaffold never clobbers", GatesFile)
		} else if !os.IsNotExist(statErr) {
			return statErr
		}
		// #nosec G304 -- workspace artifact path resolved from the operator root
		spec, _ := os.ReadFile(filepath.Join(work, "spec.md"))
		if err := state.AtomicWrite(ledgerPath, []byte(ScaffoldLedger(slug, spec)), 0o644); err != nil {
			return err
		}
		wrote = true
		return nil
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "gates scaffold: %v\n", lockErr)
		return ExitBlocked
	}
	if wrote {
		fmt.Fprintf(stdout, "created: %s\n", ledgerPath)
	}
	return ExitOK
}

func cmdStatus(root string, args []string, stdout, stderr io.Writer) int {
	slug, ok := parseSlug(args)
	if !ok {
		fmt.Fprintln(stderr, "usage: devrites-engine gates status <slug>")
		return ExitUsage
	}
	_, ledgerPath, testPlanPath, _, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates status: %v\n", err)
		return ExitBlocked
	}
	return reportStatus(slug, ledgerPath, testPlanPath, stdout, stderr)
}

// reportStatus prints the stable greppable reduction. It returns ExitOK only
// for a parse-clean ledger with every gate met and none abandoned.
func reportStatus(slug, ledgerPath, testPlanPath string, stdout, stderr io.Writer) int {
	text, err := ReadLedgerFile(ledgerPath)
	if err != nil {
		fmt.Fprintf(stderr, "gates status: %v\n", err)
		return ExitBlocked
	}
	doc := ParseLedger(text)
	approved := loadTestPlan(testPlanPath)
	fmt.Fprintf(stdout, "feature: %s\n", slug)
	fmt.Fprintf(stdout, "ledger: %s\n", GatesFile)
	for _, parseErr := range doc.Errors {
		fmt.Fprintf(stdout, "error: %s\n", parseErr)
	}
	reduction := doc.Reduce(approved)
	fmt.Fprintf(stdout, "gates: %d\nmet: %d\nunmet: %d\nstale: %d\nabandoned: %d\nunapproved: %d\n",
		reduction.Total, reduction.Met, reduction.Unmet, reduction.Stale, reduction.Abandoned, reduction.Unapproved)
	for _, line := range reductionLines(reduction) {
		fmt.Fprintln(stdout, line)
	}
	switch {
	case len(doc.Errors) > 0:
		fmt.Fprintln(stdout, "result: malformed")
		return ExitBlocked
	case reduction.HandoffRequired():
		fmt.Fprintln(stdout, "result: handoff")
		return ExitBlocked
	case !reduction.AllMet():
		fmt.Fprintln(stdout, "result: not-met")
		return ExitBlocked
	default:
		fmt.Fprintln(stdout, "result: all-met")
		return ExitOK
	}
}

// reductionLines names the specific unmet/stale/unapproved/abandoned ids so a
// report surfaces the exact compliance gap instead of a bare count.
func reductionLines(r Reduction) []string {
	var lines []string
	for _, id := range r.UnmetIDs {
		lines = append(lines, "unmet-id: "+id)
	}
	for _, id := range r.StaleIDs {
		lines = append(lines, "stale-id: "+id)
	}
	for _, id := range r.UnapprIDs {
		lines = append(lines, "unapproved-id: "+id)
	}
	for _, id := range r.Handoffs {
		lines = append(lines, "handoff-id: "+id)
	}
	return lines
}

func cmdRun(root string, args []string, reverify bool, stdout, stderr io.Writer) int {
	verb := "run"
	if reverify {
		verb = "reverify"
	}
	var slug string
	timeout := DefaultTimeout
	positional := 0
	for i := 0; i < len(args); i++ {
		if args[i] == "--timeout" {
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "gates %s: --timeout requires seconds\n", verb)
				return ExitUsage
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				fmt.Fprintf(stderr, "gates %s: --timeout must be positive seconds, got %q\n", verb, args[i])
				return ExitUsage
			}
			timeout = time.Duration(n) * time.Second
			continue
		}
		if strings.HasPrefix(args[i], "-") || positional > 0 {
			fmt.Fprintf(stderr, "usage: devrites-engine gates %s <slug> [--timeout N]\n", verb)
			return ExitUsage
		}
		slug = args[i]
		positional++
	}
	if slug == "" {
		fmt.Fprintf(stderr, "usage: devrites-engine gates %s <slug> [--timeout N]\n", verb)
		return ExitUsage
	}
	_, ledgerPath, testPlanPath, project, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates %s: %v\n", verb, err)
		return ExitBlocked
	}

	lockErr := state.WithFeatureLock(root, slug, func() error {
		text, err := ReadLedgerFile(ledgerPath)
		if err != nil {
			return err
		}
		doc := ParseLedger(text)
		if len(doc.Errors) > 0 {
			for _, parseErr := range doc.Errors {
				fmt.Fprintf(stdout, "error: %s\n", parseErr)
			}
			return nil
		}
		approved := loadTestPlan(testPlanPath)
		for _, result := range RunGates(context.Background(), project, ledgerPath, doc, approved, reverify, timeout) {
			switch {
			case !result.Ran:
				fmt.Fprintf(stdout, "gate %s: skipped (%s)\n", result.ID, result.Detail)
			case result.Passed && !result.Discarded && result.WriteErr == nil:
				fmt.Fprintf(stdout, "gate %s: pass\n", result.ID)
			default:
				fmt.Fprintf(stdout, "gate %s: fail (%s)\n", result.ID, result.Detail)
			}
		}
		return nil
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "gates %s: %v\n", verb, lockErr)
		return ExitBlocked
	}
	fmt.Fprintln(stdout, "---")
	return reportStatus(slug, ledgerPath, testPlanPath, stdout, stderr)
}

func cmdLint(root string, args []string, stdout, stderr io.Writer) int {
	strict := false
	var slug string
	for _, arg := range args {
		if arg == "--strict" {
			strict = true
			continue
		}
		if strings.HasPrefix(arg, "-") || slug != "" {
			fmt.Fprintln(stderr, "usage: devrites-engine gates lint <slug> [--strict]")
			return ExitUsage
		}
		slug = arg
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine gates lint <slug> [--strict]")
		return ExitUsage
	}
	_, ledgerPath, testPlanPath, _, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates lint: %v\n", err)
		return ExitBlocked
	}
	text, err := ReadLedgerFile(ledgerPath)
	if err != nil {
		fmt.Fprintf(stderr, "gates lint: %v\n", err)
		return ExitBlocked
	}
	doc := ParseLedger(text)
	findings := LintLedger(doc, loadTestPlan(testPlanPath))
	for _, finding := range findings {
		gate := finding.Gate
		if gate == "" {
			gate = "ledger"
		}
		fmt.Fprintf(stdout, "%s: %s: %s: %s\n", finding.Level, gate, finding.Rule, finding.Message)
	}
	errors, warnings := LintCounts(findings)
	fmt.Fprintf(stdout, "lint: %d error(s), %d warning(s)\n", errors, warnings)
	if errors > 0 || (strict && warnings > 0) {
		fmt.Fprintln(stdout, "LINT FAIL")
		return ExitBlocked
	}
	fmt.Fprintln(stdout, "LINT OK")
	return ExitOK
}

func cmdAttest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 {
		fmt.Fprintln(stderr, "usage: devrites-engine gates attest <slug> <gate-id> <evidence note>")
		return ExitUsage
	}
	slug, gateID := args[0], args[1]
	note := oneLine(strings.Join(args[2:], " "))
	if note == "" {
		fmt.Fprintln(stderr, "gates attest: evidence note must be non-blank")
		return ExitUsage
	}
	_, ledgerPath, _, _, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates attest: %v\n", err)
		return ExitBlocked
	}
	lockErr := state.WithFeatureLock(root, slug, func() error {
		text, err := ReadLedgerFile(ledgerPath)
		if err != nil {
			return err
		}
		doc := ParseLedger(text)
		var gate *Gate
		for _, candidate := range doc.Gates {
			if candidate.ID == gateID {
				gate = candidate
			}
		}
		if gate == nil {
			return fmt.Errorf("unknown gate id %q", gateID)
		}
		if _, abandoned := doc.Abandoned[gateID]; abandoned {
			return fmt.Errorf("gate %s is abandoned; a handoff cannot take evidence", gateID)
		}
		if gate.Check != "" {
			return fmt.Errorf("gate %s is runnable; evidence comes from `gates run`, not attestation", gateID)
		}
		return writeResult(ledgerPath, gate, note, true)
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "gates attest: %v\n", lockErr)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "attested: %s\n", gateID)
	return ExitOK
}

func cmdAbandon(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 {
		fmt.Fprintln(stderr, "usage: devrites-engine gates abandon <slug> <gate-id> <reason>")
		return ExitUsage
	}
	slug, gateID := args[0], args[1]
	reason := oneLine(strings.Join(args[2:], " "))
	if reason == "" {
		fmt.Fprintln(stderr, "gates abandon: reason must be non-blank")
		return ExitUsage
	}
	_, ledgerPath, _, _, err := featurePaths(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "gates abandon: %v\n", err)
		return ExitBlocked
	}
	lockErr := state.WithFeatureLock(root, slug, func() error {
		text, err := ReadLedgerFile(ledgerPath)
		if err != nil {
			return err
		}
		doc := ParseLedger(text)
		found := false
		for _, gate := range doc.Gates {
			if gate.ID == gateID {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unknown gate id %q", gateID)
		}
		if _, already := doc.Abandoned[gateID]; already {
			return fmt.Errorf("gate %s is already abandoned", gateID)
		}
		line := "ABANDON: " + gateID + " " + reason
		switch {
		case strings.HasSuffix(text, "\r\n"):
			text += line + "\r\n"
		case strings.HasSuffix(text, "\n"):
			text += line + "\n"
		default:
			text += "\n" + line + "\n"
		}
		return state.AtomicWrite(ledgerPath, []byte(text), 0o644)
	})
	if lockErr != nil {
		fmt.Fprintf(stderr, "gates abandon: %v\n", lockErr)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "abandoned: %s\n", gateID)
	return ExitOK
}
