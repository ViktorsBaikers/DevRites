package lib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Patterns for reducing an added import line down to a bare top-level package
// name. Quoted module specifiers are authoritative for JS/TS/Go/Rust style
// imports; unquoted Python imports are only considered when a line has no
// quoted specifier, so local JS binding names are never treated as packages.
var (
	quotedModule  = regexp.MustCompile(`\b(?:from|import|require)[[:space:]]*(?:\([[:space:]]*)?['"]([^'"]+)['"]`)
	pythonImport  = regexp.MustCompile(`^\+?[[:space:]]*import[[:space:]]+([A-Za-z0-9_./-]+)`)
	pythonFrom    = regexp.MustCompile(`^\+?[[:space:]]*from[[:space:]]+([A-Za-z0-9_./-]+)[[:space:]]+import\b`)
	scopedPackage = regexp.MustCompile(`^(@[^/]+/[^/]+).*`) // @scope/name/... -> @scope/name
	firstSegment  = regexp.MustCompile(`^([^@][^/]+)/.*`)   // name/sub/...     -> name
	localPath     = regexp.MustCompile(`^(\.|/|[A-Z]:)`)    // relative / absolute / drive path
	// stdlib and framework-builtin prefixes that need no manifest entry.
	builtinPackage = regexp.MustCompile(`^(os|sys|re|json|math|time|datetime|typing|collections|itertools|functools|pathlib|subprocess|logging|fmt|errors|context|strings|strconv|net|http|io|bufio|std|core|alloc|crate|self|super|react|node:|fs|path|util|crypto|events|stream|child_process)$`)
)

// manifestFiles are the dependency manifests a declared package can appear in.
var manifestFiles = []string{"package.json", "requirements.txt", "pyproject.toml", "Pipfile", "go.mod", "Cargo.toml"}

// PackageExistence catches hallucinated or typo-squatted dependencies: every new
// third-party import in the slice diff must be DECLARED in a project manifest, not
// merely imported. It is deterministic and offline — it checks the manifest, not a
// package registry. The workspace is <root>/features/<slug>.
//
//	0  clean, nothing to check, or skipped (not a git repo / no manifest)
//	2  no active workspace
//	3  an imported package is not declared in any manifest
func PackageExistence(root string, args []string, stdout, stderr io.Writer) int {
	slug := slugOrActive(root, args)
	if slug == "" {
		fmt.Fprintln(stderr, "package-existence: no active workspace.")
		return 2
	}
	d := featureDir(root, slug)
	if !isDir(d) {
		fmt.Fprintf(stderr, "package-existence: no workspace at %s.\n", d)
		return 2
	}

	cwd, _ := os.Getwd()
	gitRoot := gitToplevel(cwd)
	if gitRoot == "" {
		fmt.Fprintln(stderr, "package-existence: not a git repo — skipped.")
		return 0
	}
	ref := "HEAD"
	if b, err := os.ReadFile(filepath.Join(d, ".reconcile-base")); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			ref = t
		}
	}

	var manifests []string
	for _, m := range manifestFiles {
		p := filepath.Join(gitRoot, m)
		if isFile(p) {
			manifests = append(manifests, p)
		}
	}
	if len(manifests) == 0 {
		fmt.Fprintln(stderr, "package-existence: no recognized manifest — skipped.")
		return 0
	}

	pkgs := extractImportedPackages(gitRoot, ref)

	bad := 0
	var report strings.Builder
	for _, p := range pkgs {
		if p == "" || builtinPackage.MatchString(p) {
			continue
		}
		if !declaredInManifests(p, manifests) {
			bad++
			fmt.Fprintf(&report, "  - %s: imported but not declared in any manifest\n", p)
		}
	}

	if bad > 0 {
		fmt.Fprintf(stderr, "package-existence: %d imported package(s) are NOT declared in a manifest — verify each exists and add it via the package manager:\n", bad)
		fmt.Fprint(stderr, report.String())
		fmt.Fprintln(stderr, "An undeclared import is how hallucinated/typo-squatted packages slip in. Confirm the name on the registry before trusting it.")
		return 3
	}
	fmt.Fprintln(stdout, "package-existence: OK — every new third-party import is declared in a manifest.")
	return 0
}

// extractImportedPackages returns the sorted, unique top-level package names
// imported by the added ('+') lines of the diff against ref. Each import line is
// reduced to a package name in steps: isolate quoted module specifiers (or, for
// Python, the imported module token), collapse a scoped package or sub-path to
// its top level, and drop local/relative paths.
func extractImportedPackages(gitRoot, ref string) []string {
	out, err := exec.Command("git", "-C", gitRoot, "diff", ref).Output()
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var pkgs []string
	for _, line := range splitLinesNoTrailing(out) {
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue // added lines only, excluding the '+++' file header
		}
		for _, name := range importedModulesFromLine(line) {
			name = normalizeImportedPackage(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				pkgs = append(pkgs, name)
			}
		}
	}
	sort.Strings(pkgs)
	return pkgs
}

func importedModulesFromLine(line string) []string {
	var mods []string
	for _, match := range quotedModule.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			mods = append(mods, match[1])
		}
	}
	if len(mods) > 0 {
		return mods
	}
	if match := pythonFrom.FindStringSubmatch(line); len(match) > 1 {
		return []string{match[1]}
	}
	if match := pythonImport.FindStringSubmatch(line); len(match) > 1 {
		return []string{match[1]}
	}
	return nil
}

func normalizeImportedPackage(name string) string {
	name = scopedPackage.ReplaceAllString(name, "$1")
	name = firstSegment.ReplaceAllString(name, "$1")
	if localPath.MatchString(name) {
		return ""
	}
	return name
}

// declaredInManifests reports whether pkg appears as a whole, bounded token
// (case-insensitively) in any manifest — the name matched literally between
// non-identifier characters, so "lodash" does not match "lodasher".
func declaredInManifests(pkg string, manifests []string) bool {
	re := regexp.MustCompile(`(?im)(^|[^[:alnum:]_-])` + regexp.QuoteMeta(pkg) + `([^[:alnum:]_-]|$)`)
	for _, m := range manifests {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		if re.Match(data) {
			return true
		}
	}
	return false
}
