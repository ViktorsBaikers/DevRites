package install

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestRootReplacementCannotMutateOutside(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, mode := range []Mode{ModeInstall, ModeUninstall} {
		for _, ancestor := range []bool{false, true} {
			name := string(mode) + "/root"
			if ancestor {
				name = string(mode) + "/ancestor"
			}
			t.Run(name, func(t *testing.T) {
				parent := t.TempDir()
				swapPath := filepath.Join(parent, "alpha")
				target := swapPath
				outside := t.TempDir()
				outsideTarget := outside
				if ancestor {
					target = filepath.Join(swapPath, "project")
					outsideTarget = filepath.Join(outside, "project")
				}
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatal(err)
				}
				testutil.WriteFile(t, filepath.Join(target, "alpha.txt"), "alpha user content\n")
				payload := testPayload(t)
				opts := DefaultOptions(mode)
				opts.Target, opts.PayloadDir = target, payload
				opts.WithBinary = false
				rel := ".claude/skills/rite/SKILL.md"
				if mode == ModeUninstall {
					installOpts := opts
					installOpts.Mode = ModeInstall
					if err := Apply(installOpts); err != nil {
						t.Fatal(err)
					}
				}
				victim := filepath.Join(outsideTarget, filepath.FromSlash(rel))
				testutil.WriteFile(t, victim, "outside alpha\n")
				attack := newRootReplacementAttack(t, swapPath, filepath.Join(parent, "beta"), outside)
				t.Cleanup(func() { afterRecheckPath = nil })
				afterRecheckPath = func(path string) {
					if path == rel {
						attack.attempt()
					}
				}
				err := Apply(opts)
				afterRecheckPath = nil
				attack.assertOutcome(err)
				if got := testutil.ReadFile(t, victim); got != "outside alpha\n" {
					t.Fatalf("outside victim changed: %q", got)
				}
				original := attack.originalPath()
				if ancestor {
					original = filepath.Join(original, "project")
				}
				if got := testutil.ReadFile(t, filepath.Join(original, "alpha.txt")); got != "alpha user content\n" {
					t.Fatalf("operation changed original user content: %q", got)
				}
				managed := filepath.Join(original, filepath.FromSlash(rel))
				if attack.blocked && mode == ModeInstall || !attack.blocked && mode == ModeUninstall {
					if got := testutil.ReadFile(t, managed); got != "claude rite\n" {
						t.Fatalf("original managed content changed unexpectedly: %q", got)
					}
				} else if _, err := os.Stat(managed); !os.IsNotExist(err) {
					t.Fatalf("managed path should be absent: %v", err)
				}
				_, manifestErr := os.Stat(filepath.Join(original, ManifestName))
				if mode == ModeInstall && attack.blocked || mode == ModeUninstall && !attack.blocked {
					if manifestErr != nil {
						t.Fatalf("expected ownership manifest: %v", manifestErr)
					}
				} else if !os.IsNotExist(manifestErr) {
					t.Fatalf("unexpected ownership manifest: %v", manifestErr)
				}
			})
		}
	}
}

func TestChangedTargetBeforeRootAcquisitionIsRejected(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	parent := t.TempDir()
	target := filepath.Join(parent, "alpha")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions(ModeInstall)
	opts.Target, opts.PayloadDir = target, testPayload(t)
	opts.WithBinary = false
	r, err := newRunner(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(target, filepath.Join(parent, "beta")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.install(); err == nil {
		t.Fatal("install accepted replacement target inode")
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("replacement target changed")
	}
}

// Windows roots omit FILE_SHARE_DELETE, so an otherwise valid rename may be
// denied while the operation holds its root. Prove that barrier, not a bad fixture.
type rootReplacementAttack struct {
	t                      *testing.T
	source, moved, outside string
	info                   os.FileInfo
	attempted, blocked     bool
}

func newRootReplacementAttack(t *testing.T, source, moved, outside string) *rootReplacementAttack {
	t.Helper()
	info, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, moved); err != nil {
		t.Fatalf("pre-root rename control failed: %v", err)
	}
	if err := os.Rename(moved, source); err != nil {
		t.Fatalf("pre-root rename restore failed: %v", err)
	}
	probe := filepath.Join(filepath.Dir(source), "root-replacement-probe")
	if err := os.Symlink(outside, probe); err != nil {
		t.Fatalf("pre-root symlink control failed: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	return &rootReplacementAttack{t: t, source: source, moved: moved, outside: outside, info: info}
}

func (a *rootReplacementAttack) attempt() {
	a.t.Helper()
	if a.attempted {
		return
	}
	a.attempted = true
	if err := os.Rename(a.source, a.moved); err != nil {
		const (
			windowsAccessDenied     syscall.Errno = 5
			windowsSharingViolation syscall.Errno = 32
		)
		var linkErr *os.LinkError
		var errno syscall.Errno
		if runtime.GOOS != "windows" || !errors.As(err, &linkErr) || linkErr.Op != "rename" ||
			!errors.As(linkErr.Err, &errno) || (errno != windowsAccessDenied && errno != windowsSharingViolation) {
			a.t.Fatalf("unexpected replacement rename failure: %v", err)
		}
		a.blocked = true
		a.assertBarrierIdentity()
		return
	}
	if err := os.Symlink(a.outside, a.source); err != nil {
		a.t.Fatal(err)
	}
}

func (a *rootReplacementAttack) assertBarrierIdentity() {
	a.t.Helper()
	info, err := os.Lstat(a.source)
	if err != nil || !os.SameFile(a.info, info) || info.Mode()&os.ModeSymlink != 0 {
		a.t.Fatalf("blocked replacement changed original directory identity: %v", err)
	}
	if _, err := os.Lstat(a.moved); !os.IsNotExist(err) {
		a.t.Fatalf("blocked replacement left a moved directory: %v", err)
	}
}

func (a *rootReplacementAttack) assertOutcome(err error) {
	a.t.Helper()
	if !a.attempted {
		a.t.Fatal("operation never reached target recheck boundary")
	}
	if a.blocked {
		a.assertBarrierIdentity()
		if err != nil {
			a.t.Fatalf("operation failed despite native replacement barrier: %v", err)
		}
	} else if err == nil {
		a.t.Fatal("operation accepted a substituted target root")
	}
}

func (a *rootReplacementAttack) originalPath() string {
	if a.blocked {
		return a.source
	}
	return a.moved
}
