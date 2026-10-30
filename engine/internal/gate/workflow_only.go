package gate

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

var workflowBehaviorRE = regexp.MustCompile(`\bWA-BEH-[A-Z0-9][A-Z0-9-]*\b`)
var workflowModeRE = regexp.MustCompile(`^0[0-7]{3}$`)
var workflowReviewedIdentityRE = regexp.MustCompile(`^Workflow Artifact reviewed identity: [0-9a-f]{64}$`)
var workflowEvidenceRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// workflowOnlyProblems checks planning authority, not executable bytes or runtime
// safety. Native preflight still owns no-follow source/target and content checks.
func workflowOnlyProblems(observation *state.WorkspaceObservation, policy state.PhasePolicy) []string {
	tasks, ok := observation.Fact("tasks.md")
	if !ok || !phaseRequiresTasks(policy) || !state.ParseTaskGraph(tasks.Bytes()).WorkflowOnly {
		return nil
	}
	spec, _ := observation.Fact("spec.md")
	ids := state.ParseAcceptanceMap(spec.Bytes(), nil, nil, false, false).SpecIDs
	var problems []string
	var refs map[string]bool
	if phaseRequiresTestPlan(policy) {
		plan, _ := observation.Fact("test-plan.md")
		var err error
		refs, err = workflowAdmission(plan.Bytes(), observation.Slug())
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	for _, name := range []state.ArtifactPath{"tasks.md", "test-plan.md"} {
		if name == "test-plan.md" && !phaseRequiresTestPlan(policy) {
			continue
		}
		fact, _ := observation.Fact(name)
		structural, _ := markdowntext.Structural(fact.Bytes())
		mapped := map[string]bool{}
		for _, line := range strings.Split(string(structural), "\n") {
			_, cells, table := workflowTableCells(line)
			if !table || len(cells) != 2 {
				continue
			}
			id := strings.TrimSpace(cells[0])
			if id == "" || specACTokenRE.FindString(id) != id {
				continue
			}
			mapped[id] = true
			target := strings.Trim(strings.TrimSpace(cells[1]), "`")
			if target == "" || workflowBehaviorRE.FindString(target) != target || (phaseRequiresTestPlan(policy) && !refs[target]) {
				problems = append(problems, fmt.Sprintf("%s maps %s to invalid or unadmitted Workflow Artifact behavior %q", name, id, target))
			}
		}
		for _, id := range ids {
			if !mapped[id] {
				problems = append(problems, fmt.Sprintf("%s needs an AC-to-Workflow-Artifact row | %s | WA-BEH-ID | naming an admitted behavior", name, id))
			}
		}
	}
	return problems
}

var workflowAdmissionFields = strings.Fields("active_slug readiness_binding_command return_phase return_next_action target_order target_count_limit per_target_bytes_limit aggregate_bytes_limit transaction_file_limit diagnostic_bytes_limit journal_line_limit attempt_epoch_limit proof_command_timeout_seconds proof_aggregate_timeout_seconds proof_terminate_grace_seconds")
var workflowTargetColumns = strings.Split("Index|Path|Mode|Behavior ref|Interface ref|Positive fixture|Failure fixtures|Proof command|Proof cwd|Proof signal|Rollback|Evidence fields", "|")

func workflowAdmission(raw []byte, slug string) (map[string]bool, error) {
	fail := func() (map[string]bool, error) {
		return nil, fmt.Errorf("Workflow Artifact admission is missing or malformed; repair in /rite-plan and rerun /rite-vet")
	}
	structural, err := markdowntext.Structural(raw)
	if err != nil {
		return fail()
	}
	text := strings.ReplaceAll(string(structural), "\r\n", "\n")
	sections := map[string][]string{}
	current := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			current = strings.TrimPrefix(line, "## ")
			if _, duplicate := sections[current]; duplicate {
				return fail()
			}
			sections[current] = nil
		} else if current != "" && strings.TrimSpace(line) != "" {
			sections[current] = append(sections[current], line)
		}
	}
	block := sections["Workflow Artifact admission"]
	if len(block) == 0 || block[0] != "DevRites contract: devrites.workflow-artifact-admission.v1" {
		return fail()
	}
	fields, rest, ok := workflowFields(block[1:], workflowAdmissionFields)
	if !ok || fields["active_slug"] != slug || fields["readiness_binding_command"] != "devrites-engine check readiness --emit-binding "+slug || fields["target_order"] != "utf8-bytewise-path-ascending" {
		return fail()
	}
	if _, ok := state.PolicyFor(state.Phase(fields["return_phase"])); !ok {
		return fail()
	}
	action := strings.Fields(fields["return_next_action"])
	if len(action) < 2 || !strings.HasPrefix(action[0], "/") || action[len(action)-1] != slug {
		return fail()
	}
	bounds := map[string]uint64{}
	for _, key := range workflowAdmissionFields[5:] {
		value := fields[key]
		if value == "" || strings.Trim(value, "0123456789") != "" {
			return fail()
		}
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil || n == 0 {
			return fail()
		}
		bounds[key] = n
	}
	count, epochs := bounds["target_count_limit"], bounds["attempt_epoch_limit"]
	if count > (math.MaxUint64-6)/3 || epochs < 3 || epochs > math.MaxUint64-30 || count > math.MaxUint64-30-epochs || bounds["transaction_file_limit"] < 3*count+6 || bounds["diagnostic_bytes_limit"] != 256 || bounds["journal_line_limit"] < 30+count+epochs || bounds["journal_line_limit"] > 280 || bounds["proof_command_timeout_seconds"] <= bounds["proof_terminate_grace_seconds"] || bounds["proof_aggregate_timeout_seconds"] < bounds["proof_command_timeout_seconds"] {
		return fail()
	}
	if len(rest) < 3 || !workflowHeader(rest[0], workflowTargetColumns) || !delimiterRow(rest[1]) {
		return fail()
	}
	rows := rest[2:]
	if workflowReviewedIdentityRE.MatchString(rows[len(rows)-1]) {
		rows = rows[:len(rows)-1]
	}
	if len(rows) == 0 {
		return fail()
	}
	if uint64(len(rows)) > count {
		return fail()
	}
	refs := map[string]bool{}
	previous := ""
	for i, line := range rows {
		row, ok := workflowCells(line, len(workflowTargetColumns))
		if !ok || row[0] != fmt.Sprintf("%08x", i) || !workflowTargetPath(row[1]) || row[1] <= previous || !workflowModeRE.MatchString(row[2]) {
			return fail()
		}
		previous = row[1]
		if strings.ContainsAny(row[7], ";|\r\n") || strings.Contains(row[7], "&&") || (row[8] != "repository-root" && row[8] != "active-workspace") || len(row[9]) > 128 || row[10] != "restore-preimage-or-absence" {
			return fail()
		}
		for _, c := range row[9] {
			if c < 32 || c > 126 {
				return fail()
			}
		}
		seen := map[string]bool{}
		for _, field := range strings.Split(row[11], ",") {
			field = strings.TrimSpace(field)
			if !workflowEvidenceRE.MatchString(field) || seen[field] {
				return fail()
			}
			seen[field] = true
		}
		referenceIDs := []string{row[3], row[4], row[5]}
		referenceIDs = append(referenceIDs, strings.Split(row[6], ",")...)
		for j, id := range referenceIDs {
			id = strings.TrimSpace(id)
			kind, prefix, keys := "failure-fixture", "WA-FIX-F", strings.Fields("setup fault expected")
			switch j {
			case 0:
				kind, prefix, keys = "behavior", "WA-BEH-", strings.Fields("success observable_effect")
			case 1:
				kind, prefix, keys = "interface", "WA-IF-", strings.Fields("inputs invariants ordering errors configuration performance")
			case 2:
				kind, prefix, keys = "positive-fixture", "WA-FIX-P", strings.Fields("setup action expected")
			}
			suffix := strings.TrimPrefix(id, prefix)
			if suffix == id || suffix == "" || strings.Trim(suffix, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-") != "" || suffix[0] == '-' || refs[id] {
				return fail()
			}
			ref := sections[id]
			if len(ref) == 0 || ref[0] != "DevRites workflow reference: "+kind {
				return fail()
			}
			_, extra, ok := workflowFields(ref[1:], keys)
			if !ok || len(extra) != 0 {
				return fail()
			}
			refs[id] = true
		}
	}
	return refs, nil
}

