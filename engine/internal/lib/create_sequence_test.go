package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/devrites/devrites/internal/fsutil"
)

func sequenceCreationFixture(t *testing.T, built bool) (string, string) {
	t.Helper()
	root, dir := canonicalPreparationFixture(t, built)
	t.Setenv("DEVRITES_WORKSPACE", "")
	sequenceReplace(t, filepath.Join(dir, "state.md"), "| phase |", "| schema | 4 |\n| phase |")
	brief := filepath.Join(t.TempDir(), "approved-brief.md")
	writeFile(t, brief, "# Approved future scope\n\nImplement independent recorded work from DEC-001 without expanding scope.\n")
	return root, brief
}

func sequenceReplace(t *testing.T, path, old, replacement string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), old) {
		t.Fatalf("fixture %s missing %q", path, old)
	}
	writeFile(t, path, strings.ReplaceAll(string(raw), old, replacement))
}

func sequenceCreate(root, target, brief string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := RunCreateSequence(root, []string{"current16", target, "--brief", brief}, &out, &errOut)
	return code, out.String(), errOut.String()
}

func sequenceCreated(t *testing.T, root, target, brief string, position int, parent string) {
	t.Helper()
	code, out, diagnostic := sequenceCreate(root, target, brief)
	want := fmt.Sprintf("sequence-workspace: created target=%s position=%d parent=%s\n", target, position, parent)
	if code != 0 || out != want || diagnostic != "" {
		t.Fatalf("creation: code=%d stdout=%q stderr=%q; want %q", code, out, diagnostic, want)
	}
}

// Include links and special files without opening a FIFO.
func sequenceSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".lock") {
			return nil
		}
		if entry.IsDir() {
			return nil // Lock-only parent directories are not workflow state.
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			result[path] = "link:" + link
			return err
		}
		if entry.Type().IsRegular() {
			raw, err := os.ReadFile(path)
			result[path] = string(raw)
			return err
		}
		result[path] = entry.Type().String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func sequenceRefused(t *testing.T, root, target, brief string) {
	t.Helper()
	before := sequenceSnapshot(t, root)
	path := filepath.Join(root, "work", target)
	info, statErr := os.Lstat(path)
	code, out, diagnostic := sequenceCreate(root, target, brief)
	if code != 3 || out != "" || !strings.Contains(diagnostic, "create-sequence:") {
		t.Fatalf("refusal: code=%d stdout=%q stderr=%q", code, out, diagnostic)
	}
	if !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
		t.Fatal("refusal changed workflow bytes or links")
	}
	after, err := os.Lstat(path)
	if os.IsNotExist(statErr) {
		if !os.IsNotExist(err) {
			t.Fatalf("refusal created destination: %v", err)
		}
	} else if statErr != nil || err != nil || !os.SameFile(info, after) {
		t.Fatalf("refusal replaced destination: before=%v after=%v", statErr, err)
	}
}

func TestCreateSequenceFreshPendingSpec(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	controller := controllerSnapshot(t, filepath.Join(root, "work", "current16"))
	sequenceCreated(t, root, "future17", brief, 2, "current16")
	dir := filepath.Join(root, "work", "future17")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 || entries[0].Name() != "brief.md" || entries[1].Name() != "state.md" {
		t.Fatalf("pending workspace artifacts: %v %v", entries, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "state.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []string{"| phase | spec |", "| status | running |", "| next_action | /rite-spec future17 |", "| schema | 4 |", "| sequence_position | 2 |", "| sequence_parent | current16 |", "| sequence_workspaces_remaining | 2 |"} {
		if !strings.Contains(string(raw), row) {
			t.Fatalf("missing cursor row %q: %s", row, raw)
		}
	}
	seed, err := os.ReadFile(brief)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(dir, "brief.md"))
	if err != nil || !bytes.Equal(seed, copied) {
		t.Fatalf("brief changed: %q %v", copied, err)
	}
	text := frontierText(t, root, "current16")
	if !strings.Contains(text, "frontier-workspace: future17 location=live") || !strings.Contains(text, "frontier-concurrent: /rite-spec future17") || strings.Contains(text, "frontier-create: controller=current16 target=future17 ") {
		t.Fatalf("pending Spec not discovered: %s", text)
	}
	ready, notices, err := WorkspaceReady(filepath.Dir(root), "current16", "future17")
	if err != nil || len(ready) != 0 || len(notices) == 0 {
		t.Fatalf("pending Spec readiness: %v %v %v", ready, notices, err)
	}
	active, err := os.ReadFile(filepath.Join(root, "ACTIVE"))
	if err != nil || string(active) != "current16\n" || !reflect.DeepEqual(controller, controllerSnapshot(t, filepath.Join(root, "work", "current16"))) {
		t.Fatal("creation moved ACTIVE or changed controller")
	}
}

