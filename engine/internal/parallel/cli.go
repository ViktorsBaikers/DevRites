package parallel

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Exit codes match the engine CLI conventions.
const (
	ExitOK      = 0
	ExitUsage   = 2
	ExitBlocked = 3
)

// Run is the engine entrypoint for `parallel …` and `check path-disjoint`.
// command is either "parallel" or "path-disjoint".
func Run(command string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch command {
	case "path-disjoint":
		return runPathDisjoint(args, stdin, stdout, stderr)
	case "parallel":
		return runParallel(args, stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "parallel: unknown command %q\n", command)
		return ExitUsage
	}
}

func runPathDisjoint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := ""
	jsonPath := "-"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "usage: check path-disjoint [--root <dir>] [<json-file>|-]")
				return ExitUsage
			}
			i++
			root = args[i]
		case "-h", "--help":
			fmt.Fprintln(stdout, "usage: check path-disjoint [--root <dir>] [<json-file>|-]")
			return ExitOK
		default:
			if args[i] != "-" && strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "path-disjoint: unknown flag %s\n", args[i])
				return ExitUsage
			}
			jsonPath = args[i]
		}
	}
	data, err := readJSONInput(jsonPath, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "path-disjoint: %v\n", err)
		return ExitBlocked
	}
	slices, err := ParseSlicesJSON(data)
	if err != nil {
		fmt.Fprintf(stderr, "path-disjoint: %v\n", err)
		return ExitBlocked
	}
	ids, err := CheckPathDisjoint(slices, root)
	if err != nil {
		fmt.Fprintf(stderr, "path-disjoint: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "path-disjoint: ok (%d slices)\n", len(ids))
	return ExitOK
}

func runParallel(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, parallelUsage())
		return ExitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "create":
		return cmdCreate(rest, stdin, stdout, stderr)
	case "record-green":
		return cmdRecordGreen(rest, stdout, stderr)
	case "abort":
		return cmdAbort(rest, stdout, stderr)
	case "integrate":
		return cmdIntegrate(rest, stdout, stderr)
	case "cleanup":
		return cmdCleanup(rest, stdout, stderr)
	case "status":
		return cmdStatus(rest, stdout, stderr)
	case "lease-write", "write-lease":
		return cmdLeaseWrite(rest, stdin, stdout, stderr)
	case "lease-read", "read-lease":
		return cmdLeaseRead(rest, stdout, stderr)
	case "lease-clear", "clear-lease":
		return cmdLeaseClear(rest, stdout, stderr)
	case "check-disjoint", "path-disjoint":
		return runPathDisjoint(rest, stdin, stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprintln(stdout, parallelUsage())
		return ExitOK
	default:
		fmt.Fprintf(stderr, "parallel: unknown subcommand %q\n\n%s\n", sub, parallelUsage())
		return ExitUsage
	}
}

func parallelUsage() string {
	return strings.TrimSpace(`usage: parallel <subcommand> [options]

Subcommands:
  create          --root --slug --batch --base --json
  record-green    --root --slug --slice --commit
  abort           --root --slug [--force]
  integrate       --root --slug [--apply-to-control] [--force]
  cleanup         --root --slug [--force]
  status          --root --slug
  lease-write     --root --slug --json
  lease-read      --root --slug [--field name]
  lease-clear     --root --slug
  check-disjoint  [--root] [<json-file>|-]

Exit codes: 0 ok, 2 usage, 3 blocked`)
}

type flagSet struct {
	Root, Slug, Batch, Base, Slice, Commit, JSON, Field, Session string
	ApplyToControl, Force                                        bool
}

func parseFlags(args []string, stderr io.Writer) (flagSet, []string, int) {
	var f flagSet
	i := 0
	for i < len(args) {
		a := args[i]
		need := func(name string) (string, bool) {
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "parallel: %s requires a value\n", name)
				return "", false
			}
			i++
			return args[i], true
		}
		switch a {
		case "--root":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Root = v
		case "--slug":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Slug = v
		case "--batch":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Batch = v
		case "--base":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Base = v
		case "--slice":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Slice = v
		case "--commit":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Commit = v
		case "--json":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.JSON = v
		case "--field":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Field = v
		case "--session":
			v, ok := need(a)
			if !ok {
				return f, nil, ExitUsage
			}
			f.Session = v
		case "--apply-to-control":
			f.ApplyToControl = true
		case "--force":
			f.Force = true
		default:
			return f, args[i:], ExitOK
		}
		i++
	}
	return f, nil, ExitOK
}

func requireRootSlug(f flagSet, stderr io.Writer) int {
	if f.Root == "" || f.Slug == "" {
		fmt.Fprintln(stderr, "parallel: --root and --slug are required")
		return ExitUsage
	}
	return ExitOK
}

func readJSONInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(stdin)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("json path must not contain '..'")
	}
	return os.ReadFile(path)
}