func workflowHeader(line string, columns []string) bool {
	_, cells, ok := workflowTableCells(line)
	if !ok || len(cells) != len(columns) {
		return false
	}
	for i := range cells {
		if strings.TrimSpace(cells[i]) != columns[i] {
			return false
		}
	}
	return true
}

func workflowFields(lines, keys []string) (map[string]string, []string, bool) {
	if len(lines) < len(keys)+2 || !workflowHeader(lines[0], []string{"Field", "Value"}) || !delimiterRow(lines[1]) {
		return nil, nil, false
	}
	fields := map[string]string{}
	for i, key := range keys {
		_, cells, ok := workflowTableCells(lines[i+2])
		if !ok || len(cells) != 2 || strings.TrimSpace(cells[0]) != key {
			return nil, nil, false
		}
		value, ok := workflowCell(cells[1])
		if !ok {
			return nil, nil, false
		}
		fields[key] = value
	}
	return fields, lines[len(keys)+2:], true
}

func workflowCells(line string, count int) ([]string, bool) {
	_, cells, ok := workflowTableCells(line)
	if !ok || len(cells) != count {
		return nil, false
	}
	for i := range cells {
		cells[i], ok = workflowCell(cells[i])
		if !ok {
			return nil, false
		}
	}
	return cells, true
}

func workflowCell(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 3 || raw[0] != '`' || raw[len(raw)-1] != '`' {
		return "", false
	}
	value := raw[1 : len(raw)-1]
	lower := strings.ToLower(value)
	return value, strings.TrimSpace(value) != "" && !strings.ContainsAny(value, "`<>\r\n") && lower != "todo" && lower != "tbd"
}

func workflowTargetPath(value string) bool {
	if value == "." || path.IsAbs(value) || path.Clean(value) != value || strings.ContainsAny(value, "\\:*?[]{}|\t\r\n") {
		return false
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == ".." || strings.HasPrefix(part, ".") || part == "node_modules" || part == "vendor" {
			return false
		}
	}
	// Canonical workspace ledgers and cross-workspace roots are not executable targets.
	return parts[0] != "work" && parts[0] != "archive" && !strings.HasSuffix(value, "/") && !strings.HasSuffix(value, ".md")
}

// Admission escaping counts the whole backslash run; other Markdown tables do not.
func workflowTableCells(line string) (string, []string, bool) {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '|' || line[len(line)-1] != '|' {
		return "", nil, false
	}
	var cells []string
	var cell strings.Builder
	backslashes := 0
	for _, c := range line[1:] {
		if c == '|' {
			if backslashes%2 == 1 {
				value := cell.String()
				cell.Reset()
				cell.WriteString(value[:len(value)-1])
				cell.WriteRune(c)
			} else {
				cells = append(cells, cell.String())
				cell.Reset()
			}
		} else {
			cell.WriteRune(c)
		}
		if c == '\\' {
			backslashes++
		} else {
			backslashes = 0
		}
	}
	return "", cells, cell.Len() == 0
}
