package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/safepath"
	"github.com/devrites/devrites/internal/version"
)

const ManifestName = devritespaths.ManifestName

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
	opts              Options
	target            string
	payload           string
	payloadFS         fs.FS
	source            string
	manifest          []string
	records           map[string]string
	prev              map[string]managedRecord
	preflight         map[string]pathSnapshot
	stats             stats
	requiredBinaryTag string
	preparedBinary    string
}

type managedRecord struct {
	Hash string
}

type pathSnapshot struct {
	missing bool
	hash    string
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
	normalized := make([]string, 0, len(args))
	shortAliasesWithoutValue := false
	for _, arg := range args {
		// flag treats -short-aliases and --short-aliases alike; without this
		// rewrite a bare occurrence would consume the next argument as its value.
		if arg == "--short-aliases" || arg == "-short-aliases" {
			normalized = append(normalized, "--short-aliases=")
			shortAliasesWithoutValue = true
			continue
		}
		normalized = append(normalized, arg)
	}

	flags := flag.NewFlagSet("devrites-engine "+string(opts.Mode), flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	target := flags.String("target", opts.Target, "")
	payloadDir := flags.String("payload-dir", opts.PayloadDir, "")
	sourceDir := flags.String("source-dir", opts.SourceDir, "")
	dryRun := flags.Bool("dry-run", opts.DryRun, "")
	force := flags.Bool("force", opts.Force, "")
	noCodex := flags.Bool("no-codex", !opts.WithCodex, "")
	noAgents := flags.Bool("no-agents", !opts.WithAgents, "")
	noSkills := flags.Bool("no-skills", !opts.WithSkills, "")
	noBinary := flags.Bool("no-binary", !opts.WithBinary, "")
	keepBinary := flags.Bool("keep-binary", opts.KeepBinary, "")
	noRules := flags.Bool("no-rules", false, "")
	rulesOnly := flags.Bool("rules-only", false, "")
	noShortAliases := flags.Bool("no-short-aliases", false, "")
	shortAliases := flags.String("short-aliases", opts.AliasMode, "")
	updateCheck := flags.Bool("check", opts.UpdateCheck, "")

	if err := flags.Parse(normalized); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return errHelp
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unknown option: %s (try --help)", flags.Arg(0))
	}

	opts.Target = *target
	opts.PayloadDir = *payloadDir
	opts.SourceDir = *sourceDir
	opts.DryRun = *dryRun
	opts.Force = *force
	opts.WithCodex = !*noCodex
	opts.WithAgents = !*noAgents
	opts.WithSkills = !*noSkills
	opts.WithBinary = !*noBinary
	opts.KeepBinary = *keepBinary
	opts.UpdateCheck = *updateCheck
	for _, dep := range []struct {
		set  bool
		name string
	}{{*noRules, "--no-rules"}, {*rulesOnly, "--rules-only"}} {
		if dep.set {
			fmt.Fprintf(opts.Stderr, "warning: %s is deprecated and now a no-op - DevRites engineering standards ship inside the devrites-lib skill.\n", dep.name)
		}
	}
	if *noShortAliases {
		opts.AliasMode = "off"
	} else if flagWasSet(flags, "short-aliases") {
		switch {
		case *shortAliases == "" && shortAliasesWithoutValue:
			opts.AliasMode = "off"
			fmt.Fprintln(opts.Stderr, "warning: --short-aliases with no value is a no-op; use --short-aliases=all.")
		case *shortAliases == "all":
			opts.AliasMode = "all"
		default:
			return fmt.Errorf("unknown option: --short-aliases=%s (try --help)", *shortAliases)
		}
	}
	return nil
}

