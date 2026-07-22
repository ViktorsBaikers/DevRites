package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Import patterns are language-specific: JS/TS uses quoted specifiers, Go uses
// the standard parser, and Python/Rust use their import declaration syntax.
var (
	quotedModule  = regexp.MustCompile(`\b(?:from|import|require)[[:space:]]*(?:\([[:space:]]*)?['"]([^'"]+)['"]`)
	pythonImport  = regexp.MustCompile(`^\+?[[:space:]]*import[[:space:]]+([A-Za-z0-9_./-]+)`)
	pythonFrom    = regexp.MustCompile(`^\+?[[:space:]]*from[[:space:]]+([A-Za-z0-9_./-]+)[[:space:]]+import\b`)
	rustUse       = regexp.MustCompile(`^[[:space:]]*(?:pub(?:\([^)]*\))?[[:space:]]+)?use[[:space:]]+(?:::)?(?:r#)?([A-Za-z_][A-Za-z0-9_]*)(?:::|[[:space:]]+as\b|[[:space:]]*;)`)
	rustExtern    = regexp.MustCompile(`^[[:space:]]*(?:pub[[:space:]]+)?extern[[:space:]]+crate[[:space:]]+(?:r#)?([A-Za-z_][A-Za-z0-9_]*)\b`)
	requirement   = regexp.MustCompile(`^[[:space:]]*([A-Za-z0-9][A-Za-z0-9._-]*)[[:space:]]*(?:\[[^]]*\])?[[:space:]]*(?:@|[<>=!~;]|$)`)
	quotedValue   = regexp.MustCompile(`["']([^"']+)["']`)
	scopedPackage = regexp.MustCompile(`^(@[^/]+/[^/]+).*`) // @scope/name/... -> @scope/name
	firstSegment  = regexp.MustCompile(`^([^@][^/]+)/.*`)   // name/sub/...     -> name
	localPath     = regexp.MustCompile(`^(\.|/|[A-Z]:)`)    // relative / absolute / drive path
	// stdlib and framework-builtin prefixes that need no manifest entry.
	builtinPackage = regexp.MustCompile(`^(os|sys|re|json|math|time|datetime|typing|collections|itertools|functools|pathlib|subprocess|logging|fmt|errors|context|strings|strconv|net|http|io|bufio|std|core|alloc|crate|self|super|react|node:|fs|path|util|crypto|events|stream|child_process)$`)
)

// manifestFiles are the dependency manifests a declared package can appear in.
var manifestFiles = []string{"package.json", "requirements.txt", "pyproject.toml", "Pipfile", "go.mod", "Cargo.toml"}

type importedModule struct {
	sourcePath string
	specifier  string
}

type packageResolution struct {
	name          string
	resolved      bool
	manifestFound bool
	ignored       bool
}

type packageResolver struct {
	extensions string
	extract    func(string, string, []string) []string
	resolve    func(string, string, string) packageResolution
}

var packageResolvers = []packageResolver{
	{extensions: " .js .jsx .mjs .cjs .ts .tsx .mts .cts .svelte .vue ", extract: quotedModulesFromAddedLines, resolve: resolveJavaScriptImport},
	{extensions: " .go ", extract: goModulesFromAddedLinesForSource, resolve: resolveGoImport},
	{extensions: " .py .pyi ", extract: pythonModulesFromAddedLines, resolve: resolvePythonImport},
	{extensions: " .rs ", extract: rustModulesFromAddedLines, resolve: resolveRustImport},
}

var fallbackPackageResolver = packageResolver{extract: quotedModulesFromAddedLines, resolve: resolveGenericImport}

