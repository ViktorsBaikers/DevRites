package profile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

const schemaVersion = "1"

type FileInfo struct {
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

type Profile struct {
	ProfileSchemaVersion string              `json:"profile_schema_version"`
	RootSHA              string              `json:"root_sha"`
	HeadSHA              string              `json:"head_sha"`
	TopLevel             []string            `json:"top_level"`
	Manifests            []string            `json:"manifests"`
	Inputs               map[string]FileInfo `json:"inputs"`
}

type cacheEnvelope struct {
	ProfileSchemaVersion string  `json:"profile_schema_version"`
	RootSHA              string  `json:"root_sha"`
	HeadSHA              string  `json:"head_sha"`
	Profile              Profile `json:"profile"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "get" && args[0] != "refresh") {
		fmt.Fprintln(stderr, "usage: devrites-engine profile get|refresh")
		return 2
	}
	root, ok := gitRoot()
	if !ok {
		fmt.Fprintln(stdout, "NO-CACHE")
		return 0
	}
	rootSHA, headSHA, ok := keys(root)
	if !ok {
		fmt.Fprintln(stdout, "NO-CACHE")
		return 0
	}
	path := cachePath(rootSHA, headSHA)
	switch args[0] {
	case "get":
		if cleanProfileInputs(root) {
			if p, ok := readCache(path, rootSHA, headSHA); ok {
				fmt.Fprintln(stdout, "HIT")
				writeJSON(stdout, p)
				return 0
			}
		}
		fmt.Fprintln(stdout, "MISS")
		fmt.Fprintln(stdout, path)
		return 0
	case "refresh":
		p := derive(root, rootSHA, headSHA)
		if err := writeCache(path, p); err != nil {
			fmt.Fprintf(stderr, "devrites: profile cache write failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "WROTE\n%s\n", path)
		writeJSON(stdout, p)
		return 0
	}
	return 2
}

func gitRoot() (string, bool) {
	out, ok := git("", "rev-parse", "--show-toplevel")
	if !ok || strings.TrimSpace(out) == "" {
		return "", false
	}
	return strings.TrimSpace(out), true
}

func keys(root string) (string, string, bool) {
	roots, ok := git(root, "rev-list", "--max-parents=0", "HEAD")
	if !ok || strings.TrimSpace(roots) == "" {
		return "", "", false
	}
	parts := strings.Fields(roots)
	sort.Strings(parts)
	head, ok := git(root, "rev-parse", "HEAD")
	if !ok || strings.TrimSpace(head) == "" {
		return "", "", false
	}
	return parts[0], strings.TrimSpace(head), true
}

func cachePath(rootSHA, headSHA string) string {
	base := os.Getenv("DEVRITES_PROFILE_CACHE")
	if base == "" {
		base = "/tmp/compound-engineering/devrites/repo-profile"
	}
	return filepath.Join(base, rootSHA, headSHA+".json")
}

func readCache(path, rootSHA, headSHA string) (Profile, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, false
	}
	var env cacheEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Profile{}, false
	}
	if env.ProfileSchemaVersion != schemaVersion || env.RootSHA != rootSHA || env.HeadSHA != headSHA || env.Profile.ProfileSchemaVersion != schemaVersion {
		return Profile{}, false
	}
	return env.Profile, true
}

func writeCache(path string, p Profile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write profile cache %s: %w", path, err)
	}
	env := cacheEnvelope{ProfileSchemaVersion: schemaVersion, RootSHA: p.RootSHA, HeadSHA: p.HeadSHA, Profile: p}
	raw, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile cache: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".profile-*.tmp")
	if err != nil {
		return fmt.Errorf("write profile cache %s: %w", path, err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("write profile cache %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("write profile cache %s: %w", path, err)
	}
	return os.Rename(name, path)
}

func derive(root, rootSHA, headSHA string) Profile {
	p := Profile{ProfileSchemaVersion: schemaVersion, RootSHA: rootSHA, HeadSHA: headSHA, Inputs: map[string]FileInfo{}}
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".git") || name == "node_modules" {
			continue
		}
		if e.IsDir() {
			name += "/"
		}
		p.TopLevel = append(p.TopLevel, name)
	}
	sort.Strings(p.TopLevel)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "pack" || base == ".scratch" || base == ".research" {
				return filepath.SkipDir
			}
			return nil
		}
		if isManifest(rel) {
			p.Manifests = append(p.Manifests, rel)
		}
		if isProfileInput(rel) {
			if info, ok := fileInfo(path); ok {
				p.Inputs[rel] = info
			}
		}
		return nil
	})
	sort.Strings(p.Manifests)
	return p
}

func fileInfo(path string) (FileInfo, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return FileInfo{}, false
	}
	sum := sha256.Sum256(raw)
	return FileInfo{SHA256: hex.EncodeToString(sum[:]), Bytes: len(raw)}, true
}

func cleanProfileInputs(root string) bool {
	out, ok := git(root, "status", "--porcelain", "--untracked-files=all")
	if !ok {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		rest := ""
		if len(line) >= 3 {
			rest = line[3:]
		}
		for _, part := range strings.Split(rest, " -> ") {
			if isProfileInput(strings.Trim(part, " \t\"")) {
				return false
			}
		}
	}
	return true
}

func isProfileInput(rel string) bool {
	base := filepath.Base(rel)
	if isManifest(rel) || matchAny(base, "LICENSE", "LICENSE.md", "LICENSE.txt", "CONTEXT.md", "README.md", "AGENTS.md", "CLAUDE.md", "GEMINI.md", "package.json") {
		return true
	}
	if !strings.Contains(rel, "/") && matchAny(base, "CONTRIBUTING.md", "SECURITY.md", "Makefile") {
		return true
	}
	if strings.HasPrefix(rel, "docs/adr/") || strings.HasPrefix(rel, ".github/workflows/") || strings.HasPrefix(rel, ".devrites/principles.md") || strings.HasPrefix(rel, ".devrites/conventions.md") {
		return true
	}
	return false
}

func isManifest(rel string) bool {
	base := filepath.Base(rel)
	return matchAny(base,
		"package.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "go.mod", "go.sum", "Cargo.toml", "Cargo.lock", "pyproject.toml", "requirements.txt", "Gemfile", "Gemfile.lock", "composer.json", "composer.lock", "pom.xml", "build.gradle", "build.gradle.kts", "deno.json", "deno.lock")
}

func matchAny(s string, vals ...string) bool {
	return slices.Contains(vals, s)
}

func git(dir string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func writeJSON(w io.Writer, v any) {
	raw, _ := json.MarshalIndent(v, "", "  ")
	fmt.Fprintln(w, string(raw))
}