func flagWasSet(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func usage(mode Mode) string {
	switch mode {
	case ModeUninstall:
		return "usage: devrites-engine uninstall [--target DIR] [--dry-run] [--force] [--keep-binary]\n"
	case ModeUpdate:
		return "usage: devrites-engine update [--target DIR] [--source-dir DIR] [--payload-dir DIR] [--dry-run] [--force] [--check] [install flags]\n"
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
		return fmt.Errorf("prepare installer: %w", err)
	}
	if opts.Mode == ModeUninstall {
		return r.uninstall()
	}
	return r.install()
}

func newRunner(opts Options) (*runner, error) {
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return nil, fmt.Errorf("resolve target %s: %w", opts.Target, err)
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
		return nil, fmt.Errorf("resolve symlinks for %s: %w", target, err)
	}
	if isGlobalTarget(target) {
		return nil, fmt.Errorf("refusing to target a global agent home. DevRites is project-local; choose a project directory")
	}
	source := opts.SourceDir
	if source == "" {
		source = inferSourceDir()
	}
	if source != "" {
		source, err = filepath.Abs(source)
		if err != nil {
			return nil, fmt.Errorf("resolve source %s: %w", opts.SourceDir, err)
		}
	}
	payload := opts.PayloadDir
	if payload == "" {
		payload = inferPayloadDir(source)
	}
	if payload != "" {
		payload, err = filepath.Abs(payload)
		if err != nil {
			return nil, fmt.Errorf("resolve payload %s: %w", opts.PayloadDir, err)
		}
	}
	r := &runner{
		opts:      opts,
		target:    target,
		payload:   payload,
		payloadFS: os.DirFS(payload),
		source:    source,
		prev:      map[string]managedRecord{},
		records:   map[string]string{},
		preflight: map[string]pathSnapshot{},
	}
	if opts.Mode != ModeUninstall {
		if err := r.validatePayload(); err != nil {
			return nil, fmt.Errorf("validate payload: %w", err)
		}
	}
	r.prev, err = readManifest(filepath.Join(target, ManifestName))
	if err != nil {
		return nil, err
	}
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
		return fmt.Errorf("generated install payload under %s: %w", r.payload, err)
	}
	return nil
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
	if r.preparedBinary != "" {
		if _, err := verifyEngineBinary(r.preparedBinary, r.binaryTag(), 30*time.Second); err != nil {
			return fmt.Errorf("verify staged engine binary %s: %w", r.preparedBinary, err)
		}
	}
	if err := r.preflightInstall(); err != nil {
		return err
	}

	for _, tree := range hostpack.InstallTrees(r.opts.WithSkills, r.opts.WithAgents, r.opts.WithCodex) {
		if err := r.installTree(tree.PayloadPrefix, tree.TargetPrefix); err != nil {
			return fmt.Errorf("install tree %s: %w", tree.TargetPrefix, err)
		}
	}
	if r.opts.WithSkills && r.opts.AliasMode == "all" {
		for _, alias := range hostpack.Aliases {
			data, err := hostpack.RenderAliasSkill(alias)
			if err != nil {
				return fmt.Errorf("render alias skill %s: %w", alias.Name, err)
			}
			for _, rel := range hostpack.AliasTargets(alias, r.opts.WithCodex) {
				if err := r.installData(data, rel); err != nil {
					return fmt.Errorf("install alias: %w", err)
				}
			}
		}
	}
	if r.opts.WithSkills && r.opts.WithCodex {
		if err := r.mergeMarkerFile(hostpack.CodexAgentsMerge); err != nil {
			return fmt.Errorf("merge %s: %w", hostpack.CodexAgentsMerge.TargetRel, err)
		}
		if err := r.mergeCodexConfig(); err != nil {
			return fmt.Errorf("merge %s: %w", hostpack.CodexConfigMerge.TargetRel, err)
		}
	}
	if r.opts.WithSkills {
		if err := r.seedClaudeSettings(); err != nil {
			return fmt.Errorf("seed claude settings: %w", err)
		}
	}
	if err := r.seedDevrites(); err != nil {
		return fmt.Errorf("seed .devrites: %w", err)
	}
	if err := r.pruneDropped(); err != nil {
		return fmt.Errorf("prune dropped files: %w", err)
	}
	if !r.opts.DryRun {
		if err := r.writeManifest(); err != nil {
			return fmt.Errorf("write manifest: %w", err)
		}
	}
	if err := r.installBinary(); err != nil {
		return fmt.Errorf("install engine binary: %w", err)
	}

	fmt.Fprintln(r.opts.Stdout)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "DevRites install plan complete (dry run)")
	} else {
		fmt.Fprintln(r.opts.Stdout, "DevRites installed")
	}
	fmt.Fprintf(r.opts.Stdout, "  installed: %d   overwritten: %d   skipped(conflict): %d   pruned: %d\n", r.stats.installed, r.stats.overwrote, r.stats.skipped, r.stats.pruned)
	if !r.opts.DryRun && r.opts.WithSkills {
		if r.opts.WithCodex {
			fmt.Fprintln(r.opts.Stdout, "Next: reopen the project, then run /rite (Claude) or $rite (Codex).")
		} else {
			fmt.Fprintln(r.opts.Stdout, "Next: reopen the project, then run /rite.")
		}
	}
	return nil
}

func (r *runner) installTree(srcPrefix, relPrefix string) error {
	return fs.WalkDir(r.payloadFS, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk payload %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		return r.installFile(path, filepath.ToSlash(filepath.Join(relPrefix, rel)))
	})
}

func (r *runner) installFile(src, rel string) error {
	data, err := fs.ReadFile(r.payloadFS, src)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", src, err)
	}
	return r.installData(data, rel)
}

func (r *runner) preflightInstall() error {
	desired, err := r.desiredInstallPaths()
	if err != nil {
		return fmt.Errorf("preflight install paths: %w", err)
	}
	var conflicts []string
	for rel := range desired {
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		record, managed := r.prev[rel]
		if managed && !snapshot.missing {
			if kind := managedConflict(record, snapshot); kind != "" && !r.opts.Force {
				conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
			}
		}
	}
	for rel, record := range r.prev {
		if desired[rel] || hostpack.PreserveOnPrune(rel) {
			continue
		}
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		if !snapshot.missing {
			if kind := managedConflict(record, snapshot); kind != "" && !r.opts.Force {
				conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
			}
		}
		if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
			if _, err := r.rememberPath(merge.TargetRel); err != nil {
				return err
			}
		}
	}
	for _, rel := range r.installMergeTargets() {
		if _, err := r.rememberPath(rel); err != nil {
			return err
		}
	}
	if _, err := r.rememberPath(ManifestName); err != nil {
		return err
	}
	return managedConflictError(conflicts)
}