// PackageExistence catches hallucinated or typo-squatted dependencies: every new
// third-party import in the slice diff must be DECLARED in a project manifest, not
// merely imported. It is deterministic and offline: it checks the manifest, not a
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
		fmt.Fprintln(stderr, "package-existence: not a git repo: skipped.")
		return 0
	}
	ref := "HEAD"
	if b, err := os.ReadFile(filepath.Join(d, ".reconcile-base")); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			ref = t
		}
	}

	var rootManifests []string
	for _, m := range manifestFiles {
		p := filepath.Join(gitRoot, m)
		if isFile(p) {
			rootManifests = append(rootManifests, p)
		}
	}

	imports := extractImportedPackages(gitRoot, ref)
	hasManifest := len(rootManifests) > 0
	findings := map[string]struct{}{}
	for _, imported := range imports {
		result := resolverForSource(imported.sourcePath).resolve(gitRoot, imported.sourcePath, imported.specifier)
		if result.ignored {
			continue
		}
		if result.manifestFound {
			hasManifest = true
		}
		if result.resolved {
			continue
		}
		findings[result.name] = struct{}{}
	}

	if !hasManifest {
		fmt.Fprintln(stderr, "package-existence: no recognized manifest: skipped.")
		return 0
	}

	if len(findings) > 0 {
		packages := make([]string, 0, len(findings))
		for pkg := range findings {
			packages = append(packages, pkg)
		}
		sort.Strings(packages)
		fmt.Fprintf(stderr, "package-existence: %d imported package(s) are NOT declared in a manifest: verify each exists and add it via the package manager:\n", len(packages))
		for _, pkg := range packages {
			fmt.Fprintf(stderr, "  - %s: imported but not declared in any manifest\n", pkg)
		}
		fmt.Fprintln(stderr, "An undeclared import is how hallucinated/typo-squatted packages slip in. Confirm the name on the registry before trusting it.")
		return 3
	}
	fmt.Fprintln(stdout, "package-existence: OK: every new third-party import is declared in a manifest.")
	return 0
}

// extractImportedPackages retains the source file and raw specifier for each
// import added by the diff so workspace and alias resolution can use that
// provenance before the specifier is reduced to a top-level package name.
func extractImportedPackages(gitRoot, ref string) []importedModule {
	out, err := exec.Command("git", "-C", gitRoot, "diff", "--name-only", "-z", ref, "--").Output()
	if err != nil {
		return nil
	}
	var imports []importedModule
	for rawPath := range bytes.SplitSeq(out, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		sourcePath := filepath.Clean(string(rawPath))
		diff, err := exec.Command("git", "-C", gitRoot, "diff", "--unified=0", ref, "--", sourcePath).Output()
		if err != nil {
			continue
		}
		var addedLines []string
		for _, line := range splitLinesNoTrailing(diff) {
			if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
				continue
			}
			addedLines = append(addedLines, strings.TrimPrefix(line, "+"))
		}
		for _, specifier := range importedModulesFromAddedLines(gitRoot, sourcePath, addedLines) {
			imports = append(imports, importedModule{sourcePath: sourcePath, specifier: specifier})
		}
	}
	return imports
}

func importedModulesFromAddedLines(gitRoot, sourcePath string, lines []string) []string {
	return resolverForSource(sourcePath).extract(gitRoot, sourcePath, lines)
}

func resolverForSource(sourcePath string) packageResolver {
	extension := " " + strings.ToLower(filepath.Ext(sourcePath)) + " "
	for _, resolver := range packageResolvers {
		if strings.Contains(resolver.extensions, extension) {
			return resolver
		}
	}
	return fallbackPackageResolver
}

func resolveJavaScriptImport(gitRoot, sourcePath, specifier string) packageResolution {
	pkg := normalizeImportedPackage(specifier)
	if pkg == "" || builtinPackage.MatchString(pkg) {
		return packageResolution{ignored: true}
	}
	manifest := nearestFile(gitRoot, sourcePath, "package.json")
	return packageResolution{
		name:          pkg,
		resolved:      importMatchesPathAlias(gitRoot, sourcePath, specifier) || manifest != "" && packageJSONDeclares(pkg, manifest),
		manifestFound: manifest != "",
	}
}

