package parallel

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParallelCreateHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"create", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q", stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, usageCreate) {
		t.Fatalf("stdout=%q, want %q", got, usageCreate)
	}
}

func TestParallelCreateUnknownArgumentStillErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"create", "--nope"}, strings.NewReader(""), &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("code=%d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr.String(), `unknown argument "--nope"`) {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestParallelUnlistedSubcommandFormsRejected(t *testing.T) {
	for _, sub := range []string{"write-lease", "read-lease", "clear-lease", "check-disjoint", "path-disjoint"} {
		var stdout, stderr bytes.Buffer
		code := Run("parallel", []string{sub}, strings.NewReader(""), &stdout, &stderr)
		if code != ExitUsage {
			t.Errorf("%s: code=%d, want %d", sub, code, ExitUsage)
		}
		if want := `unknown subcommand "` + sub + `"`; !strings.Contains(stderr.String(), want) {
			t.Errorf("%s: stderr=%q, want %q", sub, stderr.String(), want)
		}
	}
}

func TestParallelCleanupExitsBlockedWhenBranchCleanupFails(t *testing.T) {
	repo, slug, _, _ := stuckBranchBatch(t)
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"cleanup", "--root", repo, "--slug", slug, "--force"}, strings.NewReader(""), &stdout, &stderr)
	if code != ExitBlocked {
		t.Fatalf("code=%d, want %d; stdout=%q stderr=%q", code, ExitBlocked, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "cleanup: done") {
		t.Fatalf("stdout reports success: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "cleanup incomplete") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

type countingWhitespaceReader struct{ n int }

func (r *countingWhitespaceReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	r.n += len(p)
	return len(p), nil
}

func TestPathDisjointRejectsOversizedInputWithoutDraining(t *testing.T) {
	const limit = 8 << 20
	src := &countingWhitespaceReader{}
	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- Run("path-disjoint", []string{"-"}, io.LimitReader(src, 64<<20), &stdout, &stderr)
	}()
	code := <-done
	if code == ExitOK {
		t.Fatalf("code=%d, want non-zero", code)
	}
	if !strings.Contains(stderr.String(), "too large") {
		t.Fatalf("stderr=%q, want 'too large'", stderr.String())
	}
	if src.n > limit+1+64<<10 {
		t.Fatalf("read %d bytes, want at most about %d", src.n, limit+1)
	}
}