func (r *runner) preflightUninstall(entries []string) error {
	var conflicts []string
	for _, rel := range entries {
		if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
			if _, err := r.rememberPath(merge.TargetRel); err != nil {
				return err
			}
		}
		if !hostpack.ShouldRemoveOnUninstall(rel, entries) {
			continue
		}
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		if snapshot.missing {
			continue
		}
		if kind := managedConflict(r.prev[rel], snapshot); kind != "" && !r.opts.Force {
			conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
		}
	}
	if _, err := r.rememberPath(ManifestName); err != nil {
		return err
	}
	return managedConflictError(conflicts)
}

func (r *runner) desiredInstallPaths() (map[string]bool, error) {
	out := map[string]bool{".devrites/README.md": true}
	for _, tree := range hostpack.InstallTrees(r.opts.WithSkills, r.opts.WithAgents, r.opts.WithCodex) {
		err := fs.WalkDir(r.payloadFS, tree.PayloadPrefix, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(tree.PayloadPrefix, path)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(filepath.Join(tree.TargetPrefix, rel))] = true
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if r.opts.WithSkills && r.opts.AliasMode == "all" {
		for _, alias := range hostpack.Aliases {
			for _, rel := range hostpack.AliasTargets(alias, r.opts.WithCodex) {
				out[rel] = true
			}
		}
	}
	if r.opts.WithSkills && r.opts.WithCodex {
		out[hostpack.CodexAgentsMerge.MarkerRel] = true
		out[hostpack.CodexConfigMerge.MarkerRel] = true
	}
	if r.opts.WithSkills {
		out[hostpack.ClaudeSettingsMerge.MarkerRel] = true
	}
	return out, nil
}

func (r *runner) installMergeTargets() []string {
	var out []string
	if r.opts.WithSkills {
		out = append(out, hostpack.ClaudeSettingsMerge.TargetRel)
		if r.opts.WithCodex {
			out = append(out, hostpack.CodexAgentsMerge.TargetRel, hostpack.CodexConfigMerge.TargetRel)
		}
	}
	return out
}

func (r *runner) rememberPath(rel string) (pathSnapshot, error) {
	rel = filepath.ToSlash(rel)
	if snapshot, ok := r.preflight[rel]; ok {
		return snapshot, nil
	}
	snapshot, err := inspectManagedPath(r.target, rel)
	if err != nil {
		return pathSnapshot{}, err
	}
	r.preflight[rel] = snapshot
	return snapshot, nil
}

func (r *runner) recheckPath(rel string) error {
	rel = filepath.ToSlash(rel)
	before, ok := r.preflight[rel]
	if !ok {
		return fmt.Errorf("path %s was not preflighted", rel)
	}
	now, err := inspectManagedPath(r.target, rel)
	if err != nil {
		return err
	}
	if now != before {
		return fmt.Errorf("refusing to change %s: file changed after preflight; retry the operation", rel)
	}
	return nil
}

func inspectManagedPath(target, rel string) (pathSnapshot, error) {
	native := filepath.FromSlash(rel)
	if !filepath.IsLocal(native) {
		return pathSnapshot{}, fmt.Errorf("refusing unsafe managed path %q", rel)
	}
	dest := filepath.Join(target, native)
	if !safepath.WithinResolved(dest, target) {
		return pathSnapshot{}, fmt.Errorf("refusing managed path outside target: %s", rel)
	}
	walk := target
	parts := strings.Split(native, string(filepath.Separator))
	for _, part := range parts {
		walk = filepath.Join(walk, part)
		info, err := os.Lstat(walk)
		if os.IsNotExist(err) {
			return pathSnapshot{missing: true}, nil
		}
		if err != nil {
			return pathSnapshot{}, fmt.Errorf("inspect managed path %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return pathSnapshot{}, fmt.Errorf("refusing symlink or junction in managed path: %s", rel)
		}
	}
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return pathSnapshot{missing: true}, nil
	}
	if err != nil {
		return pathSnapshot{}, fmt.Errorf("inspect managed path %s: %w", rel, err)
	}
	if !info.Mode().IsRegular() {
		return pathSnapshot{}, fmt.Errorf("refusing non-regular managed path: %s", rel)
	}
	f, err := os.Open(dest)
	if err != nil {
		return pathSnapshot{}, fmt.Errorf("read managed path %s: %w", rel, err)
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return pathSnapshot{}, fmt.Errorf("hash managed path %s: %w", rel, err)
	}
	return pathSnapshot{hash: "sha256:" + hex.EncodeToString(sum.Sum(nil))}, nil
}

func managedConflict(record managedRecord, snapshot pathSnapshot) string {
	if record.Hash == "" {
		return "legacy manifest entry (no hash)"
	}
	if record.Hash != snapshot.hash {
		return "customized managed file"
	}
	return ""
}

func managedConflictError(conflicts []string) error {
	if len(conflicts) == 0 {
		return nil
	}
	sort.Strings(conflicts)
	return fmt.Errorf("managed files differ from the install manifest:\n  %s\nrerun with --force to replace or remove these files", strings.Join(conflicts, "\n  "))
}

func (r *runner) installData(data []byte, rel string) error {
	dest := filepath.Join(r.target, filepath.FromSlash(rel))
	action := "install"
	if exists(dest) {
		record, managed := r.prev[rel]
		switch {
		case managed && managedConflict(record, r.preflight[filepath.ToSlash(rel)]) != "" && r.opts.Force:
			action = "overwrite(force-customized)"
		case managed:
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
	} else {
		if err := r.recheckPath(rel); err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", rel, err)
		}
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

func (r *runner) installClaudeSettingsMarker(ownsDefaultMode bool) error {
	ownership := "preexisting"
	if ownsDefaultMode {
		ownership = "added"
	}
	text := hostpack.ClaudeSettingsMerge.MarkerText + "\ndefault-mode=" + ownership
	return r.installMarker(hostpack.ClaudeSettingsMerge.MarkerRel, text)
}

func (r *runner) claudeDefaultModeOwned() bool {
	data, err := os.ReadFile(filepath.Join(r.target, filepath.FromSlash(hostpack.ClaudeSettingsMerge.MarkerRel)))
	return err == nil && strings.Contains(string(data), "\ndefault-mode=added\n")
}

func (r *runner) mergeMarkerFile(merge hostpack.MarkerMerge) error {
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if r.opts.DryRun {
		verb := "create DevRites block"
		current, readErr := os.ReadFile(dest)
		if readErr == nil {
			if bytes.Contains(current, []byte(merge.Begin)) {
				verb = "refresh DevRites block"
			} else {
				verb = "append DevRites block"
			}
		} else if !errors.Is(readErr, fs.ErrNotExist) {
			return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
		}
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (%s)\n", merge.TargetRel, verb)
	} else {
		if err := r.recheckPath(merge.TargetRel); err != nil {
			return err
		}
		next := block
		current, readErr := os.ReadFile(dest)
		if readErr == nil {
			next = hostpack.MergeMarkerBlock(current, block, merge.Begin, merge.End)
		} else if !errors.Is(readErr, fs.ErrNotExist) {
			return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
		}
		if err := fsutil.WriteFileAtomic(dest, next, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
		}
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) mergeCodexConfig() error {
	merge := hostpack.CodexConfigMerge
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (prepend DevRites permission block)\n", merge.TargetRel)
		return r.installMarker(merge.MarkerRel, merge.MarkerText)
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	var current []byte
	if data, readErr := os.ReadFile(dest); readErr == nil {
		current = stripMarkerBlock(data, merge.Begin, merge.End)
		current = stripMarkerBlock(current, "# BEGIN DEVRITES CODEX MCP", "# END DEVRITES CODEX MCP")
		current = stripMarkerBlock(current, "### BEGIN DEVRITES CODEX MCP", "### END DEVRITES CODEX MCP")
		if hasTopLevelTOMLKey(current, "default_permissions") {
			return fmt.Errorf("%s already sets top-level default_permissions; remove that project override before installing the DevRites read-only-root profile", merge.TargetRel)
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
	}
	next := append([]byte(nil), block...)
	if len(next) == 0 || next[len(next)-1] != '\n' {
		next = append(next, '\n')
	}
	if len(bytes.TrimSpace(current)) > 0 {
		next = append(next, '\n')
		next = append(next, current...)
	}
	if err := fsutil.WriteFileAtomic(dest, next, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) mergeClaudeSettings(merge hostpack.JSONMerge) error {
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	devrites, err := readJSONFS(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("load Claude settings payload: %w", err)
	}
	current := map[string]any{}
	if data, readErr := readJSON(dest); readErr == nil {
		current = data
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("load existing Claude settings: %w", readErr)
	}
	next, ownsDefaultMode, err := mergeClaudeSettingsConfig(current, devrites, r.claudeDefaultModeOwned())
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s\n", merge.DryRunText)
		return r.installClaudeSettingsMarker(ownsDefaultMode)
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode merged Claude settings: %w", err)
	}
	data = append(data, '\n')
	if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installClaudeSettingsMarker(ownsDefaultMode)
}

func (r *runner) seedClaudeSettings() error {
	return r.mergeClaudeSettings(hostpack.ClaudeSettingsMerge)
}

func (r *runner) seedDevrites() error {
	readme, err := hostpack.RenderDevritesReadme()
	if err != nil {
		return fmt.Errorf("render readme: %w", err)
	}
	if err := r.installData(readme, ".devrites/README.md"); err != nil {
		return fmt.Errorf("install readme: %w", err)
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
		return fmt.Errorf("create %s: %w", filepath.Dir(active), err)
	}
	f, err := os.OpenFile(active, os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("create %s: %w", active, err)
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
		action := "prune"
		if managedConflict(r.prev[rel], r.preflight[rel]) != "" && r.opts.Force {
			action = "prune(force-customized)"
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [%s] %s (dropped from pack)\n", action, rel)
		} else {
			if err := r.recheckPath(rel); err != nil {
				return err
			}
			if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
				if err := r.recheckPath(merge.TargetRel); err != nil {
					return err
				}
				if merge.MarkerRel == hostpack.LegacyCodexHooksMerge.MarkerRel {
					if err := stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))); err != nil {
						return fmt.Errorf("strip hooks from %s: %w", merge.TargetRel, err)
					}
				} else if merge.TargetRel == hostpack.ClaudeSettingsMerge.TargetRel {
					_ = r.stripClaudeSettings(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), true)
				} else {
					_ = stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End)
				}
			}
			if err := os.Remove(dead); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", rel, err)
			}
			pruneEmptyDirs(filepath.Dir(dead), r.target)
			fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
		}
		r.stats.pruned++
	}
	return nil
}

