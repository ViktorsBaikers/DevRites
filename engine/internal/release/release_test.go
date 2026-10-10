package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLatestAndAcquireVerifiedRelease(t *testing.T) {
	const tag = "v4.1.0"
	bundleName := "devrites-" + tag + ".tar.gz"
	bundle := testBundle(t, tag, map[string]string{
		"package.json": `{"version":"4.1.0"}`,
		"pack/generated/claude/skills/rite/SKILL.md": "rite\n",
	})
	binaryName, err := platformBinaryName()
	if err != nil {
		t.Skip(err)
	}
	engine := []byte("#!/bin/sh\nprintf success > \"$MARKER\"\nexit 23\n")
	assets := map[string][]byte{
		bundleName: bundle,
		binaryName: engine,
	}
	installAttestationStandIn(t, bundle, engine)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/latest" {
			_, _ = fmt.Fprintf(w, `{"tag_name":%q}`, tag)
			return
		}
		name := filepath.Base(r.URL.Path)
		if strings.HasSuffix(name, ".sha256") {
			assetName := strings.TrimSuffix(name, ".sha256")
			asset, ok := assets[assetName]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(asset), assetName)
			return
		}
		asset, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(asset)
	}))
	defer server.Close()
	oldAPI, oldWeb := apiBaseURL, webBaseURL
	apiBaseURL, webBaseURL = server.URL, server.URL
	t.Cleanup(func() { apiBaseURL, webBaseURL = oldAPI, oldWeb })

	gotTag, err := Latest(context.Background(), "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if gotTag != tag {
		t.Fatalf("latest tag = %q, want %q", gotTag, tag)
	}
	candidate, cleanup, err := Acquire(context.Background(), "owner/repo", tag)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(candidate.SourceDir)
	if _, err := os.Stat(filepath.Join(candidate.PayloadDir, "claude", "skills", "rite", "SKILL.md")); err != nil {
		t.Fatalf("extracted payload: %v", err)
	}
	marker := filepath.Join(t.TempDir(), "marker")
	cmd := exec.Command(candidate.EnginePath)
	cmd.Env = append(os.Environ(), "MARKER="+marker)
	if err := cmd.Run(); err == nil {
		t.Fatal("attested engine exited successfully; want positive-control exit 23")
	} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 23 {
		t.Fatalf("attested engine exit = %v, want 23", err)
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "success" {
		t.Fatalf("attested engine marker = %q, %v", got, err)
	}
	cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("cleanup kept %s: %v", root, err)
	}
}

func TestValidTag(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want bool
	}{
		{tag: "v4.1.0", want: true},
		{tag: "v4.1.0-rc.1+build.2", want: true},
		{tag: "4.1.0", want: false},
		{tag: "v4.1.0-01", want: false},
		{tag: "v4.1", want: false},
	} {
		if got := validTag(tc.tag); got != tc.want {
			t.Errorf("validTag(%q) = %t, want %t", tc.tag, got, tc.want)
		}
	}
}

func TestAcquireRejectsOriginSubstitution(t *testing.T) {
	const tag = "v4.1.0"
	trusted := testBundle(t, tag, map[string]string{
		"package.json": `{"version":"4.1.0"}`,
		"pack/generated/claude/skills/rite/SKILL.md": "rite\n",
	})
	attacker := testBundle(t, tag, map[string]string{
		"package.json": `{"version":"4.1.0"}`,
		"pack/generated/claude/skills/rite/SKILL.md": "attacker\n",
	})
	binaryName, err := platformBinaryName()
	if err != nil {
		t.Skip(err)
	}
	installAttestationStandIn(t, trusted, []byte("trusted engine"))
	assets := map[string][]byte{"devrites-" + tag + ".tar.gz": attacker, binaryName: []byte("attacker engine")}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		assetName := strings.TrimSuffix(name, ".sha256")
		asset, ok := assets[assetName]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(name, ".sha256") {
			_, _ = fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(asset), assetName)
			return
		}
		_, _ = w.Write(asset)
	}))
	defer server.Close()
	oldWeb := webBaseURL
	webBaseURL = server.URL
	t.Cleanup(func() { webBaseURL = oldWeb })

	_, _, err = Acquire(context.Background(), "owner/repo", tag)
	if err == nil || !strings.Contains(err.Error(), "attestation verification failed") {
		t.Fatalf("Acquire error = %v, want attestation refusal despite a matching origin checksum", err)
	}
}

func TestAcquireRejectsSingleAssetSubstitution(t *testing.T) {
	const tag = "v4.1.0"
	files := map[string]string{
		"package.json": `{"version":"4.1.0"}`,
		"pack/generated/claude/skills/rite/SKILL.md": "rite\n",
	}
	trustedBundle := testBundle(t, tag, files)
	files["pack/generated/claude/skills/rite/SKILL.md"] = "attacker\n"
	attackerBundle := testBundle(t, tag, files)
	trustedEngine, attackerEngine := []byte("trusted engine"), []byte("attacker engine")
	binaryName, err := platformBinaryName()
	if err != nil {
		t.Skip(err)
	}
	for _, tc := range []struct {
		name, refusedAsset string
		bundle, engine     []byte
	}{
		{"engine only", binaryName, trustedBundle, attackerEngine},
		{"bundle only", "devrites-" + tag + ".tar.gz", attackerBundle, trustedEngine},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installAttestationStandIn(t, trustedBundle, trustedEngine)
			assets := map[string][]byte{"devrites-" + tag + ".tar.gz": tc.bundle, binaryName: tc.engine}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				name := filepath.Base(r.URL.Path)
				assetName := strings.TrimSuffix(name, ".sha256")
				asset, ok := assets[assetName]
				if !ok {
					http.NotFound(w, r)
					return
				}
				if strings.HasSuffix(name, ".sha256") {
					_, _ = fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(asset), assetName)
					return
				}
				_, _ = w.Write(asset)
			}))
			defer server.Close()
			oldWeb := webBaseURL
			webBaseURL = server.URL
			t.Cleanup(func() { webBaseURL = oldWeb })

			_, cleanup, err := Acquire(context.Background(), "owner/repo", tag)
			if err == nil {
				cleanup()
				t.Fatal("Acquire accepted a release with one unattested asset")
			}
			want := "attestation verification failed for " + tc.refusedAsset
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("Acquire error = %v, want %q", err, want)
			}
		})
	}
}

