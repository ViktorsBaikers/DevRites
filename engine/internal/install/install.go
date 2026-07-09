package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/iohooks"
	"github.com/devrites/devrites/internal/version"
)

const ManifestName = devritespaths.ManifestName
const defaultRepo = "ViktorsBaikers/DevRites"

type Mode string

const (
	ModeInstall   Mode = "install"
	ModeUpdate    Mode = "update"
	ModeUninstall Mode = "uninstall"
)

type Options struct {
	Mode        Mode
	Target      string
	PayloadDir  string
	SourceDir   string
	DryRun      bool
	Force       bool
	WithSkills  bool
	WithAgents  bool
	WithCodex   bool
	WithBinary  bool
	KeepBinary  bool
	AliasMode   string
	UpdateCheck bool
	UpdateTo    string
	AllowPre    bool
	Stdout      io.Writer
	Stderr      io.Writer
}

type stats struct {
	installed int
	overwrote int
	skipped   int
	pruned    int
	removed   int
	missing   int
}

type runner struct {
	opts      Options
	target    string
	payload   string
	payloadFS fs.FS
	source    string
	manifest  []string
	records   map[string]string
	prev      map[string]bool
	stats     stats
}

func DefaultOptions(mode Mode) Options {
	return Options{
		Mode:       mode,
		Target:     ".",
		WithSkills: true,
		WithAgents: true,
		WithCodex:  true,
		WithBinary: true,
		AliasMode:  "safe",
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	}
}

func Run(args []string, stdout, stderr io.Writer, mode Mode) int {
	opts := DefaultOptions(mode)
	opts.Stdout = stdout
	opts.Stderr = stderr
	if err := parseArgs(args, &opts); err != nil {
		if errors.Is(err, errHelp) {
			fmt.Fprint(stdout, usage(mode))
			return 0
		}
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return 2
	}
	if err := Apply(opts); err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return 1
	}
	return 0
}

var errHelp = errors.New("help")

func parseArgs(args []string, opts *Options) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func(name string) (string, error) {
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s needs a value", name)
			}
			return args[i], nil
		}
		switch {
		case arg == "--target":
			v, err := next(arg)
			if err != nil {
				return err
			}
			opts.Target = v
		case strings.HasPrefix(arg, "--target="):
			opts.Target = strings.TrimPrefix(arg, "--target=")
		case arg == "--payload-dir":
			v, err := next(arg)
			if err != nil {
				return err
			}
			opts.PayloadDir = v
		case strings.HasPrefix(arg, "--payload-dir="):
			opts.PayloadDir = strings.TrimPrefix(arg, "--payload-dir=")
		case arg == "--source-dir":
			v, err := next(arg)
			if err != nil {
				return err
			}
			opts.SourceDir = v
		case strings.HasPrefix(arg, "--source-dir="):
			opts.SourceDir = strings.TrimPrefix(arg, "--source-dir=")
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--force":
			opts.Force = true
		case arg == "--no-codex":
			opts.WithCodex = false
		case arg == "--no-agents":
			opts.WithAgents = false
		case arg == "--no-skills":
			opts.WithSkills = false
		case arg == "--no-binary":
			opts.WithBinary = false
		case arg == "--keep-binary":
			opts.KeepBinary = true
		case arg == "--no-rules" || arg == "--rules-only":
			fmt.Fprintf(opts.Stderr, "warning: %s is deprecated and now a no-op - DevRites engineering standards ship inside the devrites-lib skill.\n", arg)
		case arg == "--no-short-aliases":
			opts.AliasMode = "off"
		case arg == "--short-aliases":
			opts.AliasMode = "off"
			fmt.Fprintln(opts.Stderr, "warning: --short-aliases with no value is a no-op; use --short-aliases=all.")
		case arg == "--short-aliases=all":
			opts.AliasMode = "all"
		case arg == "--check":
			opts.UpdateCheck = true
		case arg == "--pre":
			opts.AllowPre = true
		case arg == "--to":
			v, err := next(arg)
			if err != nil {
				return err
			}
			opts.UpdateTo = v
		case strings.HasPrefix(arg, "--to="):
			opts.UpdateTo = strings.TrimPrefix(arg, "--to=")
		case arg == "-h" || arg == "--help":
			return errHelp
		default:
			return fmt.Errorf("unknown option: %s (try --help)", arg)
		}
	}
	return nil
}