func resolveGoImport(gitRoot, sourcePath, specifier string) packageResolution {
	if isGoStandardLibrary(specifier) {
		return packageResolution{ignored: true}
	}
	resolved, manifestFound := goImportResolved(gitRoot, sourcePath, specifier)
	return packageResolution{name: specifier, resolved: resolved, manifestFound: manifestFound}
}

func resolvePythonImport(gitRoot, sourcePath, specifier string) packageResolution {
	module := topLevelPythonModule(specifier)
	if module == "" || isPythonStandardLibrary(module) {
		return packageResolution{ignored: true}
	}
	resolved, manifestFound := pythonImportResolved(gitRoot, sourcePath, normalizeDistributionName(module))
	return packageResolution{name: module, resolved: resolved, manifestFound: manifestFound}
}

func resolveRustImport(gitRoot, sourcePath, specifier string) packageResolution {
	crate := normalizeRustCrate(specifier)
	if crate == "" || isRustBuiltin(crate) {
		return packageResolution{ignored: true}
	}
	resolved, manifestFound := rustImportResolved(gitRoot, sourcePath, crate)
	return packageResolution{name: crate, resolved: resolved, manifestFound: manifestFound}
}

// resolveGenericImport keeps quoted imports in unregistered source types in the
// shared pipeline. It deliberately fails closed: without ecosystem semantics,
// text in an unrelated manifest cannot safely prove that an import is declared.
func resolveGenericImport(gitRoot, sourcePath, specifier string) packageResolution {
	pkg := normalizeImportedPackage(specifier)
	if pkg == "" {
		return packageResolution{ignored: true}
	}
	return packageResolution{
		name:          pkg,
		manifestFound: len(nearestFiles(gitRoot, sourcePath, manifestFiles)) > 0,
	}
}

func quotedModulesFromAddedLines(_, _ string, lines []string) []string {
	var modules []string
	for _, line := range lines {
		modules = append(modules, importedModulesFromLine(line)...)
	}
	return modules
}

func goModulesFromAddedLinesForSource(gitRoot, sourcePath string, lines []string) []string {
	return goModulesFromAddedLines(filepath.Join(gitRoot, sourcePath), lines)
}

func pythonModulesFromAddedLines(_, _ string, lines []string) []string {
	var modules []string
	for _, line := range lines {
		modules = append(modules, pythonModulesFromLine(line)...)
	}
	return modules
}

func rustModulesFromAddedLines(_, _ string, lines []string) []string {
	var modules []string
	for _, line := range lines {
		modules = append(modules, rustModulesFromLine(line)...)
	}
	return modules
}

func goModulesFromAddedLines(sourcePath string, lines []string) []string {
	file, _ := parser.ParseFile(token.NewFileSet(), sourcePath, nil, parser.ImportsOnly|parser.AllErrors)
	if file == nil {
		var modules []string
		for _, line := range lines {
			modules = append(modules, importedModulesFromLine(line)...)
		}
		return modules
	}
	var modules []string
	for _, spec := range file.Imports {
		module, err := strconv.Unquote(spec.Path.Value)
		if err != nil || !addedLinesContainImport(lines, spec.Path.Value, module) {
			continue
		}
		modules = append(modules, module)
	}
	return modules
}

func addedLinesContainImport(lines []string, quoted, module string) bool {
	for _, line := range lines {
		if strings.Contains(line, quoted) || strings.Contains(line, "`"+module+"`") {
			return true
		}
	}
	return false
}

func pythonModulesFromLine(line string) []string {
	line = strings.TrimSpace(stripConfigComment(line, '#'))
	if line == "" {
		return nil
	}
	var modules []string
	for statement := range strings.SplitSeq(line, ";") {
		statement = strings.TrimSpace(statement)
		if match := pythonFrom.FindStringSubmatch(statement); len(match) > 1 {
			if module := topLevelPythonModule(match[1]); module != "" {
				modules = append(modules, module)
			}
			continue
		}
		if !strings.HasPrefix(statement, "import ") {
			continue
		}
		for imported := range strings.SplitSeq(strings.TrimPrefix(statement, "import "), ",") {
			fields := strings.Fields(strings.TrimSpace(imported))
			if len(fields) == 0 {
				continue
			}
			if module := topLevelPythonModule(fields[0]); module != "" {
				modules = append(modules, module)
			}
		}
	}
	return modules
}

