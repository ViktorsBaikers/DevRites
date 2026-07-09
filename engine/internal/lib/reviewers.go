package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type reviewerInstance struct {
	CLI   string `json:"cli"`
	Model string `json:"model"`
	Agent string `json:"agent"`
}

var reviewerInstanceNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Reviewers reads bounded same-adapter reviewer aliases. It never executes a CLI;
// it only validates the project-local config shape for skills to consume.
func Reviewers(root string, args []string, stdout, stderr io.Writer) int {
	if argAt(args, 0) != "list" {
		fmt.Fprintln(stderr, "usage: devrites-engine reviewers list")
		return 2
	}
	instances, problems := reviewerInstances(root)
	if len(problems) > 0 {
		fmt.Fprintf(stderr, "reviewers: %d problem(s):\n", len(problems))
		for _, p := range problems {
			fmt.Fprintf(stderr, "  - %s\n", p)
		}
		return 1
	}
	if len(instances) == 0 {
		fmt.Fprintln(stdout, "reviewers: none configured")
		return 0
	}
	names := make([]string, 0, len(instances))
	for name := range instances {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		r := instances[name]
		parts := []string{fmt.Sprintf("%s -> %s", name, r.CLI)}
		if r.Model != "" {
			parts = append(parts, "model="+r.Model)
		}
		if r.Agent != "" {
			parts = append(parts, "agent="+r.Agent)
		}
		fmt.Fprintln(stdout, strings.Join(parts, " "))
	}
	return 0
}

func reviewerInstances(root string) (map[string]reviewerInstance, []string) {
	out := map[string]reviewerInstance{}
	if m, ok := readReviewerInstancesJSON(filepath.Join(root, "config.json")); ok {
		mergeReviewerInstances(out, m)
	}
	mergeReviewerInstances(out, readReviewerInstancesFlat(filepath.Join(root, "config")))
	return out, validateReviewerInstances(out)
}

func readReviewerInstancesJSON(path string) (map[string]reviewerInstance, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var raw struct {
		ReviewerInstances map[string]reviewerInstance `json:"reviewer_instances"`
		Review            struct {
			ReviewerInstances map[string]reviewerInstance `json:"reviewer_instances"`
		} `json:"review"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]reviewerInstance{"__invalid_json__": {CLI: ""}}, true
	}
	out := map[string]reviewerInstance{}
	mergeReviewerInstances(out, raw.ReviewerInstances)
	mergeReviewerInstances(out, raw.Review.ReviewerInstances)
	return out, true
}

func readReviewerInstancesFlat(base string) map[string]reviewerInstance {
	vals := readDevritesConfig(base)
	out := map[string]reviewerInstance{}
	for key, val := range vals {
		prefix := "reviewer_instances."
		if strings.HasPrefix(key, "review.reviewer_instances.") {
			prefix = "review.reviewer_instances."
		} else if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		name, field, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		r := out[name]
		switch field {
		case "cli":
			r.CLI = val
		case "model":
			r.Model = val
		case "agent":
			r.Agent = val
		}
		out[name] = r
	}
	return out
}

func mergeReviewerInstances(dst, src map[string]reviewerInstance) {
	for k, v := range src {
		dst[k] = v
	}
}

func validateReviewerInstances(instances map[string]reviewerInstance) []string {
	var problems []string
	for name, r := range instances {
		if name == "__invalid_json__" {
			problems = append(problems, "config.json must be valid JSON")
			continue
		}
		if !reviewerInstanceNameRe.MatchString(name) {
			problems = append(problems, fmt.Sprintf("instance %q must match ^[a-z0-9][a-z0-9-]*$", name))
		}
		switch r.CLI {
		case "claude", "codex":
		case "":
			problems = append(problems, fmt.Sprintf("instance %q needs cli", name))
		default:
			problems = append(problems, fmt.Sprintf("instance %q cli %q is not allowed (claude|codex only)", name, r.CLI))
		}
	}
	return problems
}
