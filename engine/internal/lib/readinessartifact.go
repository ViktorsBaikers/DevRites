package lib

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

type readinessArtifactContract struct {
	Artifact         string   `json:"artifact"`
	VerdictField     string   `json:"verdictField"`
	ReadyValue       string   `json:"readyValue"`
	DigestField      string   `json:"digestField"`
	Inputs           []string `json:"inputs"`
	RequiredHeadings []string `json:"requiredHeadings"`
	RequiredTables   []string `json:"requiredTables"`
}

type readinessContractFile struct {
	Schema      string                    `json:"schema"`
	Coverage    readinessArtifactContract `json:"coverage"`
	Engineering readinessArtifactContract `json:"engineering"`
	TestPlan    readinessArtifactContract `json:"testPlan"`
	Reasons     []readinessReason         `json:"reasons"`
}

type readinessReason struct {
	ID              string `json:"id"`
	Code            int    `json:"code"`
	Condition       string `json:"condition"`
	RemediationVerb string `json:"remediationVerb"`
}

//go:embed readiness_contract.json
var readinessContractJSON []byte

var readinessContract = func() readinessContractFile {
	var contract readinessContractFile
	if err := json.Unmarshal(readinessContractJSON, &contract); err != nil {
		panic(fmt.Sprintf("parse embedded readiness contract: %v", err))
	}
	if len(contract.Reasons) == 0 {
		panic("parse embedded readiness contract: no readiness reasons")
	}
	return contract
}()

func readinessCode(id string) int {
	for _, reason := range readinessContract.Reasons {
		if reason.ID == id {
			return reason.Code
		}
	}
	panic("readiness contract has no reason " + id)
}

var (
	readinessDigestRe      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	readinessPlaceholderRe = regexp.MustCompile(`(?i)<[^>\n]+>|\b(?:TODO|TBD|FIXME|UNKNOWN)\b`)
	acceptanceIDRe         = regexp.MustCompile(`(?i)\bAC-?[0-9]+\b`)
)

// ReadinessDigest renders the exact provenance field a clarification or vet
// artifact must record. Usage: readiness-digest <coverage|engineering> [slug].
func ReadinessDigest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: devrites-engine readiness-digest <coverage|engineering> [slug]")
		return 2
	}
	kind := strings.ToLower(strings.TrimSpace(args[0]))
	slugArgs := args[1:]
	slug := slugOrActive(root, slugArgs)
	if slug == "" {
		fmt.Fprintln(stderr, "readiness-digest: no active workspace")
		return 2
	}
	contract, ok := readinessContractFor(kind)
	if !ok {
		fmt.Fprintf(stderr, "readiness-digest: unknown kind %q (want coverage or engineering)\n", kind)
		return 2
	}
	digest, err := readinessInputsDigest(root, slug, contract.Inputs)
	if err != nil {
		fmt.Fprintf(stderr, "readiness-digest: %v\n", err)
		return 3
	}
	fmt.Fprintf(stdout, "%s: %s\n", contract.DigestField, digest)
	return 0
}

func readinessContractFor(kind string) (readinessArtifactContract, bool) {
	switch kind {
	case "coverage":
		return readinessContract.Coverage, true
	case "engineering":
		return readinessContract.Engineering, true
	default:
		return readinessArtifactContract{}, false
	}
}

// ReadinessInputDigest computes the canonical digest for tests and host
// adapters that need to stage an artifact before invoking the CLI renderer.
func ReadinessInputDigest(root, slug, kind string) (string, error) {
	contract, ok := readinessContractFor(strings.ToLower(strings.TrimSpace(kind)))
	if !ok {
		return "", fmt.Errorf("unknown readiness digest kind %q", kind)
	}
	return readinessInputsDigest(root, slug, contract.Inputs)
}