func topLevelPythonModule(module string) string {
	module = strings.TrimSpace(module)
	if module == "" || strings.HasPrefix(module, ".") {
		return ""
	}
	return strings.SplitN(module, ".", 2)[0]
}

func rustModulesFromLine(line string) []string {
	line = stripConfigComment(line, '/')
	if match := rustUse.FindStringSubmatch(line); len(match) > 1 {
		return []string{match[1]}
	}
	if match := rustExtern.FindStringSubmatch(line); len(match) > 1 {
		return []string{match[1]}
	}
	return nil
}

func isGoStandardLibrary(module string) bool {
	return strings.Contains(" C archive/tar archive/zip bufio bytes cmp compress/bzip2 compress/flate compress/gzip compress/lzw compress/zlib container/heap container/list container/ring context crypto crypto/aes crypto/cipher crypto/des crypto/dsa crypto/ecdh crypto/ecdsa crypto/ed25519 crypto/elliptic crypto/fips140 crypto/hkdf crypto/hmac crypto/hpke crypto/md5 crypto/mlkem crypto/mlkem/mlkemtest crypto/pbkdf2 crypto/rand crypto/rc4 crypto/rsa crypto/sha1 crypto/sha256 crypto/sha3 crypto/sha512 crypto/subtle crypto/tls crypto/x509 crypto/x509/pkix database/sql database/sql/driver debug/buildinfo debug/dwarf debug/elf debug/gosym debug/macho debug/pe debug/plan9obj embed encoding encoding/ascii85 encoding/asn1 encoding/base32 encoding/base64 encoding/binary encoding/csv encoding/gob encoding/hex encoding/json encoding/pem encoding/xml errors expvar flag fmt go/ast go/build go/build/constraint go/constant go/doc go/doc/comment go/format go/importer go/parser go/printer go/scanner go/token go/types go/version hash hash/adler32 hash/crc32 hash/crc64 hash/fnv hash/maphash html html/template image image/color image/color/palette image/draw image/gif image/jpeg image/png index/suffixarray io io/fs io/ioutil iter log log/slog log/syslog maps math math/big math/bits math/cmplx math/rand math/rand/v2 mime mime/multipart mime/quotedprintable net net/http net/http/cgi net/http/cookiejar net/http/fcgi net/http/httptest net/http/httptrace net/http/httputil net/http/pprof net/mail net/netip net/rpc net/rpc/jsonrpc net/smtp net/textproto net/url os os/exec os/signal os/user path path/filepath plugin reflect regexp regexp/syntax runtime runtime/cgo runtime/coverage runtime/debug runtime/metrics runtime/pprof runtime/race runtime/trace slices sort strconv strings structs sync sync/atomic syscall testing testing/cryptotest testing/fstest testing/iotest testing/quick testing/slogtest testing/synctest text/scanner text/tabwriter text/template text/template/parse time time/tzdata unicode unicode/utf16 unicode/utf8 unique unsafe weak ", " "+module+" ")
}

func goImportResolved(gitRoot, sourcePath, module string) (bool, bool) {
	manifest := nearestFile(gitRoot, sourcePath, "go.mod")
	if manifest == "" {
		return false, false
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return false, true
	}
	localModule, required, ok := parseGoMod(string(data))
	if !ok {
		return false, true
	}
	if modulePathContains(localModule, module) {
		return true, true
	}
	for _, dependency := range required {
		if modulePathContains(dependency, module) {
			return true, true
		}
	}
	return false, true
}

