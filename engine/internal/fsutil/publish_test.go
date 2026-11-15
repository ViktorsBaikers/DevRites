//go:build (linux || darwin || windows) && (amd64 || arm64)

package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func publicationRoot(t *testing.T) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	for _, dir := range []string{"staging", "work"} {
		if err := root.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func publicationStage(t *testing.T, root *os.Root, name, body string) string {
	t.Helper()
	rel := filepath.Join("staging", name)
	if err := root.Mkdir(rel, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"brief.md", "state.md"} {
		if err := root.WriteFile(filepath.Join(rel, file), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return rel
}

func assertPublicationBody(t *testing.T, root *os.Root, rel, body string) {
	t.Helper()
	for _, file := range []string{"brief.md", "state.md"} {
		got, err := root.ReadFile(filepath.Join(rel, file))
		if err != nil || string(got) != body {
			t.Fatalf("%s/%s = %q, %v; want %q", rel, file, got, err, body)
		}
	}
}

func TestPublishDirNoReplaceAndRetrySync(t *testing.T) {
	root := publicationRoot(t)
	stage := publicationStage(t, root, "first", "complete")
	if err := root.Mkdir(filepath.Join(stage, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile(filepath.Join(stage, "nested", "input"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}
	published, err := PublishDirNoReplace(root, stage, filepath.Join("work", "target"))
	if !published || err != nil {
		t.Fatalf("publish = %v, %v", published, err)
	}
	assertPublicationBody(t, root, filepath.Join("work", "target"), "complete")
	if _, err := root.Lstat(stage); !os.IsNotExist(err) {
		t.Fatalf("staging entry remains: %v", err)
	}
	if err := SyncPublishedDir(root, filepath.Join("work", "target")); err != nil {
		t.Fatal(err)
	}
	if got, err := root.ReadFile(filepath.Join("work", "target", "nested", "input")); err != nil || string(got) != "nested" {
		t.Fatalf("nested content = %q, %v", got, err)
	}
}

func TestPublishDirNoReplaceCollisions(t *testing.T) {
	for _, kind := range []string{"empty-directory", "populated-directory", "file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := publicationRoot(t)
			stage := publicationStage(t, root, "candidate", "candidate")
			target := filepath.Join("work", "target")
			switch kind {
			case "empty-directory", "populated-directory":
				if err := root.Mkdir(target, 0o755); err != nil {
					t.Fatal(err)
				}
				if kind == "populated-directory" {
					if err := root.WriteFile(filepath.Join(target, "existing"), []byte("preserved"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			case "file":
				if err := root.WriteFile(target, []byte("preserved"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := root.Symlink(filepath.Join("..", "staging", "candidate"), target); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			before, err := root.Lstat(target)
			if err != nil {
				t.Fatal(err)
			}
			published, err := PublishDirNoReplace(root, stage, target)
			if published || err == nil {
				t.Fatalf("collision = %v, %v", published, err)
			}
			after, err := root.Lstat(target)
			if err != nil || !os.SameFile(before, after) {
				t.Fatalf("collision changed destination: %v", err)
			}
			assertPublicationBody(t, root, stage, "candidate")
			if kind == "file" {
				if got, err := root.ReadFile(target); err != nil || string(got) != "preserved" {
					t.Fatalf("existing file changed: %q, %v", got, err)
				}
			}
			if kind == "populated-directory" {
				if got, err := root.ReadFile(filepath.Join(target, "existing")); err != nil || string(got) != "preserved" {
					t.Fatalf("existing directory changed: %q, %v", got, err)
				}
			}
		})
	}
}

func TestPublishDirConcurrentPublishers(t *testing.T) {
	root := publicationRoot(t)
	const count = 12
	stages := make([]string, count)
	for i := range stages {
		stages[i] = publicationStage(t, root, fmt.Sprint(i), fmt.Sprint(i))
	}
	type result struct {
		index     int
		published bool
		err       error
	}
	results := make(chan result, count)
	var workers sync.WaitGroup
	for i, stage := range stages {
		workers.Add(1)
		go func() {
			defer workers.Done()
			published, err := PublishDirNoReplace(root, stage, filepath.Join("work", "target"))
			results <- result{i, published, err}
		}()
	}
	workers.Wait()
	close(results)
	winners := 0
	for result := range results {
		if result.published {
			winners++
			if result.err != nil {
				t.Fatalf("winner sync failed: %v", result.err)
			}
			assertPublicationBody(t, root, filepath.Join("work", "target"), fmt.Sprint(result.index))
		} else {
			if result.err == nil {
				t.Fatal("loser has no error")
			}
			assertPublicationBody(t, root, stages[result.index], fmt.Sprint(result.index))
		}
	}
	if winners != 1 {
		t.Fatalf("published %d candidates, want 1", winners)
	}
}

func TestPublishDirInjectedFailures(t *testing.T) {
	injected := errors.New("injected durability failure")
	// Two files, stage directory, two preflight parents, two post-rename parents.
	for failAt := 1; failAt <= 7; failAt++ {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			root := publicationRoot(t)
			stage := publicationStage(t, root, "candidate", "complete")
			target := filepath.Join("work", "target")
			calls := 0
			published, err := publishDirNoReplace(root, stage, target, nativePublishRename, func(file *os.File) error {
				calls++
				if calls == failAt {
					return injected
				}
				return nativePublishSync(file)
			})
			if !errors.Is(err, injected) || published != (failAt >= 6) {
				t.Fatalf("failure %d = %v, %v", failAt, published, err)
			}
			if published {
				assertPublicationBody(t, root, target, "complete")
				if !strings.Contains(err.Error(), "preserve target") {
					t.Fatalf("missing recovery: %v", err)
				}
				if calls != 7 {
					t.Fatalf("did not attempt both post-rename syncs: %d", calls)
				}
				if err := SyncPublishedDir(root, target); err != nil {
					t.Fatal(err)
				}
			} else {
				assertPublicationBody(t, root, stage, "complete")
				if _, err := root.Lstat(target); !os.IsNotExist(err) {
					t.Fatalf("partial destination after failure: %v", err)
				}
			}
		})
	}
	t.Run("rename", func(t *testing.T) {
		root := publicationRoot(t)
		stage := publicationStage(t, root, "candidate", "complete")
		published, err := publishDirNoReplace(root, stage, filepath.Join("work", "target"), func(_, _, _ *os.File, _, _ string) error { return injected }, nativePublishSync)
		if published || !errors.Is(err, injected) {
			t.Fatalf("rename failure = %v, %v", published, err)
		}
		assertPublicationBody(t, root, stage, "complete")
		if _, err := root.Lstat(filepath.Join("work", "target")); !os.IsNotExist(err) {
			t.Fatalf("rename failure published target: %v", err)
		}
	})
}

func TestSyncPublishedDirDoesNotConcealRetryFailure(t *testing.T) {
	for failAt := 1; failAt <= 4; failAt++ {
		t.Run(fmt.Sprint(failAt), func(t *testing.T) {
			root := publicationRoot(t)
			stage := publicationStage(t, root, "candidate", "complete")
			target := filepath.Join("work", "target")
			if published, err := PublishDirNoReplace(root, stage, target); !published || err != nil {
				t.Fatalf("publish = %v, %v", published, err)
			}
			injected := errors.New("retry sync failed")
			calls := 0
			err := syncPublishedDir(root, target, func(file *os.File) error {
				calls++
				if calls == failAt {
					return injected
				}
				return nativePublishSync(file)
			})
			if !errors.Is(err, injected) {
				t.Fatalf("retry concealed error: %v", err)
			}
			assertPublicationBody(t, root, target, "complete")
		})
	}
}

func TestPublishDirRejectsUnsafePaths(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../outside", "/absolute", "work/../target", "work//target", "work/target/", "work/target.", "work/target ", "work/target:stream", "work/target\x00"} {
		t.Run(fmt.Sprintf("%q", bad), func(t *testing.T) {
			root := publicationRoot(t)
			stage := publicationStage(t, root, "candidate", "complete")
			for _, paths := range [][2]string{{stage, bad}, {bad, filepath.Join("work", "target")}} {
				if published, err := PublishDirNoReplace(root, paths[0], paths[1]); published || err == nil {
					t.Fatalf("unsafe paths %q = %v, %v", paths, published, err)
				}
			}
			if err := SyncPublishedDir(root, bad); err == nil {
				t.Fatal("unsafe retry accepted")
			}
			assertPublicationBody(t, root, stage, "complete")
		})
	}
}

func TestPublishDirRejectsSymlinkComponents(t *testing.T) {
	for _, part := range []string{"source-parent", "target-parent", "stage", "content"} {
		t.Run(part, func(t *testing.T) {
			root := publicationRoot(t)
			stage := publicationStage(t, root, "candidate", "complete")
			target := filepath.Join("work", "target")
			var link, referent string
			switch part {
			case "source-parent":
				link, referent = "alias", "staging"
				stage = filepath.Join("alias", "candidate")
			case "target-parent":
				link, referent = "alias", "work"
				target = filepath.Join("alias", "target")
			case "stage":
				link, referent = filepath.Join("staging", "alias"), "candidate"
				stage = link
			case "content":
				link, referent = filepath.Join(stage, "alias"), "brief.md"
			}
			if err := root.Symlink(referent, link); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			if published, err := PublishDirNoReplace(root, stage, target); published || err == nil {
				t.Fatalf("symlink accepted: %v, %v", published, err)
			}
			if err := SyncPublishedDir(root, stage); part != "target-parent" && err == nil {
				t.Fatal("symlink accepted on retry")
			}
		})
	}
}

func TestPublishDirRetainsNativeParentHandles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows forbids moving a directory with some open descendants; Unix handle-swap regression")
	}
	root := publicationRoot(t)
	stage := publicationStage(t, root, "candidate", "complete")
	outside := t.TempDir()
	published, err := publishDirNoReplace(root, stage, filepath.Join("work", "target"), func(source, dir, target *os.File, from, to string) error {
		if err := root.Rename("work", "retained-work"); err != nil {
			return err
		}
		if err := root.Symlink(outside, "work"); err != nil {
			return err
		}
		if err := root.Rename("staging", "retained-staging"); err != nil {
			return err
		}
		if err := root.Symlink(outside, "staging"); err != nil {
			return err
		}
		return nativePublishRename(source, dir, target, from, to)
	}, nativePublishSync)
	if !published || err != nil {
		t.Fatalf("retained parent publish = %v, %v", published, err)
	}
	assertPublicationBody(t, root, filepath.Join("retained-work", "target"), "complete")
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("publication escaped native parents: %v, %v", entries, err)
	}
}

func TestPublicationRefusesNonDirectoryAndSelf(t *testing.T) {
	root := publicationRoot(t)
	stage := publicationStage(t, root, "candidate", "complete")
	if published, err := PublishDirNoReplace(root, stage, stage); published || !errors.Is(err, os.ErrExist) {
		t.Fatalf("self-publication = %v, %v", published, err)
	}
	if published, err := PublishDirNoReplace(root, filepath.Join(stage, "brief.md"), filepath.Join("work", "target")); published || err == nil {
		t.Fatalf("non-directory publication = %v, %v", published, err)
	}
	if published, err := PublishDirNoReplace(root, stage, filepath.Join("missing", "target")); published || err == nil {
		t.Fatalf("missing parent = %v, %v", published, err)
	}
	if err := SyncPublishedDir(root, filepath.Join(stage, "brief.md")); err == nil {
		t.Fatal("retry accepted regular file as directory")
	}
	if err := SyncPublishedDir(root, filepath.Join("work", "missing")); err == nil {
		t.Fatal("retry accepted missing publication")
	}
	assertPublicationBody(t, root, stage, "complete")
}
