// Package iohooks contains hooks that access the network or start processes. The
// source cache revalidates citations with conditional HEAD requests, and the
// index refresher starts background reindexing. Errors do not block the caller.
package iohooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/harness"
)

const (
	// Current release tarballs are under 2 MiB and host binaries are about 13 MiB.
	// The 64 MiB cap allows growth while keeping temporary files bounded.
	maxInstallDownloadBytes = int64(64) << 20
	// Release JSON is metadata rather than an asset, so it gets a smaller cap.
	maxInstallJSONBytes = int64(4) << 20
	// WebFetch results are untrusted JSON. One MiB covers the current reading cache
	// while bounding the optional warning scan.
	maxSourcePostPayloadBytes = int64(1) << 20

	// exitOK is the fail-open success code these hooks return.
	exitOK = 0

	ingestWarningMode = "warn"

	ingestReasonHiddenControl       = "ingest_hidden_control"
	ingestReasonInstructionRedirect = "ingest_instruction_redirect"
	ingestReasonCredentialShape     = "ingest_credential_shape" // #nosec G101 -- finding category, not a credential
)

var (
	installHTTPClient = &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: checkInstallRedirect,
	}
	sourceCacheHTTPClient = &http.Client{Timeout: 5 * time.Second}

	ingestInstructionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:ignore|disregard|forget|override)\s+(?:all\s+)?(?:previous|prior|earlier|above)\s+(?:instructions?|directives?|rules?)\b`),
		regexp.MustCompile(`(?i)\b(?:follow|obey|execute)\s+(?:only\s+)?(?:these|the following|my)\s+(?:instructions?|directives?|commands?)\s+(?:instead|now)\b`),
	}
	ingestCredentialPatterns = []struct {
		re     *regexp.Regexp
		accept func(string) bool
	}{
		{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), func(s string) bool {
			return !strings.HasSuffix(s, "EXAMPLE")
		}},
		{regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,255}`), variedCredential},
		{regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`), variedCredential},
		{regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----[\r\n]+[A-Za-z0-9+/=\r\n]{64,}-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`), func(string) bool {
			return true
		}},
	}
)

// ---- source-citation cache -------------------------------------------------