func parseGoMod(data string) (string, []string, bool) {
	var module string
	var required []string
	inRequireBlock := false
	for rawLine := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(strings.SplitN(rawLine, "//", 2)[0])
		if line == "" {
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			if fields := strings.Fields(line); len(fields) >= 2 {
				required = append(required, strings.Trim(fields[0], `"`))
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			module = strings.Trim(fields[1], `"`)
		}
		if len(fields) >= 2 && fields[0] == "require" {
			if fields[1] == "(" {
				inRequireBlock = true
			} else if len(fields) >= 3 {
				required = append(required, strings.Trim(fields[1], `"`))
			}
		}
	}
	return module, required, module != "" && !inRequireBlock
}

func modulePathContains(module, imported string) bool {
	return module != "" && (imported == module || strings.HasPrefix(imported, module+"/"))
}

func normalizeDistributionName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var normalized strings.Builder
	separator := false
	for _, char := range name {
		if char == '-' || char == '_' || char == '.' {
			separator = true
			continue
		}
		if separator && normalized.Len() > 0 {
			normalized.WriteByte('-')
		}
		separator = false
		normalized.WriteRune(char)
	}
	return normalized.String()
}

func isPythonStandardLibrary(module string) bool {
	return strings.Contains(" __future__ abc aifc argparse array ast asynchat asyncio asyncore atexit audioop base64 bdb binascii bisect builtins bz2 calendar cgi cgitb chunk cmath cmd code codecs codeop collections colorsys compileall concurrent configparser contextlib contextvars copy copyreg crypt csv ctypes curses dataclasses datetime dbm decimal difflib dis doctest email encodings ensurepip enum errno faulthandler fcntl filecmp fileinput fnmatch fractions ftplib functools gc getopt getpass gettext glob graphlib grp gzip hashlib heapq hmac html http idlelib imaplib imghdr importlib inspect io ipaddress itertools json keyword linecache locale logging lzma mailbox mailcap marshal math mimetypes mmap modulefinder multiprocessing netrc nis nntplib numbers operator optparse os pathlib pdb pickle pickletools pipes pkgutil platform plistlib poplib posix pprint profile pstats pty pwd py_compile pyclbr pydoc queue quopri random re readline reprlib resource rlcompleter runpy sched secrets select selectors shelve shlex shutil signal site smtpd smtplib sndhdr socket socketserver sqlite3 ssl stat statistics string stringprep struct subprocess sunau symtable sys sysconfig tabnanny tarfile telnetlib tempfile termios textwrap threading time timeit tkinter token tokenize tomllib trace traceback tracemalloc tty turtle turtledemo types typing unicodedata unittest urllib uu uuid venv warnings wave weakref webbrowser winreg winsound wsgiref xdrlib xml xmlrpc zipapp zipfile zipimport zlib zoneinfo ", " "+module+" ")
}

func pythonImportResolved(gitRoot, sourcePath, module string) (bool, bool) {
	manifests := nearestFiles(gitRoot, sourcePath, []string{"requirements.txt", "pyproject.toml", "Pipfile"})
	if len(manifests) == 0 {
		return false, false
	}
	declared := map[string]struct{}{}
	for _, manifest := range manifests {
		data, err := os.ReadFile(manifest)
		if err != nil {
			continue
		}
		var packages map[string]struct{}
		switch filepath.Base(manifest) {
		case "requirements.txt":
			packages = requirementPackages(string(data))
		case "pyproject.toml":
			packages, _ = pyprojectPackages(string(data))
		case "Pipfile":
			packages = pipfilePackages(string(data))
		}
		for pkg := range packages {
			declared[pkg] = struct{}{}
		}
	}
	_, ok := declared[module]
	return ok, true
}

func requirementPackages(data string) map[string]struct{} {
	packages := map[string]struct{}{}
	for rawLine := range strings.SplitSeq(data, "\n") {
		addRequirementPackage(packages, stripConfigComment(rawLine, '#'))
	}
	return packages
}

func addRequirementPackage(packages map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return
	}
	if match := requirement.FindStringSubmatch(value); len(match) > 1 {
		packages[normalizeDistributionName(match[1])] = struct{}{}
	}
}

