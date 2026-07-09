package iohooks

// IO hooks: the ported bash hooks that do network or process-orchestration IO —
// the source-citation cache (a conditional-HEAD revalidation cache) and the
// code-index refresher (a detached background reindex). Unlike the deterministic
// control-plane hooks these reach the network / spawn processes; they are kept in
// their own file to mark that boundary. Every one is fail-open and never blocks.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
)

// exitOK is the fail-open success code these hooks return.
const exitOK = 0

var (
	installHTTPClient     = &http.Client{Timeout: 30 * time.Second}
	sourceCacheHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// ---- source-citation cache -------------------------------------------------

// sourceCacheEntry is the on-disk cache record: a page's post-processed reading
// plus the origin validators that let a later 304 prove it is still fresh.
type sourceCacheEntry struct {
	URL          string `json:"url"`
	Prompt       string `json:"prompt"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	Content      string `json:"content"`
	FetchedAt    string `json:"fetched_at"`
}

func sourceCacheOff() bool { return os.Getenv("DEVRITES_SOURCE_CACHE") == "off" }

// ioProjectDir is the repo root the IO hooks key off — CLAUDE_PROJECT_DIR or CWD,
// like the shell hooks. (The cache is project-global under .devrites/, not
// feature-scoped, so it does not use the feature-dir resolver.)
func ioProjectDir() string {
	if d := os.Getenv("CLAUDE_PROJECT_DIR"); d != "" {
		return d
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// sourceCachePath is the entry path for a URL: .devrites/source-cache/<key>.json,
// keyed on the first 32 hex chars of sha256(url) — matching the shell hooks.
func sourceCachePath(url string) string {
	sum := sha256.Sum256([]byte(url))
	key := hex.EncodeToString(sum[:])[:32]
	return filepath.Join(ioProjectDir(), ".devrites", "source-cache", key+".json")
}

// hookSourceCachePre serves a cached page reading ONLY after revalidating it with a
// conditional HEAD that returns 304 — a fresh verification, not a memory read.
// Ported from devrites-source-cache-pre.sh. Cache hit → the reading on stderr +
// exit 2 (Claude hands that to the agent in place of the fetch); miss/uncertain →
// exit 0 (the real WebFetch proceeds). Fail-open on every uncertainty.
func SourceCachePre(stdin io.Reader, stdout, stderr io.Writer) int {
	if sourceCacheOff() {
		return exitOK
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	url := webFetchURL(data)
	if url == "" {
		return exitOK
	}
	prompt := webFetchPrompt(data)
	raw, err := os.ReadFile(sourceCachePath(url))
	if err != nil {
		return exitOK
	}
	var e sourceCacheEntry
	if json.Unmarshal(raw, &e) != nil {
		return exitOK
	}
	// No validator on the stored entry → freshness is unprovable → never serve.
	if e.ETag == "" && e.LastModified == "" {
		return exitOK
	}
	if e.Prompt != prompt {
		return exitOK
	}

	headers := http.Header{}
	if e.ETag != "" {
		headers.Set("If-None-Match", e.ETag)
	}
	if e.LastModified != "" {
		headers.Set("If-Modified-Since", e.LastModified)
	}
	resp, err := headURL(url, headers)
	if err != nil {
		return exitOK // slow/dead server must not stall the turn
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified || e.Content == "" {
		return exitOK // changed / inconclusive → let WebFetch re-fetch
	}

	fetched := e.FetchedAt
	if fetched == "" {
		fetched = "a previous fetch"
	}
	fmt.Fprintf(stderr, "[devrites-source-cache] Cache hit for %s\n", url)
	fmt.Fprintf(stderr, "Revalidated via HTTP 304 — the page is unchanged since %s, so this cached reading is still a valid citation.\n", fetched)
	if e.Prompt != "" {
		fmt.Fprintf(stderr, "Cached under the prompt: %q. If your angle differs, judge whether this reading still covers it before re-fetching.\n", e.Prompt)
	}
	fmt.Fprintln(stderr, "----- BEGIN CACHED CONTENT -----")
	fmt.Fprintf(stderr, "%s\n", e.Content)
	fmt.Fprintln(stderr, "----- END CACHED CONTENT -----")
	return 2 // the cache-hit channel: stderr replaces the fetch
}

// hookSourceCachePost stores a completed WebFetch keyed on sha256(url) together with
// the origin's validator headers — but only when the origin supplies a validator
// (else a later 304 could never prove freshness). Ported from
// devrites-source-cache-post.sh. Always exit 0.
func SourceCachePost(stdin io.Reader, stdout, stderr io.Writer) int {
	if sourceCacheOff() {
		return exitOK
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	url := webFetchURL(data)
	if url == "" {
		return exitOK
	}
	content := webFetchContent(data)
	if content == "" {
		return exitOK
	}
	prompt := webFetchPrompt(data)

	etag, lastMod := fetchValidators(url)
	entryPath := sourceCachePath(url)
	// No validator → we could never prove freshness later → don't cache; clear stale.
	if etag == "" && lastMod == "" {
		_ = os.Remove(entryPath)
		return exitOK
	}
	e := sourceCacheEntry{
		URL:          url,
		Prompt:       prompt,
		ETag:         etag,
		LastModified: lastMod,
		Content:      content,
		FetchedAt:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	body, err := json.Marshal(e)
	if err != nil {
		return exitOK
	}
	_ = fsutil.WriteFileAtomic(entryPath, body, 0o644)
	return exitOK
}

// fetchValidators does a HEAD to the origin (following redirects) and returns the
// FINAL response's ETag / Last-Modified, empty on any failure.
func fetchValidators(url string) (etag, lastMod string) {
	resp, err := headURL(url, nil)
	if err != nil {
		return "", ""
	}
	_ = resp.Body.Close()
	return resp.Header.Get("ETag"), resp.Header.Get("Last-Modified")
}

func headURL(url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build HEAD request: %w", err)
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	return sourceCacheHTTPClient.Do(req)
}

func FetchJSON(url string, out any) error {
	resp, err := installHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("fetch JSON: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func DownloadFile(url, path string) error {
	resp, err := installHTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("download to %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// webFetchURL / webFetchPrompt / webFetchContent decode the WebFetch payload fields
// the cache hooks read, best-effort (empty on any failure).
func webFetchURL(data []byte) string {
	var p struct {
		ToolInput struct {
			URL string `json:"url"`
		} `json:"tool_input"`
	}
	_ = json.Unmarshal(data, &p)
	return p.ToolInput.URL
}

func webFetchPrompt(data []byte) string {
	var p struct {
		ToolInput struct {
			Prompt string `json:"prompt"`
		} `json:"tool_input"`
	}
	_ = json.Unmarshal(data, &p)
	return p.ToolInput.Prompt
}

// webFetchContent tries the known WebFetch response fields, then a bare string
// response — mirroring the jq fallback chain in the shell hook.
func webFetchContent(data []byte) string {
	var p struct {
		ToolResponse json.RawMessage `json:"tool_response"`
	}
	if json.Unmarshal(data, &p) != nil || len(p.ToolResponse) == 0 {
		return ""
	}
	if p.ToolResponse[0] == '"' {
		var s string
		_ = json.Unmarshal(p.ToolResponse, &s)
		return s
	}
	var obj struct {
		Result  string `json:"result"`
		Output  string `json:"output"`
		Text    string `json:"text"`
		Content string `json:"content"`
		Body    string `json:"body"`
	}
	if json.Unmarshal(p.ToolResponse, &obj) != nil {
		return ""
	}
	for _, s := range []string{obj.Result, obj.Output, obj.Text, obj.Content, obj.Body} {
		if s != "" {
			return s
		}
	}
	return ""
}

// ---- code-index refresher --------------------------------------------------

// refreshExcludedDirs are the directory names the change-scan skips (their own
// index outputs, VCS, deps, build artifacts), matching refresh-indexes.sh's find.
var refreshExcludedDirs = map[string]bool{
	".git": true, ".codegraph": true, "graphify-out": true, ".codebase-memory": true,
	".devrites": true, "node_modules": true, ".venv": true, "__pycache__": true,
	".research": true, ".cursor": true, "dist": true, "build": true,
}

// hookRefreshIndexes keeps the optional code-intelligence indexes fresh after code
// changes. Ported from devrites-refresh-indexes.sh. Trigger mode (no args) guards on
// availability + a change-stamp, then spawns a DETACHED worker so Stop returns at
// once; `--worker ROOT` is that detached work; `--force [ROOT]` runs it
// synchronously and prints a report. Silent when no index tracks the repo;
// fail-open, never blocks.
func RefreshIndexes(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if os.Getenv("DEVRITES_REFRESH_INDEXES") == "off" {
		return exitOK
	}
	// Worker branch: do the work, release the lock, exit.
	if len(args) >= 1 && args[0] == "--worker" {
		root := argOrProject(args, 1)
		st := computeRefreshState(root)
		refreshWorker(root, st, stdout)
		_ = os.RemoveAll(st.lock)
		return exitOK
	}
	force := false
	rest := args
	if len(rest) >= 1 && rest[0] == "--force" {
		force = true
		rest = rest[1:]
	}
	root := argOrProject(rest, 0)
	st := computeRefreshState(root)

	// Availability guard: nothing to do unless at least one index tracks this repo.
	if !dirExists(filepath.Join(root, ".codegraph")) &&
		!dirExists(filepath.Join(root, "graphify-out")) &&
		!cbmRegistered(root) {
		return exitOK
	}
	// Change guard (trigger mode): skip if nothing changed since the last refresh.
	if !force && fileExists(st.stamp) && !repoChangedSince(root, st.stamp) {
		return exitOK
	}
	// Lock: never run two refreshes at once; steal a lock older than 30 minutes.
	if !acquireRefreshLock(st.lock) {
		return exitOK
	}
	// Debounce: stamp now so a no-op turn doesn't re-trigger.
	_ = os.WriteFile(st.stamp, []byte(""), 0o644)

	if force {
		refreshWorker(root, st, stdout)
		_ = os.RemoveAll(st.lock)
		return exitOK
	}
	// Detached worker: Stop returns immediately.
	spawnRefreshWorker(root, st)
	return exitOK
}

type refreshState struct {
	state, stamp, lock, log string
}

// computeRefreshState derives the gitignore-safe state locations from a repo root,
// preferring .codegraph, then .git, then the root itself.
func computeRefreshState(root string) refreshState {
	state := root
	if dirExists(filepath.Join(root, ".codegraph")) {
		state = filepath.Join(root, ".codegraph")
	} else if dirExists(filepath.Join(root, ".git")) {
		state = filepath.Join(root, ".git")
	}
	return refreshState{
		state: state,
		stamp: filepath.Join(state, ".index_refresh_stamp"),
		lock:  filepath.Join(state, ".index_refresh.lock"),
		log:   filepath.Join(state, "index_refresh.log"),
	}
}

// repoChangedSince reports whether any tracked file under root is newer than the
// stamp, skipping the excluded directories — the Go form of the script's `find
// -newer`. It returns on the first newer file found.
func repoChangedSince(root, stamp string) bool {
	info, err := os.Stat(stamp)
	if err != nil {
		return true // no stamp → treat as changed
	}
	stampTime := info.ModTime()
	changed := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && refreshExcludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if fi, e := d.Info(); e == nil && fi.ModTime().After(stampTime) {
			changed = true
			return filepath.SkipAll
		}
		return nil
	})
	return changed
}