func TestCreateSequenceRollingPool(t *testing.T) {
	for _, built := range []bool{true, false} {
		t.Run(fmt.Sprintf("built=%v", built), func(t *testing.T) {
			root, brief := sequenceCreationFixture(t, built)
			sequenceCreated(t, root, "future17", brief, 2, "current16")
			if !built {
				sequenceRefused(t, root, "future18", brief)
				return
			}
			sequenceCreated(t, root, "future18", brief, 3, "future17")
			raw, err := os.ReadFile(filepath.Join(root, "work", "future18", "state.md"))
			if err != nil || !strings.Contains(string(raw), "| sequence_workspaces_remaining | 1 |") {
				t.Fatalf("derived second budget: %s %v", raw, err)
			}
			sequenceRefused(t, root, "future19", brief)
		})
	}
}

func TestCreateSequenceReusePreservesAdvancedWorkspace(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	sequenceCreated(t, root, "future17", brief, 2, "current16")
	sequenceCreated(t, root, "future18", brief, 3, "future17")
	dir := filepath.Join(root, "work", "future17")
	sequenceReplace(t, filepath.Join(dir, "state.md"), "| phase | spec |", "| phase | vet |")
	sequenceReplace(t, filepath.Join(dir, "state.md"), "/rite-spec future17", "/rite-vet future17")
	writeFile(t, filepath.Join(dir, "brief.md"), "# Evolved native brief\n\nRecorded scope refined with reviewed contracts.\n")
	writeFile(t, filepath.Join(dir, "plan.md"), "# Native plan\n\nPreserve exactly.\n")
	for _, policy := range []string{"mode: autocomplete\nmax_parallel: 2\n", "mode: autocomplete\nmax_parallel: 2\ncontinue_sequence: false\n", "mode: autocomplete\nmax_parallel: 1\n"} {
		writeFile(t, filepath.Join(root, "AFK"), policy)
		sequenceReplace(t, filepath.Join(root, "work", "current16", "state.md"), "| sequence_workspaces_remaining | 3 |", "| sequence_workspaces_remaining | 0 |")
		before := sequenceSnapshot(t, root)
		code, out, diagnostic := sequenceCreate(root, "future17", filepath.Join(t.TempDir(), "missing.md"))
		if code != 0 || out != "sequence-workspace: reused target=future17 position=2 parent=current16\n" || diagnostic != "" || !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
			t.Fatalf("reuse changed bytes or charged again: %d %q %q", code, out, diagnostic)
		}
		sequenceReplace(t, filepath.Join(root, "work", "current16", "state.md"), "| sequence_workspaces_remaining | 0 |", "| sequence_workspaces_remaining | 3 |")
	}
}