// sourceCacheEntry stores a processed page and the validators needed to confirm
// it with a later 304 response.
type sourceCacheEntry struct {
	URL          string `json:"url"`
	Prompt       string `json:"prompt"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	Content      string `json:"content"`
	FetchedAt    string `json:"fetched_at"`
}

func sourceCacheOff() bool { return os.Getenv("DEVRITES_SOURCE_CACHE") == "off" }

// ioProjectDir returns the repository root from CLAUDE_PROJECT_DIR or the current
// directory, matching the shell hooks. The cache is project wide under
// .devrites, so it does not use the feature directory resolver.
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
// keyed on the first 32 hex chars of sha256(url): matching the shell hooks.
func sourceCachePath(url string) string {
	sum := sha256.Sum256([]byte(url))
	key := hex.EncodeToString(sum[:])[:32]
	return filepath.Join(ioProjectDir(), ".devrites", "source-cache", key+".json")
}

// hookSourceCachePre serves a cached page only after a conditional HEAD request
// returns 304. A cache hit writes the reading to stderr and exits 2 so Claude
// uses it instead of fetching. A miss or uncertain result exits 0 and lets
// WebFetch continue. This matches devrites-source-cache-pre.sh.
func SourceCachePre(stdin io.Reader, stdout, stderr io.Writer) int {
	if sourceCacheOff() {
		return exitOK
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	payload := parseWebFetchPayload(data)
	url := payload.ToolInput.URL
	if url == "" {
		return exitOK
	}
	prompt := payload.ToolInput.Prompt
	raw, err := os.ReadFile(sourceCachePath(url))
	if err != nil {
		return exitOK
	}
	var e sourceCacheEntry
	if json.Unmarshal(raw, &e) != nil {
		return exitOK
	}
	// An entry without a validator cannot be revalidated.
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
		return exitOK // A slow or unavailable server does not stall the turn.
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified || e.Content == "" {
		return exitOK // Let WebFetch handle changed or inconclusive results.
	}

	fetched := e.FetchedAt
	if fetched == "" {
		fetched = "a previous fetch"
	}
	fmt.Fprintf(stderr, "[devrites-source-cache] Cache hit for %s\n", url)
	fmt.Fprintf(stderr, "Revalidated via HTTP 304: the page is unchanged since %s, so this cached reading is still a valid citation.\n", fetched)
	if e.Prompt != "" {
		fmt.Fprintf(stderr, "Cached under the prompt: %q. If your angle differs, judge whether this reading still covers it before re-fetching.\n", e.Prompt)
	}
	fmt.Fprintln(stderr, "----- BEGIN CACHED CONTENT -----")
	fmt.Fprintf(stderr, "%s\n", e.Content)
	fmt.Fprintln(stderr, "----- END CACHED CONTENT -----")
	return 2 // Claude uses stderr as the fetch result on a cache hit.
}

// SourceCachePost stores a completed WebFetch under sha256(url) after a bounded
// read. For Claude, the optional warning scan runs before the cache write and
// returns typed metadata. The hook always exits 0.
func SourceCachePost(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	warn := h == harness.Claude && os.Getenv("DEVRITES_INGEST_WARNING") == ingestWarningMode
	cacheOff := sourceCacheOff()
	if cacheOff && !warn {
		return exitOK
	}
	data, err := io.ReadAll(io.LimitReader(stdin, maxSourcePostPayloadBytes+1))
	if err != nil || int64(len(data)) > maxSourcePostPayloadBytes {
		return exitOK
	}
	payload := parseWebFetchPayload(data)
	url := payload.ToolInput.URL
	if url == "" {
		return exitOK
	}
	content := payload.content()
	if content == "" {
		return exitOK
	}
	if warn {
		findings := scanIngestContent(content, ingestOrigin(payload.ToolInput.URL))
		if len(findings) > 0 {
			_ = os.Remove(sourceCachePath(url))
			writeIngestWarning(h, findings, stdout)
			return exitOK
		}
	}
	if cacheOff {
		return exitOK
	}
	prompt := payload.ToolInput.Prompt

	etag, lastMod := fetchValidators(url)
	entryPath := sourceCachePath(url)
	// Clear stale data when no validator can support later revalidation.
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

type ingestFinding struct {
	ReasonID     string `json:"reason_id"`
	Class        string `json:"class"`
	Severity     string `json:"severity"`
	Offset       int    `json:"offset"`
	Origin       string `json:"origin"`
	CacheSkipped bool   `json:"cache_skipped"`
}

func scanIngestContent(content, origin string) []ingestFinding {
	findings := make([]ingestFinding, 0, 3)
	appendFinding := func(reasonID, class string, offset int) {
		findings = append(findings, ingestFinding{
			ReasonID:     reasonID,
			Class:        class,
			Severity:     "high",
			Offset:       offset,
			Origin:       origin,
			CacheSkipped: true,
		})
	}
	if offset, ok := hiddenControlOffset(content); ok {
		appendFinding(ingestReasonHiddenControl, "hidden_control", offset)
	}
	if offset, ok := instructionRedirectOffset(content); ok {
		appendFinding(ingestReasonInstructionRedirect, "instruction_redirect", offset)
	}
	if offset, ok := credentialOffset(content); ok {
		appendFinding(ingestReasonCredentialShape, "credential_shape", offset)
	}
	return findings
}

func hiddenControlOffset(content string) (int, bool) {
	for offset, r := range content {
		switch {
		case r >= '\u202a' && r <= '\u202e':
			return offset, true
		case r >= '\u2066' && r <= '\u2069':
			return offset, true
		case r == '\u200b' || r == '\u2060' || r == '\ufeff':
			return offset, true
		}
	}
	return 0, false
}

func instructionRedirectOffset(content string) (int, bool) {
	best := -1
	for _, pattern := range ingestInstructionPatterns {
		for cursor := 0; cursor < len(content); {
			loc := pattern.FindStringIndex(content[cursor:])
			if loc == nil {
				break
			}
			start, end := cursor+loc[0], cursor+loc[1]
			if !instructionExample(content, start, end) && (best < 0 || start < best) {
				best = start
			}
			cursor = end
		}
	}
	return best, best >= 0
}

func instructionExample(content string, start, end int) bool {
	lineStart := strings.LastIndexByte(content[:start], '\n') + 1
	beforeStart := lineStart
	if start-beforeStart > 160 {
		beforeStart = start - 160
	}
	afterEnd := len(content)
	if end+160 < afterEnd {
		afterEnd = end + 160
	}
	if n := strings.IndexByte(content[end:afterEnd], '\n'); n >= 0 {
		afterEnd = end + n
	}
	before, after := content[beforeStart:start], content[end:afterEnd]
	for _, quote := range []byte{'`', '"', '\''} {
		if strings.LastIndexByte(before, quote) >= 0 && strings.IndexByte(after, quote) >= 0 {
			return true
		}
	}
	lowerBefore := strings.ToLower(before)
	trimmedBefore := strings.TrimSpace(lowerBefore)
	if strings.HasPrefix(trimmedBefore, "//") || strings.HasPrefix(trimmedBefore, "# ") {
		return true
	}
	for _, marker := range []string{
		"do not ", "don't ", "never ", "example", "pattern", "detect",
		"security guidance", "prompt injection",
	} {
		if strings.Contains(lowerBefore, marker) {
			return true
		}
	}
	return false
}

