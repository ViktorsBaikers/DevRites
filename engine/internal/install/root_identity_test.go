package install

import (
	"os"
	"path/filepath"
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
				// Probe symlink support before installing the race hook.
				probe := filepath.Join(parent, "probe")
				if err := os.Symlink(outside, probe); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				if err := os.Remove(probe); err != nil {
					t.Fatal(err)
				}
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
				swapped := false
				t.Cleanup(func() { afterRecheckPath = nil })
				afterRecheckPath = func(path string) {
					if swapped || path != rel {
						return
					}
					swapped = true
					if err := os.Rename(swapPath, filepath.Join(parent, "beta")); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(outside, swapPath); err != nil {
						t.Fatal(err)
					}
				}
				// Refusing a changed target is safe; completing against its held inode is safe too.
				_ = Apply(opts)
				afterRecheckPath = nil
				if !swapped {
					t.Fatal("operation never reached target recheck boundary")
				}
				if got := testutil.ReadFile(t, victim); got != "outside alpha\n" {
					t.Fatalf("outside victim changed: %q", got)
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
