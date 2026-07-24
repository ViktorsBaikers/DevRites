package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
)

const (
	recoveryAttemptsFile = "recovery-attempts.jsonl"
	recoveryAttemptLimit = 3
	recoveryRouteSchema  = "recovery-route/v1"
)

type recoveryClass string
type recoveryOwner string
type recoveryAction string

const (
	recoveryIntentGap            recoveryClass = "intent_gap"
	recoverySpecGap              recoveryClass = "spec_gap"
	recoveryPlanGap              recoveryClass = "plan_gap"
	recoveryImplementationDefect recoveryClass = "implementation_defect"
	recoveryProofToolDefect      recoveryClass = "proof_tool_defect"
	recoveryEnvironmentDefect    recoveryClass = "environment_defect"
	recoveryPreexisting          recoveryClass = "preexisting"
	recoveryNotADefect           recoveryClass = "not_a_defect"
)

type recoveryRoute struct {
	Schema     string         `json:"schema"`
	Class      recoveryClass  `json:"class"`
	Owner      recoveryOwner  `json:"owner"`
	Action     recoveryAction `json:"action"`
	HumanPause bool           `json:"humanPause"`
}

var recoveryRoutes = [...]recoveryRoute{
	{recoveryRouteSchema, recoveryIntentGap, "human_clarify", "clarify_intent", true},
	{recoveryRouteSchema, recoverySpecGap, "clarify", "clarify_missing_decision", true},
	{recoveryRouteSchema, recoveryPlanGap, "plan", "rite_plan_repair", false},
	{recoveryRouteSchema, recoveryImplementationDefect, "wright", "repair_implementation_and_rerun_proof", false},
	{recoveryRouteSchema, recoveryProofToolDefect, "debug_recovery", "repair_proof_tool_and_rerun_original_proof", false},
	{recoveryRouteSchema, recoveryEnvironmentDefect, "debug_recovery", "normalize_environment_and_run_discriminator", false},
	{recoveryRouteSchema, recoveryPreexisting, "baseline", "record_baseline_and_fix_if_acceptance_blocked", false},
	{recoveryRouteSchema, recoveryNotADefect, "caller", "record_authority_and_continue", false},
}

type recoveryAttempt struct {
	Fingerprint string        `json:"fingerprint"`
	RootCause   string        `json:"rootCause"`
	Class       recoveryClass `json:"class,omitempty"`
	Attempt     int           `json:"attempt"`
	Status      string        `json:"status"`
	Failure     string        `json:"failure,omitempty"`
	At          string        `json:"at"`
}

var recoveryWhitespace = regexp.MustCompile(`\s+`)