func (r *runner) writeManifest() error {
	sort.Strings(r.manifest)
	r.manifest = slices.Compact(r.manifest)
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
	if err := r.recheckPath(ManifestName); err != nil {
		return err
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
	entries, err := readManifestList(mf)
	if err != nil {
		return err
	}
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
	if err := r.preflightUninstall(entries); err != nil {
		return err
	}

	for _, rel := range entries {
		merge, ok := hostpack.ManagedMergeForMarker(rel)
		if !ok {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [merge-remove] %s\n", merge.DryRun)
			continue
		}
		if err := r.recheckPath(rel); err != nil {
			return err
		}
		if err := r.recheckPath(merge.TargetRel); err != nil {
			return err
		}
		if merge.MarkerRel == hostpack.LegacyCodexHooksMerge.MarkerRel {
			if err := stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))); err != nil {
				return fmt.Errorf("strip hooks from %s: %w", merge.TargetRel, err)
			}
			continue
		}
		if merge.TargetRel == hostpack.ClaudeSettingsMerge.TargetRel {
			if err := r.stripClaudeSettings(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), true); err != nil {
				return fmt.Errorf("strip DevRites settings from %s: %w", merge.TargetRel, err)
			}
			continue
		}
		if err := stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End); err != nil {
			return fmt.Errorf("strip marker block from %s: %w", merge.TargetRel, err)
		}
	}

	var dirs []string
	for _, rel := range entries {
		if !hostpack.ShouldRemoveOnUninstall(rel, entries) {
			continue
		}
		dest := filepath.Join(r.target, filepath.FromSlash(rel))
		if exists(dest) {
			action := "remove"
			if managedConflict(r.prev[rel], r.preflight[rel]) != "" && r.opts.Force {
				action = "remove(force-customized)"
			}
			if r.opts.DryRun {
				fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
			} else {
				if err := r.recheckPath(rel); err != nil {
					return err
				}
				if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove %s: %w", rel, err)
				}
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
		if err := r.recheckPath(ManifestName); err != nil {
			return err
		}
		_ = os.Remove(mf)
		dirs = append(dirs, filepath.Dir(mf))
		for _, d := range dirs {
			pruneEmptyDirs(d, r.target)
		}
	}
	if err := r.removeBinary(); err != nil {
		return fmt.Errorf("remove engine binary: %w", err)
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

func stripHooksPath(path string) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load hooks config: %w", err)
	}
	next := stripDevritesHooks(current)
	if len(next) == 0 {
		return os.Remove(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode hooks config: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func (r *runner) stripClaudeSettings(path string, preserveEmpty bool) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load Claude settings: %w", err)
	}
	next := stripDevritesSettings(current, r.claudeDefaultModeOwned())
	if len(next) == 0 && !preserveEmpty {
		return os.Remove(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Claude settings: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func runUpdate(opts Options) error {
	target, err := filepath.Abs(opts.Target)
	if err != nil {
		return fmt.Errorf("resolve target %s: %w", opts.Target, err)
	}
	mf := filepath.Join(target, ManifestName)
	if !exists(mf) {
		return fmt.Errorf("no DevRites install found at %s (missing %s)", target, ManifestName)
	}
	installed := manifestHeader(mf, "devrites-version")
	if installed == "" {
		installed = "unknown"
	}
	if opts.SourceDir == "" {
		return fmt.Errorf("update requires --source-dir with the locally acquired source")
	}
	source, err := filepath.Abs(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("resolve source %s: %w", opts.SourceDir, err)
	}
	candidate := installedVersion(source)
	if candidate == "" || candidate == "unknown" {
		return fmt.Errorf("derive update version from local source %s: package.json has no version", source)
	}
	candidate = strings.TrimPrefix(candidate, "v")
	fmt.Fprintln(opts.Stdout, "DevRites update")
	fmt.Fprintf(opts.Stdout, "  project:   %s\n", target)
	fmt.Fprintf(opts.Stdout, "  installed: %s\n", installed)
	fmt.Fprintf(opts.Stdout, "  candidate: %s\n", candidate)
	if opts.UpdateCheck {
		if installed == candidate || strings.TrimPrefix(installed, "v") == candidate {
			fmt.Fprintln(opts.Stdout, "up to date.")
			return nil
		}
		return fmt.Errorf("update available: %s -> %s", installed, candidate)
	}
	if !opts.Force && (installed == candidate || strings.TrimPrefix(installed, "v") == candidate) {
		fmt.Fprintln(opts.Stdout, "already up to date (use --force to reinstall).")
		return nil
	}
	if opts.PayloadDir == "" {
		return fmt.Errorf("update requires --payload-dir with the locally prepared host payload")
	}
	payload, err := filepath.Abs(opts.PayloadDir)
	if err != nil {
		return fmt.Errorf("resolve payload %s: %w", opts.PayloadDir, err)
	}
	if err := hostpack.ValidatePayload(os.DirFS(payload), true); err != nil {
		return fmt.Errorf("local update payload under %s is invalid: %w", payload, err)
	}
	installFlags := manifestHeader(mf, "devrites-flags")
	next := DefaultOptions(ModeInstall)
	next.Stdout = opts.Stdout
	next.Stderr = opts.Stderr
	if installFlags != "" {
		if err := parseArgs(strings.Fields(installFlags), &next); err != nil {
			return fmt.Errorf("parse manifest install flags: %w", err)
		}
	}
	next.Target = target
	next.PayloadDir = payload
	next.SourceDir = source
	next.Force = opts.Force
	if opts.DryRun {
		next.DryRun = true
	}
	r, err := newRunner(next)
	if err != nil {
		return fmt.Errorf("prepare installer: %w", err)
	}
	r.requiredBinaryTag = "v" + candidate
	if next.WithBinary && os.Getenv("DEVRITES_NO_BINARY") != "1" && !next.DryRun {
		staged, cleanup, err := r.acquireBinary(r.requiredBinaryTag)
		if err != nil {
			return fmt.Errorf("prepare engine binary: %w", err)
		}
		defer cleanup()
		r.preparedBinary = staged
	}
	return r.install()
}

func stripMarkerPath(path, begin, end string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	next := stripMarkerBlock(data, begin, end)
	if strings.TrimSpace(string(next)) == "" {
		return os.Remove(path)
	}
	return fsutil.WriteFileAtomic(path, next, 0o644)
}

func stripMarkerBlock(data []byte, begin, end string) []byte {
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
	return []byte(out.String())
}

func hasTopLevelTOMLKey(data []byte, key string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			return false
		}
		name, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(name) == key {
			return true
		}
	}
	return false
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return decodeJSON(data)
}

func readJSONFS(src fs.FS, path string) (map[string]any, error) {
	data, err := fs.ReadFile(src, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return decodeJSON(data)
}

func decodeJSON(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return out, nil
}

func isDevritesHookCommand(v any) bool {
	hook, ok := v.(map[string]any)
	if !ok {
		return false
	}
	command, _ := hook["command"].(string)
	return strings.Contains(command, "devrites-engine hook ") ||
		strings.Contains(command, ".claude/hooks/devrites-") ||
		strings.Contains(command, ".codex/hooks/devrites-") ||
		(strings.Contains(command, "printf ") && strings.Contains(command, "DevRites:"))
}

func isDevritesHooksComment(comment string) bool {
	return comment == "DevRites hooks" ||
		strings.HasPrefix(comment, "DevRites hooks: every event invokes the global `devrites-engine` engine binary") ||
		strings.HasPrefix(comment, "DevRites hooks: auto-approve the read-only orientation/gate scripts") ||
		strings.HasPrefix(comment, "DevRites hooks — every event invokes the global `devrites-engine` engine binary") ||
		strings.HasPrefix(comment, "DevRites hooks — auto-approve the read-only orientation/gate scripts") ||
		strings.HasPrefix(comment, "DevRites hooks for Codex. Project hooks load only after")
}

func isDevritesSettingsComment(comment string) bool {
	return isDevritesHooksComment(comment) ||
		strings.HasPrefix(comment, "DevRites keeps the root orchestration context source-read-only.")
}

func stripDevritesHooks(config map[string]any) map[string]any {
	next := map[string]any{}
	for k, v := range config {
		if k == "$comment" {
			if s, ok := v.(string); ok && isDevritesHooksComment(s) {
				continue
			}
		}
		if k == "statusLine" && isDevritesHookCommand(v) {
			continue
		}
		next[k] = v
	}
	rawHooksValue, exists := next["hooks"]
	if !exists {
		return next
	}
	rawHooks, ok := rawHooksValue.(map[string]any)
	if !ok {
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
			entryMap, ok := entry.(map[string]any)
			if !ok {
				kept = append(kept, entry)
				continue
			}
			commands, ok := entryMap["hooks"].([]any)
			if !ok {
				kept = append(kept, entry)
				continue
			}
			var keptCommands []any
			for _, command := range commands {
				if !isDevritesHookCommand(command) {
					keptCommands = append(keptCommands, command)
				}
			}
			if len(keptCommands) > 0 {
				nextEntry := make(map[string]any, len(entryMap))
				for k, v := range entryMap {
					nextEntry[k] = v
				}
				nextEntry["hooks"] = keptCommands
				kept = append(kept, nextEntry)
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

func stripDevritesSettings(config map[string]any, removeOwnedDefaultMode bool) map[string]any {
	next := stripDevritesHooks(config)
	if comment, ok := next["$comment"].(string); ok && isDevritesSettingsComment(comment) {
		delete(next, "$comment")
	}

	rawPermissions, exists := next["permissions"]
	if !exists {
		return next
	}
	permissions, ok := rawPermissions.(map[string]any)
	if !ok {
		return next
	}
	clean := make(map[string]any, len(permissions))
	for key, value := range permissions {
		switch key {
		case "allow":
			rules, ok := value.([]any)
			if !ok {
				clean[key] = value
				continue
			}
			kept := make([]any, 0, len(rules))
			for _, rule := range rules {
				if !isDevritesPermissionRule(rule) {
					kept = append(kept, rule)
				}
			}
			if len(kept) > 0 {
				clean[key] = kept
			}
		case "defaultMode":
			if removeOwnedDefaultMode && value == "plan" {
				continue
			}
			clean[key] = value
		default:
			clean[key] = value
		}
	}
	if len(clean) == 0 {
		delete(next, "permissions")
	} else {
		next["permissions"] = clean
	}
	return next
}

func mergeClaudeSettingsConfig(current, desired map[string]any, defaultModeOwned bool) (map[string]any, bool, error) {
	next := stripDevritesSettings(current, false)
	permissions := map[string]any{}
	if raw, exists := next["permissions"]; exists {
		var ok bool
		permissions, ok = raw.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("existing Claude permissions must be a JSON object")
		}
	}

	switch mode, exists := permissions["defaultMode"]; {
	case !exists:
		defaultModeOwned = true
	case mode == "plan":
	case mode != "plan":
		return nil, false, fmt.Errorf("existing Claude permissions.defaultMode is %q; DevRites requires plan mode for the read-only root orchestrator", mode)
	}
	permissions["defaultMode"] = "plan"

	desiredPermissions, ok := desired["permissions"].(map[string]any)
	if !ok || desiredPermissions["defaultMode"] != "plan" {
		return nil, false, fmt.Errorf("DevRites Claude settings payload must declare permissions.defaultMode=plan")
	}
	desiredAllow, ok := desiredPermissions["allow"].([]any)
	if !ok {
		return nil, false, fmt.Errorf("DevRites Claude settings payload permissions.allow must be an array")
	}
	var allow []any
	if existingAllow, exists := permissions["allow"]; exists {
		allow, ok = existingAllow.([]any)
		if !ok {
			return nil, false, fmt.Errorf("existing Claude permissions.allow must be an array")
		}
	}
	seen := map[string]bool{}
	for _, rule := range allow {
		if value, ok := rule.(string); ok {
			seen[value] = true
		}
	}
	for _, rule := range desiredAllow {
		value, ok := rule.(string)
		if !ok {
			return nil, false, fmt.Errorf("DevRites Claude permission rules must be strings")
		}
		if !seen[value] {
			allow = append(allow, value)
			seen[value] = true
		}
	}
	permissions["allow"] = allow
	next["permissions"] = permissions
	if _, exists := next["$comment"]; !exists {
		if comment, ok := desired["$comment"].(string); ok {
			next["$comment"] = comment
		}
	}
	return next, defaultModeOwned, nil
}

func isDevritesPermissionRule(value any) bool {
	rule, ok := value.(string)
	return ok && strings.HasPrefix(strings.TrimSpace(rule), "Bash(devrites-engine ")
}

func (r *runner) installBinary() error {
	if !r.opts.WithBinary || os.Getenv("DEVRITES_NO_BINARY") == "1" {
		fmt.Fprintln(r.opts.Stdout, "  engine binary: skipped (--no-binary).")
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  would install the global devrites-engine control-plane binary")
		return nil
	}
	tag := r.binaryTag()
	incoming := strings.TrimPrefix(tag, "v")
	dest := binaryDest()
	if exists(dest) {
		if ev := engineVersion(dest); semverLike(ev) && semverLike(incoming) && verGT(ev, incoming) {
			fmt.Fprintf(r.opts.Stderr, "warning: engine binary: installed %s is newer than %s - refusing to downgrade (kept).\n", ev, tag)
			return nil
		}
	}
	staged, cleanup := r.preparedBinary, func() {}
	if staged == "" {
		var err error
		staged, cleanup, err = r.acquireBinary(tag)
		if err != nil {
			return r.binaryInstallFailure(err)
		}
	}
	defer cleanup()
	if _, err := verifyEngineBinary(staged, tag, 30*time.Second); err != nil {
		return fmt.Errorf("verify staged engine binary %s: %w", staged, err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		return err
	}
	backup, oldMode, hadOld, err := backupBinary(dest)
	if err != nil {
		return err
	}
	if backup != "" {
		defer os.Remove(backup)
	}
	if err := fsutil.WriteFileAtomic(dest, data, 0o755); err != nil {
		if restoreErr := restoreBinary(dest, backup, oldMode, hadOld); restoreErr != nil {
			return fmt.Errorf("install binary: %v; restore previous binary: %w", err, restoreErr)
		}
		return err
	}
	_ = os.Chmod(dest, 0o755)
	if _, err := verifyEngineBinary(dest, tag, 30*time.Second); err != nil {
		if restoreErr := restoreBinary(dest, backup, oldMode, hadOld); restoreErr != nil {
			return fmt.Errorf("verify installed engine binary: %v; restore previous binary: %w", err, restoreErr)
		}
		if hadOld {
			return fmt.Errorf("verify installed engine binary: %w (previous binary restored)", err)
		}
		return fmt.Errorf("verify installed engine binary: %w (bad binary removed)", err)
	}
	fmt.Fprintf(r.opts.Stdout, "  engine binary: installed %s\n", dest)
	return nil
}

func (r *runner) binaryTag() string {
	if r.requiredBinaryTag != "" {
		return r.requiredBinaryTag
	}
	if tag := os.Getenv("DEVRITES_REF"); semverLike(tag) {
		return tag
	}
	return "v" + strings.TrimPrefix(installedVersion(r.source), "v")
}

func backupBinary(dest string) (string, fs.FileMode, bool, error) {
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("inspect existing engine binary: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, false, fmt.Errorf("refusing to replace non-regular engine binary: %s", dest)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		return "", 0, false, fmt.Errorf("read existing engine binary: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".backup-*")
	if err != nil {
		return "", 0, false, fmt.Errorf("create engine binary backup: %w", err)
	}
	backup := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(backup)
		return "", 0, false, fmt.Errorf("create engine binary backup: %w", err)
	}
	if err := fsutil.WriteFileAtomic(backup, data, info.Mode().Perm()); err != nil {
		_ = os.Remove(backup)
		return "", 0, false, fmt.Errorf("write engine binary backup: %w", err)
	}
	return backup, info.Mode().Perm(), true, nil
}

func restoreBinary(dest, backup string, mode fs.FileMode, hadOld bool) error {
	if !hadOld {
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(dest, data, mode)
}

func (r *runner) binaryInstallFailure(err error) error {
	if r.preparedBinary != "" {
		return err
	}
	if os.Getenv("DEVRITES_ENGINE_CLI") != "" {
		return err
	}
	fmt.Fprintf(r.opts.Stderr, "warning: engine binary: %v; continuing without it.\n", err)
	return nil
}

func (r *runner) acquireBinary(tag string) (string, func(), error) {
	path := os.Getenv("DEVRITES_ENGINE_CLI")
	if path == "" {
		return "", func() {}, fmt.Errorf("no DEVRITES_ENGINE_CLI handoff")
	}
	if !exists(path) {
		return "", func() {}, fmt.Errorf("DEVRITES_ENGINE_CLI points to a missing binary: %s", path)
	}
	if _, err := verifyEngineBinary(path, tag, 30*time.Second); err != nil {
		return "", func() {}, fmt.Errorf("DEVRITES_ENGINE_CLI is incompatible: %w", err)
	}
	return path, func() {}, nil
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
		return filepath.Join(dir, engineBinaryName())
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "bin", engineBinaryName())
	}
	return filepath.Join("/usr/local/bin", engineBinaryName())
}

func engineBinaryName() string {
	if runtime.GOOS == "windows" {
		return "devrites-engine.exe"
	}
	return "devrites-engine"
}

func binaryCandidates() []string {
	candidates := []string{}
	if dir := os.Getenv("DEVRITES_BIN_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, engineBinaryName()))
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", engineBinaryName()))
	}
	return append(candidates, filepath.Join("/usr/local/bin", engineBinaryName()))
}

func engineVersion(path string) string {
	got, err := readEngineVersion(path, 30*time.Second)
	if err != nil {
		return ""
	}
	return got
}

func verifyEngineBinary(path, want string, timeout time.Duration) (string, error) {
	want = strings.TrimPrefix(strings.TrimSpace(want), "v")
	if want == "" || want == "dev" || !semverLike(want) {
		return "", fmt.Errorf("invalid requested version %q", want)
	}
	got, err := readEngineVersion(path, timeout)
	if err != nil {
		return "", err
	}
	if strings.TrimPrefix(got, "v") != want {
		return "", fmt.Errorf("version mismatch: got %s want %s", got, want)
	}
	return got, nil
}

func readEngineVersion(path string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("version command timed out after %s", timeout)
		}
		return "", fmt.Errorf("run exact path %s version: %w", path, err)
	}
	line := strings.TrimSuffix(string(out), "\n")
	line = strings.TrimSuffix(line, "\r")
	if strings.ContainsAny(line, "\r\n") {
		return "", fmt.Errorf("invalid multi-line version output")
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "dev" || !semverLike(line) {
		return "", fmt.Errorf("invalid version output %q", line)
	}
	return line, nil
}

func readManifest(path string) (map[string]managedRecord, error) {
	records, _, err := parseManifest(path)
	return records, err
}

func readManifestList(path string) ([]string, error) {
	_, entries, err := parseManifest(path)
	return entries, err
}

func parseManifest(path string) (map[string]managedRecord, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]managedRecord{}, nil, nil
		}
		return nil, nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	hashes := map[string]string{}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# managed: ") {
			record := strings.TrimPrefix(line, "# managed: ")
			i := strings.LastIndex(record, " sha256:")
			if i > 0 {
				rel := strings.TrimSpace(record[:i])
				hash := "sha256:" + strings.ToLower(strings.TrimSpace(record[i+len(" sha256:"):]))
				if filepath.IsLocal(filepath.FromSlash(rel)) && validManagedHash(hash) {
					hashes[rel] = hash
				}
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// The manifest is hand-editable; an entry that escapes the target
		// (absolute or ..-traversal) must never be overwritten or removed.
		if !filepath.IsLocal(filepath.FromSlash(line)) {
			continue
		}
		out = append(out, line)
	}
	records := make(map[string]managedRecord, len(out))
	for _, rel := range out {
		records[rel] = managedRecord{Hash: hashes[rel]}
	}
	return records, out, nil
}

func validManagedHash(hash string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) || len(hash) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(hash, prefix))
	return err == nil
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
		if err != nil {
			return "unknown"
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &pkg) != nil || pkg.Version == "" {
			return "unknown"
		}
		return pkg.Version
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
		_, _ = fmt.Sscanf(as[i], "%d", &ai)
		_, _ = fmt.Sscanf(bs[i], "%d", &bi)
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}
	return false
}
