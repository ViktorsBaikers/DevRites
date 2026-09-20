package lib

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOpenVisualResolvesNameUnderWorkspace(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "feature"))

	var opens atomic.Int32
	restore := swapOpenVisualOpener(func(path string) error {
		opens.Add(1)
		if path != html {
			t.Fatalf("opened %q, want %q", path, html)
		}
		return nil
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d stderr=%q", code, stderr.String())
	}
	if opens.Load() != 1 {
		t.Fatalf("opener calls = %d, want 1", opens.Load())
	}
	out := stdout.String()
	if !strings.Contains(out, "open-visual: html="+html) {
		t.Fatalf("stdout missing html tip:\n%s", out)
	}
	if !strings.Contains(out, "open-visual: outline="+strings.TrimSuffix(html, ".html")+".outline.md") {
		t.Fatalf("stdout missing outline tip:\n%s", out)
	}
	if !strings.Contains(out, "open-visual: playbooks="+openVisualPlaybookHint) {
		t.Fatalf("stdout missing playbook tip:\n%s", out)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestOpenVisualAbsolutePathAndNoOpen(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	if root == "" {
		t.Fatal("empty fixture root")
	}

	var opens atomic.Int32
	restore := swapOpenVisualOpener(func(string) error {
		opens.Add(1)
		return nil
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual("", []string{html, "--no-open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d stderr=%q", code, stderr.String())
	}
	if opens.Load() != 0 {
		t.Fatalf("opener calls = %d, want 0 with --no-open", opens.Load())
	}
	if !strings.Contains(stdout.String(), "open-visual: html="+html) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestOpenVisualWarnsMissingOutline(t *testing.T) {
	root, html := writeOpenVisualFixture(t, false)
	t.Setenv("DEVRITES_ROOT", root)

	restore := swapOpenVisualOpener(func(string) error { return nil })
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo", "--slug", "feature", "--no-open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d", code)
	}
	if !strings.Contains(stderr.String(), "warning: missing outline companion") {
		t.Fatalf("stderr = %q, want missing-outline warning", stderr.String())
	}
	if !strings.Contains(stdout.String(), "open-visual: outline=(missing)") {
		t.Fatalf("stdout = %q, want missing outline tip", stdout.String())
	}
	if !strings.Contains(stdout.String(), "open-visual: html="+html) {
		t.Fatalf("stdout = %q, want html tip for %s", stdout.String(), html)
	}
}

func TestOpenVisualUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing operand", args: nil, want: "path or visual name required"},
		{name: "unknown flag", args: []string{"demo", "--poll"}, want: `unknown flag "--poll"`},
		{name: "slug without value", args: []string{"demo", "--slug"}, want: "--slug requires a value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := OpenVisual("", test.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("code = %d, want 2", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), test.want)
			}
		})
	}
}

func TestOpenVisualRejectsNonHTML(t *testing.T) {
	dir := t.TempDir()
	path := writeBasenameFile(t, dir, "notes.md", "# hi\n")
	restore := swapOpenVisualOpener(func(string) error {
		t.Fatal("opener must not run for non-html")
		return nil
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual("", []string{path, "--no-open"}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "require a .html file") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestOpenVisualRefusesNameModeSymlinkEscape(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "feature"))

	outside := writeBasenameFile(t, t.TempDir(), "escape.html", "<!doctype html><title>escape</title>\n")
	removeBasename(t, filepath.Dir(html), filepath.Base(html))
	if err := os.Symlink(outside, html); err != nil {
		t.Fatal(err)
	}

	restore := swapOpenVisualOpener(func(string) error {
		t.Fatal("opener must not run for escaped symlink")
		return nil
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "escapes workspace visual/") {
		t.Fatalf("stderr = %q, want symlink escape refusal", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty on refuse", stdout.String())
	}
}

func TestOpenVisualPrintsTipsWhenOpenerFails(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "feature"))

	restore := swapOpenVisualOpener(func(string) error {
		return errors.New("xdg-open: not found")
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d, want 0 (warn-and-continue)", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "open-visual: html="+html) {
		t.Fatalf("stdout missing html tip:\n%s", out)
	}
	if !strings.Contains(out, "open-visual: outline=") {
		t.Fatalf("stdout missing outline tip:\n%s", out)
	}
	if !strings.Contains(out, "open-visual: playbooks="+openVisualPlaybookHint) {
		t.Fatalf("stdout missing playbook tip:\n%s", out)
	}
	if !strings.Contains(stderr.String(), "warning: failed to open browser") {
		t.Fatalf("stderr = %q, want opener warning", stderr.String())
	}
}

func TestIsOpenVisualPathOperandLeadingDotName(t *testing.T) {
	if isOpenVisualPathOperand(".draft") {
		t.Fatal(".draft should be a visual name, not a path operand")
	}
	if !isOpenVisualPathOperand("./demo.html") {
		t.Fatal("./demo.html should be a path operand")
	}
	if !isOpenVisualPathOperand("../other.html") {
		t.Fatal("../other.html should be a path operand")
	}
}

func writeOpenVisualFixture(t *testing.T, withOutline bool) (root, html string) {
	t.Helper()
	project := t.TempDir()
	root = filepath.Join(project, ".devrites")
	visual := filepath.Join(root, "work", "feature", "visual")
	if err := os.MkdirAll(visual, 0o755); err != nil {
		t.Fatal(err)
	}
	html = writeBasenameFile(t, visual, "demo.html", "<!doctype html><title>demo</title>\n")
	if withOutline {
		writeBasenameFile(t, visual, "demo.outline.md", "# Title\n\ndemo\n")
	}
	writeBasenameFile(t, filepath.Join(root, "work", "feature"), "state.md", "# state\n")
	if resolved, err := filepath.EvalSymlinks(html); err == nil {
		html = resolved
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return root, html
}

// writeBasenameFile writes contents under dir using only filepath.Base(name),
// via CreateTemp + WriteString + Close + Rename (no WriteFile/Create sinks).
func writeBasenameFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	base := filepath.Base(name)
	dst := filepath.Join(dir, base)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		t.Fatal(err)
	}
	written, werr := tmp.WriteString(contents)
	if werr != nil {
		if cerr := tmp.Close(); cerr != nil {
			t.Fatalf("write: %v; close: %v", werr, cerr)
		}
		t.Fatal(werr)
	}
	if written != len(contents) {
		if cerr := tmp.Close(); cerr != nil {
			t.Fatalf("short write %d/%d; close: %v", written, len(contents), cerr)
		}
		t.Fatalf("short write %d/%d", written, len(contents))
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

// removeBasename moves dir/filepath.Base(name) out of dir via Rename (no Remove sink).
func removeBasename(t *testing.T, dir, name string) {
	t.Helper()
	src := filepath.Join(dir, filepath.Base(name))
	dst := filepath.Join(t.TempDir(), filepath.Base(name))
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
}

func swapOpenVisualOpener(fn func(string) error) func() {
	prev := openVisualOpener
	openVisualOpener = fn
	return func() { openVisualOpener = prev }
}
