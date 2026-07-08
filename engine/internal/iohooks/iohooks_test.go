package iohooks

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebFetchPayloadReaders(t *testing.T) {
	data := []byte(`{"tool_input":{"url":"https://example.com","prompt":"summarize"},"tool_response":{"output":"body text"}}`)
	if got := webFetchURL(data); got != "https://example.com" {
		t.Fatalf("webFetchURL=%q", got)
	}
	if got := webFetchPrompt(data); got != "summarize" {
		t.Fatalf("webFetchPrompt=%q", got)
	}
	if got := webFetchContent(data); got != "body text" {
		t.Fatalf("webFetchContent object=%q", got)
	}
	if got := webFetchContent([]byte(`{"tool_response":"plain body"}`)); got != "plain body" {
		t.Fatalf("webFetchContent string=%q", got)
	}
	if got := webFetchContent([]byte(`{"tool_response":{"unknown":"x"}}`)); got != "" {
		t.Fatalf("webFetchContent unknown=%q, want empty", got)
	}
}

func TestSourceCachePostFailOpenWithoutValidator(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", t.TempDir())
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := SourceCachePost(strings.NewReader(`{"tool_input":{"url":"http://127.0.0.1:1/nope"},"tool_response":"content"}`), stdout, stderr)
	if code != exitOK {
		t.Fatalf("SourceCachePost code=%d, want %d", code, exitOK)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("SourceCachePost wrote stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestSourceCachePreIgnoresUnvalidatedEntry(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_PROJECT_DIR", root)
	url := "https://example.com/a"
	entryPath := sourceCachePath(url)
	if !strings.HasPrefix(entryPath, filepath.Join(root, ".devrites", "source-cache")) {
		t.Fatalf("sourceCachePath=%q is not under project cache", entryPath)
	}
	if err := os.MkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(sourceCacheEntry{URL: url, Content: "cached without validator"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entryPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	code := SourceCachePre(strings.NewReader(`{"tool_input":{"url":"https://example.com/a"}}`), &bytes.Buffer{}, &bytes.Buffer{})
	if code != exitOK {
		t.Fatalf("SourceCachePre code=%d, want fail-open %d", code, exitOK)
	}

	sum := sha256.Sum256([]byte(url))
	wantName := hex.EncodeToString(sum[:])[:32] + ".json"
	if filepath.Base(entryPath) != wantName {
		t.Fatalf("cache filename=%q, want %q", filepath.Base(entryPath), wantName)
	}
}

func TestRefreshStateAndChangeScan(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".codegraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	st := computeRefreshState(root)
	if st.state != filepath.Join(root, ".codegraph") {
		t.Fatalf("state=%q, want .codegraph", st.state)
	}

	stamp := filepath.Join(root, ".codegraph", ".index_refresh_stamp")
	if err := os.WriteFile(stamp, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stamp, old, old); err != nil {
		t.Fatal(err)
	}
	if repoChangedSince(root, stamp) {
		t.Fatalf("repoChangedSince=true immediately after stamp, want false")
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !repoChangedSince(root, stamp) {
		t.Fatalf("repoChangedSince=false after tracked file write, want true")
	}
}
