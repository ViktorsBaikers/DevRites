// Package release acquires verified DevRites release artifacts for self-update.
package release

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultRepository = "ViktorsBaikers/DevRites"
	maxAssetBytes     = int64(64) << 20
	maxJSONBytes      = int64(4) << 20
	maxChecksumBytes  = int64(4) << 10
	maxArchiveEntries = 4096
	maxEntryBytes     = int64(64) << 20
	maxExtractedBytes = int64(512) << 20
)

var (
	apiBaseURL = "https://api.github.com"
	webBaseURL = "https://github.com"
	httpClient = &http.Client{
		Timeout:       2 * time.Minute,
		CheckRedirect: checkRedirect,
	}
	repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	tagPattern        = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
)

// Candidate is one verified, extracted release and its platform engine.
type Candidate struct {
	SourceDir  string
	PayloadDir string
	EnginePath string
	BundleURL  string
}

// Latest returns the exact tag of the latest published release.
func Latest(ctx context.Context, repository string) (string, error) {
	if !repositoryPattern.MatchString(repository) {
		return "", fmt.Errorf("invalid DevRites repository %q", repository)
	}
	var metadata struct {
		TagName string `json:"tag_name"`
	}
	endpoint := strings.TrimRight(apiBaseURL, "/") + "/repos/" + repository + "/releases/latest"
	if err := fetchJSON(ctx, endpoint, &metadata); err != nil {
		return "", fmt.Errorf("resolve latest release: %w", err)
	}
	if !validTag(metadata.TagName) {
		return "", fmt.Errorf("resolve latest release: invalid tag %q", metadata.TagName)
	}
	return metadata.TagName, nil
}

// Acquire downloads, verifies, and extracts the bundle and platform engine.
func Acquire(ctx context.Context, repository, tag string) (Candidate, func(), error) {
	if !repositoryPattern.MatchString(repository) {
		return Candidate{}, func() {}, fmt.Errorf("invalid DevRites repository %q", repository)
	}
	if !validTag(tag) {
		return Candidate{}, func() {}, fmt.Errorf("invalid DevRites release tag %q", tag)
	}
	binaryName, err := platformBinaryName()
	if err != nil {
		return Candidate{}, func() {}, err
	}
	tmp, err := os.MkdirTemp("", "devrites-update-*")
	if err != nil {
		return Candidate{}, func() {}, fmt.Errorf("create update directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	fail := func(err error) (Candidate, func(), error) {
		cleanup()
		return Candidate{}, func() {}, err
	}

	base := strings.TrimRight(webBaseURL, "/") + "/" + repository + "/releases/download/" + tag
	bundleName := "devrites-" + tag + ".tar.gz"
	bundleURL := base + "/" + bundleName
	bundlePath := filepath.Join(tmp, bundleName)
	if err := downloadVerified(ctx, bundleURL, bundlePath, bundleName, 0o644); err != nil {
		return fail(fmt.Errorf("download release bundle: %w", err))
	}
	source, err := extractBundle(bundlePath, tmp, tag)
	if err != nil {
		return fail(err)
	}

	engineURL := base + "/" + binaryName
	enginePath := filepath.Join(tmp, binaryName)
	if err := downloadVerified(ctx, engineURL, enginePath, binaryName, 0o755); err != nil {
		return fail(fmt.Errorf("download engine binary: %w", err))
	}
	return Candidate{
		SourceDir:  source,
		PayloadDir: filepath.Join(source, "pack", "generated"),
		EnginePath: enginePath,
		BundleURL:  bundleURL,
	}, cleanup, nil
}

func validTag(tag string) bool {
	if !tagPattern.MatchString(tag) {
		return false
	}
	version := strings.SplitN(strings.TrimPrefix(tag, "v"), "+", 2)[0]
	_, prerelease, found := strings.Cut(version, "-")
	if !found {
		return true
	}
	for _, identifier := range strings.Split(prerelease, ".") {
		if len(identifier) > 1 && identifier[0] == '0' {
			numeric := true
			for _, char := range identifier {
				if char < '0' || char > '9' {
					numeric = false
					break
				}
			}
			if numeric {
				return false
			}
		}
	}
	return true
}

func platformBinaryName() (string, error) {
	supported := (runtime.GOOS == "darwin" || runtime.GOOS == "linux") && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		supported = true
	}
	if !supported {
		return "", fmt.Errorf("no DevRites engine release for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	name := "devrites-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name, nil
}

func fetchJSON(ctx context.Context, rawURL string, out any) error {
	resp, err := get(ctx, rawURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", resp.Request.URL.Redacted(), resp.Status)
	}
	if resp.ContentLength > maxJSONBytes {
		return fmt.Errorf("JSON response exceeds %d bytes", maxJSONBytes)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONBytes+1))
	if err != nil {
		return fmt.Errorf("read JSON response: %w", err)
	}
	if int64(len(body)) > maxJSONBytes {
		return fmt.Errorf("JSON response exceeds %d bytes", maxJSONBytes)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

func downloadVerified(ctx context.Context, rawURL, destination, filename string, mode fs.FileMode) error {
	if err := download(ctx, rawURL, destination, maxAssetBytes, mode); err != nil {
		return err
	}
	sumPath := destination + ".sha256"
	if err := download(ctx, rawURL+".sha256", sumPath, maxChecksumBytes, 0o600); err != nil {
		return fmt.Errorf("missing checksum for %s: %w", filename, err)
	}
	// #nosec G304 -- checksum downloaded into the self-update temp dir
	sum, err := os.ReadFile(sumPath)
	if err != nil {
		return fmt.Errorf("read checksum for %s: %w", filename, err)
	}
	fields := strings.Fields(string(sum))
	if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != filename {
		return fmt.Errorf("invalid checksum record for %s", filename)
	}
	want, err := hex.DecodeString(fields[0])
	if err != nil || len(want) != sha256.Size {
		return fmt.Errorf("invalid SHA-256 for %s", filename)
	}
	// #nosec G304 -- downloaded release asset in the self-update temp dir
	file, err := os.Open(destination)
	if err != nil {
		return fmt.Errorf("open %s for checksum: %w", filename, err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("hash %s: %w", filename, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", filename, closeErr)
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), hex.EncodeToString(want)) {
		return fmt.Errorf("checksum mismatch for %s", filename)
	}
	return nil
}

func download(ctx context.Context, rawURL, destination string, maxBytes int64, mode fs.FileMode) error {
	resp, err := get(ctx, rawURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s returned %s", resp.Request.URL.Redacted(), resp.Status)
	}
	if resp.ContentLength > maxBytes {
		return fmt.Errorf("download exceeds %d bytes", maxBytes)
	}
	out, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return fmt.Errorf("create download: %w", err)
	}
	tmp := out.Name()
	keep := false
	defer func() {
		_ = out.Close()
		if !keep {
			_ = os.Remove(tmp)
		}
	}()
	written, err := io.Copy(out, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return fmt.Errorf("write download: %w", err)
	}
	if written > maxBytes {
		return fmt.Errorf("download exceeds %d bytes", maxBytes)
	}
	if err := out.Chmod(mode); err != nil {
		return fmt.Errorf("set download mode: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync download: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close download: %w", err)
	}
	if err := os.Rename(tmp, destination); err != nil {
		return fmt.Errorf("install download: %w", err)
	}
	keep = true
	return nil
}

func get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build release request: %w", err)
	}
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "devrites-engine")
	return httpClient.Do(req)
}