func readinessInputsDigest(root, slug string, names []string) (string, error) {
	hash := sha256.New()
	for _, name := range names {
		path := featureFile(root, slug, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", name, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			return "", fmt.Errorf("%s is empty", name)
		}
		fmt.Fprintf(hash, "%s\x00%d\x00", name, len(data))
		_, _ = hash.Write(data)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateDecisionCoverage(root, slug string) error {
	contract := readinessContract.Coverage
	data, err := validateArtifactBase(root, slug, contract)
	if err != nil {
		return err
	}
	if err := validateNoOpenHumanGate(featureFile(root, slug, "questions.md")); err != nil {
		return err
	}

	topology, err := markdownTable(data, "Topology")
	if err != nil || len(topology) == 0 {
		return errors.New("Topology must contain at least one evidence-backed row")
	}
	for i, row := range topology {
		if len(row) < 4 || !substantiveCells(row, 0, 1, 3) {
			return fmt.Errorf("Topology row %d is not evidence-backed", i+1)
		}
	}
	rows, err := markdownTable(data, "Coverage matrix")
	if err != nil || len(rows) == 0 {
		return errors.New("Coverage matrix must contain at least one row")
	}
	allowed := []string{"closed", "agent-owned", "not-applicable", "deferred-nonblocking"}
	for i, row := range rows {
		if len(row) < 6 {
			return fmt.Errorf("Coverage matrix row %d has %d cells; want 6", i+1, len(row))
		}
		status := strings.ToLower(strings.TrimSpace(row[2]))
		if !substantiveCells(row, 0, 1, 5) {
			return fmt.Errorf("Coverage matrix row %d is incomplete", i+1)
		}
		if !slices.Contains(allowed, status) {
			return fmt.Errorf("Coverage matrix row %d has unresolved status %q", i+1, row[2])
		}
		if status == "closed" && !substantiveCells(row, 3) {
			return fmt.Errorf("Coverage matrix row %d is closed without a canonical reference", i+1)
		}
		if (status == "agent-owned" || status == "deferred-nonblocking") &&
			emptyOrNA(row[4]) {
			return fmt.Errorf("Coverage matrix row %d has status %q without an owner/validation gate", i+1, row[2])
		}
	}
	assumptions, _ := markdownTable(data, "Assumption audit")
	for i, row := range assumptions {
		if len(row) < 6 {
			return fmt.Errorf("Assumption audit row %d has %d cells; want 6", i+1, len(row))
		}
		if !noneRow(row[0]) && !substantiveCells(row, 0, 1, 2, 3, 4, 5) {
			return fmt.Errorf("Assumption audit row %d is unowned or unverifiable", i+1)
		}
	}
	residual, _ := markdownTable(data, "Residual uncertainty")
	for i, row := range residual {
		if len(row) < 4 {
			return fmt.Errorf("Residual uncertainty row %d has %d cells; want 4", i+1, len(row))
		}
		if !noneRow(row[0]) && !substantiveCells(row, 0, 1, 2, 3) {
			return fmt.Errorf("Residual uncertainty row %d is unowned or unverifiable", i+1)
		}
	}
	return nil
}

func validateEngineeringReadiness(root, slug string) error {
	contract := readinessContract.Engineering
	data, err := validateArtifactBase(root, slug, contract)
	if err != nil {
		return err
	}
	if err := validateTestPlan(root, slug); err != nil {
		return err
	}

	preflight, err := markdownTable(data, "2a. Build-entry preflight")
	if err != nil || len(preflight) == 0 {
		return errors.New("Build-entry preflight must contain at least one row")
	}
	for i, row := range preflight {
		if len(row) < 7 {
			return fmt.Errorf("Build-entry preflight row %d has %d cells; want 7", i+1, len(row))
		}
		verdict := strings.ToLower(strings.TrimSpace(row[6]))
		if verdict != "pass" && verdict != "n/a" {
			return fmt.Errorf("Build-entry preflight row %d is not passing: %q", i+1, row[6])
		}
		if !substantiveCells(row, 0) ||
			verdict == "pass" && !substantiveCells(row, 1, 2, 4, 5) {
			return fmt.Errorf("Build-entry preflight row %d lacks executable provenance", i+1)
		}
	}

	readiness, err := markdownTable(data, "2b. Implementation readiness")
	if err != nil || len(readiness) == 0 {
		return errors.New("Implementation readiness table must contain at least one row")
	}
	for i, row := range readiness {
		if len(row) < 6 || !strings.EqualFold(strings.TrimSpace(row[5]), "ready") ||
			!substantiveCells(row, 0, 1, 2, 3, 4) {
			return fmt.Errorf("Implementation readiness row %d is not ready", i+1)
		}
	}
	return nil
}

func validateArtifactBase(root, slug string, contract readinessArtifactContract) ([]byte, error) {
	path := featureFile(root, slug, contract.Artifact)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s is missing", contract.Artifact)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("%s is empty", contract.Artifact)
	}
	if readinessPlaceholderRe.Match(data) {
		return nil, fmt.Errorf("%s contains a placeholder", contract.Artifact)
	}
	for _, heading := range contract.RequiredHeadings {
		if _, ok := markdownSection(data, heading); !ok {
			return nil, fmt.Errorf("%s is missing heading %q", contract.Artifact, heading)
		}
	}
	for _, heading := range contract.RequiredTables {
		if _, err := markdownTable(data, heading); err != nil {
			return nil, fmt.Errorf("%s: %w", contract.Artifact, err)
		}
	}
	verdicts := artifactFieldValues(data, contract.VerdictField)
	if len(verdicts) != 1 || !strings.EqualFold(strings.TrimSpace(verdicts[0]), contract.ReadyValue) {
		return nil, fmt.Errorf("%s must contain exactly one %s: %s", contract.Artifact, contract.VerdictField, contract.ReadyValue)
	}
	digests := artifactFieldValues(data, contract.DigestField)
	if len(digests) != 1 || !readinessDigestRe.MatchString(strings.ToLower(strings.TrimSpace(digests[0]))) {
		return nil, fmt.Errorf("%s must contain exactly one valid %s", contract.Artifact, contract.DigestField)
	}
	want, err := readinessInputsDigest(root, slug, contract.Inputs)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(digests[0]), want) {
		return nil, fmt.Errorf("%s input digest is stale", contract.Artifact)
	}
	return data, nil
}