func installAttestationStandIn(t *testing.T, trusted ...[]byte) {
	t.Helper()
	dir := t.TempDir()
	var hashes strings.Builder
	for _, data := range trusted {
		fmt.Fprintf(&hashes, "%x\n", sha256.Sum256(data))
	}
	allowed := filepath.Join(dir, "allowed-digests")
	if err := os.WriteFile(allowed, []byte(hashes.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	gh := filepath.Join(dir, "gh")
	script := `#!/bin/sh
[ "$1" = attestation ] && [ "$2" = verify ] && [ "$4" = --repo ] && [ "$5" = owner/repo ] && [ "$6" = --signer-workflow ] && [ "$7" = owner/repo/.github/workflows/ci.yml@refs/heads/main ] || exit 64
got=$(shasum -a 256 "$3" | awk '{print $1}') || exit 65
grep -Fxq "$got" "$ATTESTED_DIGESTS"
`
	if err := os.WriteFile(gh, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ATTESTED_DIGESTS", allowed)
}

func TestAttestationStandInIsStrict(t *testing.T) {
	trusted := []byte("trusted engine")
	installAttestationStandIn(t, trusted, []byte("other"))
	dir := t.TempDir()
	good := filepath.Join(dir, "good")
	bad := filepath.Join(dir, "bad")
	if err := os.WriteFile(good, trusted, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("attacker engine"), 0o600); err != nil {
		t.Fatal(err)
	}
	const signer = "owner/repo/.github/workflows/ci.yml@refs/heads/main"
	run := func(file, repo, workflow string) error {
		return exec.Command("gh", "attestation", "verify", file, "--repo", repo, "--signer-workflow", workflow).Run()
	}
	if err := run(good, "owner/repo", signer); err != nil {
		t.Fatalf("stand-in rejected the pinned request: %v", err)
	}
	if run(good, "other/repo", signer) == nil {
		t.Error("stand-in accepted a wrong repo")
	}
	if run(good, "owner/repo", "owner/repo/.github/workflows/other.yml@refs/heads/main") == nil {
		t.Error("stand-in accepted a wrong signer workflow")
	}
	if run(bad, "owner/repo", signer) == nil {
		t.Error("stand-in accepted an unattested digest")
	}
}

func TestAcquireRejectsMissingAttestationVerifier(t *testing.T) {
	const tag = "v4.1.0"
	binaryName, err := platformBinaryName()
	if err != nil {
		t.Skip(err)
	}
	assets := map[string][]byte{
		"devrites-" + tag + ".tar.gz": testBundle(t, tag, map[string]string{
			"package.json": `{"version":"4.1.0"}`,
			"pack/generated/claude/skills/rite/SKILL.md": "rite\n",
		}),
		binaryName: []byte("engine"),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		assetName := strings.TrimSuffix(name, ".sha256")
		asset, ok := assets[assetName]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(name, ".sha256") {
			_, _ = fmt.Fprintf(w, "%x  %s\n", sha256.Sum256(asset), assetName)
			return
		}
		_, _ = w.Write(asset)
	}))
	defer server.Close()
	oldWeb := webBaseURL
	webBaseURL = server.URL
	t.Cleanup(func() { webBaseURL = oldWeb })
	t.Setenv("PATH", t.TempDir())

	_, _, err = Acquire(context.Background(), "owner/repo", tag)
	if err == nil || !strings.Contains(err.Error(), "gh (GitHub CLI) is required") {
		t.Fatalf("Acquire error = %v, want refusal when gh is not installed", err)
	}
}

func TestAcquireRejectsChecksumMismatch(t *testing.T) {
	const tag = "v4.1.0"
	bundleName := "devrites-" + tag + ".tar.gz"
	bundle := testBundle(t, tag, map[string]string{
		"package.json": `{"version":"4.1.0"}`,
		"pack/generated/claude/skills/rite/SKILL.md": "rite\n",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			_, _ = fmt.Fprintf(w, "%064d  %s\n", 0, bundleName)
			return
		}
		_, _ = w.Write(bundle)
	}))
	defer server.Close()
	oldWeb := webBaseURL
	webBaseURL = server.URL
	t.Cleanup(func() { webBaseURL = oldWeb })

	_, _, err := Acquire(context.Background(), "owner/repo", tag)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Acquire error = %v, want checksum mismatch", err)
	}
}

func TestExtractBundleRejectsPathTraversal(t *testing.T) {
	const tag = "v4.1.0"
	archive := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	data := testBundle(t, tag, map[string]string{"../escape": "bad"})
	if err := os.WriteFile(archive, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := extractBundle(archive, t.TempDir(), tag)
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("extractBundle error = %v, want unsafe path", err)
	}
}

func testBundle(t *testing.T, tag string, files map[string]string) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		data := []byte(body)
		header := &tar.Header{
			Name:     "devrites-" + tag + "/" + name,
			Mode:     0o644,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}