// acquireRefreshLock creates the lock directory, stealing one older than 30 minutes.
func acquireRefreshLock(lock string) bool {
	if os.Mkdir(lock, 0o755) == nil {
		return true
	}
	if fi, err := os.Stat(lock); err == nil && time.Since(fi.ModTime()) > 30*time.Minute {
		_ = os.RemoveAll(lock)
		return os.Mkdir(lock, 0o755) == nil
	}
	return false // another refresh in progress
}

// spawnRefreshWorker re-invokes this binary as a detached `hook refresh-indexes
// --worker ROOT`, its output appended to the refresh log, so the Stop hook returns
// immediately.
func spawnRefreshWorker(root string, st refreshState) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "hook", "refresh-indexes", "--worker", root)
	if logf, err := os.OpenFile(st.log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	cmd.SysProcAttr = detachSysProcAttr()
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// refreshWorker runs the actual reindex for each optional tool that is installed
// AND tracks this repo, each under a bounded timeout. Missing tool = no-op.
func refreshWorker(root string, st refreshState, out io.Writer) {
	fmt.Fprintf(out, "[%s] refresh start: %s\n", nowStamp(), root)
	if cbmRegistered(root) {
		fmt.Fprintln(out, "[codebase-memory] index_repository")
		runIndexTool(out, 300*time.Second, "codebase-memory-mcp", "cli", "index_repository",
			fmt.Sprintf(`{"repo_path": %q}`, root))
	}
	if dirExists(filepath.Join(root, ".codegraph")) && hasTool("codegraph") {
		fmt.Fprintln(out, "[codegraph] sync")
		runIndexTool(out, 120*time.Second, "codegraph", "sync", root)
	}
	if dirExists(filepath.Join(root, "graphify-out")) && hasTool("graphify") {
		fmt.Fprintln(out, "[graphify] update")
		runIndexTool(out, 300*time.Second, "graphify", "update", root)
	}
	fmt.Fprintf(out, "[%s] refresh done\n", nowStamp())
}

// runIndexTool runs one external index command under a timeout, tee-ing its combined
// output; a failure or timeout is reported, never fatal.
func runIndexTool(out io.Writer, timeout time.Duration, name string, args ...string) {
	cmd := exec.Command(name, args...)
	done := make(chan error, 1)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(out, "[%s] not runnable\n", name)
		return
	}
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		fmt.Fprintf(out, "[%s] timed out\n", name)
	case err := <-done:
		if s := buf.String(); s != "" {
			fmt.Fprint(out, s)
			if !strings.HasSuffix(s, "\n") {
				fmt.Fprintln(out)
			}
		}
		if err != nil {
			fmt.Fprintf(out, "[%s] failed\n", name)
		}
	}
}

// cbmRegistered reports whether codebase-memory-mcp is installed AND tracks root —
// a cheap repo-local marker first, then the project registry.
func cbmRegistered(root string) bool {
	if !hasTool("codebase-memory-mcp") {
		return false
	}
	if dirExists(filepath.Join(root, ".codebase-memory")) {
		return true
	}
	cmd := exec.Command("codebase-memory-mcp", "cli", "--raw", "list_projects")
	outBytes, err := cmd.Output()
	return err == nil && strings.Contains(string(outBytes), root)
}

func argOrProject(args []string, i int) string {
	if i < len(args) && args[i] != "" {
		return args[i]
	}
	return ioProjectDir()
}

func hasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func nowStamp() string { return time.Now().Format("2006-01-02 15:04:05") }