func usage(mode Mode) string {
	switch mode {
	case ModeUninstall:
		return "usage: devrites-engine uninstall [--target DIR] [--dry-run] [--keep-binary]\n"
	case ModeUpdate:
		return "usage: devrites-engine update [--target DIR] [--dry-run] [--force] [--check] [--to TAG] [install flags]\n"
	default:
		return "usage: devrites-engine install [--target DIR] [--dry-run] [--force] [--no-codex] [--no-agents] [--no-skills] [--no-binary] [--short-aliases=all]\n"
	}
}

func Apply(opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Mode == "" {
		opts.Mode = ModeInstall
	}
	if opts.Mode == ModeUpdate {
		return runUpdate(opts)
	}
	r, err := newRunner(opts)
	if err != nil {
		return err
	}
	if opts.Mode == ModeUninstall {
		return r.uninstall()
	}
	return r.install()
}

func newRunner(opts Options) (*runner, error) {
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("target does not exist: %s", opts.Target)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("target is not a directory: %s", opts.Target)
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return nil, err
	}
	if isGlobalTarget(target) {
		return nil, fmt.Errorf("refusing to target a global agent home. DevRites is project-local; choose a project directory")
	}
	source := opts.SourceDir
	if source == "" {
		source = inferSourceDir()
	}
	if source != "" {
		source, _ = filepath.Abs(source)
	}
	payload := opts.PayloadDir
	if payload == "" {
		payload = inferPayloadDir(source)
	}
	if payload != "" {
		payload, _ = filepath.Abs(payload)
	}
	r := &runner{opts: opts, target: target, payload: payload, payloadFS: os.DirFS(payload), source: source, prev: map[string]bool{}, records: map[string]string{}}
	if opts.Mode != ModeUninstall {
		if err := r.validatePayload(); err != nil {
			return nil, err
		}
	}
	r.prev = readManifest(filepath.Join(target, ManifestName))
	return r, nil
}

func inferSourceDir() string {
	if v := os.Getenv("DEVRITES_SOURCE_DIR"); v != "" {
		return v
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 4; i++ {
			if exists(filepath.Join(dir, "pack")) || exists(filepath.Join(dir, "engine")) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

func inferPayloadDir(source string) string {
	if v := os.Getenv("DEVRITES_HOST_ARTIFACT_DIR"); v != "" {
		return v
	}
	if source != "" {
		if exists(filepath.Join(source, "pack", "generated")) {
			return filepath.Join(source, "pack", "generated")
		}
	}
	return ""
}

func (r *runner) validatePayload() error {
	if r.payload == "" {
		return fmt.Errorf("generated install payload not found; pass --payload-dir or run scripts/build-host-artifacts.sh")
	}
	if err := hostpack.ValidatePayload(r.payloadFS, r.opts.WithCodex); err != nil {
		return fmt.Errorf("generated install payload %s under %s", err, r.payload)
	}
	return nil
}

func validPayload(payload string) bool {
	if payload == "" {
		return false
	}
	return hostpack.ValidPayload(os.DirFS(payload))
}

func (r *runner) install() error {
	fmt.Fprintln(r.opts.Stdout, "DevRites installer")
	fmt.Fprintf(r.opts.Stdout, "  target : %s\n", r.target)
	fmt.Fprintf(r.opts.Stdout, "  payload: %s\n", r.payload)
	fmt.Fprintf(r.opts.Stdout, "  skills : %s\n", yesno(r.opts.WithSkills))
	fmt.Fprintln(r.opts.Stdout, "  standards: ship inside the devrites-lib skill")
	fmt.Fprintf(r.opts.Stdout, "  agents : %s\n", yesno(r.opts.WithAgents))
	fmt.Fprintf(r.opts.Stdout, "  codex  : %s\n", yesno(r.opts.WithCodex))
	fmt.Fprintf(r.opts.Stdout, "  aliases: %s\n", r.opts.AliasMode)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  (dry run - no changes will be made)")
	}
	fmt.Fprintln(r.opts.Stdout)

	for _, tree := range hostpack.InstallTrees(r.opts.WithSkills, r.opts.WithAgents, r.opts.WithCodex) {
		if err := r.installTree(tree.PayloadPrefix, tree.TargetPrefix); err != nil {
			return err
		}
	}
	if r.opts.WithSkills && r.opts.AliasMode == "all" {
		for _, alias := range hostpack.Aliases {
			data, err := hostpack.RenderAliasSkill(alias)
			if err != nil {
				return err
			}
			for _, rel := range hostpack.AliasTargets(alias, r.opts.WithCodex) {
				if err := r.installData(data, rel); err != nil {
					return err
				}
			}
		}
	}
	for _, merge := range hostpack.MarkerMerges(r.opts.WithSkills, r.opts.WithCodex) {
		if err := r.mergeMarkerFile(merge); err != nil {
			return err
		}
	}
	for _, merge := range hostpack.JSONMerges(r.opts.WithSkills, r.opts.WithCodex) {
		if err := r.mergeCodexHooks(merge); err != nil {
			return err
		}
	}
	if r.opts.WithSkills {
		if err := r.seedClaudeSettings(); err != nil {
			return err
		}
	}
	if err := r.seedDevrites(); err != nil {
		return err
	}
	if err := r.pruneDropped(); err != nil {
		return err
	}
	if !r.opts.DryRun {
		if err := r.writeManifest(); err != nil {
			return err
		}
	}
	if err := r.installBinary(); err != nil {
		return err
	}

	fmt.Fprintln(r.opts.Stdout)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "DevRites install plan complete (dry run)")
	} else {
		fmt.Fprintln(r.opts.Stdout, "DevRites installed")
	}
	fmt.Fprintf(r.opts.Stdout, "  installed: %d   overwritten: %d   skipped(conflict): %d   pruned: %d\n", r.stats.installed, r.stats.overwrote, r.stats.skipped, r.stats.pruned)
	return nil
}