func pyprojectPackages(data string) (map[string]struct{}, bool) {
	packages := map[string]struct{}{}
	section := ""
	arrayOpen := false
	for rawLine := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(stripConfigComment(rawLine, '#'))
		if line == "" {
			continue
		}
		if arrayOpen {
			addQuotedRequirements(packages, line)
			if containsUnquotedByte(line, ']') {
				arrayOpen = false
			}
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, false
			}
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		key, value, ok := tomlAssignment(line)
		if !ok {
			continue
		}
		if section == "tool.poetry.dependencies" || strings.HasPrefix(section, "tool.poetry.group.") && strings.HasSuffix(section, ".dependencies") {
			if normalizeDistributionName(key) != "python" {
				packages[normalizeDistributionName(key)] = struct{}{}
			}
			continue
		}
		arrayDependency := section == "project" && key == "dependencies" || section == "project.optional-dependencies" || section == "dependency-groups"
		if !arrayDependency {
			continue
		}
		addQuotedRequirements(packages, value)
		arrayOpen = containsUnquotedByte(value, '[') && !containsUnquotedByte(value, ']')
	}
	if arrayOpen {
		return nil, false
	}
	return packages, true
}

func addQuotedRequirements(packages map[string]struct{}, value string) {
	for _, match := range quotedValue.FindAllStringSubmatch(value, -1) {
		if len(match) > 1 {
			addRequirementPackage(packages, match[1])
		}
	}
}

func pipfilePackages(data string) map[string]struct{} {
	packages := map[string]struct{}{}
	section := ""
	for rawLine := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(stripConfigComment(rawLine, '#'))
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		if section != "packages" && section != "dev-packages" {
			continue
		}
		if key, _, ok := tomlAssignment(line); ok {
			packages[normalizeDistributionName(key)] = struct{}{}
		}
	}
	return packages
}

func tomlAssignment(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
	return key, strings.TrimSpace(parts[1]), key != ""
}

func stripConfigComment(line string, marker byte) string {
	quote := byte(0)
	escaped := false
	for index := 0; index < len(line); index++ {
		char := line[index]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if char == '\'' || char == '"' {
			if quote == 0 {
				quote = char
			} else if quote == char {
				quote = 0
			}
			continue
		}
		if quote == 0 && char == marker {
			if marker != '/' || index+1 < len(line) && line[index+1] == '/' {
				return line[:index]
			}
		}
	}
	return line
}

func containsUnquotedByte(value string, target byte) bool {
	quote := byte(0)
	escaped := false
	for index := 0; index < len(value); index++ {
		char := value[index]
		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if char == '\'' || char == '"' {
			if quote == 0 {
				quote = char
			} else if quote == char {
				quote = 0
			}
			continue
		}
		if quote == 0 && char == target {
			return true
		}
	}
	return false
}

func normalizeRustCrate(crate string) string {
	crate = strings.TrimPrefix(strings.TrimSpace(crate), "r#")
	crate = strings.SplitN(crate, "::", 2)[0]
	return strings.ReplaceAll(crate, "-", "_")
}

func isRustBuiltin(crate string) bool {
	switch crate {
	case "std", "core", "alloc", "crate", "self", "super":
		return true
	default:
		return false
	}
}

func rustImportResolved(gitRoot, sourcePath, crate string) (bool, bool) {
	manifest := nearestFile(gitRoot, sourcePath, "Cargo.toml")
	if manifest == "" {
		return false, false
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return false, true
	}
	localCrate, dependencies := cargoPackages(string(data))
	if crate == localCrate {
		return true, true
	}
	_, ok := dependencies[crate]
	return ok, true
}

func cargoPackages(data string) (string, map[string]struct{}) {
	dependencies := map[string]struct{}{}
	section := ""
	localCrate := ""
	for rawLine := range strings.SplitSeq(data, "\n") {
		line := strings.TrimSpace(stripConfigComment(rawLine, '#'))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			if dependency := cargoDependencyTable(section); dependency != "" {
				dependencies[dependency] = struct{}{}
			}
			continue
		}
		key, value, ok := tomlAssignment(line)
		if !ok {
			continue
		}
		if section == "package" && key == "name" {
			localCrate = normalizeRustCrate(strings.Trim(value, `"'`))
			continue
		}
		if isCargoDependencySection(section) && value != "" {
			dependencies[normalizeRustCrate(key)] = struct{}{}
		}
	}
	return localCrate, dependencies
}

