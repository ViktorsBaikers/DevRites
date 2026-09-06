package install

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/hostpack"
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
	WithOmp     bool
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
		WithOmp:    true,
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
	noOmp := flags.Bool("no-omp", !opts.WithOmp, "")
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
	opts.WithOmp = !*noOmp
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
		return "usage: devrites-engine update [--target DIR] [--dry-run] [--force] [--check] [install flags] [--source-dir DIR --payload-dir DIR]\n"
	default:
		return "usage: devrites-engine install [--target DIR] [--dry-run] [--force] [--no-codex] [--no-omp] [--no-agents] [--no-skills] [--no-binary] [--short-aliases=all]\n"
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
	if err := hostpack.ValidatePayload(r.payloadFS, r.opts.WithCodex, r.opts.WithOmp); err != nil {
		return fmt.Errorf("generated install payload under %s: %w", r.payload, err)
	}
	return nil
}

func isGlobalTarget(target string) bool {
	home, _ := os.UserHomeDir()
	if target == home {
		return true
	}
	for _, global := range []string{
		filepath.Join(home, ".claude"),
		filepath.Join(home, ".codex"),
		filepath.Join(home, ".omp"),
		filepath.Join(home, ".omp", "agent"),
	} {
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
	// #nosec G703 -- existence probe on managed install paths
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
