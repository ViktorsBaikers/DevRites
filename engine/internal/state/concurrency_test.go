package state

// Issue 07: prove the concurrency primitives. These exercise in-process
// contention (goroutines); the multi-process CLI stress test lives beside the
// binary (concurrency_cli_test.go).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestAppendLogConcurrentNoInterleaveNoLoss spawns many concurrent appenders,
// each writing a distinct small record, and asserts every record landed exactly
// once and intact: no interleaving, no lost writes.
func TestAppendLogConcurrentNoInterleaveNoLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.log")
	const n = 64

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := AppendLog(path, fmt.Sprintf("record-%03d-fixed-width-payload", i)); err != nil {
				t.Errorf("append %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != n {
		t.Fatalf("got %d lines, want %d (lost or torn writes)", len(lines), n)
	}
	seen := make(map[string]bool, n)
	for _, l := range lines {
		if !strings.HasPrefix(l, "record-") || !strings.HasSuffix(l, "-fixed-width-payload") {
			t.Fatalf("interleaved/torn record: %q", l)
		}
		if seen[l] {
			t.Fatalf("duplicate record: %q", l)
		}
		seen[l] = true
	}
}

// TestAtomicWriteReaderNeverSeesPartial has many writers overwrite one path with
// distinct full-size contents while readers read concurrently. A reader must
// always observe a complete, self-consistent file: never a half-written or
// mixed one: which is exactly what temp-file + atomic rename guarantees.
func TestAtomicWriteReaderNeverSeesPartial(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not allow replacing a file while another handle is reading it")
	}
	path := filepath.Join(t.TempDir(), "structured.md")
	const size = 8192
	if err := AtomicWrite(path, bytes.Repeat([]byte{'A'}, size), 0o644); err != nil {
		t.Fatal(err)
	}

	var writers, readers sync.WaitGroup
	stop := make(chan struct{})

	// Writers: each writes a homogeneous block of a distinct byte until stopped.
	for _, b := range []byte("BCDEFGHIJKLMNOP") {
		writers.Add(1)
		go func(b byte) {
			defer writers.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if err := AtomicWrite(path, bytes.Repeat([]byte{b}, size), 0o644); err != nil {
						t.Errorf("write %c: %v", b, err)
						return
					}
				}
			}
		}(b)
	}

	// Readers: every read must be full-size and homogeneous (one writer's block),
	// proving no partial or interleaved content is ever visible.
	for r := 0; r < 4; r++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for i := 0; i < 2000; i++ {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("read: %v", err)
					return
				}
				if len(data) != size {
					t.Errorf("partial read: len=%d, want %d", len(data), size)
					return
				}
				first := data[0]
				for _, c := range data {
					if c != first {
						t.Errorf("torn read: mixed bytes %c and %c", first, c)
						return
					}
				}
			}
		}()
	}

	readers.Wait() // readers finish their fixed loops; then stop the writers
	close(stop)
	writers.Wait()

	// No temp files left behind after all writes.
	litter, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".*.tmp-*"))
	if len(litter) != 0 {
		t.Errorf("atomic write left temp litter: %v", litter)
	}
}

// TestWithFeatureLockSerializesReadModifyWrite runs a lost-update-prone counter
// increment concurrently under the per-feature lock. Serialized correctly, the
// final value is exactly the number of increments.
func TestWithFeatureLockSerializesReadModifyWrite(t *testing.T) {
	root := t.TempDir()
	slug := "counter"
	if err := os.MkdirAll(featureDir(root, slug), 0o755); err != nil {
		t.Fatal(err)
	}
	counter := filepath.Join(featureDir(root, slug), "count")
	if err := os.WriteFile(counter, []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithFeatureLock(root, slug, func() error {
				raw, err := os.ReadFile(counter)
				if err != nil {
					return err
				}
				v, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
				return os.WriteFile(counter, []byte(strconv.Itoa(v+1)), 0o644)
			})
			if err != nil {
				t.Errorf("locked increment: %v", err)
			}
		}()
	}
	wg.Wait()

	raw, _ := os.ReadFile(counter)
	if got := strings.TrimSpace(string(raw)); got != strconv.Itoa(n) {
		t.Fatalf("counter = %s, want %d: lost updates under contention", got, n)
	}
}

// TestAtomicWriteFailsCleanlyOnMissingDir asserts a write whose directory does
// not exist returns an error rather than a partial file, and leaves no temp
// litter to find.
func TestAtomicWriteFailsCleanlyOnMissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "file.md")
	if err := AtomicWrite(path, []byte("data"), 0o644); err == nil {
		t.Fatal("expected an error writing into a missing directory")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a failed atomic write left a file behind")
	}
}
