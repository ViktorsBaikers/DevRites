package main_test

// CLI black-box tests for the IO hooks (source-cache + refresh-indexes). The cache
// hooks make real HTTP requests against an httptest server, so the 304-revalidation
// contract is exercised end to end through the built binary.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sourceCacheEntryPath mirrors the engine's cache-key derivation so a test can seed
// or read an entry.
func sourceCacheEntryPath(proj, url string) string {
	sum := sha256.Sum256([]byte(url))
	key := hex.EncodeToString(sum[:])[:32]
	return filepath.Join(proj, ".devrites", "source-cache", key+".json")
}

func seedSourceCache(t *testing.T, proj, url string, entry map[string]string) {
	t.Helper()
	p := sourceCacheEntryPath(proj, url)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(entry)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func projEnv(proj string, extra ...string) []string {
	return append([]string{"CLAUDE_PROJECT_DIR=" + proj}, extra...)
}

func TestHookSourceCachePreServesOn304(t *testing.T) {
	proj := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "etag-1" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", "etag-2")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	seedSourceCache(t, proj, srv.URL, map[string]string{
		"url": srv.URL, "etag": "etag-1", "content": "CACHED READING OF THE PAGE",
		"fetched_at": "2026-01-01T00:00:00Z", "prompt": "how does X work",
	})
	payload := `{"tool_input":{"url":"` + srv.URL + `","prompt":"how does X work"}}`
	out, errOut, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-pre", "--harness=claude")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (cache-hit channel)\nstderr: %s", code, errOut)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout should be empty on a cache hit, got %q", out)
	}
	for _, want := range []string{"Cache hit", "HTTP 304", "CACHED READING OF THE PAGE", "how does X work"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("stderr missing %q\n%s", want, errOut)
		}
	}
}

func TestHookSourceCachePreFallsThroughOnPromptMismatch(t *testing.T) {
	proj := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == "etag-1" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	seedSourceCache(t, proj, srv.URL, map[string]string{
		"url": srv.URL, "etag": "etag-1", "content": "OLD PROMPT-SPECIFIC READING",
		"fetched_at": "2026-01-01T00:00:00Z", "prompt": "how does X work",
	})
	payload := `{"tool_input":{"url":"` + srv.URL + `","prompt":"what changed in Y"}}`
	out, errOut, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-pre", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" || strings.TrimSpace(errOut) != "" {
		t.Errorf("prompt mismatch must fall through silently; got exit=%d out=%q err=%q", code, out, errOut)
	}
}

func TestHookSourceCachePreFallsThroughOnChange(t *testing.T) {
	proj := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "new") // never a 304 → the page changed
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	seedSourceCache(t, proj, srv.URL, map[string]string{"url": srv.URL, "etag": "stale", "content": "OLD"})
	payload := `{"tool_input":{"url":"` + srv.URL + `"}}`
	out, _, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-pre", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("a changed page must fall through (exit 0, silent); got exit=%d out=%q", code, out)
	}
}

func TestHookSourceCachePreNoValidatorNeverServes(t *testing.T) {
	proj := t.TempDir()
	// A seeded entry with no etag/last_modified can't be proven fresh: no network,
	// no serve. (No server needed: the hook returns before any request.)
	seedSourceCache(t, proj, "http://example.test/x", map[string]string{"url": "http://example.test/x", "content": "X"})
	payload := `{"tool_input":{"url":"http://example.test/x"}}`
	out, _, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-pre", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("no-validator entry must never serve; got exit=%d out=%q", code, out)
	}
}

func TestHookSourceCachePostStoresWithValidator(t *testing.T) {
	proj := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "etag-9")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	payload := `{"tool_input":{"url":"` + srv.URL + `","prompt":"P"},"tool_response":"THE FETCHED CONTENT"}`
	if _, errOut, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-post", "--harness=claude"); code != 0 {
		t.Fatalf("exit = %d (stderr %s)", code, errOut)
	}
	raw, err := os.ReadFile(sourceCacheEntryPath(proj, srv.URL))
	if err != nil {
		t.Fatalf("entry not written: %v", err)
	}
	var e struct {
		ETag, Content, Prompt string
	}
	e.ETag = gjson(t, raw, "etag")
	e.Content = gjson(t, raw, "content")
	e.Prompt = gjson(t, raw, "prompt")
	if e.ETag != "etag-9" || e.Content != "THE FETCHED CONTENT" || e.Prompt != "P" {
		t.Errorf("stored entry wrong: %s", raw)
	}
}

func TestHookSourceCachePostSkipsWithoutValidator(t *testing.T) {
	proj := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // no ETag / Last-Modified
	}))
	defer srv.Close()
	seedSourceCache(t, proj, srv.URL, map[string]string{"url": srv.URL, "etag": "was-here", "content": "STALE"})
	payload := `{"tool_input":{"url":"` + srv.URL + `"},"tool_response":"C"}`
	if _, _, code := runDevritesIO(t, proj, payload, projEnv(proj), "hook", "source-cache-post", "--harness=claude"); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if _, err := os.Stat(sourceCacheEntryPath(proj, srv.URL)); !os.IsNotExist(err) {
		t.Errorf("a response with no validator must not cache, and must clear the stale entry")
	}
}

func TestHookSourceCacheOffSwitch(t *testing.T) {
	proj := t.TempDir()
	seedSourceCache(t, proj, "http://x.test", map[string]string{"url": "http://x.test", "etag": "e", "content": "C"})
	payload := `{"tool_input":{"url":"http://x.test"}}`
	out, _, code := runDevritesIO(t, proj, payload, projEnv(proj, "DEVRITES_SOURCE_CACHE=off"), "hook", "source-cache-pre", "--harness=claude")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("off switch must no-op (exit 0, no network); got exit=%d out=%q", code, out)
	}
}

// --- refresh-indexes ---

func TestHookRefreshIndexesNoIndexIsSilent(t *testing.T) {
	proj := t.TempDir() // no .codegraph / graphify-out / codebase-memory
	out, _, code := runDevritesIO(t, proj, "", projEnv(proj), "hook", "refresh-indexes")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("no index tracking the repo → silent exit 0; got exit=%d out=%q", code, out)
	}
}

func TestHookRefreshIndexesOffSwitch(t *testing.T) {
	proj := t.TempDir()
	mkdirAllT(t, proj, ".codegraph") // would otherwise be available
	out, _, code := runDevritesIO(t, proj, "", projEnv(proj, "DEVRITES_REFRESH_INDEXES=off"), "hook", "refresh-indexes")
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("off switch must no-op; got exit=%d out=%q", code, out)
	}
}

func TestHookRefreshIndexesForceReports(t *testing.T) {
	proj := t.TempDir()
	mkdirAllT(t, proj, ".codegraph") // available → the force worker runs synchronously
	out, errOut, code := runDevritesIO(t, proj, "", projEnv(proj), "hook", "refresh-indexes", "--force", proj)
	if code != 0 {
		t.Fatalf("exit = %d (stderr %s)", code, errOut)
	}
	if !strings.Contains(out, "refresh start") || !strings.Contains(out, "refresh done") {
		t.Errorf("--force must print a synchronous report\n%s", out)
	}
}

// gjson reads a single string field from a small JSON object, failing the test on
// a malformed document.
func gjson(t *testing.T, raw []byte, key string) string {
	t.Helper()
	m := map[string]any{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}
