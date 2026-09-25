// Package records implements the /overhaul records tool: generation staging
// and publishing plus cross-record validation of a run area.
//
//	digest FILE     sha256 of exact bytes
//	stage RUN       copy current generation to RUN/g<N+1>.tmp (prints path)
//	publish RUN     validate staged generation, write manifest, swap CURRENT last
//	validate RUN    verify CURRENT generation integrity + cross-record rules
//
// Exit: 0 ok, 1 rule violations (listed), 2 usage/IO error.
package records

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const usage = `usage: overhaul records <command> ...

  digest FILE     sha256 of exact bytes
  stage RUN       copy current generation to RUN/g<N+1>.tmp (prints path)
  publish RUN     validate staged generation, write manifest, swap CURRENT last
  validate RUN    verify CURRENT generation integrity + cross-record rules
Exit: 0 ok, 1 rule violations (listed), 2 usage/IO error.
`

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var code int
	var err error
	switch args[0] {
	case "digest":
		var d string
		if d, err = sha(args[1]); err == nil {
			fmt.Fprintln(stdout, d)
		}
	case "stage":
		code, err = stage(args[1], stdout, stderr)
	case "publish":
		code, err = publish(args[1], stdout, stderr)
	case "validate":
		code, err = validate(args[1], stdout)
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	return code
}

// current returns the CURRENT generation name; ok is false when there is none.
func current(run string) (string, bool, error) {
	p := join(run, "CURRENT")
	if !isFile(p) {
		return "", false, nil
	}
	b, err := os.ReadFile(p)
	return strings.TrimSpace(string(b)), err == nil, err
}

