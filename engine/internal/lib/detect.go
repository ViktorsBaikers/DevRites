package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// detect.go: repository command detection. Proof obligations in tasks.md and
// runnable CHECK lines in gates.md must cite commands the repository actually
// runs; hand-invented commands go stale or fail at the gate. This command
// resolves the conventional slots (unit test, integration test, lint,
// vet/typecheck, build, package manager) from the repository's own wiring so a
// phase can quote real commands instead of guessing.
//
//	devrites-engine detect commands [--root <dir>] [--json]
//
//	0  report written (unresolved slots are reported, never blocking)
//	2  usage error
//
// Per-slot precedence (first match wins): Makefile target → package.json
// script → language-manifest toolchain. A Makefile encodes the repo's own
// wiring, so it outranks a raw tool; a language manifest is the broad
// fallback. Detection is read-only and never installs tools.

type detectedCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"source"`
	Note    string `json:"note,omitempty"`
}

type commandReport struct {
	ProjectRoot    string            `json:"project_root"`
	PackageManager string            `json:"package_manager,omitempty"`
	Commands       []detectedCommand `json:"commands"`
	Unresolved     []string          `json:"unresolved,omitempty"`
}

// commandSlot is one conventional gate command and its resolvers.
var commandSlots = []string{"unit", "integration", "lint", "vet", "build"}

// makefileTargetRe matches a rule target ("build:" / "build: deps") and skips
// assignments ("VAR := x") and directives (".PHONY").
var makefileTargetRe = regexp.MustCompile(`^([A-Za-z0-9_][A-Za-z0-9_-]*)\s*:(?:\s|$)`)

// makefileSlotMap maps each slot to preferred Makefile target names, in order.
var makefileSlotMap = map[string][]string{
	"unit":        {"test", "test-unit", "unit-test", "test-unit-only", "tests"},
	"integration": {"test-integration", "integration-test", "integration", "test-e2e", "e2e"},
	"lint":        {"lint", "lint-check", "check-lint"},
	"vet":         {"vet", "typecheck", "type-check", "check", "staticcheck"},
	"build":       {"build", "all"},
}

// packageScriptMap maps each slot to preferred package.json script names.
var packageScriptMap = map[string][]string{
	"unit":        {"test:unit", "unit", "test"},
	"integration": {"test:integration", "integration", "test:e2e", "e2e"},
	"lint":        {"lint", "lint:check"},
	"vet":         {"typecheck", "type-check", "tsc", "check"},
	"build":       {"build"},
}

// lockfileManagers orders package-manager detection by lockfile.
var lockfileManagers = []struct {
	lockfile string
	manager  string
}{
	{"pnpm-lock.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"bun.lockb", "bun"},
	{"bun.lock", "bun"},
	{"package-lock.json", "npm"},
	{"npm-shrinkwrap.json", "npm"},
}

// RunDetectCommands implements `devrites-engine detect <subcommand>`; the only
// subcommand is `commands`.
func RunDetectCommands(devritesRoot string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "commands" {
		fmt.Fprintln(stderr, "usage: devrites-engine detect commands [--root <dir>] [--json]")
		return 2
	}
	args = args[1:]
	projectRoot := ""
	asJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "detect commands: --root requires a path")
				return 2
			}
			projectRoot = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "detect commands: unexpected argument %q\n", args[i])
			return 2
		}
	}
	if projectRoot == "" {
		projectRoot = projectRootFromDevritesRoot(devritesRoot)
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "detect commands: %v\n", err)
		return 2
	}
	report := detectCommands(abs)
	if asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "detect commands: %v\n", err)
			return 2
		}
		return 0
	}
	writeCommandReport(report, stdout)
	return 0
}

// detectCommands resolves every slot for one project root.
func detectCommands(root string) commandReport {
	report := commandReport{ProjectRoot: root}
	resolved := map[string]detectedCommand{}

	makeTargets := makefileTargets(root)
	for _, slot := range commandSlots {
		for _, target := range makefileSlotMap[slot] {
			if makeTargets[target] {
				resolved[slot] = detectedCommand{Name: slot, Command: "make " + target, Source: "Makefile"}
				break
			}
		}
	}

	pm := detectPackageManager(root)
	if scripts := packageScripts(root); scripts != nil {
		for _, slot := range commandSlots {
			if _, ok := resolved[slot]; ok {
				continue
			}
			for _, name := range packageScriptMap[slot] {
				if scripts[name] {
					cmd := pm + " run " + name
					if name == "test" && (pm == "npm" || pm == "pnpm" || pm == "yarn") {
						cmd = pm + " test"
					}
					resolved[slot] = detectedCommand{Name: slot, Command: cmd, Source: "package.json"}
					break
				}
			}
		}
	}

	for _, slot := range commandSlots {
		if _, ok := resolved[slot]; ok {
			continue
		}
		if cmd, ok := manifestFallback(root, slot); ok {
			resolved[slot] = cmd
		}
	}

	for _, slot := range commandSlots {
		if cmd, ok := resolved[slot]; ok {
			report.Commands = append(report.Commands, cmd)
		} else {
			report.Unresolved = append(report.Unresolved, slot)
		}
	}
	report.PackageManager = pm
	return report
}

// makefileTargets returns the set of target names defined in a root Makefile.
func makefileTargets(root string) map[string]bool {
	targets := map[string]bool{}
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		raw, err := os.ReadFile(filepath.Join(root, name)) // #nosec G304 -- repo manifest
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
				continue
			}
			if m := makefileTargetRe.FindStringSubmatch(line); m != nil {
				targets[m[1]] = true
			}
		}
		return targets
	}
	return targets
}

// packageScripts returns the script-name set of a root package.json, or nil.
func packageScripts(root string) map[string]bool {
	raw, err := os.ReadFile(filepath.Join(root, "package.json")) // #nosec G304 -- repo manifest
	if err != nil {
		return nil
	}
	var parsed struct {
		Scripts map[string]json.RawMessage `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	scripts := make(map[string]bool, len(parsed.Scripts))
	for name := range parsed.Scripts {
		scripts[name] = true
	}
	return scripts
}