func (r *runner) installTree(srcPrefix, relPrefix string) error {
	return fs.WalkDir(r.payloadFS, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return err
		}
		return r.installFile(path, filepath.ToSlash(filepath.Join(relPrefix, rel)))
	})
}

func (r *runner) installFile(src, rel string) error {
	data, err := fs.ReadFile(r.payloadFS, src)
	if err != nil {
		return err
	}
	return r.installData(data, rel)
}

func (r *runner) installData(data []byte, rel string) error {
	dest := filepath.Join(r.target, filepath.FromSlash(rel))
	action := "install"
	if exists(dest) {
		switch {
		case r.prev[rel]:
			action = "overwrite"
		case r.opts.Force:
			action = "overwrite(force)"
		default:
			if r.opts.DryRun {
				fmt.Fprintf(r.opts.Stdout, "  [skip] %s (exists, not DevRites-managed)\n", rel)
			} else {
				fmt.Fprintf(r.opts.Stderr, "warning: skip %s (exists, not DevRites-managed; use --force to overwrite)\n", rel)
			}
			r.stats.skipped++
			return nil
		}
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
	} else if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", rel, err)
	}
	r.addManifest(rel)
	r.addInstallRecord(rel, data)
	if action == "install" {
		r.stats.installed++
	} else {
		r.stats.overwrote++
	}
	return nil
}

func (r *runner) addManifest(rel string) {
	r.manifest = append(r.manifest, filepath.ToSlash(rel))
}

func (r *runner) addInstallRecord(rel string, data []byte) {
	if r.records == nil {
		r.records = map[string]string{}
	}
	sum := sha256.Sum256(data)
	r.records[filepath.ToSlash(rel)] = fmt.Sprintf("sha256:%x", sum[:])
}

func (r *runner) installMarker(rel, text string) error {
	return r.installData([]byte(text+"\n"), rel)
}