func validateTestPlan(root, slug string) error {
	contract := readinessContract.TestPlan
	path := featureFile(root, slug, contract.Artifact)
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.New("test-plan.md is missing")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return errors.New("test-plan.md is empty")
	}
	if readinessPlaceholderRe.Match(data) {
		return errors.New("test-plan.md contains a placeholder")
	}
	for _, heading := range contract.RequiredHeadings {
		if _, ok := markdownSection(data, heading); !ok {
			return fmt.Errorf("test-plan.md is missing heading %q", heading)
		}
	}
	for _, heading := range contract.RequiredTables {
		rows, err := markdownTable(data, heading)
		if err != nil || len(rows) == 0 {
			return fmt.Errorf("test-plan.md section %q must contain at least one row", heading)
		}
		for i, row := range rows {
			want := []int{0, 1, 2, 3, 5}
			if heading == "Per-gap test requirements" {
				want = []int{0, 1, 2, 3, 4, 5, 6}
			}
			if !substantiveCells(row, want...) {
				return fmt.Errorf("test-plan.md section %q row %d is incomplete", heading, i+1)
			}
		}
	}
	mapping, _ := markdownSection(data, "Acceptance → test map")
	if !strings.Contains(string(mapping), "→") && !strings.Contains(string(mapping), "->") {
		return errors.New("test-plan.md acceptance map has no mappings")
	}
	spec, err := os.ReadFile(featureFile(root, slug, "spec.md"))
	if err != nil {
		return errors.New("spec.md is missing")
	}
	for _, id := range uniqueStrings(acceptanceIDRe.FindAllString(strings.ToUpper(string(spec)), -1)) {
		if !strings.Contains(strings.ToUpper(string(mapping)), id) {
			return fmt.Errorf("test-plan.md acceptance map does not map %s", id)
		}
	}
	return nil
}

