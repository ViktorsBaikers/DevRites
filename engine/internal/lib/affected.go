package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// AffectedSelection is the validation a changed-path set requires during Build.
type AffectedSelection struct {
	Packages []string // Go packages whose tests must run
	Widen    bool     // full suite required
	Reason   string   // why the selection widened
}

// widenPrefixes force the full suite: each can change behaviour in packages
// that no changed path names.
var widenPrefixes = []string{
	".github/workflows/",
	".github/actions/",
	"pack/generated/",
	"engine/internal/lib/testdata/",
}

// widenBasenames are module manifests at any depth: engine/go.mod is not the
// only one, and a manifest change re-resolves the whole module.
var widenBasenames = map[string]bool{"go.mod": true, "go.sum": true, "go.work": true}

// SelectAffected maps changed repository paths to the narrowest safe validation.
// It fails safe: a path it cannot classify widens to the full suite instead of
// silently narrowing.
func SelectAffected(changed []string) AffectedSelection {
	seen := map[string]bool{}
	var pkgs []string
	for _, raw := range changed {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		p = path.Clean(strings.TrimPrefix(p, "./"))
		if reason, widen := widenReason(p); widen {
			return AffectedSelection{Widen: true, Reason: reason}
		}
		dir := path.Dir(p)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		pkgs = append(pkgs, dir)
	}
	sort.Strings(pkgs)
	if len(pkgs) == 0 {
		return AffectedSelection{Widen: true, Reason: "no Go package affected"}
	}
	return AffectedSelection{Packages: pkgs}
}

func widenReason(p string) (string, bool) {
	if widenBasenames[path.Base(p)] {
		return p + " is a module manifest", true
	}
	for _, w := range widenPrefixes {
		if strings.HasPrefix(p, w) {
			return p + " can change packages no changed path names", true
		}
	}
	// An unclassified path can carry behaviour this selection cannot see, so it
	// must not be dropped just because a sibling path named a package.
	if !strings.HasSuffix(p, ".go") {
		return p + " is not a Go source path", true
	}
	return "", false
}

// pkgDeps is one `go list` row: a package's directory, import path, and every
// import it reaches, test-only imports included.
type pkgDeps struct {
	Dir    string
	Import string
	Deps   []string
}

// goListDeps is a variable so tests can inject a module graph without a toolchain.
var goListDeps = runGoListDeps

func runGoListDeps(moduleRoot string) ([]pkgDeps, error) {
	const format = "{{.Dir}}\t{{.ImportPath}}\t{{join .Deps \",\"}}\t{{join .TestImports \",\"}}\t{{join .XTestImports \",\"}}"
	cmd := exec.Command("go", "list", "-f", format, "./...") // #nosec G204 -- fixed args, no caller input
	cmd.Dir = moduleRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	rows := make([]pkgDeps, 0, 64)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<22)
	for sc.Scan() {
		parts := strings.Split(sc.Text(), "\t")
		if len(parts) != 5 {
			continue
		}
		row := pkgDeps{Dir: parts[0], Import: parts[1]}
		for _, field := range parts[2:] {
			for _, imp := range strings.Split(field, ",") {
				if imp = strings.TrimSpace(imp); imp != "" {
					row.Deps = append(row.Deps, imp)
				}
			}
		}
		rows = append(rows, row)
	}
	return rows, sc.Err()
}

// moduleRootFor returns the deepest ancestor of dir holding a go.mod.
func moduleRootFor(dir string) (string, bool) {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// expandDependents adds every package whose imports reach a selected package: a
// change breaks its dependents' tests too, so those packages must run as well.
func expandDependents(selected []string, rows []pkgDeps) []string {
	importers := map[string][]string{}
	for _, r := range rows {
		for _, dep := range r.Deps {
			importers[dep] = append(importers[dep], r.Import)
		}
	}
	seen := map[string]bool{}
	queue := append([]string(nil), selected...)
	for _, s := range selected {
		seen[s] = true
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, imp := range importers[cur] {
			if !seen[imp] {
				seen[imp] = true
				queue = append(queue, imp)
			}
		}
	}
	out := make([]string, 0, len(seen))
	for imp := range seen {
		out = append(out, imp)
	}
	sort.Strings(out)
	return out
}

// locateSelection resolves selected repository-relative dirs to one module root
// plus absolute dirs. A selection spanning modules or missing a go.mod widens.
func locateSelection(root string, dirs []string) (string, []string, bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", nil, false
	}
	var moduleRoot string
	abs := make([]string, 0, len(dirs))
	for _, d := range dirs {
		a := filepath.Join(absRoot, filepath.FromSlash(d))
		mr, ok := moduleRootFor(a)
		if !ok {
			return "", nil, false
		}
		if moduleRoot == "" {
			moduleRoot = mr
		} else if moduleRoot != mr {
			return "", nil, false
		}
		abs = append(abs, a)
	}
	return moduleRoot, abs, moduleRoot != ""
}

// RunCheckAffected prints the affected-only validation plan for changed paths.
func RunCheckAffected(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine check affected <path...>")
		return 2
	}
	sel := SelectAffected(args)
	if sel.Widen {
		fmt.Fprintf(stdout, "affected: full suite (%s)\n", sel.Reason)
		return 0
	}
	moduleRoot, absDirs, ok := locateSelection(root, sel.Packages)
	if !ok {
		fmt.Fprintln(stdout, "affected: full suite (selection does not resolve to one Go module)")
		return 0
	}
	rows, err := goListDeps(moduleRoot)
	if err != nil {
		fmt.Fprintf(stdout, "affected: full suite (dependency graph unavailable: %v)\n", err)
		return 0
	}
	byDir := make(map[string]string, len(rows))
	for _, r := range rows {
		byDir[filepath.Clean(r.Dir)] = r.Import
	}
	selected := make([]string, 0, len(absDirs))
	for _, d := range absDirs {
		imp, found := byDir[filepath.Clean(d)]
		if !found {
			fmt.Fprintf(stdout, "affected: full suite (%s is not a listed Go package)\n", d)
			return 0
		}
		selected = append(selected, imp)
	}
	all := expandDependents(selected, rows)
	suffix := ""
	if len(all) > len(selected) {
		suffix = " (includes dependents)"
	}
	fmt.Fprintf(stdout, "affected: %d package(s): %s%s\n", len(all), strings.Join(all, " "), suffix)
	return 0
}