func (r *runner) mergeMarkerFile(merge hostpack.MarkerMerge) error {
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return err
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if r.opts.DryRun {
		verb := "create DevRites block"
		if exists(dest) {
			current, _ := os.ReadFile(dest)
			if bytes.Contains(current, []byte(merge.Begin)) {
				verb = "refresh DevRites block"
			} else {
				verb = "append DevRites block"
			}
		}
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (%s)\n", merge.TargetRel, verb)
	} else {
		next := block
		if current, err := os.ReadFile(dest); err == nil {
			next = hostpack.MergeMarkerBlock(current, block, merge.Begin, merge.End)
		}
		if err := fsutil.WriteFileAtomic(dest, next, 0o644); err != nil {
			return err
		}
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) mergeCodexHooks(merge hostpack.JSONMerge) error {
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	devrites, err := readJSONFS(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s\n", merge.DryRunText)
		return r.installMarker(merge.MarkerRel, merge.MarkerText)
	}
	next := devrites
	if current, err := readJSON(dest); err == nil {
		next = mergeHooks(stripDevritesHooks(current), devrites)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
		return err
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) seedClaudeSettings() error {
	rel := ".claude/settings.json"
	dest := filepath.Join(r.target, rel)
	if exists(dest) {
		fmt.Fprintln(r.opts.Stderr, "warning: skip .claude/settings.json (exists) - existing settings are preserved.")
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  [seed] .claude/settings.json (DevRites hooks - preserved on uninstall/update)")
		return nil
	}
	data, err := fs.ReadFile(r.payloadFS, "claude/settings.json")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(dest, data, 0o644)
}

func (r *runner) seedDevrites() error {
	readme, err := hostpack.RenderDevritesReadme()
	if err != nil {
		return err
	}
	if err := r.installData(readme, ".devrites/README.md"); err != nil {
		return err
	}
	active := filepath.Join(r.target, ".devrites", "ACTIVE")
	if exists(active) {
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  [seed] .devrites/ACTIVE (runtime state - preserved on uninstall)")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(active, os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	return f.Close()
}

func (r *runner) pruneDropped() error {
	if len(r.prev) == 0 {
		return nil
	}
	next := map[string]bool{}
	for _, rel := range r.manifest {
		next[rel] = true
	}
	keys := make([]string, 0, len(r.prev))
	for rel := range r.prev {
		keys = append(keys, rel)
	}
	sort.Strings(keys)
	for _, rel := range keys {
		if next[rel] || hostpack.PreserveOnPrune(rel) {
			continue
		}
		dead := filepath.Join(r.target, filepath.FromSlash(rel))
		if !exists(dead) {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [prune] %s (dropped from pack)\n", rel)
		} else {
			if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
				if merge.TargetRel == hostpack.CodexHooksMerge.TargetRel {
					_ = r.stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)))
				} else {
					_ = stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End)
				}
			}
			if err := os.Remove(dead); err != nil && !os.IsNotExist(err) {
				return err
			}
			pruneEmptyDirs(filepath.Dir(dead), r.target)
			fmt.Fprintf(r.opts.Stdout, "  [prune] %s\n", rel)
		}
		r.stats.pruned++
	}
	return nil
}

func (r *runner) writeManifest() error {
	sort.Strings(r.manifest)
	r.manifest = unique(r.manifest)
	var b strings.Builder
	b.WriteString("# DevRites install manifest - do not edit by hand.\n")
	b.WriteString("# Generated " + time.Now().UTC().Format(time.RFC3339) + ". Uninstall removes exactly these paths.\n")
	b.WriteString("# devrites-version: " + installedVersion(r.source) + "\n")
	b.WriteString("# devrites-flags: " + r.flagsString() + "\n")
	b.WriteString("# managed-records: source=npx payload=pack/generated format=rel sha256\n")
	for _, rel := range r.manifest {
		if hash := r.records[rel]; hash != "" {
			b.WriteString("# managed: " + rel + " " + hash + "\n")
		}
	}
	for _, rel := range r.manifest {
		b.WriteString(rel + "\n")
	}
	return fsutil.WriteFileAtomic(filepath.Join(r.target, ManifestName), []byte(b.String()), 0o644)
}

func (r *runner) flagsString() string {
	var flags []string
	if !r.opts.WithSkills {
		flags = append(flags, "--no-skills")
	}
	if !r.opts.WithAgents {
		flags = append(flags, "--no-agents")
	}
	if !r.opts.WithCodex {
		flags = append(flags, "--no-codex")
	}
	if !r.opts.WithBinary {
		flags = append(flags, "--no-binary")
	}
	if r.opts.AliasMode == "off" {
		flags = append(flags, "--no-short-aliases")
	}
	if r.opts.AliasMode == "all" {
		flags = append(flags, "--short-aliases=all")
	}
	return strings.Join(flags, " ")
}