func TestCreateSequenceRefusalPreservesWorkflow(t *testing.T) {
	for _, kind := range []string{"zero budget", "opt out", "serial", "source override", "missing authority", "invalid authority", "missing approval", "old controller schema", "missing predecessor", "stale ancestry", "archived", "partial", "file", "symlink", "conflicting identity", "old target schema"} {
		t.Run(kind, func(t *testing.T) {
			root, brief := sequenceCreationFixture(t, true)
			dir := filepath.Join(root, "work", "current16")
			target := "future17"
			switch kind {
			case "zero budget":
				sequenceReplace(t, filepath.Join(dir, "state.md"), "remaining | 3", "remaining | 0")
			case "opt out":
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\ncontinue_sequence: false\n")
			case "serial":
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 1\n")
			case "source override":
				t.Setenv("DEVRITES_WORKSPACE", dir)
			case "missing authority":
				if err := os.Remove(filepath.Join(dir, "roadmap.md")); err != nil {
					t.Fatal(err)
				}
			case "invalid authority":
				sequenceReplace(t, filepath.Join(dir, "roadmap.md"), "| 3 | future18 |", "| 2 | future18 |")
			case "missing approval":
				writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n")
			case "old controller schema":
				sequenceReplace(t, filepath.Join(dir, "state.md"), "| schema | 4 |", "| schema | 3 |")
			case "missing predecessor":
				target = "future18"
			case "stale ancestry":
				sequenceCreated(t, root, "future17", brief, 2, "current16")
				sequenceReplace(t, filepath.Join(root, "work", "future17", "state.md"), "| sequence_parent | current16 |", "| sequence_parent | absent |")
				target = "future18"
			case "archived":
				sequenceCreated(t, root, "future17", brief, 2, "current16")
				if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(root, "work", target), filepath.Join(root, "archive", target)); err != nil {
					t.Fatal(err)
				}
			case "partial":
				writeFile(t, filepath.Join(root, "work", target, "brief.md"), "Preserve partial brief\n")
			case "file":
				writeFile(t, filepath.Join(root, "work", target), "Preserve destination file\n")
			case "symlink":
				if err := os.Symlink(dir, filepath.Join(root, "work", target)); err != nil {
					t.Fatal(err)
				}
			case "conflicting identity", "old target schema":
				sequenceCreated(t, root, target, brief, 2, "current16")
				path := filepath.Join(root, "work", target, "state.md")
				if kind == "conflicting identity" {
					sequenceReplace(t, path, "| sequence_position | 2 |", "| sequence_position | 4 |")
				} else {
					sequenceReplace(t, path, "| schema | 4 |", "| schema | 3 |")
				}
			}
			sequenceRefused(t, root, target, brief)
		})
	}
}

func TestCreateSequenceRejectsInvalidBrief(t *testing.T) {
	for _, kind := range []string{"missing", "empty", "whitespace", "oversize", "directory", "symlink", "invalid UTF-8", "NUL"} {
		t.Run(kind, func(t *testing.T) {
			root, brief := sequenceCreationFixture(t, true)
			switch kind {
			case "missing":
				brief += ".missing"
			case "empty":
				writeFile(t, brief, "")
			case "whitespace":
				writeFile(t, brief, " \n	")
			case "oversize":
				if err := os.Truncate(brief, workspaceArtifactLimit+1); err != nil {
					t.Fatal(err)
				}
			case "directory":
				brief = filepath.Dir(brief)
			case "symlink":
				link := brief + ".link"
				if err := os.Symlink(brief, link); err != nil {
					t.Fatal(err)
				}
				brief = link
			case "invalid UTF-8":
				writeFile(t, brief, "scope\xff")
			case "NUL":
				writeFile(t, brief, "scope\x00hidden")
			}
			sequenceRefused(t, root, "future17", brief)
		})
	}
}

func TestCreateSequenceConcurrentCallersConverge(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	const callers = 8
	type result struct {
		code            int
		out, diagnostic string
	}
	results := make([]result, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i].code, results[i].out, results[i].diagnostic = sequenceCreate(root, "future17", brief)
		}(i)
	}
	close(start)
	wg.Wait()
	created, reused := 0, 0
	for _, r := range results {
		if r.code != 0 || r.diagnostic != "" {
			t.Fatalf("concurrent caller: %+v", r)
		}
		switch r.out {
		case "sequence-workspace: created target=future17 position=2 parent=current16\n":
			created++
		case "sequence-workspace: reused target=future17 position=2 parent=current16\n":
			reused++
		default:
			t.Fatalf("unexpected identity: %+v", r)
		}
	}
	if created != 1 || reused != callers-1 {
		t.Fatalf("created=%d reused=%d", created, reused)
	}
	sequenceCreated(t, root, "future18", brief, 3, "future17")
}

