package main_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestWorkspaceObservationMigrationGuards(t *testing.T) {
	engineRoot := filepath.Clean("..")

	t.Run("superseded readers are absent", func(t *testing.T) {
		program := parseMigrationProgram(t, engineRoot)
		for _, problem := range obsoleteSurfaceProblems(program) {
			t.Error(problem)
		}
	})

	t.Run("acquisition calls are exact", func(t *testing.T) {
		program := parseMigrationProgram(t, engineRoot)
		for _, problem := range acquisitionProblems(program) {
			t.Error(problem)
		}
	})

	t.Run("consumer reachability stays retained", func(t *testing.T) {
		program := parseMigrationProgram(t, engineRoot)
		for _, problem := range consumerReachabilityProblems(program, migrationConsumerEntries()) {
			t.Error(problem)
		}
	})

	t.Run("command documentation names closed contract", func(t *testing.T) {
		path := filepath.Join(engineRoot, "..", "docs", "engine", "commands.md")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Join(strings.Fields(string(content)), " ")
		for _, want := range []string{
			"`absent`", "`empty`", "`malformed`", "`unsafe`", "`unreadable`", "`present`",
			"`malformed_markdown`", "`parent_symlink`", "`final_symlink`", "`non_regular`",
			"`file_too_large`", "`permission_denied`", "`read_failure`",
			"next: repair <logical-path>: replace invalid Markdown with valid Markdown; required artifacts need substantive content",
			"next: repair <logical-path>: replace the symlinked parent with a real directory",
			"next: repair <logical-path>: replace the symlink with a regular file",
			"next: repair <logical-path>: replace the non-regular entry with a regular file",
			"next: repair <logical-path>: reduce the file to at most 1 MiB",
			"next: repair <logical-path>: grant read permission",
			"next: repair <logical-path>: restore a readable regular file",
			"; optional readiness input may instead be removed",
			"readiness input <logical-path> is malformed (malformed_markdown); replace invalid Markdown with valid Markdown",
			"readiness input <logical-path> is unsafe (parent_symlink); replace the symlinked parent with a real directory",
			"readiness input <logical-path> is unsafe (final_symlink); replace the symlink with a regular file",
			"readiness input <logical-path> is unsafe (non_regular); replace the non-regular entry with a regular file",
			"readiness input <logical-path> is unsafe (file_too_large); reduce the file to at most 1 MiB",
			"readiness input <logical-path> is unreadable (permission_denied); grant read permission",
			"readiness input <logical-path> is unreadable (read_failure); restore a readable regular file",
			"`workspace_invalid`", "`aggregate_too_large`", "`concurrent_change`",
			"workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry",
			"workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry",
			"add real content to state.md and retry", "repair state.md and retry",
			"record phase in state.md and retry", "record a known phase in state.md and retry",
			"1 MiB per file", "8 MiB aggregate",
			"artifact: <logical-path>: <state> (<code>)",
			"Status emits diagnostics without recovery or `next:` lines",
			"Gate emits diagnostics after `reason` and before recovery, `invariant`, and `retry` lines",
			"Whole observation failures use stderr, exit `2`, and no lifecycle result or reason on stdout",
			"Standalone readiness-binding failures use one stderr line, exit `3`, and empty stdout",
			"These seven codes are the closed Workspace Observation classification and recovery mapping outcomes. A selected public consumer emits only a code reachable for its consumed fixed logical path. Invalid workspace ancestry is `workspace_invalid`, not an artifact `parent_symlink` diagnostic.",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
	})
}

func TestWorkspaceObservationMigrationGuardFixtures(t *testing.T) {
	t.Run("unrelated Present is accepted", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/state/snapshot.go": `package state
	type Snapshot struct {
		Present bool
	}
	func Present() bool { return Snapshot{Present: true}.Present }
	`,
		})
		if problems := obsoleteSurfaceProblems(program); len(problems) != 0 {
			t.Fatalf("unrelated Present identifier was rejected: %v", problems)
		}
	})

	t.Run("obsolete State and Gate surfaces are rejected", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/state/feature.go": `package state
	type Feature struct {
		Present map[string]bool
		PresentFiles map[string]bool
	}
	func LoadFeature() *Feature { return nil }
	func declaredPhaseFromLedger() {}
	`,
			"internal/gate/gate.go": `package gate
	import lifecycle "github.com/devrites/devrites/internal/state"
	func Check() { _ = lifecycle.LoadFeature }
	`,
		})
		problems := obsoleteSurfaceProblems(program)
		for _, want := range []string{"state type Feature", "Feature.Present", "Feature.PresentFiles", "state function LoadFeature", "state function declaredPhaseFromLedger", "state.LoadFeature"} {
			if !migrationProblemContains(problems, want) {
				t.Errorf("obsolete surface %q was not rejected: %v", want, problems)
			}
		}
	})

	t.Run("reachable helper reopen", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/state/status.go": `package state
func Status() { statusWithCallback() }
func statusWithCallback() { reopen().Read() }
`,
			"internal/state/reopen.go": `package state
import "os"
func reopen() any {
	_, _ = os.ReadFile("state.md")
	return nil
}
`,
		})
		problems := consumerReachabilityProblems(program, []migrationEntry{{directory: "internal/state", function: "Status"}})
		if !migrationProblemContains(problems, "internal/state/reopen.go", `filesystem content-reader package "os"`) {
			t.Fatalf("reachable helper reopen was not rejected: %v", problems)
		}
	})

	t.Run("reachable receiver method reopen", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/state/status.go": `package state
	type retainedConsumer struct{}
	func Status() {
		var consumer retainedConsumer
		consumer.reopen()
	}
	`,
			"internal/state/reopen.go": `package state
	import "os"
	func (retainedConsumer) reopen() {
		_, _ = os.ReadFile("state.md")
	}
	`,
		})
		problems := consumerReachabilityProblems(program, []migrationEntry{{directory: "internal/state", function: "Status"}})
		if !migrationProblemContains(problems, "internal/state/reopen.go", `filesystem content-reader package "os"`) {
			t.Fatalf("reachable receiver-method reopen was not rejected: %v", problems)
		}
	})

	for _, test := range []struct {
		name string
		body string
	}{
		{name: "direct constructor result receiver", body: "newRetainedConsumer().reopen()"},
		{name: "assigned constructor result receiver", body: "consumer := newRetainedConsumer()\nconsumer.reopen()"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := parseMigrationFixture(t, map[string]string{
				"internal/state/status.go": fmt.Sprintf(`package state
	type retainedConsumer struct{}
	func newRetainedConsumer() *retainedConsumer { return &retainedConsumer{} }
	func Status() {
		%s
	}
	`, test.body),
				"internal/state/reopen.go": `package state
	import "os"
	func (*retainedConsumer) reopen() {
		_, _ = os.ReadFile("state.md")
	}
	`,
			})
			problems := consumerReachabilityProblems(program, []migrationEntry{{directory: "internal/state", function: "Status"}})
			if !migrationProblemContains(problems, "internal/state/reopen.go", `filesystem content-reader package "os"`) {
				t.Fatalf("constructor-result receiver method was not rejected: %v", problems)
			}
		})
	}

	t.Run("reachable internal helper package", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/gate/gate.go": `package gate
import "github.com/devrites/devrites/internal/workspacereader"
func Check() { workspacereader.Read() }
`,
		})
		problems := consumerReachabilityProblems(program, []migrationEntry{{directory: "internal/gate", function: "Check"}})
		if !migrationProblemContains(problems, "internal/gate/gate.go", `unapproved internal package "github.com/devrites/devrites/internal/workspacereader"`) {
			t.Fatalf("reachable internal helper package was not rejected: %v", problems)
		}
	})

	t.Run("aliased helper second capture", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/gate/gate.go": `package gate
import observed "github.com/devrites/devrites/internal/state"
func Check() { _, _ = observed.ObserveWorkspace("", "") }
func helper() {
	capture := observed.ObserveWorkspace
	_, _ = capture("", "")
}
`,
		})
		problems := acquisitionProblems(program)
		if !migrationProblemContains(problems, "internal/gate/gate.go:helper", "state.ObserveWorkspace") {
			t.Fatalf("aliased helper second capture was not rejected: %v", problems)
		}
	})

	t.Run("alias assigned outside loop and called in loop is rejected", func(t *testing.T) {
		program := parseCompleteMigrationAcquisitionFixture(t, `capture := state.ObserveWorkspace
for range []int{1, 2} {
	capture()
}`)
		problems := acquisitionProblems(program)
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "acquisition function reference", "state.ObserveWorkspace") {
			t.Fatalf("loop-called acquisition alias was not rejected: %v", problems)
		}
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "calls state.ObserveWorkspace 0 times, want 1") {
			t.Fatalf("acquisition alias satisfied direct-call cardinality: %v", problems)
		}
	})

	t.Run("anonymous closure assigned outside loop and called in loop is rejected", func(t *testing.T) {
		program := parseCompleteMigrationAcquisitionFixture(t, `capture := func() { state.ObserveWorkspace() }
for range []int{1, 2} {
	capture()
}`)
		problems := acquisitionProblems(program)
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "acquisition function reference", "state.ObserveWorkspace") {
			t.Fatalf("loop-called acquisition closure was not rejected: %v", problems)
		}
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "calls state.ObserveWorkspace 0 times, want 1") {
			t.Fatalf("acquisition closure satisfied direct-call cardinality: %v", problems)
		}
	})

	t.Run("uncalled function value is rejected without satisfying call count", func(t *testing.T) {
		program := parseCompleteMigrationAcquisitionFixture(t, `_ = state.ObserveWorkspace`)
		problems := acquisitionProblems(program)
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "acquisition function reference", "state.ObserveWorkspace") {
			t.Fatalf("uncalled acquisition function value was not rejected: %v", problems)
		}
		if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "calls state.ObserveWorkspace 0 times, want 1") {
			t.Fatalf("uncalled acquisition function value satisfied direct-call cardinality: %v", problems)
		}
	})

	t.Run("dot-import direct call is accepted", func(t *testing.T) {
		program := parseMigrationFixture(t, map[string]string{
			"internal/state/observation.go": `package state
func ObserveWorkspace() { observeWorkspace() }
func observeWorkspace() {}
`,
			"internal/state/status.go": `package state
func Status() { statusWithCallback() }
func statusWithCallback() { observeWorkspace() }
`,
			"internal/gate/gate.go": fmt.Sprintf(`package gate
import . %q
func Check() { ObserveWorkspace() }
`, migrationStateImport),
			"internal/gate/readiness_binding.go": fmt.Sprintf(`package gate
import %q
func ReadinessBinding() { state.ObserveWorkspace() }
`, migrationStateImport),
		})
		if problems := acquisitionProblems(program); len(problems) != 0 {
			t.Fatalf("dot-import acquisition call was rejected: %v", problems)
		}
	})

	for _, test := range []struct {
		name       string
		checkBody  string
		wantLoop   string
		wantAccept bool
	}{
		{name: "direct call in for initializer is accepted", checkBody: `for state.ObserveWorkspace(); ; {
	break
}`, wantAccept: true},
		{name: "direct call in for condition is rejected", checkBody: `for ; state.ObserveWorkspace(); {
	break
}`, wantLoop: "for"},
		{name: "direct call in for post is rejected", checkBody: `for ; false; state.ObserveWorkspace() {}`, wantLoop: "for"},
		{name: "direct call in range expression is accepted", checkBody: `for range state.ObserveWorkspace() {}`, wantAccept: true},
		{name: "nested loop initializer remains repeat capable", checkBody: `for range []int{1, 2} {
	for state.ObserveWorkspace(); ; {
		break
	}
}`, wantLoop: "range"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := parseCompleteMigrationAcquisitionFixture(t, test.checkBody)
			problems := acquisitionProblems(program)
			if test.wantAccept {
				if len(problems) != 0 {
					t.Fatalf("one-shot acquisition call was rejected: %v", problems)
				}
				return
			}
			if !migrationProblemContains(problems, "internal/gate/gate.go:Check", "calls state.ObserveWorkspace", test.wantLoop+" loop", "repeated acquisition risk") {
				t.Fatalf("repeat-capable acquisition call was not rejected: %v", problems)
			}
		})
	}

	loops := []struct {
		name string
		body func(string) string
	}{
		{
			name: "for",
			body: func(call string) string {
				return "for i := 0; i < 2; i++ {\n" + call + "\n}"
			},
		},
		{
			name: "range",
			body: func(call string) string {
				return "for range []int{1, 2} {\n" + call + "\n}"
			},
		},
	}
	sites := []struct {
		name     string
		file     string
		function string
		target   string
	}{
		{name: "state ObserveWorkspace wrapper", file: "internal/state/observation.go", function: "ObserveWorkspace", target: "observeWorkspace"},
		{name: "state statusWithCallback", file: "internal/state/status.go", function: "statusWithCallback", target: "observeWorkspace"},
		{name: "gate Check", file: "internal/gate/gate.go", function: "Check", target: "state.ObserveWorkspace"},
		{name: "gate ReadinessBinding", file: "internal/gate/readiness_binding.go", function: "ReadinessBinding", target: "state.ObserveWorkspace"},
	}
	for _, loop := range loops {
		for _, site := range sites {
			t.Run(loop.name+" around "+site.name, func(t *testing.T) {
				observationBody := "observeWorkspace()"
				statusBody := "observeWorkspace()"
				checkBody := "state.ObserveWorkspace()"
				bindingBody := "state.ObserveWorkspace()"
				switch site.file {
				case "internal/state/observation.go":
					observationBody = loop.body(observationBody)
				case "internal/state/status.go":
					statusBody = loop.body(statusBody)
				case "internal/gate/gate.go":
					checkBody = loop.body(checkBody)
				case "internal/gate/readiness_binding.go":
					bindingBody = loop.body(bindingBody)
				}
				program := parseMigrationFixture(t, map[string]string{
					"internal/state/observation.go":      fmt.Sprintf("package state\nfunc ObserveWorkspace() {\n%s\n}\nfunc observeWorkspace() {}\n", observationBody),
					"internal/state/status.go":           fmt.Sprintf("package state\nfunc Status() { statusWithCallback() }\nfunc statusWithCallback() {\n%s\n}\n", statusBody),
					"internal/gate/gate.go":              fmt.Sprintf("package gate\nimport %q\nfunc Check() {\n%s\n}\n", migrationStateImport, checkBody),
					"internal/gate/readiness_binding.go": fmt.Sprintf("package gate\nimport %q\nfunc ReadinessBinding() {\n%s\n}\n", migrationStateImport, bindingBody),
				})
				problems := acquisitionProblems(program)
				if !migrationProblemContains(problems, site.file+":"+site.function, site.target, loop.name+" loop", "repeated acquisition risk") {
					t.Fatalf("looped acquisition reference was not rejected: %v", problems)
				}
			})
		}
	}
}

const migrationStateImport = "github.com/devrites/devrites/internal/state"

type migrationEntry struct {
	directory string
	receiver  string
	function  string
}

type migrationFile struct {
	name        string
	directory   string
	packageName string
	syntax      *ast.File
	imports     map[string]string
}

type migrationFunction struct {
	file *migrationFile
	decl *ast.FuncDecl
}

type migrationProgram struct {
	files     []*migrationFile
	functions map[migrationEntry]*migrationFunction
}

type migrationAcquisitionUse struct {
	file          string
	function      string
	target        string
	loop          string
	functionValue bool
}

type migrationAcquisitionVisitor struct {
	file          *migrationFile
	function      string
	loop          string
	functionValue bool
	uses          *[]migrationAcquisitionUse
}

func (v migrationAcquisitionVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch node := node.(type) {
	case *ast.FuncLit:
		v.functionValue = true
		v.walk(node.Body)
		return nil
	case *ast.ForStmt:
		v.walk(node.Init)
		repeated := v.repeated("for")
		repeated.walk(node.Cond)
		repeated.walk(node.Post)
		repeated.walk(node.Body)
		return nil
	case *ast.RangeStmt:
		v.walk(node.Key)
		v.walk(node.Value)
		v.walk(node.X)
		v.repeated("range").walk(node.Body)
		return nil
	case *ast.CallExpr:
		if target := migrationAcquisitionTarget(v.file, node.Fun); target != "" {
			*v.uses = append(*v.uses, migrationAcquisitionUse{file: v.file.name, function: v.function, target: target, loop: v.loop, functionValue: v.functionValue})
		} else {
			v.walk(node.Fun)
		}
		for _, argument := range node.Args {
			v.walk(argument)
		}
		return nil
	case *ast.SelectorExpr:
		if target := migrationAcquisitionTarget(v.file, node); target != "" {
			*v.uses = append(*v.uses, migrationAcquisitionUse{file: v.file.name, function: v.function, target: target, loop: v.loop, functionValue: true})
		}
		v.walk(node.X)
		return nil
	case *ast.Ident:
		if target := migrationAcquisitionTarget(v.file, node); target != "" {
			*v.uses = append(*v.uses, migrationAcquisitionUse{file: v.file.name, function: v.function, target: target, loop: v.loop, functionValue: true})
		}
	}
	return v
}

func (v migrationAcquisitionVisitor) walk(node ast.Node) {
	if node != nil {
		ast.Walk(v, node)
	}
}

func (v migrationAcquisitionVisitor) repeated(loop string) migrationAcquisitionVisitor {
	if v.loop == "" {
		v.loop = loop
	}
	return v
}

func migrationAcquisitionTarget(file *migrationFile, expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return migrationAcquisitionTarget(file, expression.X)
	case *ast.SelectorExpr:
		receiver, ok := expression.X.(*ast.Ident)
		if ok && expression.Sel.Name == "ObserveWorkspace" && file.imports[receiver.Name] == migrationStateImport {
			return "state.ObserveWorkspace"
		}
	case *ast.Ident:
		if expression.Name == "observeWorkspace" && file.packageName == "state" {
			return "observeWorkspace"
		}
		if expression.Name == "ObserveWorkspace" && file.imports["."] == migrationStateImport {
			return "state.ObserveWorkspace"
		}
	}
	return ""
}

func parseMigrationProgram(t *testing.T, engineRoot string) *migrationProgram {
	t.Helper()
	program := newMigrationProgram()
	for _, name := range productionGoFiles(t, engineRoot) {
		program.addFile(t, name, filepath.Join(engineRoot, name), nil)
	}
	return program
}

func parseMigrationFixture(t *testing.T, sources map[string]string) *migrationProgram {
	t.Helper()
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	program := newMigrationProgram()
	for _, name := range names {
		program.addFile(t, name, name, []byte(sources[name]))
	}
	return program
}

func parseCompleteMigrationAcquisitionFixture(t *testing.T, checkBody string) *migrationProgram {
	t.Helper()
	return parseMigrationFixture(t, map[string]string{
		"internal/state/observation.go": `package state
func ObserveWorkspace() { observeWorkspace() }
func observeWorkspace() {}
`,
		"internal/state/status.go": `package state
func Status() { statusWithCallback() }
func statusWithCallback() { observeWorkspace() }
`,
		"internal/gate/gate.go": fmt.Sprintf(`package gate
import %q
func Check() {
%s
}
`, migrationStateImport, checkBody),
		"internal/gate/readiness_binding.go": fmt.Sprintf(`package gate
import %q
func ReadinessBinding() { state.ObserveWorkspace() }
`, migrationStateImport),
	})
}

func newMigrationProgram() *migrationProgram {
	return &migrationProgram{functions: make(map[migrationEntry]*migrationFunction)}
}

func (p *migrationProgram) addFile(t *testing.T, name, parsePath string, source []byte) {
	t.Helper()
	var parserSource any
	if source != nil {
		parserSource = source
	}
	file, err := parser.ParseFile(token.NewFileSet(), parsePath, parserSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	name = filepath.ToSlash(name)
	parsed := &migrationFile{
		name:        name,
		directory:   filepath.ToSlash(filepath.Dir(name)),
		packageName: file.Name.Name,
		syntax:      file,
		imports:     make(map[string]string),
	}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		alias := filepath.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		parsed.imports[alias] = importPath
	}
	p.files = append(p.files, parsed)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		entry := migrationEntry{
			directory: parsed.directory,
			receiver:  migrationReceiverName(function),
			function:  function.Name.Name,
		}
		p.functions[entry] = &migrationFunction{file: parsed, decl: function}
	}
}

func migrationReceiverName(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	return migrationTypeName(function.Recv.List[0].Type)
}

func migrationTypeName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return migrationTypeName(expression.X)
	case *ast.IndexExpr:
		return migrationTypeName(expression.X)
	case *ast.IndexListExpr:
		return migrationTypeName(expression.X)
	default:
		return ""
	}
}

func obsoleteSurfaceProblems(program *migrationProgram) []string {
	stateFunctions := map[string]bool{
		"LoadFeature":           true,
		"MissingFor":            true,
		"MissingWorkspaceFiles": true,
		"NewReport":             true,
	}
	stateReaders := map[string]bool{
		"declaredPhaseFromLedger": true,
		"sectionPresentAny":       true,
		"sectionPresent":          true,
	}
	gateReaders := map[string]bool{
		"openHumanGates":     true,
		"readReadinessInput": true,
	}
	gateConstants := map[string]bool{
		"maxReadinessInputBytes": true,
		"maxReadinessTotalBytes": true,
	}
	stateSelectors := map[string]bool{
		"Feature":               true,
		"LoadFeature":           true,
		"MissingFor":            true,
		"MissingWorkspaceFiles": true,
		"NewReport":             true,
	}

	seen := make(map[string]bool)
	var problems []string
	addProblem := func(problem string) {
		if !seen[problem] {
			seen[problem] = true
			problems = append(problems, problem)
		}
	}
	for _, file := range program.files {
		for _, declaration := range file.syntax.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				receiver := migrationReceiverName(declaration)
				switch {
				case file.directory == "internal/state" && receiver == "" && (stateFunctions[declaration.Name.Name] || stateReaders[declaration.Name.Name]):
					addProblem(fmt.Sprintf("%s defines superseded state function %s", file.name, declaration.Name.Name))
				case file.directory == "internal/state" && receiver == "WorkspaceObservation" && declaration.Name.Name == "MissingSections":
					addProblem(fmt.Sprintf("%s defines superseded WorkspaceObservation.MissingSections", file.name))
				case file.directory == "internal/gate" && receiver == "" && gateReaders[declaration.Name.Name]:
					addProblem(fmt.Sprintf("%s defines superseded gate reader %s", file.name, declaration.Name.Name))
				}
				entry := migrationEntry{directory: file.directory, receiver: receiver, function: declaration.Name.Name}
				function := program.functions[entry]
				if function == nil {
					continue
				}
				variableTypes := migrationVariableTypes(program, function)
				ast.Inspect(declaration.Body, func(node ast.Node) bool {
					switch node := node.(type) {
					case *ast.CallExpr:
						identifier, ok := node.Fun.(*ast.Ident)
						if !ok {
							return true
						}
						if file.directory == "internal/state" && (stateFunctions[identifier.Name] || stateReaders[identifier.Name]) {
							addProblem(fmt.Sprintf("%s:%s calls superseded state function %s", file.name, declaration.Name.Name, identifier.Name))
						}
						if file.directory == "internal/gate" && gateReaders[identifier.Name] {
							addProblem(fmt.Sprintf("%s:%s calls superseded gate reader %s", file.name, declaration.Name.Name, identifier.Name))
						}
					case *ast.SelectorExpr:
						receiverType := migrationExpressionType(program, function.file.directory, node.X, variableTypes)
						if file.directory == "internal/state" && receiverType == "Feature" && (node.Sel.Name == "Present" || node.Sel.Name == "PresentFiles") {
							addProblem(fmt.Sprintf("%s:%s selects superseded Feature.%s", file.name, declaration.Name.Name, node.Sel.Name))
						}
						if file.directory == "internal/state" && receiverType == "WorkspaceObservation" && node.Sel.Name == "MissingSections" {
							addProblem(fmt.Sprintf("%s:%s calls superseded WorkspaceObservation.MissingSections", file.name, declaration.Name.Name))
						}
					}
					return true
				})
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if file.directory == "internal/state" && spec.Name.Name == "Feature" {
							addProblem(fmt.Sprintf("%s defines superseded state type Feature", file.name))
							if structure, ok := spec.Type.(*ast.StructType); ok {
								for _, field := range structure.Fields.List {
									for _, name := range field.Names {
										if name.Name == "Present" || name.Name == "PresentFiles" {
											addProblem(fmt.Sprintf("%s defines superseded Feature.%s", file.name, name.Name))
										}
									}
								}
							}
						}
					case *ast.ValueSpec:
						if file.directory == "internal/gate" {
							for _, name := range spec.Names {
								if gateConstants[name.Name] {
									addProblem(fmt.Sprintf("%s defines superseded gate constant %s", file.name, name.Name))
								}
							}
						}
					}
				}
			}
		}
		ast.Inspect(file.syntax, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if ok && file.imports[receiver.Name] == migrationStateImport && stateSelectors[selector.Sel.Name] {
				addProblem(fmt.Sprintf("%s selects superseded state.%s", file.name, selector.Sel.Name))
			}
			return true
		})
	}
	sort.Strings(problems)
	return problems
}

func acquisitionProblems(program *migrationProgram) []string {
	allowed := map[migrationAcquisitionUse]int{
		{file: "internal/state/observation.go", function: "ObserveWorkspace", target: "observeWorkspace"}:            1,
		{file: "internal/state/status.go", function: "statusWithCallback", target: "observeWorkspace"}:               1,
		{file: "internal/gate/gate.go", function: "Check", target: "state.ObserveWorkspace"}:                         1,
		{file: "internal/gate/readiness_binding.go", function: "ReadinessBinding", target: "state.ObserveWorkspace"}: 1,
	}
	counts := make(map[migrationAcquisitionUse]int)
	var problems []string
	for _, use := range migrationAcquisitionUses(program) {
		if use.functionValue {
			problems = append(problems, fmt.Sprintf("%s:%s retains acquisition function reference %s; only direct calls are approved", use.file, use.function, use.target))
			continue
		}
		key := use
		key.loop = ""
		counts[key]++
		if _, ok := allowed[key]; !ok {
			problems = append(problems, fmt.Sprintf("%s:%s retains unapproved acquisition call %s", use.file, use.function, use.target))
		} else if use.loop != "" {
			problems = append(problems, fmt.Sprintf("%s:%s calls %s inside a %s loop; repeated acquisition risk", use.file, use.function, use.target, use.loop))
		}
	}
	for acquisition, want := range allowed {
		if got := counts[acquisition]; got != want {
			problems = append(problems, fmt.Sprintf("%s:%s calls %s %d times, want %d", acquisition.file, acquisition.function, acquisition.target, got, want))
		}
	}
	status := migrationEntry{directory: "internal/state", function: "Status"}
	statusFunction, ok := program.functions[status]
	if !ok || statusFunction.file.name != "internal/state/status.go" {
		problems = append(problems, "internal/state/status.go:Status is missing")
	} else if got := migrationLocalReferenceCount(program, status, "statusWithCallback"); got != 1 {
		problems = append(problems, fmt.Sprintf("internal/state/status.go:Status references statusWithCallback %d times, want 1", got))
	}
	sort.Strings(problems)
	return problems
}

func migrationAcquisitionUses(program *migrationProgram) []migrationAcquisitionUse {
	var uses []migrationAcquisitionUse
	for _, file := range program.files {
		for _, declaration := range file.syntax.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Body != nil {
					inspectMigrationAcquisitions(file, declaration.Name.Name, declaration.Body, &uses)
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, expression := range value.Values {
						inspectMigrationAcquisitions(file, "<package>", expression, &uses)
					}
				}
			}
		}
	}
	return uses
}

func inspectMigrationAcquisitions(file *migrationFile, function string, root ast.Node, uses *[]migrationAcquisitionUse) {
	ast.Walk(migrationAcquisitionVisitor{file: file, function: function, uses: uses}, root)
}

func inspectMigrationIdentifiers(root ast.Node, visit func(*ast.Ident)) {
	var inspect func(ast.Node) bool
	inspect = func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			ast.Inspect(selector.X, inspect)
			return false
		}
		if identifier, ok := node.(*ast.Ident); ok {
			visit(identifier)
		}
		return true
	}
	ast.Inspect(root, inspect)
}

func migrationLocalReferenceCount(program *migrationProgram, entry migrationEntry, target string) int {
	function, ok := program.functions[entry]
	if !ok {
		return 0
	}
	count := 0
	inspectMigrationIdentifiers(function.decl.Body, func(identifier *ast.Ident) {
		if identifier.Name == target {
			count++
		}
	})
	return count
}

func migrationConsumerEntries() []migrationEntry {
	return []migrationEntry{
		{directory: "internal/state", function: "Status"},
		{directory: "internal/gate", function: "Check"},
		{directory: "internal/gate", function: "ReadinessBinding"},
	}
}

func consumerReachabilityProblems(program *migrationProgram, roots []migrationEntry) []string {
	forbiddenImports := map[string]bool{
		"io":            true,
		"os":            true,
		"path/filepath": true,
		"github.com/devrites/devrites/internal/devritespaths": true,
	}
	allowedInternalImports := map[string]bool{
		"github.com/devrites/devrites/internal/markdowntext": true,
		"github.com/devrites/devrites/internal/reason":       true,
		migrationStateImport:                                 true,
	}
	queue := append([]migrationEntry(nil), roots...)
	visited := make(map[migrationEntry]bool)
	seenProblems := make(map[string]bool)
	var problems []string
	addProblem := func(problem string) {
		if !seenProblems[problem] {
			seenProblems[problem] = true
			problems = append(problems, problem)
		}
	}
	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]
		if visited[entry] {
			continue
		}
		visited[entry] = true
		function, ok := program.functions[entry]
		if !ok {
			addProblem(fmt.Sprintf("consumer entry %s.%s is missing", entry.directory, entry.function))
			continue
		}
		ast.Inspect(function.decl.Body, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			importPath := function.file.imports[receiver.Name]
			switch {
			case forbiddenImports[importPath]:
				addProblem(fmt.Sprintf("reachable consumer file %s imports filesystem content-reader package %q", function.file.name, importPath))
			case strings.HasPrefix(importPath, "github.com/devrites/devrites/internal/") && !allowedInternalImports[importPath]:
				addProblem(fmt.Sprintf("reachable consumer file %s delegates through unapproved internal package %q", function.file.name, importPath))
			}
			return true
		})
		inspectMigrationIdentifiers(function.decl.Body, func(identifier *ast.Ident) {
			callee := migrationEntry{directory: entry.directory, function: identifier.Name}
			if _, exists := program.functions[callee]; exists && !migrationAcquisitionBoundary(entry, callee) {
				queue = append(queue, callee)
			}
		})
		for _, callee := range migrationMethodCallees(program, entry, function) {
			if !migrationAcquisitionBoundary(entry, callee) {
				queue = append(queue, callee)
			}
		}
	}
	sort.Strings(problems)
	return problems
}

func migrationMethodCallees(program *migrationProgram, caller migrationEntry, function *migrationFunction) []migrationEntry {
	variableTypes := migrationVariableTypes(program, function)
	var callees []migrationEntry
	ast.Inspect(function.decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver := migrationExpressionType(program, function.file.directory, selector.X, variableTypes)
		if receiver == "" {
			return true
		}
		callee := migrationEntry{directory: caller.directory, receiver: receiver, function: selector.Sel.Name}
		if _, exists := program.functions[callee]; exists {
			callees = append(callees, callee)
		}
		return true
	})
	return callees
}

func migrationVariableTypes(program *migrationProgram, function *migrationFunction) map[string]string {
	types := make(map[string]string)
	addFields := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			typeName := migrationTypeName(field.Type)
			for _, name := range field.Names {
				types[name.Name] = typeName
			}
		}
	}
	addFields(function.decl.Recv)
	addFields(function.decl.Type.Params)
	addFields(function.decl.Type.Results)

	ast.Inspect(function.decl.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.ValueSpec:
			for i, name := range node.Names {
				typeName := migrationTypeName(node.Type)
				if typeName == "" && i < len(node.Values) {
					typeName = migrationExpressionType(program, function.file.directory, node.Values[i], types)
				}
				if typeName != "" {
					types[name.Name] = typeName
				}
			}
		case *ast.AssignStmt:
			if len(node.Lhs) != len(node.Rhs) {
				return true
			}
			for i, left := range node.Lhs {
				identifier, ok := left.(*ast.Ident)
				if !ok {
					continue
				}
				typeName := migrationExpressionType(program, function.file.directory, node.Rhs[i], types)
				if typeName != "" {
					types[identifier.Name] = typeName
				}
			}
		}
		return true
	})
	return types
}

func migrationExpressionType(program *migrationProgram, directory string, expression ast.Expr, variableTypes map[string]string) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return variableTypes[expression.Name]
	case *ast.CompositeLit:
		return migrationTypeName(expression.Type)
	case *ast.UnaryExpr:
		return migrationExpressionType(program, directory, expression.X, variableTypes)
	case *ast.ParenExpr:
		return migrationExpressionType(program, directory, expression.X, variableTypes)
	case *ast.CallExpr:
		identifier, ok := expression.Fun.(*ast.Ident)
		if !ok {
			return ""
		}
		if identifier.Name == "new" && len(expression.Args) == 1 {
			return migrationTypeName(expression.Args[0])
		}
		return migrationFunctionResultType(program.functions[migrationEntry{directory: directory, function: identifier.Name}])
	}
	return ""
}

func migrationFunctionResultType(function *migrationFunction) string {
	if function == nil || function.decl.Type.Results == nil || len(function.decl.Type.Results.List) != 1 {
		return ""
	}
	result := function.decl.Type.Results.List[0]
	if len(result.Names) > 1 {
		return ""
	}
	return migrationTypeName(result.Type)
}

func migrationAcquisitionBoundary(caller, callee migrationEntry) bool {
	return caller == (migrationEntry{directory: "internal/state", function: "statusWithCallback"}) &&
		callee == (migrationEntry{directory: "internal/state", function: "observeWorkspace"})
}

func migrationProblemContains(problems []string, fragments ...string) bool {
	for _, problem := range problems {
		matched := true
		for _, fragment := range fragments {
			matched = matched && strings.Contains(problem, fragment)
		}
		if matched {
			return true
		}
	}
	return false
}

func productionGoFiles(t *testing.T, engineRoot string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(engineRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(engineRoot, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