func genNumber(gen string) (int, error) {
	if gen == "" {
		return 0, fmt.Errorf("invalid generation name %q", gen)
	}
	n, err := strconv.Atoi(gen[1:])
	if err != nil {
		return 0, fmt.Errorf("invalid generation name %q", gen)
	}
	return n, nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func manifestOK(run, genDir string) ([]string, map[string]any, error) {
	m, err := ovio.LoadObject(join(genDir, "manifest.json"))
	if err != nil {
		return nil, nil, err
	}
	var errs []string
	files, revs := ovio.Obj(m["files"]), ovio.Obj(m["revisions"])
	check := func(src map[string]any) error {
		for _, rel := range sortedKeys(src) {
			base := run
			if _, ok := files[rel]; ok {
				base = genDir
			}
			p := join(base, rel)
			ok := isFile(p)
			if ok {
				d, err := sha(p)
				if err != nil {
					return err
				}
				ok = eq(d, src[rel])
			}
			if !ok {
				errs = append(errs, fmt.Sprintf("mixed or modified generation: %s does not match manifest", rel))
			}
		}
		return nil
	}
	if err := check(files); err != nil {
		return nil, nil, err
	}
	if err := check(revs); err != nil {
		return nil, nil, err
	}
	return errs, m, nil
}

// copyTree is shutil.copytree(src, dst, ignore=ignore_patterns("manifest.json", "views")):
// symlinks are followed and the two names are skipped at every level.
func copyTree(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, fi.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() == "manifest.json" || e.Name() == "views" {
			continue
		}
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		st, err := os.Stat(s)
		if err != nil {
			return err
		}
		if st.IsDir() {
			err = copyTree(s, d)
		} else {
			var b []byte
			if b, err = os.ReadFile(s); err == nil {
				err = os.WriteFile(d, b, st.Mode().Perm())
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func stage(run string, stdout, stderr io.Writer) (int, error) {
	if err := os.MkdirAll(join(run, "revisions"), 0o777); err != nil {
		return 0, err
	}
	cur, has, err := current(run)
	if err != nil {
		return 0, err
	}
	n, curName := 1, "None"
	if has {
		curName = cur
	}
	if cur != "" {
		g, err := genNumber(cur)
		if err != nil {
			return 0, err
		}
		n = g + 1
	}
	final := join(run, fmt.Sprintf("g%04d", n))
	tmp := final + ".tmp"
	if _, err := os.Stat(final); err == nil {
		fmt.Fprintf(stderr, "g%04d exists but CURRENT is %s: interrupted publish; inspect it, then repair CURRENT or move it aside\n", n, curName)
		return 2, nil
	}
	if _, err := os.Stat(tmp); err == nil {
		fmt.Fprintf(stderr, "staged generation already exists: %s\n", tmp)
		return 2, nil
	}
	base := "none"
	if cur != "" {
		base = cur
		err = copyTree(join(run, cur), tmp)
	} else {
		err = os.MkdirAll(tmp, 0o777)
	}
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(join(tmp, ".base"), []byte(base), 0o666); err != nil {
		return 0, err
	}
	fmt.Fprintln(stdout, tmp)
	return 0, nil
}

var stagedName = regexp.MustCompile(`^g[0-9]{4}\.tmp$`)

func publish(run string, stdout, stderr io.Writer) (int, error) {
	entries, err := os.ReadDir(run)
	if err != nil {
		return 0, err
	}
	var staged []string
	for _, e := range entries {
		if stagedName.MatchString(e.Name()) {
			staged = append(staged, e.Name())
		}
	}
	if len(staged) != 1 {
		fmt.Fprintf(stderr, "expected exactly one staged generation, found %s\n", Repr(staged))
		return 2, nil
	}
	tmp := join(run, staged[0])
	prev, hasPrev, err := current(run)
	if err != nil {
		return 0, err
	}
	b, err := os.ReadFile(join(tmp, ".base"))
	if err != nil {
		return 0, err
	}
	want, prevName := "none", "None"
	if hasPrev {
		prevName = prev
	}
	if prev != "" {
		want = prev
	}
	if base := strings.TrimSpace(string(b)); base != want {
		fmt.Fprintf(stderr, "stale stage: CURRENT moved from %s to %s; re-stage and reapply\n", base, prevName)
		return 1, nil
	}
	prevDir := ""
	if prev != "" {
		prevDir = join(run, prev)
	}
	errs, err := Check(run, tmp, prevDir)
	if err != nil {
		return 0, err
	}
	revs := map[string]any{}
	if prevDir != "" {
		pmErrs, pm, err := manifestOK(run, prevDir)
		if err != nil {
			return 0, err
		}
		errs = append(errs, pmErrs...)
		prevRevs := ovio.Obj(pm["revisions"])
		for _, rel := range sortedKeys(prevRevs) {
			p := join(run, rel)
			ok := isFile(p)
			if ok {
				d, err := sha(p)
				if err != nil {
					return 0, err
				}
				ok = eq(d, prevRevs[rel])
			}
			if !ok {
				errs = append(errs, "immutable revision changed: "+rel)
			}
			revs[rel] = prevRevs[rel]
		}
	}
	revEntries, err := os.ReadDir(join(run, "revisions"))
	if err != nil {
		return 0, err
	}
	for _, e := range revEntries {
		d, err := sha(join(run, "revisions", e.Name()))
		if err != nil {
			return 0, err
		}
		revs["revisions/"+e.Name()] = d
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(stdout, "VIOLATION: "+e)
		}
		return 1, nil
	}
	if err := os.Remove(join(tmp, ".base")); err != nil {
		return 0, err
	}
	files := map[string]any{}
	err = filepath.WalkDir(tmp, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 && isDir(p) {
			return nil // os.walk lists a symlinked directory but never descends into it
		}
		rel, err := filepath.Rel(tmp, p)
		if err != nil {
			return err
		}
		digest, err := sha(p)
		files[filepath.ToSlash(rel)] = digest
		return err
	})
	if err != nil {
		return 0, err
	}
	gen := strings.TrimSuffix(staged[0], ".tmp")
	var previous any
	if hasPrev {
		previous = prev
	}
	manifest := map[string]any{"schema": "overhaul.manifest/1", "generation": gen, "previous": previous,
		"files": files, "revisions": revs}
	if err := ovio.WriteJSON(join(tmp, "manifest.json"), manifest, 0o666); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, join(run, gen)); err != nil {
		return 0, err
	}
	if err := os.WriteFile(join(run, "CURRENT.tmp"), []byte(gen+"\n"), 0o666); err != nil {
		return 0, err
	}
	if err := os.Rename(join(run, "CURRENT.tmp"), join(run, "CURRENT")); err != nil {
		return 0, err
	}
	fmt.Fprintln(stdout, "published "+gen)
	return 0, nil
}

func validate(run string, stdout io.Writer) (int, error) {
	b, err := os.ReadFile(join(run, "CURRENT"))
	if err != nil {
		return 0, err
	}
	gen := strings.TrimSpace(string(b))
	genDir := join(run, gen)
	errs, m, err := manifestOK(run, genDir)
	if err != nil {
		return 0, err
	}
	prevDir := ""
	if p := m["previous"]; ovio.Truthy(p) {
		prevDir = join(run, PyStr(p))
	}
	more, err := Check(run, genDir, prevDir)
	if err != nil {
		return 0, err
	}
	errs = append(errs, more...)
	n, err := genNumber(gen)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(run)
	if err != nil {
		return 0, err
	}
	next := fmt.Sprintf("g%04d", n+1)
	for _, e := range entries {
		if d := e.Name(); strings.HasSuffix(d, ".tmp") || d == next {
			fmt.Fprintf(stdout, "NOTE: %s is newer than CURRENT (crash or in-progress publish)\n", d)
		}
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(stdout, "VIOLATION: "+e)
		}
		return 1, nil
	}
	fmt.Fprintln(stdout, "valid "+gen)
	return 0, nil
}