// detectPackageManager resolves the JS package manager from lockfiles.
func detectPackageManager(root string) string {
	if _, err := os.Stat(filepath.Join(root, "package.json")); err != nil {
		return ""
	}
	for _, lm := range lockfileManagers {
		if _, err := os.Stat(filepath.Join(root, lm.lockfile)); err == nil {
			return lm.manager
		}
	}
	return "npm"
}

// manifestFallback resolves one slot from a language manifest's toolchain.
func manifestFallback(root string, slot string) (detectedCommand, bool) {
	type manifest struct {
		file   string
		source string
		cmds   map[string]string
	}
	probe := func(name string) string {
		if _, err := exec.LookPath(name); err != nil {
			return "binary not on PATH"
		}
		return ""
	}
	manifests := []manifest{
		{"go.mod", "go.mod", map[string]string{
			"unit": "go test ./...", "integration": "go test -tags=integration ./...",
			"lint": "golangci-lint run", "vet": "go vet ./...", "build": "go build ./...",
		}},
		{"Cargo.toml", "Cargo.toml", map[string]string{
			"unit": "cargo test", "integration": "cargo test -- --ignored",
			"lint": "cargo clippy --all-targets", "vet": "cargo check", "build": "cargo build",
		}},
		{"pyproject.toml", "pyproject.toml", map[string]string{
			"unit": "python -m pytest", "integration": "python -m pytest -m integration",
			"lint": "ruff check", "vet": "mypy .", "build": "python -m build",
		}},
		{"setup.py", "setup.py", map[string]string{
			"unit": "python -m pytest", "integration": "python -m pytest -m integration",
			"lint": "flake8", "vet": "mypy .",
		}},
		{"pom.xml", "pom.xml", map[string]string{
			"unit": "mvn test", "integration": "mvn verify",
			"vet": "mvn -q compile", "build": "mvn package",
		}},
		{"build.gradle", "build.gradle", map[string]string{
			"unit": "gradle test", "integration": "gradle integrationTest",
			"lint": "gradle check", "build": "gradle build",
		}},
		{"composer.json", "composer.json", map[string]string{
			"unit": "vendor/bin/phpunit", "lint": "vendor/bin/phpcs",
			"vet": "vendor/bin/phpstan analyse",
		}},
		{"Gemfile", "Gemfile", map[string]string{
			"unit": "bundle exec rspec", "integration": "bundle exec rspec --tag integration",
			"lint": "bundle exec rubocop",
		}},
		{"mix.exs", "mix.exs", map[string]string{
			"unit": "mix test", "lint": "mix format --check-formatted",
			"vet": "mix compile --warnings-as-errors", "build": "mix compile",
		}},
	}
	for _, m := range manifests {
		if _, err := os.Stat(filepath.Join(root, m.file)); err != nil {
			continue
		}
		cmd, ok := m.cmds[slot]
		if !ok {
			return detectedCommand{}, false
		}
		out := detectedCommand{Name: slot, Command: cmd, Source: m.source}
		binary := strings.Fields(cmd)[0]
		if binary == "vendor/bin/phpunit" || binary == "vendor/bin/phpcs" || binary == "vendor/bin/phpstan" {
			if _, err := os.Stat(filepath.Join(root, binary)); err != nil {
				out.Note = "not installed under vendor/bin"
			}
		} else if binary == "golangci-lint" || binary == "ruff" || binary == "mypy" || binary == "flake8" {
			if note := probe(binary); note != "" {
				out.Note = note
			}
		}
		return out, true
	}
	// .NET solution/project files need a directory walk, not a fixed name.
	if slot == "unit" || slot == "lint" || slot == "build" {
		if entries, err := os.ReadDir(root); err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasSuffix(name, ".sln") || strings.HasSuffix(name, ".csproj") {
					cmds := map[string]string{
						"unit": "dotnet test", "lint": "dotnet format --verify-no-changes",
						"build": "dotnet build",
					}
					return detectedCommand{Name: slot, Command: cmds[slot], Source: name}, true
				}
			}
		}
	}
	return detectedCommand{}, false
}

// writeCommandReport renders the resolved command set as aligned text.
func writeCommandReport(report commandReport, stdout io.Writer) {
	fmt.Fprintf(stdout, "project-root: %s\n", report.ProjectRoot)
	if report.PackageManager != "" {
		fmt.Fprintf(stdout, "package-manager: %s\n", report.PackageManager)
	}
	sort.Slice(report.Commands, func(i, j int) bool { return report.Commands[i].Name < report.Commands[j].Name })
	for _, cmd := range report.Commands {
		line := fmt.Sprintf("%-12s = %-34s (source: %s)", cmd.Name, cmd.Command, cmd.Source)
		if cmd.Note != "" {
			line += " — " + cmd.Note
		}
		fmt.Fprintln(stdout, line)
	}
	for _, slot := range report.Unresolved {
		fmt.Fprintf(stdout, "%-12s = unresolved — no Makefile target, package script, or known manifest covers it\n", slot)
	}
}
