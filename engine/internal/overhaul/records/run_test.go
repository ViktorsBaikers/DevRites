package records

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func generations(t *testing.T, run string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(run, "g[0-9][0-9][0-9][0-9]"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range m {
		m[i] = filepath.Base(m[i])
	}
	return m
}

func TestPruneKeepsCurrentAndPrevious(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 5; i++ {
		f.mustPublish(f.records())
	}
	if got := generations(t, f.run); len(got) != 5 {
		t.Fatalf("publish must not prune on its own: %v", got)
	}
	if code, out, errs := f.cmd("prune", f.run, "--keep", "2"); code != 0 {
		t.Fatalf("prune: %d %s %s", code, out, errs)
	}
	if got := strings.Join(generations(t, f.run), " "); got != "g0004 g0005" {
		t.Fatalf("generations after prune = %q", got)
	}
	if code, out, _ := f.cmd("validate", f.run); code != 0 || strings.TrimSpace(out) != "valid g0005" {
		t.Fatalf("validate: %d %s", code, out)
	}
	f.mustPublish(f.records())
	if got := strings.Join(generations(t, f.run), " "); got != "g0004 g0005 g0006" {
		t.Fatalf("generations after publish = %q", got)
	}
}

func TestPruneKeepWindowAndGuards(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 5; i++ {
		f.mustPublish(f.records())
	}
	if code, _, _ := f.cmd("prune", f.run, "--keep", "1"); code != 2 {
		t.Fatalf("keep below 2 must be a usage error, got %d", code)
	}
	if code, _, _ := f.cmd("prune", f.run, "--keep", "x"); code != 2 {
		t.Fatalf("non-numeric keep must be a usage error, got %d", code)
	}
	if got := generations(t, f.run); len(got) != 5 {
		t.Fatalf("rejected prune removed generations: %v", got)
	}
	if code, _, errs := f.cmd("prune", f.run, "--keep", "3"); code != 0 {
		t.Fatalf("prune: %d %s", code, errs)
	}
	if got := strings.Join(generations(t, f.run), " "); got != "g0003 g0004 g0005" {
		t.Fatalf("generations after prune = %q", got)
	}
	if err := os.MkdirAll(filepath.Join(f.run, "g0006.tmp"), 0o700); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := f.cmd("prune", f.run); code != 0 {
		t.Fatalf("default prune: %d %s", code, errs)
	}
	if got := strings.Join(generations(t, f.run), " "); got != "g0004 g0005" {
		t.Fatalf("generations after default prune = %q", got)
	}
	if _, err := os.Stat(filepath.Join(f.run, "g0006.tmp")); err != nil {
		t.Fatalf("staged generation must survive prune: %v", err)
	}
}
