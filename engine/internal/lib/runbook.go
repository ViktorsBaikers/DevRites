package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

type runbookStep struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type runbookState struct {
	RunID  string `json:"run_id"`
	Source string `json:"source"`
	Next   int    `json:"next"`
	Status string `json:"status"`
}

// Runbook executes tiny project-local automation from .devrites/runbooks/*.yaml.
// It is intentionally not a second lifecycle engine: four flat step kinds only.
func Runbook(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "list":
		return runbookList(root, stdout, stderr)
	case "validate":
		path, err := runbookPath(root, argAt(args, 1))
		if err != nil {
			fmt.Fprintf(stderr, "runbook: %v\n", err)
			return 2
		}
		steps, err := parseRunbook(path)
		if err != nil {
			fmt.Fprintf(stderr, "runbook: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "runbook: OK: %d step(s) in %s\n", len(steps), displayRunbookPath(root, path))
		return 0
	case "run":
		return runbookRun(root, args[1:], stdout, stderr)
	case "resume":
		return runbookResume(root, argAt(args, 1), stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine runbook list|validate|run|resume [...]")
		return 2
	}
}

func runbookList(root string, stdout, stderr io.Writer) int {
	dir := filepath.Join(root, "runbooks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "runbooks: none (.devrites/runbooks/ empty or absent)")
			return 0
		}
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 1
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !(strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			continue
		}
		count++
		fmt.Fprintln(stdout, strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".yaml"), ".yml"))
	}
	if count == 0 {
		fmt.Fprintln(stdout, "runbooks: none (.devrites/runbooks/ empty or absent)")
	}
	return 0
}

func runbookRun(root string, args []string, stdout, stderr io.Writer) int {
	name := argAt(args, 0)
	if name == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine runbook run <name|path> [--dry-run]")
		return 2
	}
	dry := false
	for _, a := range args[1:] {
		if a == "--dry-run" {
			dry = true
			continue
		}
		fmt.Fprintf(stderr, "runbook: unknown option %s\n", a)
		return 2
	}
	path, err := runbookPath(root, name)
	if err != nil {
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 2
	}
	return executeRunbook(root, path, 0, newRunID(), dry, stdout, stderr)
}

func runbookResume(root, id string, stdout, stderr io.Writer) int {
	if id == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine runbook resume <run-id>")
		return 2
	}
	st, err := readRunbookState(root, id)
	if err != nil {
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 2
	}
	return executeRunbook(root, st.Source, st.Next, st.RunID, false, stdout, stderr)
}

func executeRunbook(root, path string, start int, runID string, dry bool, stdout, stderr io.Writer) int {
	steps, err := parseRunbook(path)
	if err != nil {
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 1
	}
	for i := start; i < len(steps); i++ {
		step := steps[i]
		fmt.Fprintf(stdout, "runbook: step %d/%d %s: %s\n", i+1, len(steps), step.Kind, step.Value)
		if dry {
			continue
		}
		switch step.Kind {
		case "engine":
			if code := runEngineStep(root, step.Value, stdout, stderr); code != 0 {
				return code
			}
		case "rite":
			cmd := workflow.ForVerb(step.Value)
			fmt.Fprintf(stdout, "runbook: dispatch manually: %s\n", cmd.Both())
		case "shell":
			if code := runShellStep(filepath.Dir(root), step.Value, stdout, stderr); code != 0 {
				return code
			}
		case "gate":
			st := runbookState{RunID: runID, Source: path, Next: i + 1, Status: "paused"}
			if err := writeRunbookState(root, st); err != nil {
				fmt.Fprintf(stderr, "runbook: %v\n", err)
				return 1
			}
			fmt.Fprintf(stdout, "runbook: paused at gate %q; resume with `devrites-engine runbook resume %s`\n", step.Value, runID)
			return 3
		}
	}
	_ = writeRunbookState(root, runbookState{RunID: runID, Source: path, Next: len(steps), Status: "completed"})
	fmt.Fprintf(stdout, "runbook: completed %s\n", runID)
	return 0
}

func runbookPath(root, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("missing runbook name")
	}
	if strings.ContainsAny(name, `/\`) || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		path := name
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(root), path)
		}
		return filepath.Clean(path), nil
	}
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(root, "runbooks", name+ext)
		if isFile(path) {
			return path, nil
		}
	}
	return filepath.Join(root, "runbooks", name+".yaml"), nil
}

func parseRunbook(path string) ([]runbookStep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read runbook: %w", err)
	}
	var steps []runbookStep
	for _, line := range splitLinesNoTrailing(data) {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		key, val, ok := strings.Cut(body, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		switch key {
		case "engine", "rite", "shell", "gate":
			if val == "" {
				return nil, fmt.Errorf("%s: empty %s step", path, key)
			}
			steps = append(steps, runbookStep{Kind: key, Value: val})
		default:
			return nil, fmt.Errorf("%s: unknown step kind %q", path, key)
		}
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("%s: no steps (expected items like `- engine: doctor`)", path)
	}
	return steps, nil
}

func runEngineStep(root, line string, stdout, stderr io.Writer) int {
	fields := strings.Fields(strings.TrimPrefix(line, "devrites-engine "))
	if len(fields) == 0 {
		return 0
	}
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 1
	}
	cmd := exec.Command(exe, fields...)
	cmd.Env = append(os.Environ(), "DEVRITES_ROOT="+root)
	cmd.Dir = filepath.Dir(root)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			return ex.ExitCode()
		}
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 1
	}
	return 0
}

func runShellStep(project, line string, stdout, stderr io.Writer) int {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", line)
	} else {
		cmd = exec.Command("sh", "-c", line)
	}
	cmd.Dir = project
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ex, ok := err.(*exec.ExitError); ok {
			return ex.ExitCode()
		}
		fmt.Fprintf(stderr, "runbook: %v\n", err)
		return 1
	}
	return 0
}

func writeRunbookState(root string, st runbookState) error {
	path := filepath.Join(root, "runs", st.RunID, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create run dir: %w", err)
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run state: %w", err)
	}
	return state.AtomicWrite(path, append(b, '\n'), 0o644)
}

func readRunbookState(root, id string) (runbookState, error) {
	var st runbookState
	b, err := os.ReadFile(filepath.Join(root, "runs", id, "state.json"))
	if err != nil {
		return st, fmt.Errorf("read run state: %w", err)
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, fmt.Errorf("parse run state %s: %w", id, err)
	}
	if st.Status != "paused" {
		return st, fmt.Errorf("run %s is %s, not paused", id, st.Status)
	}
	return st, nil
}

func newRunID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func displayRunbookPath(root, path string) string {
	if rel, err := filepath.Rel(filepath.Dir(root), path); err == nil {
		return rel
	}
	return path
}
