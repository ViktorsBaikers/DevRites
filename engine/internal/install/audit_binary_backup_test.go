//go:build !windows

package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestInstallBinaryBackupOwnership(t *testing.T) {
	for _, outcome := range []string{"installed", "restored", "restore-failed"} {
		t.Run(outcome, func(t *testing.T) {
			binDir := t.TempDir()
			dest := filepath.Join(binDir, engineBinaryName())
			old := "#!/bin/sh\necho 1.0.0\n"
			testutil.WriteExecutable(t, dest, old)
			if err := os.Chmod(dest, 0o700); err != nil {
				t.Fatal(err)
			}
			staged := filepath.Join(t.TempDir(), "candidate")
			installedAction := "echo 1.2.3"
			if outcome == "restored" {
				installedAction = "echo 1.2.4"
			} else if outcome == "restore-failed" {
				installedAction = "/bin/rm -- \"$0\" && /bin/mkdir -- \"$0\" && printf blocked > \"$0/blocker\"; echo 1.2.4"
			}
			body := fmt.Sprintf("#!/bin/sh\ncase \"$0\" in %q) %s;; *) echo 1.2.3;; esac\n", dest, installedAction)
			testutil.WriteExecutable(t, staged, body)
			t.Setenv("DEVRITES_NO_BINARY", "")
			t.Setenv("DEVRITES_BIN_DIR", binDir)
			r := runner{opts: DefaultOptions(ModeInstall), preparedBinary: staged, requiredBinaryTag: "v1.2.3"}

			err := r.installBinary()
			backups, globErr := filepath.Glob(filepath.Join(binDir, "."+engineBinaryName()+".backup-*"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if outcome == "restore-failed" {
				if err == nil || !strings.Contains(err.Error(), "version mismatch") || !strings.Contains(err.Error(), "restore previous binary") {
					t.Fatalf("install error = %v, want verification and restore failures", err)
				}
				if len(backups) != 1 {
					t.Fatalf("recoverable backups = %v, want one saved previous binary", backups)
				}
				if got := testutil.ReadFile(t, backups[0]); got != old {
					t.Fatalf("backup bytes = %q, want %q", got, old)
				}
				info, statErr := os.Stat(backups[0])
				if statErr != nil {
					t.Fatal(statErr)
				}
				if info.Mode().Perm() != 0o700 {
					t.Fatalf("backup mode = %v, want 0700", info.Mode().Perm())
				}
				if !strings.Contains(err.Error(), backups[0]) {
					t.Fatalf("install error = %v, want recoverable backup location %s", err, backups[0])
				}
				return
			}
			if len(backups) != 0 {
				t.Fatalf("completed install/restore left backups: %v", backups)
			}
			want, mode := body, os.FileMode(0o755)
			if outcome == "restored" {
				if err == nil || !strings.Contains(err.Error(), "previous binary restored") {
					t.Fatalf("install error = %v, want successful rollback", err)
				}
				want, mode = old, 0o700
			} else if err != nil {
				t.Fatal(err)
			}
			if got := testutil.ReadFile(t, dest); got != want {
				t.Fatalf("installed bytes = %q, want %q", got, want)
			}
			info, statErr := os.Stat(dest)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if info.Mode().Perm() != mode {
				t.Fatalf("installed mode = %v, want %v", info.Mode().Perm(), mode)
			}
		})
	}
}
