package lib

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/state"
)

// migrateStep is one deterministic action of a migration plan.
type migrateStep struct {
	kind   string // "phase", "cursor", "schema", or "stub"
	path   string // workspace-relative file a stub creates
	detail string
}

type migrateQuestion struct {
	id     string
	prompt string
}

type migratePlan struct {
	phase     string
	steps     []migrateStep
	questions []migrateQuestion
}

// Migrate normalizes a pre-v5 feature workspace to the current schema: it
// converts legacy bullet cursor fields into canonical table rows, records the
// schema row, and creates missing required artifacts as empty stubs. Bound
// proof files are preserved byte-exact; content is never synthesized. The
// command is one-shot and fail-closed: on ambiguity it writes nothing, prints
// its questions, and exits non-zero; answers arrive on rerun via
// --answer id=choice. --dry-run prints the plan and always writes nothing.
func Migrate(root string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dryRun := flags.Bool("dry-run", false, "print the plan without writing")
	var answerPairs stringSlice
	flags.Var(&answerPairs, "answer", "answer a migration question: id=choice")
	// Parse with interleaving support: stdlib stops at the first positional,
	// so re-parse the remainder and collect positionals along the way.
	var positionals []string
	remaining := args
	for {
		if err := flags.Parse(remaining); err != nil {
			fmt.Fprintf(stderr, "devrites: %v\n", err)
			return 2
		}
		rest := flags.Args()
		if len(rest) == 0 {
			break
		}
		positionals = append(positionals, rest[0])
		remaining = rest[1:]
	}
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine migrate <slug> [--dry-run] [--answer id=choice]")
		return 2
	}
	slug := positionals[0]

	work, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "migrate: %v\n", err)
		return 2
	}
	answered := map[string]string{}
	for _, pair := range answerPairs {
		id, value, ok := strings.Cut(pair, "=")
		if !ok || strings.TrimSpace(id) == "" {
			fmt.Fprintf(stderr, "devrites: invalid --answer %q (want id=choice)\n", pair)
			return 2
		}
		answered[strings.TrimSpace(id)] = strings.TrimSpace(value)
	}

	// Read, plan, and apply under one feature lock so a concurrent
	// `state resolve`/`close` can never be clobbered by a stale read.
	ledger := filepath.Join(work, "state.md")
	var applied int
	var exit int
	var result string
	err = state.WithFeatureLock(root, slug, func() error {
		// #nosec G304 -- workspace ledger path resolved from the operator root
		raw, err := os.ReadFile(ledger)
		if err != nil {
			return err
		}
		lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")

		version, err := state.WorkspaceSchema(root, slug)
		if err != nil {
			return err
		}
		if version > state.SchemaVersion {
			fmt.Fprintf(stderr, "migrate: workspace %s declares schema %d, newer than this engine's %d; upgrade devrites\n", slug, version, state.SchemaVersion)
			exit = 3
			return nil
		}
		if version == state.SchemaVersion {
			result = fmt.Sprintf("migrate: %s already at schema %d; nothing to do\n", slug, version)
			exit = 0
			return nil
		}

		plan := buildMigratePlan(root, slug, lines, answered)
		if len(plan.questions) > 0 {
			for _, q := range plan.questions {
				fmt.Fprintf(stderr, "migrate: question %s: %s\n", q.id, q.prompt)
			}
			fmt.Fprintln(stderr, "migrate: nothing written; re-run with --answer id=choice")
			exit = 3
			return nil
		}

		if *dryRun {
			fmt.Fprintf(stdout, "migrate: plan for %s\n", slug)
			for _, step := range plan.steps {
				fmt.Fprintf(stdout, "  [%s] %s\n", step.kind, step.detail)
			}
			fmt.Fprintln(stdout, "migrate: dry run; nothing written")
			exit = 0
			return nil
		}

		// New stub files go first: they are idempotent, never bound, and
		// state.md is the commit point written last.
		for _, step := range plan.steps {
			if step.kind != "stub" {
				continue
			}
			path := filepath.Join(work, filepath.FromSlash(step.path))
			if _, statErr := os.Stat(path); statErr == nil {
				return fmt.Errorf("stub target %s appeared during migration; re-run migrate", step.path)
			}
			if err := fsutil.WriteFileAtomic(path, nil, 0o644); err != nil {
				return err
			}
			applied++
		}
		next := lines
		for _, step := range plan.steps {
			if step.kind == "phase" {
				next = state.UpsertCursorField(next, state.CursorPhase, plan.phase)
			}
		}
		if converted, changed := state.ConvertCursorToTable(next); changed {
			next = converted
		}
		next = state.UpsertCursorField(next, state.CursorSchema, fmt.Sprint(state.SchemaVersion))
		if err := fsutil.WriteFileAtomic(ledger, []byte(strings.Join(next, "\n")), 0o644); err != nil {
			return err
		}
		result = fmt.Sprintf("migrate: %s normalized to schema %d (%d stub(s) created)\n", slug, state.SchemaVersion, applied)
		exit = 0
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "migrate: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, result)
	return exit
}

func buildMigratePlan(root, slug string, lines []string, answered map[string]string) migratePlan {
	plan := migratePlan{}
	phase, hasPhase := state.CursorField(lines, state.CursorPhase)

	if value, ok := answered["phase"]; ok {
		plan.steps = append(plan.steps, migrateStep{kind: "phase", detail: "record phase " + value + " from answer"})
		phase, hasPhase = value, true
	}
	if !hasPhase {
		plan.questions = append(plan.questions, migrateQuestion{id: "phase",
			prompt: "state.md records no phase; re-run with --answer phase=<phase>"})
		return plan
	}
	if _, ok := state.PolicyFor(state.Phase(phase)); !ok {
		plan.questions = append(plan.questions, migrateQuestion{id: "phase",
			prompt: "state.md records unknown phase " + phase + "; re-run with --answer phase=<phase>"})
		return plan
	}
	plan.phase = phase

	if state.CursorForm(lines) == "legacy" {
		plan.steps = append(plan.steps, migrateStep{kind: "cursor", detail: "convert legacy cursor bullets to canonical table rows"})
	}
	plan.steps = append(plan.steps, migrateStep{kind: "schema", detail: fmt.Sprintf("record schema %d in state.md", state.SchemaVersion)})

	policy, _ := state.PolicyFor(state.Phase(phase))
	for _, artifact := range policy.RequiredArtifacts {
		name := string(artifact)
		if name == state.LedgerFile {
			continue
		}
		if _, err := os.Stat(filepath.Join(devritespaths.FeatureDir(root, slug), filepath.FromSlash(name))); err == nil {
			continue
		}
		plan.steps = append(plan.steps, migrateStep{kind: "stub", path: name, detail: "create missing " + name + " as an empty stub"})
	}
	return plan
}

// stringSlice collects repeated string flag values.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