// RecoveryAttempts durably accounts bounded technical recovery across agents
// and sessions.
//
//	recovery route <class>
//	recovery record [--class <class>] "<root cause>" "<failure>" [slug]
//	recovery check "<root cause>" [slug]
//	recovery clear [--class <class>] "<root cause>" [slug]
//	recovery fingerprint "<root cause>"
func RecoveryAttempts(root string, args []string, stdout, stderr io.Writer) int {
	sub := strings.ToLower(strings.TrimSpace(argAt(args, 0)))
	if sub == "route" {
		return writeRecoveryRoute(args[1:], stdout, stderr)
	}

	commandArgs := args[1:]
	var class recoveryClass
	if sub == "record" || sub == "clear" {
		var err error
		class, commandArgs, err = parseRecoveryClass(commandArgs)
		if err != nil {
			fmt.Fprintf(stderr, "recovery: %v\n", err)
			return 2
		}
	}
	rootCause := strings.TrimSpace(argAt(commandArgs, 0))
	if sub == "fingerprint" {
		if rootCause == "" || len(commandArgs) != 1 {
			fmt.Fprintln(stderr, `usage: devrites-engine recovery fingerprint "<root cause>"`)
			return 2
		}
		fmt.Fprintln(stdout, recoveryFingerprint(rootCause))
		return 0
	}
	if sub != "record" && sub != "check" && sub != "clear" {
		fmt.Fprintln(stderr, `usage: devrites-engine recovery <route|record|check|clear|fingerprint> ...`)
		return 2
	}
	if rootCause == "" {
		fmt.Fprintln(stderr, "recovery: root cause is required")
		return 2
	}

	var failure string
	var slugArgs []string
	if sub == "record" {
		failure = strings.TrimSpace(argAt(commandArgs, 1))
		if failure == "" {
			fmt.Fprintln(stderr, "recovery: record requires a concise failure")
			return 2
		}
		slugArgs = commandArgs[2:]
	} else {
		slugArgs = commandArgs[1:]
	}
	if len(slugArgs) > 1 {
		fmt.Fprintln(stderr, "recovery: at most one slug is allowed")
		return 2
	}
	slug := slugOrActive(root, slugArgs)
	if slug == "" {
		fmt.Fprintln(stderr, "recovery: no active workspace")
		return 2
	}
	path := featureFile(root, slug, recoveryAttemptsFile)
	entries, err := readRecoveryAttempts(path)
	if err != nil {
		fmt.Fprintf(stderr, "recovery: %v\n", err)
		return 2
	}
	fingerprint := recoveryFingerprint(rootCause)
	attempts := activeRecoveryAttempts(entries, fingerprint)

	switch sub {
	case "check":
		if attempts >= recoveryAttemptLimit {
			fmt.Fprintf(stderr, "recovery: exhausted %d/%d for %s\n", attempts, recoveryAttemptLimit, fingerprint)
			return 3
		}
		fmt.Fprintf(stdout, "recovery: available %d/%d used for %s\n", attempts, recoveryAttemptLimit, fingerprint)
		return 0
	case "record":
		if attempts >= recoveryAttemptLimit {
			fmt.Fprintf(stderr, "recovery: exhausted %d/%d for %s; no new attempt recorded\n", attempts, recoveryAttemptLimit, fingerprint)
			return 3
		}
		attempts++
		entries = append(entries, recoveryAttempt{
			Fingerprint: fingerprint,
			RootCause:   normalizeRecoveryCause(rootCause),
			Class:       class,
			Attempt:     attempts,
			Status:      "failed",
			Failure:     failure,
			At:          nowUTC(),
		})
		if err := writeRecoveryAttempts(path, entries); err != nil {
			fmt.Fprintf(stderr, "recovery: persist attempt: %v\n", err)
			return 1
		}
		if attempts >= recoveryAttemptLimit {
			fmt.Fprintf(stderr, "recovery: recorded attempt %d/%d; automatic recovery exhausted for %s\n", attempts, recoveryAttemptLimit, fingerprint)
			return 3
		}
		fmt.Fprintf(stdout, "recovery: recorded attempt %d/%d for %s\n", attempts, recoveryAttemptLimit, fingerprint)
		return 0
	case "clear":
		entries = append(entries, recoveryAttempt{
			Fingerprint: fingerprint,
			RootCause:   normalizeRecoveryCause(rootCause),
			Class:       class,
			Status:      "cleared",
			At:          nowUTC(),
		})
		if err := writeRecoveryAttempts(path, entries); err != nil {
			fmt.Fprintf(stderr, "recovery: clear attempts: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "recovery: cleared attempts for %s\n", fingerprint)
		return 0
	}
	return 2
}

func writeRecoveryRoute(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine recovery route <class>")
		return 2
	}
	route, ok := recoveryRouteFor(recoveryClass(strings.TrimSpace(args[0])))
	if !ok {
		fmt.Fprintf(stderr, "recovery: unknown class %q\n", args[0])
		return 2
	}
	if err := json.NewEncoder(stdout).Encode(route); err != nil {
		fmt.Fprintf(stderr, "recovery: encode route: %v\n", err)
		return 1
	}
	return 0
}

func recoveryRouteFor(class recoveryClass) (recoveryRoute, bool) {
	for _, route := range recoveryRoutes {
		if route.Class == class {
			return route, true
		}
	}
	return recoveryRoute{}, false
}

func parseRecoveryClass(args []string) (recoveryClass, []string, error) {
	if argAt(args, 0) != "--class" {
		return "", args, nil
	}
	if len(args) < 2 {
		return "", nil, fmt.Errorf("--class requires a value")
	}
	class := recoveryClass(strings.TrimSpace(args[1]))
	if _, ok := recoveryRouteFor(class); !ok {
		return "", nil, fmt.Errorf("unknown class %q", args[1])
	}
	return class, args[2:], nil
}

func recoveryFingerprint(rootCause string) string {
	sum := sha256.Sum256([]byte(normalizeRecoveryCause(rootCause)))
	return hex.EncodeToString(sum[:])
}

func normalizeRecoveryCause(rootCause string) string {
	return strings.ToLower(recoveryWhitespace.ReplaceAllString(strings.TrimSpace(rootCause), " "))
}

func readRecoveryAttempts(path string) ([]recoveryAttempt, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", recoveryAttemptsFile, err)
	}
	var entries []recoveryAttempt
	for index, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry recoveryAttempt
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("%s line %d is invalid: %w", recoveryAttemptsFile, index+1, err)
		}
		if entry.Fingerprint == "" || entry.Status == "" {
			return nil, fmt.Errorf("%s line %d is incomplete", recoveryAttemptsFile, index+1)
		}
		if entry.Class != "" {
			if _, ok := recoveryRouteFor(entry.Class); !ok {
				return nil, fmt.Errorf("%s line %d has unknown class %q", recoveryAttemptsFile, index+1, entry.Class)
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func activeRecoveryAttempts(entries []recoveryAttempt, fingerprint string) int {
	attempts := 0
	for _, entry := range entries {
		if entry.Fingerprint != fingerprint {
			continue
		}
		if entry.Status == "cleared" {
			attempts = 0
			continue
		}
		if entry.Status == "failed" && entry.Attempt > attempts {
			attempts = entry.Attempt
		}
	}
	return attempts
}

func writeRecoveryAttempts(path string, entries []recoveryAttempt) error {
	var out strings.Builder
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		out.Write(data)
		out.WriteByte('\n')
	}
	return fsutil.WriteFileAtomic(path, []byte(out.String()), 0o600)
}