func (r *runner) uninstall() error {
	mf := filepath.Join(r.target, ManifestName)
	entries := readManifestList(mf)
	if len(entries) == 0 {
		return fmt.Errorf("no DevRites manifest at %s - nothing to uninstall", mf)
	}
	fmt.Fprintln(r.opts.Stdout, "DevRites uninstaller")
	fmt.Fprintf(r.opts.Stdout, "  target  : %s\n", r.target)
	fmt.Fprintf(r.opts.Stdout, "  manifest: %s\n", mf)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  (dry run - no changes)")
	}
	fmt.Fprintln(r.opts.Stdout)

	for _, rel := range entries {
		merge, ok := hostpack.ManagedMergeForMarker(rel)
		if !ok {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [merge-remove] %s\n", merge.DryRun)
			continue
		}
		if merge.TargetRel == hostpack.CodexHooksMerge.TargetRel {
			if err := r.stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))); err != nil {
				return err
			}
			continue
		}
		if err := stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End); err != nil {
			return err
		}
	}

	var dirs []string
	for _, rel := range entries {
		if !hostpack.ShouldRemoveOnUninstall(rel, entries) {
			continue
		}
		dest := filepath.Join(r.target, filepath.FromSlash(rel))
		if exists(dest) {
			if r.opts.DryRun {
				fmt.Fprintf(r.opts.Stdout, "  [remove] %s\n", rel)
			} else if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}
			dirs = append(dirs, filepath.Dir(dest))
			r.stats.removed++
		} else {
			r.stats.missing++
		}
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [remove] %s\n", ManifestName)
	} else {
		_ = os.Remove(mf)
		dirs = append(dirs, filepath.Dir(mf))
		for _, d := range dirs {
			pruneEmptyDirs(d, r.target)
		}
	}
	if err := r.removeBinary(); err != nil {
		return err
	}
	fmt.Fprintln(r.opts.Stdout)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "DevRites uninstall plan complete (dry run)")
	} else {
		fmt.Fprintln(r.opts.Stdout, "DevRites uninstalled")
	}
	fmt.Fprintf(r.opts.Stdout, "  removed: %d   already-absent: %d\n", r.stats.removed, r.stats.missing)
	if exists(filepath.Join(r.target, ".devrites", "work")) {
		fmt.Fprintln(r.opts.Stdout, "  kept .devrites/work/ (your feature data)")
	}
	if exists(filepath.Join(r.target, ".devrites", "ACTIVE")) {
		fmt.Fprintln(r.opts.Stdout, "  kept .devrites/ACTIVE (active-feature cursor)")
	}
	return nil
}