func credentialOffset(content string) (int, bool) {
	best := -1
	for _, pattern := range ingestCredentialPatterns {
		for cursor := 0; cursor < len(content); {
			loc := pattern.re.FindStringIndex(content[cursor:])
			if loc == nil {
				break
			}
			start, end := cursor+loc[0], cursor+loc[1]
			if pattern.accept(content[start:end]) && (best < 0 || start < best) {
				best = start
				break
			}
			cursor = end
		}
	}
	return best, best >= 0
}

func variedCredential(s string) bool {
	var lower, upper, digit bool
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		}
	}
	return lower && upper && digit
}

func ingestOrigin(rawURL string) string {
	origin := "claude/WebFetch/PostToolUse"
	if u, err := url.Parse(rawURL); err == nil && u.Hostname() != "" {
		host := strings.ToLower(u.Hostname())
		if len(host) <= 253 && strings.IndexFunc(host, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == ':')
		}) < 0 {
			origin += "@" + host
		}
	}
	return origin
}

func writeIngestWarning(h harness.Harness, findings []ingestFinding, stdout io.Writer) {
	metadata, err := json.Marshal(findings)
	if err != nil {
		return
	}
	context := "DevRites experimental ingestion warning: suspicious untrusted WebFetch content detected. Review this metadata before using the result: " + string(metadata)
	envelope, err := h.PostToolContext(context)
	if err == nil {
		fmt.Fprintln(stdout, envelope)
	}
}

// fetchValidators sends a HEAD request to the origin, follows redirects, and
// returns the final response's ETag and Last-Modified values. Failures return
// empty values.
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
	resp, err := installGet(url)
	if err != nil {
		return fmt.Errorf("fetch JSON: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", resp.Request.URL.Redacted(), resp.Status)
	}
	if resp.ContentLength > maxInstallJSONBytes {
		return fmt.Errorf("JSON response exceeds %d bytes", maxInstallJSONBytes)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxInstallJSONBytes+1))
	if err != nil {
		return fmt.Errorf("read JSON response: %w", err)
	}
	if int64(len(body)) > maxInstallJSONBytes {
		return fmt.Errorf("JSON response exceeds %d bytes", maxInstallJSONBytes)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

func DownloadFile(url, path string) error {
	resp, err := installGet(url)
	if err != nil {
		return fmt.Errorf("download to %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_ = resp.Body.Close()
		return fmt.Errorf("%s returned %s", resp.Request.URL.Redacted(), resp.Status)
	}
	if resp.ContentLength > maxInstallDownloadBytes {
		_ = resp.Body.Close()
		return fmt.Errorf("download exceeds %d bytes", maxInstallDownloadBytes)
	}
	dir := filepath.Dir(path)
	out, err := os.CreateTemp(dir, "."+filepath.Base(path)+".download-*")
	if err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("create output file: %w", err)
	}
	tmp := out.Name()
	keep := false
	defer func() {
		_ = out.Close()
		if !keep {
			_ = os.Remove(tmp)
		}
	}()
	written, err := io.Copy(out, io.LimitReader(resp.Body, maxInstallDownloadBytes+1))
	if err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if written > maxInstallDownloadBytes {
		_ = resp.Body.Close()
		return fmt.Errorf("download exceeds %d bytes", maxInstallDownloadBytes)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("close download %s: %w", resp.Request.URL.Redacted(), err)
	}
	if err := out.Chmod(0o644); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("install %s: %w", path, err)
	}
	keep = true
	return nil
}