func isCargoDependencySection(section string) bool {
	return section == "dependencies" || section == "dev-dependencies" || section == "build-dependencies" ||
		strings.HasPrefix(section, "target.") && (strings.HasSuffix(section, ".dependencies") || strings.HasSuffix(section, ".dev-dependencies") || strings.HasSuffix(section, ".build-dependencies"))
}

func cargoDependencyTable(section string) string {
	for _, prefix := range []string{"dependencies.", "dev-dependencies.", "build-dependencies."} {
		if dependency, ok := strings.CutPrefix(section, prefix); ok {
			return normalizeRustCrate(strings.Trim(dependency, `"'`))
		}
	}
	if strings.HasPrefix(section, "target.") {
		for _, marker := range []string{".dependencies.", ".dev-dependencies.", ".build-dependencies."} {
			if _, dependency, ok := strings.Cut(section, marker); ok {
				return normalizeRustCrate(strings.Trim(dependency, `"'`))
			}
		}
	}
	return ""
}

func nearestFiles(gitRoot, sourcePath string, names []string) []string {
	dir := filepath.Dir(filepath.Join(gitRoot, sourcePath))
	for {
		var manifests []string
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if isFile(candidate) {
				manifests = append(manifests, candidate)
			}
		}
		if len(manifests) > 0 || dir == gitRoot {
			return manifests
		}
		dir = filepath.Dir(dir)
	}
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

func nearestFile(gitRoot, sourcePath, name string) string {
	dir := filepath.Dir(filepath.Join(gitRoot, sourcePath))
	for {
		candidate := filepath.Join(dir, name)
		if isFile(candidate) {
			return candidate
		}
		if dir == gitRoot {
			return ""
		}
		dir = filepath.Dir(dir)
	}
}

func packageJSONDeclares(pkg, manifest string) bool {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return false
	}
	var packageJSON struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return false
	}
	for _, dependencies := range []map[string]string{
		packageJSON.Dependencies,
		packageJSON.DevDependencies,
		packageJSON.PeerDependencies,
		packageJSON.OptionalDependencies,
	} {
		if _, ok := dependencies[pkg]; ok {
			return true
		}
	}
	return false
}

func importMatchesPathAlias(gitRoot, sourcePath, specifier string) bool {
	dir := filepath.Dir(filepath.Join(gitRoot, sourcePath))
	for {
		foundConfig := false
		config := filepath.Join(dir, "tsconfig.json")
		if !isFile(config) {
			config = filepath.Join(dir, "jsconfig.json")
		}
		for _, config := range []string{config, filepath.Join(dir, ".svelte-kit", "tsconfig.json")} {
			if isFile(config) {
				foundConfig = true
				if configMatchesPathAlias(config, specifier) {
					return true
				}
			}
		}
		if foundConfig {
			return false
		}
		if dir == gitRoot {
			return false
		}
		dir = filepath.Dir(dir)
	}
}

func configMatchesPathAlias(configPath, specifier string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var config struct {
		CompilerOptions struct {
			Paths map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}
	for alias, targets := range config.CompilerOptions.Paths {
		if len(targets) > 0 && matchesAlias(specifier, alias) {
			return true
		}
	}
	return false
}

func matchesAlias(specifier, alias string) bool {
	if strings.Count(alias, "*") == 0 {
		return specifier == alias
	}
	if strings.Count(alias, "*") != 1 {
		return false
	}
	parts := strings.SplitN(alias, "*", 2)
	return len(specifier) >= len(parts[0])+len(parts[1]) &&
		strings.HasPrefix(specifier, parts[0]) && strings.HasSuffix(specifier, parts[1])
}