func (r *runner) stripHooksPath(path string) error {
	current, err := readJSON(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	next := stripDevritesHooks(current)
	if len(next) == 0 {
		return os.Remove(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if string(bytes.TrimSpace(data)) == "{}" {
		return os.Remove(path)
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func runUpdate(opts Options) error {
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return err
	}
	mf := filepath.Join(target, ManifestName)
	if !exists(mf) {
		return fmt.Errorf("no DevRites install found at %s (missing %s)", target, ManifestName)
	}
	installed := manifestHeader(mf, "devrites-version")
	if installed == "" {
		installed = "unknown"
	}
	latestTag, err := resolveUpdateTag(opts)
	if err != nil {
		return err
	}
	latest := strings.TrimPrefix(latestTag, "v")
	fmt.Fprintln(opts.Stdout, "DevRites update")
	fmt.Fprintf(opts.Stdout, "  project:   %s\n", target)
	fmt.Fprintf(opts.Stdout, "  installed: %s\n", installed)
	fmt.Fprintf(opts.Stdout, "  latest:    %s\n", latest)
	if opts.UpdateCheck {
		if installed == latest || strings.TrimPrefix(installed, "v") == latest {
			fmt.Fprintln(opts.Stdout, "up to date.")
			return nil
		}
		return fmt.Errorf("update available: %s -> %s", installed, latest)
	}
	if !opts.Force && (installed == latest || strings.TrimPrefix(installed, "v") == latest) {
		fmt.Fprintln(opts.Stdout, "already up to date (use --force to reinstall).")
		return nil
	}
	source, cleanup, err := acquireUpdateBundle(latestTag, opts.Stdout)
	if err != nil {
		return err
	}
	defer cleanup()
	payload := filepath.Join(source, "pack", "generated")
	if !validPayload(payload) {
		if err := buildHostArtifacts(source, payload); err != nil {
			return err
		}
	}
	installFlags := manifestHeader(mf, "devrites-flags")
	next := DefaultOptions(ModeInstall)
	next.Stdout = opts.Stdout
	next.Stderr = opts.Stderr
	if installFlags != "" {
		if err := parseArgs(strings.Fields(installFlags), &next); err != nil {
			return err
		}
	}
	next.Target = target
	next.PayloadDir = payload
	next.SourceDir = source
	next.Force = true
	if opts.DryRun {
		next.DryRun = true
	}
	return Apply(next)
}

func resolveUpdateTag(opts Options) (string, error) {
	if opts.UpdateTo != "" {
		return normalizeTag(opts.UpdateTo), nil
	}
	if os.Getenv("DEVRITES_UPDATE_BUNDLE") != "" && opts.SourceDir != "" {
		if v := installedVersion(opts.SourceDir); v != "" && v != "unknown" {
			return normalizeTag(v), nil
		}
	}
	repo := os.Getenv("DEVRITES_REPO")
	if repo == "" {
		repo = defaultRepo
	}
	endpoint := "latest"
	if opts.AllowPre {
		endpoint = ""
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/%s", repo, endpoint)
	if opts.AllowPre {
		var releases []struct {
			TagName string `json:"tag_name"`
		}
		if err := iohooks.FetchJSON(url, &releases); err != nil {
			return "", fmt.Errorf("resolve latest release: %w", err)
		}
		for _, rel := range releases {
			if rel.TagName != "" {
				return normalizeTag(rel.TagName), nil
			}
		}
		return "", fmt.Errorf("resolve latest release: no releases found")
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := iohooks.FetchJSON(url, &release); err != nil {
		return "", fmt.Errorf("resolve latest release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("resolve latest release: empty tag")
	}
	return normalizeTag(release.TagName), nil
}

func acquireUpdateBundle(tag string, stdout io.Writer) (string, func(), error) {
	tmp, err := os.MkdirTemp("", "devrites-update-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	artifact := fmt.Sprintf("devrites-%s.tar.gz", tag)
	tarball := filepath.Join(tmp, artifact)
	if bundle := os.Getenv("DEVRITES_UPDATE_BUNDLE"); bundle != "" {
		if !exists(bundle) {
			cleanup()
			return "", func() {}, fmt.Errorf("DEVRITES_UPDATE_BUNDLE not readable: %s", bundle)
		}
		fmt.Fprintf(stdout, "  bundle:   %s\n", bundle)
		return extractUpdateBundle(bundle, tmp, cleanup)
	}

	repo := os.Getenv("DEVRITES_REPO")
	if repo == "" {
		repo = defaultRepo
	}
	releaseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, artifact)
	if err := iohooks.DownloadFile(releaseURL, tarball); err == nil {
		sumPath := tarball + ".sha256"
		if err := iohooks.DownloadFile(releaseURL+".sha256", sumPath); err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("could not verify release artifact %s: missing checksum: %w", artifact, err)
		}
		if err := verifySHA256(tarball, sumPath); err != nil {
			cleanup()
			return "", func() {}, fmt.Errorf("could not verify release artifact %s: %w", artifact, err)
		}
		fmt.Fprintf(stdout, "  bundle:   %s\n", releaseURL)
		return extractUpdateBundle(tarball, tmp, cleanup)
	}

	sourceURL := fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.tar.gz", repo, tag)
	sourceTarball := filepath.Join(tmp, "src-"+tag+".tar.gz")
	if err := iohooks.DownloadFile(sourceURL, sourceTarball); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("download update bundle: %w", err)
	}
	fmt.Fprintf(stdout, "  bundle:   %s\n", sourceURL)
	return extractUpdateBundle(sourceTarball, tmp, cleanup)
}

func verifySHA256(path, sumPath string) error {
	data, err := os.ReadFile(sumPath)
	if err != nil {
		return err
	}
	want := strings.Fields(string(data))
	if len(want) == 0 {
		return fmt.Errorf("empty checksum file")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	got := fmt.Sprintf("%x", sha256.Sum256(body))
	if got != want[0] {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, want[0])
	}
	return nil
}

func extractUpdateBundle(tarball, tmp string, cleanup func()) (string, func(), error) {
	if err := extractTarGz(tarball, tmp); err != nil {
		cleanup()
		return "", func() {}, err
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(tmp, entry.Name())
		if exists(filepath.Join(dir, "install.sh")) {
			return dir, cleanup, nil
		}
	}
	cleanup()
	return "", func() {}, fmt.Errorf("extracted update bundle has no install.sh")
}

func extractTarGz(tarball, dest string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	root, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(hdr.Name)
		if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
			return fmt.Errorf("unsafe path in update bundle: %s", hdr.Name)
		}
		target := filepath.Join(root, name)
		if !strings.HasPrefix(target, root+string(os.PathSeparator)) && target != root {
			return fmt.Errorf("unsafe path in update bundle: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, fs.FileMode(hdr.Mode)&0o777); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			continue
		}
	}
}

func buildHostArtifacts(source, payload string) error {
	script := filepath.Join(source, "scripts", "build-host-artifacts.sh")
	if !exists(script) {
		return fmt.Errorf("generated install payload missing at %s and no builder found at %s", payload, script)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = source
	cmd.Env = append(os.Environ(), "DEVRITES_HOST_ARTIFACT_DIR="+payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build generated install payload: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" || strings.HasPrefix(tag, "v") {
		return tag
	}
	return "v" + tag
}

func stripMarkerPath(path, begin, end string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var out strings.Builder
	inBlock := false
	for _, line := range strings.SplitAfter(string(data), "\n") {
		trim := strings.TrimSuffix(line, "\n")
		trim = strings.TrimSuffix(trim, "\r")
		switch trim {
		case begin:
			inBlock = true
			continue
		case end:
			inBlock = false
			continue
		}
		if !inBlock {
			out.WriteString(line)
		}
	}
	if strings.TrimSpace(out.String()) == "" {
		return os.Remove(path)
	}
	return fsutil.WriteFileAtomic(path, []byte(out.String()), 0o644)
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeJSON(data)
}

func readJSONFS(src fs.FS, path string) (map[string]any, error) {
	data, err := fs.ReadFile(src, path)
	if err != nil {
		return nil, err
	}
	return decodeJSON(data)
}

func decodeJSON(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func isDevritesHook(v any) bool {
	data, _ := json.Marshal(v)
	s := string(data)
	return strings.Contains(s, "devrites-engine hook ") ||
		strings.Contains(s, ".codex/hooks/devrites-") ||
		strings.Contains(s, "DevRites:") ||
		strings.Contains(s, "DEVRITES_")
}

func stripDevritesHooks(config map[string]any) map[string]any {
	next := map[string]any{}
	for k, v := range config {
		if k == "$comment" {
			if s, ok := v.(string); ok && strings.Contains(s, "DevRites hooks for Codex") {
				continue
			}
		}
		next[k] = v
	}
	rawHooks, ok := next["hooks"].(map[string]any)
	if !ok {
		delete(next, "hooks")
		return next
	}
	hooks := map[string]any{}
	for event, entries := range rawHooks {
		arr, ok := entries.([]any)
		if !ok {
			hooks[event] = entries
			continue
		}
		var kept []any
		for _, entry := range arr {
			if !isDevritesHook(entry) {
				kept = append(kept, entry)
			}
		}
		if len(kept) > 0 {
			hooks[event] = kept
		}
	}
	if len(hooks) == 0 {
		delete(next, "hooks")
	} else {
		next["hooks"] = hooks
	}
	return next
}

func mergeHooks(existing, devrites map[string]any) map[string]any {
	hooks, _ := existing["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	devHooks, _ := devrites["hooks"].(map[string]any)
	for event, entries := range devHooks {
		arr, _ := hooks[event].([]any)
		devArr, _ := entries.([]any)
		hooks[event] = append(arr, devArr...)
	}
	existing["hooks"] = hooks
	return existing
}

func (r *runner) installBinary() error {
	if !r.opts.WithBinary || os.Getenv("DEVRITES_NO_BINARY") == "1" {
		fmt.Fprintln(r.opts.Stdout, "  engine binary: skipped (--no-binary); hooks fail open.")
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  would install the global devrites-engine control-plane binary")
		return nil
	}
	tag := os.Getenv("DEVRITES_REF")
	if tag == "" {
		tag = "v" + strings.TrimPrefix(installedVersion(r.source), "v")
	}
	incoming := strings.TrimPrefix(tag, "v")
	dest := binaryDest()
	if exists(dest) {
		if ev := engineVersion(dest); semverLike(ev) && semverLike(incoming) && verGT(ev, incoming) {
			fmt.Fprintf(r.opts.Stderr, "warning: engine binary: installed %s is newer than %s - refusing to downgrade (kept).\n", ev, tag)
			return nil
		}
	}
	staged, cleanup, err := r.acquireBinary(tag)
	if err != nil {
		fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v - hooks fail open until you install it.\n", err)
		return nil
	}
	defer cleanup()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v - hooks fail open until you install it.\n", err)
		return nil
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v - hooks fail open until you install it.\n", err)
		return nil
	}
	if err := fsutil.WriteFileAtomic(dest, data, 0o755); err != nil {
		fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v - hooks fail open until you install it.\n", err)
		return nil
	}
	_ = os.Chmod(dest, 0o755)
	fmt.Fprintf(r.opts.Stdout, "  engine binary: installed %s\n", dest)
	if !binaryReachableFromPATH(dest) {
		fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %s is not on PATH; installed hooks will fail open until devrites-engine is reachable.\n", dest)
	}
	return nil
}

func (r *runner) acquireBinary(tag string) (string, func(), error) {
	if v := os.Getenv("DEVRITES_ENGINE_CLI"); v != "" {
		if exists(v) {
			return v, func() {}, nil
		}
		return "", func() {}, fmt.Errorf("DEVRITES_ENGINE_CLI points to a missing binary: %s", v)
	}
	tmp, err := os.MkdirTemp("", "devrites-engine-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	staged := filepath.Join(tmp, "devrites-engine")
	if r.source != "" && exists(filepath.Join(r.source, "engine", "go.mod")) {
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w -X github.com/devrites/devrites/internal/version.Version="+tag, "-o", staged, ".")
		cmd.Dir = filepath.Join(r.source, "engine")
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", cleanup, fmt.Errorf("could not build from source: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return staged, cleanup, nil
	}
	return "", cleanup, fmt.Errorf("could not obtain a binary (no DEVRITES_ENGINE_CLI handoff, no Go source)")
}

func (r *runner) removeBinary() error {
	if r.opts.KeepBinary {
		fmt.Fprintln(r.opts.Stdout, "  kept the global devrites-engine binary (--keep-binary).")
		return nil
	}
	for _, p := range binaryCandidates() {
		if p == "" || !exists(p) {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [remove] %s (global engine binary)\n", p)
			continue
		}
		if err := os.Remove(p); err == nil {
			fmt.Fprintf(r.opts.Stdout, "  [remove] %s\n", p)
			return nil
		}
	}
	return nil
}

func binaryDest() string {
	if dir := os.Getenv("DEVRITES_BIN_DIR"); dir != "" {
		return filepath.Join(dir, "devrites-engine")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "bin", "devrites-engine")
	}
	return "/usr/local/bin/devrites-engine"
}

func binaryCandidates() []string {
	candidates := []string{}
	if dir := os.Getenv("DEVRITES_BIN_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, "devrites-engine"))
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", "devrites-engine"))
	}
	return append(candidates, "/usr/local/bin/devrites-engine")
}

func binaryReachableFromPATH(dest string) bool {
	if _, err := exec.LookPath("devrites-engine"); err == nil {
		return true
	}
	return pathContainsDir(filepath.Dir(dest))
}

func pathContainsDir(dir string) bool {
	if dir == "" {
		return false
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		want = filepath.Clean(dir)
	}
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == "" {
			continue
		}
		got, err := filepath.Abs(entry)
		if err != nil {
			got = filepath.Clean(entry)
		}
		if got == want {
			return true
		}
	}
	return false
}

func engineVersion(path string) string {
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readManifest(path string) map[string]bool {
	out := map[string]bool{}
	for _, rel := range readManifestList(path) {
		out[rel] = true
	}
	return out
}

func readManifestList(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func manifestHeader(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := "# " + key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func installedVersion(source string) string {
	if source != "" {
		data, err := os.ReadFile(filepath.Join(source, "package.json"))
		if err == nil {
			var pkg struct {
				Version string `json:"version"`
			}
			if json.Unmarshal(data, &pkg) == nil && pkg.Version != "" {
				return pkg.Version
			}
		}
	}
	if version.Version != "" && version.Version != "dev" {
		return strings.TrimPrefix(version.Version, "v")
	}
	return "unknown"
}

func isGlobalTarget(target string) bool {
	home, _ := os.UserHomeDir()
	if target == home {
		return true
	}
	for _, global := range []string{filepath.Join(home, ".claude"), filepath.Join(home, ".codex")} {
		if target == global || strings.HasPrefix(target, global+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func pruneEmptyDirs(start, stop string) {
	for {
		if start == stop || start == filepath.Dir(start) || start == string(os.PathSeparator) {
			return
		}
		if err := os.Remove(start); err != nil {
			return
		}
		start = filepath.Dir(start)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func unique(in []string) []string {
	var out []string
	var last string
	for i, v := range in {
		if i == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}

func yesno(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func semverLike(v string) bool {
	v = strings.TrimPrefix(v, "v")
	return len(v) > 0 && v[0] >= '0' && v[0] <= '9'
}

func verGT(a, b string) bool {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	as := strings.Split(strings.Split(a, "-")[0], ".")
	bs := strings.Split(strings.Split(b, "-")[0], ".")
	for len(as) < 3 {
		as = append(as, "0")
	}
	for len(bs) < 3 {
		bs = append(bs, "0")
	}
	for i := 0; i < 3; i++ {
		var ai, bi int
		fmt.Sscanf(as[i], "%d", &ai)
		fmt.Sscanf(bs[i], "%d", &bi)
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}
	return false
}