func validateNoOpenHumanGate(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var record []string
	check := func(lines []string) error {
		status, ok := state.CursorField(lines, "status")
		if !ok || !strings.EqualFold(strings.TrimSpace(status), "open") {
			return nil
		}
		gate, _ := state.CursorField(lines, "gate")
		switch strings.ToLower(strings.TrimSpace(gate)) {
		case "blocking", "validating", "escalating":
			return fmt.Errorf("questions.md has an open %s question", strings.ToLower(strings.TrimSpace(gate)))
		}
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			if err := check(record); err != nil {
				return err
			}
			record = record[:0]
		}
		record = append(record, line)
	}
	if err := check(record); err != nil {
		return err
	}
	return validateNoOpenQuestionTable(data)
}

func validateNoOpenQuestionTable(data []byte) error {
	statusIndex, gateIndex := -1, -1
	inTable := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			inTable = false
			statusIndex, gateIndex = -1, -1
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if statusIndex < 0 {
			for i, cell := range cells {
				switch strings.ToLower(cell) {
				case "status":
					statusIndex = i
				case "gate":
					gateIndex = i
				}
			}
			inTable = statusIndex >= 0 && gateIndex >= 0
			continue
		}
		if !inTable || statusIndex >= len(cells) || gateIndex >= len(cells) ||
			!strings.EqualFold(cells[statusIndex], "open") {
			continue
		}
		switch gate := strings.ToLower(cells[gateIndex]); gate {
		case "blocking", "validating", "escalating":
			return fmt.Errorf("questions.md has an open %s question", gate)
		}
	}
	return nil
}

func artifactFieldValues(data []byte, key string) []string {
	var values []string
	for _, line := range strings.Split(string(data), "\n") {
		if value, ok := state.CursorField([]string{line}, key); ok {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
}

func markdownSection(data []byte, want string) ([]byte, bool) {
	lines := strings.Split(string(data), "\n")
	start := -1
	level := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		hashes := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
		title := strings.TrimSpace(trimmed[hashes:])
		if start < 0 {
			if strings.EqualFold(title, want) {
				start, level = i+1, hashes
			}
			continue
		}
		if hashes <= level {
			return []byte(strings.Join(lines[start:i], "\n")), true
		}
	}
	if start >= 0 {
		return []byte(strings.Join(lines[start:], "\n")), true
	}
	return nil, false
}

func markdownTable(data []byte, heading string) ([][]string, error) {
	section, ok := markdownSection(data, heading)
	if !ok {
		return nil, fmt.Errorf("missing heading %q", heading)
	}
	var rows [][]string
	headerSeen := false
	for _, line := range strings.Split(string(section), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
			continue
		}
		raw := strings.Split(strings.Trim(trimmed, "|"), "|")
		cells := make([]string, len(raw))
		separator := true
		for i, cell := range raw {
			cells[i] = strings.TrimSpace(cell)
			clean := strings.Trim(cells[i], " :-")
			if clean != "" {
				separator = false
			}
		}
		if separator {
			continue
		}
		if !headerSeen {
			headerSeen = true
			continue
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("heading %q has no table data rows", heading)
	}
	return rows, nil
}

func emptyOrNA(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "" || value == "n/a" || value == "none" || value == "-"
}

func substantiveCells(row []string, indexes ...int) bool {
	for _, index := range indexes {
		if index >= len(row) || emptyOrNA(row[index]) {
			return false
		}
	}
	return true
}

func noneRow(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "none" || value == "no material assumptions" || value == "no residual uncertainty"
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
