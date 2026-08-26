package state

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TaskSlice is one SLICE-### block parsed from tasks.md.
type TaskSlice struct {
	ID           string
	Dependencies []string
}

// TaskGraphResult is the deterministic outcome of parsing a tasks.md slice graph.
type TaskGraphResult struct {
	Slices   []TaskSlice
	Cycle    []string // empty when acyclic
	Unknown  []string // dependency ids with no defining slice
	Problems []string // human-readable blockers
}

var (
	sliceHeaderRE  = regexp.MustCompile(`(?m)^##\s+(SLICE-\d+)\b`)
	dependenciesRE = regexp.MustCompile(`(?m)^Dependencies:\s*(.+)\s*$`)
	dependsOnRE    = regexp.MustCompile(`(?m)^depends_on:\s*(.+)\s*$`)
	sliceIDValidRE = regexp.MustCompile(`^SLICE-\d+$`)
)

// ParseTaskGraph reads tasks.md content and validates the slice dependency DAG.
func ParseTaskGraph(tasksMarkdown []byte) TaskGraphResult {
	text := string(tasksMarkdown)
	result := TaskGraphResult{}
	if strings.TrimSpace(text) == "" {
		result.Problems = append(result.Problems, "tasks.md is empty")
		return result
	}

	headers := sliceHeaderRE.FindAllStringSubmatchIndex(text, -1)
	if len(headers) == 0 {
		result.Problems = append(result.Problems, "no SLICE-### sections found")
		return result
	}

	known := make(map[string]bool, len(headers))
	firstIndex := make(map[string]int, len(headers))
	for i, match := range headers {
		id := text[match[2]:match[3]]
		if _, seen := firstIndex[id]; seen {
			result.Problems = append(result.Problems, fmt.Sprintf("duplicate slice id %s", id))
			continue
		}
		firstIndex[id] = i
		known[id] = true
	}

	for i, match := range headers {
		id := text[match[2]:match[3]]
		if firstIndex[id] != i {
			continue
		}
		start := match[1]
		end := len(text)
		if i+1 < len(headers) {
			end = headers[i+1][0]
		}
		block := text[start:end]
		deps, problems := parseSliceGraph(id, block)
		result.Problems = append(result.Problems, problems...)
		result.Slices = append(result.Slices, TaskSlice{ID: id, Dependencies: deps})
		for _, dep := range deps {
			if !known[dep] {
				result.Unknown = append(result.Unknown, dep)
				result.Problems = append(result.Problems, fmt.Sprintf("%s depends on unknown slice %s", id, dep))
			}
		}
	}

	if cycle := findTaskCycle(result.Slices); len(cycle) > 0 {
		result.Cycle = cycle
		result.Problems = append(result.Problems, "dependency cycle: "+strings.Join(cycle, " -> "))
	}
	return result
}

func parseSliceGraph(id, block string) ([]string, []string) {
	var problems []string
	depsLine := firstLineValue(dependenciesRE, block)
	mirrorLine := firstLineValue(dependsOnRE, block)
	deps, malformed := parseIDList(depsLine)
	for _, token := range malformed {
		problems = append(problems, fmt.Sprintf("%s has malformed dependency %q", id, token))
	}
	mirror, mirrorMalformed := parseIDList(mirrorLine)
	for _, token := range mirrorMalformed {
		problems = append(problems, fmt.Sprintf("%s has malformed depends_on %q", id, token))
	}
	if depsLine != "" && mirrorLine != "" && !sameIDSet(deps, mirror) {
		problems = append(problems, fmt.Sprintf("%s Dependencies and depends_on sets differ", id))
	}
	if depsLine == "" && mirrorLine == "" {
		problems = append(problems, fmt.Sprintf("%s is missing Dependencies", id))
	}
	if depsLine == "" {
		deps = mirror
	}
	return deps, problems
}

func firstLineValue(re *regexp.Regexp, block string) string {
	match := re.FindStringSubmatch(block)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func parseIDList(raw string) ([]string, []string) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "[]")
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") {
		return nil, nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	})
	var ids []string
	var malformed []string
	seen := make(map[string]bool)
	for _, part := range parts {
		id := strings.Trim(strings.TrimSpace(part), "`'\"(){}")
		if id == "" {
			continue
		}
		if !sliceIDValidRE.MatchString(id) {
			malformed = append(malformed, id)
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, malformed
}

func sameIDSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			return false
		}
	}
	return true
}

func findTaskCycle(slices []TaskSlice) []string {
	graph := make(map[string][]string, len(slices))
	for _, slice := range slices {
		graph[slice.ID] = append([]string(nil), slice.Dependencies...)
	}
	visited := make(map[string]uint8, len(slices))
	stack := make([]string, 0, len(slices))
	var cycle []string

	var visit func(id string) bool
	visit = func(id string) bool {
		switch visited[id] {
		case 2:
			return false
		case 1:
			for i := len(stack) - 1; i >= 0; i-- {
				cycle = append(cycle, stack[i])
				if stack[i] == id {
					break
				}
			}
			for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
				cycle[i], cycle[j] = cycle[j], cycle[i]
			}
			return true
		}
		visited[id] = 1
		stack = append(stack, id)
		for _, dep := range graph[id] {
			if visit(dep) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		visited[id] = 2
		return false
	}

	for _, slice := range slices {
		if visited[slice.ID] == 0 {
			cycle = nil
			if visit(slice.ID) {
				return cycle
			}
		}
	}
	return nil
}