func installGet(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build install request: %w", err)
	}
	if err := validateInstallURL(req.URL); err != nil {
		return nil, err
	}
	return installHTTPClient.Do(req)
}

func validateInstallURL(u *url.URL) error {
	if u == nil || u.Hostname() == "" {
		return errors.New("install URL has no host")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && ip.Unmap().IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("install URL must use HTTPS (HTTP is allowed only for literal loopback test addresses): %s", u.Redacted())
}

func checkInstallRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if err := validateInstallURL(req.URL); err != nil {
		return err
	}
	if len(via) == 0 {
		return nil
	}
	if !isLoopbackURL(via[0].URL) && isLoopbackURL(req.URL) {
		return fmt.Errorf("remote install URL cannot redirect to loopback: %s", req.URL.Redacted())
	}
	if !strings.EqualFold(via[len(via)-1].URL.Host, req.URL.Host) {
		req.URL.User = nil
		for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
			req.Header.Del(name)
		}
	}
	return nil
}

func isLoopbackURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Unmap().IsLoopback()
}

type webFetchPayload struct {
	ToolInput struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt"`
	} `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
}

func parseWebFetchPayload(data []byte) webFetchPayload {
	var p webFetchPayload
	_ = json.Unmarshal(data, &p)
	return p
}

// content tries the known WebFetch response fields, then a bare string response.
func (p webFetchPayload) content() string {
	if len(p.ToolResponse) == 0 {
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

// hookRefreshIndexes refreshes optional code indexes after source changes. With
// no arguments it checks availability and a change stamp, then starts a detached
// worker so Stop can return. `--worker ROOT` runs that work, while
// `--force [ROOT]` runs it synchronously and prints a report. This matches
// devrites-refresh-indexes.sh and stays silent when no index tracks the repo.
func RefreshIndexes(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if os.Getenv("DEVRITES_REFRESH_INDEXES") == "off" {
		return exitOK
	}
	// Run the detached worker, then release its lock.
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

	// Skip repositories that no configured index tracks.
	if !dirExists(filepath.Join(root, ".codegraph")) &&
		!dirExists(filepath.Join(root, "graphify-out")) &&
		!cbmRegistered(root) {
		return exitOK
	}
	// In trigger mode, skip unchanged repositories.
	if !force && fileExists(st.stamp) && !repoChangedSince(root, st.stamp) {
		return exitOK
	}
	// Allow one refresh at a time and replace locks older than 30 minutes.
	if !acquireRefreshLock(st.lock) {
		return exitOK
	}
	// Stamp the attempt so a no-op turn does not trigger it again.
	_ = os.WriteFile(st.stamp, []byte(""), 0o644)

	if force {
		refreshWorker(root, st, stdout)
		_ = os.RemoveAll(st.lock)
		return exitOK
	}
	// Start a detached worker so Stop can return immediately.
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
// stamp, skipping the excluded directories: the Go form of the script's `find
// -newer`. It returns on the first newer file found.
func repoChangedSince(root, stamp string) bool {
	info, err := os.Stat(stamp)
	if err != nil {
		return true // no stamp → treat as changed
	}
	stampTime := info.ModTime()
	changed := false
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
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
	}); err != nil {
		return true
	}
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

// refreshWorker runs each installed indexer that tracks the repository, with a
// timeout for each command.
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

// runIndexTool runs one index command with a timeout and copies its combined
// output. Failures are reported but do not stop other indexers.
func runIndexTool(out io.Writer, timeout time.Duration, name string, args ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintf(out, "[%s] timed out\n", name)
		return
	}
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintf(out, "[%s] not runnable\n", name)
			return
		}
		if s := buf.String(); s != "" {
			fmt.Fprint(out, s)
			if !strings.HasSuffix(s, "\n") {
				fmt.Fprintln(out)
			}
		}
		fmt.Fprintf(out, "[%s] failed\n", name)
		return
	}
	if s := buf.String(); s != "" {
		fmt.Fprint(out, s)
		if !strings.HasSuffix(s, "\n") {
			fmt.Fprintln(out)
		}
	}
}

// cbmRegistered reports whether codebase-memory-mcp is installed and tracks
// root. It checks a repository marker before consulting the project registry.
func cbmRegistered(root string) bool {
	if !hasTool("codebase-memory-mcp") {
		return false
	}
	if dirExists(filepath.Join(root, ".codebase-memory")) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codebase-memory-mcp", "cli", "--raw", "list_projects")
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
