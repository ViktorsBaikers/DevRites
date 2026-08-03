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
	assets := map[string][]byte{
		bundleName: bundle,
		binaryName: []byte("test engine"),
	}
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
	if got, err := os.ReadFile(candidate.EnginePath); err != nil || string(got) != "test engine" {
		t.Fatalf("engine = %q, %v", got, err)
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