func TestCreateSequencePublicationFailureCleansOwnedStage(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	before := sequenceSnapshot(t, root)
	var out, diagnostic bytes.Buffer
	calls := 0
	publish := func(selected *os.Root, from, to string) (bool, error) {
		calls++
		if !strings.HasPrefix(from, sequenceStagingDir+string(filepath.Separator)) || to != filepath.Join("work", "future17") {
			t.Fatalf("unsafe publication paths: %s %s", from, to)
		}
		for _, name := range []string{"brief.md", "state.md"} {
			if _, err := selected.Stat(filepath.Join(from, name)); err != nil {
				t.Fatal(err)
			}
		}
		return false, errors.New("injected publication failure")
	}
	code := runCreateSequence(root, []string{"current16", "future17", "--brief", brief}, &out, &diagnostic, publish, fsutil.SyncPublishedDir)
	if code != 3 || calls != 1 || out.Len() != 0 || !strings.Contains(diagnostic.String(), "injected publication failure") {
		t.Fatalf("failure: %d %s %s", code, &out, &diagnostic)
	}
	if !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
		t.Fatal("failed publication changed workflow")
	}
	if _, err := os.Lstat(filepath.Join(root, "work", "future17")); !os.IsNotExist(err) {
		t.Fatalf("partial target: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, sequenceStagingDir))
	if err != nil || len(entries) != 0 {
		t.Fatalf("owned stage leaked: %v %v", entries, err)
	}
	if strings.Contains(frontierText(t, root, "current16"), "frontier-workspace: future17 location=live") {
		t.Fatal("failed target discovered")
	}
	sequenceCreated(t, root, "future17", brief, 2, "current16")
}

func TestCreateSequencePublishedSyncFailureRequiresDurableRetry(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	args := []string{"current16", "future17", "--brief", brief}
	var out, diagnostic bytes.Buffer
	publish := func(selected *os.Root, from, to string) (bool, error) {
		published, err := fsutil.PublishDirNoReplace(selected, from, to)
		if err != nil {
			return published, err
		}
		return true, errors.New("injected post-publication sync failure")
	}
	code := runCreateSequence(root, args, &out, &diagnostic, publish, fsutil.SyncPublishedDir)
	if code != 3 || out.String() != "sequence-workspace: created target=future17 position=2 parent=current16\n" || !strings.Contains(diagnostic.String(), "identity already published") {
		t.Fatalf("lost published identity: %d %s %s", code, &out, &diagnostic)
	}
	before := sequenceSnapshot(t, root)
	for _, failPath := range []string{filepath.Join("work", "future17"), sequenceStagingDir} {
		out.Reset()
		diagnostic.Reset()
		called := false
		syncFail := func(selected *os.Root, path string) error {
			if path == failPath {
				called = true
				return errors.New("still uncertain")
			}
			return fsutil.SyncPublishedDir(selected, path)
		}
		neverPublish := func(*os.Root, string, string) (bool, error) { t.Fatal("retry republished identity"); return false, nil }
		code = runCreateSequence(root, args, &out, &diagnostic, neverPublish, syncFail)
		if code != 3 || !called || out.String() != "sequence-workspace: reused target=future17 position=2 parent=current16\n" || !strings.Contains(diagnostic.String(), "still uncertain") || !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
			t.Fatalf("retry hid uncertainty: %d %s %s", code, &out, &diagnostic)
		}
	}
	code, stdout, stderr := sequenceCreate(root, "future17", brief)
	if code != 0 || !strings.Contains(stdout, "reused target=future17") || stderr != "" || !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
		t.Fatalf("durable recovery: %d %q %q", code, stdout, stderr)
	}
}

func TestCreateSequenceRejectsStalePredecessorSchema(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	sequenceCreated(t, root, "future17", brief, 2, "current16")
	sequenceReplace(t, filepath.Join(root, "work", "future17", "state.md"), "| schema | 4 |", "| schema | 3 |")
	sequenceRefused(t, root, "future18", brief)
}

func TestCreateSequenceUsageDoesNotWrite(t *testing.T) {
	root, brief := sequenceCreationFixture(t, true)
	before := sequenceSnapshot(t, root)
	for _, args := range [][]string{nil, {"current16", "future17"}, {"current16", "current16", "--brief", brief}, {"current16", "../escape", "--brief", brief}, {"current16", "future17", "--brief", "-"}, {"current16", "future17", "--brief", brief, "--unknown"}} {
		var out, diagnostic bytes.Buffer
		if code := RunCreateSequence(root, args, &out, &diagnostic); code != 2 || out.Len() != 0 || !strings.Contains(diagnostic.String(), CreateSequenceUsage) {
			t.Fatalf("usage %v: %d %s %s", args, code, &out, &diagnostic)
		}
		if !reflect.DeepEqual(before, sequenceSnapshot(t, root)) {
			t.Fatal("invalid arguments changed workflow")
		}
	}
}
