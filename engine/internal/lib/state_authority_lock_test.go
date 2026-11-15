//go:build unix || windows

package lib

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/parallel"
)

func TestAuthorityMutationsWaitForControllerBeforeFeatureLock(t *testing.T) {
	for _, tc := range []struct {
		name, entrypoint, file, before, after string
	}{
		{"set", "RunStateSet", "state.md", editState + "| sequence_workspaces_remaining | 2 |\n", "| sequence_workspaces_remaining | 0 |"},
		{"unset", "RunStateSet", "state.md", editState + "| sequence_parent | predecessor |\n", ""},
		{"append", "RunStateAppend", "decisions.md", "# Decisions\n", "Sequence openings revoked."},
		{"bundle-roadmap", "RunApplyBundle", "roadmap.md", "# Roadmap\n\nApproved future work.\n", "# Roadmap\n\nFuture work withdrawn.\n"},
		{"bundle-decisions", "RunApplyBundle", "decisions.md", "# Decisions\n\nSequence approved.\n", "# Decisions\n\nSequence withdrawn.\n"},
		{"bundle-cursor", "RunApplyBundle", "state.md", editState + "| sequence_workspaces_remaining | 2 |\n", editState + "| sequence_workspaces_remaining | 0 |\n"},
		{"migrate", "Migrate", "state.md", strings.Replace(editState, "| schema | 4 |", "| schema | 3 |", 1), "| schema | 4 |"},
		{"resolve", "Resolve", "questions.md", "## q-1 blocking\nstatus: open\n", "status: answered"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := editWorkspace(t)
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.before), 0o644); err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			var mutate func() int
			switch tc.name {
			case "set":
				mutate = func() int {
					return RunStateSet(root, []string{"feat", "sequence_workspaces_remaining=0"}, io.Discard, &stderr)
				}
			case "unset":
				mutate = func() int {
					return RunStateSet(root, []string{"feat", "--unset", "sequence_parent"}, io.Discard, &stderr)
				}
			case "append":
				mutate = func() int {
					return RunStateAppend(root, []string{"feat", "decisions.md", "--heading", "Revoked", "--body-file", "-"}, strings.NewReader(tc.after), io.Discard, &stderr)
				}
			case "migrate":
				mutate = func() int {
					return Migrate(root, []string{"feat"}, io.Discard, &stderr)
				}
			case "resolve":
				writeFile(t, filepath.Join(root, "ACTIVE"), "feat\n")
				mutate = func() int {
					return Resolve(root, []string{"q-1", "answer"}, io.Discard, &stderr)
				}
			default:
				bundle := writeBundle(t, dir, []candidateFile{{Path: tc.file, Operation: "replace", Content: &tc.after}}, map[string]string{tc.file: pinOf(t, dir, tc.file)})
				mutate = func() int {
					return RunApplyBundle(root, []string{"feat", bundle}, io.Discard, &stderr)
				}
			}
			done := make(chan int, 1)
			if err := parallel.WithControlLock(root, func() error {
				go func() { done <- mutate() }()
				waitForAuthorityLock(t, tc.entrypoint, done)
				if got := readEdit(t, dir, tc.file); got != tc.before {
					t.Errorf("authority changed inside controller transaction:\n%s", got)
				}
				if _, err := os.Lstat(filepath.Join(root, ".locks", "features", "feat", ".lock")); !os.IsNotExist(err) {
					t.Errorf("feature lock acquired before controller lock: %v", err)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			select {
			case code := <-done:
				if code != 0 {
					t.Fatalf("mutation exit %d: %s", code, stderr.String())
				}
			case <-time.After(10 * time.Second):
				t.Fatal("mutation did not resume after controller transaction")
			}
			got := readEdit(t, dir, tc.file)
			if tc.name == "unset" {
				if strings.Contains(got, "sequence_parent") {
					t.Fatalf("sequence parent not revoked:\n%s", got)
				}
			} else if !strings.Contains(got, tc.after) {
				t.Fatalf("authority change missing after controller transaction:\n%s", got)
			}
		})
	}
}

// Observe the blocked acquisition itself; elapsed time is never evidence of exclusion.
func waitForAuthorityLock(t *testing.T, entrypoint string, done <-chan int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	stacks := make([]byte, 64<<10)
	for {
		select {
		case code := <-done:
			t.Fatalf("%s returned %d while controller transaction held its lock", entrypoint, code)
		default:
		}
		n := runtime.Stack(stacks, true)
		if n == len(stacks) {
			stacks = make([]byte, len(stacks)*2)
			continue
		}
		for _, stack := range strings.Split(string(stacks[:n]), "\n\n") {
			if strings.Contains(stack, "lib."+entrypoint+"(") && strings.Contains(stack, "parallel.WithControlLock(") && strings.Contains(stack, "state.lockOpened(") {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not reach controller lock:\n%s", entrypoint, stacks[:n])
		}
		runtime.Gosched()
	}
}

func TestApplyBundleDryRunDoesNotWaitForController(t *testing.T) {
	root, dir := editWorkspace(t)
	before := readEdit(t, dir, "state.md")
	after := before + "| sequence_workspaces_remaining | 0 |\n"
	bundle := writeBundle(t, dir, []candidateFile{{Path: "state.md", Operation: "replace", Content: &after}}, map[string]string{"state.md": pinOf(t, dir, "state.md")})
	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	if err := parallel.WithControlLock(root, func() error {
		go func() {
			done <- RunApplyBundle(root, []string{"feat", bundle, "--dry-run"}, &stdout, &stderr)
		}()
		select {
		case code := <-done:
			if code != 0 {
				t.Fatalf("dry-run exit %d: %s", code, stderr.String())
			}
		case <-time.After(10 * time.Second):
			t.Fatal("dry-run waited for controller lock")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := readEdit(t, dir, "state.md"); got != before {
		t.Fatalf("dry-run changed cursor:\n%s", got)
	}
	if !strings.Contains(stdout.String(), "dry-run clean") {
		t.Fatalf("missing dry-run result: %s", stdout.String())
	}
}