func cmdCreate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	if f.Batch == "" || f.Base == "" || f.JSON == "" {
		fmt.Fprintln(stderr, "usage: parallel create --root --slug --batch --base --json")
		return ExitUsage
	}
	data, err := readJSONInput(f.JSON, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "parallel create: %v\n", err)
		return ExitBlocked
	}
	slices, err := ParseSlicesJSON(data)
	if err != nil {
		fmt.Fprintf(stderr, "parallel create: %v\n", err)
		return ExitBlocked
	}
	lease, err := Create(CreateOpts{
		Root:    f.Root,
		Slug:    f.Slug,
		BatchID: f.Batch,
		BaseSHA: f.Base,
		Session: f.Session,
		Slices:  slices,
	})
	if err != nil {
		fmt.Fprintf(stderr, "parallel create: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "created: %d worktrees; batch=%s status=%s\n", lease.N, lease.BatchID, lease.Status)
	return ExitOK
}

func cmdRecordGreen(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	if f.Slice == "" || f.Commit == "" {
		fmt.Fprintln(stderr, "usage: parallel record-green --root --slug --slice --commit")
		return ExitUsage
	}
	lease, err := RecordGreen(f.Root, f.Slug, f.Slice, f.Commit)
	if err != nil {
		fmt.Fprintf(stderr, "parallel record-green: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "record-green: slice=%s status=%s\n", f.Slice, lease.Status)
	return ExitOK
}

func cmdAbort(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	lease, err := Abort(f.Root, f.Slug, f.Force)
	if err != nil {
		fmt.Fprintf(stderr, "parallel abort: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "aborted: status=%s base=%s\n", lease.Status, lease.BaseSHA)
	return ExitOK
}

func cmdIntegrate(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	tip, lease, err := Integrate(IntegrateOpts{
		Root:           f.Root,
		Slug:           f.Slug,
		ApplyToControl: f.ApplyToControl,
		Force:          f.Force,
	})
	if err != nil {
		fmt.Fprintf(stderr, "parallel integrate: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "integrate-ok: tip=%s status=%s apply_to_control=%v\n", tip, lease.Status, f.ApplyToControl)
	return ExitOK
}

func cmdCleanup(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	if err := Cleanup(f.Root, f.Slug, f.Force); err != nil {
		fmt.Fprintf(stderr, "parallel cleanup: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintln(stdout, "cleanup: done")
	return ExitOK
}

func cmdStatus(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	text, err := StatusReport(f.Root, f.Slug)
	if err != nil {
		fmt.Fprintf(stderr, "parallel status: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprint(stdout, text)
	return ExitOK
}

func cmdLeaseWrite(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	if f.JSON == "" {
		fmt.Fprintln(stderr, "usage: parallel lease-write --root --slug --json")
		return ExitUsage
	}
	data, err := readJSONInput(f.JSON, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "parallel lease-write: %v\n", err)
		return ExitBlocked
	}
	var lease Lease
	if err := json.Unmarshal(data, &lease); err != nil {
		fmt.Fprintf(stderr, "parallel lease-write: %v\n", err)
		return ExitBlocked
	}
	if lease.CreatedAt == "" {
		lease.CreatedAt = NowUTC()
	}
	path, err := LeasePath(f.Root, f.Slug)
	if err != nil {
		fmt.Fprintf(stderr, "parallel lease-write: %v\n", err)
		return ExitUsage
	}
	if err := WriteLease(path, &lease); err != nil {
		fmt.Fprintf(stderr, "parallel lease-write: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintln(stdout, path)
	return ExitOK
}

func cmdLeaseRead(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	path, err := LeasePath(f.Root, f.Slug)
	if err != nil {
		fmt.Fprintf(stderr, "parallel lease-read: %v\n", err)
		return ExitUsage
	}
	lease, err := ReadLease(path)
	if err != nil {
		fmt.Fprintf(stderr, "parallel lease-read: %v\n", err)
		return ExitBlocked
	}
	switch f.Field {
	case "", "yaml", "md":
		text, err := EmitLeaseMarkdown(lease)
		if err != nil {
			fmt.Fprintf(stderr, "parallel lease-read: %v\n", err)
			return ExitBlocked
		}
		fmt.Fprint(stdout, text)
	case "json":
		text, err := LeaseJSON(lease)
		if err != nil {
			fmt.Fprintf(stderr, "parallel lease-read: %v\n", err)
			return ExitBlocked
		}
		fmt.Fprintln(stdout, text)
	case "status":
		fmt.Fprintln(stdout, lease.Status)
	case "base_sha":
		fmt.Fprintln(stdout, lease.BaseSHA)
	case "batch_id":
		fmt.Fprintln(stdout, lease.BatchID)
	case "n":
		fmt.Fprintln(stdout, lease.N)
	default:
		fmt.Fprintf(stderr, "parallel lease-read: unknown field %q\n", f.Field)
		return ExitUsage
	}
	return ExitOK
}

func cmdLeaseClear(args []string, stdout, stderr io.Writer) int {
	f, _, code := parseFlags(args, stderr)
	if code != ExitOK {
		return code
	}
	if code := requireRootSlug(f, stderr); code != ExitOK {
		return code
	}
	path, err := LeasePath(f.Root, f.Slug)
	if err != nil {
		fmt.Fprintf(stderr, "parallel lease-clear: %v\n", err)
		return ExitUsage
	}
	if err := ClearLease(path); err != nil {
		fmt.Fprintf(stderr, "parallel lease-clear: %v\n", err)
		return ExitBlocked
	}
	fmt.Fprintf(stdout, "cleared: %s\n", path)
	return ExitOK
}