func validateURL(u *url.URL) error {
	if u == nil || u.Hostname() == "" {
		return errors.New("release URL has no host")
	}
	if strings.EqualFold(u.Scheme, "https") {
		return nil
	}
	if strings.EqualFold(u.Scheme, "http") {
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && ip.Unmap().IsLoopback() {
			return nil
		}
	}
	return fmt.Errorf("release URL must use HTTPS: %s", u.Redacted())
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if err := validateURL(req.URL); err != nil {
		return err
	}
	if len(via) > 0 && !isLoopback(via[0].URL) && isLoopback(req.URL) {
		return fmt.Errorf("remote release URL cannot redirect to loopback: %s", req.URL.Redacted())
	}
	return nil
}

func isLoopback(u *url.URL) bool {
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

func extractBundle(archive, destination, tag string) (string, error) {
	if err := extractTarGz(archive, destination); err != nil {
		return "", fmt.Errorf("extract release bundle: %w", err)
	}
	source := filepath.Join(destination, "devrites-"+tag)
	if _, err := os.Stat(filepath.Join(source, "package.json")); err != nil {
		return "", fmt.Errorf("release bundle is missing package.json: %w", err)
	}
	if info, err := os.Stat(filepath.Join(source, "pack", "generated")); err != nil || !info.IsDir() {
		return "", fmt.Errorf("release bundle is missing pack/generated")
	}
	return source, nil
}

func extractTarGz(archive, destination string) error {
	// #nosec G304 -- downloaded release archive in the self-update temp dir
	file, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer func() { _ = file.Close() }()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer func() { _ = gz.Close() }()
	reader := tar.NewReader(gz)
	root, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve extraction directory: %w", err)
	}
	entries := 0
	var total int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		entries++
		if entries > maxArchiveEntries {
			return fmt.Errorf("release bundle has more than %d entries", maxArchiveEntries)
		}
		if header.Size < 0 || header.Size > maxEntryBytes || header.Size > maxExtractedBytes-total {
			return fmt.Errorf("release bundle entry %q exceeds extraction limits", header.Name)
		}
		total += header.Size
		name := path.Clean(header.Name)
		if name != strings.TrimSuffix(header.Name, "/") || !fs.ValidPath(name) || name == "." {
			return fmt.Errorf("unsafe path in release bundle: %s", header.Name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe path in release bundle: %s", header.Name)
		}
		mode := fs.FileMode(header.Mode & 0o777)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return fmt.Errorf("create directory %s: %w", name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create directory for %s: %w", name, err)
			}
			// #nosec G304 -- tar member name ValidPath-checked and confined to the extraction root above
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			written, copyErr := io.Copy(out, io.LimitReader(reader, maxEntryBytes+1))
			closeErr := out.Close()
			if copyErr != nil {
				return fmt.Errorf("extract %s: %w", name, copyErr)
			}
			if written != header.Size || written > maxEntryBytes {
				return fmt.Errorf("release bundle entry %q has inconsistent size", header.Name)
			}
			if closeErr != nil {
				return fmt.Errorf("close %s: %w", name, closeErr)
			}
		default:
			return fmt.Errorf("release bundle entry %q has unsupported type", header.Name)
		}
	}
}
